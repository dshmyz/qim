package response

import (
	stderrors "errors"
	"net/http"

	"github.com/dshmyz/qim/qim-server/pkg/errors"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errors.ErrCodeSuccess,
		Message: "success",
		Data:    data,
	})
}

func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errors.ErrCodeSuccess,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, statusCode int, code int, message string) {
	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

func ErrorWithDetail(c *gin.Context, statusCode int, code int, message string, detail interface{}) {
	c.JSON(statusCode, gin.H{
		"code":    code,
		"message": message,
		"detail":  detail,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, errors.ErrCodeInvalidParams, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, errors.ErrCodeUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, errors.ErrCodeForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, errors.ErrCodeNotFound, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, errors.ErrCodeConflict, message)
}

func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, errors.ErrCodeInternalError, message)
}

func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, errors.ErrCodeTooManyRequests, message)
}

func SuccessWithPagination(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	Success(c, gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func FromBusinessError(c *gin.Context, err *errors.BusinessError) {
	statusCode := http.StatusInternalServerError
	switch err.Code {
	case errors.ErrCodeInvalidParams:
		statusCode = http.StatusBadRequest
	case errors.ErrCodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.ErrCodeForbidden:
		statusCode = http.StatusForbidden
	case errors.ErrCodeNotFound:
		statusCode = http.StatusNotFound
	case errors.ErrCodeConflict:
		statusCode = http.StatusConflict
	case errors.ErrCodeTooManyRequests:
		statusCode = http.StatusTooManyRequests
	}
	Error(c, statusCode, err.Code, err.Message)
}

// ErrorFrom 统一错误出口：err 为 *errors.BusinessError 时按其下发（显式 Status 优先，
// 否则按 code 映射 HTTP 状态），其它错误回退通用 500。service 层返回带状态/码的业务错误后，
// handler 只需一行 response.ErrorFrom(c, err) 即可下发稳定业务码，客户端可程序化分支。
func ErrorFrom(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var be *errors.BusinessError
	if stderrors.As(err, &be) {
		if be.Status != 0 {
			Error(c, be.Status, be.Code, be.Message)
			return
		}
		FromBusinessError(c, be)
		return
	}
	Error(c, http.StatusInternalServerError, errors.ErrCodeInternalError, "服务器内部错误")
}
