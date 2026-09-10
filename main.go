package main

import (
	"errors"
	"os"

	"github.com/sprucepeak/kun/cmd/kun"
	"github.com/sprucepeak/kun/pkg/output"
)

// go run main.go create db "root:123456@tcp(127.0.0.1:3306)/dbname" *
// go run main.go create hdl demo
// go run main.go create svc demo
// go run main.go create hs demo
func main() {
	// 子命令失败时由各命令自行 output.Error 提示,这里仅负责退出码,
	// 避免重复打印(Error: ... 前缀)和误打印 usage。
	err := kun.Execute()
	if err != nil {
		output.Error("execute error: %v", err)
		var cmdErr *kun.CommandError
		if errors.As(err, &cmdErr) {
			output.Tip("Run '%s -h' for more information.", cmdErr.CmdPath)
		} else {
			output.Tip("Run 'kun -h' for more information.")
		}
		os.Exit(1)
	}
}
