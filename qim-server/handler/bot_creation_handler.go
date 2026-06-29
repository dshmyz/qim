package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateBotRequest 创建 Bot 请求体
type CreateBotRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Description    string                 `json:"description" binding:"required"`
	Type           string                 `json:"type" binding:"required"` // assistant, custom
	Provider       string                 `json:"provider"`
	CustomModelURL string                 `json:"custom_model_url"`
	LobsterURL     string                 `json:"lobster_url"`
	Avatar         string                 `json:"avatar"`
	IsTemplate     bool                   `json:"is_template"`
	Config         map[string]interface{} `json:"config"`
}

// GetMyBots 获取我创建的 Bot 列表
func GetMyBots(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var bots []model.Bot
	if err := db.Where("creator_id = ?", userID).Order("created_at DESC").Find(&bots).Error; err != nil {
		response.InternalServerError(c, "获取 Bot 列表失败")
		return
	}

	// 组装包含审批状态的响应
	type BotWithApproval struct {
		model.Bot
		ApprovalStatus string `json:"approval_status"`
	}

	result := make([]BotWithApproval, 0, len(bots))
	for _, bot := range bots {
		approvalStatus := "approved"
		if !bot.IsActive {
			// 查询最新的审批记录（按时间倒序）
			var approval model.Approval
			err := db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeBot, bot.ID).
				Order("created_at DESC").
				First(&approval).Error

			if err == nil {
				// 如果最新审批是 approved，但 bot 未激活，说明是管理员手动停用
				if approval.Status == model.ApprovalStatusApproved {
					approvalStatus = "inactive"
				} else {
					approvalStatus = string(approval.Status)
				}
			} else {
				// 找不到审批记录，可能是历史数据或管理员直接停用
				approvalStatus = "inactive"
			}
		}
		result = append(result, BotWithApproval{
			Bot:            bot,
			ApprovalStatus: approvalStatus,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// GetMyBotCount 获取我已创建的 Bot 数量
func GetMyBotCount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var count int64
	if err := db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ?", userID, []string{model.BotTypeCustom, model.BotTypeAssistant}).Count(&count).Error; err != nil {
		response.InternalServerError(c, "获取数量失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"count": count},
	})
}

// GetTemplates 获取模板 Bot 列表
func GetTemplates(c *gin.Context) {
	db := database.GetDB()

	var bots []model.Bot
	if err := db.Where("is_template = ? AND is_active = ?", true, true).Find(&bots).Error; err != nil {
		response.InternalServerError(c, "获取模板列表失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

// CreateBot 创建 Bot
func CreateBot(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		response.InternalServerError(c, "用户信息错误")
		return
	}
	db := database.GetDB()

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	// 校验 Type 字段
	if req.Type != model.BotTypeAssistant && req.Type != model.BotTypeCustom {
		response.BadRequest(c, "Bot 类型无效")
		return
	}

	// 普通用户不能创建模板 Bot，强制设置为 false
	req.IsTemplate = false

	// 检查用户是否已达到创建上限
	var count int64
	db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ?", userID, []string{model.BotTypeCustom, model.BotTypeAssistant}).Count(&count)
	if count >= getMaxBotsPerUser(db) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "已达到创建上限，请联系管理员",
		})
		return
	}

	// 构建 Config JSON
	configJSON := []byte("{}")
	if req.Config != nil {
		var err error
		configJSON, err = json.Marshal(req.Config)
		if err != nil {
			response.BadRequest(c, "配置格式错误")
			return
		}
	}

	var creator model.User
	db.Select("nickname").First(&creator, "id = ?", userID)
	creatorName := creator.Nickname
	if creatorName == "" {
		creatorName = creator.Username
	}

	needsApproval := di.GlobalContainer.ApprovalService.IsApprovalEnabled(model.ApprovalTypeBot)

	bot := model.Bot{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Avatar:      req.Avatar,
		Config:      string(configJSON),
		IsActive:    !needsApproval, // 需要审批时默认禁用，否则直接启用
		CreatorID:   userID,
		CreatorName: creatorName,
		IsTemplate:  false,
	}

	// 开启事务
	tx := db.Begin()

	// 创建 Bot
	if err := tx.Create(&bot).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建 Bot 失败")
		return
	}

	// 创建虚拟用户
	virtualUser := model.User{
		Username: fmt.Sprintf("bot_%d", bot.ID),
		Nickname: bot.Name,
		Avatar:   bot.Avatar,
		Type:     "bot",
	}
	if err := tx.Create(&virtualUser).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建虚拟用户失败")
		return
	}

	// 更新 Bot 的 VirtualUserID
	bot.VirtualUserID = &virtualUser.ID
	if err := tx.Save(&bot).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "更新 Bot 失败")
		return
	}

	// 需要审批时创建审批记录
	approvalStatus := model.ApprovalStatusApproved
	if needsApproval {
		approval := model.Approval{
			TargetType: model.ApprovalTypeBot,
			TargetID:   bot.ID,
			Status:     model.ApprovalStatusPending,
			AppliedAt:  time.Now(),
			AppliedBy:  userID,
		}
		if err := tx.Create(&approval).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建审批记录失败")
			return
		}
		approvalStatus = model.ApprovalStatusPending
	}

	if err := tx.Commit().Error; err != nil {
		response.InternalServerError(c, "提交事务失败")
		return
	}

	di.GlobalContainer.OperationLogService.LogUserOperation(c, "bot", "create_bot")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":              bot.ID,
			"name":            bot.Name,
			"virtual_user_id": virtualUser.ID,
			"approval_status": approvalStatus,
		},
	})
}

// UpdateMyBot 更新我的 Bot（仅允许待审批和已拒绝状态）
func UpdateMyBot(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		response.InternalServerError(c, "用户信息错误")
		return
	}

	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 Bot ID")
		return
	}

	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND creator_id = ?", uint(botID), userID).First(&bot).Error; err != nil {
		response.NotFound(c, "Bot 不存在或无权操作")
		return
	}

	// 从approvals表获取审批状态
	var approval model.Approval
	approvalStatus := model.ApprovalStatusNone
	if err := db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeBot, bot.ID).First(&approval).Error; err == nil {
		approvalStatus = approval.Status
	}

	// 仅允许待审批和已拒绝状态的 Bot 编辑
	if approvalStatus != model.ApprovalStatusPending && approvalStatus != model.ApprovalStatusRejected {
		response.BadRequest(c, "仅可编辑待审批或已拒绝的 Bot")
		return
	}

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	// 校验 Type 字段
	if req.Type != model.BotTypeAssistant && req.Type != model.BotTypeCustom {
		response.BadRequest(c, "Bot 类型无效")
		return
	}

	configJSON := []byte("{}")
	if req.Config != nil {
		var err error
		configJSON, err = json.Marshal(req.Config)
		if err != nil {
			response.BadRequest(c, "配置格式错误")
			return
		}
	}

	if err := db.Model(&bot).Updates(map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"avatar":      req.Avatar,
		"config":      string(configJSON),
	}).Error; err != nil {
		response.InternalServerError(c, "更新 Bot 失败")
		return
	}

	// 重新加载获取最新数据
	db.First(&bot, bot.ID)

	di.GlobalContainer.OperationLogService.LogUserOperation(c, "bot", "update_bot")

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": bot})
}

// DeleteMyBot 删除我的 Bot
func DeleteMyBot(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		response.InternalServerError(c, "用户信息错误")
		return
	}

	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 Bot ID")
		return
	}

	db := database.GetDB()

	result := db.Where("id = ? AND creator_id = ?", uint(botID), userID).Delete(&model.Bot{})
	if result.Error != nil || result.RowsAffected == 0 {
		response.NotFound(c, "Bot 不存在或无权操作")
		return
	}

	di.GlobalContainer.OperationLogService.LogUserOperation(c, "bot", "delete_bot")

	response.SuccessWithMessage(c, "删除成功", nil)
}

// getMaxBotsPerUser 从系统配置获取每个用户的最大 Bot 数量
func getMaxBotsPerUser(db *gorm.DB) int64 {
	// 默认 5，后续从 system_configs 表读取
	return 5
}
