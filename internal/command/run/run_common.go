package run

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/spruce1698/kun/config"
	"github.com/spruce1698/kun/pkg/helper"
	"github.com/spruce1698/kun/pkg/output"
)

var quit = make(chan os.Signal, 1)

var excludeDir string
var includeExt string

func init() {
	CmdRun.Flags().StringVarP(&excludeDir, "excludeDir", "", config.RunExcludeDir, `eg: kun run --excludeDir="tmp,vendor,.git,.idea"`)
	CmdRun.Flags().StringVarP(&includeExt, "includeExt", "", config.RunIncludeExt, `eg: kun run --includeExt="go,tpl,tmpl,html,yaml,yml,toml,ini,json"`)
}

var CmdRun = &cobra.Command{
	Use:     "run",
	Short:   "kun run [main.go path]",
	Long:    "kun run [main.go path]",
	Example: "kun run cmd",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdArgs, programArgs := helper.SplitArgs(cmd, args)
		var dir string
		if len(cmdArgs) > 0 {
			dir = cmdArgs[0]
		}
		base, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd error: %w", err)
		}
		if dir == "" {
			cmdPath, err := helper.FindMain(base, excludeDir)

			if err != nil {
				return err
			}
			switch len(cmdPath) {
			case 0:
				return fmt.Errorf("the cmd directory cannot be found in the current directory")
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
					// 交互中断,静默退出
					return nil
				}
				dir = cmdPath[dir]
			}
		}
		signal.Notify(quit, signals()...)
		output.Success("kun run %s.", dir)
		output.Success("Watch excludeDir %s", excludeDir)
		output.Success("Watch includeExt %s", includeExt)
		watch(dir, programArgs)
		return nil
	},
}

func watch(dir string, programArgs []string) {

	// Listening file path
	watchPath := "./"

	// Create a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		output.Error("Error: %s", err)
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
		if helper.IsExcluded(path, excludeDirArr) {
			return filepath.SkipDir
		}
		if addErr := watcher.Add(path); addErr != nil {
			output.Error("Error: %s", addErr)
		}
		return nil
	})
	if err != nil {
		output.Error("Error: %s", err)
		return
	}

	cmd := start(dir, programArgs)

	// 接收子进程(go run)退出,捕获编译/运行错误
	cmdExit := make(chan error, 1)
	if cmd != nil {
		go func() { cmdExit <- cmd.Wait() }()
	}

	// Loop listening file modification
	for {
		select {
		case <-quit:
			if cmd != nil && cmd.Process != nil {
				if err := killProcessGroup(cmd); err != nil {
					output.Error("server exit error: %s", err)
				}
			}
			output.Success("server exiting...")
			return

		case event := <-watcher.Events:
			// 文件被修改或删除时重启
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Remove == fsnotify.Remove {
				if !shouldRestart(event.Name) {
					continue
				}
				output.Success("file modified: %s", event.Name)
				if cmd != nil && cmd.Process != nil {
					_ = killProcessGroup(cmd)
					waitProcessExit(cmd)
					// 消费旧的退出信号,避免重启后误触发
					select {
					case <-cmdExit:
					default:
					}
				}
				cmd = start(dir, programArgs)
				if cmd != nil {
					cmdExit = make(chan error, 1)
					go func() { cmdExit <- cmd.Wait() }()
				}
			}
			// 新建目录时加入 watcher，使其内部文件也被监听
			if event.Op&fsnotify.Create == fsnotify.Create {
				if helper.IsExcluded(event.Name, excludeDirArr) {
					continue
				}
				if fi, fiErr := os.Stat(event.Name); fiErr == nil && fi.IsDir() {
					_ = watcher.Add(event.Name)
				}
			}
		case err := <-watcher.Errors:
			output.Error("Error: %s", err)
		case err := <-cmdExit:
			// 子进程退出(编译失败/运行 panic),报告并停止
			if err != nil {
				output.Error("process exited: %s", err)
			} else {
				output.Error("process exited")
			}
			return
		}
	}
}

// shouldRestart 判断指定文件是否应触发重启：仅当扩展名在 includeExt 白名单内时。
func shouldRestart(name string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	if ext == "" {
		return false
	}
	for _, e := range strings.Split(includeExt, ",") {
		if strings.TrimSpace(e) == ext {
			return true
		}
	}
	return false
}

// waitProcessExit 等待进程组退出，避免端口/资源尚未释放就重启导致冲突。
func waitProcessExit(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

func isProcessRunning(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return isProcessAlive(cmd)
}

// buildProbeDelay 是判断 `go run` 是否"立即编译失败"的等待时长。
// 这是无奈的启发式:`go run` 先编译再运行,无法可靠区分"还在编译"与"已成功启动"。
// - 取太小:大项目编译耗时超过该值会被误判为成功(实际仍在编译,真正的失败要等 cmdExit)。
// - 取太大:小项目失败时拖慢重启节奏。
// 300ms 是经验值;真正可靠的方案应是先 `go build` 到临时二进制再执行,此处为简化实现而妥协。
const buildProbeDelay = 300 * time.Millisecond

func start(dir string, programArgs []string) *exec.Cmd {
	cmd := exec.Command("go", append([]string{"run", dir}, programArgs...)...)
	applyProcAttr(cmd)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		output.Error("cmd run failed: %s", err)
		return nil
	}
	// go run 需先编译再运行,短暂等待判断是否编译失败立即退出。
	// 注意: 这里不调用 cmd.Wait(),Wait 由 watch 统一管理(同一 cmd 只能 Wait 一次)。
	output.Success("building & running...") // 提示用户正在编译启动,避免误以为卡住
	time.Sleep(buildProbeDelay)
	if !isProcessRunning(cmd) {
		// 进程已退出,Wait 一次拿回编译错误(此时 Wait 不会阻塞,进程已死)
		if waitErr := cmd.Wait(); waitErr != nil {
			output.Error("process exited immediately: %s", waitErr)
		} else {
			output.Error("process exited immediately")
		}
		return nil
	}
	output.Success("running...")
	return cmd
}
