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

	mu      sync.Mutex // 保护 curCmd
	curCmd  *exec.Cmd
	stopCh  chan struct{} // Stop() 关闭,通知 supervisor goroutine 退出
	cmdExit chan error    // 子进程退出事件(err 由 cmd.Wait 返回)
	done    chan struct{} // supervisor 在子进程退出(并完成 Wait)后关闭,Stop 据此等待,不依赖 cmdExit 投递时序
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
		childProcess(s.Conf, s.task)
		return nil
	}

	s.healthSvr = healthServer(s.Conf, s.logger)
	s.stopCh = make(chan struct{})
	s.cmdExit = make(chan error, 1)
	s.done = make(chan struct{})

	// supervisor goroutine:启动并监控子进程。
	// 子进程异常退出则父进程也退出;收到 stopCh 关闭则正常退出。
	go s.supervisor()

	return nil
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
		_ = cmd.Process.Signal(syscall.SIGTERM)

		// 等待 supervisor 中 cmd.Wait() 返回并关闭 done;
		// 不依赖 cmdExit 的投递时序(它可能因 cap=1 满了而投递失败)。
		select {
		case <-s.done:
			s.logger.Warn("Broker child process exited")
		case <-time.After(time.Second * 5):
			s.logger.Warn("Broker child process shutdown timeout, killing")
			_ = cmd.Process.Kill()
			<-s.done
		}
	} else {
		// 无子进程(启动失败或已退出),仍等待 supervisor 收尾,避免 close(stopCh) 竞态。
		<-s.done
	}

	// 3. 关闭 db
	if s.db != nil {
		sqlDB, _ := s.db.DB()
		if sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				s.logger.Warn(fmt.Sprintf("db Close err:%v", err))
			}
			s.logger.Warn("Close Db")
		}
	}
	// 关闭 redis
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			s.logger.Warn(fmt.Sprintf("redis Close err:%v", err))
		}
		s.logger.Warn("Close redis")
	}

	// TODO 其他关闭

	s.logger.Warn("Broker server stopped")

	// 4. 通知 supervisor 退出(若子进程是异常退出已 return,这里无影响)
	close(s.stopCh)

	// 日志异步(此时 xlog 可能已关闭,错误信息走 stderr)
	if err := s.logger.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
	}
	// 关闭日志组件底层资源(如日志 Kafka Writer)
	if err := xlog.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to close xlog: %v\n", err)
	}
}

func isChild() bool {
	return os.Getenv(ChildKey) == ChildVal
}

func healthServer(conf *xconfig.Conf, log *xlog.Logger) *http.Server {
	// 在创建引擎前设置 gin 全局模式,与 http server 保持一致,避免多实例重复覆盖全局状态。
	if conf.Env == xconfig.EnvStaging || conf.Env == xconfig.EnvRelease {
		gin.SetMode(gin.ReleaseMode)
		// 禁止gin的默认输出
		gin.DefaultWriter = io.Discard
	} else {
		gin.SetMode(conf.Env)
	}
	engine := gin.New()

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
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           engine,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: 10 * time.Second, // 防御 Slowloris
		WriteTimeout:      writeTimeout,
	}

	// 启动成功
	log.Warn(fmt.Sprintf("Broker Server 启动 (Host: 0.0.0.0:%d  Pid:%d)", port, os.Getpid()))

	go func() {
		if err := httpSvr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn(fmt.Sprintf("listen: %s", err))
		}
	}()
	return httpSvr
}

func childProcess(conf *xconfig.Conf, task *event.Task) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 子进程独立初始化 TracerProvider,使 kafka/asynq 消费者产生的 span 能上报。
	// 父进程通过 fork exec 启动子进程,全局 TracerProvider 不会继承,必须重新初始化。
	tp := middleware.InitTracer(middleware.TracingConfig{
		ServiceName:    conf.Broker.Name,
		ServiceVersion: conf.Version,
		Endpoint:       conf.Jaeger.Endpoint,
		SampleRate:     conf.Jaeger.SampleRate,
	})
	if tp != nil {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			_ = tp.Shutdown(shutdownCtx)
		}()
	}

	// 子进程统一监听退出信号:取消 ctx,驱动各消费者(kafka/asynq)优雅退出。
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-signals
		cancel()
	}()

	var wg sync.WaitGroup
	sub := event.NewSub(&wg, conf, task, ctx)

	// kafka subscriber
	sub.Kafka()
	// asynq subscriber
	if err := sub.Asynq(); err != nil {
		// 启动失败:cancel ctx 触发已启动的消费者退出,等待收尾后退出进程。
		// 子进程此时 logger 尚未初始化,错误信息走 stderr。
		fmt.Fprintf(os.Stderr, "!!!BrokerErr!!! asynq subscriber start failed: %v\n", err)
		cancel()
		wg.Wait()
		sub.Close()
		os.Exit(1)
	}
	// asynq CronPush(cron 启动失败不致命,记录后继续运行主消费)
	if err := sub.AsynqCron(); err != nil {
		fmt.Fprintf(os.Stderr, "!!!BrokerErr!!! asynq cron start failed: %v\n", err)
	}

	wg.Wait()
	sub.Close()
}
