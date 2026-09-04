/**
 * @Author: spruce
 * @Date: 2024-03-28 14:59
 * @Desc: http Engine
 *
 * 仅负责 HTTP 生命周期(listen / graceful shutdown / 资源回收),
 * 不再组装中间件与路由--组装由 internal/app 完成,依赖方向 internal -> pkg。
 */

package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"basic/pkg/utils"
	"basic/pkg/xconfig"
	"basic/pkg/xlog"
	"basic/pkg/xserver"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Conf   *xconfig.Conf
	logger *xlog.Logger

	engine *gin.Engine
	closer *xserver.Closer

	server *http.Server

	// stopOnce 守卫 Stop:监听失败与 AwaitSignal 信号都可能触发 Stop,
	// 用 Once 保证整套关闭(db/redis/tracer/xlog)只执行一次,避免二次关闭。
	stopOnce sync.Once
}

// New 创建 http 服务。engine 为已装配好中间件/路由的 gin 引擎,
// closer 聚合进程退出时需回收的资源(由 internal/app 注册)。
func New(
	conf *xconfig.Conf,
	log *xlog.Logger,
	engine *gin.Engine,
	closer *xserver.Closer,
) (xserver.Engine, error) {
	if closer == nil {
		closer = xserver.NewCloser()
	}
	return &Server{
		Conf:   conf,
		logger: log,
		engine: engine,
		closer: closer,
	}, nil
}

func (s *Server) Start() error {
	readTimeout, writeTimeout := 120*time.Second, 120*time.Second
	// 默认 2min
	if s.Conf.Server.ReadTimeout != 0 {
		readTimeout = time.Duration(s.Conf.Server.ReadTimeout) * time.Second
	}
	if s.Conf.Server.WriteTimeout != 0 {
		writeTimeout = time.Duration(s.Conf.Server.WriteTimeout) * time.Second
	}
	// ReadHeaderTimeout 防御 Slowloris 慢速攻击:仅限制读请求头阶段,
	// 不影响 streaming/大上传的请求体读取。固定 10s。
	const readHeaderTimeout = 10 * time.Second
	port := s.Conf.Server.Port
	if port == 0 {
		port = utils.AvailablePort()
	}
	// 初始化HTTP/JOB服务
	svr := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           s.engine,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
	}

	// 启动成功
	s.logger.Warn(fmt.Sprintf("Http server 启动 (Host: 0.0.0.0:%d Pid:%d)", port, os.Getpid()))

	go func() {
		if err := svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error(fmt.Sprintf("%s ", err))
			// 监听失败:走统一的 Stop(仅会执行一次),回收资源后退出进程。
			s.Stop("")
			os.Exit(1)
		}
	}()

	s.server = svr
	return nil
}

func (s *Server) Stop(signal string) {
	s.stopOnce.Do(func() {
		s.stop(signal)
	})
}

func (s *Server) stop(signal string) {
	s.logger.Warn("Receive a signal", xlog.KVStr("signal", signal))
	s.logger.Warn("Http server stopping ...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()

	// 优雅关闭
	// 关闭 http server
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Warn(fmt.Sprintf("http server shutdown err:%v", err))
		}
		s.logger.Warn("Close http server")
	}

	// 关闭其余资源(db/redis/tracer/xlog):统一交给 Closer 按 LIFO 顺序关闭。
	if s.closer != nil {
		for _, err := range s.closer.Close() {
			s.logger.Warn(fmt.Sprintf("close resource err:%v", err))
		}
	}

	s.logger.Warn("Http server stopped")

	// 日志异步(此时 xlog 可能已关闭,错误信息走 stderr)
	if err := s.logger.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
	}
}
