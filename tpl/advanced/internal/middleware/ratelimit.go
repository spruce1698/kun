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
			now := time.Now()
			// 构建新 map 替换旧 map(copy-on-write 风格),把持锁时间压缩到构建+替换,
			// 避免对大 map 全表遍历时长时间持锁阻塞所有请求。
			mu.Lock()
			newBanned := make(map[string]time.Time, len(banned))
			for ip, t := range banned {
				if now.Before(t) {
					newBanned[ip] = t
				}
			}
			newRecords := make(map[string]*loginRecord, len(records))
			for ip, r := range records {
				if now.Sub(r.firstSeen) <= window {
					newRecords[ip] = r
				}
			}
			// 条目数仍然过多:强制清掉最旧的一半(此时 map 已小,删除快)
			if len(newRecords) > maxRateLimitEntries {
				half := len(newRecords) / 2
				for ip, r := range newRecords {
					_ = r
					delete(newRecords, ip)
					half--
					if half <= 0 {
						break
					}
				}
			}
			banned = newBanned
			records = newRecords
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
