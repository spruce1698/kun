package {{.PackageName}}

import (
	"context"

	"gorm.io/gorm"
)

//go:generate mockgen -source=./{{.InterfaceName}}.go -destination=../../../test/mocks/repository/db/{{.InterfaceName}}.go -package mock_repo_db -aux_files {{.PackageName}}=./{{.InterfaceName}}_gen.go

var _ {{.StructName}}Db = (*custom{{.StructName}}Db)(nil)

type (
	{{.StructName}}Db interface {
		{{.InterfaceName}}Db

        ListWithTotal(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, int64, error)
		ListWithMore(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, bool, error)

    	// TODO: add custom functions here and delete this line
	}

	custom{{.StructName}}Db struct {
		*default{{.StructName}}Db
	}

    {{.StructName}}Search struct {
		SearchPage
	}

	// TODO: add struct here and delete this line
)

func New{{.StructName}}Db(c *Conn) {{.StructName}}Db {
	return &custom{{.StructName}}Db{
		default{{.StructName}}Db: new{{.StructName}}Db(c),
	}
}


func (c *custom{{.StructName}}Db) ListWithTotal(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, int64, error) {
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
		return nil, 0, nil
	}

	order := c.HandleRank(
		args.OrderField,
		args.OrderType,
		{{.StructName}}Fields,
		{{if .HasPrimaryKey}}Table{{.StructName}}+".{{.PrimaryKeyColumn}}"{{else}}""{{end}},
	)

	offset, limit := c.HandlePage(args.Page, args.PageSize)

	var model *gorm.DB
	{{if .HasPrimaryKey -}}
	switch {
	case args.LastId > 0 && args.Page > 1: // 游标分页: time>lastTime or (time==lastTime and id>lastId)
		lastCond := "`{{.PrimaryKeyColumn}}` < ?"
		if args.OrderType == 1 {
			lastCond = "`{{.PrimaryKeyColumn}}` > ?"
		}
		model = filter().Where(lastCond, args.LastId).Order(order).Limit(limit)
	case offset > 100000: // 深分页: SELECT * FROM {{.TableName}} INNER JOIN (SELECT id FROM {{.TableName}} WHERE ... ORDER BY id LIMIT ?,?) AS tmp USING({{.PrimaryKeyColumn}})
		subQuery := filter().Order(order).Select("{{.PrimaryKeyColumn}}").Offset(offset).Limit(limit)
		model = c.WithContext(ctx).Model(c.model).Order(order).Joins("INNER JOIN (?) AS tmp USING({{.PrimaryKeyColumn}})", subQuery)
	default:
		model = filter().Order(order).Offset(offset).Limit(limit)
	}
	{{- else -}}
	model = filter().Order(order).Offset(offset).Limit(limit)
	{{- end}}

	result := make([]*{{.StructName}}, 0, limit)
	if err := model.Find(&result).Error; err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (c *custom{{.StructName}}Db) ListWithMore(ctx context.Context, args *{{.StructName}}Search) ([]*{{.StructName}}, bool, error) {
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
		{{.StructName}}Fields,
		{{if .HasPrimaryKey}}Table{{.StructName}}+".{{.PrimaryKeyColumn}}"{{else}}""{{end}},
	)

	offset, limit := c.HandlePage(args.Page, args.PageSize)
	// 在请求的数据基础上+1，以此来判断是否还有数据
	want := limit
	limit++

	var model *gorm.DB
	{{if .HasPrimaryKey -}}
	switch {
	case args.LastId > 0 && args.Page > 1: // 游标分页
		lastCond := "`{{.PrimaryKeyColumn}}` < ?"
		if args.OrderType == 1 {
			lastCond = "`{{.PrimaryKeyColumn}}` > ?"
		}
		model = filter().Where(lastCond, args.LastId).Order(order).Limit(limit)
	case offset > 100000: // 深分页
		subQuery := filter().Order(order).Select("{{.PrimaryKeyColumn}}").Offset(offset).Limit(limit)
		model = c.WithContext(ctx).Model(c.model).Order(order).Joins("INNER JOIN (?) AS tmp USING({{.PrimaryKeyColumn}})", subQuery)
	default:
		model = filter().Order(order).Offset(offset).Limit(limit)
	}
	{{- else -}}
	model = filter().Order(order).Offset(offset).Limit(limit)
	{{- end}}

	result := make([]*{{.StructName}}, 0, limit)
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

// TODO: add your code here and delete this line

