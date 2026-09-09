package kun

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/config"
	"github.com/spruce1698/kun/internal/create"
	"github.com/spruce1698/kun/internal/new"
	"github.com/spruce1698/kun/internal/run"
	"github.com/spruce1698/kun/internal/upgrade"
	"github.com/spruce1698/kun/internal/wire"
)

var CmdRoot = &cobra.Command{
	Use:               "kun",
	Example:           "kun new demo",
	Short:             config.Short,
	Version:           config.Version,
	SilenceErrors:     true, // 子命令失败由其自行 output.Error 提示,避免重复打印 "Error: ..."
	SilenceUsage:      true, // 运行期错误不打印 usage,仅参数校验错误由 cobra 打印
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func init() {
	// E6: 各子命令包通过 Register 自行维护命令树，root 只负责顶层注册，
	// 新增子命令只需在对应包中修改，不会遗漏。
	new.Register(CmdRoot)
	run.Register(CmdRoot)
	upgrade.Register(CmdRoot)
	create.Register(CmdRoot)
	wire.Register(CmdRoot)
}

// ErrSilent 表示命令已自行处理输出（例如打印了 Help），调用方只需退出，无需再次打印错误。
var ErrSilent = errors.New("silent exit")

// IsArgsError 判断错误是否为参数个数相关错误。
func IsArgsError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "arg(s), received ") ||
		strings.Contains(msg, "arg, received ") ||
		(strings.Contains(msg, "accepts between ") && strings.Contains(msg, "arg")) ||
		(strings.Contains(msg, "requires at least ") && strings.Contains(msg, "arg")) ||
		(strings.Contains(msg, "accepts at most ") && strings.Contains(msg, "arg")) ||
		(strings.Contains(msg, "accepts ") && strings.Contains(msg, "arg(s)"))
}

// Execute executes the root command.
func Execute() error {
	cmd, err := CmdRoot.ExecuteC()
	if err != nil {
		if IsArgsError(err) {
			if cmd != nil {
				_ = cmd.Help()
			} else {
				_ = CmdRoot.Help()
			}
			return ErrSilent
		}
		return err
	}
	return nil
}
