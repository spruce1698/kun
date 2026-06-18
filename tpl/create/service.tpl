package {{ .PackageName }}

import (
   "context"


	// TODO: Import repo files example
    // "{{ .ProjectName }}/internal/repository/cache"
   	// "{{ .ProjectName }}/internal/repository/db"

{{- if ne .PackageName "svc" }}
   	"{{ .ProjectName }}/internal/service/svc"
{{- end }}
   	"{{ .ProjectName }}/pkg/xerror"

   	// TODO: Import files example
   	// "{{ .ProjectName }}/pkg/xlog"

    // TODO: Import files example
   	// "github.com/jinzhu/copier"
   	// "github.com/pkg/errors"
)

//go:generate mockgen -source=./{{ .FileNameTitleLower }}.go -destination=../../../{{ .AddUPPath }}test/mocks/service/{{ .FilePath }}{{ .FileNameTitleLower }}.go  -package mock_service

var _ {{ .FileName }}Svc = (*{{ .FileNameTitleLower }}Svc)(nil)

type (
	{{ .FileName }}Svc interface {
		Detail(ctx context.Context, id int64) (*{{ .FileName }}, error)
	}

	{{ .FileName }}Ctx struct {
{{- if eq .PackageName "svc" }}
	    *Ctx
{{- else }}
	    *svc.Ctx
{{- end }}
		
	    // TODO: add "db/create/event" comment here and delete this line
    }

	{{ .FileNameTitleLower }}Svc struct {
		ctx *{{ .FileName }}Ctx
	}

    {{ .FileName }} struct {
	    // TODO: add struct fields here and delete this line
    }
	// TODO: add struct here and delete this line
)

func New{{ .FileName }}Svc(ctx *{{ .FileName }}Ctx) {{ .FileName }}Svc {
	return &{{ .FileNameTitleLower }}Svc{
		ctx: ctx,
	}
}

func ({{ .FileNameFirstChar }} *{{ .FileNameTitleLower }}Svc) Detail(ctx context.Context, id int64) (*{{ .FileName }}, error) {
	if id > 0 {
		result := &{{ .FileName }}{}
		// TODO: Priority query cache example
		// cacheData, cacheErr := {{ .FileNameFirstChar }}.ctx.{{ .FileName }}Cache.Get(ctx, id)
		// if cacheErr == nil {
		// 	if err := copier.Copy(result, cacheData); err == nil {
		// 		return result, nil
		// 	}
		// }

		// TODO: Query db example
		// {{ .FileNameTitleLower }}, dbErr := {{ .FileNameFirstChar }}.ctx.{{ .FileName }}Db.Find(ctx, id)
		// if dbErr != nil {
		// 	xlog.Errorf(ctx,"{{ .FileName }} Detail db query fail, id: %d, err: %v", id, dbErr)
		// 	if errors.Is(dbErr, db.ErrNotFound) {
		// 		return nil, xerror.NewError(ctx,xerror.BusinessError, "No relevant records", dbErr)
		// 	}
		// 	return result, xerror.NewError(ctx,xerror.BusinessError, "{{ .FileName }} Detail fail", dbErr)
		// }
		// if err := copier.Copy(result, {{ .FileNameTitleLower }}); err != nil {
		// 	return nil, xerror.NewError(ctx, xerror.BusinessError, "data copy fail", err)
		// }

		// TODO: Write cache after query db successfully example
		// _ = {{ .FileNameFirstChar }}.ctx.{{ .FileName }}Cache.Set(ctx, id, cacheData, expiration)

		return result, nil
	}
	return nil, xerror.NewError(ctx,xerror.BusinessError, "{{ .FileName }} Detail fail", nil)
}

// TODO: add your code here and delete this line