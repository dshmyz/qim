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
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
)

func CreateChannel(c *gin.Context) {
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

	var req struct {
		Name              string `json:"name" binding:"required"`
		Description       string `json:"description"`
		Avatar            string `json:"avatar"`
		PublishPermission string `json:"publish_permission"`
		CommentPermission string `json:"comment_permission"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	publishPermission := req.PublishPermission
	if publishPermission == "" {
		publishPermission = "creator_only"
	}
	if publishPermission != "creator_only" && publishPermission != "all_subscribers" {
		response.BadRequest(c, "无效的发布权限")
		return
	}

	commentPermission := req.CommentPermission
	if commentPermission == "" {
		commentPermission = "all_subscribers"
	}
	if commentPermission != "all_subscribers" && commentPermission != "disabled" {
		response.BadRequest(c, "无效的评论权限")
		return
	}

	db := database.GetDB()

	// 查询创建者信息用于审批记录快照
	var creator model.User
	db.Select("nickname", "avatar", "username").First(&creator, "id = ?", userID)
	creatorName := creator.Nickname
	if creatorName == "" {
		creatorName = creator.Username
	}

	needsApproval := di.GlobalContainer.ApprovalService.IsApprovalEnabled(model.ApprovalTypeChannel)

	// 需要审批时频道状态为 pending，否则直接 active
	channelStatus := model.ChannelStatusActive
	if needsApproval {
		channelStatus = model.ChannelStatusPending
	}

	channel := model.Channel{
		Name:              req.Name,
		Description:       req.Description,
		Avatar:            req.Avatar,
		CreatorID:         userID,
		Status:            channelStatus,
		PublishPermission: publishPermission,
		CommentPermission: commentPermission,
	}

	tx := db.Begin()

	if err := tx.Create(&channel).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建频道失败")
		return
	}

	// 需要审批时创建审批记录
	approvalStatus := model.ApprovalStatusApproved
	if needsApproval {
		approval := model.Approval{
			TargetType:        model.ApprovalTypeChannel,
			TargetID:          channel.ID,
			Status:            model.ApprovalStatusPending,
			AppliedAt:         time.Now(),
			AppliedBy:         userID,
			TargetName:        channel.Name,
			TargetDescription: channel.Description,
			CreatorName:       creatorName,
			CreatorAvatar:     creator.Avatar,
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

	db.Preload("Creator").First(&channel, channel.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":              channel.ID,
			"name":            channel.Name,
			"description":     channel.Description,
			"avatar":          channel.Avatar,
			"creator_id":      channel.CreatorID,
			"status":          channel.Status,
			"approval_status": approvalStatus,
			"creator":         channel.Creator,
		},
	})
}

func GetChannels(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)
	db := database.GetDB()

	// pending/rejected 频道仅创建者可见，active 频道所有人可见
	var channels []model.Channel
	db.Preload("Creator").
		Where("status = ? OR creator_id = ?", model.ChannelStatusActive, uid).
		Find(&channels)

	var subscriptions []model.ChannelSubscriber
	db.Where("user_id = ?", userID).Find(&subscriptions)

	subscribedMap := make(map[uint]bool)
	for _, sub := range subscriptions {
		subscribedMap[sub.ChannelID] = true
	}

	// 懒订阅兜底：为当前用户补订阅未订阅的默认频道（覆盖钩子上线前的存量用户）
	for _, channel := range channels {
		if channel.IsDefault && !subscribedMap[channel.ID] {
			db.Where("channel_id = ? AND user_id = ?", channel.ID, userID).
				FirstOrCreate(&model.ChannelSubscriber{
					ChannelID: channel.ID,
					UserID:    userID.(uint),
					JoinedAt:  time.Now(),
				})
			subscribedMap[channel.ID] = true
		}
	}

	type ChannelWithSubscription struct {
		model.Channel
		IsSubscribed bool `json:"is_subscribed"`
	}

	var channelsWithSubscription []ChannelWithSubscription
	for _, channel := range channels {
		channelsWithSubscription = append(channelsWithSubscription, ChannelWithSubscription{
			Channel:      channel,
			IsSubscribed: subscribedMap[channel.ID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": channelsWithSubscription,
	})
}

func SubscribeChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	channelIDStr := c.Param("id")

	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的频道ID")
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		response.NotFound(c, "频道不存在")
		return
	}

	var existingSubscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&existingSubscription).Error; err == nil {
		response.SuccessWithMessage(c, "已经订阅该频道", nil)
		return
	}

	subscription := model.ChannelSubscriber{
		ChannelID: uint(channelID),
		UserID:    userID.(uint),
		JoinedAt:  time.Now(),
	}

	if err := db.Create(&subscription).Error; err != nil {
		response.InternalServerError(c, "订阅频道失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "订阅频道成功",
		"data":    subscription,
	})
}

func UnsubscribeChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	channelIDStr := c.Param("id")

	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的频道ID")
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err == nil && channel.IsDefault {
		response.BadRequest(c, "默认频道不可取消订阅")
		return
	}

	var subscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
		response.BadRequest(c, "未订阅该频道")
		return
	}

	if err := db.Delete(&subscription).Error; err != nil {
		response.InternalServerError(c, "取消订阅失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "取消订阅成功",
	})
}

func CreateChannelMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	channelIDStr := c.Param("id")

	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的频道ID")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
		Type    string `json:"type" default:"text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		response.NotFound(c, "频道不存在")
		return
	}

	// 频道状态校验：仅 active 频道可发布消息（审批流程的关键保护层）
	if err := di.GlobalContainer.ChannelService.EnsureChannelUsable(&channel); err != nil {
		response.Forbidden(c, err.Error())
		return
	}

	if channel.CreatorID != userID.(uint) {
		// 系统管理员/频道管理员可在任意频道发布消息（不受 creator_only 限制）
		roles, _ := di.GlobalContainer.UserService.GetUserRoles(userID.(uint))
		isAdmin := false
		for _, r := range roles {
			if r == "system_admin" || r == "channel_manager" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			if channel.PublishPermission == "creator_only" {
				response.Forbidden(c, "无权限发布消息，仅频道创建者可发布")
				return
			}
			var subscription model.ChannelSubscriber
			if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
				response.Forbidden(c, "无权限发布消息，需先订阅该频道")
				return
			}
		}
	}

	channelMessage := model.ChannelMessage{
		ChannelID: uint(channelID),
		SenderID:  userID.(uint),
		Content:   req.Content,
		Type:      req.Type,
	}

	if err := db.Create(&channelMessage).Error; err != nil {
		response.InternalServerError(c, "发布消息失败")
		return
	}

	db.Preload("Sender").First(&channelMessage, channelMessage.ID)

	var subscribers []model.ChannelSubscriber
	db.Where("channel_id = ?", uint(channelID)).Find(&subscribers)

	for _, subscriber := range subscribers {
		payload, _ := json.Marshal(map[string]interface{}{
			"channel_id": channel.ID,
			"channel_name": channel.Name,
		})
		notification := model.Notification{
			UserID:         subscriber.UserID,
			Type:           "channel_message",
			Title:          fmt.Sprintf("频道消息: %s", channel.Name),
			Content:        req.Content,
			ActionType:     "channel_message",
			ActionPayload:  string(payload),
		}
		db.Create(&notification)

		if ws.GlobalHub != nil {
			notificationMsg := ws.WSMessage{
				Type: "notification",
				Data: notification,
			}
			jsonMsg, _ := json.Marshal(notificationMsg)
			ws.GlobalHub.SendToUser(subscriber.UserID, jsonMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": channelMessage,
	})
}

func GetChannelMessages(c *gin.Context) {
	channelIDStr := c.Param("id")

	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的频道ID")
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		response.NotFound(c, "频道不存在")
		return
	}

	var messages []model.ChannelMessage
	db.Where("channel_id = ?", uint(channelID)).Preload("Sender").Order("created_at DESC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": messages,
	})
}

func LikeChannelMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	messageIDStr := c.Param("messageId")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	db := database.GetDB()

	var msg model.ChannelMessage
	if err := db.First(&msg, uint(messageID)).Error; err != nil {
		response.NotFound(c, "消息不存在")
		return
	}

	var existing model.ChannelMessageLike
	if err := db.Where("message_id = ? AND user_id = ?", uint(messageID), uid).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已点赞"})
		return
	}

	like := model.ChannelMessageLike{
		MessageID: uint(messageID),
		UserID:    uid,
	}
	if err := db.Create(&like).Error; err != nil {
		response.InternalServerError(c, "点赞失败")
		return
	}

	var count int64
	db.Model(&model.ChannelMessageLike{}).Where("message_id = ?", uint(messageID)).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"like_count": count,
			"is_liked":   true,
		},
	})
}

func UnlikeChannelMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	messageIDStr := c.Param("messageId")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	db := database.GetDB()

	if err := db.Where("message_id = ? AND user_id = ?", uint(messageID), uid).Delete(&model.ChannelMessageLike{}).Error; err != nil {
		response.InternalServerError(c, "取消点赞失败")
		return
	}

	var count int64
	db.Model(&model.ChannelMessageLike{}).Where("message_id = ?", uint(messageID)).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"like_count": count,
			"is_liked":   false,
		},
	})
}

func CommentChannelMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	messageIDStr := c.Param("messageId")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "评论内容不能为空")
		return
	}

	db := database.GetDB()

	var msg model.ChannelMessage
	if err := db.First(&msg, uint(messageID)).Error; err != nil {
		response.NotFound(c, "消息不存在")
		return
	}

	comment := model.ChannelMessageComment{
		MessageID: uint(messageID),
		UserID:    uid,
		Content:   req.Content,
	}
	if err := db.Create(&comment).Error; err != nil {
		response.InternalServerError(c, "评论失败")
		return
	}

	db.Preload("User").First(&comment, comment.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": comment,
	})
}

func GetChannelMessageComments(c *gin.Context) {
	messageIDStr := c.Param("messageId")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	db := database.GetDB()

	var comments []model.ChannelMessageComment
	db.Where("message_id = ?", uint(messageID)).Preload("User").Order("created_at ASC").Find(&comments)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": comments,
	})
}

func GetChannelMessageLikes(c *gin.Context) {
	messageIDStr := c.Param("messageId")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	db := database.GetDB()

	var count int64
	db.Model(&model.ChannelMessageLike{}).Where("message_id = ?", uint(messageID)).Count(&count)

	var existing model.ChannelMessageLike
	isLiked := db.Where("message_id = ? AND user_id = ?", uint(messageID), uid).First(&existing).Error == nil

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"like_count": count,
			"is_liked":   isLiked,
		},
	})
}

// UpdateMyChannel 更新我的频道（仅允许 pending/rejected 状态）。
// rejected 频道修改后自动重新提交审批。
func UpdateMyChannel(c *gin.Context) {
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

	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的频道ID")
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.Where("id = ? AND creator_id = ?", uint(channelID), userID).First(&channel).Error; err != nil {
		response.NotFound(c, "频道不存在或无权操作")
		return
	}

	// 仅允许 pending/rejected 状态的频道编辑
	if channel.Status != model.ChannelStatusPending && channel.Status != model.ChannelStatusRejected {
		response.BadRequest(c, "仅可编辑审批中或已拒绝的频道")
		return
	}

	var req struct {
		Name              *string `json:"name"`
		Description       *string `json:"description"`
		Avatar            *string `json:"avatar"`
		PublishPermission *string `json:"publish_permission"`
		CommentPermission *string `json:"comment_permission"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.PublishPermission != nil {
		if *req.PublishPermission != "creator_only" && *req.PublishPermission != "all_subscribers" {
			response.BadRequest(c, "无效的发布权限")
			return
		}
		updates["publish_permission"] = *req.PublishPermission
	}
	if req.CommentPermission != nil {
		if *req.CommentPermission != "all_subscribers" && *req.CommentPermission != "disabled" {
			response.BadRequest(c, "无效的评论权限")
			return
		}
		updates["comment_permission"] = *req.CommentPermission
	}

	// rejected 频道修改后重新提交审批：状态改回 pending + 重置审批记录
	approvalStatus := channel.Status
	if channel.Status == model.ChannelStatusRejected {
		updates["status"] = model.ChannelStatusPending
		approvalStatus = model.ApprovalStatusPending
	}

	if err := db.Model(&channel).Updates(updates).Error; err != nil {
		response.InternalServerError(c, "更新频道失败")
		return
	}

	// rejected → 重新申请：重置审批记录为 pending
	if channel.Status == model.ChannelStatusRejected {
		var approval model.Approval
		if err := db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeChannel, channel.ID).
			First(&approval).Error; err == nil {
			// 更新审批记录的快照字段
			db.Model(&approval).Updates(map[string]interface{}{
				"status":            model.ApprovalStatusPending,
				"target_name":       updates["name"],
				"target_description": updates["description"],
				"reject_reason":     "",
				"approved_at":       nil,
				"approved_by":       nil,
			})
		}
	}

	// 重新加载
	db.Preload("Creator").First(&channel, channel.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":              channel.ID,
			"name":            channel.Name,
			"description":     channel.Description,
			"avatar":          channel.Avatar,
			"status":          channel.Status,
			"approval_status": approvalStatus,
		},
	})
}
