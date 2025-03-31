package {{.PackageName}}

import (
	"context"
)

//go:generate mockgen -source=./{{.InterfaceName}}.go -destination=../../../test/mocks/repository/db/{{.InterfaceName}}.go  -package mock_repo_db -aux_files mysql=./{{.InterfaceName}}_gen.go

var _ {{.StructName}}Db = (*custom{{.StructName}}Db)(nil)

type (
	{{.StructName}}Db interface {
		{{.InterfaceName}}Db

    	// TODO: add your code here and delete this line

	}
	custom{{.StructName}}Db struct {
		*default{{.StructName}}Db
	}

	// TODO: add your code here and delete this line
)

func New{{.StructName}}Db(c *Conn) {{.StructName}}Db {
	return &custom{{.StructName}}Db{
		default{{.StructName}}Db: new{{.StructName}}Db(c),
	}
}

// TODO: add your code here and delete this line

