package {{ .PackageName }}

import (
   "context"

    "{{ .ProjectName }}/internal/repository/cache"
   	"{{ .ProjectName }}/internal/repository/db"
   	"{{ .ProjectName }}/pkg/xerror"
   	"{{ .ProjectName }}/pkg/xlog"

   	"github.com/pkg/errors"
)

//go:generate mockgen -source=./{{ .FileNameTitleLower }}.go -destination=../../../{{ .AddUPPath }}test/mocks/service/{{ .FilePath }}{{ .FileNameTitleLower }}.go  -package mock_service

var _ {{ .FileName }}Svc = (*{{ .FileNameTitleLower }}Svc)(nil)

type (
	{{ .FileName }}Svc interface {
		Get{{ .FileName }}(ctx context.Context, id int64) (*{{ .FileName }}, error)
	}

	{{ .FileName }}Ctx struct {
	    *Ctx

	    // TODO: add your code here and delete this line
    }

	{{ .FileNameTitleLower }}Svc struct {
		ctx *{{ .FileName }}Ctx
	}

	// TODO: add your code here and delete this line
)

func New{{ .FileName }}Svc(ctx *{{ .FileName }}Ctx) {{ .FileName }}Svc {
	return &{{ .FileNameTitleLower }}Svc{
		ctx: ctx,
	}
}

func ({{ .FileNameFirstChar }} *{{ .FileNameTitleLower }}Svc) Get{{ .FileName }}(ctx context.Context, id int64) (*{{ .FileName }}, error) {
	if id > 0 {
		result = &{{ .FileName }}{}
		{{ .FileNameTitleLower }}, dbErr := {{ .FileNameFirstChar }}.ctx.{{ .FileName }}Db.FindOne(ctx, id)
		if dbErr != nil {
			if errors.Is(dbErr, db.ErrNotFound) {
				return nil, xerror.NewError(xerror.BusinessError, "没有相关记录", dbErr)
			}
			return result, xerror.NewError(xerror.BusinessError, "Get{{ .FileName }} 失败", dbErr)
		}
		_ = copier.Copy(result, &{{ .FileNameTitleLower }})
		return result, nil
	}
	return nil, xerror.NewError(xerror.BusinessError, "Get{{ .FileName }} 失败", nil)
}

// TODO: add your code here and delete this line