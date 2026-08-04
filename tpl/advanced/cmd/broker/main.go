/**
 * @Author: spruce
 * @Date: 2024-03-28 16:57
 * @Desc: 事件代理(调度器) 入口文件
 */

package main

import (
	"flag"

	"advanced/cmd/broker/wire"
)

func main() {
	var envConf = flag.String("conf", "config/local.yml", "config path, eg: -conf ./config/dev.yml")
	flag.Parse()

	app, err := wire.WireApp(*envConf)
	if err != nil {
		panic(err)
	}
	if err = app.Start(); err != nil {
		panic(err)
	}
	app.AwaitSignal()
}
