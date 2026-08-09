/**
 * @Author:
 * @Date: 2024-03-28 15:10
 * @Desc: error 错误
 */

package xerror

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"strings"

	"advanced/pkg/xlog"
)

type (
	Error interface {
		error
		Code() int
		Error() string
		Log(ctx context.Context)
	}

	defaultError struct {
		code    int
		message string
		err     error
	}
)

// 编译期确保 defaultError 满足 Error 接口
var _ Error = (*defaultError)(nil)

// errCodeReplacer 预编译错误码替换规则，避免每次调用都重建 Replacer
var errCodeReplacer = strings.NewReplacer(
	"O", "0",
	"I", "1",
)

func NewError(ctx context.Context, code int, message string, err error) error {
	e := &defaultError{
		code:    code,
		message: message,
		err:     err,
	}
	if e.err != nil {
		e.message = fmt.Sprintf("%s!错误代码:%4s", e.message, errCode(e.err))
		e.Log(ctx)
	}
	return e
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

func (e *defaultError) Log(ctx context.Context) {
	if e.err != nil {
		xlog.Errorf(ctx, "错误代码:%4s %v", errCode(e.err), e.err)
	}
}

func errCode(err error) string {
	if err == nil {
		return "0000"
	}
	h := md5.Sum([]byte(err.Error()))
	s := base36Encode(binary.BigEndian.Uint64(h[:8]) & 0xFFFFF)
	// 手动补零到4位（fmt %04s 对字符串无效，会补空格而非零）
	if len(s) < 4 {
		s = "0000"[4-len(s):] + s
	}
	return errCodeReplacer.Replace(s)
}
