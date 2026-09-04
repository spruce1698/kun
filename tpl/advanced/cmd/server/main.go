/**
 * @Author: spruce
 * @Date: 2024-03-28 16:57
 * @Desc: 服务入口文件
 */

package main

import (
	"flag"

	"advanced/cmd/server/wire"
)

// @title prajna 项目
// @version 1.0
// @description 描述信息
// @contact.name spruce
// @host localhost:9009
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
