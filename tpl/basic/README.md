# basic — 基础布局

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
├── cmd
│   └── server                               服务入口目录
│       ├── wire
│       │   ├── wire.go                      Wire 依赖注入声明
│       │   └── wire_gen.go                  Wire 自动生成代码
│       └── main.go                          服务入口文件
├── config
│   └── local.yml                            本地环境配置
├── docs
│   └── sql
│       └── basic.sql                        数据库初始化脚本
├── internal
│   ├── handler                              HTTP 处理器层
│   │   ├── demo.go                          Demo 处理器
│   │   └── serverDI.go                      Server 处理器 DI 配置
│   ├── global                               全局常量与枚举
│   │   ├── ctx.go                           上下文 key 常量
│   │   └── router.go                        路由前缀常量
│   ├── middleware                           HTTP 中间件
│   │   ├── auth.go                          授权验证中间件
│   │   ├── cors.go                          跨域中间件
│   │   ├── otel.go                          OpenTelemetry 链路追踪
│   │   ├── recovery.go                      异常恢复（panic 处理）
│   │   └── tracing.go                       请求日志追踪
│   ├── repository                           数据访问层
│   │   ├── cache                            缓存层
│   │   │   ├── demo.go                      Demo 缓存实现
│   │   │   ├── keys.go                      缓存 Key 统一管理
│   │   │   └── local.go                     本地缓存（go-cache）
│   │   ├── db                               数据库层
│   │   │   ├── demo.go                      Demo 数据访问（自定义实现）
│   │   │   ├── demo_gen.go                  Demo 数据访问（SqlGen 自动生成）
│   │   │   └── mysql.go                     MySQL 通用操作封装
│   │   └── serverDI.go                      Server 存储层 DI 配置
│   ├── router                               路由注册层
│   │   ├── v0
│   │   │   └── demo.go                      v0 版本 Demo 路由
│   │   ├── router.go                        通用路由 & 404 处理
│   │   └── serverDI.go                      Server 路由 DI 配置
│   └── service                              业务逻辑层
│       ├── svc                              核心业务逻辑
│       │   ├── context.go                   业务上下文（配置/DB/Redis）
│       │   └── demo.go                      Demo 业务实现
│       └── serverDI.go                      Server 服务 DI 配置
├── pkg                                      公共工具包
│   ├── encrypt                              加密工具
│   │   ├── encrypt.go                       AES/DES/MD5/SHA 等加密
│   │   ├── rsa.go                           RSA 加解密
│   │   └── rsaExt.go                        RSA 扩展（PKCS1/PKCS8）
│   ├── token                                JWT Token 管理
│   │   ├── error.go                         JWT 错误定义
│   │   └── jwt.go                           JWT 签名 & 验签
│   ├── utils                                通用工具函数
│   │   ├── convert.go                       Base62/Base58 编码转换
│   │   ├── decimal.go                       浮点数精度处理
│   │   ├── file.go                          文件操作
│   │   ├── func.go                          通用函数工具
│   │   ├── net.go                           网络工具
│   │   ├── pool.go                          协程池
│   │   ├── random.go                        随机数生成
│   │   ├── time.go                          时间工具
│   │   ├── uuid.go                          UUID 生成
│   │   └── verify.go                        验证工具（手机号/邮箱等）
│   ├── validator                            参数校验器
│   │   └── validator.go                     系统输入参数校验（中英文）
│   ├── xconfig                              配置管理（Viper）
│   │   └── xconfig.go                       多环境配置加载
│   ├── xdb                                  数据库扩展
│   │   ├── xdb.go                           GORM 数据库通用操作
│   │   └── xdblog.go                        数据库操作日志
│   ├── xerror                               统一错误处理
│   │   ├── base36.go                        错误码 Base36 编码
│   │   ├── code.go                          错误码定义
│   │   └── error.go                         错误接口与处理
│   ├── xhttp                                HTTP 请求/响应封装
│   │   ├── request.go                       请求参数封装
│   │   └── response.go                      统一响应输出
│   ├── xlog                                 日志封装（Zap）
│   │   ├── filter.go                        敏感字段过滤
│   │   ├── http.go                          HTTP 请求信息结构
│   │   ├── options.go                       自定义日志选项
│   │   └── xlog.go                          日志核心封装
│   ├── xredis                               Redis 客户端封装
│   │   └── xredis.go                        Redis 操作 & 链路追踪
│   └── xserver                              服务管理
│       ├── xserver.go                       服务生命周期接口
│       └── http                             HTTP 服务实现
│           └── http.go                      Gin HTTP 引擎
└── README.md                                项目说明
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
4. **日志**: 使用 `xlog`（基于 Zap）记录结构化日志，支持链路追踪。
5. **缓存 Key**: 统一在 `repository/cache/keys.go` 中管理，避免散落各处。
