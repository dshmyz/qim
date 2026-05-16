package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func OperationLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		var body []byte
		if c.Request.Body != nil {
			var err error
			body, err = io.ReadAll(c.Request.Body)
			if err != nil {
				body = []byte{}
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		blw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = blw

		c.Next()

		if shouldSkipLog(c) {
			return
		}

		duration := time.Since(start).Milliseconds()

		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")

		var uid uint
		if id, ok := userID.(uint); ok {
			uid = id
		}

		var uname string
		if name, ok := username.(string); ok {
			uname = name
		}

		if uid == 0 {
			return
		}

		log := model.OperationLog{
			UserID:     uid,
			Username:   uname,
			Action:     extractAction(c),
			Module:     extractModule(c),
			IP:         c.ClientIP(),
			UserAgent:  c.GetHeader("User-Agent"),
			RequestURL: c.Request.URL.Path,
			RequestBody: truncateString(string(body), 2000),
			Response:   truncateString(blw.body.String(), 2000),
			Duration:   int(duration),
		}

		db := database.GetDB()
		db.Create(&log)
	}
}

func shouldSkipLog(c *gin.Context) bool {
	path := c.Request.URL.Path

	skipPrefixes := []string{
		"/api/v1/health",
		"/api/v1/logs/operation",
		"/api/v1/system/config",
		"/api/v1/auth/login",
		"/api/v1/auth/logout",
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	skipMethods := []string{"OPTIONS", "HEAD"}
	for _, method := range skipMethods {
		if c.Request.Method == method {
			return true
		}
	}

	return false
}

func extractAction(c *gin.Context) string {
	method := c.Request.Method
	path := c.Request.URL.Path

	switch method {
	case "POST":
		if strings.Contains(path, "/login") {
			return "login"
		}
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	case "GET":
		if strings.Contains(path, "/export") {
			return "export"
		}
		return "view"
	default:
		return method
	}
}

func extractModule(c *gin.Context) string {
	path := c.Request.URL.Path

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return "unknown"
	}

	return parts[2]
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
