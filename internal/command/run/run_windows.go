//go:build windows
// +build windows

package run

import (
    "os"
    "os/exec"
    "os/signal"
    "path/filepath"
    "sort"
    "strconv"
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
    includeExtArr := strings.Split(includeExt, ",")
    includeExtMap := make(map[string]struct{})
    for _, s := range includeExtArr {
        includeExtMap[s] = struct{}{}
    }
    // Add files to watcher
    err = filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        for _, s := range excludeDirArr {
            if s == "" {
                continue
            }
            if strings.HasPrefix(path, s) {
                return nil
            }
        }
        if !info.IsDir() {
            ext := filepath.Ext(info.Name())
            if _, ok := includeExtMap[strings.TrimPrefix(ext, ".")]; ok {
                err = watcher.Add(path)
                if err != nil {
                    fmt.Error("Error: %s", err)
                }
            }
            
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
            err = killProcess(cmd)
            if err != nil {
                fmt.Error("server exit error: %s", err)
                return
            }
            fmt.Success("server exiting...")
            os.Exit(0)
        
        case event := <-watcher.Events:
            // The file has been modified or created
            if event.Op&fsnotify.Create == fsnotify.Create ||
                event.Op&fsnotify.Write == fsnotify.Write ||
                event.Op&fsnotify.Remove == fsnotify.Remove {
                fmt.Success("file modified: %s", event.Name)
                _ = killProcess(cmd)
                
                cmd = start(dir, programArgs)
            }
        case err := <-watcher.Errors:
            fmt.Error("Error: %s", err)
        }
    }
}

func killProcess(cmd *exec.Cmd) error {
    if cmd.Process == nil {
        return nil
    }
    // 获取进程ID
    pid := cmd.Process.Pid
    // 构造taskkill命令
    taskkill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
    err := taskkill.Run()
    if err != nil {
        return err
    }
    return nil
}

func start(dir string, programArgs []string) *exec.Cmd {
    cmd := exec.Command("go", append([]string{"run", dir}, programArgs...)...)
    // Set a new process group to kill all child processes when the program exits
    
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    err := cmd.Start()
    if err != nil {
        fmt.Error("cmd run failed")
    }
    time.Sleep(time.Second)
    fmt.Success("running...")
    return cmd
}
