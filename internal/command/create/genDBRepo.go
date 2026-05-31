/**
 * @Author: spruce
 * @Date: 2024-04-23 17:13
 * @Desc: 根据数据库生成 repository/db
 */

package create

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/internal/command/create/kernel"
	"github.com/spruce1698/kun/pkg/fmt"
	"github.com/xwb1989/sqlparser"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
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
	SQLFile string   // .sql file path
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
		DBType:  "mysql",
		OutPath: DefaultOutPath,
	}

	if len(args) == 1 {
		// single argument, assume it's a sql file
		if strings.HasSuffix(strings.ToLower(args[0]), ".sql") {
			cmdConf.SQLFile = args[0]
			genFromSQLFile(cmdConf)
			return
		}
	}

	if len(args) == 2 {
		cmdConf.DSN = args[0]
		if args[1] == "*" {
			cmdConf.Tables = []string{}
		} else {
			cmdConf.Tables = strings.Split(args[1], ",")
		}
	} else if len(args) == 1 && !strings.HasSuffix(strings.ToLower(args[0]), ".sql") {
		// for backward compatibility, if single arg is not a sql file, treat as DSN and get all tables
		cmdConf.DSN = args[0]
		cmdConf.Tables = []string{}
	} else if len(args) == 0 {
		fmt.Error("missing arguments")
		return
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
	// 自定义命名策略
	gormDb.Config.NamingStrategy = schema.NamingStrategy{
		TablePrefix:   cmdConf.Prefix, // 表名前缀，
		SingularTable: true,           // 使用单数表名，例如 "user" 而不是 "users"
		NameReplacer:  nil,            // 可选：替换名称中的特定字符
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

func genFromSQLFile(cmdConf *CmdParams) {
	f, err := os.Open(cmdConf.SQLFile)
	if err != nil {
		fmt.Error("open sql file error: %s", err)
		return
	}
	defer f.Close()

	outPath, _ := filepath.Abs(cmdConf.OutPath)
	g := kernel.NewGenerator(kernel.SQLConfig{
		OutPath:     outPath,
		PackageName: "db",
	})

	tokenizer := sqlparser.NewTokenizer(f)
	for {
		stmt, err := sqlparser.ParseNext(tokenizer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Error("sql parser error: %s", err)
			continue
		}

		ddl, ok := stmt.(*sqlparser.DDL)
		if !ok || ddl.Action != "create" {
			continue
		}

		tableName := ddl.NewName.Name.String()
		structName := strings.ReplaceAll(strings.Title(strings.ReplaceAll(tableName, "_", " ")), " ", "")
		fileName := string(unicode.ToLower(rune(structName[0]))) + structName[1:]

		var fields []*kernel.Field
		var primaryKeyType string
		for _, col := range ddl.TableSpec.Columns {
			isPrimaryKey := false
			if col.Type.KeyOpt == 1 { // 1 = primary key
				isPrimaryKey = true
				primaryKeyType = toGoType(col.Type.Type)
			}

			var comment string
			if col.Type.Comment != nil && len(col.Type.Comment.Val) > 0 {
				comment = fmt.Sprintf("// %s", col.Type.Comment.Val)
			}

			colName := col.Name.String()
			// Convert snake_case to CamelCase for the Go struct field name.
			fieldName := strings.ReplaceAll(strings.Title(strings.ReplaceAll(colName, "_", " ")), " ", "")

			fields = append(fields, &kernel.Field{
				Name:         fieldName,
				Type:         toGoType(col.Type.Type),
				GORMTag:      buildGormTag(col),
				JSONTag:      colName, // Keep original column name for json tag
				CommentTag:   comment,
				IsPrimaryKey: isPrimaryKey,
			})
		}
		if primaryKeyType == "" {
			primaryKeyType = "int64" // default
		}

		meta := &kernel.StructMeta{
			FileName:       fileName,
			InterfaceName:  fileName,
			StructName:     structName,
			TableName:      tableName,
			PackageName:    "db",
			PrimaryKeyType: primaryKeyType,
			Fields:         fields,
		}
		g.Repos()[structName] = meta
	}

	g.Execute()
}

func toGoType(sqlType string) string {
	switch strings.ToLower(sqlType) {
	case "int", "integer", "mediumint", "smallint":
		return "int"
	case "bigint":
		return "int64"
	case "varchar", "char", "text", "mediumtext", "longtext", "json":
		return "string"
	case "tinyint":
		return "int8"
	case "float", "double", "decimal":
		return "float64"
	case "date", "datetime", "timestamp":
		return "time.Time"
	case "blob", "binary", "varbinary":
		return "[]byte"
	case "boolean", "bool":
		return "bool"
	default:
		return "string"
	}
}

func buildGormTag(col *sqlparser.ColumnDefinition) string {
	var tags []string
	tags = append(tags, "column:"+col.Name.String())
	tags = append(tags, "type:"+col.Type.Type)
	if col.Type.NotNull {
		tags = append(tags, "not null")
	}
	if col.Type.KeyOpt == 1 { // 1 = primary key
		tags = append(tags, "primaryKey")
	}
	return strings.Join(tags, ";")
}
