/**
 * @Author: albert
 * @Date: 2024-07-02
 * @Desc: service context
 */

package svc

import (
	"advanced/internal/repository/db"
	"advanced/pkg/xconfig"
	"advanced/pkg/xredis"
)

type (
	Ctx struct {
		Conf     *xconfig.Conf
		Conn     *db.Conn
		RedisCli *xredis.Client
	}
)
