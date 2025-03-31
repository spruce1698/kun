/**
 * @Author: spruce
 * @Date: 2024-04-23 17:13
 * @Desc: 根据数据库生成 repository/db
 */

package create

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/internal/command/create/kernel"
	"github.com/spruce1698/kun/pkg/fmt"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database type
type DBType string

const (
	DefaultOutPath = "./internal/repository/db"
	VersionText    = "数据库生成GORM Repository文件"

	// dbMySQL Gorm Drivers mysql || postgres || clickhouse
	dbMySQL      DBType = "mysql"
	dbPostgres   DBType = "postgres"
	dbClickHouse DBType = "clickhouse"
)

// CmdParams is command line parameters
type CmdParams struct {
	DSN     string   // user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	Tables  []string // 输入所需的数据表或将其留空,留空数据库中所有的数据表
	OutPath string   // 指定输出目录
	Prefix  string   // 表前缀,不为空则model不包含前缀
	DBType  string   // 数据库类型
}

// connectDB 连接数据库 选择用于连接到数据库的数据库类型
func connectDB(t DBType, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("dsn cannot be empty")
	}
	switch t {
	case dbMySQL:
		return gorm.Open(mysql.Open(dsn))
	case dbPostgres:
		return gorm.Open(postgres.Open(dsn))
	case dbClickHouse:
		return gorm.Open(clickhouse.Open(dsn))
	default:
		return nil, fmt.Errorf("unknow db %q (support mysql || postgres || sqlite || clickhouse for now)", t)
	}
}

func genDBRepo(cmd *cobra.Command, args []string) {
	cmdConf := &CmdParams{
		DSN:     args[0],
		DBType:  "mysql",
		OutPath: DefaultOutPath,
	}
	if args[1] != "" {
		if args[1] == "*" {
			cmdConf.Tables = []string{}
		} else {
			cmdConf.Tables = strings.Split(args[1], ",")
		}
	}

	outPath, err := filepath.Abs(cmdConf.OutPath)
	if err != nil {
		fmt.Error("outPath is invalid: %s", err)
		return
	}

	gormDb, err := connectDB(DBType(cmdConf.DBType), cmdConf.DSN)
	if err != nil {
		fmt.Error("connect db server fail: %s", err)
		return
	}
	if gormDb == nil {
		fmt.Error("gorm db is nil")
		return
	}

	g := kernel.NewGenerator(kernel.SQLConfig{
		DbConn:            gormDb,
		OutPath:           outPath, // 指定输出目录
		PackageName:       "db",    // Repo代码的包名称,同数据库类型相同。
		FieldCoverable:    false,   // 当字段具有默认值时生成指针，以解决无法分配零值的问题
		FieldNullable:     true,    // 当字段可为空时生成指针
		FieldWithIndexTag: true,    // 生成字段包含 索引 标记
		FieldWithTypeTag:  true,    // 生成字段包含 列类型 标记
		FieldSignable:     false,   // 检测整数字段的无符号类型，调整生成的数据类型
	})

	var tablesList []string
	if len(cmdConf.Tables) == 0 {
		// Execute tasks for all tables in the database
		tablesList, err = gormDb.Migrator().GetTables()
		if err != nil {
			fmt.Error("GORM migrator get all tables fail: %s", err)
			return
		}
	} else {
		tablesList = cmdConf.Tables
	}
	for _, tableName := range tablesList {
		g.GenerateRepo(tableName)
	}

	g.Execute()

}
