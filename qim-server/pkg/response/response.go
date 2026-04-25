package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

func Error(c *gin.Context, statusCode int, code int, message string) {
	c.JSON(statusCode, gin.H{
		"code":    code,
		"message": message,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, 400, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, 401, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, 403, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, 404, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, 409, message)
}

func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, 500, message)
}
