/**
 * @Author: spruce
 * @Date: 2024-03-28 14:59
 * @Desc: pubsub broker 服务
 */

package broker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"advanced/internal/event"
	"advanced/internal/middleware"
	"advanced/internal/router"
	"advanced/pkg/utils"
	"advanced/pkg/xconfig"
	"advanced/pkg/xdb"
	"advanced/pkg/xlog"
	"advanced/pkg/xredis"
	"advanced/pkg/xserver"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
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

	task *event.Task

	// 父进程运行态(仅父进程使用)
	healthSvr *http.Server

	mu       sync.Mutex // 保护 curCmd
	curCmd   *exec.Cmd
	stopCh   chan struct{} // Stop() 关闭,通知 supervisor goroutine 退出
	cmdExit  chan error    // 子进程退出事件(err 由 cmd.Wait 返回)
	stopOnce sync.Once     // 保证 stopCh 只关闭一次,防止二次调用 panic
}

func New(conf *xconfig.Conf, log *xlog.Logger, db *xdb.Client, redis *xredis.Client, task *event.Task) (xserver.Engine, error) {
	return &Server{
		Conf:   conf,
		logger: log,

		db:    db,
		redis: redis,

		task: task,
	}, nil
}

// Start 多进程实现
func (s *Server) Start() error {

	// 是否是子进程
	if isChild() {
		childProcess(s.Conf, s.task, s.db, s.redis)
		return nil
	}

	s.healthSvr = healthServer(s.Conf, s.logger)
	s.stopCh = make(chan struct{})
	s.cmdExit = make(chan error, 1)

	// supervisor goroutine:启动并监控子进程。
	// 子进程异常退出则父进程也退出;收到 stopCh 关闭则正常退出。
	go s.supervisor()

	return nil
}

// supervisor 启动子进程并等待其退出。
// 子进程退出事件通过 s.cmdExit 传递给 Stop();Stop() 关闭 s.stopCh 通知本协程结束。
func (s *Server) supervisor() {
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
			xlog.Warn(context.Background(), "Broker start child process err: "+err.Error())
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

		// 通知 Stop()(若正在等待)
		select {
		case s.cmdExit <- waitErr:
		default:
		}

		// 是否已收到停止信号
		select {
		case <-s.stopCh:
			xlog.Warn(context.Background(), "Broker parent process exit")
			return
		default:
		}

		// 未主动停止却退出 -> 子进程异常,父进程也退出
		if waitErr != nil {
			xlog.Warnf(context.Background(), "Broker child process %d exit err: %s", cmd.Process.Pid, waitErr.Error())
		} else {
			xlog.Warnf(context.Background(), "Broker child process %d exit", cmd.Process.Pid)
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
	xlog.Warn(context.Background(), "Receive a signal", xlog.KVStr("signal", signal))
	xlog.Warn(context.Background(), "Broker server stopping ...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// 1. 先关 health server,探针变红,上游摘流量
	if s.healthSvr != nil {
		if err := s.healthSvr.Shutdown(ctx); err != nil {
			xlog.Warnf(context.Background(), "http server shutdown err:%v", err)
		}
		xlog.Warn(context.Background(), "Close http server")
	}

	// 2. 通知子进程退出并等待
	s.mu.Lock()
	cmd := s.curCmd
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)

		select {
		case <-s.cmdExit:
			xlog.Warn(context.Background(), "Broker child process exited")
		case <-time.After(time.Second * 5):
			xlog.Warn(context.Background(), "Broker child process shutdown timeout, killing")
			_ = cmd.Process.Kill()
			<-s.cmdExit
		}
	}

	// 3. 关闭 db
	if s.db != nil {
		sqlDB, _ := s.db.DB()
		if sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				xlog.Warnf(context.Background(), "db Close err:%v", err)
			}
			xlog.Warn(context.Background(), "Close Db")
		}
	}
	// 关闭 redis
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			xlog.Warnf(context.Background(), "redis Close err:%v", err)
		}
		xlog.Warn(context.Background(), "Close redis")
	}

	// TODO 其他关闭

	xlog.Warn(context.Background(), "Broker server stopped")

	// 4. 通知 supervisor 退出(若子进程是异常退出已 return,这里无影响)
	s.stopOnce.Do(func() { close(s.stopCh) })

	// 刷新日志缓冲并关闭底层资源(如日志 Kafka Writer),避免异步日志丢日志
	if err := xlog.Close(); err != nil {
		fmt.Printf("Failed to close xlog: %v\n", err)
	}
}

func isChild() bool {
	return os.Getenv(ChildKey) == ChildVal
}

func healthServer(conf *xconfig.Conf, log *xlog.Logger) *http.Server {
	engine := gin.New()

	if conf.Env == xconfig.EnvStaging || conf.Env == xconfig.EnvRelease {
		gin.SetMode(gin.ReleaseMode)
		// 禁止gin的默认输出
		gin.DefaultWriter = io.Discard
	} else {
		gin.SetMode(conf.Env)
	}

	// 性能分析 - 正式环境不要使用！！！
	if conf.Env == xconfig.EnvDebug {
		pprof.Register(engine)
	}

	// 接管gin框架默认的日志和捕获异常
	engine.Use(
		middleware.TracingWithLogger(log, conf.Broker.Name),
		middleware.Recovery(conf.Env == xconfig.EnvDebug || conf.Env == xconfig.EnvTest), // 测试或开发环境
	)

	// 探针路由
	router.Ping(engine)

	port := conf.Broker.Port
	if port == 0 {
		port = utils.AvailablePort()
	}
	readTimeout, writeTimeout := 120*time.Second, 120*time.Second
	// 默认 2min
	if conf.Broker.ReadTimeout != 0 {
		readTimeout = time.Duration(conf.Broker.ReadTimeout) * time.Second
	}
	if conf.Broker.WriteTimeout != 0 {
		writeTimeout = time.Duration(conf.Broker.WriteTimeout) * time.Second
	}

	// 初始化HTTP服务
	httpSvr := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	// 启动成功
	xlog.Warnf(context.Background(), "Broker Server 启动 (Host: 0.0.0.0:%d  Pid:%d)", port, os.Getpid())

	go func() {
		if err := httpSvr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			xlog.Warnf(context.Background(), "listen: %s", err)
		}
	}()
	return httpSvr
}

func childProcess(conf *xconfig.Conf, task *event.Task, db *xdb.Client, redis *xredis.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 子进程统一监听退出信号:取消 ctx,驱动各消费者(kafka/asynq)优雅退出。
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-signals
		cancel()
	}()

	// closeResources 关闭子进程持有的 db/redis 连接,通知服务端主动断开,避免 Aborted connection。
	closeResources := func() {
		if db != nil {
			if sqlDB, _ := db.DB(); sqlDB != nil {
				_ = sqlDB.Close()
			}
		}
		if redis != nil {
			_ = redis.Close()
		}
		// 刷新日志缓冲并关闭底层资源(已含 Sync)
		_ = xlog.Close()
	}

	var wg sync.WaitGroup
	sub := event.NewSub(&wg, conf, task, ctx)

	// kafka subscriber
	sub.Kafka()
	// asynq subscriber
	if err := sub.Asynq(); err != nil {
		// 启动失败:cancel ctx 触发已启动的消费者退出,等待收尾后退出进程
		fmt.Printf("!!!BrokerErr!!! asynq subscriber start failed: %v\n", err)
		cancel()
		wg.Wait()
		sub.Close()
		closeResources()
		os.Exit(1)
	}
	// asynq CronPush(cron 启动失败不致命,记录后继续运行主消费)
	if err := sub.AsynqCron(); err != nil {
		fmt.Printf("!!!BrokerErr!!! asynq cron start failed: %v\n", err)
	}

	wg.Wait()
	sub.Close()
	closeResources()
}
