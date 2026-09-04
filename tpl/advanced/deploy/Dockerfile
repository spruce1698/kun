# 打包依赖阶段使用golang作为基础镜像
FROM golang:1.25-alpine AS builder

# SERVICE_NAME 服务名 broker,server
ARG SERVICE_NAME
#  LISTEN_PORT  监听端口
ARG LISTEN_PORT

# 启用go module
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /builder

# 先只拷依赖清单并下载,使依赖层能被 docker 缓存复用:
# 只要 go.mod/go.sum 未变,改源码不会重新下载依赖。
# 用 go mod download 而非 go mod tidy —— tidy 会改写 go.mod/go.sum
# (可能静默升级依赖),破坏可重现构建。
COPY go.mod go.sum ./
RUN go mod download

# 再拷源码
COPY . .

# CGO_ENABLED=0 生成静态二进制,否则产物动态链接 glibc,在 alpine(musl)上
# 会报 "no such file or directory"。
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./application ./cmd/${SERVICE_NAME}

# 运行阶段
FROM alpine:3.21

# ARG 的作用域仅限单个构建阶段,运行阶段要用必须重新声明,
# 否则下面的 EXPOSE 会展开为空值。
ARG LISTEN_PORT

#建立程序存放目录
WORKDIR /advanced

# 设置时区(--no-cache 避免残留 apk 索引使镜像膨胀)
RUN apk add --no-cache tzdata ca-certificates wget

# 复制文件
COPY --from=builder /builder/application /advanced/app
COPY --from=builder /builder/config /advanced/config

# 以非 root 用户运行:容器内一旦被 RCE,攻击者不会直接拿到 root。
RUN adduser -D -u 10001 appuser && chown -R appuser:appuser /advanced
USER appuser

EXPOSE ${LISTEN_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${LISTEN_PORT}/ping" || exit 1

# 注意:flag 名是 -conf(见 cmd/server/main.go),不是 -f;
# 传错 flag 会让 Go 的 flag 包打印 usage 并以退出码 2 结束。
ENTRYPOINT ["/advanced/app", "-conf", "/advanced/config/release.yml"]
