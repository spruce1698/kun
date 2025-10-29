package {{ .PackageName }}

import (
	"github.com/gin-gonic/gin"

	"{{ .ProjectName }}/pkg/xerror"
    "{{ .ProjectName }}/pkg/xhttp"
	"{{ .ProjectName }}/internal/service/svc"
)

type (
   {{ .FileName }}Ctrl struct {
       {{ .FileName }}Svc svc.{{ .FileName }}Svc
   }

   // TODO: add your code here and delete this line
)

func ({{ .FileNameFirstChar }} *{{ .FileName }}Ctrl) Detail(ctx *gin.Context) {
    req := &struct {
		Id   int64  `form:"id"   json:"id"`
    }{}
    if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
    // TODO: add your code here and delete this line
    data, err := {{ .FileNameFirstChar }}.{{ .FileName }}Svc.Detail(ctx.Request.Context(), req.Id)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "{{ .FileName }} Detail success", data)
}

// TODO: add your code here and delete this line