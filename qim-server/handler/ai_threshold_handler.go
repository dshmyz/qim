package handler

import (
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetAIThresholds 返回所有 AI 阈值的当前值。
func GetAIThresholds(c *gin.Context) {
	svc := di.GlobalContainer.AiThresholdService
	if svc == nil {
		response.InternalServerError(c, "阈值服务未初始化")
		return
	}
	response.Success(c, svc.GetAll())
}

// UpdateAIThresholds 批量更新 AI 阈值（热更新，无需重启）。
func UpdateAIThresholds(c *gin.Context) {
	svc := di.GlobalContainer.AiThresholdService
	if svc == nil {
		response.InternalServerError(c, "阈值服务未初始化")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil || len(req) == 0 {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := svc.BatchUpdate(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.SuccessWithMessage(c, "阈值已更新，下次请求即生效", nil)
}

// GetAIThresholdSchema 返回阈值的元数据（名称、范围、说明），供前端渲染表单。
func GetAIThresholdSchema(c *gin.Context) {
	type schema struct {
		Key         string  `json:"key"`
		Label       string  `json:"label"`
		Description string  `json:"description"`
		Default     float64 `json:"default"`
		Min         float64 `json:"min"`
		Max         float64 `json:"max"`
	}
	result := make([]schema, 0, len(config.Thresholds))
	for _, t := range config.Thresholds {
		result = append(result, schema{
			Key:         t.Key,
			Label:       t.Label,
			Description: t.Description,
			Default:     t.Default,
			Min:         t.Min,
			Max:         t.Max,
		})
	}
	response.Success(c, result)
}
