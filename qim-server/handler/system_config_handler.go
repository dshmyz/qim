package handler

import (
	"qim-server/di"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetSystemConfig(c *gin.Context) {
	configSvc := di.GlobalContainer.SystemConfigService
	result, err := configSvc.GetAllConfigs()
	if err != nil {
		response.InternalServerError(c, "获取配置失败")
		return
	}

	response.Success(c, result)
}

func UpdateSystemConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	configSvc := di.GlobalContainer.SystemConfigService
	if err := configSvc.BatchUpdate(req); err != nil {
		response.InternalServerError(c, "配置保存失败")
		return
	}

	response.Success(c, gin.H{"message": "配置保存成功"})
}
