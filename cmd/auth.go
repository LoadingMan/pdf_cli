package cmd

import (
	"encoding/json"
	"fmt"

	"pdf-cli/internal/auth"
	"pdf-cli/internal/client"
	clierr "pdf-cli/internal/errors"
	"pdf-cli/internal/output"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "登录、登出与账号状态",
	Long: `认证管理命令。

示例：
  pdf-cli auth login --email you@example.com
  pdf-cli auth status
  pdf-cli auth logout`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "登录账号",
	Long: `使用邮箱和密码登录。

示例：
  pdf-cli auth login --email you@example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			return clierr.ParamError("请提供邮箱地址", "使用 --email 参数")
		}

		password, err := readPasswordInteractive("请输入密码: ")
		if err != nil {
			return clierr.ParamError("读取密码失败: "+err.Error(), "")
		}
		if password == "" {
			return clierr.ParamError("密码不能为空", "")
		}

		c := client.NewBaseClient()
		resp, err := c.PostJSONAPI("user/basic/base/login", map[string]interface{}{
			"userEmail": email,
			"password":  password,
		})
		if err != nil {
			return err
		}

		tokenStr := ""
		if err := json.Unmarshal(resp.Data, &tokenStr); err != nil || tokenStr == "" {
			var data map[string]interface{}
			if err := json.Unmarshal(resp.Data, &data); err != nil {
				return clierr.New(clierr.ExitInternal, clierr.TypeInternal, "解析登录响应失败: "+err.Error(), "", false)
			}
			token, ok := data["token"]
			if !ok {
				return clierr.New(clierr.ExitInternal, clierr.TypeInternal, "登录响应中未找到 token 字段", "", false)
			}
			tokenStr = fmt.Sprintf("%v", token)
		}

		if err := auth.SaveToken(tokenStr); err != nil {
			return clierr.ConfigError("保存 token 失败: "+err.Error(), "请检查 ~/.config/pdf-cli/ 目录权限")
		}

		output.PrintSuccess("登录成功")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看登录状态",
	Long: `查看当前登录状态和用户信息。

示例：
  pdf-cli auth status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !auth.IsLoggedIn() {
			fmt.Println("当前未登录")
			fmt.Println("Hint: 请执行 pdf-cli auth login --email you@example.com")
			return nil
		}

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
			output.PrintSuccess("已登录")
			return nil
		}

		fmt.Println("登录状态: 已登录")
		keys := []string{"邮箱", "昵称", "会员等级", "剩余额度"}
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
		output.PrintOrderedPretty(keys, fields)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "退出登录",
	Long: `退出当前登录状态。

示例：
  pdf-cli auth logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if auth.IsLoggedIn() {
			c := client.NewBaseClient()
			c.Get("user/basic/login/out", nil)
		}

		if err := auth.ClearToken(); err != nil {
			return clierr.ConfigError("清除 token 失败: "+err.Error(), "请检查 ~/.config/pdf-cli/ 目录权限")
		}

		output.PrintSuccess("已退出登录")
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("email", "", "登录邮箱")
	_ = authLoginCmd.MarkFlagRequired("email")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}

func requireAuth() error {
	if !auth.IsLoggedIn() {
		return clierr.AuthError("未登录", "请先执行 pdf-cli auth login --email you@example.com")
	}
	return nil
}
