package kun

import (
    "github.com/spruce1698/kun/config"
    "github.com/spruce1698/kun/internal/command/wire"
    
    "github.com/spf13/cobra"
    "github.com/spruce1698/kun/internal/command/create"
    "github.com/spruce1698/kun/internal/command/new"
    "github.com/spruce1698/kun/internal/command/run"
    "github.com/spruce1698/kun/internal/command/upgrade"
)

var CmdRoot = &cobra.Command{
    Use:     "kun",
    Example: "kun new demo",
    Short:   config.Short,
    Version: config.Short,
}

func init() {
    CmdRoot.AddCommand(new.CmdNew)
    CmdRoot.AddCommand(run.CmdRun)
    CmdRoot.AddCommand(upgrade.CmdUpgrade)
    CmdRoot.AddCommand(create.CmdCreate)
    
    create.CmdCreate.AddCommand(create.CmdCreateController)
    create.CmdCreate.AddCommand(create.CmdCreateService)
    create.CmdCreate.AddCommand(create.CmdCreateDBRepository)
    create.CmdCreate.AddCommand(create.CmdCreateCacheRepository)
    create.CmdCreate.AddCommand(create.CmdCreateCAndS)
    
    CmdRoot.AddCommand(wire.CmdWire)
    wire.CmdWire.AddCommand(wire.CmdWireAll)
}

// executes the root command.
func Execute() error {
    return CmdRoot.Execute()
}
