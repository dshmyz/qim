package errors

import (
	"fmt"
	"net/http"
)

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
	// HTTP 状态码：response.ErrorFrom 据此统一落响应；0=未指定（沿用旧的 response.Xxx 路径）
	Status int `json:"-"`
	// wrapped 由 WithMessage 设置，保持与原哨兵的 errors.Is 身份链
	wrapped error
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

// NewStatusError 带 HTTP 状态码的业务错误：response.ErrorFrom 据此统一落响应
// （HTTP 状态 + 业务码 + 文案一次性下发），service 层无需再依赖 handler 逐哨兵映射。
func NewStatusError(status, code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message, Status: status}
}

// WithMessage 派生带站点文案的业务错误：沿用原错误的 Status/Code（响应形状不变），
// 仅替换 Message，并通过 Unwrap 保持与原错误的 errors.Is 身份链——
// 同一语义在不同调用点文案不同时（如"无权限发送"/"只能撤回自己发送的消息"），
// 各返回点返回 ErrXxx.WithMessage("站点文案")，errors.Is(err, ErrXxx) 仍成立。
func (e *BusinessError) WithMessage(message string) *BusinessError {
	return &BusinessError{
		Code:    e.Code,
		Message: message,
		Status:  e.Status,
		wrapped: e,
	}
}

// Unwrap 使 WithMessage 派生的错误与原哨兵保持 errors.Is 兼容。
func (e *BusinessError) Unwrap() error {
	return e.wrapped
}

// 便捷构造：按 HTTP 语义映射到通用业务码，供 service 层返回可被客户端程序化分支的错误。
func BadRequestError(message string) *BusinessError {
	return NewStatusError(http.StatusBadRequest, ErrCodeInvalidParams, message)
}
func UnauthorizedError(message string) *BusinessError {
	return NewStatusError(http.StatusUnauthorized, ErrCodeUnauthorized, message)
}
func ForbiddenError(message string) *BusinessError {
	return NewStatusError(http.StatusForbidden, ErrCodeForbidden, message)
}
func NotFoundError(message string) *BusinessError {
	return NewStatusError(http.StatusNotFound, ErrCodeNotFound, message)
}
func ConflictError(message string) *BusinessError {
	return NewStatusError(http.StatusConflict, ErrCodeConflict, message)
}
func InternalError(message string) *BusinessError {
	return NewStatusError(http.StatusInternalServerError, ErrCodeInternalError, message)
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
