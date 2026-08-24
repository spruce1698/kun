/**
 * @Author:
 * @Date: 2024-03-28 13:45
 * @Desc: redis
 */

package xredis

import (
	"context"
	"fmt"
	"time"

	"advanced/pkg/xconfig"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

const Nil = redis.Nil

type Client struct {
	redis.UniversalClient
}

func New(conf *xconfig.Conf) (*Client, error) {
	if len(conf.Redis.Source) == 0 {
		return nil, fmt.Errorf("redis 配置错误")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var client redis.UniversalClient
	if conf.Redis.Cluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    conf.Redis.Source,
			Password: conf.Redis.Password,
			PoolSize: conf.Redis.PoolSize,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     conf.Redis.Source[0],
			Password: conf.Redis.Password, // no password set
			DB:       conf.Redis.DB,       // redis db 索引
			PoolSize: conf.Redis.PoolSize, // 连接池大小
		})
	}

	// 使用官方的 OpenTelemetry 插件
	if err := redisotel.InstrumentTracing(client); err != nil {
		return nil, fmt.Errorf("InstrumentTracing redis[%v] failed: %w", conf.Redis.Source[0], err)
	}

	c := &Client{client}

	_, err := c.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("ping redis[%v]失败: %w", conf.Redis.Source[0], err)
	}
	return c, nil
}
