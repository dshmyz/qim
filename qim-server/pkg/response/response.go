package response

import (
	"net/http"

	"qim-server/pkg/errors"

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
