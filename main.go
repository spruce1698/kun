package main

import (
	"github.com/spruce1698/kun/cmd/kun"
	"github.com/spruce1698/kun/pkg/fmt"
)

// go run main.go create db "root:123456@tcp(127.0.0.1:3306)/dbname" *
// go run main.go create ctrl demo
// go run main.go create svc demo
// go run main.go create cs demo
func main() {
	err := kun.Execute()
	if err != nil {
		fmt.Error("execute error: ", err.Error())
	}
}
