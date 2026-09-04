package middleware

import (
	"context"
	"math"
	"sync"
	"time"

	"basic/pkg/xhttp"
	"basic/pkg/xredis"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// loginRecord 登录尝试记录
type loginRecord struct {
	count     int       // 窗口内尝试次数
	firstSeen time.Time // 窗口起始时间
}

const (
	// maxRateLimitEntries 限制 map 最大条目数，防止内存泄漏
	maxRateLimitEntries = 10000

	// redisRateLimitLuaScript 使用 Redis ZSet 实现的分布式原子滑动窗口限流脚本
	// KEYS[1]: 限流 key
	// ARGV[1]: 当前时间戳(毫秒)
	// ARGV[2]: 时间窗口(毫秒)
	// ARGV[3]: 窗口内最大请求数
	redisRateLimitLuaScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local clearBefore = now - window

redis.call('ZREMRANGEBYSCORE', key, 0, clearBefore)
local current = redis.call('ZCARD', key)
if current < limit then
    redis.call('ZADD', key, now, now)
    redis.call('PEXPIRE', key, math.ceil(window))
    return 1
else
    return 0
end
`
)

// RateLimiter 基于 IP 的单机内存滑动窗口限流中间件(分段锁降低并发冲突,值类型降低堆分配)
// maxAttempts: 窗口内最大允许次数; window: 时间窗口; cooldown: 超限后封禁时长;
// cancelCtx: 可选的上下文,用于控制内部清理 goroutine 生命周期,进程退出或测试结束时取消即可。
func RateLimiter(maxAttempts int, window, cooldown time.Duration, cancelCtx ...context.Context) gin.HandlerFunc {
	ctx := context.Background()
	if len(cancelCtx) > 0 && cancelCtx[0] != nil {
		ctx = cancelCtx[0]
	}

	const numShards = 16
	type shard struct {
		mu      sync.Mutex
		records map[string]loginRecord
		banned  map[string]time.Time // IP -> 解封时间
	}

	shards := make([]shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = shard{
			records: make(map[string]loginRecord),
			banned:  make(map[string]time.Time),
		}
	}

	getShard := func(key string) *shard {
		var h uint32 = 2166136261
		for i := 0; i < len(key); i++ {
			h *= 16777619
			h ^= uint32(key[i])
		}
		return &shards[h%numShards]
	}

	// 定期清理协程——监听 ctx.Done() 退出，防止测试场景下 goroutine + Ticker 泄漏。
	// 生产环境中，ctx 由 app.NewHttp 注册进 xserver.Closer，进程退出时统一取消。
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				for i := 0; i < numShards; i++ {
					s := &shards[i]
					s.mu.Lock()
					for ip, t := range s.banned {
						if now.After(t) {
							delete(s.banned, ip)
						}
					}
					for ip, r := range s.records {
						if now.Sub(r.firstSeen) > window {
							delete(s.records, ip)
						}
					}
					if len(s.records) > maxRateLimitEntries/numShards {
						// 仍然超标时按最旧条目清理
						for ip, r := range s.records {
							if now.Sub(r.firstSeen) > window/2 {
								delete(s.records, ip)
								if len(s.records) <= (maxRateLimitEntries/numShards)*3/4 {
									break
								}
							}
						}
					}
					s.mu.Unlock()
				}
			}
		}
	}()

	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		s := getShard(ip)

		s.mu.Lock()
		// 检查是否在封禁中
		if until, ok := s.banned[ip]; ok {
			if time.Now().Before(until) {
				retryAfter := int(time.Until(until).Seconds()) + 1
				s.mu.Unlock()
				abortTooManyRequests(ctx, retryAfter)
				return
			}
			delete(s.banned, ip)
		}

		r, ok := s.records[ip]
		now := time.Now()
		if !ok || now.Sub(r.firstSeen) > window {
			s.records[ip] = loginRecord{count: 1, firstSeen: now}
			s.mu.Unlock()
			ctx.Next()
			return
		}

		r.count++
		if r.count > maxAttempts {
			s.banned[ip] = now.Add(cooldown)
			delete(s.records, ip)
			s.mu.Unlock()
			abortTooManyRequests(ctx, int(cooldown.Seconds()))
			return
		}
		s.records[ip] = r
		s.mu.Unlock()

		ctx.Next()
	}
}

// RedisRateLimiter 基于 Redis ZSet 的分布式原子滑动窗口限流中间件。
// 适用于多副本集群环境下的敏感接口(如验证码发送、登录鉴权、高频下单)。
// - rdb: redis 客户端指针
// - keyPrefix: 缓存键前缀,如 "ratelimit:login:"
// - maxRequests: 滑动窗口内最大允许请求次数
// - window: 滑动窗口大小(如 1*time.Minute)
// - keyFn: 可选的自定义 key 生成函数,默认按客户端真实 IP 限流
func RedisRateLimiter(
	rdb *xredis.Client,
	keyPrefix string,
	maxRequests int64,
	window time.Duration,
	keyFn ...func(ctx *gin.Context) string,
) gin.HandlerFunc {
	script := redis.NewScript(redisRateLimitLuaScript)
	windowMs := window.Milliseconds()
	if windowMs <= 0 {
		windowMs = 1000
	}

	return func(ctx *gin.Context) {
		if rdb == nil {
			// Redis 未注入时优雅降级放行
			ctx.Next()
			return
		}

		var identifier string
		if len(keyFn) > 0 && keyFn[0] != nil {
			identifier = keyFn[0](ctx)
		} else {
			identifier = ctx.ClientIP()
		}

		if identifier == "" {
			identifier = "unknown"
		}

		fullKey := keyPrefix + identifier
		nowMs := time.Now().UnixNano() / int64(time.Millisecond)

		res, err := script.Run(ctx.Request.Context(), rdb, []string{fullKey}, nowMs, windowMs, maxRequests).Result()
		if err != nil {
			// Redis 异常时放行以保障可用性
			ctx.Next()
			return
		}

		if allowed, ok := res.(int64); ok && allowed == 1 {
			ctx.Next()
			return
		}

		retryAfter := int(math.Ceil(window.Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
		abortTooManyRequests(ctx, retryAfter)
	}
}

// abortTooManyRequests 返回统一的 JSON 错误体并带上 Retry-After。
func abortTooManyRequests(ctx *gin.Context, retryAfterSec int) {
	xhttp.RateLimitFail(ctx, retryAfterSec)
}
