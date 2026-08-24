/**
 * @Author: spruce
 * @Date: 2024-08-14
 * @Desc: 应用装配层
 *
 * 将 gin 引擎的组装(中间件/JWT/限流/路由)与资源关闭器注册从 pkg/xserver/http
 * 上移到此处,使 pkg/xserver/http 不再反向依赖 internal/*,依赖方向回到 internal -> pkg。
 * 返回组装好的 *gin.Engine 与统一的资源关闭器 *xserver.Closer。
 *
 * jwt 由 wire 注入(与 service 层共用同一单例,保证黑名单/轮换全局生效)。
 */

package app

import (
	"context"
	"time"

	"basic/internal/handler"
	"basic/internal/middleware"
	"basic/internal/router"
	"basic/pkg/token"
	"basic/pkg/validator"
	"basic/pkg/xconfig"
	"basic/pkg/xdb"
	"basic/pkg/xlog"
	"basic/pkg/xredis"
	"basic/pkg/xserver"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

// Assembly 装配产物。
type Assembly struct {
	Engine *gin.Engine
	// Closer 聚合进程退出时需回收的全部资源(db/redis/tracer/xlog),
	// 交给 http.Server 在 Stop 时按 LIFO 统一关闭。
	Closer *xserver.Closer
}

// NewHttp 装配 http 引擎并注册资源关闭器:
// gin 模式、中间件、JWT、限流、路由,以及 db/redis/tracer/xlog 的回收。
// jwt 由 wire 注入,与 service 层共用同一实例。
func NewHttp(
	conf *xconfig.Conf,
	log *xlog.Logger,
	db *xdb.Client,
	redis *xredis.Client,
	jwt *token.Jwt,
	hdl *handler.Ctx,
	rtrs []router.Router,
) (*Assembly, error) {
	// 初始化校验器与中文翻译器
	validator.New("zh")

	// 在创建引擎前统一设置 gin 全局模式与输出,避免 gin.New() 先于 SetMode,
	// 也避免多 server 实例对全局 DefaultWriter 的重复覆盖。
	xserver.InitGinMode(conf.Env)
	eng := gin.New()

	// 性能分析 - 正式环境不要使用！！！
	if conf.Env == xconfig.EnvDebug {
		pprof.Register(eng)
	}

	// 初始化 全链路跟踪 tracer
	tp := xlog.InitTracer(xlog.TracingConfig{
		ServiceName:    conf.Server.Name,
		ServiceVersion: conf.Version,
		Endpoint:       conf.Jaeger.Endpoint,
		SampleRate:     conf.Jaeger.SampleRate,
	})

	// 接管gin框架默认的日志和捕获异常。
	// Recovery 必须最先注册(位于中间件链最外层),否则 Tracing 自身 panic 时无人兜住,
	// 会直接冒泡到 net/http 变成空的 500 且不打业务日志。
	eng.Use(
		middleware.Recovery(conf.Env == xconfig.EnvDebug || conf.Env == xconfig.EnvTest), // 测试或开发环境回显堆栈
		xlog.TracingWithLogger(log, conf.Server.Name),
	)

	// 跨域
	eng.Use(middleware.CORS(conf.Server.AllowOrigins))

	// Prometheus 吞吐与耗时指标监控
	eng.Use(middleware.Metrics(conf.Server.Name))

	// 全局: 有 token 就解析写入 context,没有也放行(不强制)
	// 配合各业务组的 Auth() 中间件实现"默认公开,显式加锁"
	eng.Use(middleware.WriteAuth(conf, jwt))

	// 全局限流: 每 IP 每分钟最多 600 次,超限封禁 1 分钟。
	// 登录等敏感接口在各自路由上再叠加更严格的 RateLimiter / RedisRateLimiter。
	// 注意:状态在进程内存,多副本部署时实际阈值 = 该值 × 副本数;
	// 需要精确全局限流请换成基于 Redis 的令牌桶。
	eng.Use(middleware.RateLimiter(600, time.Minute, time.Minute))

	// 缺失路由
	eng.NoRoute(router.NotFoundHandle)

	// 探针路由
	router.Ping(eng)
	// Prometheus 监控指标端点
	router.Metrics(eng)

	// 业务路由
	for _, r := range rtrs {
		r(eng, jwt, hdl)
	}

	// ============ 资源关闭器(LIFO 关闭)============
	// 注册顺序与期望关闭顺序相反:后注册的先关。
	// 期望关闭顺序:http(由 Server.Stop 直接关) -> db -> redis -> tracer -> xlog
	// 因此注册顺序:xlog(最先注册,最后关) -> tracer -> redis -> db
	closer := xserver.NewCloser()
	closer.Add(func() error { return xlog.Close() }, "xlog")
	if tp != nil {
		closer.Add(func() error {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return tp.Shutdown(shutdownCtx)
		}, "tracer")
	}
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

	return &Assembly{
		Engine: eng,
		Closer: closer,
	}, nil
}
