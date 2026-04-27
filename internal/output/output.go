package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PrintJSON 输出 JSON 格式
func PrintJSON(data interface{}) {
	out, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(out))
}

// PrintPretty 输出 key-value 格式
func PrintPretty(fields map[string]interface{}) {
	maxLen := 0
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}
	for _, k := range keys {
		v := fields[k]
		fmt.Printf("  %-*s : %v\n", maxLen, k, v)
	}
}

// PrintOrderedPretty 按指定顺序输出 key-value
func PrintOrderedPretty(keys []string, fields map[string]interface{}) {
	maxLen := 0
	for _, k := range keys {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}
	for _, k := range keys {
		if v, ok := fields[k]; ok {
			fmt.Printf("  %-*s : %v\n", maxLen, k, v)
		}
	}
}

// PrintTable 输出表格格式
func PrintTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println("  (无数据)")
		return
	}

	// 计算列宽
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// 打印表头
	headerLine := "  "
	sepLine := "  "
	for i, h := range headers {
		headerLine += fmt.Sprintf("%-*s", widths[i]+2, h)
		sepLine += strings.Repeat("-", widths[i]) + "  "
	}
	fmt.Println(headerLine)
	fmt.Println(sepLine)

	// 打印数据行
	for _, row := range rows {
		line := "  "
		for i, cell := range row {
			if i < len(widths) {
				line += fmt.Sprintf("%-*s", widths[i]+2, cell)
			}
		}
		fmt.Println(line)
	}
}

// PrintSuccess 输出成功消息
func PrintSuccess(msg string) {
	fmt.Println("OK: " + msg)
}

// PrintError 输出错误消息
func PrintError(msg string) {
	fmt.Fprintln(os.Stderr, "Error: "+msg)
}

// GetFormat 获取输出格式
func GetFormat(flag string) string {
	if flag != "" {
		return strings.ToLower(flag)
	}
	return "pretty"
}

// PrintByFormat 根据格式输出数据
func PrintByFormat(format string, jsonData interface{}, prettyFn func()) {
	switch format {
	case "json":
		PrintJSON(jsonData)
	default:
		prettyFn()
	}
}
