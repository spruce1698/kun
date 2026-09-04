/**
 * @Author:
 * @Date: 2024-03-28 15:06
 * @Desc: 统一错误码与错误定义
 */

package xerror

// 定义错误码
const (
	Success int = 0 // 成功
	Unknown int = 1 // 未知错误,服务不可用

	// 参数错误 1001-1999
	NotFoundPath    int = 1001 // 地址不存在
	ParamError      int = 1002 // 请求参数错误
	InvalidArgument int = 1003 // 参数无效

	// 用户错误 2001-2999
	Unauthorized   int = 2001 // 未授权
	AuthError      int = 2002 // 授权信息错误
	AuthTimeOut    int = 2003 // 授权信息超时
	AuthRefreshErr int = 2004 // 刷新token失败
	PermitError    int = 2005 // 权限错误

	// 业务错误 3001-3999
	BusinessError int = 3001 // 业务处理失败
)

var CodeMap = map[int]string{
	Success: "成功",
	Unknown: "未知错误,服务不可用",

	NotFoundPath:    "地址不存在",
	ParamError:      "请求参数错误",
	InvalidArgument: "参数无效",

	Unauthorized: "未知授权信息",

	AuthError:      "授权信息错误",
	AuthTimeOut:    "授权信息超时",
	AuthRefreshErr: "刷新token失败",
	PermitError:    "权限错误",

	BusinessError: "服务器错误",
}

type (
	Error interface {
		error
		Code() int
		Error() string
	}

	defaultError struct {
		code    int
		message string
		err     error
	}
)

// 编译期确保 defaultError 满足 Error 接口
var _ Error = (*defaultError)(nil)

// NewError 创建业务错误
func NewError(code int, message string, err error) Error {
	return &defaultError{
		code:    code,
		message: message,
		err:     err,
	}
}

func (e *defaultError) Code() int {
	return e.code
}

func (e *defaultError) Error() string {
	return e.message
}

// Unwrap 支持 errors.Is / errors.As 穿透错误链
func (e *defaultError) Unwrap() error {
	return e.err
}
