package errors

import "fmt"

const (
	ErrCodeSuccess         = 0
	ErrCodeInternalError   = 1000
	ErrCodeInvalidParams   = 1001
	ErrCodeUnauthorized    = 1002
	ErrCodeForbidden       = 1003
	ErrCodeNotFound        = 1004
	ErrCodeConflict        = 1005
	ErrCodeTooManyRequests = 1006

	ErrCodeUserNotFound      = 2000
	ErrCodeUserAlreadyExists = 2001
	ErrCodeInvalidPassword   = 2002
	ErrCodeUserDisabled      = 2003

	ErrCodeConversationNotFound  = 3000
	ErrCodeConversationForbidden = 3001
	ErrCodeNotMember             = 3002

	ErrCodeMessageNotFound  = 4000
	ErrCodeMessageForbidden = 4001
	ErrCodeMessageRecalled  = 4002

	ErrCodeFileNotFound     = 5000
	ErrCodeFileTooLarge     = 5001
	ErrCodeFileUploadFailed = 5002

	ErrCodeGroupNotFound = 6000
	ErrCodeNotGroupOwner = 6001
	ErrCodeGroupFull     = 6002
)

type BusinessError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

var (
	ErrInternalError   = NewBusinessError(ErrCodeInternalError, "服务器内部错误")
	ErrInvalidParams   = NewBusinessError(ErrCodeInvalidParams, "参数错误")
	ErrUnauthorized    = NewBusinessError(ErrCodeUnauthorized, "未授权")
	ErrForbidden       = NewBusinessError(ErrCodeForbidden, "无权限")
	ErrNotFound        = NewBusinessError(ErrCodeNotFound, "资源不存在")
	ErrConflict        = NewBusinessError(ErrCodeConflict, "资源冲突")
	ErrTooManyRequests = NewBusinessError(ErrCodeTooManyRequests, "请求过于频繁")
)
