//go:build windows
// +build windows

package run

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
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

// isProcessAlive 探测进程是否存活。
// 注意:windows 不支持 Signal(0)(对存活进程也返回 "not supported by windows"),
// 因此必须用 GetExitCodeProcess 判断:仍在运行时返回 STILL_ACTIVE(259)。
func isProcessAlive(cmd *exec.Cmd) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cmd.Process.Pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	// STILL_ACTIVE(259):进程尚未退出。
	const stillActive = 259
	return code == stillActive
}

// applyProcAttr windows 下无需额外进程组属性(taskkill /T 负责杀子进程树)。
func applyProcAttr(cmd *exec.Cmd) {}
