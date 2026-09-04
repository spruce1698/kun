/**
 * @Author: spruce
 * @Date: 2024-03-28 14:59
 * @Desc: pubsub broker 服务
 */

package broker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"advanced/pkg/utils"
	"advanced/pkg/xconfig"
	"advanced/pkg/xdb"
	"advanced/pkg/xlog"
	"advanced/pkg/xredis"
	"advanced/pkg/xserver"

	"github.com/gin-gonic/gin"
)

const (
	ChildKey = "BrokerChildKey"
	ChildVal = "BrokerChildProcess"
)

type Server struct {
	Conf   *xconfig.Conf
	logger *xlog.Logger

	db    *xdb.Client
	redis *xredis.Client

	// childRun 子进程入口:由装配层(internal/app)注入,内部负责初始化 tracer、
	// 启动 kafka/asynq 消费者并阻塞等待退出。broker 引擎只负责进程生命周期,不依赖 internal。
	childRun func()

	// healthEngine 父进程健康探针 gin 引擎,由装配层(internal/app)构建,broker 只负责启动。
	healthEngine *gin.Engine

	// closer 聚合 db/redis/xlog 的关闭,与 http.Server 复用同一套 LIFO 回收逻辑。
	closer *xserver.Closer

	// 父进程运行态(仅父进程使用)
	healthSvr *http.Server

	mu      sync.Mutex // 保护 curCmd
	curCmd  *exec.Cmd
	stopCh  chan struct{} // Stop() 关闭,通知 supervisor goroutine 退出
	cmdExit chan error    // 子进程退出事件(err 由 cmd.Wait 返回)
	done    chan struct{} // supervisor 在子进程退出(并完成 Wait)后关闭,Stop 据此等待,不依赖 cmdExit 投递时序
}

func New(conf *xconfig.Conf, log *xlog.Logger, db *xdb.Client, redis *xredis.Client, healthEngine *gin.Engine, childRun func()) (xserver.Engine, error) {
	// 资源关闭器(LIFO):xlog 先注册(最后关),redis 次之,db 最后注册(最先关)。
	// 期望关闭顺序:db -> redis -> xlog。
	closer := xserver.NewCloser()
	closer.Add(func() error { return xlog.Close() }, "xlog")
	if redis != nil {
		closer.Add(func() error { return redis.Close() }, "redis")
	}
	if db != nil {
		closer.Add(func() error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		}, "db")
	}

	return &Server{
		Conf:   conf,
		logger: log,

		db:    db,
		redis: redis,

		childRun: childRun,

		healthEngine: healthEngine,

		closer: closer,
	}, nil
}

// Start 多进程实现
func (s *Server) Start() error {

	// 是否是子进程
	if IsChild() {
		s.childRun()
		os.Exit(0)
		return nil
	}

	s.healthSvr = s.startHealthServer()
	s.stopCh = make(chan struct{})
	s.cmdExit = make(chan error, 1)
	s.done = make(chan struct{})

	// supervisor goroutine:启动并监控子进程。
	// 子进程异常退出则父进程也退出;收到 stopCh 关闭则正常退出。
	go s.supervisor()

	return nil
}

// startHealthServer 用装配层构建的 healthEngine 创建并启动父进程健康探针。
func (s *Server) startHealthServer() *http.Server {
	engine := s.healthEngine
	if engine == nil {
		return nil
	}

	port := s.Conf.Broker.Port
	if port == 0 {
		port = utils.AvailablePort()
	}
	readTimeout, writeTimeout := 120*time.Second, 120*time.Second
	// 默认 2min
	if s.Conf.Broker.ReadTimeout != 0 {
		readTimeout = time.Duration(s.Conf.Broker.ReadTimeout) * time.Second
	}
	if s.Conf.Broker.WriteTimeout != 0 {
		writeTimeout = time.Duration(s.Conf.Broker.WriteTimeout) * time.Second
	}

	// 初始化HTTP服务
	httpSvr := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           engine,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: 10 * time.Second, // 防御 Slowloris
		WriteTimeout:      writeTimeout,
	}

	// 启动成功
	s.logger.Warn(fmt.Sprintf("Broker Server 启动 (Host: 0.0.0.0:%d Pid:%d)", port, os.Getpid()))

	go func() {
		if err := httpSvr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Warn(fmt.Sprintf("listen: %s", err))
		}
	}()
	return httpSvr
}

// supervisor 启动子进程并等待其退出。
// 子进程退出事件通过 s.cmdExit 传递给 Stop();Stop() 关闭 s.stopCh 通知本协程结束。
func (s *Server) supervisor() {
	// 无论子进程是否启动成功、是否异常退出,都在本协程结束时关闭 done,
	// 让 Stop() 能可靠地等待到"子进程已退出"这一事件,而不依赖 cmdExit 的投递时序。
	defer close(s.done)

	childEnv := append(os.Environ(), fmt.Sprintf("%s=%s", ChildKey, ChildVal))

	for {
		cmd := exec.Cmd{
			Path:   os.Args[0],
			Args:   os.Args,
			Env:    childEnv,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		if err := cmd.Start(); err != nil {
			s.logger.Warn("Broker start child process err: " + err.Error())
			return
		}

		s.mu.Lock()
		s.curCmd = &cmd
		s.mu.Unlock()

		waitErr := cmd.Wait()

		// 清空当前 cmd 引用
		s.mu.Lock()
		s.curCmd = nil
		s.mu.Unlock()

		// 通知 Stop()(若正在等待);cmdExit 容量为 1,投递失败说明已有值,丢弃即可。
		select {
		case s.cmdExit <- waitErr:
		default:
		}

		// 是否已收到停止信号
		select {
		case <-s.stopCh:
			s.logger.Warn("Broker parent process exit")
			return
		default:
		}

		// 未主动停止却退出 -> 子进程异常,父进程也退出
		if waitErr != nil {
			s.logger.Warn(fmt.Sprintf("Broker child process %d exit err: %s", cmd.Process.Pid, waitErr.Error()))
		} else {
			s.logger.Warn(fmt.Sprintf("Broker child process %d exit", cmd.Process.Pid))
		}
		return
	}
}

// Stop 由 AwaitSignal 在收到信号后调用,是唯一的关闭入口。
// 1. 先关 health server,让探针尽快变红摘流量;
// 2. 转发 SIGTERM 给子进程并等待其退出(子进程收尾时仍可能用到 db/redis);
// 3. 关闭 db/redis;
// 4. 通知 supervisor 退出。
func (s *Server) Stop(signal string) {
	s.logger.Warn("Receive a signal", xlog.KVStr("signal", signal))
	s.logger.Warn("Broker server stopping ...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// 1. 先关 health server,探针变红,上游摘流量
	if s.healthSvr != nil {
		if err := s.healthSvr.Shutdown(ctx); err != nil {
			s.logger.Warn(fmt.Sprintf("http server shutdown err:%v", err))
		}
		s.logger.Warn("Close http server")
	}

	// 2. 通知子进程退出并等待
	s.mu.Lock()
	cmd := s.curCmd
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = terminateProcess(cmd.Process)

		// 等待 supervisor 中 cmd.Wait() 返回并关闭 done;
		// 不依赖 cmdExit 的投递时序(它可能因 cap=1 满了而投递失败)。
		if s.done != nil {
			select {
			case <-s.done:
				s.logger.Warn("Broker child process exited")
			case <-time.After(time.Second * 5):
				s.logger.Warn("Broker child process shutdown timeout, killing")
				_ = cmd.Process.Kill()
				<-s.done
			}
		}
	} else if s.done != nil {
		// 无子进程(启动失败或已退出),仍等待 supervisor 收尾,避免 close(stopCh) 竞态。
		<-s.done
	}

	s.logger.Warn("Broker server stopped")

	// 3. 关闭 db/redis/xlog:统一交给 Closer 按 LIFO 顺序关闭,与 http.Server 复用同一套回收逻辑。
	// 先 sync logger 刷出缓冲日志,再由 Closer 关闭底层 Kafka Writer(在 xlog.Close 内)。
	if err := s.logger.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
	}
	for _, err := range s.closer.Close() {
		fmt.Fprintf(os.Stderr, "close resource err:%v\n", err)
	}

	// 4. 通知 supervisor 退出(若子进程是异常退出已 return,这里无影响)
	if s.stopCh != nil {
		close(s.stopCh)
	}
}

// IsChild 判断当前是否为 broker 子进程
func IsChild() bool {
	return os.Getenv(ChildKey) == ChildVal
}

func terminateProcess(p *os.Process) error {
	if p == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		return p.Kill()
	}
	return p.Signal(syscall.SIGTERM)
}

