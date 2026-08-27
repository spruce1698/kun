.PHONY: build-linux build-darwin build-windows vet test fmt lint clean \
        build-linux-full build-darwin-full build-windows-full

# ── 默认构建：仅含 mysql + postgres 驱动（CGO_ENABLED=0，纯静态）──
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/kun .

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/kun_amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ./bin/kun_arm64 .

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/kun.exe .

# ── 完整构建：含所有数据库驱动（sqlite 需 CGO）──
build-linux-full:
	GOOS=linux GOARCH=amd64 go build -tags "with_sqlite with_clickhouse" -ldflags="-s -w" -o ./bin/kun-full .

build-darwin-full:
	GOOS=darwin GOARCH=amd64 go build -tags "with_sqlite with_clickhouse" -ldflags="-s -w" -o ./bin/kun-full_amd64 .
	GOOS=darwin GOARCH=arm64 go build -tags "with_sqlite with_clickhouse" -ldflags="-s -w" -o ./bin/kun-full_arm64 .

build-windows-full:
	GOOS=windows GOARCH=amd64 go build -tags "with_sqlite with_clickhouse" -ldflags="-s -w" -o ./bin/kun-full.exe .

# 静态检查
vet:
	go vet ./...

# 运行测试
test:
	go test ./...

# 打包模板 zip 文件
pack-tpl:
	go test -run TestPackAndExtractTemplates ./internal/new/...

# 格式化(仅脚手架自身代码,不含 tpl/ 生成目标模板)
fmt:
	gofmt -w main.go cmd/ internal/ config/ pkg/ tpl/embed.go

# 整理依赖
lint: vet
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipped"

clean:
	rm -rf ./bin
