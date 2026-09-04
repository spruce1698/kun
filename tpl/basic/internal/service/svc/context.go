/**
 * @Author: albert
 * @Date: 2024-07-02
 * @Desc: service context
 */

package svc

import (
	"basic/internal/repository/db"
	"basic/pkg/xconfig"
	"basic/pkg/xredis"
)

type (
	Ctx struct {
		Conf     *xconfig.Conf
		Conn     *db.Conn
		RedisCli *xredis.Client
	}
)
