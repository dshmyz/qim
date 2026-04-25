package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

func CreateChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	channel := model.Channel{
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		CreatorID:   userID.(uint),
		Status:      "active",
	}

	if err := db.Create(&channel).Error; err != nil {
		response.InternalServerError(c, "创建频道失败")
		return
	}

	db.Preload("Creator").First(&channel, channel.ID)

	response.Success(c, channel)
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

	response.Success(c, channelsWithSubscription)
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
		response.Success(c, gin.H{"message": "已经订阅该频道"})
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

	response.Success(c, gin.H{
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

	var subscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
		response.BadRequest(c, "未订阅该频道")
		return
	}

	if err := db.Delete(&subscription).Error; err != nil {
		response.InternalServerError(c, "取消订阅失败")
		return
	}

	response.Success(c, gin.H{
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
		response.Forbidden(c, "无权限发布消息")
		return
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

	response.Success(c, channelMessage)
}

func GetChannelMessages(c *gin.Context) {
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

	var subscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
		response.Forbidden(c, "无权限访问")
		return
	}

	var messages []model.ChannelMessage
	db.Where("channel_id = ?", uint(channelID)).Preload("Sender").Order("created_at DESC").Find(&messages)

	response.Success(c, messages)
}
