package handler

import (
	"errors"
	"log"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

// POST/GET/PUT 相关的 AI 模型路由管理端点，用于在管理后台可视化配置
// 「任务类型 → provider/model」的映射，持久化到 SystemConfig 并在运行时热更 AIService。

// routerSvc 便捷取用容器内的 AIRouterService（懒加载，避免依赖注入顺序问题）。
func routerSvc() *service.AIRouterService {
	if di.GlobalContainer.AIRouterService != nil {
		return di.GlobalContainer.AIRouterService
	}
	return service.NewAIRouterService(database.GetDB())
}

// GetAIRouter 获取当前生效的 AI 模型路由与可用供应商候选。
// @Summary 获取 AI 模型路由
// @Description 返回当前生效路由（DB 覆盖优先，否则 config.yaml 默认）；同时返回已配置供应商作为 provider 下拉候选
// @Tags AI
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/admin/ai/router [get]
func GetAIRouter(c *gin.Context) {
	rc, providers, err := routerSvc().GetEffectiveRouter()
	if err != nil {
		response.InternalServerError(c, "获取路由配置失败: "+err.Error())
		return
	}

	candidates := make([]gin.H, 0, len(providers))
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		candidates = append(candidates, gin.H{
			"id":     p.ID,
			"name":   p.Name,
			"type":   p.APIType,
			"models": p.Models,
		})
	}

	dbOverride, _ := routerSvc().HasDBRouter()

	response.Success(c, gin.H{
		"defaultTask": rc.DefaultTask,
		"routes":      rc.Routes,
		"providers":   candidates,
		"usingDb":     dbOverride,
	})
}

// SaveAIRouter 保存 AI 模型路由到 DB 并热更新运行时。
// @Summary 保存 AI 模型路由
// @Description 将任务→provider/model 路由写入 SystemConfig 并热更新 AIService，无需重启
// @Tags AI
// @Accept json
// @Produce json
// @Param body body object true "路由配置 {defaultTask, routes}"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/ai/router [put]
func SaveAIRouter(c *gin.Context) {
	var req struct {
		DefaultTask ai.TaskType              `json:"defaultTask"`
		Routes      map[ai.TaskType]ai.Route `json:"routes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if req.Routes == nil {
		response.BadRequest(c, "routes 不能为空")
		return
	}

	rc := &ai.RouterConfig{DefaultTask: req.DefaultTask, Routes: req.Routes}
	if err := routerSvc().SaveRouter(di.GlobalContainer.AIService, rc); err != nil {
		log.Printf("[AIRouterHandler] save router failed: %v", err)
		if errors.Is(err, service.ErrUnknownRouterProvider) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "保存路由失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"usingDb": true})
}

// ClearAIRouter 清除 DB 路由覆盖，回到 config.yaml 默认路由。
// @Summary 恢复 config.yaml 默认路由
// @Description 删除 DB 路由覆盖并热更新回 config.yaml 配置，无需重启
// @Tags AI
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/admin/ai/router [delete]
func ClearAIRouter(c *gin.Context) {
	if err := routerSvc().ClearRouter(di.GlobalContainer.AIService); err != nil {
		log.Printf("[AIRouterHandler] clear router failed: %v", err)
		response.InternalServerError(c, "清除路由覆盖失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"usingDb": false})
}
