/**
 * @Author:
 * @Date: 2024-03-28 15:10
 * @Desc: error 错误定义(纯粹错误值，无日志副作用)
 */

package xerror

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

// New 创建业务错误(支持包装底层 error)
func New(code int, message string, err ...error) Error {
	e := &defaultError{
		code:    code,
		message: message,
	}
	if len(err) > 0 {
		e.err = err[0]
	}
	return e
}

// NewError 兼容现有调用签名，创建业务错误
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
