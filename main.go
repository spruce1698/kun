package main

import (
	"os"

	"github.com/spruce1698/kun/cmd/kun"
	"github.com/spruce1698/kun/pkg/output"
)

// go run main.go create db "root:123456@tcp(127.0.0.1:3306)/dbname" *
// go run main.go create ctrl demo
// go run main.go create svc demo
// go run main.go create cs demo
func main() {
	// 子命令失败时由各命令自行 output.Error 提示,这里仅负责退出码,
	// 避免重复打印(Error: ... 前缀)和误打印 usage。
	err := kun.Execute()
	if err != nil {
		output.Error("execute error: %v", err)
		os.Exit(1)
	}
}
