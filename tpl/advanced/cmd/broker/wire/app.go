/**
 * @Author: spruce
 * @Date: 2024-08-15
 * @Desc: Broker 应用装配层
 *
 * 作为 Composition Root，在 cmd/broker/wire 中组装 broker 父进程的健康探针 server(gin 模式/中间件/路由)
 * 以及构建子进程消费执行入口(kafka/asynq 消费者与生命周期信号管理)。
 */

package wire

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"advanced/internal/event"
	"advanced/internal/middleware"
	"advanced/internal/router"
	"advanced/pkg/xconfig"
	"advanced/pkg/xlog"
	"advanced/pkg/xserver"
	"advanced/pkg/xserver/broker"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

// NewBrokerHealth 装配 broker 父进程的健康探针 gin 引擎。
// 由 broker.Server 负责创建 http.Server 并启动。子进程无需承载 HTTP 探针，直接跳过构建以节约资源并避免重复打印日志。
func NewBrokerHealth(conf *xconfig.Conf, log *xlog.Logger) *gin.Engine {
	if broker.IsChild() {
		return nil
	}

	// 统一设置 gin 全局模式与输出,与 http server 保持一致,避免多实例重复覆盖全局状态。
	xserver.InitGinMode(conf.Env)
	engine := gin.New()

	// 性能分析 - 正式环境不要使用！！！
	if conf.Env == xconfig.EnvDebug {
		pprof.Register(engine)
	}

	// 接管gin框架默认的日志和捕获异常。
	// Recovery 必须最先注册(位于中间件链最外层),否则 Tracing 自身 panic 时无人兜住。
	engine.Use(
		middleware.Recovery(conf.Env == xconfig.EnvDebug || conf.Env == xconfig.EnvTest), // 测试或开发环境
		xlog.TracingWithLogger(log, conf.Broker.Name),
	)

	// 探针路由
	router.Ping(engine)

	return engine
}

// NewBrokerChildRun 构建 broker 子进程入口(由 broker.Server 在子进程内调用)。
// 职责:初始化 tracer、创建订阅器、启动 kafka/asynq 消费者并阻塞等待退出信号。
func NewBrokerChildRun(conf *xconfig.Conf, task *event.Task) func() {
	return func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 子进程独立初始化 TracerProvider,使 kafka/asynq 消费者产生的 span 能上报。
		// 父进程通过 fork exec 启动子进程,全局 TracerProvider 不会继承,必须重新初始化。
		tp := xlog.InitTracer(xlog.TracingConfig{
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
}
