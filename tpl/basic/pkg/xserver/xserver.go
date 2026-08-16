/**
 * @Author: spruce
 * @Date: 2024-03-28 15:34
 * @Desc: x服务接口
 */

package xserver

import (
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
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
// 收到退出信号后同步调用 eng.Stop 完成优雅关闭(db/redis 等),
// 然后 os.Exit(0) 退出。注意 os.Exit 会跳过 main 中的 defer,因此所有资源
// 回收必须在 eng.Stop 内完成(当前 http 的 Stop 均已包含完整清理)。
func (s *Server) AwaitSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Reset(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	s.eng.Stop(sig.String())
	os.Exit(0)
}

// InitGinMode 统一设置 gin 全局模式与输出。
// 多 server 实例(http + broker health)共用同一进程的 gin 全局态,
// 集中到一处设置,避免各处重复覆盖 DefaultWriter。
// 使用反向代理/多实例时仍需各自正确配置 SetTrustedProxies。
func InitGinMode(env string) {
	switch env {
	case "staging", "release":
		gin.SetMode(gin.ReleaseMode)
		// 禁止gin的默认输出
		gin.DefaultWriter = io.Discard
	default:
		gin.SetMode(env)
	}
}

// Closer 聚合一组按 LIFO 顺序执行的关闭函数,统一管理进程退出时的资源回收
// (db/redis/tracer/xlog 等),避免各 server 各抄一份关闭序列、新增资源要改多处。
// 并发安全:Close 仅执行一次,重复调用直接返回。
//
// 用法:
//
//	c := xserver.NewCloser()
//	c.Add(func() error { return sqlDB.Close() }, "db")
//	c.Add(func() error { return redis.Close() }, "redis")
//	// 退出时:errs := c.Close()  // 按 LIFO: redis -> db
type Closer struct {
	mu      sync.Mutex
	closers []closeEntry
	closed  bool
}

type closeEntry struct {
	name string
	fn   func() error
}

// NewCloser 创建一个空的资源关闭聚合器。
func NewCloser() *Closer {
	return &Closer{}
}

// Add 注册一个关闭函数,后注册的先关闭(LIFO)。
// name 仅用于日志标识。返回 Closer 自身以便链式调用。
func (c *Closer) Add(fn func() error, name string) *Closer {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closers = append(c.closers, closeEntry{name: name, fn: fn})
	return c
}

// Close 按 LIFO 顺序执行所有关闭函数。任意一步出错只记录到 errs,不中断后续关闭。
// 重复调用安全:仅首次执行,后续直接返回 nil。
func (c *Closer) Close() []error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	// 复制一份倒序,锁内不执行 IO
	entries := make([]closeEntry, len(c.closers))
	for i, e := range c.closers {
		entries[len(c.closers)-1-i] = e
	}
	c.mu.Unlock()

	var errs []error
	for _, e := range entries {
		if err := e.fn(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
