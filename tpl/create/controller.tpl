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

func ({{ .FileNameFirstChar }} *{{ .FileName }}Ctrl) Get{{ .FileName }}(ctx *gin.Context) {
    data, err := {{ .FileNameFirstChar }}.{{ .FileName }}Svc.Get{{ .FileName }}(ctx.Request.Context(), req.Id)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Get{{ .FileName }} 成功", data)
}

// TODO: add your code here and delete this line