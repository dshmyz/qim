package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/ws"

	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// AvatarTask 分身任务
type AvatarTask struct {
	UserID         uint
	ConversationID uint
	TriggerMessage string
	TriggerUserID  uint
	IsGroupChat    bool
	GroupName      string
	TriggerName    string
}

// AvatarWorkerPool 分身工作池
type AvatarWorkerPool struct {
	queue        chan AvatarTask
	workers      int
	limiter      *rate.Limiter
	userLimiters sync.Map
	service      *AvatarService
	db           *gorm.DB
}

// NewAvatarWorkerPool 创建分身工作池
func NewAvatarWorkerPool(workers int, globalRPM int, service *AvatarService) *AvatarWorkerPool {
	pool := &AvatarWorkerPool{
		queue:   make(chan AvatarTask, 100),
		workers: workers,
		limiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(globalRPM)), globalRPM),
		service: service,
		db:      service.db,
	}

	for i := 0; i < workers; i++ {
		go pool.run()
	}

	return pool
}

// Submit 提交任务
func (p *AvatarWorkerPool) Submit(task AvatarTask) error {
	select {
	case p.queue <- task:
		return nil
	default:
		return fmt.Errorf("队列已满，请稍后重试")
	}
}

// run 运行工作协程
func (p *AvatarWorkerPool) run() {
	for task := range p.queue {
		p.process(task)
	}
}

// process 处理任务
func (p *AvatarWorkerPool) process(task AvatarTask) {
	ctx := context.Background()

	logger.WithModule("AvatarWorkerPool").Info("开始处理分身任务", "userID", task.UserID, "convID", task.ConversationID, "triggerUserID", task.TriggerUserID)

	if err := p.limiter.Wait(ctx); err != nil {
		logger.WithModule("AvatarWorkerPool").Error("全局限流等待失败", "userID", task.UserID, "error", err)
		return
	}

	userLimiter := p.getUserLimiter(task.UserID)
	if err := userLimiter.Wait(ctx); err != nil {
		logger.WithModule("AvatarWorkerPool").Error("用户限流等待失败", "userID", task.UserID, "error", err)
		return
	}

	var session model.AvatarSession
	err := p.db.Where("user_id = ? AND conversation_id = ?", task.UserID, task.ConversationID).First(&session).Error
	if err == nil && session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		logger.WithModule("AvatarWorkerPool").Info("分身接管期内，跳过回复", "userID", task.UserID, "takeoverUntil", session.TakeoverUntil)
		return
	}

	reply, err := p.service.GenerateReply(task.UserID, task.ConversationID, task.TriggerMessage)
	if err != nil {
		logger.WithModule("AvatarWorkerPool").Error("分身回复生成失败", "user", task.UserID, "conv", task.ConversationID, "error", err)
		return
	}

	// 空回复表示分身选择不回复（如知识范围外且配置为不回复）
	if reply == "" {
		logger.WithModule("AvatarWorkerPool").Info("分身选择不回复", "user", task.UserID, "conv", task.ConversationID)
		return
	}

	// 一次性查询分身用户信息和自定义名称，后续发送函数复用
	var avatarUser model.User
	if err := p.db.First(&avatarUser, task.UserID).Error; err != nil {
		logger.WithModule("AvatarWorkerPool").Error("获取分身用户信息失败", "user", task.UserID, "error", err)
		return
	}
	avatarCfgName := ""
	var avatarConfig model.AvatarConfig
	if p.db.Where("user_id = ?", task.UserID).First(&avatarConfig).Error == nil && avatarConfig.Name != "" {
		avatarCfgName = avatarConfig.Name
	}

	if task.IsGroupChat {
		p.sendPrivateReply(task, reply, &avatarUser, avatarCfgName)
	} else {
		p.sendDirectReply(task, reply, &avatarUser, avatarCfgName)
	}

	now := time.Now()
	p.db.Model(&session).Update("last_reply_at", now)
}

// getUserLimiter 获取用户级别的限流器
func (p *AvatarWorkerPool) getUserLimiter(userID uint) *rate.Limiter {
	limiterAny, _ := p.userLimiters.LoadOrStore(userID, rate.NewLimiter(rate.Every(time.Minute/10), 10))
	return limiterAny.(*rate.Limiter)
}

// sendPrivateReply 发送私聊回复（群聊场景）
func (p *AvatarWorkerPool) sendPrivateReply(task AvatarTask, reply string, avatarUser *model.User, avatarCfgName string) {
	// 1. 找到或创建分身用户与触发者的私聊会话
	convService := NewConversationService(database.GetDB())
	conv, err := convService.CreateSingleConversation(task.UserID, task.TriggerUserID)
	if err != nil {
		logger.WithModule("AvatarWorkerPool").Error("创建私聊会话失败",
			"user", task.UserID, "trigger", task.TriggerUserID, "error", err)
		return
	}

	// 2. 创建消息
	msg := model.Message{
		ConversationID: conv.ID,
		SenderID:       task.UserID,
		Type:           "text",
		Content:        reply,
		IsRead:         false,
		AIType:         "avatar",
	}

	if err := p.db.Create(&msg).Error; err != nil {
		logger.WithModule("AvatarWorkerPool").Error("保存分身消息失败", "conv", conv.ID, "error", err)
		return
	}

	// 5. 预加载发送者信息
	p.db.Preload("Sender").First(&msg, msg.ID)

	// 6. 更新会话最后消息
	now := time.Now()
	p.db.Model(&model.Conversation{}).Where("id = ?", conv.ID).Updates(map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	})

	// 7. 增加触发者的未读数
	p.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", conv.ID, task.UserID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

	// 8. 广播消息到私聊会话
	responseData := map[string]interface{}{
		"id":              msg.ID,
		"conversation_id": msg.ConversationID,
		"sender_id":       msg.SenderID,
		"type":            msg.Type,
		"content":         msg.Content,
		"is_read":         msg.IsRead,
		"created_at":      msg.CreatedAt,
		"is_avatar_reply": msg.AIType == "avatar",
		"ai_type":         msg.AIType,
		"sender":          msg.Sender,
		"avatar_name":     avatarCfgName,
	}

	if ws.GlobalHub != nil {
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		jsonMsg, _ := json.Marshal(wsMsg)
		logger.WithModule("sendPrivateReply").Debug("Broadcasting",
			"conv", conv.ID, "ai_type", msg.AIType, "sender_id", msg.SenderID, "sender_name", msg.Sender.Nickname)
		ws.GlobalHub.SendToConversation(conv.ID, 0, jsonMsg)
	}

	logger.WithModule("AvatarWorkerPool").Info("分身私聊回复已发送", "conv", conv.ID, "msgID", msg.ID)
}

// sendDirectReply 发送直接回复（私聊场景）
func (p *AvatarWorkerPool) sendDirectReply(task AvatarTask, reply string, avatarUser *model.User, avatarCfgName string) {
	// 1. 创建消息
	msg := model.Message{
		ConversationID: task.ConversationID,
		SenderID:       task.UserID,
		Type:           "text",
		Content:        reply,
		IsRead:         false,
		AIType:         "avatar",
	}

	if err := p.db.Create(&msg).Error; err != nil {
		logger.WithModule("AvatarWorkerPool").Error("保存分身消息失败", "conv", task.ConversationID, "error", err)
		return
	}

	// 4. 预加载发送者信息
	p.db.Preload("Sender").First(&msg, msg.ID)

	// 5. 更新会话最后消息
	now := time.Now()
	p.db.Model(&model.Conversation{}).Where("id = ?", task.ConversationID).Updates(map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	})

	// 6. 增加其他成员的未读数
	p.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", task.ConversationID, task.UserID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

	// 7. 广播消息到当前会话
	responseData := map[string]interface{}{
		"id":              msg.ID,
		"conversation_id": msg.ConversationID,
		"sender_id":       msg.SenderID,
		"type":            msg.Type,
		"content":         msg.Content,
		"is_read":         msg.IsRead,
		"created_at":      msg.CreatedAt,
		"is_avatar_reply": msg.AIType == "avatar",
		"ai_type":         msg.AIType,
		"sender":          msg.Sender,
		"avatar_name":     avatarCfgName,
	}

	if ws.GlobalHub != nil {
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		jsonMsg, _ := json.Marshal(wsMsg)
		logger.WithModule("sendDirectReply").Debug("Broadcasting",
			"conv", task.ConversationID, "ai_type", msg.AIType, "sender_id", msg.SenderID, "sender_name", msg.Sender.Nickname)
		ws.GlobalHub.SendToConversation(task.ConversationID, 0, jsonMsg)
	}

	logger.WithModule("AvatarWorkerPool").Info("分身直接回复已发送", "conv", task.ConversationID, "msgID", msg.ID)
}