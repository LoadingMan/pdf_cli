package cmd

import (
	"os"
	"strings"

	"pdf-cli/internal/build"
	clierr "pdf-cli/internal/errors"

	"github.com/spf13/cobra"
)

var (
	formatFlag string
	outputFlag string
)

var rootCmd = &cobra.Command{
	Use:     "pdf-cli",
	Short:   "PDF 翻译与工具命令行客户端",
	Version: build.Version,
	Long: `pdf-cli 是一个面向用户的 PDF 翻译与工具命令行客户端。

支持以下功能模块：
  translate  - PDF 翻译、AI 翻译、术语库
  tools      - PDF 合并、转换、拆分、旋转、压缩、页面处理与安全工具
  auth       - 登录、登出、账号状态
  user       - 个人资料、文件记录、API Key、团队
  member     - 会员信息、权益、订单
  other      - 公告、版本、帮助

快速开始：
  pdf-cli auth login --email you@example.com
  pdf-cli translate upload --file ./paper.pdf
  pdf-cli translate start --file-key xxx --to zh
  pdf-cli tools merge --files a.pdf,b.pdf
  pdf-cli tools job status --query-key <key>`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		clierr.SetJSONMode(strings.ToLower(formatFlag) == "json")
	},
}

func Execute() {
	// JSON error envelope must be available even when command resolution
	// fails (unknown subcommand, unknown flag), which happens before
	// PersistentPreRun. Sniff --format from raw argv up front.
	if jsonModeFromArgs(os.Args[1:]) {
		clierr.SetJSONMode(true)
	}
	applyFlagErrorFunc(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		clierr.Handle(remapCobraError(err))
	}
}

func jsonModeFromArgs(args []string) bool {
	for i, a := range args {
		if a == "--format=json" || a == "--format=JSON" {
			return true
		}
		if (a == "--format" || a == "-format") && i+1 < len(args) {
			return strings.EqualFold(args[i+1], "json")
		}
	}
	return false
}

// remapCobraError converts cobra's unknown-command / unknown-flag errors
// (which arrive as plain errors, not *CLIError) into UsageError so they
// exit with code 2 instead of falling through to Internal=20.
func remapCobraError(err error) error {
	if _, ok := err.(*clierr.CLIError); ok {
		return err
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	if strings.Contains(low, "unknown command") ||
		strings.Contains(low, "unknown flag") ||
		strings.Contains(low, "unknown shorthand") ||
		strings.Contains(low, "invalid argument") {
		return clierr.UsageError(msg, "请检查命令拼写与参数，参考 --help")
	}
	return err
}

// applyFlagErrorFunc walks the command tree and hardens every node:
//
//   - install the usage-error remapper (cobra does not propagate
//     SetFlagErrorFunc to children).
//   - silence cobra's auto usage/error printing on subcommands so the
//     JSON envelope is the only stderr output in --format json mode.
//   - install a RunE on parent group commands that have none, so
//     `pdf-cli translate notacommand` exits 2 (Usage) instead of
//     silently printing help and exiting 0. cobra's Args: NoArgs does
//     not catch this case for parents that own subcommands.
func applyFlagErrorFunc(cmd *cobra.Command) {
	cmd.SetFlagErrorFunc(remapUsageError)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if cmd.HasSubCommands() && cmd.RunE == nil && cmd.Run == nil {
		cmd.RunE = parentGroupRunE
	}
	for _, sub := range cmd.Commands() {
		applyFlagErrorFunc(sub)
	}
}

// parentGroupRunE is the default RunE for group commands. With no args
// it prints help and exits 0; with leftover args it returns a UsageError
// because the user typed an unknown subcommand.
func parentGroupRunE(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return clierr.UsageError(
			"unknown subcommand: "+strings.Join(args, " "),
			"运行 "+cmd.CommandPath()+" --help 查看可用子命令",
		)
	}
	return cmd.Help()
}

// remapUsageError converts cobra's flag-parsing errors into a typed CLIError
// so they exit with code 2 (Usage) instead of falling through to ExitInternal.
// Required-flag and unknown-flag errors carry distinct semantics: missing
// required is invalid_argument (3), structural problems are usage (2).
func remapUsageError(_ *cobra.Command, err error) error {
	msg := err.Error()
	low := strings.ToLower(msg)
	if strings.Contains(low, "required flag") {
		return clierr.ParamError(msg, "请补齐必填参数后重试")
	}
	return clierr.UsageError(msg, "请检查命令拼写与参数，参考 --help")
}

func init() {
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "", "输出格式: json, table, pretty (默认 pretty)")
	rootCmd.PersistentFlags().StringVar(&outputFlag, "output", "", "输出到文件路径")
	rootCmd.SetFlagErrorFunc(remapUsageError)
}
