package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"qim-server/model"
	"qim-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AlertHandler struct {
	db *gorm.DB
}

func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{db: db}
}

func (h *AlertHandler) GetAlertRules(c *gin.Context) {
	var rules []model.AlertRule
	if err := h.db.Order("created_at desc").Find(&rules).Error; err != nil {
		logger.WithModule("alert").Error("获取告警规则失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取告警规则失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": rules,
	})
}

func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	var req struct {
		Name          string   `json:"name" binding:"required"`
		Metric        string   `json:"metric" binding:"required"`
		Condition     string   `json:"condition" binding:"required"`
		Threshold     float64  `json:"threshold" binding:"required"`
		Duration      int      `json:"duration" binding:"required"`
		NotifyMethods []string `json:"notifyMethods"`
		NotifyTargets []string `json:"notifyTargets"`
		Enabled       bool     `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	notifyMethods, _ := json.Marshal(req.NotifyMethods)
	notifyTargets, _ := json.Marshal(req.NotifyTargets)

	rule := model.AlertRule{
		Name:          req.Name,
		Metric:        req.Metric,
		Condition:     req.Condition,
		Threshold:     req.Threshold,
		Duration:      req.Duration,
		NotifyMethods: string(notifyMethods),
		NotifyTargets: string(notifyTargets),
		Enabled:       req.Enabled,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.db.Create(&rule).Error; err != nil {
		logger.WithModule("alert").Error("创建告警规则失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建告警规则失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    rule,
	})
}

func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	var req struct {
		Name          *string   `json:"name"`
		Metric        *string   `json:"metric"`
		Condition     *string   `json:"condition"`
		Threshold     *float64  `json:"threshold"`
		Duration      *int      `json:"duration"`
		NotifyMethods *[]string `json:"notifyMethods"`
		NotifyTargets *[]string `json:"notifyTargets"`
		Enabled       *bool     `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var rule model.AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "告警规则不存在",
		})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Metric != nil {
		updates["metric"] = *req.Metric
	}
	if req.Condition != nil {
		updates["condition"] = *req.Condition
	}
	if req.Threshold != nil {
		updates["threshold"] = *req.Threshold
	}
	if req.Duration != nil {
		updates["duration"] = *req.Duration
	}
	if req.NotifyMethods != nil {
		notifyMethods, _ := json.Marshal(*req.NotifyMethods)
		updates["notify_methods"] = string(notifyMethods)
	}
	if req.NotifyTargets != nil {
		notifyTargets, _ := json.Marshal(*req.NotifyTargets)
		updates["notify_targets"] = string(notifyTargets)
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	updates["updated_at"] = time.Now()

	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		logger.WithModule("alert").Error("更新告警规则失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新告警规则失败",
		})
		return
	}

	h.db.First(&rule, id)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    rule,
	})
}

func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	if err := h.db.Delete(&model.AlertRule{}, id).Error; err != nil {
		logger.WithModule("alert").Error("删除告警规则失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除告警规则失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

func (h *AlertHandler) GetAlertHistory(c *gin.Context) {
	ruleID := c.Query("ruleId")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var history []model.AlertHistory
	var total int64

	query := h.db.Model(&model.AlertHistory{}).Preload("AlertRule")

	if ruleID != "" {
		query = query.Where("rule_id = ?", ruleID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&history).Error; err != nil {
		logger.WithModule("alert").Error("获取告警历史失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取告警历史失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     history,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
