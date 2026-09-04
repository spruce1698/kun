# basic — 基础布局

## 技术栈

| 类别     | 库                                                     | 说明              |
| -------- | ------------------------------------------------------ | ----------------- |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin)                 | HTTP 路由与中间件 |
| ORM      | [Gorm](https://github.com/go-gorm/gorm)                 | 数据库操作        |
| 依赖注入 | [Wire](https://github.com/google/wire)                  | 编译期依赖注入    |
| 配置管理 | [Viper](https://github.com/spf13/viper)                 | 多环境配置        |
| 日志     | [Zap](https://github.com/uber-go/zap)                   | 高性能结构化日志  |
| 链路追踪 | [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go) | 分布式全链路追踪 |
| 指标监控 | [Prometheus](https://github.com/prometheus/client_golang) | HTTP 吞吐与耗时监控 |
| JWT      | [Golang-jwt](https://github.com/golang-jwt/jwt)         | Token 鉴权        |
| 缓存     | [Go-redis](https://github.com/go-redis/redis)           | Redis 客户端      |
| 校验     | [Validator](https://github.com/go-playground/validator) | 参数校验          |

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
- **pkg**: 跨项目的公共组件（日志、配置、加密等）。

## 目录结构

```
.
├── cmd/                           服务入口目录
│   └── server/                    HTTP 服务入口
│       ├── main.go                Wire DI 启动入口，信号优雅关闭
│       └── wire/                  Wire 依赖注入与应用装配
│           ├── app.go             HTTP 应用装配（Gin 模式、中间件、路由与资源回收）
│           ├── wire.go            声明 Server 全栈分层依赖
│           └── wire_gen.go        Wire 自动生成代码
├── config/                        配置文件目录
│   ├── local.yml                  本地环境配置（YAML 格式）
│   └── release.yml                生产环境配置
├── docs/                          项目文档
│   └── sql/                       数据库建表脚本
│       └── basic.sql              数据库初始化 SQL
├── internal/                      内部业务逻辑
│   ├── handler/                   HTTP 处理器层（参数校验、调用 Service）
│   │   ├── demo.go                Demo 处理器
│   │   └── serverDI.go            Wire DI 容器，注册所有处理器
│   ├── global/                    全局常量与枚举
│   │   └── constants.go           上下文 Key 常量与 API 路由前缀常量
│   ├── middleware/                HTTP 中间件
│   │   ├── auth.go                JWT 鉴权验证（支持强制/可选模式）
│   │   ├── cors.go                跨域资源共享中间件
│   │   ├── metrics.go             Prometheus 监控指标收集中间件
│   │   ├── ratelimit.go           单机/分布式限流器
│   │   └── recovery.go            Panic 异常恢复与堆栈捕获
│   ├── repository/                数据访问层
│   │   ├── cache/                 缓存层
│   │   │   ├── demo.go            二级缓存（本地 + Redis）实现
│   │   │   ├── keys.go            缓存 Key 统一管理
│   │   │   └── local.go           线程安全本地内存缓存（go-cache 封装）
│   │   ├── db/                    数据库层
│   │   │   ├── demo.go            Demo 数据访问（自定义查询）
│   │   │   ├── demo_gen.go        SqlGen 自动生成基础 CRUD
│   │   │   └── mysql.go           GORM 连接池/事务/分页/排序封装
│   │   └── serverDI.go            Wire DI 注册所有 Repository 实现
│   ├── router/                    路由注册层
│   │   ├── router.go              全局通用路由（404 处理/健康检查/Metrics）
│   │   ├── serverDI.go            Wire DI 注册所有路由定义
│   │   └── v0/demo.go             v0 版本 Demo RESTful 路由组
│   └── service/                   业务逻辑层
│       ├── svc/                   核心业务逻辑
│       │   ├── context.go         业务服务基础上下文（配置/DB/Redis）
│       │   └── demo.go            Demo 业务实现（CRUD + 缓存）
│       └── serverDI.go            Wire DI 注册 HTTP 服务层依赖
├── pkg/                           公共工具包
│   ├── encrypt/                   加解密工具
│   │   ├── bcrypt.go              Bcrypt 密码加密与校验
│   │   ├── encrypt.go             对称加解密（AES/DES）与哈希工具（MD5/SHA）
│   │   └── rsa.go                 RSA 完整套件（OAEP 加解密/签名/验签）
│   ├── token/                     JWT Token 管理
│   │   └── jwt.go                 令牌签发/校验/刷新/黑名单撤销
│   ├── utils/                     通用工具函数
│   │   ├── convert.go             进制转换（Base62/Base58）与货币转换（元/分）
│   │   ├── file.go                文件读写与系统操作
│   │   ├── func.go                字符串工具（中文检测/脱敏/排序）
│   │   ├── net.go                 网络工具（端口/IP 检测）
│   │   ├── pool.go                Goroutine 工作池
│   │   ├── random.go              随机字符、数字及 UUID 生成
│   │   ├── snowflake.go           Snowflake 分布式 ID 生成器
│   │   ├── time.go                时间计算与格式化
│   │   └── verify.go              正则与格式校验
│   ├── validator/                 参数校验器
│   │   └── validator.go           参数规则校验（中英文翻译/自定义规则）
│   ├── xconfig/                   配置加载器（Viper/YAML/多环境）
│   │   └── xconfig.go             多环境配置动态加载
│   ├── xdb/                       数据库扩展
│   │   ├── xdb.go                 GORM 初始化与连接池管理
│   │   └── xdblog.go              GORM 自定义日志与链路追踪集成
│   ├── xerror/                    统一错误码与错误模型
│   │   └── xerror.go              结构化业务错误与错误码定义
│   ├── xhttp/                     统一 HTTP 响应输出与通用分页请求
│   │   └── http.go                统一 JSON 响应与分页入参封装
│   ├── xlog/                      日志与链路追踪（Zap + OpenTelemetry）
│   │   ├── filter.go              敏感字段脱敏过滤
│   │   ├── http.go                HTTP 请求信息提取
│   │   ├── middleware.go          Gin 请求日志中间件
│   │   ├── options.go             日志自定义选项
│   │   ├── tracer.go              OpenTelemetry 全链路追踪初始化
│   │   └── xlog.go                Zap 高性能结构化日志核心
│   ├── xredis/                    Redis 客户端封装
│   │   └── xredis.go              Redis 连接池与操作封装
│   └── xserver/                   服务管理
│       ├── http/                  HTTP 服务实现
│       │   └── http.go            Gin HTTP 引擎与优雅关闭
│       └── xserver.go             服务通用抽象与资源关闭器（Closer）
└── README.md                      项目说明
```

## 要求

您需要在系统上安装以下软件：

- Golang 1.25.11 或更高版本
- Git
- Wire

## 创建组件

您可以使用以下命令为项目创建 handler、service 和 repository 等组件：

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

这些命令将分别创建以 `UserHandler` 和 `UserSvc` 命名的组件，并将它们放置在正确的目录中。

## 常用命令

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
4. **日志与追踪**: 使用 `xlog`（基于 Zap）记录结构化日志，无缝集成 OpenTelemetry 链路追踪。
5. **缓存 Key**: 统一在 `repository/cache/keys.go` 中管理，避免散落各处。
