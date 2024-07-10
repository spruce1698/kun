package {{ .PackageName }}

import (
   "context"

   	"{{ .ProjectName }}/internal/repository/mysql"
   	"{{ .ProjectName }}/pkg/xerror"

   	"github.com/pkg/errors"
)

//go:generate mockgen -source=./{{ .FileNameTitleLower }}.go -destination=../../test/mocks/logic/{{ .FilePath }}{{ .FileNameTitleLower }}.go  -package mock_logic

var _ {{ .FileName }}Logic = (*{{ .FileNameTitleLower }}Logic)(nil)



type (
	{{ .FileName }}Logic interface {
		Get{{ .FileName }}(id int64) (result *{{ .FileName }}, err error)
	}

	{{ .FileName }} struct {
    }

	{{ .FileNameTitleLower }}Logic struct {
		*Logic
		{{ .FileNameTitleLower }}Mysql mysql.{{ .FileName }}Repo
	}
)

func New{{ .FileName }}Logic(logic *Logic, {{ .FileNameTitleLower }}Mysql mysql.{{ .FileName }}Repo) {{ .FileName }}Logic {
	return &{{ .FileNameTitleLower }}Logic{
		Logic:        logic,
		{{ .FileNameTitleLower }}Mysql: {{ .FileNameTitleLower }}Mysql,
	}
}

func (l *{{ .FileNameTitleLower }}Logic) Get{{ .FileName }}(id int64) (result *{{ .FileName }}, err error) {

    ctx := context.Background()
	if id > 0 {
		result = &{{ .FileName }}{}
		{{ .FileNameTitleLower }}, dbErr := l.{{ .FileNameTitleLower }}Mysql.FindOne(ctx, id)
		if dbErr != nil {
			if errors.Is(dbErr, mysql.ErrNotFound) {
				return nil, xerror.NewError(xerror.BusinessError, "没有相关记录", dbErr)
			}
			return result, xerror.NewError(xerror.BusinessError, "Get{{ .FileName }} 失败", dbErr)
		}
		_ = copier.Copy(result, &{{ .FileNameTitleLower }})
		return result, nil
	}

    result = &{{ .FileName }}{}
	return result, xerror.NewError(xerror.BusinessError, "Get{{ .FileName }} 失败", nil)

}
