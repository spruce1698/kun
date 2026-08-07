/**
 * @Author: spruce
 * @Date: 2024-04-23 17:13
 * @Desc: 根据数据库生成 repository/db
 */

package create

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/internal/command/create/kernel"
	"github.com/spruce1698/kun/pkg/output"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Database type
type DBType string

const (
	DefaultOutPath = "./internal/repository/db"
	VersionText    = "数据库生成GORM Repository文件"

	// dbMySQL Gorm Drivers mysql || postgres || clickhouse || sqlite
	dbMySQL      DBType = "mysql"
	dbPostgres   DBType = "postgres"
	dbClickHouse DBType = "clickhouse"
	dbSQLite     DBType = "sqlite"
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
	case dbSQLite:
		return gorm.Open(sqlite.Open(dsn))
	default:
		return nil, fmt.Errorf("unknow db %q (support mysql || postgres || sqlite || clickhouse for now)", t)
	}
}

func detectDBType(dsn string) DBType {
	dsnLower := strings.ToLower(dsn)
	if strings.Contains(dsnLower, "host=") || strings.Contains(dsnLower, "sslmode=") || strings.HasPrefix(dsnLower, "postgres://") || strings.HasPrefix(dsnLower, "postgresql://") {
		return dbPostgres
	}
	if strings.Contains(dsnLower, "clickhouse://") || (strings.Contains(dsnLower, "tcp://") && strings.Contains(dsnLower, "9000")) {
		return dbClickHouse
	}
	if strings.HasSuffix(dsnLower, ".db") || strings.HasSuffix(dsnLower, ".sqlite") || strings.HasSuffix(dsnLower, ".sqlite3") || strings.Contains(dsnLower, "file:") {
		return dbSQLite
	}
	return dbMySQL
}

func genDBRepo(cmd *cobra.Command, args []string) {
	// 如果参数是 .sql 文件，则解析 SQL 文件生成 repo
	if strings.HasSuffix(args[0], ".sql") {
		genDBRepoFromSQL(args)
		return
	}

	cmdConf := &CmdParams{
		DSN:     args[0],
		OutPath: DefaultOutPath,
	}
	cmdConf.DBType = string(detectDBType(cmdConf.DSN))
	if len(args) > 1 && args[1] != "" {
		if args[1] == "*" {
			cmdConf.Tables = []string{}
		} else {
			cmdConf.Tables = strings.Split(args[1], ",")
		}
	}

	gormDb, err := connectDB(DBType(cmdConf.DBType), cmdConf.DSN)
	if err != nil {
		output.Error("connect db server fail: %s", err)
		return
	}
	if gormDb == nil {
		output.Error("gorm db is nil")
		return
	}
	// 自定义命名策略
	gormDb.Config.NamingStrategy = schema.NamingStrategy{
		TablePrefix:   cmdConf.Prefix,
		SingularTable: true,
		NameReplacer:  nil,
	}
	conf := defaultSQLConfig()
	conf.DbConn = gormDb
	g := kernel.NewGenerator(conf)

	var tablesList []string
	if len(cmdConf.Tables) == 0 {
		// Execute tasks for all tables in the database
		tablesList, err = gormDb.Migrator().GetTables()
		if err != nil {
			output.Error("GORM migrator get all tables fail: %s", err)
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

// defaultSQLConfig 构建默认 SQLConfig（DB 和 SQL 文件路径共用）
func defaultSQLConfig() kernel.SQLConfig {
	outPath, err := filepath.Abs(DefaultOutPath)
	if err != nil {
		output.Error("outPath is invalid: %s, using default", err)
		outPath = DefaultOutPath
	}
	return kernel.SQLConfig{
		OutPath:           outPath,
		PackageName:       "db",
		FieldCoverable:    false, // 当字段具有默认值时生成指针，以解决无法分配零值的问题
		FieldNullable:     true,  // 当字段可为空时生成指针。注意：此默认值会让所有可空列生成 *T 指针类型，若希望生成值类型请改为 false
		FieldWithIndexTag: true,  // 生成字段包含 索引 标记
		FieldWithTypeTag:  true,  // 生成字段包含 列类型 标记
		FieldSignable:     false, // 检测整数字段的无符号类型，调整生成的数据类型
	}
}

// genDBRepoFromSQL 从 .sql 文件解析 CREATE TABLE 语句并生成 repo
func genDBRepoFromSQL(args []string) {
	conf := defaultSQLConfig()

	metas, err := kernel.ParseSQLFile(args[0], &conf)
	if err != nil {
		output.Error("parse sql file fail: %s", err)
		return
	}
	if len(metas) == 0 {
		output.Warn("no CREATE TABLE found in file: %s", args[0])
		return
	}

	var tableFilter []string
	if len(args) > 1 && args[1] != "" && args[1] != "*" && args[1] != "." {
		tableFilter = strings.Split(args[1], ",")
	}

	g := kernel.NewGenerator(conf)

	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if len(tableFilter) > 0 {
			found := false
			for _, t := range tableFilter {
				if strings.EqualFold(meta.TableName, t) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		g.AddRepoMeta(meta)
	}

	g.Execute()
}
