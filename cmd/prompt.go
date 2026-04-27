package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// openTTY 直接打开 /dev/tty，绕过 stdin/stdout 被管道/IDE 包装导致 IsTerminal 误判的场景。
// 返回文件句柄用于读写；若无控制终端则返回 nil。
func openTTY() *os.File {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	if !term.IsTerminal(int(f.Fd())) {
		_ = f.Close()
		return nil
	}
	return f
}

// selectOption 选项选择：优先用 /dev/tty 的箭头键菜单；没有 TTY 时改从 stdin 读编号。
// stdin 也无输入（EOF / 空行 / 无效）时打印日志并返回默认值，不阻塞 CI。
// Ctrl-C / EOF 返回错误表示取消。
func selectOption(title string, options []string, defaultIdx int) (int, error) {
	idx, _, err := selectOptionStrict(title, options, defaultIdx)
	return idx, err
}

// selectOptionStrict 与 selectOption 一致，额外返回 auto=true 表示"无任何人工输入、用的默认值"
// 场景：既没有 /dev/tty，stdin 读取也立即 EOF 或空行。
// 调用方据此决定是否把默认值视作错误（例如目标语言这种关键决策）。
func selectOptionStrict(title string, options []string, defaultIdx int) (int, bool, error) {
	if len(options) == 0 {
		return -1, false, fmt.Errorf("selectOption: 空选项")
	}
	if defaultIdx < 0 || defaultIdx >= len(options) {
		defaultIdx = 0
	}

	if tty := openTTY(); tty != nil {
		defer tty.Close()
		idx, err := selectOptionTTY(tty, title, options, defaultIdx)
		return idx, false, err
	}

	// 无控制 TTY → 尝试 stdin；stdin 读取返回 (idx, auto, err)
	return selectOptionStdinStrict(title, options, defaultIdx)
}

// selectOptionTTY 通过 /dev/tty 渲染箭头键选择菜单。
func selectOptionTTY(tty *os.File, title string, options []string, defaultIdx int) (int, error) {
	fd := int(tty.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return defaultIdx, err
	}
	defer term.Restore(fd, oldState)

	sel := defaultIdx
	writeTTY := func(s string) { _, _ = tty.WriteString(s) }

	writeTTY(fmt.Sprintf("%s  (↑/↓ 选择, Enter 确认, Ctrl-C 取消)\r\n", title))
	for i, opt := range options {
		writeTTY(renderLine(i == sel, opt) + "\r\n")
	}

	redraw := func() {
		writeTTY(fmt.Sprintf("\x1b[%dA", len(options)))
		for i, opt := range options {
			writeTTY("\r\x1b[K" + renderLine(i == sel, opt) + "\r\n")
		}
	}

	buf := make([]byte, 3)
	for {
		n, err := tty.Read(buf)
		if err != nil || n == 0 {
			return defaultIdx, err
		}
		if n >= 3 && buf[0] == 0x1b && buf[1] == '[' {
			prev := sel
			switch buf[2] {
			case 'A':
				if sel > 0 {
					sel--
				}
			case 'B':
				if sel < len(options)-1 {
					sel++
				}
			default:
				continue
			}
			if prev != sel {
				redraw()
			}
			continue
		}
		switch buf[0] {
		case '\r', '\n':
			writeTTY("\r\n")
			return sel, nil
		case 0x03, 0x04:
			writeTTY("\r\n")
			return -1, fmt.Errorf("用户取消")
		case 'k', 'K':
			if sel > 0 {
				sel--
				redraw()
			}
		case 'j', 'J':
			if sel < len(options)-1 {
				sel++
				redraw()
			}
		default:
			if buf[0] >= '1' && buf[0] <= '9' {
				idx := int(buf[0] - '1')
				if idx < len(options) {
					sel = idx
					redraw()
					writeTTY("\r\n")
					return sel, nil
				}
			}
		}
	}
}

// selectOptionStdinStrict stdin 回退：打印编号列表，读一行，解析数字。
// 返回 (idx, auto, err)：auto=true 表示 stdin EOF 或空行导致用了默认值（无人工输入）。
func selectOptionStdinStrict(title string, options []string, defaultIdx int) (int, bool, error) {
	fmt.Println(title)
	for i, opt := range options {
		marker := "  "
		if i == defaultIdx {
			marker = "▶ "
		}
		fmt.Printf("  %d) %s%s\n", i+1, marker, opt)
	}
	fmt.Printf("请输入编号 1-%d（回车使用默认 %d）： ", len(options), defaultIdx+1)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	trimmed := strings.TrimSpace(line)
	if err != nil {
		if trimmed == "" {
			fmt.Fprintf(os.Stderr, "[stdin 无输入] 自动选择：%s\n", options[defaultIdx])
			return defaultIdx, true, nil
		}
		// 读到了部分内容再遇错 → 按输入处理；失败则按默认
		n, perr := strconv.Atoi(trimmed)
		if perr != nil || n < 1 || n > len(options) {
			return defaultIdx, true, nil
		}
		return n - 1, false, nil
	}
	if trimmed == "" {
		return defaultIdx, true, nil
	}
	n, perr := strconv.Atoi(trimmed)
	if perr != nil || n < 1 || n > len(options) {
		fmt.Fprintf(os.Stderr, "输入无效 %q，使用默认：%s\n", trimmed, options[defaultIdx])
		return defaultIdx, true, nil
	}
	return n - 1, false, nil
}

// readPasswordInteractive 读取密码：优先 /dev/tty（不回显）；管道/无 TTY 时从 stdin 读一行（明文）。
// 用于 agent CLI、CI、测试等非 TTY 场景登录。
func readPasswordInteractive(prompt string) (string, error) {
	if tty := openTTY(); tty != nil {
		defer tty.Close()
		fmt.Fprint(tty, prompt)
		pw, err := term.ReadPassword(int(tty.Fd()))
		fmt.Fprintln(tty)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(pw), "\r\n"), nil
	}
	// 无控制 TTY → 从 stdin 读一行（注意：此时输入可见，适合脚本/pipe 场景）
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if err != nil && line == "" {
		return "", err
	}
	return line, nil
}

// selectYesNo 二选一（是/否）。
func selectYesNo(title string, defaultYes bool) (bool, error) {
	defaultIdx := 1
	if defaultYes {
		defaultIdx = 0
	}
	idx, err := selectOption(title, []string{"是", "否"}, defaultIdx)
	if err != nil {
		return defaultYes, err
	}
	return idx == 0, nil
}

func renderLine(selected bool, text string) string {
	if selected {
		return "\x1b[36m▶ " + text + "\x1b[0m"
	}
	return "  " + text
}
