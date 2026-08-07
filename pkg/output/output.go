package output

import (
	"github.com/fatih/color"
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
