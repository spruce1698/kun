package output

import (
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

func init() {
	// 非 TTY(管道/重定向/CI 日志)下禁用 ANSI 转义,避免日志文件里满是颜色码。
	// fatih/color 自身也会探测,但显式设置 NoColor 更稳妥。
	color.NoColor = !isTerminal(os.Stdout)
}

// isTerminal 判断 f 是否为终端。
func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func Green(format string, args ...any) {
	color.Green(format, args...)
}

func Success(format string, args ...any) {
	color.Green(" [√] "+format, args...)
}

func Error(format string, args ...any) {
	// 错误信息走 stderr,即使 stdout 被重定向到管道也能正确展示与采集。
	c := color.New(color.FgRed)
	c.Fprintf(os.Stderr, " [X] "+format+"\n", args...)
}

func Warn(format string, args ...any) {
	// 警告同样走 stderr,与 Error 保持一致。
	c := color.New(color.FgYellow)
	c.Fprintf(os.Stderr, " [!] "+format+"\n", args...)
}
