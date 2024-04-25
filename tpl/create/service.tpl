package service

import (
   "context"

   	"{{ .ProjectName }}/internal/repository/db"
   	"{{ .ProjectName }}/pkg/xerror"

   	"github.com/pkg/errors"
)

//go:generate mockgen -source=./{{ .FileNameTitleLower }}.go -destination=../../test/mocks/service/{{ .FileNameTitleLower }}.go  -package mock_service

var _ {{ .FileName }}Svc = (*{{ .FileNameTitleLower }}Svc)(nil)



type (
	{{ .FileName }}Svc interface {
		Get{{ .FileName }}(id int64) (result *{{ .FileName }}, err error)
	}

    {{ .FileName }} struct {
    }

	{{ .FileNameTitleLower }}Svc struct {
		*Service
		{{ .FileNameTitleLower }}Db db.{{ .FileName }}Repo
	}
)

func New{{ .FileName }}Svc(svc *Service, {{ .FileNameTitleLower }}db db.{{ .FileName }}Repo) {{ .FileName }}Svc {
	return &{{ .FileNameTitleLower }}Service{
		Service:        svc,
		{{ .FileNameTitleLower }}Db: {{ .FileNameTitleLower }}db,
	}
}

func (s *{{ .FileNameTitleLower }}Svc) Get{{ .FileName }}(id int64) (result *{{ .FileName }}, err error) {

    ctx := context.Background()
	if id > 0 {
		result = &{{ .FileName }}{}
		{{ .FileNameTitleLower }}, dbErr := s.{{ .FileNameTitleLower }}Db.FindOne(ctx, id)
		if dbErr != nil {
			if errors.Is(dbErr, db.ErrNotFound) {
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
