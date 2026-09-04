# 工具版本固定,避免 @latest 在某天引入破坏性更新让 CI 突然失败,
# 也避免不同开发者本地版本不一致导致 wire_gen.go 反复抖动。
WIRE_VERSION    ?= v0.6.0
MOCKGEN_VERSION ?= v0.5.0
SWAG_VERSION    ?= v1.16.4

.PHONY: init
init:
	go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)

.PHONY: bootstrap
bootstrap:
	cd ./docs/ci && docker compose up -d
	kun run ./cmd/server

.PHONY: mock
mock:
	go generate ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -race -cover ./...

.PHONY: lint
lint:
	golangci-lint run ./...

# CGO_ENABLED=0 生成静态二进制,否则产物在 alpine(musl)镜像里无法运行。
.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/server ./cmd/server
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/broker ./cmd/broker

# 每个 --build-arg 只能带一个 KEY=VALUE,写成 `--build-arg A=1 B=2`
# 会让 docker 把 B=2 当成构建上下文路径。
.PHONY: docker
docker:
	docker build -f docs/docker/Dockerfile --build-arg SERVICE_NAME=server --build-arg LISTEN_PORT=9998 -t advanced/server:latest .
	docker build -f docs/docker/Dockerfile --build-arg SERVICE_NAME=broker --build-arg LISTEN_PORT=9999 -t advanced/broker:latest .

# 输出目录必须是 ./swagger:代码里 import 的是 "advanced/swagger"
# (见 internal/router/router.go),输出到 ./docs 生成的文档永远不会被编译进程序。
.PHONY: swag
swag:
	swag init -g cmd/server/main.go -o ./swagger --parseDependency
