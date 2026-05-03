package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

func CreateChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name              string `json:"name" binding:"required"`
		Description       string `json:"description"`
		Avatar            string `json:"avatar"`
		PublishPermission string `json:"publish_permission"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	publishPermission := req.PublishPermission
	if publishPermission == "" {
		publishPermission = "creator_only"
	}
	if publishPermission != "creator_only" && publishPermission != "all_subscribers" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的发布权限"})
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
	}

	if err := db.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建频道失败"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	var existingSubscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&existingSubscription).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已经订阅该频道"})
		return
	}

	subscription := model.ChannelSubscriber{
		ChannelID: uint(channelID),
		UserID:    userID.(uint),
		JoinedAt:  time.Now(),
	}

	if err := db.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "订阅频道失败"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	db := database.GetDB()

	var subscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未订阅该频道"})
		return
	}

	if err := db.Delete(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消订阅失败"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
		Type    string `json:"type" default:"text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	if channel.CreatorID != userID.(uint) {
		if channel.PublishPermission == "creator_only" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发布消息，仅频道创建者可发布"})
			return
		}
		var subscription model.ChannelSubscriber
		if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发布消息，需先订阅该频道"})
			return
		}
	}

	channelMessage := model.ChannelMessage{
		ChannelID: uint(channelID),
		SenderID:  userID.(uint),
		Content:   req.Content,
		Type:      req.Type,
	}

	if err := db.Create(&channelMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发布消息失败"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	var messages []model.ChannelMessage
	db.Where("channel_id = ?", uint(channelID)).Preload("Sender").Order("created_at DESC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": messages,
	})
}
