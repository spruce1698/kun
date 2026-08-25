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
	// DB/Redis 连接等启动期固化的资源不会重建,热重载仅对运行期通过 Get() 读取的逻辑生效。
	Conf struct {
		// Env 运行期可读,热重载即时生效(需通过 Get() 读取)。
		Env string
		// 改配置需重启
		Version string
		Server  ServerConf
		Token   TokenConf
		Log     LogConf
		Mysql   MysqlConf
		Redis   RedisConf
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
		Source       []string
		Password     string
		DB           int // Redis 数据库索引(单机模式有效)
		Cluster      bool
		PoolSize     int // 连接池大小
		MinIdleConns int // 最小空闲连接数
	}
	JaegerConf struct {
		Endpoint   string
		SampleRate float64 // 采样率 0.0~1.0,0 表示用默认 10%
	}
)

func New(path string) *Conf {
	confPath := os.Getenv("CONF")
	if confPath == "" {
		confPath = path
	}
	v := viper.New()
	// 启用环境变量必须在 ReadInConfig/Unmarshal 之前,否则首次 Unmarshal 时
	// 环境变量覆盖(如 SERVER_PORT)不会生效。
	v.AutomaticEnv()                                   // 自动识别环境变量
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 将 . 替换为 _   SERVER_PORT=10088 覆盖文件中 server下面port值

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

	// 热重载:配置文件变更时,Unmarshal 到全新 *Conf,原子替换 current 指针(供 Get())。
	// 字段本身不更新,运行期读取必须走 Get() 才能拿到新值。
	v.OnConfigChange(func(event fsnotify.Event) {
		nc := new(Conf)
		if e := v.Unmarshal(nc); e != nil {
			// 不能在 goroutine 中 panic,改为打印到 stderr(此时 xlog 可能尚未就绪)
			fmt.Fprintf(os.Stderr, "config hot reload unmarshal failed: %s \n", e)
			return
		}
		// current 自指:与初始快照一致,使任意 Get() 返回的快照再次调用 Get() 也能拿到自身,
		// 避免对热重载后的快照再 Get() 时拿到零值 nil。
		nc.current.Store(nc)
		c.current.Store(nc)
	})
	v.WatchConfig()

	return c
}

// Get 返回当前配置的稳定快照指针,热重载后返回新快照。
//
// ⚠️ 重要使用准则:
//  1. 运行期重复读取配置的逻辑必须通过此方法(如 conf.Get().Env),而非直接访问字段。
//  2. 始终通过依赖注入持有的根 *Conf 实例调用 Get(),严禁长期缓存 Get() 返回的快照指针,
//     否则后续配置再次热重载时将无法感知最新变更。
//  3. 结构体内嵌 atomic.Pointer 不可值复制(禁止 copy value / 传值),全程须以指针 (*Conf) 传递。
//
// 内部为 atomic.Pointer.Load,零分配、零竞态。
func (c *Conf) Get() *Conf {
	return c.current.Load()
}
