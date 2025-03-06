package kun

import (
	"fmt"

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
	Example: "kun new demo-api",
	Short:   "\n _    _            \n| |  / )           \n| | / /_   _ ____  \n| |< <| | | |  _ \\ \n| | \\ \\ |_| | | | |\n|_|  \\_)____|_| |_| \n \n" + "\x1B[38;2;66;211;146mA\x1B[39m \x1B[38;2;67;209;149mC\x1B[39m\x1B[38;2;68;206;152mL\x1B[39m\x1B[38;2;69;204;155mI\x1B[39m \x1B[38;2;70;201;158mt\x1B[39m\x1B[38;2;71;199;162mo\x1B[39m\x1B[38;2;72;196;165mo\x1B[39m\x1B[38;2;73;194;168ml\x1B[39m \x1B[38;2;74;192;171mf\x1B[39m\x1B[38;2;75;189;174mo\x1B[39m\x1B[38;2;76;187;177mr\x1B[39m \x1B[38;2;77;184;180mb\x1B[39m\x1B[38;2;78;182;183mu\x1B[39m\x1B[38;2;79;179;186mi\x1B[39m\x1B[38;2;80;177;190ml\x1B[39m\x1B[38;2;81;175;193md\x1B[39m\x1B[38;2;82;172;196mi\x1B[39m\x1B[38;2;83;170;199mn\x1B[39m\x1B[38;2;83;167;202mg\x1B[39m \x1B[38;2;84;165;205mg\x1B[39m\x1B[38;2;85;162;208mo\x1B[39m \x1B[38;2;86;160;211ma\x1B[39m\x1B[38;2;87;158;215mp\x1B[39m\x1B[38;2;88;155;218ml\x1B[39m\x1B[38;2;89;153;221mi\x1B[39m\x1B[38;2;90;150;224mc\x1B[39m\x1B[38;2;91;148;227ma\x1B[39m\x1B[38;2;92;145;230mt\x1B[39m\x1B[38;2;93;143;233mi\x1B[39m\x1B[38;2;94;141;236mo\x1B[39m\x1B[38;2;95;138;239mn\x1B[39m\x1B[38;2;96;136;243m.\x1B[39m",
	Version: fmt.Sprintf("\n _    _            \n| |  / )           \n| | / /_   _ ____  \n| |< <| | | |  _ \\ \n| | \\ \\ |_| | | | |\n|_|  \\_)____|_| |_| \n \nkun %s - Copyright (c) 2024 kun\nReleased under the MIT License.\n\n", config.Version),
}

func init() {
	CmdRoot.AddCommand(new.CmdNew)
	CmdRoot.AddCommand(run.CmdRun)
	CmdRoot.AddCommand(upgrade.CmdUpgrade)
	CmdRoot.AddCommand(create.CmdCreate)

	create.CmdCreate.AddCommand(create.CmdCreateController)
	create.CmdCreate.AddCommand(create.CmdCreateService)
	create.CmdCreate.AddCommand(create.CmdCreateRepository)
	create.CmdCreate.AddCommand(create.CmdCreateAll)

	CmdRoot.AddCommand(wire.CmdWire)
	wire.CmdWire.AddCommand(wire.CmdWireAll)
}

// Execute executes the root command.
func Execute() error {
	return CmdRoot.Execute()
}
