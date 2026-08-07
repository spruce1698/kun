/**
 * @Author: spruce
 * @Date: 2024-03-28 21:36
 * @Desc: demo cache
 */

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"advanced/pkg/xredis"

	"errors"
)

var _ DemoCache = (*demoCache)(nil)

type (
	DemoCache interface {
		// 查询缓存
		Get(ctx context.Context, id int64) (data *Demo, err error)
		// 添加缓存
		Set(ctx context.Context, id int64, data *Demo, expiration int64) (err error)
		// 删除缓存(Redis + 本地),用于数据更新/删除后失效缓存
		Delete(ctx context.Context, id int64) error
	}
	demoCache struct {
		common     *xredis.Client
		localCache *LocalCache
	}

	//  Demo缓存数据
	Demo struct {
		Id     int64
		Name   string // 名称
		Test4  int32  // 测试4
		RoleId int64  // 角色id
	}
)

const (
	// defaultExpiration 当调用方未指定(<=0)TTL 时的默认缓存时长。
	// 避免传 0 时 Redis Set 永不过期,导致脏数据长期残留(需手动 Delete 才能失效)。
	defaultExpiration = 5 * time.Minute
)

func NewDemoCache(c *xredis.Client) DemoCache {
	return &demoCache{
		common:     c,
		localCache: NewLocalCache(5*time.Minute, 10*time.Minute),
	}
}

func (d *demoCache) Get(ctx context.Context, id int64) (*Demo, error) {
	key := fmt.Sprintf(DemoInfoKey, id)

	// 1. 查询本地缓存
	if value, ok := d.localCache.Get(key); ok {
		if demo, ok := value.(*Demo); ok {
			return demo, nil
		}
	}

	// 2. 查询Redis缓存
	var demo *Demo
	data, err := d.common.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, xredis.Nil) {
			return nil, fmt.Errorf("cache miss: %w", xredis.Nil)
		}
		return nil, fmt.Errorf("get cache failed: %w", err)
	}
	err = json.Unmarshal(data, &demo)
	if err != nil {
		return nil, err
	}
	// 设置本地缓存
	d.localCache.Set(key, demo, 5*time.Minute)
	return demo, nil
}

func (d *demoCache) Set(ctx context.Context, id int64, data *Demo, expiration int64) error {
	key := fmt.Sprintf(DemoInfoKey, id)

	value, valueErr := json.Marshal(data)
	if valueErr != nil {
		return valueErr
	}
	// expiration<=0 时用默认 TTL,避免 Redis key 永不过期导致脏数据残留
	ttl := time.Duration(expiration) * time.Second
	if expiration <= 0 {
		ttl = defaultExpiration
	}
	err := d.common.Set(ctx, key, string(value), ttl).Err()
	if err != nil {
		return err
	}
	return nil
}

func (d *demoCache) Delete(ctx context.Context, id int64) error {
	key := fmt.Sprintf(DemoInfoKey, id)

	// 本地缓存是 Redis 的副本,无论 Redis 删除是否成功都先清本地,避免残留旧值。
	d.localCache.Delete(key)

	// 删除 Redis,失败时包装原始错误(不吞原因)。
	if err := d.common.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete cache failed: %w", err)
	}
	return nil
}

// TODO Warmup 缓存预热
func (d *demoCache) Warmup(ctx context.Context) error {

	return nil
}
