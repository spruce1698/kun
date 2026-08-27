//go:build with_clickhouse
// +build with_clickhouse

// P1: clickhouse 驱动体积大（引入整个 ClickHouse SDK），默认不编译。
// 启用方式: go build -tags with_clickhouse ./...
package create

import (
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

func init() {
	registerDriver(dbClickHouse, func(dsn string) gorm.Dialector { return clickhouse.Open(dsn) })
}
