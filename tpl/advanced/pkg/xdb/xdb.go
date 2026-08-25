/**
 * @Author:
 * @Date: 2024-03-28 17:01
 * @Desc: 数据库操作
 */

package xdb

import (
	"fmt"
	"time"

	"advanced/pkg/xconfig"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
	"gorm.io/plugin/opentelemetry/tracing"
)

type Client = gorm.DB

func New(conf *xconfig.Conf) (*Client, error) {
	if len(conf.Mysql.Source) == 0 {
		return nil, fmt.Errorf("mysql 配置错误")
	}

	logLevel := gormLogger.Warn
	if conf.Mysql.LogLevel == "debug" {
		logLevel = gormLogger.Info
	}

	// xdbLogger 所有日志方法走 xlog,不需要 writer,传 nil 即可。
	newLogger := initLog(
		nil,
		gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond, // 慢 SQL 阈值
			IgnoreRecordNotFoundError: true,                   // 忽略 ErrRecordNotFound 记录到日志
			LogLevel:                  logLevel,               // Log level  gormlog.Silent
		},
	)

	masterConfig := mysql.New(mysql.Config{
		DSN:                       conf.Mysql.Source[0],
		DefaultStringSize:         1024, // string 类型字段的默认长度
		DisableDatetimePrecision:  true, // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true, // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true, // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: true, // 跳过版本自动探测(automatic server version initialization);配置固定,不依赖运行时 MySQL 版本自适应
	})

	var err error
	db, err := gorm.Open(masterConfig, &gorm.Config{
		Logger:                   newLogger,
		QueryFields:              true,
		DisableNestedTransaction: true, // 禁用嵌套事务
	})
	if err != nil {
		return nil, fmt.Errorf("初始化MySql数据库失败: %w", err)
	}
	connMaxLifetime := conf.Mysql.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 3600 // 默认 1 小时
	}

	if len(conf.Mysql.Source) > 1 {
		replicas := make([]gorm.Dialector, 0, len(conf.Mysql.Source)-1)
		for _, src := range conf.Mysql.Source[1:] {
			replicas = append(replicas, mysql.New(mysql.Config{
				DSN:                       src,
				DefaultStringSize:         1024,
				DisableDatetimePrecision:  true,
				DontSupportRenameIndex:    true,
				DontSupportRenameColumn:   true,
				SkipInitializeWithVersion: true,
			}))
		}
		// dbresolver 连接池设置作用于从库副本(Replicas)连接池;
		// 主库连接池由下方的 sqlDB 直接设置,两者作用域不同,分属不同连接池。
		err = db.Use(dbresolver.Register(dbresolver.Config{
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}).
			SetMaxIdleConns(conf.Mysql.MaxIdleConns).
			SetMaxOpenConns(conf.Mysql.MaxOpenConns).
			SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second).
			SetConnMaxIdleTime(time.Minute * 10))
		if err != nil {
			return nil, fmt.Errorf("初始化从数据库失败: %w", err)
		}
	}

	// 添加 OpenTelemetry 插件
	if err = db.Use(tracing.NewPlugin()); err != nil {
		return nil, fmt.Errorf("初始化MySql tracing插件失败: %w", err)
	}

	// 设置主库底层 sql.DB 连接池(若未启用 dbresolver,这也是唯一生效的连接池)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取sql.DB失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(conf.Mysql.MaxIdleConns) // 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxOpenConns(conf.Mysql.MaxOpenConns) // 设置打开连接池中连接的最大数量
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	// 限制空闲连接最大存活时间,避免长期空闲的连接被 MySQL wait_timeout 单方面关闭后,
	// 下次复用时拿到坏连接报 bad connection。
	sqlDB.SetConnMaxIdleTime(time.Minute * 10)

	return db, nil
}
