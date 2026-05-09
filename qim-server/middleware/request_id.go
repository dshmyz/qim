package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

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
				log.Printf("[Recovery] panic recovered: %v\nstack trace:\n%s", err, stack)

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

		fmt.Printf("[Request] ID=%s | Method=%s | Path=%s | Status=%d | Duration=%v\n",
			requestID,
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}
