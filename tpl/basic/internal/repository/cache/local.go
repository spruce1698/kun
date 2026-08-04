package cache

import (
	"time"

	goCache "github.com/patrickmn/go-cache"
)

// LocalCache 本地缓存,直接转发到 go-cache(go-cache 本身并发安全,无需额外加锁)。
type LocalCache struct {
	cache *goCache.Cache
}

// 创建本地缓存
func NewLocalCache(defaultExpiration, cleanupInterval time.Duration) *LocalCache {
	return &LocalCache{
		cache: goCache.New(defaultExpiration, cleanupInterval),
	}
}

// 设置缓存
func (l *LocalCache) Set(key string, value any, expiration time.Duration) {
	l.cache.Set(key, value, expiration)
}

// 获取缓存
func (l *LocalCache) Get(key string) (any, bool) {
	return l.cache.Get(key)
}

// 删除缓存
func (l *LocalCache) Delete(key string) {
	l.cache.Delete(key)
}
