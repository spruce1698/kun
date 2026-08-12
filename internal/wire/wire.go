package wire

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/pkg/helper"
	"github.com/spruce1698/kun/pkg/output"
)

var CmdWire = &cobra.Command{
	Use:     "wire",
	Short:   "kun wire [wire.go path]",
	Long:    "kun wire [wire.go path]",
	Example: "kun wire server/wire",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdArgs, _ := helper.SplitArgs(cmd, args)
		var dir string
		if len(cmdArgs) > 0 {
			dir = cmdArgs[0]
		}
		base, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd error: %w", err)
		}
		if dir == "" {
			// find the directory containing the cmd/*
			wirePath, err := findWire(base)

			if err != nil {
				return err
			}
			switch len(wirePath) {
			case 0:
				return fmt.Errorf("the wire.go cannot be found in the current directory")
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
					// 交互中断,静默退出
					return nil
				}
				dir = wirePath[dir]
			}
		}
		return wireRun(dir)
	},
}
var CmdWireAll = &cobra.Command{
	Use:     "all",
	Short:   "kun wire all",
	Long:    "kun wire all",
	Example: "kun wire all",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdArgs, _ := helper.SplitArgs(cmd, args)
		var dir string
		if len(cmdArgs) > 0 {
			dir = cmdArgs[0]
		}
		base, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd error: %w", err)
		}
		// 指定了目录:以该目录为根查找其下所有 wire.go 并逐个执行。
		// 此前这里直接 return nil,导致 `kun wire all <dir>` 静默什么都不做且退出码为 0。
		searchRoot := base
		if dir != "" {
			if filepath.IsAbs(dir) {
				searchRoot = dir
			} else {
				searchRoot = filepath.Join(base, dir)
			}
		}

		wirePath, err := findWire(searchRoot)
		if err != nil {
			return err
		}
		if len(wirePath) == 0 {
			return fmt.Errorf("the wire.go cannot be found in %s", searchRoot)
		}
		// 逐个执行;收集错误但继续处理剩余 wire.go,最终汇总返回。
		var errs []error
		for _, v := range wirePath {
			if err := wireRun(v); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	},
}

// wireRun 在指定目录执行 wire 命令。
func wireRun(wirePath string) error {
	output.Success("wire.go path: %s", wirePath)
	cmd := exec.Command("wire")
	cmd.Dir = wirePath
	out, err := cmd.CombinedOutput()
	output.Success(string(out))
	if err != nil {
		return fmt.Errorf("wire fail (%s): %w", wirePath, err)
	}
	return nil
}
func findWire(base string) (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	// 统一为正斜杠,避免 Windows 下 / 与 \ 混合导致 TrimPrefix 失效
	wd = filepath.ToSlash(wd)
	if !strings.HasSuffix(wd, "/") {
		wd += "/"
	}

	// walkDir 在 dir 范围内搜索 wire.go 文件，返回找到的 wire.go 路径映射和是否已到达项目根目录（含 go.mod）。
	// 注意: walkErr 使用局部 := 声明,避免捕获/污染外层 err。
	walkDir := func(dir string) (map[string]string, bool, error) {
		wirePath := make(map[string]string)
		foundRoot := false
		walkErr := filepath.Walk(dir, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// 多级目录不在 wire.go 所在目录下搜索
			if strings.HasSuffix(walkPath, "wire.go") {
				p, _ := filepath.Split(walkPath)
				wirePath[strings.TrimPrefix(filepath.ToSlash(walkPath), wd)] = p
				return nil
			}
			if info.Name() == "go.mod" {
				foundRoot = true
			}
			return nil
		})
		return wirePath, foundRoot, walkErr
	}

	// 向上最多搜索 5 层目录。正常 monorepo/嵌套项目 cmd 层级不会超过此深度;
	// 到达含 go.mod 的根目录即停止。若仍找不到,返回空映射由调用方报错。
	const maxUpLevels = 5
	for i := 0; i < maxUpLevels; i++ {
		cmd, reachedRoot, walkErr := walkDir(base)
		if walkErr != nil {
			return nil, walkErr
		}
		if len(cmd) > 0 {
			return cmd, nil
		}
		if reachedRoot {
			break
		}
		base = filepath.Join(base, "..")
	}
	return map[string]string{}, nil
}
