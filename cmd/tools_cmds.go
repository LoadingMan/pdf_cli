package cmd

import (
	"strings"

	clierr "pdf-cli/internal/errors"

	"github.com/spf13/cobra"
)

var toolsConvertCmd = &cobra.Command{
	Use:   "convert",
	Short: "PDF 格式转换",
}

var toolsConvertPDFToWordCmd = &cobra.Command{
	Use:   "pdf-to-word",
	Short: "PDF 转 Word",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 2)
		if err != nil {
			return err
		}
		return toolRunAction("CoverPDFTo", 2, map[string]interface{}{
			"files": []FileEntry{*entry},
			"cover": "docx",
		})
	},
}

var toolsMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "合并多个 PDF",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePaths, err := toolRequireFiles(cmd)
		if err != nil {
			return err
		}
		entries, err := toolUploadFiles(filePaths, 1)
		if err != nil {
			return err
		}
		createBookmarks, _ := cmd.Flags().GetBool("create-bookmarks")
		return toolRunAction("CombineFile", 1, map[string]interface{}{
			"files":           entries,
			"createBookmarks": createBookmarks,
		})
	},
}

var toolsSplitCmd = &cobra.Command{
	Use:   "split",
	Short: "拆分 PDF",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		mode, _ := cmd.Flags().GetString("mode")
		pagesPerPDF, _ := cmd.Flags().GetInt("pages-per-pdf")
		splitPoints, _ := cmd.Flags().GetString("split-points")
		mappedMode, err := toolSplitMode(mode)
		if err != nil {
			return err
		}
		if mappedMode == "pagesPerPdf" && pagesPerPDF <= 0 {
			return clierr.ParamError("请提供每个拆分文件的页数", "使用 --pages-per-pdf 参数")
		}
		if mappedMode == "custom" && strings.TrimSpace(splitPoints) == "" {
			return clierr.ParamError("请提供自定义拆分页码", "使用 --split-points 参数，例如 1,3,5")
		}
		entry, err := toolUploadFile(filePath, 5)
		if err != nil {
			return err
		}
		data := map[string]interface{}{
			"files": []FileEntry{*entry},
			"mode":  mappedMode,
		}
		if mappedMode == "pagesPerPdf" {
			data["pagesPerPdf"] = pagesPerPDF
		}
		if mappedMode == "custom" {
			data["splitPoints"] = []string{splitPoints}
		}
		return toolRunAction("SplitFile", 5, data)
	},
}

var toolsReorderCmd = &cobra.Command{
	Use:   "reorder",
	Short: "重排 PDF 页面",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		order, _ := cmd.Flags().GetString("order")
		pages, err := toolParseIntList(order)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 6)
		if err != nil {
			return err
		}
		return toolRunAction("RearrangePDFPages", 6, map[string]interface{}{
			"sortInfo": []map[string]interface{}{{
				"file":  *entry,
				"pages": pages,
			}},
		})
	},
}

var toolsRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "旋转 PDF 页面",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		pagesRaw, _ := cmd.Flags().GetString("pages")
		if strings.EqualFold(strings.TrimSpace(pagesRaw), "all") {
			return clierr.ParamError("当前仅支持指定页码旋转", "使用 --pages 1,2,3 这样的页码列表")
		}
		pages, err := toolParseIntList(pagesRaw)
		if err != nil {
			return err
		}
		angle, _ := cmd.Flags().GetInt("angle")
		angle, err = toolNormalizeRotateAngle(angle)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 16)
		if err != nil {
			return err
		}
		return toolRunAction("RotatePDFPages", 16, map[string]interface{}{
			"rotateInfo": []map[string]interface{}{{
				"file":   *entry,
				"rotate": toolBuildRotateList(pages, angle),
			}},
		})
	},
}

func toolSplitMode(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "pages-per-pdf", "pagesperpdf":
		return "pagesPerPdf", nil
	case "even-odd", "evenodd":
		return "evenOdd", nil
	case "cut-in-half", "cutinhalf":
		return "cutInHalf", nil
	case "custom":
		return "custom", nil
	default:
		return "", clierr.ParamError("不支持的拆分模式", "可选值: pages-per-pdf, even-odd, cut-in-half, custom")
	}
}

func toolNormalizeRotateAngle(angle int) (int, error) {
	switch angle {
	case -270:
		return 90, nil
	case -180:
		return 180, nil
	case -90:
		return 270, nil
	case 90, 180, 270:
		return angle, nil
	default:
		return 0, clierr.ParamError("不支持的旋转角度", "可选值: 90, 180, 270")
	}
}

func init() {
	toolsConvertPDFToWordCmd.Flags().String("file", "", "待转换 PDF 文件路径")
	_ = toolsConvertPDFToWordCmd.MarkFlagRequired("file")
	toolsConvertCmd.AddCommand(toolsConvertPDFToWordCmd)

	toolsMergeCmd.Flags().StringSlice("files", nil, "待合并 PDF 文件路径列表")
	toolsMergeCmd.Flags().Bool("create-bookmarks", false, "合并时创建书签")
	_ = toolsMergeCmd.MarkFlagRequired("files")

	toolsSplitCmd.Flags().String("file", "", "待拆分 PDF 文件路径")
	toolsSplitCmd.Flags().String("mode", "pages-per-pdf", "拆分模式: pages-per-pdf, even-odd, cut-in-half, custom")
	toolsSplitCmd.Flags().Int("pages-per-pdf", 0, "按固定页数拆分时每个文件的页数")
	toolsSplitCmd.Flags().String("split-points", "", "自定义拆分页码，例如 1,3,5")
	_ = toolsSplitCmd.MarkFlagRequired("file")

	toolsReorderCmd.Flags().String("file", "", "待重排 PDF 文件路径")
	toolsReorderCmd.Flags().String("order", "", "新的页码顺序，例如 3,1,2")
	_ = toolsReorderCmd.MarkFlagRequired("file")
	_ = toolsReorderCmd.MarkFlagRequired("order")

	toolsRotateCmd.Flags().String("file", "", "待旋转 PDF 文件路径")
	toolsRotateCmd.Flags().String("pages", "", "需要旋转的页码，例如 1,3,5")
	toolsRotateCmd.Flags().Int("angle", 90, "旋转角度: 90, 180, 270")
	_ = toolsRotateCmd.MarkFlagRequired("file")
	_ = toolsRotateCmd.MarkFlagRequired("pages")

	toolsCmd.AddCommand(toolsConvertCmd)
	toolsCmd.AddCommand(toolsMergeCmd)
	toolsCmd.AddCommand(toolsSplitCmd)
	toolsCmd.AddCommand(toolsReorderCmd)
	toolsCmd.AddCommand(toolsRotateCmd)
}
