package cache

import (
	"sync/atomic"
	"time"

	goCache "github.com/patrickmn/go-cache"
)

// defaultMaxEntries 本地缓存的默认条目上限。
//
// go-cache 只按 TTL 淘汰,不按容量淘汰:没有上限时,攻击者遍历 id 请求详情接口,
// 就能在一个 TTL 窗口内把任意多条记录塞进进程内存,直到 OOM(可远程触发)。
const defaultMaxEntries = 10000

// LocalCache 本地缓存。go-cache 本身并发安全,这里额外加一层容量上限。
type LocalCache struct {
	cache      *goCache.Cache
	maxEntries int
	evicting   atomic.Bool
}

// NewLocalCache 创建本地缓存,使用默认容量上限。
func NewLocalCache(defaultExpiration, cleanupInterval time.Duration) *LocalCache {
	return NewLocalCacheWithLimit(defaultExpiration, cleanupInterval, defaultMaxEntries)
}

// NewLocalCacheWithLimit 创建本地缓存并指定条目上限。maxEntries <= 0 时使用默认值。
func NewLocalCacheWithLimit(defaultExpiration, cleanupInterval time.Duration, maxEntries int) *LocalCache {
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}
	return &LocalCache{
		cache:      goCache.New(defaultExpiration, cleanupInterval),
		maxEntries: maxEntries,
	}
}

// Set 设置缓存。超过容量上限时先做一次淘汰,保证内存占用有界。
func (l *LocalCache) Set(key string, value any, expiration time.Duration) {
	// 已存在的 key 是覆盖写,不会增加条目数,无需淘汰。
	if _, exists := l.cache.Get(key); !exists && l.cache.ItemCount() >= l.maxEntries {
		if l.evicting.CompareAndSwap(false, true) {
			l.evict()
			l.evicting.Store(false)
		}
	}
	l.cache.Set(key, value, expiration)
}

// evict 先清理已过期条目;若仍超限,则清空本地缓存重置。
// 本地缓存只是 Redis 的前置 L1 副本,清理后由后续请求重新载入,避免 Items() 全量克隆 Map。
func (l *LocalCache) evict() {
	l.cache.DeleteExpired()
	if l.cache.ItemCount() >= l.maxEntries {
		l.cache.Flush()
	}
}

// 获取缓存
func (l *LocalCache) Get(key string) (any, bool) {
	return l.cache.Get(key)
}

// 删除缓存
func (l *LocalCache) Delete(key string) {
	l.cache.Delete(key)
}

// ItemCount 返回当前缓存条目数(用于监控/测试)。
func (l *LocalCache) ItemCount() int {
	return l.cache.ItemCount()
}
