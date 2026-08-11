package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// loginRecord 登录尝试记录
type loginRecord struct {
	count     int       // 窗口内尝试次数
	firstSeen time.Time // 窗口起始时间
}

const (
	// maxRateLimitEntries 限制 map 最大条目数，防止内存泄漏
	maxRateLimitEntries = 10000
)

// RateLimiter 基于 IP 的简单滑动窗口限流中间件
// maxAttempts: 窗口内最大允许次数; window: 时间窗口; cooldown: 超限后封禁时长
// 每次调用都会启动一个绑定到本实例闭包变量的清理协程,随实例的 records/banned 一起回收,
// 避免使用包级 sync.Once 导致只清理首个实例的 map、其余实例内存泄漏。
func RateLimiter(maxAttempts int, window, cooldown time.Duration) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		records = make(map[string]*loginRecord)
		banned  = make(map[string]time.Time) // IP -> 解封时间
	)

	// 清理协程:绑定本实例的 records/banned,随进程退出或实例不再被引用而结束。
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, t := range banned {
				if now.After(t) {
					delete(banned, ip)
				}
			}
			for ip, r := range records {
				if now.Sub(r.firstSeen) > window {
					delete(records, ip)
				}
			}
			// 条目数超限时强制删除一半以控制 map 规模。
			// 注意:Go map 遍历顺序随机,这里删除的是"随机一半"而非"最旧一半",
			// 仅用于兜底限流,不保证按 firstSeen 淘汰。
			if len(records) > maxRateLimitEntries {
				half := len(records) / 2
				for ip := range records {
					delete(records, ip)
					half--
					if half <= 0 {
						break
					}
				}
			}
			mu.Unlock()
		}
	}()

	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()

		mu.Lock()
		// 检查是否在封禁中
		if until, ok := banned[ip]; ok && time.Now().Before(until) {
			mu.Unlock()
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		r, ok := records[ip]
		now := time.Now()
		if !ok || now.Sub(r.firstSeen) > window {
			// 新窗口
			records[ip] = &loginRecord{count: 1, firstSeen: now}
			mu.Unlock()
			ctx.Next()
			return
		}

		r.count++
		if r.count > maxAttempts {
			// 超限，封禁
			banned[ip] = now.Add(cooldown)
			delete(records, ip)
			mu.Unlock()
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		mu.Unlock()

		ctx.Next()
	}
}
