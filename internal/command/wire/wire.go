package wire

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/pkg/fmt"
	"github.com/spruce1698/kun/pkg/helper"
)

var CmdWire = &cobra.Command{
	Use:     "wire",
	Short:   "kun wire [wire.go path]",
	Long:    "kun wire [wire.go path]",
	Example: "kun wire server/wire",
	Run: func(cmd *cobra.Command, args []string) {
		cmdArgs, _ := helper.SplitArgs(cmd, args)
		var dir string
		if len(cmdArgs) > 0 {
			dir = cmdArgs[0]
		}
		base, err := os.Getwd()
		if err != nil {
			fmt.Error("Error: %s", err)
			return
		}
		if dir == "" {
			// find the directory containing the cmd/*
			wirePath, err := findWire(base)

			if err != nil {
				fmt.Error("Error: %s", err)
				return
			}
			switch len(wirePath) {
			case 0:
				fmt.Error("Error: The wire.go cannot be found in the current directory")
				return
			case 1:
				for _, v := range wirePath {
					dir = v
				}
			default:
				var wirePaths []string
				for k := range wirePath {
					wirePaths = append(wirePaths, k)
				}
				sort.Strings(wirePaths)
				prompt := &survey.Select{
					Message:  "Which directory do you want to run?",
					Options:  wirePaths,
					PageSize: 10,
				}
				e := survey.AskOne(prompt, &dir)
				if e != nil || dir == "" {
					return
				}
				dir = wirePath[dir]
			}
		}
		wire(dir)
	},
}
var CmdWireAll = &cobra.Command{
	Use:     "all",
	Short:   "kun wire all",
	Long:    "kun wire all",
	Example: "kun wire all",
	Run: func(cmd *cobra.Command, args []string) {
		cmdArgs, _ := helper.SplitArgs(cmd, args)
		var dir string
		if len(cmdArgs) > 0 {
			dir = cmdArgs[0]
		}
		base, err := os.Getwd()
		if err != nil {
			fmt.Error("Error: %s", err)
			return
		}
		if dir == "" {
			// find the directory containing the cmd/*
			wirePath, err := findWire(base)

			if err != nil {
				fmt.Error("Error: %s", err)
				return
			}
			switch len(wirePath) {
			case 0:
				fmt.Error("Error: The wire.go cannot be found in the current directory")
				return
			default:
				for _, v := range wirePath {
					wire(v)
				}
			}
		}

	},
}

func wire(wirePath string) {
	fmt.Success("wire.go path: %s", wirePath)
	cmd := exec.Command("wire")
	cmd.Dir = wirePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Error("wire fail: %s", err)
	}
	fmt.Success(string(out))
}
func findWire(base string) (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(wd, "/") {
		wd += "/"
	}

	var root bool
	next := func(dir string) (map[string]string, error) {
		wirePath := make(map[string]string)
		err = filepath.Walk(dir, func(walkPath string, info os.FileInfo, err error) error {
			// multi level directory is not allowed under the wirePath directory, so it is judged that the path ends with wirePath.
			if strings.HasSuffix(walkPath, "wire.go") {
				p, _ := filepath.Split(walkPath)
				wirePath[strings.TrimPrefix(walkPath, wd)] = p
				return nil
			}
			if info.Name() == "go.mod" {
				root = true
			}
			return nil
		})
		return wirePath, err
	}
	for i := 0; i < 5; i++ {
		tmp := base
		cmd, err := next(tmp)
		if err != nil {
			return nil, err
		}
		if len(cmd) > 0 {
			return cmd, nil
		}
		if root {
			break
		}
		_ = filepath.Join(base, "..")
	}
	return map[string]string{"": base}, nil
}
