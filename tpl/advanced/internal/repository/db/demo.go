package db

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

//go:generate mockgen -source=./demo.go -destination=../../../test/mocks/repository/db/demo.go -package mock_repo_db -aux_files db=./demo_gen.go

var _ DemoDb = (*customDemoDb)(nil)

type (
	DemoDb interface {
		demoDb

		ListWithTotal(ctx context.Context, args *DemoSearch) ([]*Demo, int64, error)
		ListWithMore(ctx context.Context, args *DemoSearch) ([]*Demo, bool, error)
		UpdateTrans(ctx context.Context, id int64, upData *Demo) error
		FindStream(ctx context.Context, handler func(*Demo) error) error
		FindByName(ctx context.Context, name string) (*Demo, error)

		// TODO: add custom functions here and delete this line
	}

	customDemoDb struct {
		*defaultDemoDb
	}

	DemoSearch struct {
		SearchPage
	}

	// TODO: add struct here and delete this line
)

func NewDemoDb(c *Conn) DemoDb {
	return &customDemoDb{
		defaultDemoDb: newDemoDb(c),
	}
}

func (c *customDemoDb) ListWithTotal(ctx context.Context, args *DemoSearch) ([]*Demo, int64, error) {
	// filter 每次都从 c.WithContext(ctx).Model(c.model) 起全新的 *gorm.DB 再叠加过滤条件,
	// 各调用之间不共享 Statement,天然无污染,无需手动 Session 克隆。过滤逻辑只写这一处。
	filter := func() *gorm.DB {
		d := c.WithContext(ctx).Model(c.model)
		// TODO 自定义条件处理

		return d
	}

	var total int64
	if err := filter().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, ErrNotFound
	}

	order := c.HandleRank(
		args.OrderField,
		args.OrderType,
		DemoFields,
		TableDemo+".id",
	)

	offset, limit := c.HandlePage(args.Page, args.PageSize)

	var model *gorm.DB
	switch {
	case args.LastId > 0 && args.Page > 1: // 游标分页: time>lastTime or (time==lastTime and id>lastId)
		lastCond := "`id` < ?"
		if args.OrderType == 1 {
			lastCond = "`id` > ?"
		}
		model = filter().Where(lastCond, args.LastId).Order(order).Limit(limit)
	case offset > 100000: // 深分页: SELECT * FROM Demo INNER JOIN (SELECT id FROM Demo WHERE ... ORDER BY id LIMIT ?,?) AS tmp USING(id)
		subQuery := filter().Order(order).Select("id").Offset(offset).Limit(limit)
		model = c.WithContext(ctx).Model(c.model).Order(order).Joins("INNER JOIN (?) AS tmp USING(id)", subQuery)
	default:
		model = filter().Order(order).Offset(offset).Limit(limit)
	}

	result := make([]*Demo, 0, limit)
	if err := model.Find(&result).Error; err != nil {
		return nil, 0, err
	}
	// total>0 但本页结果为空(如并发删除/分页越界),返回空列表 + 真实 total,不当作 ErrNotFound,
	// 让上层据此正确渲染分页(而非误判"无数据")。total==0 在上面已提前返回 ErrNotFound。
	return result, total, nil
}

func (c *customDemoDb) ListWithMore(ctx context.Context, args *DemoSearch) ([]*Demo, bool, error) {
	// filter 每次都从 c.WithContext(ctx).Model(c.model) 起全新的 *gorm.DB 再叠加过滤条件,
	// 各调用之间不共享 Statement,天然无污染。过滤逻辑只写这一处。
	filter := func() *gorm.DB {
		d := c.WithContext(ctx).Model(c.model)
		// TODO 自定义条件处理

		return d
	}

	order := c.HandleRank(
		args.OrderField,
		args.OrderType,
		DemoFields,
		TableDemo+".id",
	)

	offset, limit := c.HandlePage(args.Page, args.PageSize)
	// 在请求的数据基础上+1，以此来判断是否还有数据
	want := limit
	limit++

	var model *gorm.DB
	switch {
	case args.LastId > 0 && args.Page > 1: // 游标分页
		lastCond := "`id` < ?"
		if args.OrderType == 1 {
			lastCond = "`id` > ?"
		}
		model = filter().Where(lastCond, args.LastId).Order(order).Limit(limit)
	case offset > 100000: // 深分页
		subQuery := filter().Order(order).Select("id").Offset(offset).Limit(limit)
		model = c.WithContext(ctx).Model(c.model).Order(order).Joins("INNER JOIN (?) AS tmp USING(id)", subQuery)
	default:
		model = filter().Order(order).Offset(offset).Limit(limit)
	}

	result := make([]*Demo, 0, limit)
	if err := model.Find(&result).Error; err != nil {
		return nil, false, err
	}

	ln := len(result)
	if ln == 0 {
		return nil, false, ErrNotFound
	}
	var hasMore bool
	if ln > want {
		result = result[:want]
		hasMore = true
	}

	return result, hasMore, nil
}

func (c *customDemoDb) UpdateTrans(ctx context.Context, id int64, upData *Demo) error {
	err := c.Conn.Tx(ctx, func(ctx context.Context) error {
		preOne := id - 1
		demo, err := c.Find(ctx, preOne)
		if err != nil {
			return err
		}
		err = c.Delete(ctx, []int64{preOne})
		if err != nil {
			return err
		}
		if demo.Id == 3 {
			return errors.New("id是3,不能修改")
		}
		_, err = c.UpdateFields(ctx, id, map[string]any{"name": upData.Name, "test1": upData.Test1, "test4": upData.Test4})
		return err
	})
	return err
}

func (c *customDemoDb) FindStream(ctx context.Context, handler func(*Demo) error) error {
	// 复用同一 *gorm.DB session 扫描,避免每行都 WithContext 重新派生。
	// rows 生命周期由 ctx 控制:ctx 取消时 Rows()/Next() 返回错误退出。
	db := c.WithContext(ctx).Model(c.model).Order("`id` ASC")
	rows, err := db.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item Demo
		if err := db.ScanRows(rows, &item); err != nil {
			return err
		}
		if err := handler(&item); err != nil {
			return err
		}
	}
	return rows.Err()
}

// FindByName 根据名称查找(登录账号)
func (c *customDemoDb) FindByName(ctx context.Context, name string) (*Demo, error) {
	result := &Demo{}
	err := c.WithContext(ctx).Where(" `name` = ? ", name).First(result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}
