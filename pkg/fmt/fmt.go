package fmt

import (
	"fmt"

	"github.com/fatih/color"
)

// 重导出标准库 fmt 的常用函数，使导入 pkg/fmt 的包可直接使用 fmt.Sprintf / fmt.Errorf / fmt.Fscanf。
var (
	Fscanf  = fmt.Fscanf
	Sprintf = fmt.Sprintf
	Errorf  = fmt.Errorf
)

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
