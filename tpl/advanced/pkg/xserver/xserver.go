/**
 * @Author: spruce
 * @Date: 2024-03-28 15:34
 * @Desc: x服务接口
 */

package xserver

import (
	"os"
	"os/signal"
	"syscall"
)

type Engine interface {
	Start() error
	Stop(signal string)
}

type Server struct {
	eng Engine
}

func New(eng Engine) *Server {
	return &Server{
		eng: eng,
	}
}

// Start server
func (s *Server) Start() error {
	if err := s.eng.Start(); err != nil {
		return err
	}
	return nil
}

// AwaitSignal for exit server
// 收到退出信号后同步调用 eng.Stop 完成优雅关闭(db/redis/kafka 等),
// 然后 os.Exit(0) 退出。注意 os.Exit 会跳过 main 中的 defer,因此所有资源
// 回收必须在 eng.Stop 内完成(当前 http/broker 的 Stop 均已包含完整清理)。
func (s *Server) AwaitSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Reset(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	s.eng.Stop(sig.String())
	os.Exit(0)
}
