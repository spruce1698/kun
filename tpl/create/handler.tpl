package {{ .PackageName }}

import (
	"github.com/gin-gonic/gin"

	"{{ .ProjectName }}/pkg/xerror"
    "{{ .ProjectName }}/pkg/xhttp"

	// TODO: import service files here and delete this line
	// "{{ .ProjectName }}/internal/service/svc"
)

type (
   {{ .FileName }}Handler struct {
	   // TODO: add service here and delete this line
       // {{ .FileName }}Svc svc.{{ .FileName }}Svc
   }

   // TODO: add struct here and delete this line
)

func ({{ .FileNameFirstChar }} *{{ .FileName }}Handler) Detail(ctx *gin.Context) {
    req := &struct {
		Id   int64  `form:"id"   json:"id" binding:"required,gt=0"`
    }{}
    if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
    // TODO: add service call logic here and delete this line
    // data, err := {{ .FileNameFirstChar }}.{{ .FileName }}Svc.Detail(ctx.Request.Context(), req.Id)
	// if err != nil {
	// 	xhttp.BusFail(ctx, err)
	// 	return
	// }
	// xhttp.Data(ctx, "{{ .FileName }} Detail success", data)
}

// TODO: add your code here and delete this line