package cmd

import (
	"encoding/json"
	"fmt"

	"pdf-cli/internal/client"
	"pdf-cli/internal/output"

	"github.com/spf13/cobra"
)

var otherCmd = &cobra.Command{
	Use:   "other",
	Short: "公告、版本与帮助",
}

var otherVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "查看版本历史",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/history/version/list", map[string]string{
			"clientType": "1",
		})
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		var listData []interface{}
		if resp.List != nil {
			json.Unmarshal(resp.List, &listData)
		}
		if listData == nil && resp.Data != nil {
			json.Unmarshal(resp.Data, &listData)
		}
		if listData == nil && resp.List != nil {
			json.Unmarshal(resp.List, &listData)
		}

		if format == "json" {
			output.PrintJSON(listData)
			return nil
		}

		if len(listData) == 0 {
			fmt.Println("  暂无版本信息")
			return nil
		}

		headers := []string{"版本号", "更新内容", "发布时间"}
		var rows [][]string
		for _, item := range listData {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, []string{
					fmt.Sprintf("%v", m["versionNo"]),
					fmt.Sprintf("%v", m["content"]),
					fmt.Sprintf("%v", m["createTime"]),
				})
			}
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

var otherNoticeCmd = &cobra.Command{
	Use:   "notice",
	Short: "查看公告",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/config/homepage", nil)
		if err != nil {
			return err
		}

		var data interface{}
		json.Unmarshal(resp.Data, &data)

		format := output.GetFormat(formatFlag)
		if format == "json" {
			output.PrintJSON(data)
		} else {
			output.PrintJSON(data)
		}
		return nil
	},
}

var otherHelpCmd = &cobra.Command{
	Use:   "help-guide",
	Short: "使用指南",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`pdf-cli 使用指南
================

1. 登录
   pdf-cli auth login --email you@example.com

2. 翻译 PDF
   pdf-cli translate upload --file ./paper.pdf
   pdf-cli translate start --file-key <key> --to zh
   pdf-cli translate status --task-id <id> --wait
   pdf-cli translate download --task-id <id>

3. PDF 工具
   pdf-cli tools merge --files a.pdf,b.pdf
   pdf-cli tools convert pdf-to-word --file a.pdf
   pdf-cli tools page extract --file a.pdf --pages 1-3
   pdf-cli tools compress --file a.pdf --dpi 144 --image-quality 75
   pdf-cli tools security encrypt --file a.pdf --password 123456
   pdf-cli tools job status --query-key <key>

4. 用户管理
   pdf-cli user profile
   pdf-cli user api-key list
   pdf-cli user api-key create --name "my-key"

5. 会员
   pdf-cli member info
   pdf-cli member redeem --code XXXX-XXXX

6. 全局选项
   --format json    输出 JSON 格式
   --output file    输出到文件

更多帮助: pdf-cli <command> --help`)
	},
}

func init() {
	otherCmd.AddCommand(otherVersionCmd)
	otherCmd.AddCommand(otherNoticeCmd)
	otherCmd.AddCommand(otherHelpCmd)
	rootCmd.AddCommand(otherCmd)
}
