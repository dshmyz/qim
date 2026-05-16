package params

import (
	"fmt"
	"strconv"

	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未认证")
		return 0, false
	}
	userID, ok := val.(uint)
	if !ok {
		response.Unauthorized(c, "用户信息异常")
		return 0, false
	}
	return userID, true
}

func GetUintParam(c *gin.Context, key string) (uint, error) {
	idStr := c.Param(key)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("无效的%s", key)
	}
	return uint(id), nil
}

func MustGetUintParam(c *gin.Context, key string) (uint, bool) {
	id, err := GetUintParam(c, key)
	if err != nil {
		response.BadRequest(c, err.Error())
		return 0, false
	}
	return id, true
}

func ParseUintParam(c *gin.Context, key string) (uint, bool) {
	str := c.Param(key)
	id, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("无效的%s", key))
		return 0, false
	}
	return uint(id), true
}

func ParseUintQuery(c *gin.Context, key string) (uint, bool) {
	str := c.Query(key)
	if str == "" {
		return 0, true
	}
	id, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("无效的%s", key))
		return 0, false
	}
	return uint(id), true
}
