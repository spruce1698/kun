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
│   │   └── wire/                  Wire 依赖注入声明与自动生成
│   │       ├── wire.go            声明 Broker 全链路依赖
│   │       └── wire_gen.go        Wire 自动生成代码
│   └── server/                    HTTP 服务入口
│       ├── main.go                Wire DI 启动入口，信号优雅关闭
│       └── wire/                  Wire 依赖注入声明与自动生成
│           ├── wire.go            声明 Server 全栈分层依赖
│           └── wire_gen.go        Wire 自动生成代码
├── config/                        配置文件目录
│   └── local.yml                  本地环境配置（YAML 格式）
├── docs/                          项目文档
│   ├── ci/                        CI 编排
│   │   └── docker-compose.yml     容器化编排
│   ├── docker/                    镜像构建
│   │   └── Dockerfile             Docker 构建文件
│   ├─ sql/                        数据库建表脚本
│   └── scripts/                       辅助脚本
│       └── swagger.sh                 自动生成 Swagger 文档
├── swagger/                       Swagger 接口文档                
├── internal/                      内部业务逻辑
│   ├── handler/                   HTTP 处理器层（参数校验、调用 Service）
│   │   ├── demo/demo.go           Demo CRUD 处理器（含缓存、事件发布）
│   │   └── serverDI.go            Wire DI 容器，注册所有处理器
│   ├── event/                     事件发布/订阅抽象层
│   │   ├── publisher.go           多后端发布器（Kafka 同步 + Asynq 异步/延时/定时）
│   │   └── subscriber.go          订阅管理器（Kafka 消费组 + Asynq Task Worker）
│   ├── global/                    全局常量与枚举
│   │   ├── ctx.go                 上下文 Key 常量（认证 Token、账户 ID）
│   │   ├── event.go               事件类型/主题/消费组枚举及消息结构
│   │   ├── pay.go                 支付状态与支付方式枚举
│   │   └── router.go              API 路由前缀常量
│   ├── middleware/                HTTP 中间件
│   │   ├── auth.go                JWT 鉴权（强制/可选/写入三种模式）
│   │   ├── cors.go                跨域资源共享
│   │   ├── csrf.go                CSRF 防护（gorilla/csrf）
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
│   │   ├── router.go              全局路由工具（404 处理/健康检查/Swagger）
│   │   ├── serverDI.go            Wire DI 注册所有路由定义
│   │   └── v0/demo.go             v0 版本 Demo RESTful 路由组
│   └── service/                   业务逻辑层
│       ├── svc/                   核心业务逻辑
│       │   ├── context.go         业务服务基础上下文（配置/DB/Redis/日志）
│       │   ├── demo.go            Demo 业务编排（CRUD + 缓存 + 事件发布）
│       │   └── broker.go          事件消费处理器（Kafka + Asynq 任务路由）
│       ├── brokerDI.go            Wire DI 注册 Broker 事件订阅任务
│       └── serverDI.go            Wire DI 注册 HTTP 服务层依赖
├── pkg/                           公共工具包
│   ├── asynq/                     Asynq 异步/延时/定时任务队列封装
│   ├── encrypt/                   加密工具
│   │   ├── encrypt.go             MD5/SHA/PBKDF2 哈希工具
│   │   ├── rsa.go                 RSA 加解密（公钥加密/私钥解密）
│   │   └── rsaExt.go              RSA PEM 解析/分块加解密扩展
│   ├── kafka/                     Kafka 客户端封装
│   │   ├── client.go              主题管理客户端（列举/创建/删除）
│   │   ├── publisher.go           消息发布器（压缩/超时/批量异步）
│   │   └── subscriber.go          消费者（两种订阅模式/消费组/偏移量）
│   ├── token/                     JWT Token 管理
│   │   ├── error.go               JWT 错误类型定义（位标志）
│   │   └── jwt.go                 令牌创建/验证/刷新/Redis 撤销（单例模式）
│   ├── utils/                     通用工具函数
│   │   ├── convert.go             进制转换（base62/base58）
│   │   ├── decimal.go             货币单位转换（元 ⇔ 分）
│   │   ├── file.go                文件系统操作（存在校验/读写/遍历）
│   │   ├── func.go                字符串工具（中文检测/脱敏/字典排序）
│   │   ├── net.go                 网络工具（端口/IP 检测）
│   │   ├── pool.go                Goroutine 工作池
│   │   ├── random.go              随机字符串/数字生成
│   │   ├── snowflake.go           Twitter Snowflake 分布式 ID
│   │   ├── time.go                时间计算与转换工具
│   │   ├── uuid.go                UUID v4 生成
│   │   └── verify.go              密码强度校验
│   ├── validator/                 参数校验器（中英文错误翻译/自定义规则）
│   ├── xconfig/                   配置加载器（Viper/YAML/环境变量/热重载）
│   ├── xdb/                       数据库扩展
│   │   ├── xdb.go                 GORM 初始化（连接池/读写分离/链路追踪）
│   │   └── xdblog.go              GORM 自定义日志（SQL 追踪集成）
│   ├── xerror/                    错误码与错误处理
│   │   ├── base36.go              Base36 编解码（错误码序列化）
│   │   ├── code.go                业务错误码常量定义
│   │   └── error.go               结构化错误（错误码/指纹/日志集成）
│   ├── xhttp/                     HTTP 请求/响应封装
│   │   ├── request.go             请求参数结构（分页/排序）
│   │   └── response.go            统一响应结构（数据/列表/分页）
│   ├── xlog/                      日志与链路追踪（基于 Zap + OpenTelemetry）
│   │   ├── filter.go              敏感数据过滤（密码/Token 脱敏）
│   │   ├── http.go                HTTP 请求/响应日志结构
│   │   ├── middleware.go          Gin 追踪中间件（请求日志 + Span/Body/Header/耗时）
│   │   ├── options.go             日志选项（级别/轮转/Kafka Hook）
│   │   ├── tracer.go              OpenTelemetry 链路追踪初始化（OTLP gRPC）
│   │   └── xlog.go                Zap 初始化（多输出/链路追踪）
│   ├── xredis/                    Redis 客户端（连接池/集群/链路追踪）
│   └── xserver/                   服务管理
│       ├── broker/                Broker 引擎（Kafka + Asynq 订阅/健康检查）
│       ├── http/                  HTTP 引擎（Gin 初始化/中间件栈/路由注册）
│       └── xserver.go             服务抽象层（信号处理/优雅关闭）
├── test/                          单元测试与 Mock
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

### 常用命令

```bash
# 启动项目
kun run

# 编译 Wire 依赖注入
kun wire
```

## 项目约定

1. **依赖注入**: 所有层之间通过 `Wire` 进行编译期依赖注入，避免手动管理依赖。
2. **配置管理**: 使用 `Viper` 加载 `config/` 下的配置文件，支持多环境。
3. **错误处理**: 统一使用 `xerror` 包管理错误码，返回结构化错误响应。
4. **日志**: 使用 `xlog`（基于 Zap）记录结构化日志，支持链路追踪。
5. **缓存 Key**: 统一在 `repository/cache/keys.go` 中管理，避免散落各处。
