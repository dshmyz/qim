package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// Pagination 分页参数结构体
type Pagination struct {
	Page     int
	PageSize int
	Offset   int
}

// Parse 从 gin.Context 的 query 参数中解析分页参数
func Parse(c *gin.Context) Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return Pagination{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

// Middleware 分页参数解析中间件，自动将分页参数注入到 gin.Context
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := Parse(c)
		c.Set("pagination", p)
		c.Next()
	}
}
