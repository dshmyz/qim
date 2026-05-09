package params

import (
	"fmt"
	"strconv"

	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetUintParam 从 gin.Context 的 URL path 参数中解析 uint 类型的值
func GetUintParam(c *gin.Context, key string) (uint, error) {
	idStr := c.Param(key)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("无效的%s", key)
	}
	return uint(id), nil
}

// MustGetUintParam 从 gin.Context 的 URL path 参数中解析 uint 类型的值，
// 如果解析失败则自动返回 BadRequest 响应
func MustGetUintParam(c *gin.Context, key string) (uint, bool) {
	id, err := GetUintParam(c, key)
	if err != nil {
		response.BadRequest(c, err.Error())
		return 0, false
	}
	return id, true
}
