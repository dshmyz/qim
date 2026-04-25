package errors

import (
	"fmt"
)

var (
	ErrBadRequest        = NewAppError(400, "请求参数错误")
	ErrUnauthorized      = NewAppError(401, "未授权")
	ErrForbidden         = NewAppError(403, "禁止访问")
	ErrNotFound          = NewAppError(404, "资源不存在")
	ErrConflict          = NewAppError(409, "资源冲突")
	ErrInternalServer    = NewAppError(500, "服务器内部错误")
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code: %d, message: %s, error: %s", e.Code, e.Message, e.Err.Error())
	}
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func NewAppErrorWithError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func BadRequest(message string) *AppError {
	return NewAppError(400, message)
}

func Unauthorized(message string) *AppError {
	return NewAppError(401, message)
}

func Forbidden(message string) *AppError {
	return NewAppError(403, message)
}

func NotFound(message string) *AppError {
	return NewAppError(404, message)
}

func Conflict(message string) *AppError {
	return NewAppError(409, message)
}

func InternalServer(message string) *AppError {
	return NewAppError(500, message)
}

func InternalServerWithError(message string, err error) *AppError {
	return NewAppErrorWithError(500, message, err)
}
