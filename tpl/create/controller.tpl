package controller

import (
	"github.com/gin-gonic/gin"

	"{{ .ProjectName }}/pkg/xerror"
    "{{ .ProjectName }}/pkg/xhttp"
	"{{ .ProjectName }}/internal/service"
)

type {{ .FileName }}Ctl struct {
	{{ .FileNameTitleLower }}Svc service.{{ .FileName }}Svc
}

func New{{ .FileName }}Ctl({{ .FileNameTitleLower }}Svc service.{{ .FileName }}Svc) *{{ .FileName }}Ctl {
	return &{{ .FileName }}Ctl{
		{{ .FileNameTitleLower }}Svc: {{ .FileNameTitleLower }}Svc,
	}
}

func (c *{{ .FileName }}Ctl) Get{{ .FileName }}(ctx *gin.Context) {

}
