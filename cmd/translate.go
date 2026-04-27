package cmd

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"pdf-cli/internal/auth"
	"pdf-cli/internal/client"
	"pdf-cli/internal/config"
	clierr "pdf-cli/internal/errors"
	"pdf-cli/internal/output"
	"pdf-cli/internal/userconfig"

	"github.com/spf13/cobra"
)

var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "PDF 翻译与 AI 翻译",
	Long: `翻译模块，支持 PDF 文档翻译全流程。

示例：
  pdf-cli translate languages
  pdf-cli translate engines
  pdf-cli translate upload --file ./paper.pdf
  pdf-cli translate start --file-key xxx --to zh
  pdf-cli translate free --file-key xxx --to zh
  pdf-cli translate arxiv --arxiv-id 2301.00001 --to zh
  pdf-cli translate text --record-id 123 --text "Hello" --engine 1
  pdf-cli translate status --task-id xxx
  pdf-cli translate download --task-id xxx`,
}

// translatePrecheck 先查询当前用户的 homepage 配置，再对即将执行的操作做限制条件校验。
// 网络/解析失败时仅打印警告，不中断流程，避免限制条件接口暂时不可用时影响正常使用。
type precheckOpts struct {
	kind     string
	fileSize int64
	filePath string // 用于 upload-free 计算 PDF 页数
	engine   string
	ocr      bool
	textLen  int // 用于 text 命令校验字符长度
}

// fetchVipLevel 返回当前登录用户的 vipLevel (0 = 非会员)，未登录或失败返回 -1。
func fetchVipLevel() int {
	if !auth.IsLoggedIn() {
		return -1
	}
	c := client.NewBaseClient()
	resp, err := c.GetAPI("user/basic/token/userinfo", nil)
	if err != nil {
		return -1
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return -1
	}
	if v, ok := data["vipLevel"]; ok {
		switch x := v.(type) {
		case float64:
			return int(x)
		case string:
			if n, err := strconv.Atoi(x); err == nil {
				return n
			}
		}
	}
	return 0
}

func isMemberUser() bool { return fetchVipLevel() > 0 }

// engineFlagTruthy 统一处理后端 0/1/2 或 "1"/"true" 的标志位。
func engineFlagTruthy(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x > 0
	case int:
		return x > 0
	case string:
		return x != "" && x != "0" && !strings.EqualFold(x, "false")
	}
	return false
}

// isEngineAvailable 根据 showFlag 判断引擎是否对当前用户可用。
func isEngineAvailable(em map[string]interface{}) bool {
	return engineFlagTruthy(em["showFlag"])
}

// engineInfo 返回匹配 engine (engineId 或 engineName) 的引擎元数据。
func engineInfo(engine string) map[string]interface{} {
	if engine == "" {
		return nil
	}
	c := client.NewBaseClient()
	resp, err := c.GetAPI("core/pdf/engines", nil)
	if err != nil {
		return nil
	}
	var engines map[string]interface{}
	if err := json.Unmarshal(resp.Data, &engines); err != nil {
		return nil
	}
	for _, e := range engines {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		id := fmt.Sprintf("%v", em["engineId"])
		name := fmt.Sprintf("%v", em["engineName"])
		if id == engine || name == engine {
			return em
		}
	}
	return nil
}

// isPremiumEngine 根据 highLevelFlag 判断高级引擎 (1/2 皆视为高级)。
func isPremiumEngine(engine string) bool {
	em := engineInfo(engine)
	if em == nil {
		return false
	}
	return engineFlagTruthy(em["highLevelFlag"])
}

// fetchLanguageList 拉取语言列表。返回 [(code, name)...]
func fetchLanguageList() ([][2]string, error) {
	c := client.NewBaseClient()
	resp, err := c.GetAPI("core/pdf/lang/list", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	rawList, _ := data["langList"].([]interface{})
	out := make([][2]string, 0, len(rawList))
	for _, l := range rawList {
		if lm, ok := l.(map[string]interface{}); ok {
			out = append(out, [2]string{
				fmt.Sprintf("%v", lm["code"]),
				fmt.Sprintf("%v", lm["name"]),
			})
		}
	}
	return out, nil
}

// selectLanguageInteractive 交互式选择语言 code。
// 严格模式：非交互环境（无 TTY + 无 stdin 输入）下不允许静默默认，直接报 ParamError，
// 避免把文件错翻成列表第一项（通常是 zh-CN）。defaultCode 仅用于 TTY 菜单的初始高亮。
func selectLanguageInteractive(title string, defaultCode string) (string, error) {
	langs, err := fetchLanguageList()
	if err != nil || len(langs) == 0 {
		return "", clierr.NetError("获取语言列表失败: "+fmt.Sprint(err), "可手动指定 --to <语言代码>")
	}
	labels := make([]string, len(langs))
	defaultIdx := 0
	for i, l := range langs {
		labels[i] = fmt.Sprintf("%s  (%s)", l[1], l[0])
		if l[0] == defaultCode {
			defaultIdx = i
		}
	}
	idx, auto, err := selectOptionStrict(title, labels, defaultIdx)
	if err != nil || idx < 0 {
		return "", err
	}
	if auto {
		return "", clierr.ParamError(
			"当前环境无交互终端，无法选择目标语言",
			"请显式指定 --to <语言代码>，如 --to en；查看语言代码: pdf-cli translate languages")
	}
	return langs[idx][0], nil
}

// fetchVisibleEngines 拉取 showFlag=1 的可见引擎元数据。
type visibleEngine struct {
	engineID       string
	engineName     string
	engineShowName string
	tokenCostRatio string
	isPremium      bool
	sortOrder      int
}

func fetchVisibleEngines() ([]visibleEngine, error) {
	c := client.NewBaseClient()
	resp, err := c.GetAPI("core/pdf/engines", nil)
	if err != nil {
		return nil, err
	}
	var engines map[string]interface{}
	if err := json.Unmarshal(resp.Data, &engines); err != nil {
		return nil, err
	}
	out := make([]visibleEngine, 0, len(engines))
	for _, e := range engines {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if !engineFlagTruthy(em["showFlag"]) {
			continue
		}
		sortOrder := 0
		switch x := em["sort"].(type) {
		case float64:
			sortOrder = int(x)
		case int:
			sortOrder = x
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(x)); err == nil {
				sortOrder = n
			}
		}
		out = append(out, visibleEngine{
			engineID:       fmt.Sprintf("%v", em["engineId"]),
			engineName:     fmt.Sprintf("%v", em["engineName"]),
			engineShowName: fmt.Sprintf("%v", em["engineShowName"]),
			tokenCostRatio: fmt.Sprintf("%v", em["tokenCostRatio"]),
			isPremium:      engineFlagTruthy(em["highLevelFlag"]),
			sortOrder:      sortOrder,
		})
	}
	return out, nil
}

func pickTopEnginesByType(engines []visibleEngine, premium bool) []visibleEngine {
	filtered := make([]visibleEngine, 0, len(engines))
	for _, engine := range engines {
		if engine.isPremium == premium {
			filtered = append(filtered, engine)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left := filtered[i]
		right := filtered[j]
		if left.sortOrder != right.sortOrder {
			if left.sortOrder == 0 {
				return false
			}
			if right.sortOrder == 0 {
				return true
			}
			return left.sortOrder < right.sortOrder
		}
		return left.engineName < right.engineName
	})
	if len(filtered) > 3 {
		return filtered[:3]
	}
	return filtered
}

// selectEngineInteractive 交互式选择引擎 name。
func selectEngineInteractive(title string) (string, error) {
	engines, err := fetchVisibleEngines()
	if err != nil || len(engines) == 0 {
		return "", clierr.NetError("获取引擎列表失败或无可用引擎: "+fmt.Sprint(err), "可手动指定 --engine <engineName>")
	}
	typeLabels := []string{"普通引擎", "高级引擎"}
	typeIdx, auto, err := selectOptionStrict("先选引擎类型", typeLabels, 0)
	if err != nil || typeIdx < 0 {
		return "", err
	}
	if auto {
		return "", clierr.ParamError(
			"当前环境无交互终端，无法选择引擎类型",
			"请显式指定 --engine <engineName>；查看引擎列表: pdf-cli translate engines")
	}
	selectedPremium := typeIdx == 1
	candidates := pickTopEnginesByType(engines, selectedPremium)
	if len(candidates) == 0 {
		return "", clierr.ParamError("当前分组无可用引擎", "运行 pdf-cli translate engines 查看完整列表")
	}
	labels := make([]string, 0, len(candidates)+1)
	for _, engine := range candidates {
		ratio := engine.tokenCostRatio
		if ratio == "" || ratio == "<nil>" {
			ratio = "?"
		}
		labels = append(labels, fmt.Sprintf("%s · %sx", engine.engineShowName, ratio))
	}
	labels = append(labels, "other")
	idx, auto, err := selectOptionStrict(title, labels, 0)
	if err != nil || idx < 0 {
		return "", err
	}
	if auto {
		return "", clierr.ParamError(
			"当前环境无交互终端，无法选择翻译引擎",
			"请显式指定 --engine <engineName>；查看引擎列表: pdf-cli translate engines")
	}
	if idx == len(labels)-1 {
		return "", clierr.ParamError(
			"请手动指定引擎名称",
			"使用 --engine <engineName>，或先运行 pdf-cli translate engines 查看完整列表")
	}
	return candidates[idx].engineName, nil
}

// countPDFPages 用系统 pdfinfo 数 PDF 页数。pdfinfo 不在 PATH 时返回 (0, nil)（让上层忽略此项预检）。
func countPDFPages(path string) (int, error) {
	bin, err := exec.LookPath("pdfinfo")
	if err != nil {
		return 0, nil
	}
	out, err := exec.Command(bin, path).Output()
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			n, perr := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
			if perr == nil {
				return n, nil
			}
		}
	}
	return 0, nil
}

// translatePreviewURL 根据 task-id (blobFileName 去掉扩展) 拼出公开预览 URL。
// 游客虽无法走 download 落盘，但可在浏览器中打开此链接"打印"查看译文。
func translatePreviewURL(taskID string) string {
	bases := translateDownloadBases()
	if len(bases) == 0 {
		return ""
	}
	return strings.TrimRight(bases[0], "/") + "/pdf/" + taskID + ".pdf"
}

// printGuestTranslatedText 按新流程图要求，游客完成翻译后在控制台打印所有译文。
// 实现：下载 PDF 到临时路径 → 调用系统 pdftotext 提取 → 打印到 stdout → 清理临时文件。
// pdftotext 不可用或解析失败时，降级为打印预览 URL 让用户在浏览器查看。
func printGuestTranslatedText(taskID string) error {
	previewURL := translatePreviewURL(taskID)

	pdftotextBin, err := exec.LookPath("pdftotext")
	if err != nil {
		fmt.Println("未检测到 pdftotext（poppler-utils），无法在控制台打印译文。")
		if previewURL != "" {
			fmt.Printf("请在浏览器中打开预览 URL: %s\n", previewURL)
		}
		return nil
	}

	tmpPDF, err := os.CreateTemp("", "pdf-cli-trans-*.pdf")
	if err != nil {
		return clierr.New(clierr.ExitInternal, clierr.TypeInternal, "创建临时文件失败: "+err.Error(), "", false)
	}
	tmpPath := tmpPDF.Name()
	_ = tmpPDF.Close()
	defer os.Remove(tmpPath)

	blobFileName := taskID + ".pdf"
	downloaded := false
	for _, base := range translateDownloadBases() {
		dlURL := strings.TrimRight(base, "/") + "/pdf/" + blobFileName
		resp, err := http.Get(dlURL)
		if err != nil {
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			continue
		}
		f, err := os.Create(tmpPath)
		if err != nil {
			resp.Body.Close()
			return clierr.NetError("写入临时文件失败: "+err.Error(), "")
		}
		_, copyErr := io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		if copyErr == nil {
			downloaded = true
			break
		}
	}
	if !downloaded {
		if previewURL != "" {
			fmt.Printf("无法获取译文 PDF，请在浏览器中打开: %s\n", previewURL)
		}
		return clierr.NotFoundError("所有下载源均失败", "task-id 可能还未完成或已过期")
	}

	cmd := exec.Command(pdftotextBin, "-layout", "-q", tmpPath, "-")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "pdftotext 执行失败: "+err.Error())
		if previewURL != "" {
			fmt.Printf("请在浏览器中打开预览 URL: %s\n", previewURL)
		}
		return nil
	}
	return nil
}

// printGuestNotice 按流程图要求，免费/游客流程前提示身份与功能限制。
func printGuestNotice(hp *userconfig.Homepage) {
	if auth.IsLoggedIn() {
		fmt.Println("[免费流程] 当前为登录用户 — 所有用户（游客/普通/会员）均可使用免费翻译。")
	} else {
		fmt.Println("[游客模式] 未登录 — 将以游客身份走免费翻译流程。")
		fmt.Println("  · 翻译结果仅可在线查看，不可通过 CLI 下载保存")
		fmt.Println("  · 如需下载结果，请先执行 pdf-cli auth login --email you@example.com")
	}
	if hp == nil {
		return
	}
	if hp.FreeTransLimitNumPerDay > 0 {
		fmt.Printf("  · 免费翻译今日次数: %.0f/%.0f\n",
			hp.FreeTransYetUsedNum, hp.FreeTransLimitNumPerDay)
	}
	if hp.FreeTransMaxFileSize > 0 {
		// 后端 freeTransMaxFileSize 的单位是 MB，换算成字节后再打印
		fmt.Printf("  · 免费翻译文件大小上限: %s\n", userconfig.HumanBytes(float64(hp.FreeTransMaxFileSize)*1024*1024))
	}
	if hp.FreeTransMaxPages > 0 {
		fmt.Printf("  · 免费翻译页数上限: %.0f 页\n", hp.FreeTransMaxPages)
	}
}

// printStartNotice 按流程图要求，高级翻译流程前提示会员/非会员限制与剩余配额。
func printStartNotice(hp *userconfig.Homepage, vip int) {
	if vip > 0 {
		fmt.Println("[高级翻译] 会员用户 — 可选用任意翻译引擎（普通/高级）。")
	} else {
		fmt.Println("[高级翻译] 普通用户 — 仅可选用普通引擎，并受每日次数/字符数限制。")
	}
	if hp == nil {
		return
	}
	if hp.RemainTransCountByDay > 0 || hp.RemainCharsCountByDay > 0 {
		fmt.Printf("  · 今日剩余次数: %.0f    今日剩余字符: %.0f\n",
			hp.RemainTransCountByDay, hp.RemainCharsCountByDay)
	}
}

// quotaIsExhausted 解析后端 remain* 字段：null → 无限制 (false)，
// 显式 0 / "0" → 已耗尽 (true)，>0 → 有剩余 (false)。
func quotaIsExhausted(v interface{}) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case float64:
		return x == 0
	case int:
		return x == 0
	case string:
		if x == "" {
			return false
		}
		n, err := strconv.ParseFloat(x, 64)
		return err == nil && n == 0
	case bool:
		return !x
	}
	return false
}

func translatePrecheck(opts precheckOpts) error {
	hp, err := userconfig.Fetch()
	if err != nil || hp == nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 获取用户限制条件失败，跳过预检: %v\n", err)
		}
		return nil
	}

	// 游客进行中任务检查：upload/free/start 均适用（后端对游客限同时一个任务）
	if !auth.IsLoggedIn() {
		if busy, name := hp.VisitorHasInProgress(); busy {
			label := "当前文件"
			if name != "" {
				label = name
			}
			return clierr.QuotaError(
				fmt.Sprintf("游客已有翻译中的文件: %s", label),
				"请等待该任务完成（pdf-cli translate status），或登录后用会员额度翻译")
		}
	}
	// 卡翻译警告（仅提示，不阻断）
	if warn, msg := hp.StuckTransWarn(); warn && msg != "" {
		fmt.Fprintln(os.Stderr, "警告: 存在卡翻译中的文件 "+msg)
	}

	switch opts.kind {
	case "upload-free":
		// 后端 freeTransMaxFileSize 单位是 MB，换算为字节后再与文件实际字节数比较
		maxBytes := float64(hp.FreeTransMaxFileSize) * 1024 * 1024
		if maxBytes > 0 && opts.fileSize > 0 && float64(opts.fileSize) > maxBytes {
			return clierr.QuotaError(
				fmt.Sprintf("免费翻译文件大小超限: 文件 %s > 上限 %s",
					userconfig.HumanBytes(float64(opts.fileSize)), userconfig.HumanBytes(maxBytes)),
				"请升级会员或使用更小的文件")
		}
		// 页数预检：需要系统 pdfinfo，缺失则跳过
		if hp.FreeTransMaxPages > 0 && opts.filePath != "" {
			if pages, _ := countPDFPages(opts.filePath); pages > 0 && pages > int(hp.FreeTransMaxPages) {
				return clierr.QuotaError(
					fmt.Sprintf("免费翻译页数超限: 文档 %d 页 > 上限 %.0f 页", pages, float64(hp.FreeTransMaxPages)),
					"请升级会员或拆分文档后再试")
			}
		}
	case "free":
		if hp.FreeTransLimitNumPerDay > 0 && hp.FreeTransYetUsedNum >= hp.FreeTransLimitNumPerDay {
			return clierr.QuotaError(
				fmt.Sprintf("免费翻译今日次数已用尽: %.0f/%.0f",
					hp.FreeTransYetUsedNum, hp.FreeTransLimitNumPerDay),
				"请明日再试，或使用 translate start 以会员身份翻译")
		}
	case "text":
		// AI 文本翻译预检：游客看 visitorRemainAiTransCount，用户看 aiTransTrialLimitNum*
		if !auth.IsLoggedIn() {
			if v, ok := hp.Raw["visitorRemainAiTransCount"]; ok && v != nil {
				if x, ok := v.(float64); ok && x <= 0 {
					return clierr.QuotaError(
						"游客 AI 翻译今日试用次数已用尽",
						"请登录后继续使用；会员享更高额度")
				}
			}
		} else {
			// aiTransTrialLimitNumPerDay / PerMon：null 无限制；0 耗尽；>0 有剩余
			if quotaIsExhausted(hp.Raw["aiTransTrialLimitNumPerDay"]) {
				return clierr.QuotaError(
					"AI 翻译今日试用次数已用尽",
					"请明日再试或升级会员")
			}
			if quotaIsExhausted(hp.Raw["aiTransTrialLimitNumPerMon"]) {
				return clierr.QuotaError(
					"AI 翻译本月试用次数已用尽",
					"请下月再试或升级会员")
			}
		}
	case "start", "arxiv":
		// 区分三种状态：null = 无限制（VIP/SVIP），0 = 已耗尽，>0 = 有剩余
		// 只有两项都"显式为 0"才判定耗尽；null 视为无限制直接放行
		transExhausted := quotaIsExhausted(hp.Raw["remainTransCountByDay"])
		charsExhausted := quotaIsExhausted(hp.Raw["remainCharsCountByDay"])
		if transExhausted && charsExhausted {
			return clierr.QuotaError(
				"今日翻译额度已用尽 (次数/字符均为 0)",
				"请等待每日重置、充值加油包或升级会员")
		}
		if opts.engine != "" {
			em := engineInfo(opts.engine)
			if em == nil {
				return clierr.ParamError(
					"引擎不存在或不可用: "+opts.engine,
					"使用 pdf-cli translate engines 查看可用引擎列表")
			}
			if !isEngineAvailable(em) {
				return clierr.ParamError(
					"引擎当前不对外开放: "+opts.engine,
					"使用 pdf-cli translate engines 查看可用引擎列表")
			}
			// 高级引擎仅会员可用 (普通用户/游客 仅普通引擎)
			if engineFlagTruthy(em["highLevelFlag"]) && !isMemberUser() {
				return clierr.QuotaError(
					"高级引擎仅会员可用",
					"请升级会员后使用 --engine 指定高级引擎，或使用普通引擎 / 默认引擎")
			}
		}
	}
	return nil
}

func translateRecordList(resp *client.APIResponse) []interface{} {
	var listData []interface{}
	if resp.DataList != nil {
		json.Unmarshal(resp.DataList, &listData)
	}
	if listData == nil && resp.Data != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(resp.Data, &data); err == nil {
			if dl, ok := data["dataList"]; ok {
				if arr, ok := dl.([]interface{}); ok {
					listData = arr
				}
			}
		}
	}
	if listData == nil && resp.List != nil {
		json.Unmarshal(resp.List, &listData)
	}
	return listData
}

func translateTaskKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.LastIndex(raw, "."); idx > 0 {
		return raw[:idx]
	}
	return raw
}

// translateExtractBlobFromDownInfo 从 user/operate/record/down/info 响应数据里
// 按优先级抽取 blob 文件名：noWater > hasWater > 历史字段 blobFileName。
func translateExtractBlobFromDownInfo(data map[string]interface{}) string {
	for _, key := range []string{"noWaterBlobFileName", "hasWaterBlobFileName", "blobFileName"} {
		if v, ok := data[key]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

// translateResolveBlobFile 把 task-id（数字 record-id 或 blob 字符串）解析成可用于
// CDN 下载的 blobFileName（含 .pdf）。数字 id 走 down/info，字符串视作已是 blob key。
func translateResolveBlobFile(taskID string) (string, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", clierr.ParamError("请提供 task-id", "使用 --task-id 参数")
	}
	isNumeric := true
	for _, ch := range taskID {
		if ch < '0' || ch > '9' {
			isNumeric = false
			break
		}
	}
	if !isNumeric {
		if strings.HasSuffix(taskID, ".pdf") {
			return taskID, nil
		}
		return taskID + ".pdf", nil
	}

	c := client.NewBaseClient()
	resp, err := c.GetAPI("user/operate/record/down/info", map[string]string{
		"operateRecordId": taskID,
	})
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", err
	}
	blob := translateExtractBlobFromDownInfo(data)
	if blob == "" {
		return "", clierr.NotFoundError("未取到 blob 文件名", "task-id 可能未生成翻译记录或已过期")
	}
	return blob, nil
}

// translateProbeBlobReady 对各下载源做 HEAD 请求；任一返回 2xx 即视为译文已生成。
// 这是目前最可靠的"翻译完成"信号，因为后端 core/pdf/query/status 当前总返回空对象。
func translateProbeBlobReady(blobFileName string) bool {
	if blobFileName == "" {
		return false
	}
	bases := translateDownloadBases()
	for _, base := range bases {
		req, err := http.NewRequest("HEAD", strings.TrimRight(base, "/")+"/pdf/"+blobFileName, nil)
		if err != nil {
			continue
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true
		}
	}
	return false
}

func translateResolveRecordID(taskID string) (string, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", clierr.ParamError("请提供 task-id", "使用 --task-id 参数")
	}
	isNumeric := true
	for _, ch := range taskID {
		if ch < '0' || ch > '9' {
			isNumeric = false
			break
		}
	}
	if isNumeric {
		return taskID, nil
	}

	c := client.NewBaseClient()
	resp, err := c.PostJSONAPI("user/operate/record/list/page", map[string]interface{}{
		"pageNo":   1,
		"pageSize": 100,
	})
	if err != nil {
		return "", err
	}

	for _, item := range translateRecordList(resp) {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if translateTaskKey(fmt.Sprintf("%v", m["blobFileName"])) == taskID {
			return fmt.Sprintf("%v", m["id"]), nil
		}
	}

	return "", clierr.ParamError("未找到对应翻译记录", "请使用 start 返回的 record-id，或先执行 pdf-cli translate history")
}

var translateLanguagesCmd = &cobra.Command{
	Use:   "languages",
	Short: "获取支持的语言列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewBaseClient()
		resp, err := c.GetAPI("core/pdf/lang/list", nil)
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		if format == "json" {
			var data interface{}
			json.Unmarshal(resp.Data, &data)
			output.PrintJSON(data)
			return nil
		}

		var data map[string]interface{}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			output.PrintJSON(json.RawMessage(resp.Data))
			return nil
		}

		if langList, ok := data["langList"]; ok {
			if langs, ok := langList.([]interface{}); ok {
				headers := []string{"语言代码", "语言名称"}
				var rows [][]string
				for _, l := range langs {
					if lm, ok := l.(map[string]interface{}); ok {
						code := fmt.Sprintf("%v", lm["code"])
						name := fmt.Sprintf("%v", lm["name"])
						rows = append(rows, []string{code, name})
					}
				}
				output.PrintTable(headers, rows)
				return nil
			}
		}

		output.PrintJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var translateEnginesCmd = &cobra.Command{
	Use:   "engines",
	Short: "获取可用翻译引擎列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewBaseClient()
		resp, err := c.GetAPI("core/pdf/engines", nil)
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		if format == "json" {
			var data interface{}
			json.Unmarshal(resp.Data, &data)
			output.PrintJSON(data)
			return nil
		}

		var engines map[string]interface{}
		if err := json.Unmarshal(resp.Data, &engines); err != nil {
			output.PrintJSON(json.RawMessage(resp.Data))
			return nil
		}

		headers := []string{"引擎 ID", "引擎名称", "显示名称", "等级"}
		var rows [][]string
		for _, e := range engines {
			em, ok := e.(map[string]interface{})
			if !ok {
				continue
			}
			// showFlag = 1 表示对当前用户可用；0 的引擎对外不开放
			if !engineFlagTruthy(em["showFlag"]) {
				continue
			}
			id := fmt.Sprintf("%v", em["engineId"])
			name := fmt.Sprintf("%v", em["engineName"])
			showName := fmt.Sprintf("%v", em["engineShowName"])
			level := "普通引擎"
			if engineFlagTruthy(em["highLevelFlag"]) {
				level = "高级引擎"
			}
			rows = append(rows, []string{id, name, showName, level})
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

func translateDownloadBases() []string {
	cfg := config.Load()
	primary := strings.TrimRight(cfg.GetDownloadURL(), "/")
	bases := []string{primary}
	fallbacks := []string{"https://res.gdpdf.com", "https://res.doclingo.ai"}
	for _, fb := range fallbacks {
		if fb != primary {
			bases = append(bases, fb)
		}
	}
	return bases
}

func translateDownloadFile(baseURL, blobFileName, outputPath string) error {
	fmt.Printf("正在下载到: %s\n", outputPath)

	bases := translateDownloadBases()
	// 如果调用方传入了明确的 baseURL 且不在列表中，优先使用
	if baseURL != "" {
		trimmed := strings.TrimRight(baseURL, "/")
		found := false
		for _, b := range bases {
			if b == trimmed {
				found = true
				break
			}
		}
		if !found {
			bases = append([]string{trimmed}, bases...)
		}
	}

	for _, base := range bases {
		dlURL := base + "/pdf/" + blobFileName
		resp, err := http.Get(dlURL)
		if err != nil {
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			continue
		}

		f, err := os.Create(outputPath)
		if err != nil {
			resp.Body.Close()
			return clierr.ParamError("无法创建输出文件: "+outputPath, "")
		}
		_, copyErr := io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		if copyErr != nil {
			return clierr.NetError("写入文件失败: "+copyErr.Error(), "")
		}

		output.PrintSuccess(fmt.Sprintf("下载完成: %s", outputPath))
		return nil
	}

	return clierr.NotFoundError("所有下载源均失败", "task-id 可能已过期或不存在，先用 history 确认")
}

func translateFileMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func translateUploadToAWS(awsURL, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return clierr.ParamError("无法打开文件: "+filePath, "请检查文件路径")
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return clierr.ParamError("无法读取文件: "+filePath, "请检查文件路径")
	}

	req, err := http.NewRequest(http.MethodPut, awsURL, f)
	if err != nil {
		return clierr.NetError("创建上传请求失败: "+err.Error(), "")
	}
	req.ContentLength = info.Size()
	req.TransferEncoding = nil
	req.Header.Set("Content-Type", "application/pdf")

	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
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

// doUpload 执行"AWS 预签名 → S3 直传 → 注册"三步上传，返回 file-key。
// freeTag: 0=登录态会员通道；1=免费/游客通道。
func doUpload(filePath string, freeTag int) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", clierr.ParamError("无法读取文件: "+filePath, "请检查文件路径")
	}

	c := client.NewBaseClient()

	fmt.Println("正在准备上传...")
	resp, err := c.PostJSONAPI("core/pdf/trans/aws/pre/upload", map[string]interface{}{
		"fileRealName": filepath.Base(filePath),
		"freeTag":      freeTag,
	})
	if err != nil {
		return "", err
	}

	var prepData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &prepData); err != nil {
		return "", clierr.New(clierr.ExitInternal, clierr.TypeInternal, "解析上传准备响应失败: "+err.Error(), "", false)
	}
	awsURL := fmt.Sprintf("%v", prepData["awsUploadUrl"])
	blobFileName := fmt.Sprintf("%v", prepData["blobFileName"])
	if awsURL == "" || blobFileName == "" {
		return "", clierr.New(clierr.ExitInternal, clierr.TypeInternal, "上传准备响应不完整: 缺少 awsUploadUrl/blobFileName", "", false)
	}

	fmt.Println("正在上传文件...")
	if err := translateUploadToAWS(awsURL, filePath); err != nil {
		return "", err
	}

	fileMD5, _ := translateFileMD5(filePath)

	fmt.Println("正在注册文件...")
	resp2, err := c.PostJSONAPI("core/pdf/file/new/upload", map[string]interface{}{
		"blobFileName":  blobFileName,
		"fileMd5String": fileMD5,
		"fileRealName":  filepath.Base(filePath),
		"fileSize":      info.Size(),
		"freeTag":       freeTag,
	})
	if err != nil {
		return "", err
	}

	fileKey := blobFileName
	var data map[string]interface{}
	if err := json.Unmarshal(resp2.Data, &data); err == nil {
		if k, ok := data["tmpFileName"]; ok {
			fileKey = fmt.Sprintf("%v", k)
		} else if k, ok := data["key"]; ok {
			fileKey = fmt.Sprintf("%v", k)
		}
	}
	fmt.Printf("上传成功\n  file-key : %s\n", fileKey)
	return fileKey, nil
}

// runFreeTranslate 发起免费翻译：含游客身份提示、语言交互选择、预检、API 调用与输出。
// 被 translate upload（交互接力）、translate free（直接命令）共用。
func runFreeTranslate(fileKey, toLang, fromLang, fileFormat, pages, visitorEmail string) error {
	if fileKey == "" {
		return clierr.ParamError("请提供 file-key", "先执行 pdf-cli translate upload --file xxx")
	}

	hp, _ := userconfig.Fetch()
	printGuestNotice(hp)

	// 流程图：选择目标语言（选项来自 core/pdf/lang/list）
	if toLang == "" {
		picked, perr := selectLanguageInteractive("选择目标语言", "zh")
		if perr != nil {
			return perr
		}
		if picked == "" {
			return clierr.ParamError("请提供目标语言", "使用 --to 参数，如 --to en；或在交互模式下选择语言")
		}
		toLang = picked
	}

	if err := translatePrecheck(precheckOpts{kind: "free"}); err != nil {
		return err
	}

	data := map[string]interface{}{
		"fileKey":    fileKey,
		"targetLang": toLang,
		"sourceLang": nil,
	}
	if fromLang != "" {
		data["sourceLang"] = fromLang
	}
	if fileFormat != "" {
		data["fileFmtType"] = fileFormat
	} else {
		data["fileFmtType"] = "pdf"
	}
	if pages != "" {
		data["pages"] = pages
	}
	if visitorEmail != "" {
		data["visitorEmail"] = visitorEmail
	}

	c := client.NewBaseClient()
	resp, err := c.PostJSONAPI("core/pdf/free/translate", data)
	if err != nil {
		return err
	}

	format := output.GetFormat(formatFlag)
	if format == "json" {
		var result interface{}
		json.Unmarshal(resp.Data, &result)
		output.PrintJSON(result)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err == nil {
		if blobFile, ok := result["blobFileName"]; ok {
			blobStr := fmt.Sprintf("%v", blobFile)
			taskID := translateTaskKey(blobStr)
			recordID := result["id"]
			fmt.Printf("免费翻译已发起\n  task-id    : %s\n", taskID)
			if recordID != nil {
				fmt.Printf("  record-id  : %v\n", recordID)
			}
			if queue, ok := result["existFreeTransCount"]; ok {
				fmt.Printf("  排队数量   : %v\n", queue)
			}
			if previewURL := translatePreviewURL(taskID); previewURL != "" {
				fmt.Printf("  译文预览 URL: %s  (翻译完成后在浏览器中打开可查看/打印)\n", previewURL)
			}
			fmt.Println("\n下一步: pdf-cli translate status --task-id " + taskID + " --wait")
			return nil
		}
	}
	output.PrintSuccess("免费翻译已发起")
	output.PrintJSON(json.RawMessage(resp.Data))
	return nil
}

// runAdvancedTranslate 发起高级翻译（登录 + 会员/普通引擎 + 可选 OCR）。
// ocrExplicit 用于区分"用户未选（需询问）"与"用户选了否"。
func runAdvancedTranslate(fileKey, toLang, fromLang, engine string, ocrExplicit, ocrFlag bool, termIDs, fileFormat string, promptType int) error {
	if !auth.IsLoggedIn() {
		return clierr.AuthError(
			"高级翻译需要登录",
			"请先执行 pdf-cli auth login --email you@example.com；或改走 pdf-cli translate free 免费流程")
	}
	if fileKey == "" {
		return clierr.ParamError("请提供 file-key", "先执行 pdf-cli translate upload --file xxx")
	}

	vip := fetchVipLevel()
	hp, _ := userconfig.Fetch()
	printStartNotice(hp, vip)

	// 流程图：选择目标语言
	if toLang == "" {
		picked, perr := selectLanguageInteractive("选择目标语言", "zh")
		if perr != nil {
			return perr
		}
		if picked == "" {
			return clierr.ParamError("请提供目标语言", "使用 --to 参数，如 --to en；或在交互模式下选择语言")
		}
		toLang = picked
	}
	// 流程图：先选引擎类型，再选具体引擎（普通/高级各展示前 3 个 + other）
	if engine == "" {
		picked, perr := selectEngineInteractive("选择具体引擎")
		if perr != nil {
			return perr
		}
		engine = picked
	}
	// 流程图：是否使用 OCR 功能？
	if !ocrExplicit {
		yes, _ := selectYesNo("是否启用 OCR 功能？（扫描件/图片型 PDF 建议启用）", false)
		ocrFlag = yes
	}

	if err := translatePrecheck(precheckOpts{kind: "start", engine: engine, ocr: ocrFlag}); err != nil {
		return err
	}

	ocrVal := 0
	if ocrFlag {
		ocrVal = 1
	}
	data := map[string]interface{}{
		"fileKey":    fileKey,
		"targetLang": toLang,
		"ocrFlag":    ocrVal,
	}
	if fromLang != "" {
		data["sourceLang"] = fromLang
	}
	if engine != "" {
		data["transEngineType"] = engine
	}
	if termIDs != "" {
		data["termIds"] = termIDs
	}
	if fileFormat != "" {
		data["fileFmtType"] = fileFormat
	}
	if promptType != 0 {
		data["promptType"] = promptType
	}

	c := client.NewBaseClient()
	resp, err := c.PostJSONAPI("core/pdf/translate", data)
	if err != nil {
		return err
	}

	format := output.GetFormat(formatFlag)
	if format == "json" {
		var result interface{}
		json.Unmarshal(resp.Data, &result)
		output.PrintJSON(result)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err == nil {
		if blobFile, ok := result["blobFileName"]; ok {
			blobStr := fmt.Sprintf("%v", blobFile)
			taskID := translateTaskKey(blobStr)
			recordID := result["id"]
			fmt.Printf("翻译已发起\n  task-id    : %s\n", taskID)
			if recordID != nil {
				fmt.Printf("  record-id  : %v\n", recordID)
			}
			fmt.Println("\n下一步: pdf-cli translate status --task-id " + taskID + " --wait")
			return nil
		}
	}
	output.PrintSuccess("翻译已发起")
	output.PrintJSON(json.RawMessage(resp.Data))
	return nil
}

var translateUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "上传待翻译文件",
	Long: `PDF 翻译总入口。按流程图先根据登录态分流再上传：

  · 已登录 → 询问"是否使用免费翻译？"（--free / --advanced 可跳过）
      是 → 上传（freeTag=1）→ 选择目标语言 → 使用免费 CLI
      否 → 上传（freeTag=0）→ 选择目标语言 + 引擎 + OCR → 使用高级 CLI
  · 未登录 → 询问"是否使用高级翻译？"（--free / --advanced 可跳过）
      是 → 提示 pdf-cli auth login --email ... 并退出
      否 → 提示游客身份 → 上传 → 选择目标语言 → 使用免费 CLI

翻译完成后：
  · 登录用户：运行 pdf-cli translate download 下载到本地
  · 游客：translate status --wait 或 translate download 自动控制台打印译文

Agent/CI 非交互场景推荐全显式：
  pdf-cli translate upload --file ./paper.pdf --free --to en
  pdf-cli translate upload --file ./paper.pdf --advanced --to zh --engine google --ocr

示例：
  pdf-cli translate upload --file ./paper.pdf`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		// 非交互透传 flag（Agent CLI / CI / 脚本场景用）：提供时跳过对应菜单
		toLang, _ := cmd.Flags().GetString("to")
		engine, _ := cmd.Flags().GetString("engine")
		ocrFlag, _ := cmd.Flags().GetBool("ocr")
		ocrExplicit := cmd.Flags().Changed("ocr")
		freeFlag, _ := cmd.Flags().GetBool("free")
		advancedFlag, _ := cmd.Flags().GetBool("advanced")

		if filePath == "" {
			return clierr.ParamError("请提供文件路径", "使用 --file 参数")
		}
		if _, err := os.Stat(filePath); err != nil {
			return clierr.ParamError("无法读取文件: "+filePath, "请检查文件路径")
		}
		if freeFlag && advancedFlag {
			return clierr.ParamError("--free 与 --advanced 不能同时使用", "二选一")
		}

		loggedIn := auth.IsLoggedIn()

		// 决定走"高级"还是"免费"——优先级：flag > 交互菜单 > 登录态默认
		var useAdvanced bool
		switch {
		case advancedFlag:
			useAdvanced = true
		case freeFlag:
			useAdvanced = false
		case loggedIn:
			// 已登录 → 问"是否使用免费翻译？"（选否 = 高级）
			useFree, perr := selectYesNo("是否使用免费翻译？（否则进入高级翻译流程，消耗会员/普通用户额度）", false)
			if perr != nil {
				return clierr.ParamError("已取消", "")
			}
			useAdvanced = !useFree
		default:
			// 未登录 → 问"是否使用高级翻译？"
			ua, perr := selectYesNo("是否使用高级翻译？（需要登录；否则走默认免费流程）", false)
			if perr != nil {
				return clierr.ParamError("已取消", "")
			}
			useAdvanced = ua
		}

		if useAdvanced && !loggedIn {
			// 构造原始命令回显，便于用户登录后直接重跑
			rerun := "pdf-cli translate upload --file " + filePath + " --advanced"
			if toLang != "" {
				rerun += " --to " + toLang
			}
			if engine != "" {
				rerun += " --engine " + engine
			}
			if ocrFlag {
				rerun += " --ocr"
			}

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "╭─────────────────────────────────────────────────────────────╮")
			fmt.Fprintln(os.Stderr, "│ 高级翻译需要登录                                              │")
			fmt.Fprintln(os.Stderr, "├─────────────────────────────────────────────────────────────┤")
			fmt.Fprintln(os.Stderr, "│ 请按以下步骤操作：                                            │")
			fmt.Fprintln(os.Stderr, "│                                                             │")
			fmt.Fprintln(os.Stderr, "│ ① 登录（将提示输入密码）：                                    │")
			fmt.Fprintln(os.Stderr, "│    pdf-cli auth login --email you@example.com               │")
			fmt.Fprintln(os.Stderr, "│                                                             │")
			fmt.Fprintln(os.Stderr, "│ ② 重跑翻译：                                                 │")
			fmt.Fprintf(os.Stderr, "│    %s\n", rerun)
			fmt.Fprintln(os.Stderr, "│                                                             │")
			fmt.Fprintln(os.Stderr, "│ 或改走免费流程：在命令行把 --advanced 替换为 --free           │")
			fmt.Fprintln(os.Stderr, "╰─────────────────────────────────────────────────────────────╯")
			fmt.Fprintln(os.Stderr, "")

			return clierr.AuthError(
				"高级翻译需要登录",
				"请先执行 pdf-cli auth login --email you@example.com；登录后重跑上面显示的命令")
		}

		// 按选择执行：freeTag → upload → 接力翻译
		freeTag := 1
		if useAdvanced {
			freeTag = 0
		}

		if !useAdvanced && !loggedIn {
			// 游客免费流程：提前提示身份 + 文件大小/页数预检
			hp, _ := userconfig.Fetch()
			printGuestNotice(hp)
			info, _ := os.Stat(filePath)
			if err := translatePrecheck(precheckOpts{
				kind:     "upload-free",
				fileSize: info.Size(),
				filePath: filePath,
			}); err != nil {
				return err
			}
		}

		fileKey, err := doUpload(filePath, freeTag)
		if err != nil {
			return err
		}
		fmt.Println()
		if useAdvanced {
			return runAdvancedTranslate(fileKey, toLang, "", engine, ocrExplicit, ocrFlag, "", "", 0)
		}
		return runFreeTranslate(fileKey, toLang, "", "", "", "")
	},
}

var translateStartCmd = &cobra.Command{
	Use:   "start",
	Short: "发起翻译",
	Long: `发起 PDF 翻译任务。

示例：
  pdf-cli translate start --file-key xxx --to zh
  pdf-cli translate start --file-key xxx --to zh --from en --engine google --ocr`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileKey, _ := cmd.Flags().GetString("file-key")
		toLang, _ := cmd.Flags().GetString("to")
		fromLang, _ := cmd.Flags().GetString("from")
		engine, _ := cmd.Flags().GetString("engine")
		ocrFlag, _ := cmd.Flags().GetBool("ocr")
		ocrExplicit := cmd.Flags().Changed("ocr")
		termIDs, _ := cmd.Flags().GetString("term-ids")
		fileFormat, _ := cmd.Flags().GetString("file-format")
		promptType, _ := cmd.Flags().GetInt("prompt-type")
		return runAdvancedTranslate(fileKey, toLang, fromLang, engine, ocrExplicit, ocrFlag, termIDs, fileFormat, promptType)
	},
}

var translateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询翻译进度",
	Long: `查询翻译任务的当前状态和进度。

示例：
  pdf-cli translate status --task-id xxx
  pdf-cli translate status --task-id xxx --wait`,
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, _ := cmd.Flags().GetString("task-id")
		wait, _ := cmd.Flags().GetBool("wait")

		if taskID == "" {
			return clierr.ParamError("请提供 task-id", "使用 --task-id 参数")
		}

		c := client.NewBaseClient()

		// 后端 core/pdf/query/status 当前对很多任务返回空对象；
		// 用 CDN HEAD 探测 blob 是否就绪作为 fallback 完成信号。
		// 解析失败不阻断流程（例如未登录时 down/info 不可用），仅跳过 fallback。
		blobFile, _ := translateResolveBlobFile(taskID)

		for {
			resp, err := c.GetAPI("core/pdf/query/status", map[string]string{
				"queryFileKey": taskID,
			})
			if err != nil {
				return err
			}

			format := output.GetFormat(formatFlag)
			if format == "json" {
				var data interface{}
				json.Unmarshal(resp.Data, &data)
				output.PrintJSON(data)
				if !wait {
					return nil
				}
			}

			var data map[string]interface{}
			json.Unmarshal(resp.Data, &data)

			statusData := data
			if inner, ok := data[taskID]; ok {
				if innerMap, ok := inner.(map[string]interface{}); ok {
					statusData = innerMap
				}
			}

			rawStatus, _ := statusData["status"]
			statusStr := ""
			if rawStatus != nil {
				statusStr = strings.TrimSpace(fmt.Sprintf("%v", rawStatus))
			}
			status := statusStr // for messages
			if status == "" {
				status = "<未知>"
			}
			rate := statusData["translateRate"]

			done := statusStr == "done" || statusStr == "success" || statusStr == "finish" || statusStr == "finished"
			cdnReady := false
			if !done && statusStr != "fail" && statusStr != "cancel" {
				cdnReady = translateProbeBlobReady(blobFile)
				if cdnReady {
					done = true
					status = "done (CDN 探测)"
				}
			}

			if format != "json" {
				fmt.Printf("  状态   : %s\n", status)
				if rate != nil {
					fmt.Printf("  进度   : %v%%\n", rate)
				}
				if reason, ok := statusData["failReason"]; ok && reason != nil && fmt.Sprintf("%v", reason) != "" && fmt.Sprintf("%v", reason) != "0" {
					fmt.Printf("  失败原因: %v\n", reason)
				}
			}

			terminal := done || statusStr == "fail" || statusStr == "cancel"
			if !wait || terminal {
				if done {
					// 流程图：登录用户可下载到本地；游客在控制台打印所有译文
					if auth.IsLoggedIn() {
						fmt.Println("\n下一步: pdf-cli translate download --task-id " + taskID)
					} else {
						fmt.Println("\n翻译已完成（游客模式 — 在控制台打印译文内容）：")
						fmt.Println(strings.Repeat("-", 60))
						if err := printGuestTranslatedText(taskID); err != nil {
							return err
						}
						fmt.Println(strings.Repeat("-", 60))
						fmt.Println("（如需保存为本地文件，请先执行 pdf-cli auth login --email you@example.com）")
					}
					return nil
				}
				if statusStr == "fail" || statusStr == "cancel" {
					reason := ""
					if r, ok := statusData["failReason"]; ok && r != nil {
						reason = fmt.Sprintf("%v", r)
					}
					msg := "翻译任务终止: status=" + statusStr
					if reason != "" && reason != "0" {
						msg += " reason=" + reason
					}
					e := clierr.TaskFailedError(msg, "查看 history 或重新发起 upload→start").
						WithDetail("task_id", taskID).
						WithDetail("status", statusStr)
					if reason != "" && reason != "0" {
						e = e.WithDetail("fail_reason", reason)
					}
					return e
				}
				return nil
			}

			fmt.Println("  等待中...")
			time.Sleep(5 * time.Second)
		}
	},
}

var translateContinueCmd = &cobra.Command{
	Use:   "continue",
	Short: "继续翻译",
	Long: `继续一个暂停的翻译任务。

示例：
  pdf-cli translate continue --task-id xxx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		taskID, _ := cmd.Flags().GetString("task-id")
		if taskID == "" {
			return clierr.ParamError("请提供 task-id", "使用 --task-id 参数")
		}

		recordID, err := translateResolveRecordID(taskID)
		if err != nil {
			return err
		}

		c := client.NewBaseClient()
		_, err = c.PostJSONAPI("core/pdf/trans/continue", map[string]interface{}{
			"operateRecordId": recordID,
		})
		if err != nil {
			return err
		}

		output.PrintSuccess("翻译已继续")
		return nil
	},
}

var translateCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "取消翻译",
	Long: `取消一个进行中的翻译任务。

示例：
  pdf-cli translate cancel --task-id xxx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		taskID, _ := cmd.Flags().GetString("task-id")
		if taskID == "" {
			return clierr.ParamError("请提供 task-id", "使用 --task-id 参数")
		}

		recordID, err := translateResolveRecordID(taskID)
		if err != nil {
			return err
		}

		c := client.NewBaseClient()
		_, err = c.PostJSONAPI("core/pdf/cancel/trans", map[string]interface{}{
			"operateRecordId": recordID,
		})
		if err != nil {
			return err
		}

		output.PrintSuccess("翻译已取消")
		return nil
	},
}

var translateDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "下载翻译结果",
	Long: `下载已完成的翻译结果文件（支持免费用户和会员用户）。

示例：
  pdf-cli translate download --task-id xxx
  pdf-cli translate download --task-id xxx --output ./result.pdf`,
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, _ := cmd.Flags().GetString("task-id")
		outputPath, _ := cmd.Flags().GetString("output")

		if taskID == "" {
			return clierr.ParamError("请提供 task-id", "使用 --task-id 参数")
		}

		// 流程图：游客不能下载到本地，只会在控制台打印所有译文
		if !auth.IsLoggedIn() {
			fmt.Println("游客模式 — 在控制台打印译文内容：")
			fmt.Println(strings.Repeat("-", 60))
			if err := printGuestTranslatedText(taskID); err != nil {
				return err
			}
			fmt.Println(strings.Repeat("-", 60))
			fmt.Println("（如需保存为本地文件，请先执行 pdf-cli auth login --email you@example.com）")
			return nil
		}

		c := client.NewBaseClient()
		cfg := config.Load()
		downloadBaseURL := strings.TrimRight(cfg.GetDownloadURL(), "/")

		// 尝试通过 record-id 方式获取下载信息（已登录用户）
		if auth.IsLoggedIn() {
			recordID, err := translateResolveRecordID(taskID)
			if err == nil {
				resp, err := c.GetAPI("user/operate/record/down/info", map[string]string{
					"operateRecordId": recordID,
				})
				if err == nil {
					format := output.GetFormat(formatFlag)
					if format == "json" {
						var data interface{}
						json.Unmarshal(resp.Data, &data)
						output.PrintJSON(data)
						return nil
					}

					var data map[string]interface{}
					if err := json.Unmarshal(resp.Data, &data); err == nil {
						blobFile := ""
						// 后端字段：noWaterBlobFileName（无水印，VIP 优先）/ hasWaterBlobFileName（带水印，普通）
						if bf, ok := data["noWaterBlobFileName"]; ok && bf != nil {
							if s := fmt.Sprintf("%v", bf); s != "" && s != "<nil>" {
								blobFile = s
							}
						}
						if blobFile == "" {
							if bf, ok := data["hasWaterBlobFileName"]; ok && bf != nil {
								if s := fmt.Sprintf("%v", bf); s != "" && s != "<nil>" {
									blobFile = s
								}
							}
						}
						// 兼容历史字段名
						if blobFile == "" {
							if bf, ok := data["blobFileName"]; ok && bf != nil {
								if s := fmt.Sprintf("%v", bf); s != "" && s != "<nil>" {
									blobFile = s
								}
							}
						}
						if outputPath == "" {
							origName := ""
							if name, ok := data["origFileName"]; ok && name != nil {
								origName = fmt.Sprintf("%v", name)
							} else if name, ok := data["originFileName"]; ok && name != nil {
								origName = fmt.Sprintf("%v", name)
							}
							if origName != "" && origName != "<nil>" {
								outputPath = fmt.Sprintf("translated_%s", origName)
							} else {
								outputPath = "translated_output.pdf"
							}
						}
						if blobFile != "" {
							return translateDownloadFile(downloadBaseURL, blobFile, outputPath)
						}
					}
				}
			}
		}

		// 免费用户：直接用 task-id 构造下载 URL
		// task-id 就是 blobFileName 去掉扩展名，加上 .pdf 即可
		blobFileName := taskID + ".pdf"

		if outputPath == "" {
			outputPath = "translated_output.pdf"
		}

		if err := translateDownloadFile(downloadBaseURL, blobFileName, outputPath); err != nil {
			return err
		}
		return nil
	},
}

var translateHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "查看翻译记录",
	Long: `查看历史翻译记录列表。

示例：
  pdf-cli translate history
  pdf-cli translate history --page 1 --page-size 20`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		c := client.NewBaseClient()
		resp, err := c.PostJSONAPI("user/operate/record/list/page", map[string]interface{}{
			"pageNo":   page,
			"pageSize": pageSize,
		})
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		listData := translateRecordList(resp)
		if format == "json" {
			output.PrintJSON(listData)
			return nil
		}

		if len(listData) == 0 {
			fmt.Println("  暂无翻译记录")
			return nil
		}

		headers := []string{"ID", "文件名", "状态", "时间"}
		var rows [][]string
		for _, item := range listData {
			if m, ok := item.(map[string]interface{}); ok {
				id := fmt.Sprintf("%v", m["id"])
				name := fmt.Sprintf("%v", m["origFileName"])
				status := fmt.Sprintf("%v", m["operateTag"])
				createTime := fmt.Sprintf("%v", m["createTime"])
				rows = append(rows, []string{id, name, status, createTime})
			}
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

var translateFreeCmd = &cobra.Command{
	Use:   "free",
	Short: "免费翻译 (无需登录)",
	Long: `免费翻译，无需登录即可使用。

示例：
  pdf-cli translate free --file-key xxx --to zh
  pdf-cli translate free --file-key xxx --to zh --from en --file-format pdf
  pdf-cli translate free --file-key xxx --to zh --pages "1,2,3"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileKey, _ := cmd.Flags().GetString("file-key")
		toLang, _ := cmd.Flags().GetString("to")
		fromLang, _ := cmd.Flags().GetString("from")
		fileFormat, _ := cmd.Flags().GetString("file-format")
		pages, _ := cmd.Flags().GetString("pages")
		visitorEmail, _ := cmd.Flags().GetString("visitor-email")
		return runFreeTranslate(fileKey, toLang, fromLang, fileFormat, pages, visitorEmail)
	},
}

var translateArxivCmd = &cobra.Command{
	Use:   "arxiv",
	Short: "arXiv 论文下载并翻译",
	Long: `通过 arXiv ID 下载论文并翻译。

示例：
  pdf-cli translate arxiv --arxiv-id 2301.00001 --to zh
  pdf-cli translate arxiv --arxiv-id 2301.00001 --to zh --engine 1 --term-ids "1,2"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		arxivID, _ := cmd.Flags().GetString("arxiv-id")
		toLang, _ := cmd.Flags().GetString("to")
		fromLang, _ := cmd.Flags().GetString("from")
		engine, _ := cmd.Flags().GetString("engine")
		ocrFlag, _ := cmd.Flags().GetBool("ocr")
		termIDs, _ := cmd.Flags().GetString("term-ids")

		if arxivID == "" {
			return clierr.ParamError("请提供 arXiv ID", "使用 --arxiv-id 参数，如 --arxiv-id 2301.00001")
		}
		if toLang == "" {
			return clierr.ParamError("请提供目标语言", "使用 --to 参数，如 --to zh")
		}

		if err := translatePrecheck(precheckOpts{kind: "arxiv", engine: engine, ocr: ocrFlag}); err != nil {
			return err
		}

		ocrVal := 0
		if ocrFlag {
			ocrVal = 1
		}

		data := map[string]interface{}{
			"arxivId":    arxivID,
			"targetLang": toLang,
			"ocrFlag":    ocrVal,
		}
		if fromLang != "" {
			data["sourceLang"] = fromLang
		}
		if engine != "" {
			data["translateEngineId"] = engine
		}
		if termIDs != "" {
			data["termIds"] = termIDs
		}

		c := client.NewBaseClient()
		resp, err := c.PostJSONAPI("core/pdf/arxiv/translate", data)
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		if format == "json" {
			var result interface{}
			json.Unmarshal(resp.Data, &result)
			output.PrintJSON(result)
			return nil
		}

		var result map[string]interface{}
		if err := json.Unmarshal(resp.Data, &result); err == nil {
			if blobFile, ok := result["blobFileName"]; ok {
				blobStr := fmt.Sprintf("%v", blobFile)
				taskID := translateTaskKey(blobStr)
				recordID := result["id"]
				origName := result["origFileName"]
				fmt.Printf("arXiv 翻译已发起\n  task-id    : %s\n", taskID)
				if recordID != nil {
					fmt.Printf("  record-id  : %v\n", recordID)
				}
				if origName != nil {
					fmt.Printf("  文件名     : %v\n", origName)
				}
				fmt.Println("\n下一步: pdf-cli translate status --task-id " + taskID)
				return nil
			}
		}

		output.PrintSuccess("arXiv 翻译已发起")
		output.PrintJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var translateArxivInfoCmd = &cobra.Command{
	Use:   "arxiv-info",
	Short: "查询 arXiv 论文摘要信息",
	Long: `通过 arXiv ID 查询论文的摘要信息。

示例：
  pdf-cli translate arxiv-info --arxiv-id 2301.00001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		arxivID, _ := cmd.Flags().GetString("arxiv-id")
		if arxivID == "" {
			return clierr.ParamError("请提供 arXiv ID", "使用 --arxiv-id 参数")
		}

		c := client.NewBaseClient()
		resp, err := c.GetAPI("core/pdf/query/arxiv/summary", map[string]string{
			"arxivId": arxivID,
		})
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		if format == "json" {
			var data interface{}
			json.Unmarshal(resp.Data, &data)
			output.PrintJSON(data)
			return nil
		}

		var data map[string]interface{}
		if err := json.Unmarshal(resp.Data, &data); err == nil {
			if title, ok := data["title"]; ok {
				fmt.Printf("  标题   : %v\n", title)
			}
			if authors, ok := data["authors"]; ok {
				fmt.Printf("  作者   : %v\n", authors)
			}
			if summary, ok := data["summary"]; ok {
				fmt.Printf("  摘要   : %v\n", summary)
			}
			return nil
		}

		output.PrintJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var translateTextCmd = &cobra.Command{
	Use:   "text",
	Short: "文本内容翻译",
	Long: `翻译指定文本内容到目标语言（使用 AI 翻译引擎，通过 SSE 流式返回）。

无需登录即可使用（游客模式）。

示例：
  pdf-cli translate text --text "Hello World" --to zh-CN
  pdf-cli translate text --text "Hello World" --to zh-CN --from en
  pdf-cli translate text --text "你好世界" --to en --engine gpt-4o-mini`,
	RunE: func(cmd *cobra.Command, args []string) error {
		text, _ := cmd.Flags().GetString("text")
		toLang, _ := cmd.Flags().GetString("to")
		fromLang, _ := cmd.Flags().GetString("from")
		engine, _ := cmd.Flags().GetString("engine")

		if text == "" {
			return clierr.ParamError("请提供待翻译文本", "使用 --text 参数")
		}
		if toLang == "" {
			return clierr.ParamError("请提供目标语言", "使用 --to 参数，如 --to zh-CN")
		}
		if engine == "" {
			engine = "gpt-4o-mini"
		}

		// 按 homepage 字段对 AI 翻译做额度预检（游客 visitorRemainAiTransCount / 用户 aiTransTrialLimitNum*）
		if err := translatePrecheck(precheckOpts{kind: "text", textLen: len(text)}); err != nil {
			return err
		}

		data := map[string]interface{}{
			"originText":   text,
			"bizType":      1,
			"sourceLang":   fromLang,
			"targetLang":   toLang,
			"aiEngineName": engine,
			"chatId":       1,
		}

		cfg := config.Load()
		apiURL := strings.TrimRight(cfg.GetBaseURL(), "/") + "/core/ai/translate/askstream"

		jsonData, _ := json.Marshal(data)
		req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonData)))
		if err != nil {
			return clierr.NetError("创建请求失败: "+err.Error(), "")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("clientType", "cli")
		deviceID := auth.LoadDeviceID()
		if deviceID != "" {
			req.Header.Set("deviceId", deviceID)
		}
		token, _ := auth.LoadToken()
		if token != "" {
			req.Header.Set("token", token)
			req.Header.Set("streamtoken", token)
		}

		httpClient := &http.Client{Timeout: 2 * time.Minute}
		resp, err := httpClient.Do(req)
		if err != nil {
			return client.ClassifyTransportError("请求失败", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return client.ClassifyHTTPStatus(resp.StatusCode, body)
		}

		// 读取 SSE 流
		format := output.GetFormat(formatFlag)
		var result strings.Builder
		var lastEvent string
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") {
				lastEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				eventData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if eventData == "" {
					continue
				}
				if lastEvent == "[ERROR]" {
					return clierr.TaskFailedError("文本翻译失败: "+eventData, "可重新发起 translate text 命令")
				}
				if lastEvent == "[FINISH]" {
					break
				}
				if lastEvent == "[DATA]" || lastEvent == "" {
					var chunk map[string]interface{}
					if err := json.Unmarshal([]byte(eventData), &chunk); err == nil {
						if t, ok := chunk["text"]; ok {
							s := fmt.Sprintf("%v", t)
							result.WriteString(s)
							if format != "json" {
								fmt.Print(s)
							}
						}
					}
				}
			}
		}
		if format != "json" && result.Len() > 0 {
			fmt.Println()
		}

		if format == "json" {
			output.PrintJSON(map[string]string{
				"translatedText": result.String(),
			})
		}
		return nil
	},
}

func init() {
	translateUploadCmd.Flags().String("file", "", "待翻译文件路径")
	translateUploadCmd.Flags().Bool("free", false, "强制免费流程 (跳过 高级/免费 菜单；适用于 Agent CLI / 脚本)")
	translateUploadCmd.Flags().Bool("advanced", false, "强制高级流程 (需登录；跳过 高级/免费 菜单)")
	translateUploadCmd.Flags().String("to", "", "目标语言代码 (跳过语言选择菜单)")
	translateUploadCmd.Flags().String("engine", "", "翻译引擎 (仅高级流程生效，跳过引擎选择菜单)")
	translateUploadCmd.Flags().Bool("ocr", false, "启用 OCR (仅高级流程生效，跳过 OCR 选择菜单)")
	_ = translateUploadCmd.MarkFlagRequired("file")

	translateStartCmd.Flags().String("file-key", "", "上传后获得的文件 key")
	translateStartCmd.Flags().String("to", "", "目标语言代码 (如 zh, en, ja)")
	translateStartCmd.Flags().String("from", "", "源语言代码 (可选，自动检测)")
	translateStartCmd.Flags().String("engine", "", "翻译引擎 (可选)")
	translateStartCmd.Flags().Bool("ocr", false, "启用 OCR 模式 (扫描件翻译)")
	translateStartCmd.Flags().String("term-ids", "", "术语表 ID，多个用逗号分隔 (可选)")
	translateStartCmd.Flags().String("file-format", "", "文件格式，如 pdf, docx (可选)")
	translateStartCmd.Flags().Int("prompt-type", 0, "翻译风格/提示类型 (可选)")

	translateStatusCmd.Flags().String("task-id", "", "翻译任务 ID")
	translateStatusCmd.Flags().Bool("wait", false, "等待翻译完成")

	translateContinueCmd.Flags().String("task-id", "", "翻译任务 ID")
	translateCancelCmd.Flags().String("task-id", "", "翻译任务 ID")

	translateDownloadCmd.Flags().String("task-id", "", "翻译任务 ID")
	translateDownloadCmd.Flags().StringP("output", "o", "", "输出文件路径")

	translateHistoryCmd.Flags().Int("page", 1, "页码")
	translateHistoryCmd.Flags().Int("page-size", 20, "每页数量")

	translateFreeCmd.Flags().String("file-key", "", "上传后获得的文件 key")
	translateFreeCmd.Flags().String("to", "", "目标语言代码 (如 zh, en, ja)")
	translateFreeCmd.Flags().String("from", "", "源语言代码 (可选，自动检测)")
	translateFreeCmd.Flags().String("file-format", "", "文件格式，如 pdf, docx (可选)")
	translateFreeCmd.Flags().String("pages", "", "指定翻译页码，逗号分隔 (可选)")
	translateFreeCmd.Flags().String("visitor-email", "", "游客邮箱 (可选)")

	translateArxivCmd.Flags().String("arxiv-id", "", "arXiv 论文 ID (如 2301.00001)")
	translateArxivCmd.Flags().String("to", "", "目标语言代码 (如 zh, en, ja)")
	translateArxivCmd.Flags().String("from", "", "源语言代码 (可选，自动检测)")
	translateArxivCmd.Flags().String("engine", "", "翻译引擎 ID (可选)")
	translateArxivCmd.Flags().Bool("ocr", false, "启用 OCR 模式 (可选)")
	translateArxivCmd.Flags().String("term-ids", "", "术语表 ID，多个用逗号分隔 (可选)")

	translateArxivInfoCmd.Flags().String("arxiv-id", "", "arXiv 论文 ID (如 2301.00001)")

	translateTextCmd.Flags().String("text", "", "待翻译的文本内容")
	translateTextCmd.Flags().String("to", "", "目标语言代码 (如 zh-CN, en, ja)")
	translateTextCmd.Flags().String("from", "", "源语言代码 (可选，自动检测)")
	translateTextCmd.Flags().String("engine", "", "AI 引擎名称 (可选，默认 gpt-4o-mini)")

	translateCmd.AddCommand(translateLanguagesCmd)
	translateCmd.AddCommand(translateEnginesCmd)
	translateCmd.AddCommand(translateUploadCmd)
	translateCmd.AddCommand(translateStartCmd)
	translateCmd.AddCommand(translateFreeCmd)
	translateCmd.AddCommand(translateArxivCmd)
	translateCmd.AddCommand(translateArxivInfoCmd)
	translateCmd.AddCommand(translateTextCmd)
	translateCmd.AddCommand(translateStatusCmd)
	translateCmd.AddCommand(translateContinueCmd)
	translateCmd.AddCommand(translateCancelCmd)
	translateCmd.AddCommand(translateDownloadCmd)
	translateCmd.AddCommand(translateHistoryCmd)
	rootCmd.AddCommand(translateCmd)
}
