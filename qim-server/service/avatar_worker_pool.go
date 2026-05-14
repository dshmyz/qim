package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

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

	if err := p.limiter.Wait(ctx); err != nil {
		return
	}

	userLimiter := p.getUserLimiter(task.UserID)
	if err := userLimiter.Wait(ctx); err != nil {
		return
	}

	var session model.AvatarSession
	err := p.db.Where("user_id = ? AND conversation_id = ?", task.UserID, task.ConversationID).First(&session).Error
	if err == nil && session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		return
	}

	reply, err := p.service.GenerateReply(task.UserID, task.ConversationID, task.TriggerMessage)
	if err != nil {
		return
	}

	// 读取 disclaimerStyle 配置，决定是否追加免责声明
	var config model.AvatarConfig
	if err := p.db.Where("user_id = ?", task.UserID).First(&config).Error; err == nil {
		var replyStrategy model.AvatarReplyStrategy
		if config.ReplyStrategyJSON != "" {
			if err := json.Unmarshal([]byte(config.ReplyStrategyJSON), &replyStrategy); err == nil {
				if replyStrategy.DisclaimerStyle == "footer" || replyStrategy.DisclaimerStyle == "both" {
					reply = reply + "\n\n（此回复由AI分身生成）"
				}
			}
		}
	}

	if task.IsGroupChat {
		p.sendPrivateReply(task, reply)
	} else {
		p.sendDirectReply(task, reply)
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
func (p *AvatarWorkerPool) sendPrivateReply(task AvatarTask, reply string) {
	// 1. 找到或创建分身用户与触发者的私聊会话
	convService := NewConversationService(database.GetDB())
	conv, err := convService.CreateSingleConversation(task.UserID, task.TriggerUserID)
	if err != nil {
		log.Printf("[AvatarWorkerPool] 创建私聊会话失败: user=%d, trigger=%d, error=%v", 
			task.UserID, task.TriggerUserID, err)
		return
	}

	// 2. 获取分身用户的昵称
	var avatarUser model.User
	if err := p.db.First(&avatarUser, task.UserID).Error; err != nil {
		log.Printf("[AvatarWorkerPool] 获取分身用户信息失败: user=%d, error=%v", task.UserID, err)
		return
	}

	avatarName := avatarUser.Nickname
	if avatarName == "" {
		avatarName = avatarUser.Username
	}

	// 3. 构建消息内容
	content := fmt.Sprintf("[群聊 %s 中 @你] %s 的分身代为回复：%s", task.GroupName, avatarName, reply)

	// 4. 创建消息
	msg := model.Message{
		ConversationID: conv.ID,
		SenderID:       task.UserID,
		Type:           "text",
		Content:        content,
		IsRead:         false,
		IsAvatarReply:  true,
	}

	if err := p.db.Create(&msg).Error; err != nil {
		log.Printf("[AvatarWorkerPool] 保存分身消息失败: conv=%d, error=%v", conv.ID, err)
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
		"is_avatar_reply": msg.IsAvatarReply,
		"sender":          msg.Sender,
	}

	if ws.GlobalHub != nil {
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		jsonMsg, _ := json.Marshal(wsMsg)
		log.Printf("[sendPrivateReply] Broadcasting to conv %d, excludeUserID=0, is_avatar_reply=%v, sender_id=%d, sender_name=%s",
			conv.ID, msg.IsAvatarReply, msg.SenderID, msg.Sender.Nickname)
		ws.GlobalHub.SendToConversation(conv.ID, 0, jsonMsg)
	}

	log.Printf("[AvatarWorkerPool] 分身私聊回复已发送: conv=%d, msgID=%d", conv.ID, msg.ID)
}

// sendDirectReply 发送直接回复（私聊场景）
func (p *AvatarWorkerPool) sendDirectReply(task AvatarTask, reply string) {
	// 1. 获取分身用户的昵称
	var avatarUser model.User
	if err := p.db.First(&avatarUser, task.UserID).Error; err != nil {
		log.Printf("[AvatarWorkerPool] 获取分身用户信息失败: user=%d, error=%v", task.UserID, err)
		return
	}

	avatarName := avatarUser.Nickname
	if avatarName == "" {
		avatarName = avatarUser.Username
	}

	// 2. 构建消息内容
	content := fmt.Sprintf("%s 的分身代为回复：%s", avatarName, reply)

	// 3. 创建消息
	msg := model.Message{
		ConversationID: task.ConversationID,
		SenderID:       task.UserID,
		Type:           "text",
		Content:        content,
		IsRead:         false,
		IsAvatarReply:  true,
	}

	if err := p.db.Create(&msg).Error; err != nil {
		log.Printf("[AvatarWorkerPool] 保存分身消息失败: conv=%d, error=%v", task.ConversationID, err)
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
		"is_avatar_reply": msg.IsAvatarReply,
		"sender":          msg.Sender,
	}

	if ws.GlobalHub != nil {
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		jsonMsg, _ := json.Marshal(wsMsg)
		log.Printf("[sendDirectReply] Broadcasting to conv %d, excludeUserID=0, is_avatar_reply=%v, sender_id=%d, sender_name=%s",
			task.ConversationID, msg.IsAvatarReply, msg.SenderID, msg.Sender.Nickname)
		ws.GlobalHub.SendToConversation(task.ConversationID, 0, jsonMsg)
	}

	log.Printf("[AvatarWorkerPool] 分身直接回复已发送: conv=%d, msgID=%d", task.ConversationID, msg.ID)
}
