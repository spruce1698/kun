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
	color.Red(" [X] "+format, args...)
}

func Warn(format string, args ...any) {
	color.Yellow(" [!] "+format, args...)
}
