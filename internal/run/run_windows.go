//go:build windows
// +build windows

package run

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// signals 返回本平台需要监听的退出信号。
func signals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}

// killProcessGroup 杀死整个进程树(windows 用 taskkill /T)。
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	taskkill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	return taskkill.Run()
}

// isProcessAlive 通过 signal 0 探测进程是否存活。
func isProcessAlive(cmd *exec.Cmd) bool {
	return cmd.Process.Signal(syscall.Signal(0)) == nil
}

// applyProcAttr windows 下无需额外进程组属性(taskkill /T 负责杀子进程树)。
func applyProcAttr(cmd *exec.Cmd) {}
