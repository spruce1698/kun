package cache

import "errors"

// ErrNotFound 缓存未命中或键不存在
var ErrNotFound = errors.New("cache: key not found")

// CacheKey 缓存键统一管理
const (
	DemoInfoKey = "demo:%d"

	WSTicketKey = "ws:ticket:%s" // WebSocket 一次性升级票据
)
