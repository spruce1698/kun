package cache

import (
	"sync"
	"time"

	goCache "github.com/patrickmn/go-cache"
)

// defaultMaxEntries 本地缓存的默认条目上限。
//
// go-cache 只按 TTL 淘汰,不按容量淘汰:没有上限时,攻击者遍历 id 请求详情接口,
// 就能在一个 TTL 窗口内把任意多条记录塞进进程内存,直到 OOM(可远程触发)。
const defaultMaxEntries = 10000

// LocalCache 本地缓存。go-cache 本身并发安全,这里额外加一层容量上限控制。
type LocalCache struct {
	cache      *goCache.Cache
	maxEntries int
	// evictMu 保护 evict + Set 的组合操作。
	// 使用互斥锁而非 atomic.Bool：当 A 正在淘汰时，B 不应跳过等待直接写入，
	// 否则 N 个并发 goroutine 会同时突破 maxEntries 上限，导致内存无界增长。
	// double-check 降低锁粒度：只在确认超限时才进入临界区。
	evictMu sync.Mutex
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
		l.evictMu.Lock()
		// double-check：多个 goroutine 同时发现超限后，只有一个先拿到锁做淘汰，
		// 后续等待者拿到锁时重新判断，若已不超限则直接跳过，避免重复淘汰。
		if l.cache.ItemCount() >= l.maxEntries {
			l.evict()
		}
		l.evictMu.Unlock()
	}
	l.cache.Set(key, value, expiration)
}

// evict 先清理已过期条目;若仍超限,再删除一批(go-cache 无 LRU 元数据,
// map 遍历顺序随机,因此这里是随机淘汰而非最近最少使用)。
// 本地缓存只是 Redis 的前置副本,淘汰错了最多多一次 Redis 往返,不影响正确性。
// 调用方须持有 evictMu 锁。
func (l *LocalCache) evict() {
	l.cache.DeleteExpired()
	over := l.cache.ItemCount() - l.maxEntries
	if over < 0 {
		return
	}
	// 多删 1/8 容量作为缓冲,避免每次 Set 都触发淘汰。
	target := over + 1 + l.maxEntries/8
	for k := range l.cache.Items() {
		if target <= 0 {
			break
		}
		l.cache.Delete(k)
		target--
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
