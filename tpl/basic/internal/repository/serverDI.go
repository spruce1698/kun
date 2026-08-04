package repository

import (
	"basic/internal/repository/cache"
	"basic/internal/repository/db"

	"github.com/google/wire"
)

// 存储层
var WireServerSet = wire.NewSet(
	db.NewDemoDb,
	cache.NewDemoCache,
	// ==== Add Repo before this line, don't edit this line.====
)
