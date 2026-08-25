package svc

import (
	"context"
	"fmt"

	"basic/internal/repository/cache"
	"basic/internal/repository/db"
	"basic/pkg/xerror"
	"basic/pkg/xlog"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
)

//go:generate mockgen -source=./demo.go -destination=../../../test/mocks/service/demo.go  -package mock_service

var _ DemoSvc = (*demoSvc)(nil)

type (
	DemoSvc interface {
		Detail(ctx context.Context, id int64) (*Demo, error)
		FindList(ctx context.Context, args *DemoListArgs) ([]*Demo, int64, error)
		Create(ctx context.Context, args *Demo) error
		Update(ctx context.Context, args *Demo) error
		Delete(ctx context.Context, id int64) error
		SoftDelete(ctx context.Context, id int64) error
	}
	DemoCtx struct {
		*Ctx
		DemoDb    db.DemoDb
		DemoCache cache.DemoCache
	}
	demoSvc struct {
		ctx *DemoCtx
		sf  singleflight.Group
	}

	DemoListArgs struct {
		OrderField string // 排序字段
		OrderType  int64  // 排序类型 0:降序(默认),1:升序
		Page       int64  // 添加验证规则
		PageSize   int64  // 添加验证规则
		// LastId 游标分页的上一页最大/最小 id。必须存在这个字段,否则 handler 里
		// xhttp.PageArg.LastId 经 copier.Copy 后无处可去被丢弃,
		// repo 层的游标分页分支(args.LastId > 0)永远进不去,深分页优化形同虚设。
		LastId int64

		Id   *int64
		Name string
	}

	Demo struct {
		Id    int64
		Name  string
		Test1 float64
		Test4 int32
	}
)

func NewDemoSvc(ctx *DemoCtx) DemoSvc {
	return &demoSvc{
		ctx: ctx,
	}
}

// 查找一个(接入 singleflight 防缓存击穿)
func (d *demoSvc) Detail(ctx context.Context, id int64) (*Demo, error) {
	if id <= 0 {
		return nil, xerror.NewError(xerror.InvalidArgument, "Get Demo Detail invalid id", nil)
	}

	xlog.Info(ctx, "Get Demo Detail", "测试手工日志")

	// 1. 优先查缓存 (LocalCache + Redis)
	demo, cacheErr := d.ctx.DemoCache.Get(ctx, id)
	if cacheErr == nil {
		result := &Demo{}
		_ = copier.Copy(result, demo)
		return result, nil
	}

	// 2. 缓存未命中: singleflight 合并同 ID 并发请求,防缓存击穿
	val, err, _ := d.sf.Do(fmt.Sprintf("demo:detail:%d", id), func() (any, error) {
		// double check 缓存,避免并发排队协程重复打库
		if dCached, cErr := d.ctx.DemoCache.Get(ctx, id); cErr == nil {
			return dCached, nil
		}

		demoDb, dbErr := d.ctx.DemoDb.Find(ctx, id)
		if dbErr != nil {
			return nil, dbErr
		}

		cacheData := &cache.Demo{
			Id:    demoDb.Id,
			Name:  demoDb.Name,
			Test1: demoDb.Test1,
			Test4: demoDb.Test4,
		}
		// 缓存写入 5 分钟过期,与本地缓存对齐;失败仅记日志不阻断主流程。
		if cErr := d.ctx.DemoCache.Set(ctx, id, cacheData, 300); cErr != nil {
			xlog.Error(ctx, fmt.Sprintf("set demo cache failed, id=%d", id), cErr)
		}
		return cacheData, nil
	})

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, xerror.NewError(xerror.BusinessError, "没有相关记录", err)
		}
		return nil, xerror.NewError(xerror.BusinessError, "Get Demo Detail 失败", err)
	}

	result := &Demo{}
	_ = copier.Copy(result, val)
	return result, nil
}

// 查找列表
func (d *demoSvc) FindList(ctx context.Context, args *DemoListArgs) ([]*Demo, int64, error) {

	dbArgs := &db.DemoSearch{}
	_ = copier.Copy(dbArgs, args)

	list, total, listErr := d.ctx.DemoDb.ListWithTotal(ctx, dbArgs)
	if listErr != nil {
		if errors.Is(listErr, db.ErrNotFound) {
			return []*Demo{}, 0, nil
		}
		return nil, 0, xerror.NewError(xerror.BusinessError, "Get Demo List 失败", listErr)
	}
	result := make([]*Demo, 0, len(list))
	if len(list) > 0 {
		for _, demo := range list {
			var item Demo
			_ = copier.Copy(&item, demo)
			result = append(result, &item)
		}
	}
	return result, total, nil
}

// 创建
func (d *demoSvc) Create(ctx context.Context, args *Demo) error {
	_, dbErr := d.ctx.DemoDb.Insert(ctx, &db.Demo{
		Name:  args.Name,
		Test1: args.Test1,
		Test4: args.Test4,
	})
	if dbErr != nil {
		return xerror.NewError(xerror.BusinessError, "Create Demo 失败", dbErr)
	}
	return nil
}

// 修改(事务示例)
func (d *demoSvc) Update(ctx context.Context, args *Demo) error {
	if args.Id > 0 {
		// 两种事务方式
		//方式1:
		//err := d.ctx.Conn.Tx(ctx, func(ctx context.Context) error {
		//	preOne := args.Id - 1
		//	demo, err := d.ctx.DemoDb.Find(ctx, preOne)
		//	if err != nil {
		//		return err
		//	}
		//	err = d.ctx.DemoDb.Delete(ctx, []int64{preOne})
		//	if err != nil {
		//		return err
		//	}
		//	if demo.Id == 3 {
		//		return errors.New("id是3,不能修改")
		//	}
		//	_, err = d.ctx.DemoDb.UpdateFields(ctx, args.Id, map[string]any{"name": args.Name, "test1": args.Test1, "test4": args.Test4})
		//	return err
		//})
		// 方式2:
		err := d.ctx.DemoDb.DemoUpdateTrans(ctx, args.Id, &db.Demo{
			Name:  args.Name,
			Test1: args.Test1,
			Test4: args.Test4,
		})

		if err != nil {
			return xerror.NewError(xerror.BusinessError, "Update Demo 失败", err)
		}
		// DemoUpdateTrans 内部会删除 id-1 的记录,故 args.Id 与 args.Id-1 的缓存都要失效。
		d.invalidateCache(ctx, args.Id)
		d.invalidateCache(ctx, args.Id-1)
		return nil
	}
	return xerror.NewError(xerror.BusinessError, "Update Demo 失败", nil)
}

// 物理删除
func (d *demoSvc) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return xerror.NewError(xerror.InvalidArgument, "Delete Demo invalid id", nil)
	}
	err := d.ctx.DemoDb.Delete(ctx, []int64{id})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "Delete Demo 失败", err)
	}
	d.invalidateCache(ctx, id)
	return nil
}

// 逻辑删除
func (d *demoSvc) SoftDelete(ctx context.Context, id int64) error {
	if id <= 0 {
		return xerror.NewError(xerror.InvalidArgument, "SoftDelete Demo invalid id", nil)
	}
	err := d.ctx.DemoDb.SoftDelete(ctx, []int64{id})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "SoftDelete Demo 失败", err)
	}
	d.invalidateCache(ctx, id)
	return nil
}

// invalidateCache 失效指定 id 的缓存(Redis + 本地)。
// 缓存删除失败不阻断主流程(数据已落库),仅记录日志,等 TTL 自然过期。
func (d *demoSvc) invalidateCache(ctx context.Context, id int64) {
	if id <= 0 {
		return
	}
	if err := d.ctx.DemoCache.Delete(ctx, id); err != nil {
		xlog.Error(ctx, fmt.Sprintf("invalidate demo cache failed, id=%d", id), err)
	}
}
