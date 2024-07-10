package {{ .PackageName }}

import (
	"github.com/gin-gonic/gin"

	"{{ .ProjectName }}/pkg/xerror"
    "{{ .ProjectName }}/pkg/xhttp"
	"{{ .ProjectName }}/internal/logic"
)

type {{ .FileName }}Ctl struct {
	{{ .FileNameTitleLower }}Logic logic.{{ .FileName }}Logic
}

func New{{ .FileName }}Ctl({{ .FileNameTitleLower }}Logic logic.{{ .FileName }}Logic) *{{ .FileName }}Ctl {
	return &{{ .FileName }}Ctl{
		{{ .FileNameTitleLower }}Logic: {{ .FileNameTitleLower }}Logic,
	}
}

func (c *{{ .FileName }}Ctl) Get{{ .FileName }}(ctx *gin.Context) {

}
