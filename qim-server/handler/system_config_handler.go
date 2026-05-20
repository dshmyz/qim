package handler

import (
	"encoding/json"
	"qim-server/di"
	"qim-server/pkg/response"
	"qim-server/ws"

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

func GetPublicSystemConfig(c *gin.Context) {
	configSvc := di.GlobalContainer.SystemConfigService
	result, err := configSvc.GetPublicConfigs()
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

	publicConfigs, _ := configSvc.GetPublicConfigs()
	wsMsg := ws.WSMessage{Type: "system_config_updated", Data: publicConfigs}
	jsonData, _ := json.Marshal(wsMsg)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastToAllOnlineUsers(jsonData)
	}

	response.Success(c, gin.H{"message": "配置保存成功"})
}