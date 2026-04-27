package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pdf-cli/internal/auth"
	"pdf-cli/internal/config"
	clierr "pdf-cli/internal/errors"
)

// ClassifyTransportError is the exported alias used by command code that
// performs raw HTTP requests outside the Client helpers.
func ClassifyTransportError(action string, err error) *clierr.CLIError {
	return classifyTransportError(action, err)
}

// ClassifyHTTPStatus is the exported alias used by command code that
// performs raw HTTP requests outside the Client helpers.
func ClassifyHTTPStatus(status int, body []byte) *clierr.CLIError {
	return classifyHTTPStatus(status, body)
}

// ClassifyBusinessError is the exported alias for command code that owns
// its own envelope parser (e.g. tools_extra) and needs to map a backend
// failure code+message into the typed taxonomy.
func ClassifyBusinessError(httpStatus int, code interface{}, message string) *clierr.CLIError {
	return classifyBusinessError(httpStatus, code, message)
}

// classifyTransportError maps a low-level HTTP transport error onto the
// typed exit-code taxonomy. Timeouts get their own code so agents can apply
// a different retry strategy than for connection failures.
func classifyTransportError(action string, err error) *clierr.CLIError {
	msg := action + ": " + err.Error()
	if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
		return clierr.TimeoutError(msg, "请求超时，可重试或考虑提高超时阈值")
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return clierr.TimeoutError(msg, "请求超时，可重试或考虑提高超时阈值")
	}
	return clierr.NetError(msg, "请检查网络连接")
}

// classifyHTTPStatus turns an HTTP status code + response body into a typed
// CLIError. Caller should only invoke this when status >= 400.
func classifyHTTPStatus(status int, body []byte) *clierr.CLIError {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	msg := fmt.Sprintf("HTTP %d: %s", status, snippet)
	var e *clierr.CLIError
	switch {
	case status == 401:
		e = clierr.AuthError(msg, "token 无效或已过期，执行 pdf-cli auth login 重新登录")
	case status == 402:
		e = clierr.QuotaError(msg, "额度不足，请充值或等待周期重置")
	case status == 403:
		e = clierr.PermissionError(msg, "权限不足，可能需要升级会员或更换账号")
	case status == 404:
		e = clierr.NotFoundError(msg, "资源不存在，请检查 ID 是否正确或已过期")
	case status == 409:
		e = clierr.ConflictError(msg, "资源状态冲突，请改用查询命令而不是重复发起")
	case status == 429:
		e = clierr.RateLimitedError(msg, "被限流，建议退避后重试")
	case status >= 500:
		e = clierr.ServerError(msg, "服务端错误，可退避后重试")
	default:
		e = clierr.New(clierr.ExitUnknown, clierr.TypeUnknown, msg, "", false)
	}
	return e.WithHTTP(status)
}

// classifyBusinessError handles HTTP 200 + backend code != success. The
// backend collapses many semantic failures behind a single envelope, so we
// fall back to keyword matching on the message to recover the most useful
// distinctions (auth/quota/permission). When the backend code itself looks
// HTTP-ish (numeric 4xx/5xx), it gets first-class treatment. Anything we
// cannot recognise becomes Unknown with backend_code attached so agents can
// still branch on it.
func classifyBusinessError(httpStatus int, code interface{}, message string) *clierr.CLIError {
	if message == "" {
		message = "请求失败"
	}
	codeStr := ""
	if code != nil {
		codeStr = fmt.Sprintf("%v", code)
	}
	lower := strings.ToLower(message)

	keywordMatch := func() *clierr.CLIError {
		switch {
		case strings.Contains(message, "未登录") ||
			strings.Contains(message, "请登录") ||
			strings.Contains(message, "token") ||
			strings.Contains(lower, "unauthor") ||
			strings.Contains(lower, "not login"):
			return clierr.AuthError(message, "请执行 pdf-cli auth login 重新登录")
		case strings.Contains(message, "额度") ||
			strings.Contains(message, "余额") ||
			strings.Contains(message, "次数不足") ||
			strings.Contains(lower, "quota") ||
			strings.Contains(lower, "insufficient"):
			return clierr.QuotaError(message, "额度不足，请充值或等待周期重置")
		case strings.Contains(message, "权限") ||
			strings.Contains(message, "会员") ||
			strings.Contains(lower, "forbidden") ||
			strings.Contains(lower, "permission"):
			return clierr.PermissionError(message, "权限不足，可能需要升级会员或更换账号")
		case strings.Contains(message, "不存在") ||
			strings.Contains(message, "未找到") ||
			strings.Contains(lower, "not found"):
			return clierr.NotFoundError(message, "资源不存在，请检查 ID 是否正确")
		case strings.Contains(message, "已存在") ||
			strings.Contains(message, "正在") ||
			strings.Contains(lower, "conflict") ||
			strings.Contains(lower, "already"):
			return clierr.ConflictError(message, "状态冲突，请改用查询命令")
		case strings.Contains(message, "参数") ||
			strings.Contains(lower, "invalid argument") ||
			strings.Contains(lower, "invalid parameter") ||
			strings.Contains(lower, "bad request"):
			return clierr.ParamError(message, "请检查命令参数是否合法")
		}
		return nil
	}

	var e *clierr.CLIError
	if hit := keywordMatch(); hit != nil {
		e = hit
	} else if num, ok := numericCode(codeStr); ok {
		// Backend code looks HTTP-shaped — reuse the HTTP status mapping.
		e = classifyHTTPStatus(num, []byte(message))
		// Avoid double-prefixing "HTTP %d:" when message already has it.
		e.Message = message
	} else {
		e = clierr.New(clierr.ExitUnknown, clierr.TypeUnknown, message, "", false)
	}
	if httpStatus != 0 {
		e = e.WithHTTP(httpStatus)
	}
	if codeStr != "" {
		e = e.WithBackendCode(codeStr)
	}
	return e
}

// numericCode returns the int value of a backend code if it parses cleanly
// as an HTTP-shaped status (100-599). Anything else returns false so the
// caller falls back to keyword matching.
func numericCode(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		n = n*10 + int(ch-'0')
		if n > 999 {
			return 0, false
		}
	}
	if n < 100 || n > 599 {
		return 0, false
	}
	return n, true
}

// APIResponse 是 baseURL 服务的标准响应格式
type APIResponse struct {
	Code     interface{}     `json:"code"`
	Data     json.RawMessage `json:"data,omitempty"`
	Message  string          `json:"message,omitempty"`
	List     json.RawMessage `json:"list,omitempty"`
	DataList json.RawMessage `json:"dataList,omitempty"`
}

func (r *APIResponse) IsSuccess() bool {
	switch v := r.Code.(type) {
	case string:
		return v == "1"
	case float64:
		return v == 1 || v == 200
	}
	return false
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	isToolAPI  bool
}

func NewBaseClient() *Client {
	cfg := config.Load()
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		baseURL:    strings.TrimRight(cfg.GetBaseURL(), "/"),
	}
}

func NewToolClient() *Client {
	cfg := config.Load()
	return &Client{
		httpClient: &http.Client{Timeout: 300 * time.Second},
		baseURL:    strings.TrimRight(cfg.GetToolURL(), "/"),
		isToolAPI:  true,
	}
}

func (c *Client) buildURL(path string) string {
	path = strings.TrimLeft(path, "/")
	return c.baseURL + "/" + path
}

func (c *Client) setHeaders(req *http.Request) {
	if c.isToolAPI {
		token, _ := auth.LoadToken()
		if token != "" {
			req.Header.Set("X-API-KEY", token)
		}
	} else {
		token, _ := auth.LoadToken()
		if token != "" {
			req.Header.Set("token", token)
		}
		deviceID := auth.LoadDeviceID()
		if deviceID != "" {
			req.Header.Set("deviceId", deviceID)
		}
		req.Header.Set("clientType", "cli")
		req.Header.Set("internationalCode", "zh-CN")
	}
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, classifyTransportError("网络请求失败", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, classifyTransportError("读取响应失败", err)
	}
	if resp.StatusCode >= 400 {
		return body, classifyHTTPStatus(resp.StatusCode, body)
	}
	return body, nil
}

// Get 发起 GET 请求
func (c *Client) Get(path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.buildURL(path), nil)
	if err != nil {
		return nil, err
	}
	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	return c.do(req)
}

// PostJSON 发起 JSON POST 请求
func (c *Client) PostJSON(path string, data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.buildURL(path), bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// PostForm 发起 form-urlencoded POST 请求
func (c *Client) PostForm(path string, data map[string]string) ([]byte, error) {
	form := make([]string, 0, len(data))
	for k, v := range data {
		form = append(form, k+"="+v)
	}
	body := strings.Join(form, "&")
	req, err := http.NewRequest("POST", c.buildURL(path), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req)
}

// PostMultipart 上传文件
func (c *Client) PostMultipart(path string, files map[string]string, fields map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for fieldName, filePath := range files {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, clierr.ParamError("无法打开文件: "+filePath, "请检查文件路径")
		}
		part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			f.Close()
			return nil, err
		}
		_, err = io.Copy(part, f)
		f.Close()
		if err != nil {
			return nil, err
		}
	}

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	writer.Close()

	req, err := http.NewRequest("POST", c.buildURL(path), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.do(req)
}

// PostMultipartFiles 上传多个文件到同一字段
func (c *Client) PostMultipartFiles(path string, fieldName string, filePaths []string, fields map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, filePath := range filePaths {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, clierr.ParamError("无法打开文件: "+filePath, "请检查文件路径")
		}
		part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			f.Close()
			return nil, err
		}
		_, err = io.Copy(part, f)
		f.Close()
		if err != nil {
			return nil, err
		}
	}

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	writer.Close()

	req, err := http.NewRequest("POST", c.buildURL(path), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.do(req)
}

// parseAPIResponse decodes the standard envelope and, if the backend
// indicates failure, returns a typed CLIError populated with HTTP status
// and backend code.
func parseAPIResponse(body []byte) (*APIResponse, error) {
	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, clierr.New(clierr.ExitInternal, clierr.TypeInternal, "响应解析失败: "+err.Error(), "", false)
	}
	if !resp.IsSuccess() {
		return &resp, classifyBusinessError(0, resp.Code, resp.Message)
	}
	return &resp, nil
}

// GetAPI 发起 GET 请求并解析为 APIResponse
func (c *Client) GetAPI(path string, params map[string]string) (*APIResponse, error) {
	body, err := c.Get(path, params)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse(body)
}

// PostJSONAPI 发起 JSON POST 并解析为 APIResponse
func (c *Client) PostJSONAPI(path string, data interface{}) (*APIResponse, error) {
	body, err := c.PostJSON(path, data)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse(body)
}

// PostFormAPI 发起 form POST 并解析为 APIResponse
func (c *Client) PostFormAPI(path string, data map[string]string) (*APIResponse, error) {
	body, err := c.PostForm(path, data)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse(body)
}

// PostMultipartAPI 上传文件并解析为 APIResponse
func (c *Client) PostMultipartAPI(path string, files map[string]string, fields map[string]string) (*APIResponse, error) {
	body, err := c.PostMultipart(path, files, fields)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse(body)
}

// DownloadFile 下载文件到本地，优先使用 Content-Disposition 中的文件名
func (c *Client) DownloadFile(path string, params map[string]string, outputPath string) error {
	req, err := http.NewRequest("GET", c.buildURL(path), nil)
	if err != nil {
		return err
	}
	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return classifyTransportError("下载失败", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return classifyHTTPStatus(resp.StatusCode, body)
	}

	// 从 Content-Disposition 提取服务端返回的真实文件名，修正输出扩展名
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, cdParams, err := mime.ParseMediaType(cd); err == nil {
			if serverName := cdParams["filename"]; serverName != "" {
				serverExt := filepath.Ext(serverName)
				localExt := filepath.Ext(outputPath)
				if serverExt != "" && serverExt != localExt {
					outputPath = strings.TrimSuffix(outputPath, localExt) + serverExt
				}
			}
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return clierr.ParamError("无法创建输出文件: "+outputPath, "")
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// GetRaw 发起 GET 请求返回原始响应体
func (c *Client) GetRaw(path string, params map[string]string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", c.buildURL(path), nil)
	if err != nil {
		return nil, 0, err
	}
	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, classifyTransportError("网络请求失败", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, classifyTransportError("读取响应失败", err)
	}
	return body, resp.StatusCode, nil
}
