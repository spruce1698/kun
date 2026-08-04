.PHONY: init
init:
	go install github.com/google/wire/cmd/wire@latest
	go install go.uber.org/mock/mockgen@latest
	go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: bootstrap
bootstrap:
	cd ./deploy/ci && docker compose up -d && cd ../../
	kun run ./cmd/server

.PHONY: mock
mock:
	go generate ./...


.PHONY: build
build:
	go build -ldflags="-s -w" -o ./bin/server ./cmd/server

.PHONY: docker
docker:
	docker build -f deploy/docker/Dockerfile --build-arg SERVICE_NAME=broker LISTEN_PORT=9999 -t 127.0.0.0:9999/broker .
	docker run --rm -i 127.0.0.0:9999/broker

.PHONY: swag
swag:
	swag init  -g cmd/server/main.go -o ./docs --parseDependency