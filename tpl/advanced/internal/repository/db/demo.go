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
		// 空结果返回 (nil,0,nil) 而非 ErrNotFound;
		// ErrNotFound 仅用于单条 Find 找不到,避免"空列表"与"查询错误"语义混淆。
		return nil, 0, nil
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
	case args.LastId > 0 && args.Page > 1: // 游标分页
		// 游标按主键 id 比较,因此排序也必须按 id,否则(如按 name 排序而按 id 取游标)
		// 翻页结果会重复或漏行。需要按其它列做游标分页时,应改为 (orderCol, id) 复合游标。
		cursorOrder := TableDemo + ".id DESC"
		lastCond := "`id` < ?"
		if args.OrderType == 1 {
			cursorOrder = TableDemo + ".id ASC"
			lastCond = "`id` > ?"
		}
		model = filter().Where(lastCond, args.LastId).Order(cursorOrder).Limit(limit)
	case offset > 100000: // 深分页: SELECT * FROM demo INNER JOIN (SELECT id FROM demo WHERE ... ORDER BY id LIMIT ?,?) AS tmp USING(id)
		subQuery := filter().Order(order).Select("id").Offset(offset).Limit(limit)
		model = c.WithContext(ctx).Model(c.model).Order(order).Joins("INNER JOIN (?) AS tmp USING(id)", subQuery)
	default:
		model = filter().Order(order).Offset(offset).Limit(limit)
	}

	result := make([]*Demo, 0, limit)
	if err := model.Find(&result).Error; err != nil {
		return nil, 0, err
	}
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
		cursorOrder := TableDemo + ".id DESC"
		lastCond := "`id` < ?"
		if args.OrderType == 1 {
			cursorOrder = TableDemo + ".id ASC"
			lastCond = "`id` > ?"
		}
		model = filter().Where(lastCond, args.LastId).Order(cursorOrder).Limit(limit)
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
		// 空结果返回 nil error;ErrNotFound 仅用于单条 Find 找不到。
		return nil, false, nil
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

// TODO: add your code here and delete this line
