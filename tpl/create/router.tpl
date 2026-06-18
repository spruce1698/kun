
package {{ .PackageName }}

import (
    "{{ .ProjectName }}/internal/controller"
    "{{ .ProjectName }}/pkg/token"

    "github.com/gin-gonic/gin"
)


func {{ .FileName }}(e *gin.Engine, jwt *token.Jwt, ctx *controller.ServerCtrlCtx) {
	// TODO: add routes here and delete this line
}

