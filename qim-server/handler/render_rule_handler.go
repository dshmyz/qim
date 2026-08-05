package handler

import (
	"net/http"
	"strconv"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/errors"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
)

// GetRenderRules 客户端拉取渲染规则（带版本号增量同步）
// GET /api/v1/render-rules?version=xxx
func GetRenderRules(c *gin.Context) {
	svc := di.GlobalContainer.RenderRuleService

	clientVersion, _ := strconv.ParseInt(c.Query("version"), 10, 64)
	serverVersion, err := svc.GetVersion()
	if err != nil {
		response.InternalServerError(c, "获取版本失败")
		return
	}

	// 版本一致，返回 304
	if clientVersion == serverVersion && clientVersion > 0 {
		c.JSON(http.StatusNotModified, gin.H{})
		return
	}

	rules, err := svc.GetRules()
	if err != nil {
		response.InternalServerError(c, "获取规则失败")
		return
	}

	// 只返回启用的规则
	enabled := make([]service.RenderRule, 0, len(rules))
	for _, r := range rules {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}

	// 兼容性响应：同时返回新旧两种形状的渲染规则。
	//  - data.rules / data.version：新客户端（renderRules.ts 读取 res.data.rules）
	//  - 顶层 rules / version：旧客户端（历史版本 renderRules.ts 误读顶层 res.rules）
	// 统一响应封装 response.Success 固定 {code,data,message}，无法在顶层平铺字段，
	// 故此处手动构造等形结构（使用同一 code 常量），仅影响 /render-rules 单端点，不影响全局约定。
	//
	// TODO(临时兼容): 2026-10-01 后移除顶层平铺的 rules/version 字段。
	// 届时老客户端（不再误读顶层字段的版本）已全部升级，只需保留 data 一种形状，
	// 恢复为 response.Success(c, gin.H{"rules": enabled, "version": serverVersion})。
	c.JSON(http.StatusOK, gin.H{
		"code":    errors.ErrCodeSuccess,
		"message": "success",
		"data": gin.H{
			"rules":   enabled,
			"version": serverVersion,
		},
		"rules":   enabled,
		"version": serverVersion,
	})
}

// AdminGetRenderRules 管理后台用：获取全部规则（含禁用的）
// GET /api/v1/admin/render-rules
func AdminGetRenderRules(c *gin.Context) {
	svc := di.GlobalContainer.RenderRuleService
	rules, err := svc.GetAllRules()
	if err != nil {
		response.InternalServerError(c, "获取规则失败")
		return
	}
	version, _ := svc.GetVersion()
	response.Success(c, gin.H{
		"rules":   rules,
		"version": version,
	})
}

// AdminSaveRenderRules 管理后台用：批量覆盖保存规则
// PUT /api/v1/admin/render-rules
func AdminSaveRenderRules(c *gin.Context) {
	var req struct {
		Rules []service.RenderRule `json:"rules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	svc := di.GlobalContainer.RenderRuleService
	if err := svc.SaveRules(req.Rules); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.SuccessWithMessage(c, "保存成功", nil)
}

// AdminTestRenderRule 管理后台用：测试单条规则在样例文本上的匹配效果
// POST /api/v1/admin/render-rules/test
func AdminTestRenderRule(c *gin.Context) {
	var req struct {
		Rule       service.RenderRule `json:"rule"`
		SampleText string             `json:"sample_text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	svc := di.GlobalContainer.RenderRuleService
	results, err := svc.TestRule(req.Rule, req.SampleText)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"results": results})
}
