# 打包依赖阶段使用golang作为基础镜像
FROM golang:1.22 as builder

# SERVICE_NAME 服务名 broker,server
ARG SERVICE_NAME
#  LISTEN_PORT  监听端口
ARG LISTEN_PORT

# 启用go module
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct


WORKDIR /builder

COPY .. .

RUN go mod tidy


# 测试改变打包方式
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o  ./application ./cmd/$SERVICE_NAME/main.go

# 运行阶段指定 alpine:3.15 作为基础镜像
FROM alpine:3.15

#建立程序存放目录
WORKDIR /advanced

# 设置时区
RUN apk update && apk add tzdata

# 从build阶段复制时区
#COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /etc/localtime


# 复制文件
COPY --from=builder /builder/application /advanced/app
COPY --from=builder /builder/config /advanced/config

EXPOSE ${LISTEN_PORT}

#CMD ["/bin/sh"]

ENTRYPOINT ["/advanced/app" , "-f", "/advanced/config/release.yml"]