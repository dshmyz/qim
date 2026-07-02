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
	userID, _ := c.Get("user_id")

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

	channel := model.Channel{
		Name:              req.Name,
		Description:       req.Description,
		Avatar:            req.Avatar,
		CreatorID:         userID.(uint),
		Status:            "active",
		PublishPermission: publishPermission,
		CommentPermission: commentPermission,
	}

	if err := db.Create(&channel).Error; err != nil {
		response.InternalServerError(c, "创建频道失败")
		return
	}

	db.Preload("Creator").First(&channel, channel.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": channel,
	})
}

func GetChannels(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var channels []model.Channel
	db.Preload("Creator").Find(&channels)

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

	if channel.CreatorID != userID.(uint) {
		// 系统管理员可在任意频道发布消息（不受 creator_only 限制）
		roles, _ := di.GlobalContainer.UserService.GetUserRoles(userID.(uint))
		isAdmin := false
		for _, r := range roles {
			if r == "system_admin" {
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
		notification := model.Notification{
			UserID:  subscriber.UserID,
			Type:    "channel_message",
			Title:   fmt.Sprintf("频道消息: %s", channel.Name),
			Content: req.Content,
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
