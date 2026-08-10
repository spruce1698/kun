package kun

import (
	"github.com/spruce1698/kun/config"
	"github.com/spruce1698/kun/internal/create"
	"github.com/spruce1698/kun/internal/new"
	"github.com/spruce1698/kun/internal/run"
	"github.com/spruce1698/kun/internal/upgrade"
	"github.com/spruce1698/kun/internal/wire"

	"github.com/spf13/cobra"
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
	CmdRoot.AddCommand(new.CmdNew)
	CmdRoot.AddCommand(run.CmdRun)
	CmdRoot.AddCommand(upgrade.CmdUpgrade)
	CmdRoot.AddCommand(create.CmdCreate)

	create.CmdCreate.AddCommand(create.CmdCreateHandler)
	create.CmdCreate.AddCommand(create.CmdCreateService)
	create.CmdCreate.AddCommand(create.CmdCreateHandlerAndService)
	create.CmdCreate.AddCommand(create.CmdCreateRouter)
	create.CmdCreate.AddCommand(create.CmdCreateDBRepository)
	create.CmdCreate.AddCommand(create.CmdCreateCacheRepository)

	CmdRoot.AddCommand(wire.CmdWire)
	wire.CmdWire.AddCommand(wire.CmdWireAll)
}

// executes the root command.
func Execute() error {
	return CmdRoot.Execute()
}
