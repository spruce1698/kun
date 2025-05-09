package upgrade

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/config"
	"github.com/spruce1698/kun/pkg/fmt"
)

var CmdUpgrade = &cobra.Command{
	Use:     "upgrade",
	Short:   "Upgrade the kun command.",
	Long:    "Upgrade the kun command.",
	Example: "kun upgrade",
	Run: func(_ *cobra.Command, _ []string) {
		cmd := exec.Command("go", "clean", "-modcache")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		fmt.Success("go install %s", config.KunUrl)
		cmd = exec.Command("go", "install", config.KunUrl)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Error("go install %s error", err)
		}
		fmt.Success("kun upgrade successfully!")
	},
}
