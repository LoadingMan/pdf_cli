package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	clierr "pdf-cli/internal/errors"
	"pdf-cli/internal/output"

	"github.com/spf13/cobra"
)

var toolsWatermarkCmd = &cobra.Command{
	Use:   "watermark",
	Short: "给 PDF 添加文字水印",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		text, _ := cmd.Flags().GetString("text")
		if strings.TrimSpace(text) == "" {
			return clierr.ParamError("请提供水印文本", "使用 --text 参数")
		}
		fontSize, _ := cmd.Flags().GetInt("font-size")
		color, _ := cmd.Flags().GetString("color")
		alpha, _ := cmd.Flags().GetString("alpha")
		position, _ := cmd.Flags().GetString("position")
		angle, _ := cmd.Flags().GetString("angle")
		spaceX, _ := cmd.Flags().GetString("space-x")
		spaceY, _ := cmd.Flags().GetString("space-y")
		entry, err := toolUploadFile(filePath, 17)
		if err != nil {
			return err
		}
		return toolRunAction("AddWatermark", 17, map[string]interface{}{
			"files":      []FileEntry{*entry},
			"pattern":    text,
			"position":   position,
			"fontFamily": "sans",
			"fontWeight": "normal",
			"fontStyle":  "normal",
			"fontSize":   fmt.Sprintf("%d", fontSize),
			"color":      color,
			"alpha":      alpha,
			"angle":      angle,
			"spaceX":     spaceX,
			"spaceY":     spaceY,
		})
	},
}

var toolsExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "提取 PDF 内容",
}

var toolsExtractImageCmd = &cobra.Command{
	Use:   "image",
	Short: "提取 PDF 图片",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 9)
		if err != nil {
			return err
		}
		return toolRunAction("ExtractImage", 9, map[string]interface{}{
			"files": []FileEntry{*entry},
		})
	},
}

var toolsExtractTextCmd = &cobra.Command{
	Use:   "text",
	Short: "提取 PDF 文本",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 13)
		if err != nil {
			return err
		}
		return toolRunAction("CoverPDFTo", 13, map[string]interface{}{
			"files": []FileEntry{*entry},
			"cover": "txt",
		})
	},
}

var toolsMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "修改 PDF 元数据",
}

var toolsMetadataSetCmd = &cobra.Command{
	Use:   "set",
	Short: "设置 PDF 元数据",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		title, _ := cmd.Flags().GetString("title")
		author, _ := cmd.Flags().GetString("author")
		subject, _ := cmd.Flags().GetString("subject")
		keywords, _ := cmd.Flags().GetString("keywords")
		if title == "" && author == "" && subject == "" && keywords == "" {
			return clierr.ParamError("请至少提供一个元数据字段", "使用 --title、--author、--subject 或 --keywords")
		}
		entry, err := toolUploadFile(filePath, 10)
		if err != nil {
			return err
		}
		return toolRunAction("EditPdfMetaData", 10, map[string]interface{}{
			"files":    []FileEntry{*entry},
			"title":    title,
			"author":   author,
			"subject":  subject,
			"keywords": keywords,
		})
	},
}

var toolsMetadataRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "移除 PDF 元数据",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 11)
		if err != nil {
			return err
		}
		return toolRunAction("RemovePdfMetaData", 11, map[string]interface{}{
			"files": []FileEntry{*entry},
		})
	},
}

var toolsCompressCmd = &cobra.Command{
	Use:   "compress",
	Short: "压缩 PDF",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		dpi, _ := cmd.Flags().GetInt("dpi")
		imageQuality, _ := cmd.Flags().GetInt("image-quality")
		grayscale, _ := cmd.Flags().GetBool("grayscale")
		colorMode, _ := cmd.Flags().GetString("color-mode")
		entry, err := toolUploadFile(filePath, 14)
		if err != nil {
			return err
		}
		data := map[string]interface{}{
			"files":        []FileEntry{*entry},
			"dpi":          dpi,
			"imageQuality": imageQuality,
		}
		switch strings.ToLower(strings.TrimSpace(colorMode)) {
		case "gray", "grey", "grayscale":
			data["colorMode"] = "Gray"
		case "color", "":
			if grayscale {
				data["colorMode"] = "Gray"
			}
		default:
			return clierr.ParamError("不支持的颜色模式", "可选值: color, gray")
		}
		return toolRunAction("CompressPDF", 14, data)
	},
}

var toolsSecurityCmd = &cobra.Command{
	Use:   "security",
	Short: "PDF 加密与解密",
}

var toolsSecurityEncryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "给 PDF 加密",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		password, _ := cmd.Flags().GetString("password")
		if strings.TrimSpace(password) == "" {
			return clierr.ParamError("请提供密码", "使用 --password 参数")
		}
		allowAssemble, _ := cmd.Flags().GetBool("allow-assemble")
		allowExtract, _ := cmd.Flags().GetBool("allow-extract")
		allowAccessibility, _ := cmd.Flags().GetBool("allow-accessibility")
		allowFillForm, _ := cmd.Flags().GetBool("allow-fill-form")
		allowModify, _ := cmd.Flags().GetBool("allow-modify")
		allowAnnotate, _ := cmd.Flags().GetBool("allow-annotate")
		allowPrint, _ := cmd.Flags().GetBool("allow-print")
		allowPrintHQ, _ := cmd.Flags().GetBool("allow-print-hq")
		entry, err := toolUploadFile(filePath, 3)
		if err != nil {
			return err
		}
		return toolRunAction("LockPDF", 3, map[string]interface{}{
			"files":                      []FileEntry{*entry},
			"canAssembleDocument":        allowAssemble,
			"canExtractContent":          allowExtract,
			"canExtractForAccessibility": allowAccessibility,
			"canFillInForm":              allowFillForm,
			"canModify":                  allowModify,
			"canModifyAnnotations":       allowAnnotate,
			"canPrint":                   allowPrint,
			"canPrintHighQuality":        allowPrintHQ,
			"userPass":                   password,
		})
	},
}

var toolsSecurityDecryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "移除 PDF 密码",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		password, _ := cmd.Flags().GetString("password")
		if strings.TrimSpace(password) == "" {
			return clierr.ParamError("请提供密码", "使用 --password 参数")
		}
		entry, err := toolUploadFile(filePath, 4)
		if err != nil {
			return err
		}
		return toolRunAction("UnlockPDF", 4, map[string]interface{}{
			"files":    []FileEntry{*entry},
			"userPass": password,
		})
	},
}

var toolsOverlayCmd = &cobra.Command{
	Use:   "overlay",
	Short: "叠加两个 PDF",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		overlayFile, _ := cmd.Flags().GetString("overlay-file")
		if strings.TrimSpace(overlayFile) == "" {
			return clierr.ParamError("请提供叠加 PDF 文件路径", "使用 --overlay-file 参数")
		}
		repeatLastPage, _ := cmd.Flags().GetBool("repeat-last-overlay-page")
		position, _ := cmd.Flags().GetString("position")
		baseEntry, err := toolUploadFile(filePath, 15)
		if err != nil {
			return err
		}
		overlayEntry, err := toolUploadFile(overlayFile, 15)
		if err != nil {
			return err
		}
		return toolRunAction("OverlayPDF", 15, map[string]interface{}{
			"files":                 []FileEntry{*baseEntry},
			"allPagesOverlay":       *overlayEntry,
			"repeatLastOverlayPage": repeatLastPage,
			"overlayPosition":       position,
		})
	},
}

var toolsPageCmd = &cobra.Command{
	Use:   "page",
	Short: "PDF 页面提取与删除",
}

var toolsPageExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "提取指定页面为新 PDF",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		pagesRaw, _ := cmd.Flags().GetString("pages")
		pages, err := toolParsePageSpec(pagesRaw)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 12)
		if err != nil {
			return err
		}
		return toolRunAction("ExtractPdfPages", 12, map[string]interface{}{
			"extractInfo": toolBuildPageSelections(*entry, pages),
		})
	},
}

var toolsPageDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除指定页面",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		pagesRaw, _ := cmd.Flags().GetString("pages")
		pages, err := toolParsePageSpec(pagesRaw)
		if err != nil {
			return err
		}
		entry, err := toolUploadFile(filePath, 8)
		if err != nil {
			return err
		}
		return toolRunAction("RemovePDFPages", 8, map[string]interface{}{
			"removeInfo": toolBuildPageSelections(*entry, pages),
		})
	},
}

var toolsPageNumberCmd = &cobra.Command{
	Use:   "page-number",
	Short: "PDF 页码工具",
}

var toolsPageNumberAddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加页码",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		filePath, err := toolRequireFile(cmd)
		if err != nil {
			return err
		}
		pattern, _ := cmd.Flags().GetString("pattern")
		position, _ := cmd.Flags().GetString("position")
		fontFamily, _ := cmd.Flags().GetString("font-family")
		fontSize, _ := cmd.Flags().GetString("font-size")
		fontWeight, _ := cmd.Flags().GetString("font-weight")
		fontStyle, _ := cmd.Flags().GetString("font-style")
		color, _ := cmd.Flags().GetString("color")
		alpha, _ := cmd.Flags().GetString("alpha")
		angle, _ := cmd.Flags().GetString("angle")
		spaceX, _ := cmd.Flags().GetString("space-x")
		spaceY, _ := cmd.Flags().GetString("space-y")
		pageNumOffset, _ := cmd.Flags().GetString("page-num-offset")
		entry, err := toolUploadFile(filePath, 7)
		if err != nil {
			return err
		}
		return toolRunAction("AddPageNumbers", 7, map[string]interface{}{
			"files":         []FileEntry{*entry},
			"pattern":       pattern,
			"position":      position,
			"fontFamily":    fontFamily,
			"fontSize":      fontSize,
			"fontWeight":    fontWeight,
			"fontStyle":     fontStyle,
			"color":         color,
			"alpha":         alpha,
			"angle":         angle,
			"spaceX":        spaceX,
			"spaceY":        spaceY,
			"pageNumOffset": pageNumOffset,
		})
	},
}

var toolsJobCmd = &cobra.Command{
	Use:   "job",
	Short: "工具任务管理",
}

var toolsJobStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看工具任务状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		queryKey, err := toolRequireQueryKey(cmd)
		if err != nil {
			return err
		}
		resp, err := toolGetTask(queryKey)
		if err != nil {
			return err
		}
		return toolRenderJobStatus(queryKey, resp)
	},
}

var toolsJobDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "下载工具任务结果",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		queryKey, err := toolRequireQueryKey(cmd)
		if err != nil {
			return err
		}
		resp, err := toolPollTask(queryKey)
		if err != nil {
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
	},
}

func toolRenderJobStatus(queryKey string, resp *toolEnvelope) error {
	format := output.GetFormat(formatFlag)
	var result interface{}
	_ = json.Unmarshal(resp.Data, &result)
	state := strings.ToUpper(resp.State)
	if format == "json" {
		output.PrintJSON(map[string]interface{}{
			"queryKey": queryKey,
			"jobId":    queryKey,
			"state":    resp.State,
			"result":   result,
		})
		if state == "FAILURE" {
			return toolTaskFailureError(queryKey, resp)
		}
		return nil
	}
	switch state {
	case "SUCCESS":
		return toolPrintTaskResult(queryKey, resp)
	case "FAILURE":
		fmt.Println("任务状态: 失败")
		output.PrintOrderedPretty([]string{"查询键", "错误信息"}, map[string]interface{}{
			"查询键":  queryKey,
			"错误信息": resp.Message,
		})
		return toolTaskFailureError(queryKey, resp)
	default:
		fmt.Println("任务状态: 处理中")
		output.PrintOrderedPretty([]string{"查询键", "状态"}, map[string]interface{}{
			"查询键": queryKey,
			"状态":   resp.State,
		})
	}
	return nil
}

func init() {
	toolsWatermarkCmd.Flags().String("file", "", "待添加水印的 PDF 文件路径")
	toolsWatermarkCmd.Flags().String("text", "", "水印文本")
	toolsWatermarkCmd.Flags().Int("font-size", 40, "水印字体大小")
	toolsWatermarkCmd.Flags().String("color", "#000000", "水印颜色")
	toolsWatermarkCmd.Flags().String("alpha", "0.4", "透明度")
	toolsWatermarkCmd.Flags().String("position", "center-center", "位置: top-left, top-center, top-right, center-left, center-center, center-right, bottom-left, bottom-center, bottom-right")
	toolsWatermarkCmd.Flags().String("angle", "-45", "旋转角度")
	toolsWatermarkCmd.Flags().String("space-x", "5", "水平间距")
	toolsWatermarkCmd.Flags().String("space-y", "5", "垂直间距")
	_ = toolsWatermarkCmd.MarkFlagRequired("file")
	_ = toolsWatermarkCmd.MarkFlagRequired("text")

	toolsExtractImageCmd.Flags().String("file", "", "待提取图片的 PDF 文件路径")
	_ = toolsExtractImageCmd.MarkFlagRequired("file")
	toolsExtractTextCmd.Flags().String("file", "", "待提取文本的 PDF 文件路径")
	_ = toolsExtractTextCmd.MarkFlagRequired("file")
	toolsExtractCmd.AddCommand(toolsExtractImageCmd)
	toolsExtractCmd.AddCommand(toolsExtractTextCmd)

	toolsMetadataSetCmd.Flags().String("file", "", "待修改元数据的 PDF 文件路径")
	toolsMetadataSetCmd.Flags().String("title", "", "标题")
	toolsMetadataSetCmd.Flags().String("author", "", "作者")
	toolsMetadataSetCmd.Flags().String("subject", "", "主题")
	toolsMetadataSetCmd.Flags().String("keywords", "", "关键字")
	_ = toolsMetadataSetCmd.MarkFlagRequired("file")
	toolsMetadataRemoveCmd.Flags().String("file", "", "待移除元数据的 PDF 文件路径")
	_ = toolsMetadataRemoveCmd.MarkFlagRequired("file")
	toolsMetadataCmd.AddCommand(toolsMetadataSetCmd)
	toolsMetadataCmd.AddCommand(toolsMetadataRemoveCmd)

	toolsCompressCmd.Flags().String("file", "", "待压缩 PDF 文件路径")
	toolsCompressCmd.Flags().Int("dpi", 144, "压缩 DPI")
	toolsCompressCmd.Flags().Int("image-quality", 75, "图片质量，范围 0-100")
	toolsCompressCmd.Flags().Bool("grayscale", false, "压缩时转为灰度")
	toolsCompressCmd.Flags().String("color-mode", "", "颜色模式: color, gray")
	_ = toolsCompressCmd.MarkFlagRequired("file")

	toolsSecurityEncryptCmd.Flags().String("file", "", "待加密 PDF 文件路径")
	toolsSecurityEncryptCmd.Flags().String("password", "", "加密密码")
	toolsSecurityEncryptCmd.Flags().Bool("allow-assemble", true, "允许组装文档")
	toolsSecurityEncryptCmd.Flags().Bool("allow-extract", true, "允许提取内容")
	toolsSecurityEncryptCmd.Flags().Bool("allow-accessibility", true, "允许辅助功能提取")
	toolsSecurityEncryptCmd.Flags().Bool("allow-fill-form", true, "允许填写表单")
	toolsSecurityEncryptCmd.Flags().Bool("allow-modify", true, "允许修改内容")
	toolsSecurityEncryptCmd.Flags().Bool("allow-annotate", true, "允许修改注释")
	toolsSecurityEncryptCmd.Flags().Bool("allow-print", true, "允许打印")
	toolsSecurityEncryptCmd.Flags().Bool("allow-print-hq", true, "允许高质量打印")
	_ = toolsSecurityEncryptCmd.MarkFlagRequired("file")
	_ = toolsSecurityEncryptCmd.MarkFlagRequired("password")
	toolsSecurityDecryptCmd.Flags().String("file", "", "待解密 PDF 文件路径")
	toolsSecurityDecryptCmd.Flags().String("password", "", "解密密码")
	_ = toolsSecurityDecryptCmd.MarkFlagRequired("file")
	_ = toolsSecurityDecryptCmd.MarkFlagRequired("password")
	toolsSecurityCmd.AddCommand(toolsSecurityEncryptCmd)
	toolsSecurityCmd.AddCommand(toolsSecurityDecryptCmd)

	toolsOverlayCmd.Flags().String("file", "", "底层 PDF 文件路径")
	toolsOverlayCmd.Flags().String("overlay-file", "", "叠加 PDF 文件路径")
	toolsOverlayCmd.Flags().String("position", "background", "叠加位置: background, foreground")
	toolsOverlayCmd.Flags().Bool("repeat-last-overlay-page", false, "叠加页不足时重复最后一页")
	_ = toolsOverlayCmd.MarkFlagRequired("file")
	_ = toolsOverlayCmd.MarkFlagRequired("overlay-file")

	toolsPageExtractCmd.Flags().String("file", "", "待提取页面的 PDF 文件路径")
	toolsPageExtractCmd.Flags().String("pages", "", "页码范围，例如 1-3,5")
	_ = toolsPageExtractCmd.MarkFlagRequired("file")
	_ = toolsPageExtractCmd.MarkFlagRequired("pages")
	toolsPageDeleteCmd.Flags().String("file", "", "待删除页面的 PDF 文件路径")
	toolsPageDeleteCmd.Flags().String("pages", "", "页码范围，例如 1-3,5")
	_ = toolsPageDeleteCmd.MarkFlagRequired("file")
	_ = toolsPageDeleteCmd.MarkFlagRequired("pages")
	toolsPageCmd.AddCommand(toolsPageExtractCmd)
	toolsPageCmd.AddCommand(toolsPageDeleteCmd)

	toolsPageNumberAddCmd.Flags().String("file", "", "待添加页码的 PDF 文件路径")
	toolsPageNumberAddCmd.Flags().String("pattern", "{NUM}/{CNT}", "页码格式，例如 {NUM}/{CNT} 或 {NUM}")
	toolsPageNumberAddCmd.Flags().String("position", "bottom-right", "位置: top-left, top-center, top-right, center-left, center-center, center-right, bottom-left, bottom-center, bottom-right")
	toolsPageNumberAddCmd.Flags().String("font-family", "sans", "字体族")
	toolsPageNumberAddCmd.Flags().String("font-size", "8", "字体大小")
	toolsPageNumberAddCmd.Flags().String("font-weight", "normal", "字重: normal, bold")
	toolsPageNumberAddCmd.Flags().String("font-style", "italic", "字形: normal, italic")
	toolsPageNumberAddCmd.Flags().String("color", "#000000", "颜色")
	toolsPageNumberAddCmd.Flags().String("alpha", "0.8", "透明度")
	toolsPageNumberAddCmd.Flags().String("angle", "0", "旋转角度")
	toolsPageNumberAddCmd.Flags().String("space-x", "5", "水平间距")
	toolsPageNumberAddCmd.Flags().String("space-y", "5", "垂直间距")
	toolsPageNumberAddCmd.Flags().String("page-num-offset", "0", "起始页码偏移")
	_ = toolsPageNumberAddCmd.MarkFlagRequired("file")
	toolsPageNumberCmd.AddCommand(toolsPageNumberAddCmd)

	toolsJobStatusCmd.Flags().String("query-key", "", "任务查询键")
	toolsJobStatusCmd.Flags().String("job-id", "", "旧参数别名，将使用同一个查询键")
	toolsJobDownloadCmd.Flags().String("query-key", "", "任务查询键")
	toolsJobDownloadCmd.Flags().String("job-id", "", "旧参数别名，将使用同一个查询键")
	toolsJobCmd.AddCommand(toolsJobStatusCmd)
	toolsJobCmd.AddCommand(toolsJobDownloadCmd)

	toolsCmd.AddCommand(toolsWatermarkCmd)
	toolsCmd.AddCommand(toolsExtractCmd)
	toolsCmd.AddCommand(toolsMetadataCmd)
	toolsCmd.AddCommand(toolsCompressCmd)
	toolsCmd.AddCommand(toolsSecurityCmd)
	toolsCmd.AddCommand(toolsOverlayCmd)
	toolsCmd.AddCommand(toolsPageCmd)
	toolsCmd.AddCommand(toolsPageNumberCmd)
	toolsCmd.AddCommand(toolsJobCmd)
}
