package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pdf-cli/internal/client"
	"pdf-cli/internal/config"
	clierr "pdf-cli/internal/errors"
	"pdf-cli/internal/output"

	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "PDF 工具集",
	Long: `PDF 工具模块，基于 core/tools 接口提供合并、转换、拆分、旋转、压缩等操作。

示例：
  pdf-cli tools merge --files a.pdf,b.pdf
  pdf-cli tools convert pdf-to-word --file a.pdf
  pdf-cli tools split --file a.pdf --mode pages-per-pdf --pages-per-pdf 2
  pdf-cli tools security encrypt --file a.pdf --password 123456`,
}

type FileEntry struct {
	Filename string `json:"filename"`
	Name     string `json:"name"`
}

type toolEnvelope struct {
	Code      interface{}     `json:"code"`
	Data      json.RawMessage `json:"data"`
	Message   string          `json:"message"`
	State     string          `json:"state"`
	IsSuccess bool            `json:"isSuccess"`
}

type toolUploadPrepare struct {
	AwsUploadURL string `json:"awsUploadUrl"`
	BlobFileName string `json:"blobFileName"`
	FileRealName string `json:"fileRealName"`
}

func toolSuccessCode(code interface{}) bool {
	switch v := code.(type) {
	case string:
		return v == "1" || v == "200"
	case float64:
		return v == 1 || v == 200
	case int:
		return v == 1 || v == 200
	default:
		return false
	}
}

func toolPost(path string, payload interface{}) (*toolEnvelope, error) {
	c := client.NewBaseClient()
	body, err := c.PostJSON(path, payload)
	if err != nil {
		return nil, err
	}
	var resp toolEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, clierr.New(clierr.ExitInternal, clierr.TypeInternal, "解析响应失败: "+err.Error(), "", false)
	}
	if !toolSuccessCode(resp.Code) {
		return nil, client.ClassifyBusinessError(0, resp.Code, resp.Message)
	}
	return &resp, nil
}

func toolGet(path string, params map[string]string) (*toolEnvelope, error) {
	c := client.NewBaseClient()
	body, err := c.Get(path, params)
	if err != nil {
		return nil, err
	}
	var resp toolEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, clierr.New(clierr.ExitInternal, clierr.TypeInternal, "解析响应失败: "+err.Error(), "", false)
	}
	if strings.TrimLeft(path, "/") != "core/tools/operate/status" && !toolSuccessCode(resp.Code) {
		return nil, client.ClassifyBusinessError(0, resp.Code, resp.Message)
	}
	return &resp, nil
}

func toolPrepareUpload(filePath string) (*toolUploadPrepare, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, clierr.ParamError("无法读取文件: "+filePath, "请检查文件路径")
	}
	resp, err := toolPost("core/tools/box/file/aws/pre/upload", map[string]interface{}{
		"fileRealName": filepath.Base(filePath),
		"fileSize":     info.Size(),
	})
	if err != nil {
		return nil, err
	}
	var data toolUploadPrepare
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, clierr.New(clierr.ExitInternal, clierr.TypeInternal, "解析上传准备响应失败: "+err.Error(), "", false)
	}
	if data.AwsUploadURL == "" || data.BlobFileName == "" {
		return nil, clierr.New(clierr.ExitInternal, clierr.TypeInternal, "上传准备响应不完整: 缺少 awsUploadUrl/blobFileName", "", false)
	}
	return &data, nil
}

func toolUploadToAWS(uploadURL, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return clierr.ParamError("无法打开文件: "+filePath, "请检查文件路径")
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return clierr.ParamError("无法读取文件: "+filePath, "请检查文件路径")
	}

	req, err := http.NewRequest(http.MethodPut, uploadURL, f)
	if err != nil {
		return clierr.NetError("创建上传请求失败: "+err.Error(), "")
	}
	req.ContentLength = info.Size()
	req.TransferEncoding = nil
	req.Header.Set("Content-Type", "application/pdf")

	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return client.ClassifyTransportError("上传文件失败", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return client.ClassifyHTTPStatus(resp.StatusCode, body)
	}
	return nil
}

func toolRegisterUpload(filePath string, pdfToolCode int, blobFileName string) (*FileEntry, error) {
	resp, err := toolPost("core/tools/box/file/new/upload", map[string]interface{}{
		"pdfToolCode":  pdfToolCode,
		"fileRealName": filepath.Base(filePath),
		"blobFileName": blobFileName,
	})
	if err != nil {
		return nil, err
	}
	var data struct {
		BlobFileName   string `json:"blobFileName"`
		OriginFileName string `json:"originFileName"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, clierr.New(clierr.ExitInternal, clierr.TypeInternal, "解析上传登记响应失败: "+err.Error(), "", false)
	}
	return &FileEntry{Filename: data.BlobFileName, Name: data.OriginFileName}, nil
}

func toolUploadFile(filePath string, pdfToolCode int) (*FileEntry, error) {
	prepare, err := toolPrepareUpload(filePath)
	if err != nil {
		return nil, err
	}
	if err := toolUploadToAWS(prepare.AwsUploadURL, filePath); err != nil {
		return nil, err
	}
	return toolRegisterUpload(filePath, pdfToolCode, prepare.BlobFileName)
}

func toolUploadFiles(filePaths []string, pdfToolCode int) ([]FileEntry, error) {
	files := make([]FileEntry, 0, len(filePaths))
	for _, filePath := range filePaths {
		entry, err := toolUploadFile(filePath, pdfToolCode)
		if err != nil {
			return nil, err
		}
		files = append(files, *entry)
	}
	return files, nil
}

func toolSubmitAction(action string, pdfToolCode int, data interface{}) (string, error) {
	resp, err := toolPost("core/tools/todo/operate", map[string]interface{}{
		"action":      action,
		"data":        data,
		"pdfToolCode": pdfToolCode,
	})
	if err != nil {
		return "", err
	}
	var queryKey string
	if err := json.Unmarshal(resp.Data, &queryKey); err == nil && queryKey != "" {
		return queryKey, nil
	}
	return "", clierr.New(clierr.ExitInternal, clierr.TypeInternal, "任务创建响应不完整: 缺少 queryKey", "", false)
}

func toolGetTask(queryKey string) (*toolEnvelope, error) {
	resp, err := toolGet("core/tools/operate/status", map[string]string{"queryKey": queryKey})
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess {
		return nil, toolTaskFailureError(queryKey, resp)
	}
	return resp, nil
}

func toolTaskFailureError(queryKey string, resp *toolEnvelope) error {
	msg := strings.TrimSpace(resp.Message)
	if msg == "" {
		msg = "任务执行失败"
	}
	hint := "用 tools job status --query-key " + queryKey + " 查看详情"
	var e *clierr.CLIError
	if raw := strings.TrimSpace(string(resp.Data)); raw != "" && raw != "null" && raw != "{}" && raw != "[]" {
		var text string
		if err := json.Unmarshal(resp.Data, &text); err == nil && strings.TrimSpace(text) != "" {
			e = clierr.TaskFailedError(msg+": "+strings.TrimSpace(text), hint)
		} else {
			e = clierr.TaskFailedError(msg+": "+raw, hint)
		}
	} else {
		e = clierr.TaskFailedError(msg, hint)
	}
	return e.WithDetail("query_key", queryKey)
}

func toolPollTask(queryKey string) (*toolEnvelope, error) {
	for i := 0; i < 120; i++ {
		resp, err := toolGetTask(queryKey)
		if err != nil {
			return nil, err
		}
		switch strings.ToUpper(resp.State) {
		case "SUCCESS":
			return resp, nil
		case "FAILURE":
			return nil, toolTaskFailureError(queryKey, resp)
		}
		time.Sleep(2 * time.Second)
	}
	return nil, clierr.TimeoutError(
		"任务处理超时",
		"任务可能仍在跑，使用 pdf-cli tools job status --query-key "+queryKey+" 继续轮询",
	).WithDetail("query_key", queryKey)
}

func toolParseResultFile(data json.RawMessage) (*FileEntry, bool) {
	var file FileEntry
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, false
	}
	if file.Filename == "" {
		return nil, false
	}
	return &file, true
}

func toolResultSavePath(file *FileEntry) (string, error) {
	name := file.Name
	if name == "" {
		name = file.Filename
	}
	if filepath.Ext(name) == "" && filepath.Ext(file.Filename) != "" {
		name += filepath.Ext(file.Filename)
	}
	if outputFlag == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, name), nil
	}
	if info, err := os.Stat(outputFlag); err == nil && info.IsDir() {
		return filepath.Join(outputFlag, name), nil
	}
	return outputFlag, nil
}

func toolDownloadBases() []string {
	cfg := config.Load()
	bases := []string{strings.TrimRight(cfg.GetDownloadURL(), "/")}

	baseURL, err := url.Parse(cfg.GetBaseURL())
	if err != nil || baseURL.Host == "" {
		return bases
	}

	altHost := ""
	switch {
	case strings.Contains(baseURL.Host, "pre.gdpdf.com"):
		altHost = "res.pre.gdpdf.com"
	case strings.Contains(baseURL.Host, "gdpdf.com"):
		altHost = "res.gdpdf.com"
	case strings.Contains(baseURL.Host, "doclingo.ai"):
		altHost = "res.doclingo.ai"
	}
	if altHost == "" {
		return bases
	}

	altBase := baseURL.Scheme + "://" + altHost
	for _, base := range bases {
		if base == altBase {
			return bases
		}
	}
	return append(bases, altBase)
}

func toolDownloadResult(filename, outputPath string) error {
	var lastErr error
	for _, base := range toolDownloadBases() {
		downloadURL := base + "/pdf/box/" + url.PathEscape(filename)
		resp, err := http.Get(downloadURL)
		if err != nil {
			lastErr = client.ClassifyTransportError("下载结果失败", err)
			continue
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = client.ClassifyHTTPStatus(resp.StatusCode, body)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			resp.Body.Close()
			return clierr.ParamError("无法创建输出目录", "")
		}
		f, err := os.Create(outputPath)
		if err != nil {
			resp.Body.Close()
			return clierr.ParamError("无法创建输出文件: "+outputPath, "")
		}
		_, err = io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		return err
	}
	return lastErr
}

func toolPrintTaskResult(queryKey string, resp *toolEnvelope) error {
	format := output.GetFormat(formatFlag)
	var result interface{}
	_ = json.Unmarshal(resp.Data, &result)
	if format == "json" {
		output.PrintJSON(map[string]interface{}{
			"queryKey": queryKey,
			"jobId":    queryKey,
			"state":    resp.State,
			"result":   result,
		})
		return nil
	}
	if file, ok := toolParseResultFile(resp.Data); ok {
		fmt.Println("任务状态: 已完成")
		output.PrintOrderedPretty([]string{"查询键", "结果文件", "下载文件名"}, map[string]interface{}{
			"查询键":   queryKey,
			"结果文件":  file.Filename,
			"下载文件名": file.Name,
		})
		return nil
	}
	fmt.Println("任务状态: 已完成")
	fmt.Printf("  查询键 : %s\n", queryKey)
	output.PrintJSON(result)
	return nil
}

func toolRunAction(action string, pdfToolCode int, data interface{}) error {
	queryKey, err := toolSubmitAction(action, pdfToolCode, data)
	if err != nil {
		return err
	}
	resp, err := toolPollTask(queryKey)
	if err != nil {
		if cliError, ok := err.(*clierr.CLIError); ok && cliError.Hint == "" {
			cliError.Hint = "queryKey: " + queryKey
		}
		return err
	}
	file, ok := toolParseResultFile(resp.Data)
	if !ok {
		return toolPrintTaskResult(queryKey, resp)
	}
	savePath, err := toolResultSavePath(file)
	if err != nil {
		return err
	}
	if err := toolDownloadResult(file.Filename, savePath); err != nil {
		format := output.GetFormat(formatFlag)
		if format == "json" {
			output.PrintJSON(map[string]interface{}{
				"queryKey":      queryKey,
				"jobId":         queryKey,
				"state":         resp.State,
				"filename":      file.Filename,
				"name":          file.Name,
				"savedTo":       savePath,
				"downloadError": err.Error(),
			})
		} else {
			fmt.Println("任务已完成，但下载失败")
			output.PrintOrderedPretty([]string{"查询键", "结果文件", "计划保存到", "下载错误"}, map[string]interface{}{
				"查询键":   queryKey,
				"结果文件":  file.Name,
				"计划保存到": savePath,
				"下载错误":  err.Error(),
			})
		}
		return err
	}
	format := output.GetFormat(formatFlag)
	if format == "json" {
		output.PrintJSON(map[string]interface{}{
			"queryKey": queryKey,
			"jobId":    queryKey,
			"state":    resp.State,
			"filename": file.Filename,
			"name":     file.Name,
			"savedTo":  savePath,
		})
		return nil
	}
	fmt.Println("任务完成")
	output.PrintOrderedPretty([]string{"查询键", "结果文件", "保存到"}, map[string]interface{}{
		"查询键":  queryKey,
		"结果文件": file.Name,
		"保存到":  savePath,
	})
	return nil
}

func toolRequireFile(cmd *cobra.Command) (string, error) {
	filePath, _ := cmd.Flags().GetString("file")
	if strings.TrimSpace(filePath) == "" {
		return "", clierr.ParamError("请提供文件路径", "使用 --file 参数")
	}
	return filePath, nil
}

func toolRequireFiles(cmd *cobra.Command) ([]string, error) {
	files, _ := cmd.Flags().GetStringSlice("files")
	if len(files) == 0 {
		return nil, clierr.ParamError("请提供文件路径", "使用 --files 参数")
	}
	return files, nil
}

func toolRequireQueryKey(cmd *cobra.Command) (string, error) {
	queryKey, _ := cmd.Flags().GetString("query-key")
	if strings.TrimSpace(queryKey) != "" {
		return strings.TrimSpace(queryKey), nil
	}
	jobID, _ := cmd.Flags().GetString("job-id")
	if strings.TrimSpace(jobID) != "" {
		return strings.TrimSpace(jobID), nil
	}
	return "", clierr.ParamError("请提供查询键", "使用 --query-key 参数")
}

func toolParseIntList(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	vals := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(part, "%d", &n); err != nil || n <= 0 {
			return nil, clierr.ParamError("参数格式错误: "+raw, "请使用逗号分隔的正整数，例如 1,3,5")
		}
		vals = append(vals, n)
	}
	if len(vals) == 0 {
		return nil, clierr.ParamError("参数不能为空", "请使用逗号分隔的正整数，例如 1,3,5")
	}
	return vals, nil
}

func toolParsePageSpec(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	seen := map[int]bool{}
	pages := make([]int, 0)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				return nil, clierr.ParamError("页码范围格式错误: "+part, "请使用 1-3,5 这样的格式")
			}
			var start, end int
			if _, err := fmt.Sscanf(strings.TrimSpace(rangeParts[0]), "%d", &start); err != nil || start <= 0 {
				return nil, clierr.ParamError("页码范围格式错误: "+part, "请使用 1-3,5 这样的格式")
			}
			if _, err := fmt.Sscanf(strings.TrimSpace(rangeParts[1]), "%d", &end); err != nil || end <= 0 || end < start {
				return nil, clierr.ParamError("页码范围格式错误: "+part, "请使用 1-3,5 这样的格式")
			}
			for i := start; i <= end; i++ {
				if !seen[i] {
					seen[i] = true
					pages = append(pages, i)
				}
			}
			continue
		}
		var n int
		if _, err := fmt.Sscanf(part, "%d", &n); err != nil || n <= 0 {
			return nil, clierr.ParamError("页码格式错误: "+raw, "请使用 1-3,5 这样的格式")
		}
		if !seen[n] {
			seen[n] = true
			pages = append(pages, n)
		}
	}
	if len(pages) == 0 {
		return nil, clierr.ParamError("页码不能为空", "请使用 1-3,5 这样的格式")
	}
	sort.Ints(pages)
	return pages, nil
}

func toolBuildRotateList(pages []int, angle int) []int {
	maxPage := 0
	for _, page := range pages {
		if page > maxPage {
			maxPage = page
		}
	}
	rotates := make([]int, maxPage)
	for _, page := range pages {
		rotates[page-1] = angle
	}
	return rotates
}

func toolBuildPageSelections(file FileEntry, pages []int) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(pages))
	for _, page := range pages {
		items = append(items, map[string]interface{}{
			"filename":  file.Filename,
			"name":      file.Name,
			"fileIndex": 0,
			"pageIndex": page - 1,
		})
	}
	return items
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
