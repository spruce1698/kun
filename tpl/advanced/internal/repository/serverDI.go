package repository

import (
	"advanced/internal/repository/cache"
	"advanced/internal/repository/db"

	"github.com/google/wire"
)

// 存储层
var WireServerSet = wire.NewSet(
	db.NewDemoDb,
	cache.NewDemoCache,
	// ==== Add Repo before this line, don't edit this line.====
)
