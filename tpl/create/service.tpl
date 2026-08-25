package {{ .PackageName }}

import (
   "context"


	// TODO: import repo files here and delete this line
    // "{{ .ProjectName }}/internal/repository/cache"
   	// "{{ .ProjectName }}/internal/repository/db"

{{- if ne .PackageName "svc" }}
   	"{{ .ProjectName }}/internal/service/svc"
{{- end }}
   	"{{ .ProjectName }}/pkg/xerror"

   	// TODO: import files here and delete this line
   	// "{{ .ProjectName }}/pkg/xlog"

    // TODO: import files here and delete this line
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
		// {{ .FileName }}Db db.{{ .FileName }}Db
		// {{ .FileName }}Cache dbcache.{{ .FileName }}Cache
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
		// TODO: priority query cache here and delete this line
		// cacheData, cacheErr := {{ .FileNameFirstChar }}.ctx.{{ .FileName }}Cache.Get(ctx, id)
		// if cacheErr == nil {
		// 	if err := copier.Copy(result, cacheData); err == nil {
		// 		return result, nil
		// 	}
		// }

		// TODO: query db here and delete this line
		// {{ .FileNameTitleLower }}, dbErr := {{ .FileNameFirstChar }}.ctx.{{ .FileName }}Db.Find(ctx, id)
		// if dbErr != nil {
		// 	xlog.Errorf(ctx,"{{ .FileName }} Detail db query fail, id: %d, err: %v", id, dbErr)
		// 	if errors.Is(dbErr, db.ErrNotFound) {
		// 		return nil, xerror.NewError(xerror.BusinessError, "No relevant records", dbErr)
		// 	}
		// 	return result, xerror.NewError(xerror.BusinessError, "{{ .FileName }} Detail fail", dbErr)
		// }
		// if err := copier.Copy(result, {{ .FileNameTitleLower }}); err != nil {
		// 	return nil, xerror.NewError(xerror.BusinessError, "data copy fail", err)
		// }

		// TODO: write cache after query db successfully here and delete this line
		// _ = {{ .FileNameFirstChar }}.ctx.{{ .FileName }}Cache.Set(ctx, id, cacheData, expiration)

		return result, nil
	}
	return nil, xerror.NewError(xerror.BusinessError, "{{ .FileName }} Detail fail", nil)
}

// TODO: add your code here and delete this line