package main

import (
	"fmt"

	"github.com/spruce1698/kun/cmd/kun"
)

func main() {
	err := kun.Execute()
	if err != nil {
		fmt.Println("execute error: ", err.Error())
	}
}
