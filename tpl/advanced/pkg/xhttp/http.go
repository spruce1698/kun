/**
 * @Author:
 * @Date: 2024-03-28 15:06
 * @Desc: 统一 HTTP 响应输出(支持泛型)
 */

package xhttp

import (
	"errors"
	"net/http"
	"strconv"

	"advanced/pkg/validator"
	"advanced/pkg/xerror"
	"advanced/pkg/xlog"

	"github.com/gin-gonic/gin"
)

type (
	// ReqPage 通用分页查询入参
	ReqPage struct {
		OrderField string `json:"orderField" form:"orderField"`                                         // 排序字段
		OrderType  int64  `json:"orderType" form:"orderType"`                                           // 排序类型 0:降序(默认),1:升序
		Page       int64  `json:"page" form:"page,default=1" binding:"required,min=1"`                  // 添加验证规则
		PageSize   int64  `json:"pageSize" form:"pageSize,default=20" binding:"required,min=1,max=100"` // 添加验证规则
		LastId     int64  `json:"lastId" form:"lastId"`                                                 // 上一页最大id
	}

	// PageArg 兼容别名
	PageArg = ReqPage

	Response struct {
		Code    int    `json:"code"`    // 错误码,非0为错误
		Message string `json:"message"` // 错误消息
	}
	RespData[T any] struct {
		Response
		Data T `json:"data"` // 响应数据
	}
	RespList[T any] struct {
		Response
		Page     int64 `json:"page"`     // 当前页
		PageSize int64 `json:"pageSize"` // 每页数据
		Total    int64 `json:"total"`    // 总条数
		Items    []T   `json:"items"`    // 数据列表
	}
	RespMore[T any] struct {
		Response
		Page     int64 `json:"page"`     // 当前页
		PageSize int64 `json:"pageSize"` // 每页数据
		HasMore  bool  `json:"hasMore"`  // 是否还有 true 还有下一页
		Items    []T   `json:"items"`    // 数据列表
	}
)

// RateLimitFail 请求过于频繁(429 Too Many Requests)
func RateLimitFail(ctx *gin.Context, retryAfterSec int) {
	if retryAfterSec < 1 {
		retryAfterSec = 1
	}
	ctx.Header("Retry-After", strconv.Itoa(retryAfterSec))
	ctx.AbortWithStatusJSON(http.StatusTooManyRequests, &RespData[string]{
		Response: Response{
			Code:    http.StatusTooManyRequests,
			Message: "请求过于频繁,请稍后重试",
		},
		Data: "",
	})
}

// TooRequest 请求繁忙(兼容旧签名)
func TooRequest(ctx *gin.Context) {
	RateLimitFail(ctx, 1)
}

// AuthFail 未授权。返回 HTTP 401(标准语义) + 业务 code,前端按业务 code 处理。
func AuthFail(ctx *gin.Context, code ...int) {
	authCode := xerror.Unauthorized
	if len(code) > 0 {
		authCode = code[0]
	}
	msg := xerror.CodeMap[authCode]
	r := &RespData[string]{
		Response: Response{
			Code:    authCode,
			Message: msg,
		},
		Data: "",
	}
	ctx.JSON(http.StatusUnauthorized, r)
}

// WithNotFoundPath 未找到路由
func WithNotFoundPath(ctx *gin.Context) {
	r := &RespData[string]{
		Response: Response{
			Code:    xerror.NotFoundPath,
			Message: xerror.CodeMap[xerror.NotFoundPath],
		},
		Data: "",
	}
	ctx.JSON(http.StatusNotFound, r)
}

// rootCause 获取最底层的根因错误
func rootCause(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// BusFail 业务错误
// 业务错误(xerror.Error): 记录 Warn 日志并附带底层 root cause，向客户端暴露其 message;
// 系统内部错误: 记录 Error 日志(含堆栈)，向客户端返回通用掩码，避免信息泄漏。
func BusFail(ctx *gin.Context, err error) {
	r := &RespData[string]{
		Data: "",
	}

	var e xerror.Error
	if ok := errors.As(err, &e); ok {
		// 直接取 defaultError 内层 cause（Unwrap()），避免 rootCause 指针比较歧义：
		// 当无 wrap 时 rootCause 返回 err 自身，cause==err 条件 false 走 else，
		// 看似正确但逻辑令人困惑；改为 e.Unwrap() 语义明确。
		if inner := errors.Unwrap(e); inner != nil {
			xlog.Warnf(ctx.Request.Context(), "api business error: %s, cause: %v", e.Error(), inner)
		} else {
			xlog.Warnf(ctx.Request.Context(), "api business error: %s", e.Error())
		}
		r.Code = e.Code()
		r.Message = e.Error()
	} else {
		if err != nil {
			xlog.Errorf(ctx.Request.Context(), "api internal error: %+v", err)
		}
		// 非业务错误:返回通用消息
		r.Code = xerror.BusinessError
		r.Message = xerror.CodeMap[xerror.BusinessError]
	}

	ctx.JSON(http.StatusOK, r)
}

// BusCode 业务code
func BusCode(ctx *gin.Context, code int, err error) {
	r := &Response{
		Code: code,
	}

	if err != nil {
		var valErrs validator.Errors
		// validator.Errors 类型错误进行翻译返回
		if errors.As(err, &valErrs) {
			valMsg := validator.Message(valErrs)
			if len(valMsg) > 0 {
				r.Message = valMsg[0]
			}
		} else { // 非validator.Errors 类型错误直接返回
			r.Message = err.Error()
		}
	}
	// 没有错误信息根据code返回
	if r.Message == "" {
		r.Message = xerror.CodeMap[code]
	}
	ctx.JSON(http.StatusOK, r)
}

// Success 返回成功消息(无附加数据)
func Success(ctx *gin.Context, message string) {
	if message == "" {
		message = xerror.CodeMap[xerror.Success]
	}
	ctx.JSON(http.StatusOK, Response{
		Code:    xerror.Success,
		Message: message,
	})
}

// Data 返回数据(泛型支持)
func Data[T any](ctx *gin.Context, message string, data T) {
	if message == "" {
		message = xerror.CodeMap[xerror.Success]
	}
	ctx.JSON(http.StatusOK, RespData[T]{
		Response: Response{
			Code:    xerror.Success,
			Message: message,
		},
		Data: data,
	})
}

// List 返回分页列表(泛型支持)
func List[T any](ctx *gin.Context, message string, page, pageSize, total int64, items []T) {
	// 确保 list 字段返回 [] 而非 null
	if items == nil {
		items = []T{}
	}
	if message == "" {
		message = xerror.CodeMap[xerror.Success]
	}
	ctx.JSON(http.StatusOK, RespList[T]{
		Response: Response{
			Code:    xerror.Success,
			Message: message,
		},
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Items:    items,
	})
}

// More 返回更多列表(泛型支持)
func More[T any](ctx *gin.Context, message string, page, pageSize int64, hasMore bool, items []T) {
	// 确保 list 字段返回 [] 而非 null
	if items == nil {
		items = []T{}
	}
	if message == "" {
		message = xerror.CodeMap[xerror.Success]
	}
	ctx.JSON(http.StatusOK, RespMore[T]{
		Response: Response{
			Code:    xerror.Success,
			Message: message,
		},
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
		Items:    items,
	})
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
