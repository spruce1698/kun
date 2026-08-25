package db

import (
	"context"
	"regexp"
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
	// reIdentifier 校验排序列名/表名单段(仅字母/数字/下划线),防 ORDER BY 注入。
	reIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type (
	Conn struct {
		conn *xdb.Client
	}

	SearchPage struct {
		OrderField string `json:"orderField"` // 排序字段
		OrderType  int64  `json:"orderType"`  // 排序类型0:降序(默认),1:升序
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
			// 必须再绑一次 ctx:事务里的 *gorm.DB 携带的是 Tx 开启时的 context,
			// 直接复用会让事务内所有 SQL 不受当前 ctx 的取消/超时约束
			// (客户端断连或 gin 超时后,UPDATE/DELETE 仍会跑完),trace 也会断链。
			return tx.WithContext(ctx)
		}
	}
	return r.conn.WithContext(ctx)
}

func (r *Conn) Tx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 用新变量而不是对捕获的 ctx 形参赋值:GORM 在重试/嵌套场景下可能多次调用
		// 本闭包,复用被改写过的 ctx 会把上一次(已提交或已回滚)的 tx 带进来。
		txCtx := context.WithValue(ctx, ctxDbKey, tx)
		return fn(txCtx)
	})
}

// HandleRank 构造排序子句字符串,如 "demo.id DESC"。(manager专用)
//
// 安全性:
//   - orderField 来自用户请求,支持裸列名 name 或跨表 table.name 两段。
//     列名必须命中字段白名单 fields(逗号分隔的真实列名,如 DemoFields),否则回退 defaultCol;
//     若带 table 前缀,表名须经 reIdentifier 校验(仅字母/数字/下划线)。双重校验杜绝注入。
//   - 裸列名命中时会补上 defaultCol 的表前缀(如 demo.),避免深分页 JOIN 时列名歧义。
//   - orderType=0:降序(默认),1:升序。
func (r *Conn) HandleRank(orderField string, orderType int64, fields string, defaultCol string) string {
	col := defaultCol
	if orderField != "" {
		table := ""
		name := orderField
		if i := strings.IndexAny(orderField, "."); i >= 0 {
			table, name = orderField[:i], orderField[i+1:]
		}
		// 列名须在白名单;带表名前缀时表名须为合法标识符
		if reIdentifier.MatchString(name) && strings.Contains(","+fields+",", ","+name+",") {
			if table != "" {
				if !reIdentifier.MatchString(table) {
					col = defaultCol
				} else {
					// 跨表排序:保留用户指定表前缀,如 "order_item" + "." + "id"
					col = table + "." + name
				}
			} else if i := strings.IndexByte(defaultCol, '.'); i >= 0 {
				col = defaultCol[:i+1] + name
			} else {
				col = name
			}
		}
	}
	if orderType == 1 {
		return col + " ASC"
	}
	return col + " DESC"
}

func (r *Conn) HandlePage(page, pageSize int64) (offset, limit int) {
	if page <= 0 {
		page = 1
	}
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
