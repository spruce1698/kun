//go:build with_sqlite
// +build with_sqlite

// P1: sqlite 驱动依赖 CGO（mattn/go-sqlite3），默认不编译，避免破坏 CGO_ENABLED=0 静态构建。
// 启用方式: go build -tags with_sqlite ./...
// Makefile 示例: build-with-sqlite 目标。
package create

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	registerDriver(dbSQLite, func(dsn string) gorm.Dialector { return sqlite.Open(dsn) })
}
