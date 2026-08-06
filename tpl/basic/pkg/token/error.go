/**
 * @Author:
 * @Date: 2024-03-28 18:31
 * @Desc: jwt error
 */

package token

type Error struct {
	Inner  error
	Errors uint32
	text   string
}

func newError(errorText string, errorFlags uint32) *Error {
	return &Error{
		text:   errorText,
		Errors: errorFlags,
	}
}

// Validation error is an error type
func (o *Error) Error() string {
	if o.Inner != nil {
		return o.Inner.Error()
	}

	return o.text
}

// No errors
func (o *Error) valid() bool {
	return o.Errors == 0
}
