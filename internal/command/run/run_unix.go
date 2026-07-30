//go:build !plan9 && !windows
// +build !plan9,!windows

package run

import (
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/config"
	"github.com/spruce1698/kun/pkg/fmt"
	"github.com/spruce1698/kun/pkg/helper"
)

var quit = make(chan os.Signal, 1)

type Run struct {
}

var excludeDir string
var includeExt string

func init() {
	CmdRun.Flags().StringVarP(&excludeDir, "excludeDir", "", excludeDir, `eg: kun run --excludeDir="tmp,vendor,.git,.idea"`)
	CmdRun.Flags().StringVarP(&includeExt, "includeExt", "", includeExt, `eg: kun run --includeExt="go,tpl,tmpl,html,yaml,yml,toml,ini,json"`)
	if excludeDir == "" {
		excludeDir = config.RunExcludeDir
	}
	if includeExt == "" {
		includeExt = config.RunIncludeExt
	}
}

var CmdRun = &cobra.Command{
	Use:     "run",
	Short:   "kun run [main.go path]",
	Long:    "kun run [main.go path]",
	Example: "kun run cmd",
	Run: func(cmd *cobra.Command, args []string) {
		cmdArgs, programArgs := helper.SplitArgs(cmd, args)
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
			cmdPath, err := helper.FindMain(base, excludeDir)

			if err != nil {
				fmt.Error("Error: %s", err)
				return
			}
			switch len(cmdPath) {
			case 0:
				fmt.Error("Error: The cmd directory cannot be found in the current directory")
				return
			case 1:
				for _, v := range cmdPath {
					dir = v
				}
			default:
				var cmdPaths []string
				for k := range cmdPath {
					cmdPaths = append(cmdPaths, k)
				}
				sort.Strings(cmdPaths)

				prompt := &survey.Select{
					Message:  "Which directory do you want to run?",
					Options:  cmdPaths,
					PageSize: 10,
				}
				e := survey.AskOne(prompt, &dir)
				if e != nil || dir == "" {
					return
				}
				dir = cmdPath[dir]
			}
		}
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		fmt.Success("kun run %s.", dir)
		fmt.Success("Watch excludeDir %s", excludeDir)
		fmt.Success("Watch includeExt %s", includeExt)
		watch(dir, programArgs)
	},
}

func watch(dir string, programArgs []string) {

	// Listening file path
	watchPath := "./"

	// Create a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Error("Error: %s", err)
		return
	}
	defer watcher.Close()

	excludeDirArr := strings.Split(excludeDir, ",")

	// 添加所有非排除目录到 watcher（监听目录而非单个文件，新文件自动被覆盖）
	err = filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		for _, s := range excludeDirArr {
			if s != "" && (path == s || strings.HasPrefix(path, s+"/") || strings.HasPrefix(path, s+"\\")) {
				return filepath.SkipDir
			}
		}
		if addErr := watcher.Add(path); addErr != nil {
			fmt.Error("Error: %s", addErr)
		}
		return nil
	})
	if err != nil {
		fmt.Error("Error: %s", err)
		return
	}

	cmd := start(dir, programArgs)

	// Loop listening file modification
	for {
		select {
		case <-quit:
			if cmd.Process != nil {
				err = syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
				if err != nil {
					fmt.Error("server exit error: %s", err)
					return
				}
			}
			fmt.Success("server exiting...")
			os.Exit(0)

		case event := <-watcher.Events:
			// 文件被修改或删除时重启
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Success("file modified: %s", event.Name)
				if cmd.Process != nil {
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				}
				cmd = start(dir, programArgs)
			}
			// 新建目录时加入 watcher，使其内部文件也被监听
			if event.Op&fsnotify.Create == fsnotify.Create {
				evPath := strings.ReplaceAll(event.Name, "\\", "/")
				for _, s := range excludeDirArr {
					if s != "" && (evPath == s || strings.HasPrefix(evPath, s+"/")) {
						goto skipWatch
					}
				}
				if fi, fiErr := os.Stat(event.Name); fiErr == nil && fi.IsDir() {
					_ = watcher.Add(event.Name)
				}
			skipWatch:
			}
		case err := <-watcher.Errors:
			fmt.Error("Error: %s", err)
		}
	}
}

func isProcessRunning(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	err := cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

func start(dir string, programArgs []string) *exec.Cmd {
	cmd := exec.Command("go", append([]string{"run", dir}, programArgs...)...)
	// Set a new process group to kill all child processes when the program exits
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Error("cmd run failed")
	}
	// 等待进程真正启动，而非固定 sleep 1 秒
	time.Sleep(200 * time.Millisecond)
	if !isProcessRunning(cmd) {
		fmt.Error("process exited immediately after start")
	}
	fmt.Success("running...")
	return cmd
}
