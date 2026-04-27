package cmd

import (
	"encoding/json"
	"fmt"

	"pdf-cli/internal/client"
	clierr "pdf-cli/internal/errors"
	"pdf-cli/internal/output"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "用户信息与管理",
	Long: `用户模块，管理个人资料、文件记录、API Key 等。

示例：
  pdf-cli user profile
  pdf-cli user files list
  pdf-cli user api-key list`,
}

var userProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "查看个人资料",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/basic/token/userinfo", nil)
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

		keys := []string{"邮箱", "昵称", "会员等级", "剩余额度", "注册时间"}
		fields := map[string]interface{}{}
		if v, ok := data["userEmail"]; ok {
			fields["邮箱"] = v
		}
		if v, ok := data["nickName"]; ok {
			fields["昵称"] = v
		}
		if v, ok := data["vipLevel"]; ok {
			fields["会员等级"] = v
		}
		if v, ok := data["remainScore"]; ok {
			fields["剩余额度"] = v
		}
		if v, ok := data["createTime"]; ok {
			fields["注册时间"] = v
		}
		output.PrintOrderedPretty(keys, fields)
		return nil
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "修改个人资料",
	Long: `修改用户昵称等个人信息。

示例：
  pdf-cli user update --name Alice`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return clierr.ParamError("请提供要修改的信息", "使用 --name 参数")
		}

		c := client.NewBaseClient()
		_, err := c.PostJSONAPI("user/basic/modify/userinfo", map[string]interface{}{
			"nickName": name,
		})
		if err != nil {
			return err
		}

		output.PrintSuccess("资料已更新")
		return nil
	},
}

// files 子命令组
var userFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "文件管理",
}

var userFilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看上传文件列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		pageSize, _ := cmd.Flags().GetInt("page-size")
		if pageSize <= 0 {
			pageSize = 20
		}

		c := client.NewBaseClient()
		resp, err := c.PostJSONAPI("user/source/file/list", map[string]interface{}{
			"pageSize": pageSize,
		})
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		var listData []interface{}
		if resp.DataList != nil {
			json.Unmarshal(resp.DataList, &listData)
		}
		if listData == nil && resp.List != nil {
			json.Unmarshal(resp.List, &listData)
		}

		if format == "json" {
			output.PrintJSON(listData)
			return nil
		}

		if len(listData) == 0 {
			fmt.Println("  暂无文件")
			return nil
		}

		headers := []string{"ID", "文件名", "大小", "上传时间"}
		var rows [][]string
		for _, item := range listData {
			if m, ok := item.(map[string]interface{}); ok {
				// 尝试多个可能的文件名字段
				fileName := m["originFileName"]
				if fileName == nil {
					fileName = m["origFileName"]
				}
				if fileName == nil {
					fileName = m["fileName"]
				}
				rows = append(rows, []string{
					fmt.Sprintf("%v", m["id"]),
					fmt.Sprintf("%v", fileName),
					fmt.Sprintf("%v", m["fileSize"]),
					fmt.Sprintf("%v", m["createTime"]),
				})
			}
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

// records 子命令组
var userRecordsCmd = &cobra.Command{
	Use:   "records",
	Short: "使用记录",
}

var userRecordsListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看使用记录列表",
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

		if format == "json" {
			output.PrintJSON(listData)
			return nil
		}

		if len(listData) == 0 {
			fmt.Println("  暂无记录")
			return nil
		}

		headers := []string{"ID", "文件名", "状态", "时间"}
		var rows [][]string
		for _, item := range listData {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, []string{
					fmt.Sprintf("%v", m["id"]),
					fmt.Sprintf("%v", m["origFileName"]),
					fmt.Sprintf("%v", m["operateTag"]),
					fmt.Sprintf("%v", m["createTime"]),
				})
			}
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

var userRecordsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "查看记录详情",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return clierr.ParamError("请提供记录 ID", "使用 --id 参数")
		}

		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/operate/record/get", map[string]string{
			"thisId": id,
		})
		if err != nil {
			return err
		}

		var data interface{}
		json.Unmarshal(resp.Data, &data)
		output.PrintJSON(data)
		return nil
	},
}

// api-key 子命令组
var userApiKeyCmd = &cobra.Command{
	Use:   "api-key",
	Short: "API Key 管理",
}

var userApiKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看 API Key 列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/apicommon/sk/list", nil)
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

		if format == "json" {
			output.PrintJSON(listData)
			return nil
		}

		if len(listData) == 0 {
			fmt.Println("  暂无 API Key")
			return nil
		}

		headers := []string{"ID", "名称", "Secret Key"}
		var rows [][]string
		for _, item := range listData {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, []string{
					fmt.Sprintf("%v", m["id"]),
					fmt.Sprintf("%v", m["keyName"]),
					fmt.Sprintf("%v", m["secretKey"]),
				})
			}
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

var userApiKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建 API Key",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return clierr.ParamError("请提供 Key 名称", "使用 --name 参数")
		}

		c := client.NewBaseClient()
		_, err := c.PostJSONAPI("user/apicommon/sk/add", map[string]interface{}{
			"keyName": name,
		})
		if err != nil {
			return err
		}

		output.PrintSuccess("API Key 已创建")
		return nil
	},
}

var userApiKeyDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除 API Key",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return clierr.ParamError("请提供 Key ID", "使用 --id 参数")
		}

		c := client.NewBaseClient()
		_, err := c.PostJSONAPI("user/apicommon/sk/del", map[string]interface{}{
			"thisId": id,
		})
		if err != nil {
			return err
		}

		output.PrintSuccess("API Key 已删除")
		return nil
	},
}

// feedback
var userFeedbackCmd = &cobra.Command{
	Use:   "feedback",
	Short: "反馈管理",
}

var userFeedbackSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "提交反馈",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		title, _ := cmd.Flags().GetString("title")
		content, _ := cmd.Flags().GetString("content")
		if title == "" || content == "" {
			return clierr.ParamError("请提供标题和内容", "使用 --title 和 --content 参数")
		}

		c := client.NewBaseClient()
		_, err := c.PostJSONAPI("user/feedback/save", map[string]interface{}{
			"title":   title,
			"content": content,
		})
		if err != nil {
			return err
		}

		output.PrintSuccess("反馈已提交")
		return nil
	},
}

func init() {
	userUpdateCmd.Flags().String("name", "", "新昵称")

	userFilesListCmd.Flags().Int("page-size", 20, "每页数量")
	userFilesCmd.AddCommand(userFilesListCmd)

	userRecordsListCmd.Flags().Int("page", 1, "页码")
	userRecordsListCmd.Flags().Int("page-size", 20, "每页数量")
	userRecordsGetCmd.Flags().String("id", "", "记录 ID")
	userRecordsCmd.AddCommand(userRecordsListCmd)
	userRecordsCmd.AddCommand(userRecordsGetCmd)

	userApiKeyCreateCmd.Flags().String("name", "", "Key 名称")
	userApiKeyDeleteCmd.Flags().String("id", "", "Key ID")
	userApiKeyCmd.AddCommand(userApiKeyListCmd)
	userApiKeyCmd.AddCommand(userApiKeyCreateCmd)
	userApiKeyCmd.AddCommand(userApiKeyDeleteCmd)

	userFeedbackSubmitCmd.Flags().String("title", "", "反馈标题")
	userFeedbackSubmitCmd.Flags().String("content", "", "反馈内容")
	userFeedbackCmd.AddCommand(userFeedbackSubmitCmd)

	userCmd.AddCommand(userProfileCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userFilesCmd)
	userCmd.AddCommand(userRecordsCmd)
	userCmd.AddCommand(userApiKeyCmd)
	userCmd.AddCommand(userFeedbackCmd)
	rootCmd.AddCommand(userCmd)
}
