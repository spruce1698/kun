
package {{ .PackageName }}

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"{{ .ProjectName }}/pkg/xredis"
)

var _ {{ .FileName }}Cache = (*{{ .FileNameTitleLower }}Cache)(nil)

type (
	{{ .FileName }}Cache interface {
		// 查询缓存
		Get(ctx context.Context, id int64) (*{{ .FileName }}, error)
		// 添加缓存
		Set(ctx context.Context, id int64, data *{{ .FileName }}, expiration int64) error
        // 删除缓存
        Delete(ctx context.Context, id int64) error
        // 预热缓存
        Warmup(ctx context.Context) error
        // TODO: add your code here and delete this line
	}
	{{ .FileNameTitleLower }}Cache struct {
		common     *xredis.Client
		lCache *LocalCache
	}

	//  缓存数据的结构体
	{{ .FileName }} struct {
        // TODO: add your code here and delete this line
	}

    // TODO: add your code here and delete this line
)

func New{{ .FileName }}Cache(c *xredis.Client) {{ .FileName }}Cache {
	return &{{ .FileNameTitleLower }}Cache{
		common:     c,
		lCache: NewLocalCache(5*time.Minute, 10*time.Minute),
	}
}

func ({{ .FileNameFirstChar }} *{{ .FileNameTitleLower }}Cache) Get(ctx context.Context, id int64) (*{{ .FileName }}, error) {
	key := fmt.Sprintf({{ .FileName }}DataKey, id)

	// 1.查询本地缓存
	if value, ok := {{ .FileNameFirstChar }}.lCache.Get(key); ok {
		if {{ .FileNameTitleLower }}, ok := value.(*{{ .FileName }}); ok {
			return {{ .FileNameTitleLower }}, nil
		}
	}

	// 2.查询Redis缓存
	{{ .FileNameTitleLower }} := &{{ .FileName }}{}
	data, err := {{ .FileNameFirstChar }}.common.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, xredis.Nil) {
			return nil, errors.New("cache miss")
		}
		return nil, errors.New("get cache failed")
	}
	err = json.Unmarshal(data, {{ .FileNameTitleLower }})
	if err != nil {
		return nil, err
	}
	// 3.设置本地缓存
	{{ .FileNameFirstChar }}.lCache.Set(key, {{ .FileNameTitleLower }}, 5*time.Minute)
	return {{ .FileNameTitleLower }}, nil
}

func ({{ .FileNameFirstChar }} *{{ .FileNameTitleLower }}Cache) Set(ctx context.Context, id int64, data *{{ .FileName }}, expiration int64) error {
	key := fmt.Sprintf({{ .FileName }}DataKey, id)

	value, valueErr := json.Marshal(data)
	if valueErr != nil {
		return valueErr
	}
	err := {{ .FileNameFirstChar }}.common.Set(ctx, key, string(value), time.Duration(expiration)*time.Second).Err()
	if err != nil {
		return err
	}
	return nil
}

func ({{ .FileNameFirstChar }} *{{ .FileNameTitleLower }}Cache) Delete(ctx context.Context, id int64) error {
	key := fmt.Sprintf({{ .FileName }}DataKey, id)

	if err := {{ .FileNameFirstChar }}.common.Del(ctx, key).Err(); err != nil {
		return errors.New("delete cache failed")
	}

	// 设置本地缓存
	{{ .FileNameFirstChar }}.lCache.Delete(key)
	return nil
}


func ({{ .FileNameFirstChar }} *{{ .FileNameTitleLower }}Cache) Warmup(ctx context.Context) error {
	return nil
}

// TODO: add your code here and delete this line