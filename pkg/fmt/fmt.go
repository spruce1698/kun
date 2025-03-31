package fmt

import (
    "fmt"
    
    "github.com/fatih/color"
)

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
