package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")

				stack := string(debug.Stack())
				logger.WithModule("Recovery").Error("panic recovered", "error", err, "stack", stack)

				c.JSON(http.StatusInternalServerError, gin.H{
					"code":       -1,
					"message":    "服务器内部错误",
					"request_id": requestID,
				})

				c.Abort()
			}
		}()
		c.Next()
	}
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID, _ := c.Get("request_id")

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		// 普通访问日志降到 Debug 级别：走 logger 包，尊重 LOG_LEVEL / LOG_FORMAT / 文件拆分，
		// 避免每个 GET 请求（更新检查、render-rules 等热点）都刷 Info 输出，淹没真正的错误。
		// 只有 5xx 或耗时异常（>5s，常为 SQLite 单连接锁等待）才提到 Info，便于在风暴里定位。
		fields := []any{
			"request_id", requestID,
			"status", status,
			"duration_ms", duration.Milliseconds(),
		}
		if status >= 500 || duration > 5*time.Second {
			logger.WithModule("HTTP").Info(c.Request.Method+" "+c.Request.URL.Path, fields...)
		} else {
			logger.WithModule("HTTP").Debug(c.Request.Method+" "+c.Request.URL.Path, fields...)
		}
	}
}
