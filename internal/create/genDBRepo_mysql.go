// P1: mysql 和 postgres 驱动默认编译，无 CGO 依赖，支持纯静态构建（CGO_ENABLED=0）。
package create

import (
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	registerDriver(dbMySQL, func(dsn string) gorm.Dialector { return mysql.Open(dsn) })
	registerDriver(dbPostgres, func(dsn string) gorm.Dialector { return postgres.Open(dsn) })
}
