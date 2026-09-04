package db

import (
	"context"
	"errors"
)

//go:generate mockgen -source=./demo.go -destination=../../../test/mocks/repository/db/demo.go -package mock_repo_db -aux_files db=./demo_gen.go

var _ DemoDb = (*customDemoDb)(nil)

type (
	DemoDb interface {
		demoDb

		ListWithTotal(ctx context.Context, args *DemoSearch) ([]*Demo, int64, error)
		ListWithMore(ctx context.Context, args *DemoSearch) ([]*Demo, bool, error)
		DemoUpdateTrans(ctx context.Context, id int64, upData *Demo) error
		FindStream(ctx context.Context, handler func(*Demo) error) error
		FindByName(ctx context.Context, name string) (*Demo, error)

		// TODO: add custom functions here and delete this line
	}

	customDemoDb struct {
		*defaultDemoDb
	}

	DemoSearch struct {
		SearchPage

		Id   *int64 `json:"id"`   // id
		Name string `json:"name"` // 名称
	}

	// TODO: add struct here and delete this line
)

func NewDemoDb(c *Conn) DemoDb {
	return &customDemoDb{
		defaultDemoDb: newDemoDb(c),
	}
}

func (c *customDemoDb) buildListFilter(ctx context.Context, args *DemoSearch) *DB {
	d := c.WithContext(ctx).Model(c.model)
	// TODO 自定义条件处理

	if args.Name != "" {
		d = d.Where(" name LIKE ? ", "%"+args.Name+"%")
	}
	if args.Id != nil {
		d = d.Where(" id = ? ", args.Id)
	}

	return d
}

func (c *customDemoDb) buildListQuery(ctx context.Context, args *DemoSearch, limit int) *DB {
	filter := c.buildListFilter(ctx, args)
	order := c.HandleRank(
		args.OrderField,
		args.OrderType,
		DemoFields,
		TableDemo+".id",
	)

	offset, _ := c.HandlePage(args.Page, args.PageSize)

	switch {
	case args.LastId > 0: // 游标分页
		// 游标按主键 id 比较,因此排序也必须按 id,否则(如按其它列排序而按主键取游标)
		// 翻页结果会重复或漏行。需要按其它列做游标分页时,应改为 (orderCol, id) 复合游标。
		cursorOrder := TableDemo + ".id DESC"
		lastCond := TableDemo + ".`id` < ?"
		if args.OrderType == 1 {
			cursorOrder = TableDemo + ".id ASC"
			lastCond = TableDemo + ".`id` > ?"
		}
		return filter.Where(lastCond, args.LastId).Order(cursorOrder).Limit(limit)
	case offset > 100000: // 深分页: SELECT * FROM demo INNER JOIN (SELECT id FROM demo WHERE ... ORDER BY id LIMIT ?,?) AS tmp USING(id)
		subQuery := filter.Select("id").Offset(offset).Limit(limit)
		if order != "" {
			subQuery = subQuery.Order(order)
		}
		engine := c.WithContext(ctx).Model(c.model)
		if order != "" {
			engine = engine.Order(order)
		}
		return engine.Joins("INNER JOIN (?) AS tmp USING(id)", subQuery)
	default:
		if order != "" {
			filter = filter.Order(order)
		}
		return filter.Offset(offset).Limit(limit)
	}
}

func (c *customDemoDb) ListWithTotal(ctx context.Context, args *DemoSearch) ([]*Demo, int64, error) {
	var total int64
	if err := c.buildListFilter(ctx, args).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	_, limit := c.HandlePage(args.Page, args.PageSize)
	model := c.buildListQuery(ctx, args, limit)

	result := make([]*Demo, 0, limit)
	if err := model.Find(&result).Error; err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (c *customDemoDb) ListWithMore(ctx context.Context, args *DemoSearch) ([]*Demo, bool, error) {
	_, limit := c.HandlePage(args.Page, args.PageSize)
	want := limit
	limit++

	model := c.buildListQuery(ctx, args, limit)

	result := make([]*Demo, 0, limit)
	if err := model.Find(&result).Error; err != nil {
		return nil, false, err
	}

	ln := len(result)
	if ln == 0 {
		return nil, false, nil
	}
	var hasMore bool
	if ln > want {
		result = result[:want]
		hasMore = true
	}

	return result, hasMore, nil
}

func (c *customDemoDb) DemoUpdateTrans(ctx context.Context, id int64, upData *Demo) error {
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
		// demo 事务错误数据回滚
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

// TODO: add your code here and delete this line
