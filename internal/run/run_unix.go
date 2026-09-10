//go:build !plan9 && !windows
// +build !plan9,!windows

package run

import (
	"os"
	"os/exec"
	"syscall"
)

// signals 返回本平台需要监听的退出信号。
func signals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}

// killProcessGroup 杀死整个进程组(unix 用负 pid + SIGKILL)。
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

// isProcessAlive 通过 signal 0 探测进程是否存活。
func isProcessAlive(cmd *exec.Cmd) bool {
	return cmd.Process.Signal(syscall.Signal(0)) == nil
}

// applyProcAttr 设置进程组属性,unix 下用 Setpgid 以便按进程组 kill。
func applyProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
