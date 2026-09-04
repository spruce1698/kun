# advanced — 高级布局

## 技术栈

| 类别     | 库                                                     | 说明              |
| -------- | ------------------------------------------------------ | ----------------- |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin)                 | HTTP 路由与中间件 |
| ORM      | [Gorm](https://github.com/go-gorm/gorm)                 | 数据库操作        |
| 依赖注入 | [Wire](https://github.com/google/wire)                  | 编译期依赖注入    |
| 配置管理 | [Viper](https://github.com/spf13/viper)                 | 多环境配置        |
| 日志     | [Zap](https://github.com/uber-go/zap)                   | 高性能结构化日志  |
| JWT      | [Golang-jwt](https://github.com/golang-jwt/jwt)         | Token 鉴权        |
| 缓存     | [Go-redis](https://github.com/go-redis/redis)           | Redis 客户端      |
| 消息队列 | [Asynq](https://github.com/hibiken/asynq) / Kafka       | 异步任务与消息    |
| 校验     | [Validator](https://github.com/go-playground/validator) | 参数校验          |
| 文档     | [Swaggo](https://github.com/swaggo/swag)                | 接口文档生成      |

## 分层架构说明

```
┌─────────────────────────────────────────────┐
│               router / handler              │  ── 路由 & 处理器（处理 HTTP 请求）
├─────────────────────────────────────────────┤
│                 service / svc               │  ── 业务逻辑层
├─────────────────────────────────────────────┤
│      repository/db  |   repository/cache    │  ── 数据访问层（DB + 缓存）
├─────────────────────────────────────────────┤
│                    pkg / x*                 │  ── 基础设施与公共工具
└─────────────────────────────────────────────┘
```

- **handler**: 接收 HTTP 请求，参数校验，调用 service，返回响应。
- **service**: 核心业务逻辑编排，事务管理，调用 repository。
- **repository**: 数据存储，屏蔽 DB / 缓存实现细节。
- **pkg**: 跨项目的公共组件（日志、配置、加密、消息队列等）。

## 目录结构

```
.
├── cmd/                           项目入口目录
│   ├── broker/                    消息服务器入口（异步消息、延时任务、周期性任务）
│   │   ├── main.go                Wire DI 启动入口，信号优雅关闭
│   │   └── wire/                  Wire 依赖注入与装配
│   │       ├── app.go             Broker 应用装配（健康探针与消费者启动）
│   │       ├── wire.go            声明 Broker 全链路依赖
│   │       └── wire_gen.go        Wire 自动生成代码
│   └── server/                    HTTP 服务入口
│       ├── main.go                Wire DI 启动入口，信号优雅关闭
│       └── wire/                  Wire 依赖注入与装配
│           ├── app.go             HTTP 应用装配（Gin 模式、中间件、路由与资源回收）
│           ├── wire.go            声明 Server 全栈分层依赖
│           └── wire_gen.go        Wire 自动生成代码
├── config/                        配置文件目录
│   ├── local.yml                  本地环境配置（YAML 格式）
│   └── release.yml                生产环境配置
├── deploy/                        部署与容器化编排
│   ├── .env.example               环境变量示例
│   ├── Dockerfile                 多阶段构建镜像 Dockerfile
│   └── docker-compose.yml         本地中间件一键拉起编排
├── docs/                          项目文档
│   └── sql/                       数据库建表脚本
│       └── advanced.sql           数据库初始化 SQL
├── swagger/                       Swagger 接口文档            
├── internal/                      内部业务逻辑
│   ├── handler/                   HTTP 处理器层（参数校验、调用 Service）
│   │   ├── demo/demo.go           Demo CRUD 处理器（含缓存、事件发布）
│   │   └── serverDI.go            Wire DI 容器，注册所有处理器
│   ├── event/                     事件发布/订阅抽象层
│   │   ├── publisher.go           多后端发布器（Kafka 同步 + Asynq 异步/延时/定时）
│   │   ├── subscriber.go          订阅管理器（Kafka 消费组 + Asynq Task Worker）
│   │   └── types.go               订阅任务定义（Task / Kafka / Asynq）
│   ├── global/                    全局常量与枚举
│   │   ├── constants.go           上下文 Key 常量与 API 路由前缀常量
│   │   ├── event.go               事件类型/主题/消费组枚举及消息结构
│   │   └── pay.go                 支付状态与支付方式枚举
│   ├── middleware/                HTTP 中间件
│   │   ├── auth.go                JWT 鉴权（强制/可选/写入三种模式）
│   │   ├── cors.go                跨域资源共享
│   │   ├── csrf.go                CSRF 防护（gorilla/csrf）
│   │   ├── metrics.go             Prometheus 指标统计中间件
│   │   ├── ratelimit.go           单机/分布式限流器
│   │   └── recovery.go            Panic 恢复（区分 broken pipe 与普通 panic）
│   ├── repository/                数据访问层
│   │   ├── cache/                 缓存层
│   │   │   ├── demo.go            二级缓存（本地 + Redis）实现
│   │   │   ├── keys.go            缓存 Key 模板统一管理
│   │   │   └── local.go           线程安全本地内存缓存（go-cache 封装）
│   │   ├── db/                    数据库层
│   │   │   ├── demo.go            Demo 自定义查询（分页/游标分页/事务更新）
│   │   │   ├── demo_gen.go        SqlGen 自动生成基础 CRUD
│   │   │   └── mysql.go           GORM 连接池/事务/分页/排序封装
│   │   └── serverDI.go            Wire DI 注册所有 Repository 实现
│   ├── router/                    路由注册层
│   │   ├── router.go              全局路由工具（404 处理/健康检查/Swagger/Metrics）
│   │   ├── serverDI.go            Wire DI 注册所有路由定义
│   │   └── v0/demo.go             v0 版本 Demo RESTful 路由组
│   └── service/                   业务逻辑层
│       ├── svc/                   核心业务逻辑
│       │   ├── context.go         业务服务基础上下文（配置/DB/Redis）
│       │   ├── demo.go            Demo 业务编排（CRUD + 缓存 + 事件发布）
│       │   └── broker.go          事件消费处理器（Kafka + Asynq 任务路由）
│       ├── brokerDI.go            Wire DI 注册 Broker 事件订阅任务
│       └── serverDI.go            Wire DI 注册 HTTP 服务层依赖
├── pkg/                           公共工具包
│   ├── asynq/                     Asynq 异步/延时/定时任务队列封装
│   ├── encrypt/                   加解密工具（MD5/AES/RSA/Bcrypt）
│   │   ├── bcrypt.go              Bcrypt 密码加密与校验
│   │   ├── encrypt.go             对称加解密与哈希工具
│   │   └── rsa.go                 RSA 完整套件（OAEP 加密/解密/签名/验签）
│   ├── kafka/                     Kafka 客户端封装
│   │   ├── client.go              主题管理客户端（列举/创建/删除）
│   │   ├── publisher.go           消息发布器（压缩/超时/批量异步）
│   │   ├── subscriber.go          消费者（消费组/偏移量提交/自动重试）
│   │   └── tracing.go             Kafka 跨进程链路追踪
│   ├── token/                     JWT Token 管理
│   │   └── jwt.go                 令牌签发/校验/刷新/黑名单撤销
│   ├── utils/                     通用工具函数
│   │   ├── convert.go             进制转换（Base62/Base58）与货币单位转换（元/分）
│   │   ├── file.go                文件系统操作
│   │   ├── func.go                字符串工具（中文检测/脱敏/排序）
│   │   ├── net.go                 网络工具（端口/IP 检测）
│   │   ├── pool.go                Goroutine 工作池
│   │   ├── random.go              随机字符、数字及 UUID 生成
│   │   ├── snowflake.go           Snowflake 分布式 ID 生成器
│   │   ├── time.go                时间计算与格式化
│   │   └── verify.go              正则与格式校验
│   ├── validator/                 参数校验器（中英文翻译/自定义规则）
│   ├── xconfig/                   配置加载器（Viper/YAML/多环境）
│   ├── xdb/                       数据库集成（GORM/连接池/日志追踪）
│   ├── xerror/                    统一错误码与错误模型（xerror.go）
│   ├── xhttp/                     统一 HTTP 响应输出与通用分页请求（http.go）
│   ├── xlog/                      日志与链路追踪（Zap + OpenTelemetry）
│   ├── xredis/                    Redis 客户端（连接池/健康检查/追踪）
│   └── xserver/                   服务抽象与生命周期管理
│       ├── broker/                Broker 守护进程（Supervisor/子进程保活/探针）
│       ├── http/                  HTTP Server（生命周期/优雅退出）
│       └── xserver.go             通用 Server 抽象与资源清理器（Closer）
├── test/                          单元测试与基准测试
├── Makefile                       构建与命令快捷入口
└── README.md                      项目说明
```

## 要求

您需要在系统上安装以下软件：

* Golang 1.25.11或更高版本
* Git
* wire

### 创建组件

您可以使用以下命令为项目创建handler、service和repository等组件：

### 创建组件

```bash
# 创建处理器
kun create hdl user

# 创建业务服务
kun create svc user

# 创建数据仓库（支持自动建表）
kun create repo "name:pwd@tcp(127.0.0.1:3306)/dbname" [t1,t2|t1|*]

# 创建数据库脚本
kun create db "*.sql" "[t1,t2|t1|*]"

# 创建缓存
kun create cache cache

# 同时创建处理器 + 服务
kun create hs user
```

## 常用命令

```bash
# 启动 HTTP 服务
kun run

# 启动 Broker 调度服务
kun run ./cmd/broker

# 编译 Wire 依赖注入
kun wire
```

## 事件发布与订阅开发指南

Advanced Layout 提供了由 `internal/event` 封装的多后端消息发布与订阅架构（Kafka + Asynq）：

### 核心角色与概念

- **调度服务器 (Broker)**：作为事件中转调度枢纽，管理事件和监听器之间的映射关系，负责触发与分发任务（支持异步消息、延时消息、周期性 Cron 任务、Job 消费处理等）。
- **发布者 (Publisher / Producer)**：发送数据的实体，负责向指定的主题（Topic）投递消息。
- **订阅者 (Subscriber / Consumer)**：接收并处理数据的实体，订阅特定主题并在收到新事件时触发对应 Handler 执行业务逻辑。

### 1. 消息发布 (Publisher)

在 Service 层注入 `*event.Publisher`，即可进行多场景消息投递：

```go
// 异步投递普通任务
entryID, err := s.eventPub.AsynqSync(ctx, "default", "task:demo", payload)

// 投递延时任务 (如 30 分钟后超时取消未支付订单)
entryID, err := s.eventPub.AsynqDelay(ctx, "default", "order:timeout", payload, 30*time.Minute)

// 发布 Kafka 业务事件 (带业务 Key)
err := s.eventPub.KafkaWithKey(ctx, "topic_user_event", "user_1001", []byte("registered"))
```

### 2. 消息消费 (Subscriber)

在 `internal/service/svc/broker.go` 中编写消费者处理器：

```go
// Kafka 消息处理
func (s *Service) DemoKafkaHandler(ctx context.Context, key, value string) error {
    // 处理业务逻辑
    return nil
}

// Asynq 任务处理
func (s *Service) DemoAsynqHandler(ctx context.Context, task *asynq.Task) error {
    payload := task.Payload()
    // 处理业务逻辑
    return nil
}
```

并在 `internal/service/brokerDI.go` 中注册到 `WireBrokerSet`，即可随 `broker` 进程启动自动消费。

## 项目约定

1. **依赖注入**: 所有层之间通过 `Wire` 进行编译期依赖注入，避免手动管理依赖。
2. **配置管理**: 使用 `Viper` 加载 `config/` 下的配置文件，支持多环境。
3. **错误处理**: 统一使用 `xerror` 包管理错误码，返回结构化错误响应。
4. **日志**: 使用 `xlog`（基于 Zap）记录结构化日志，支持链路追踪。
5. **缓存 Key**: 统一在 `repository/cache/keys.go` 中管理，避免散落各处。

```go
// ==============github.com/robfig/cron 周期性任务=====================================

// "github.com/robfig/cron/v3"
//  CronFun func(corner *cron.Cron)

// // 周期性任务服务精确到秒
// func (s *Sub) Cron() {
// 	if s.task.CronFunSet == nil || len(s.task.CronFunSet) <= 0 {
// 		return
// 	}
//
// 	loc, _ := time.LoadLocation("Asia/Shanghai")
// 	cs := cron.New(cron.WithSeconds(), cron.WithLocation(loc))
// 	s.wg.Add(1)
// 	go func() {
// 		// 添加任务
// 		for _, fun := range s.task.CronFunSet {
// 			fun(cs)
// 		}
// 		cs.Start()
//
// 		defer s.wg.Done()
// 	}()
// 	defer cs.Stop() // 需要在协程结束时关闭
// }

// // 支付事件消费
// func (s *Service) PayDemo(key, value string) error {
// 	switch enum.MsgType(key) {
// 	case enum.MsgTypePayRecharge: // 充值
//
// 		msg := &producer.PayRecharge{}
// 		_ = json.Unmarshal([]byte(value), msg)
//
// 		fmt.Printf("KafkaSubscriber-Fetch:消费时间:%s:%s \n", utils.GetTimeStr(time.Now()), value)
// 		// 逻辑处理start...
// 		// TODO: do something
// 		// err := c.DemoCache.SetDemo(context.Background(), 111111, &cache.Demo{Id: 111111, Name: "asdfsadfasf"}, 0)
// 		// fmt.Println(err)
//
// 		return nil
// 	default:
// 		log.Fatalln("未知的支付事件类型", key)
// 	}
// 	return nil
// }
//
// // 支付事件消费
// func (s *Service) KQPayDemo(key, value string) error {
// 	msg := &producer.PayRecharge{}
// 	_ = json.Unmarshal([]byte(value), msg)
//
// 	fmt.Printf("KafkaSubscriber:key 消费时间:%s:%s \n", utils.GetTimeStr(time.Now()), value)
// 	// 逻辑处理start...
// 	// TODO: do something
//
// 	// err := c.DemoCache.SetDemo(context.Background(), 111111, &cache.Demo{Id: 111111, Name: "asdfsadfasf"}, 0)
// 	// fmt.Println(err)
//
// 	return nil
// }
//
// // aq消费者
// func (s *Service) AqSubscriberDemo(ctx context.Context, task *asynq.Task) error {
// 	fmt.Printf("AqSubscriber:消费时间:%s: %s %s  \n", utils.GetTimeStr(time.Now()), task.Type(), task.Payload())
// 	// 逻辑处理start...
// 	// TODO: do something
// 	return nil
// }
//
// // Cron 消费者
// func (s *Service) CronSubscriberDemo(corner *cron.Cron) {
// 	// 每个偶数秒执行
// 	spec := "*/2 * * * * *"
// 	var demoLock sync.Mutex
// 	entryID, err := corner.AddFunc(spec, func() {
// 		demoLock.Lock()
// 		defer demoLock.Unlock()
//
// 		fmt.Printf("CronDemo:消费时间:%s \n", utils.GetTimeStr(time.Now()))
//
// 		// 逻辑处理start...
// 		// TODO: do something
// 	})
// 	if err != nil {
// 		fmt.Printf("启动[demo]定时任务失败:%v %v \n", entryID, err)
// 	}
// }
```
