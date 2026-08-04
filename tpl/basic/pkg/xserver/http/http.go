/**
 * @Author: spruce
 * @Date: 2024-03-28 14:59
 * @Desc: http Engine
 */

package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"basic/internal/controller"
	"basic/internal/middleware"
	"basic/internal/router"
	"basic/pkg/token"
	"basic/pkg/utils"
	"basic/pkg/validator"
	"basic/pkg/xconfig"
	"basic/pkg/xdb"
	"basic/pkg/xlog"
	"basic/pkg/xredis"
	"basic/pkg/xserver"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Server struct {
	Conf   *xconfig.Conf
	logger *xlog.Logger
	tp     *sdktrace.TracerProvider
	db     *xdb.Client
	Redis  *xredis.Client

	Engine *gin.Engine
	server *http.Server
}

func New(
	conf *xconfig.Conf,
	log *xlog.Logger,
	db *xdb.Client,
	redis *xredis.Client,
	ctl *controller.ServerCtrlCtx,
	rtrs []router.Router,
) (xserver.Engine, error) {
	eng := gin.New()

	if conf.Env == xconfig.EnvStaging || conf.Env == xconfig.EnvRelease {
		gin.SetMode(gin.ReleaseMode)
		// 禁止gin的默认输出
		gin.DefaultWriter = io.Discard
	} else {
		gin.SetMode(conf.Env)
	}

	// 性能分析 - 正式环境不要使用！！！
	if conf.Env == xconfig.EnvDebug {
		pprof.Register(eng)
	}
	// 初始化 全链路跟踪 tracer
	tp := middleware.InitTracer(middleware.TracingConfig{
		ServiceName:    conf.Server.Name,
		ServiceVersion: conf.Version,
		Endpoint:       conf.Jaeger.Endpoint,
	})

	// 接管gin框架默认的日志和捕获异常
	eng.Use(
		middleware.TracingWithLogger(log, conf.Server.Name),
		middleware.Recovery(conf.Env == xconfig.EnvDebug || conf.Env == xconfig.EnvTest), // 测试或开发环境
	)

	// 跨域
	eng.Use(middleware.CORS(conf.Server.AllowOrigins))

	// 全局: 有 token 就解析写入 context,没有也放行(不强制)
	// 配合各业务组的 Auth() 中间件实现"默认公开,显式加锁"
	jwt, err := token.NewJwt(conf, redis)
	if err != nil {
		return nil, fmt.Errorf("jwt 初始化失败: %w", err)
	}
	eng.Use(middleware.WriteAuth(conf, jwt))
	// 限制
	// eng.Use(middleware.Limit(10))

	// 缺失路由
	eng.NoRoute(router.NotFoundHandle)

	// 探针路由
	router.Ping(eng)

	// 业务路由
	for _, r := range rtrs {
		r(eng, jwt, ctl)
	}

	return &Server{
		Conf:   conf,
		logger: log,
		tp:     tp,

		db:     db,
		Redis:  redis,
		Engine: eng,
	}, nil
}

func (s *Server) Start() error {
	// 验证器翻译
	validator.New("zh")

	readTimeout, writeTimeout := 120*time.Second, 120*time.Second
	// 默认 2min
	if s.Conf.Server.ReadTimeout != 0 {
		readTimeout = time.Duration(s.Conf.Server.ReadTimeout) * time.Second
	}
	if s.Conf.Server.WriteTimeout != 0 {
		writeTimeout = time.Duration(s.Conf.Server.WriteTimeout) * time.Second
	}
	port := s.Conf.Server.Port
	if port == 0 {
		port = utils.AvailablePort()
	}
	// 初始化HTTP/JOB服务
	svr := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.Engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	// 启动成功
	s.logger.Warn(fmt.Sprintf("Http server 启动 (Host: 0.0.0.0:%d  Pid:%d)", port, os.Getpid()))

	go func() {
		if err := svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error(fmt.Sprintf("%s ", err))
			s.Stop("")
		}
	}()

	s.server = svr
	return nil
}

func (s *Server) Stop(signal string) {

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
	// 关闭 db
	if s.db != nil {
		sqlDB, _ := s.db.DB()
		if sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				s.logger.Warn(fmt.Sprintf("db shutdown err:%v", err))
			}
		}
		s.logger.Warn("Close Db")
	}

	// 关闭 redis
	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			s.logger.Warn(fmt.Sprintf("redis shutdown err:%v", err))
		}
		s.logger.Warn("Close redis")
	}
	// 关闭tracer
	if s.tp != nil {
		if err := s.tp.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to shutdown tracer provider", xlog.KVErr(err))
		}
	}

	// TODO 其他关闭
	s.logger.Warn("Http server stopped")

	// 日志异步
	if err := s.logger.Sync(); err != nil {
		fmt.Printf("Failed to sync logger: %v\n", err)
	}
	// 关闭日志组件底层资源(如日志 Kafka Writer)
	if err := xlog.Close(); err != nil {
		fmt.Printf("Failed to close xlog: %v\n", err)
	}
}
