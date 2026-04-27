// Package userconfig 获取并缓存用户限制条件（user/config/homepage）。
// 独立于 internal/config 以避免与 internal/client 之间的循环依赖。
package userconfig

import (
	"encoding/json"
	"fmt"
	"strconv"

	"pdf-cli/internal/client"
	clierr "pdf-cli/internal/errors"
)

// FlexFloat 兼容后端混用数值与字符串的字段（例如 "10485760" 或 10485760）。
// 基于 float64 以保持与现有比较运算代码的兼容性。
type FlexFloat float64

// UnmarshalJSON 接受 number、string("12" / "12.3" / "") 或 null。
func (f *FlexFloat) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*f = 0
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			*f = 0
			return nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("FlexFloat: 无法解析字符串 %q 为 float: %w", s, err)
		}
		*f = FlexFloat(v)
		return nil
	}
	var v float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*f = FlexFloat(v)
	return nil
}

// Homepage 对应后端 user/config/homepage 返回的用户限制条件。
// 数值字段统一用 FlexFloat，兼容后端偶尔以字符串形式下发。
// JSON 对象类型字段用 Raw 访问（visitorTransDoingJson / stuckTransWarnJson 等）。
type Homepage struct {
	// 基础配置
	ToolsBoxFileMaxSize FlexFloat `json:"toolsBoxFileMaxSize"` // pdf 小工具文件大小限制 (MB)
	CheckSameFileSwitch FlexFloat `json:"checkSameFileSwitch"` // 重复上传文件提醒开关
	MaybePdfPageChars   FlexFloat `json:"maybePdfPageChars"`   // 每页预估字符数

	// 会员翻译额度
	RemainCharsCountByDay FlexFloat `json:"remainCharsCountByDay"` // 今日剩余字符数（null = 无限制）
	RemainTransCountByDay FlexFloat `json:"remainTransCountByDay"` // 今日剩余次数（null = 无限制）

	// 免费翻译限制（游客/用户）
	FreeTransLimitNumPerDay FlexFloat `json:"freeTransLimitNumPerDay"` // 每日限制次数
	FreeTransMaxFileSize    FlexFloat `json:"freeTransMaxFileSize"`    // 最大文件 MB
	FreeTransMaxPages       FlexFloat `json:"freeTransMaxPages"`       // 最大页数
	FreeTransYetUsedNum     FlexFloat `json:"freeTransYetUsedNum"`     // 已用次数

	// AI 翻译试用限制
	AiTransTrialLimitNumPerDay FlexFloat `json:"aiTransTrialLimitNumPerDay"` // 每日试用
	AiTransTrialLimitNumPerMon FlexFloat `json:"aiTransTrialLimitNumPerMon"` // 每月试用
	AiTransHistoryLimit        FlexFloat `json:"aiTransHistoryLimit"`        // 历史聊天框上限

	// AI 会话试用限制
	AiChatTrialNumPerDay FlexFloat `json:"aiChatTrialNumPerDay"` // 每日试用
	AiChatTrialNumPerMon FlexFloat `json:"aiChatTrialNumPerMon"` // 每月试用

	// 术语表
	TermCount FlexFloat `json:"termCount"` // 术语表词条总数

	// 试看（预览）
	TrySeeDayCountLimit FlexFloat `json:"trySeeDayCountLimit"` // 每日试看次数
	TrySeeOneCharsLimit FlexFloat `json:"trySeeOneCharsLimit"` // 每次试看字符数
	TrySeeUserYetCount  FlexFloat `json:"trySeeUserYetCount"`  // 已试看次数

	// 游客配置
	VisitorFileExpireTime     FlexFloat `json:"visitorFileExpireTime"`     // 游客文件过期时间
	VisitorRemainAiTransCount FlexFloat `json:"visitorRemainAiTransCount"` // 游客 AI 翻译剩余次数

	// 其它
	SubscriptionSwitch    FlexFloat `json:"subscriptionSwitch"`    // 订阅开关
	HasLastTransParamFlag bool      `json:"hasLastTransParamFlag"` // 是否有上次翻译参数
	RedPointNum           FlexFloat `json:"redPointNum"`           // 红点数量

	// Raw 保留完整响应用于访问复杂 JSON 字段
	Raw map[string]interface{} `json:"-"`
}

// Fetch 查询当前用户的限制条件（未登录时后端会返回游客维度的同 shape 数据）。
func Fetch() (*Homepage, error) {
	c := client.NewBaseClient()
	resp, err := c.GetAPI("user/config/homepage", nil)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if len(resp.Data) > 0 {
		_ = json.Unmarshal(resp.Data, &raw)
	}
	h := &Homepage{Raw: raw}
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, h); err != nil {
			return nil, clierr.New(clierr.ExitInternal, clierr.TypeInternal,
				"解析 homepage 配置失败: "+err.Error(), "", false)
		}
	}
	return h, nil
}

// AsInt 从 Raw 中按 key 取整数值，兼容 number / string。
func (h *Homepage) AsInt(key string) (int64, bool) {
	if h == nil || h.Raw == nil {
		return 0, false
	}
	v, ok := h.Raw[key]
	if !ok || v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case string:
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

// VisitorHasInProgress 解析 visitorTransDoingJson 字段。非空表示游客有一个正在翻译的任务，
// 返回 (true, 文件名)；否则返回 (false, "")。
func (h *Homepage) VisitorHasInProgress() (bool, string) {
	if h == nil || h.Raw == nil {
		return false, ""
	}
	v, ok := h.Raw["visitorTransDoingJson"]
	if !ok || v == nil {
		return false, ""
	}
	// 后端返回可能是 string（JSON 字符串）、map 或 null
	switch x := v.(type) {
	case string:
		if x == "" || x == "null" || x == "{}" {
			return false, ""
		}
		var inner map[string]interface{}
		if json.Unmarshal([]byte(x), &inner) == nil {
			return true, extractFileName(inner)
		}
		return true, ""
	case map[string]interface{}:
		if len(x) == 0 {
			return false, ""
		}
		return true, extractFileName(x)
	}
	return false, ""
}

// StuckTransWarn 解析 stuckTransWarnJson 字段。返回警告信息描述（文件名/剩余字符等）。
func (h *Homepage) StuckTransWarn() (bool, string) {
	if h == nil || h.Raw == nil {
		return false, ""
	}
	v, ok := h.Raw["stuckTransWarnJson"]
	if !ok || v == nil {
		return false, ""
	}
	switch x := v.(type) {
	case string:
		if x == "" || x == "null" || x == "{}" {
			return false, ""
		}
		return true, x
	case map[string]interface{}:
		if len(x) == 0 {
			return false, ""
		}
		b, _ := json.Marshal(x)
		return true, string(b)
	}
	return false, ""
}

func extractFileName(m map[string]interface{}) string {
	for _, key := range []string{"fileName", "origFileName", "fileRealName", "blobFileName", "tmpFileName"} {
		if v, ok := m[key]; ok && v != nil {
			if s := fmt.Sprintf("%v", v); s != "" {
				return s
			}
		}
	}
	return ""
}

// HumanBytes 以人类可读单位格式化字节大小。
func HumanBytes(n float64) string {
	if n >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", n/1024/1024)
	}
	if n >= 1024 {
		return fmt.Sprintf("%.1f KB", n/1024)
	}
	return fmt.Sprintf("%.0f B", n)
}
