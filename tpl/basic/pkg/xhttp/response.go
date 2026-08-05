/**
 * @Author:
 * @Date: 2024-03-28 15:06
 * @Desc: 统一输出
 */

package xhttp

import (
	"errors"
	"net/http"

	"basic/pkg/validator"
	"basic/pkg/xerror"

	"github.com/gin-gonic/gin"
)

type (
	Response struct {
		Code    int    `json:"code"`    // 错误码,非0为错误
		Message string `json:"message"` // 错误消息
	}
	RespData struct {
		Response
		Data any `json:"data"` // 响应数据
	}
	RespList struct {
		Response
		Page     int64 `json:"page"`     // 当前页
		PageSize int64 `json:"pageSize"` // 每页数据
		Total    int64 `json:"total"`    // 总条数
		Items    any   `json:"items"`    // 数据列表
	}
	RespMore struct {
		Response
		Page     int64 `json:"page"`     // 当前页
		PageSize int64 `json:"pageSize"` // 每页数据
		HasMore  bool  `json:"hasMore"`  // 是否还有 true 还有下一页
		Items    any   `json:"items"`    // 数据列表
	}
)

// TooRequest 请求繁忙
func TooRequest(ctx *gin.Context) {
	r := &Response{
		Code:    http.StatusServiceUnavailable,
		Message: "Too many request.",
	}
	ctx.JSON(http.StatusServiceUnavailable, r)
}

// AuthFail 未知授权
func AuthFail(ctx *gin.Context, code ...int) {
	r := &RespData{
		Data: "",
	}
	r.Code = xerror.Unauthorized
	if len(code) > 0 {
		r.Code = code[0]
	}
	r.Message = xerror.CodeMap[r.Code]
	// 这里我感觉觉是个关键点，我看别人写的，过期了返回401，但是前端的axios的响应拦截器里捕获不到，所以我用201状态码，
	ctx.JSON(http.StatusCreated, r)
}

// WithNotFoundPath 未找到路由
func WithNotFoundPath(ctx *gin.Context) {
	r := &RespData{
		Data: "",
	}
	r.Code = xerror.NotFoundPath
	r.Message = xerror.CodeMap[xerror.NotFoundPath]

	ctx.JSON(http.StatusNotFound, r)
}

// BusFail 业务错误
func BusFail(ctx *gin.Context, err error) {
	r := &RespData{
		Data: "",
	}
	r.Code = xerror.BusinessError
	r.Message = err.Error()

	var e xerror.Error
	if ok := errors.As(err, &e); ok {
		r.Code = e.Code()
		r.Message = e.Error()
	}

	ctx.JSON(http.StatusOK, r)
}

// BusCode 业务code
func BusCode(ctx *gin.Context, code int, err error) {
	r := &Response{
		Code: code,
	}

	var valErrs validator.Errors
	// validator.Errors 类型错误直接返回
	if err != nil && errors.As(err, &valErrs) {
		// 对 validator.Errors 类型错误则进行翻译
		valMsg := validator.Message(valErrs)
		if len(valMsg) > 0 {
			r.Message = valMsg[0]
		}
	} else if err != nil { // 非validator.Errors 类型错误直接返回
		r.Message = err.Error()
	}
	// 没有错误信息根据code返回
	if r.Message == "" {
		r.Message = xerror.CodeMap[code]
	}
	ctx.JSON(http.StatusOK, r)
}

// Data 返回数据
func Data(ctx *gin.Context, message string, data any) {
	r := &RespData{
		Data: data,
	}
	r.Code = xerror.Success
	r.Message = message

	if r.Message == "" {
		r.Message = xerror.CodeMap[xerror.Success]
	}

	ctx.JSON(http.StatusOK, r)
}

// List 返回分页列表
func List(ctx *gin.Context, message string, page, pageSize, total int64, items any) {
	// 确保 list 字段返回 [] 而非 null
	if items == nil {
		items = []any{}
	}
	r := &RespList{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Items:    items,
	}
	r.Code = xerror.Success
	r.Message = message

	if r.Message == "" {
		r.Message = xerror.CodeMap[xerror.Success]
	}

	ctx.JSON(http.StatusOK, r)
}

// More 返回更多列表
func More(ctx *gin.Context, message string, page, pageSize int64, hasMore bool, items any) {
	// 确保 list 字段返回 [] 而非 null
	if items == nil {
		items = []any{}
	}
	r := &RespMore{
		Page:     page,
		PageSize: pageSize,
		Items:    items,
		HasMore:  hasMore,
	}
	r.Code = xerror.Success
	r.Message = message

	if r.Message == "" {
		r.Message = xerror.CodeMap[xerror.Success]
	}

	ctx.JSON(http.StatusOK, r)
}

// Text 直接输出文本字符串
func Text(ctx *gin.Context, code int, message string) {
	ctx.String(code, message)
}

// Redirect 服务器重定向 temporarily  302临时跳转,301永久跳转
func Redirect(ctx *gin.Context, code int, url string) {
	ctx.Redirect(code, url)
}

// Html 根据模板输出html页面
func Html(ctx *gin.Context, code int, name string, data any) {
	ctx.HTML(code, name, data)
}
