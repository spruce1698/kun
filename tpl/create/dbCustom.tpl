package {{.PackageName}}

import (
	"context"
)

//go:generate mockgen -source=./{{.InterfaceName}}.go -destination=../../../test/mocks/repository/db/{{.InterfaceName}}.go  -package mock_repo_db -aux_files mysql=./{{.InterfaceName}}_gen.go

var _ {{.StructName}}Db = (*custom{{.StructName}}Db)(nil)

type (
	{{.StructName}}Db interface {
		{{.InterfaceName}}Db

        ListWithTotal(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, int64, error)
		ListWithMore(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, bool, error)
    	// TODO: add your code here and delete this line
	}

	custom{{.StructName}}Db struct {
		*default{{.StructName}}Db
	}

    {{.StructName}}Search struct {
		SearchPage
	}

	// TODO: add your code here and delete this line
)

func New{{.StructName}}Db(c *Conn) {{.StructName}}Db {
	return &custom{{.StructName}}Db{
		default{{.StructName}}Db: new{{.StructName}}Db(c),
	}
}


func (c *custom{{.StructName}}Db) ListWithTotal(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, int64, error) {
	model := c.WithContext(ctx).Model(c.model)



	var total int64
	err := model.WithContext(ctx).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, ErrNotFound
	}
	model = model.Order(
		c.HandleRank(
			args.OrderField,
			args.OrderType,
			"`"+Table{{.StructName}}+"`.`id`",
		),
	)
	offset, limit := c.HandlePage(args.Page, args.PageSize)
	//  time>lastTime or (time==lastTime and id>lastId)
	if args.LastId > 0 && args.Page > 1 {
		if args.OrderType == 0 {
			model = model.Where("`id` > ?", args.LastId).Limit(limit)
		} else {
			model = model.Where("`id` < ?", args.LastId).Limit(limit)
		}
	} else if offset > 100000 { // 考虑深分页 SELECT * FROM demo INNER JOIN (SELECT id FROM demo LIMIT 100000, 10) AS tmp USING(id);
        model = c.WithContext(ctx).Model(c.model).
      		Joins("INNER JOIN (?) AS tmp USING(id)", model.Select("id").Offset(offset).Limit(limit))
	} else {
		model = model.Offset(offset).Limit(limit)
	}

	result := make([]*{{.StructName}}, 0, limit)
	err = model.Find(&result).Error
    if err != nil {
		return nil, 0, err
	}
	if len(result) == 0 {
		return nil, 0, ErrNotFound
	}
	return result, total, nil
}

func (c *custom{{.StructName}}Db) ListWithMore(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, bool, error) {
	model := c.WithContext(ctx).Model(c.model)


	model = model.Order(
		c.HandleRank(
			args.OrderField,
			args.OrderType,
			"`"+Table{{.StructName}}+"`.`id`",
		),
	)
	offset, limit := c.HandlePage(args.Page, args.PageSize)
	// 在请求的数据基础上+1，以此来判断是否还有数据
	limit += 1

	if args.LastId > 0 && args.Page > 1 {
		if args.OrderType == 0 {
			model = model.Where("`id` > ?", args.LastId).Limit(limit)
		} else {
			model = model.Where("`id` < ?", args.LastId).Limit(limit)
		}
	} else if offset > 100000 { // 考虑深分页
    	model = c.WithContext(ctx).Model(c.model).
    		Joins("INNER JOIN (?) AS tmp USING(id)", model.Select("id").Offset(offset).Limit(limit))
	} else {
		model = model.Offset(offset).Limit(limit)
	}

	result := make([]*{{.StructName}}, 0, limit)
	err := model.Find(&result).Error
    if err != nil {
		return nil, false, err
	}

	ln := len(result)
	if ln == 0 {
		return nil, false, ErrNotFound
	}
	var hasMore bool
	if ln == limit {
		result = result[:ln-1]
		hasMore = true
	}

	return result, hasMore, nil
}

// TODO: add your code here and delete this line

