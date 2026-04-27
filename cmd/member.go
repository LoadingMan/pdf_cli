package cmd

import (
	"encoding/json"
	"fmt"

	"pdf-cli/internal/client"
	clierr "pdf-cli/internal/errors"
	"pdf-cli/internal/output"

	"github.com/spf13/cobra"
)

var memberCmd = &cobra.Command{
	Use:   "member",
	Short: "会员信息与管理",
	Long: `会员模块，查看会员信息、权益、定价、订单等。

示例：
  pdf-cli member info
  pdf-cli member rights
  pdf-cli member pricing
  pdf-cli member order list`,
}

var memberInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "查看会员配置信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/config/vip/cfg", nil)
		if err != nil {
			return err
		}

		// 数据可能在 data 或 list 字段
		var rawData interface{}
		if resp.List != nil {
			json.Unmarshal(resp.List, &rawData)
		}
		if rawData == nil && resp.Data != nil {
			json.Unmarshal(resp.Data, &rawData)
		}

		format := output.GetFormat(formatFlag)
		if format == "json" {
			output.PrintJSON(rawData)
			return nil
		}

		// 表格形式输出
		list, ok := rawData.([]interface{})
		if !ok {
			output.PrintJSON(rawData)
			return nil
		}

		headers := []string{"等级", "每月翻译数", "多文件数", "最大文件(MB)", "存储天数"}
		var rows [][]string
		for _, item := range list {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			vipLevel := fmt.Sprintf("%v", m["vipLevel"])
			levelName := vipLevel
			switch vipLevel {
			case "1":
				levelName = "VIP"
			case "2":
				levelName = "SVIP"
			}

			funCfg, _ := m["funCfg"].(map[string]interface{})
			transCount := ""
			multiFile := ""
			maxSize := ""
			storageDay := ""
			if funCfg != nil {
				if v, ok := funCfg["transCountPerMon"]; ok {
					transCount = fmt.Sprintf("%v", v)
				}
				if v, ok := funCfg["multiFileCount"]; ok {
					multiFile = fmt.Sprintf("%v", v)
				}
				if v, ok := funCfg["maxFileSize"]; ok {
					maxSize = fmt.Sprintf("%v", v)
				}
				if v, ok := funCfg["storageDayNum"]; ok {
					storageDay = fmt.Sprintf("%v", v)
				}
			}
			rows = append(rows, []string{levelName, transCount, multiFile, maxSize, storageDay})
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

var memberRightsCmd = &cobra.Command{
	Use:   "rights",
	Short: "查看会员权益",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/basic/get/vip/functions", nil)
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

var memberPricingCmd = &cobra.Command{
	Use:   "pricing",
	Short: "查看定价方案",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/config/all/price/cfg", nil)
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

// order 子命令组
var memberOrderCmd = &cobra.Command{
	Use:   "order",
	Short: "订单管理",
}

var memberOrderListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看订单列表",
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
		resp, err := c.PostJSONAPI("user/trade/list", map[string]interface{}{
			"pageNo":   page,
			"pageSize": pageSize,
		})
		if err != nil {
			return err
		}

		format := output.GetFormat(formatFlag)
		var listData []interface{}
		if resp.Data != nil {
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
			fmt.Println("  暂无订单")
			return nil
		}

		headers := []string{"订单号", "金额", "状态", "时间"}
		var rows [][]string
		for _, item := range listData {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, []string{
					fmt.Sprintf("%v", m["orderNo"]),
					fmt.Sprintf("%v", m["amount"]),
					fmt.Sprintf("%v", m["status"]),
					fmt.Sprintf("%v", m["createTime"]),
				})
			}
		}
		output.PrintTable(headers, rows)
		return nil
	},
}

var memberOrderGetCmd = &cobra.Command{
	Use:   "get",
	Short: "查看订单详情",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		orderNo, _ := cmd.Flags().GetString("order-no")
		if orderNo == "" {
			return clierr.ParamError("请提供订单号", "使用 --order-no 参数")
		}

		c := client.NewBaseClient()
		resp, err := c.GetAPI("user/trade/get", map[string]string{
			"orderNo": orderNo,
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

var memberRedeemCmd = &cobra.Command{
	Use:   "redeem",
	Short: "兑换会员码",
	Long: `使用兑换码激活会员。

示例：
  pdf-cli member redeem --code XXXX-XXXX-XXXX`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		code, _ := cmd.Flags().GetString("code")
		if code == "" {
			return clierr.ParamError("请提供兑换码", "使用 --code 参数")
		}

		c := client.NewBaseClient()
		_, err := c.PostJSONAPI("user/vipcode/bind", map[string]interface{}{
			"vipCode": code,
		})
		if err != nil {
			return err
		}

		output.PrintSuccess("兑换成功")
		return nil
	},
}

func init() {
	memberOrderListCmd.Flags().Int("page", 1, "页码")
	memberOrderListCmd.Flags().Int("page-size", 20, "每页数量")
	memberOrderGetCmd.Flags().String("order-no", "", "订单号")
	memberOrderCmd.AddCommand(memberOrderListCmd)
	memberOrderCmd.AddCommand(memberOrderGetCmd)

	memberRedeemCmd.Flags().String("code", "", "兑换码")

	memberCmd.AddCommand(memberInfoCmd)
	memberCmd.AddCommand(memberRightsCmd)
	memberCmd.AddCommand(memberPricingCmd)
	memberCmd.AddCommand(memberOrderCmd)
	memberCmd.AddCommand(memberRedeemCmd)
	rootCmd.AddCommand(memberCmd)
}
