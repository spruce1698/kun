package xconfig

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	EnvDebug   = "debug"
	EnvTest    = "test"
	EnvStaging = "staging"
	EnvRelease = "release"
)

type (
	// Conf 是配置容器。
	//
	// 读取方式分两类:
	//  1. 启动期一次性读取(如 xdb.New 读 Mysql.Source 建连接):直接访问字段 conf.Mysql.Source。
	//     New 返回时字段即首份配置,且启动期之后字段不再变化,无竞态。
	//  2. 运行期重复读取(如每条消息读 Env):用 conf.Get().Xxx,返回原子快照,
	//     零竞态,能拿到热重载后的最新值。不要长期缓存 Get() 返回值。
	//
	// 热重载:配置文件变更时,Unmarshal 出新 *Conf 并原子替换 current 指针(供 Get())。
	// 注意:字段本身不更新,故运行期读取必须走 Get(),否则拿不到热重载后的值。
	// DB/Redis/Kafka 连接等启动期固化的资源不会重建,热重载仅对运行期通过 Get() 读取的逻辑生效。
	Conf struct {
		// Env 运行期可读(如消费日志),热重载即时生效(需通过 Get() 读取)。
		Env string
		// 改配置需重启
		Version string
		Server  ServerConf
		Broker  BrokerConf
		Token   TokenConf
		Log     LogConf
		Mysql   MysqlConf
		Redis   RedisConf
		Kafka   KafkaConf
		Jaeger  JaegerConf

		// current 原子持有"当前配置快照"指针,供运行期 Get() 读取,零竞态。
		current atomic.Pointer[Conf]
	}

	ServerConf struct {
		Name         string
		Port         int64
		ReadTimeout  int64
		WriteTimeout int64
		AllowOrigins []string // CORS 允许的来源白名单,为空则允许所有
	}
	BrokerConf struct {
		Name         string
		Port         int64
		ReadTimeout  int64
		WriteTimeout int64
	}

	TokenConf struct {
		Secret        string
		RefreshSecret string
		CSRFKey       string
		Expire        int64
		Refresh       int64

		QueryKey   string
		CacheKey   string
		MultiLogin bool
	}

	LogConf struct {
		Level        string
		FilePath     string
		Stdout       bool
		KafkaTopic   string
		KafkaBrokers []string
	}

	MysqlConf struct {
		LogLevel        string
		Source          []string
		MaxIdleConns    int // 空闲连接池中连接的最大数量
		MaxOpenConns    int // 打开数据库连接的最大数量
		ConnMaxLifetime int // 连接可复用的最大时间(秒)
	}
	RedisConf struct {
		Source   []string
		Password string
		Cluster  bool
		PoolSize int // 连接池大小
	}
	KafkaConf struct {
		Brokers []string
	}
	JaegerConf struct {
		Endpoint string
	}
)

func New(path string) *Conf {
	confPath := os.Getenv("CONF")
	if confPath == "" {
		confPath = path
	}
	v := viper.New()
	// 设置配置文件信息
	v.SetConfigFile(confPath) // 设置文件的类型

	// 读取配置文件
	if e := v.ReadInConfig(); e != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", e))
	}
	c := new(Conf)
	if e := v.Unmarshal(c); e != nil {
		panic(fmt.Errorf("Fatal error config to Conf: %s \n", e))
	}
	// current 初始指向自身,使 Get() 在热重载前后都能返回最新快照
	c.current.Store(c)

	// 启用环境变量
	v.AutomaticEnv()                                   // 自动识别环境变量
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 将 . 替换为 _   SERVER_PORT=10088 覆盖文件中 server下面port值

	// 热重载:配置文件变更时,Unmarshal 到全新 *Conf,原子替换 current 指针(供 Get())。
	// 字段本身不更新,运行期读取必须走 Get() 才能拿到新值。
	v.OnConfigChange(func(event fsnotify.Event) {
		nc := new(Conf)
		if e := v.Unmarshal(nc); e != nil {
			// 不能在 goroutine 中 panic,改为打印错误
			fmt.Printf("config hot reload unmarshal failed: %s \n", e)
			return
		}
		nc.current.Store(nc)
		c.current.Store(nc)
	})
	go v.WatchConfig()

	return c
}

// Get 返回当前配置的稳定快照指针,热重载后返回新快照。
// 运行期重复读取配置的逻辑应通过此方法,而非直接访问字段。
// 内部为 atomic.Pointer.Load,零分配、零竞态。不要长期缓存返回值。
func (c *Conf) Get() *Conf {
	return c.current.Load()
}
