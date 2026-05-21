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

	result = mapConfigToFrontend(result)
	response.Success(c, result)
}

func mapConfigToFrontend(raw map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range raw {
		switch k {
		case "file_upload:max_size":
			if n, ok := v.(int); ok {
				out["maxFileSize"] = n / (1024 * 1024)
			} else {
				out["maxFileSize"] = 50
			}
		case "file_upload:allowed_extensions":
			if s, ok := v.(string); ok {
				out["allowedFileTypes"] = s
			}
		default:
			out[k] = v
		}
	}
	if _, ok := out["maxFileSize"]; !ok {
		out["maxFileSize"] = 50
	}
	if _, ok := out["allowedFileTypes"]; !ok {
		out["allowedFileTypes"] = defaultAllowedExtJSON()
	}
	return out
}

func defaultAllowedExtJSON() string {
	return `[".jpg",".jpeg",".png",".gif",".bmp",".webp",".pdf",".doc",".docx",".xls",".xlsx",".ppt",".pptx",".txt",".md",".csv",".zip",".rar",".7z",".mp3",".wav",".mp4",".avi",".mov"]`
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

	req = mapConfigFromFrontend(req)

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

func mapConfigFromFrontend(req map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range req {
		switch k {
		case "maxFileSize":
			if n, ok := v.(float64); ok {
				out["file_upload:max_size"] = int64(n) * 1024 * 1024
			}
		case "allowedFileTypes":
			out["file_upload:allowed_extensions"] = v
		default:
			out[k] = v
		}
	}
	return out
}