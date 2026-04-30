package handler

import (
	"fmt"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetSystemConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []model.SystemConfig
	db.Find(&configs)

	result := make(map[string]interface{})
	for _, cfg := range configs {
		switch cfg.Type {
		case "number":
			var val int
			fmt.Sscanf(cfg.Value, "%d", &val)
			result[cfg.Key] = val
		case "boolean":
			result[cfg.Key] = cfg.Value == "true"
		case "json":
			result[cfg.Key] = cfg.Value
		default:
			result[cfg.Key] = cfg.Value
		}
	}

	response.Success(c, result)
}

func UpdateSystemConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	for key, value := range req {
		var cfg model.SystemConfig
		result := db.Where("key = ?", key).First(&cfg)

		strValue := fmt.Sprintf("%v", value)
		cfgType := "string"

		if _, ok := value.(float64); ok {
			cfgType = "number"
		} else if _, ok := value.(bool); ok {
			cfgType = "boolean"
		}

		if result.Error != nil {
			cfg = model.SystemConfig{
				Key:   key,
				Value: strValue,
				Type:  cfgType,
			}
			db.Create(&cfg)
		} else {
			cfg.Value = strValue
			cfg.Type = cfgType
			db.Save(&cfg)
		}
	}

	response.Success(c, gin.H{"message": "配置保存成功"})
}
