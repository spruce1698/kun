package main

import (
	"fmt"

	"github.com/spruce1698/kun/cmd/kun"
)

// go run main.go  create repo  "root:12345678@tcp(127.0.0.1:3306)/prajna" *
// go run main.go  create ctrl demo
// go run main.go  create svc demo
// go run main.go  create all demo
func main() {
	err := kun.Execute()
	if err != nil {
		fmt.Println("execute error: ", err.Error())
	}
}
