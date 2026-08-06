package db

import (
	"context"
	"fmt"
	"strings"

	"advanced/pkg/xdb"

	"gorm.io/gorm"
)

//go:generate mockgen -source=./mysql.go -destination=../../../test/mocks/repository/db/mysql.go  -package mock_repo_db

type ctxDbKeyType struct{}

const (
	defaultPageSize int64 = 20
)

var (
	ErrNotFound = gorm.ErrRecordNotFound
	ctxDbKey    = ctxDbKeyType{}
)

type (
	Conn struct {
		conn *xdb.Client
	}

	SearchPage struct {
		OrderField string `json:"orderField"` // 排序字段
		OrderType  int64  `json:"orderType"`  // 排序类型
		Page       int64  `json:"page"`       // 当前页
		PageSize   int64  `json:"pageSize"`   // 每页条数
		LastId     int64  `json:"lastId"`     // 上一页最大id
	}
)

func NewConn(cli *xdb.Client) *Conn {
	return &Conn{
		conn: cli,
	}
}

func (r *Conn) WithContext(ctx context.Context) *xdb.Client {
	v := ctx.Value(ctxDbKey)
	if v != nil {
		if tx, ok := v.(*xdb.Client); ok {
			return tx
		}
	}
	return r.conn.WithContext(ctx)
}

func (r *Conn) Tx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, ctxDbKey, tx)
		return fn(ctx)
	})
}

// 获取 排序字段(manager专用)
func (r *Conn) HandleRank(orderField string, orderType int64, defaultOrder string) (orderStr string) {
	orderStr = defaultOrder
	if orderField != "" &&
		!strings.Contains(orderField, ".``") &&
		strings.Index(orderField, ".") != len(orderField)-1 {
		orderStr = orderField
	}
	if orderType > 0 {
		orderStr = fmt.Sprintf(" %s  DESC", orderStr)
	} else {
		orderStr = fmt.Sprintf(" %s  ASC", orderStr)
	}
	return orderStr
}

func (r *Conn) HandlePage(page, pageSize int64) (offset, limit int) {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	limit = int(pageSize)
	offset = int((page - 1) * pageSize)
	if offset <= 0 {
		offset = 0
	}
	return
}
