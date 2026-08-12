package service

import (
	"encoding/json"
	"fmt"
	"strings"
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

// avatarSendMeta 分身回复发送所需的分身侧元数据。
// 将回复发送所需的非核心参数收敛为单个参数对象，避免回复发送函数参数膨胀。
type avatarSendMeta struct {
	avatarCfgName   string
	disclaimerStyle string
	sources         []KnowledgeSource
}

// AvatarWorkerPool 分身工作池
//
// 队列、worker 并发上限、全局限流、按分身用户限流等通用并发治理统一委托给
// ReplyOrchestrator；本类型只保留分身专属的处理/发送逻辑（生成、仿真人延迟、
// 私聊回弹、接管期跳过、WS 通知、来源压缩持久化等），并作为其处理闭包。
type AvatarWorkerPool struct {
	orch     *ReplyOrchestrator
	service  *AvatarService
	db       *gorm.DB
	delaySem chan struct{} // 限制延迟发送 goroutine 并发上限
}

// NewAvatarWorkerPool 创建分身工作池。
// workers 为并发 worker 数；globalRPM 为全局限流（每分钟允许的 AI 调用数）；
// 并按分身用户做更细的限流（每用户 10/min），分身不要求会话串行（仿真人频率治理优先）。
func NewAvatarWorkerPool(workers int, globalRPM int, service *AvatarService) *AvatarWorkerPool {
	pool := &AvatarWorkerPool{
		orch: NewReplyOrchestrator(ReplyOrchestratorOpts{
			Workers: workers,
			// 全局：每分钟 globalRPM 次 AI 调用
			GlobalRate: rate.NewLimiter(rate.Every(time.Minute/time.Duration(globalRPM)), globalRPM),
			// 每分身用户：10/min，防单个高分身刷屏
			PerKeyRate:  rate.Every(time.Minute / 10),
			PerKeyBurst: 10,
			Serialize:   false,
		}),
		delaySem: make(chan struct{}, 100), // 最多 100 个延迟 goroutine 驻留
		service:  service,
		db:       service.db,
	}

	return pool
}

// Close 关闭分身回复编排引擎，回收 worker goroutine。供服务优雅退出时调用。
func (p *AvatarWorkerPool) Close() {
	p.orch.Close()
}

// Submit 提交分身任务。队列满时不再立即丢弃，而是阻塞等待最多 2 秒；
// 仍失败则记 Warn 并通过 WS 通知分身主人“回复被跳过”，避免静默丢失。
// 并发治理（限并发/限流）由内部 ReplyOrchestrator 承担，processing 闭包携带完整任务。
func (p *AvatarWorkerPool) Submit(task AvatarTask) error {
	if err := p.orch.Submit(task.UserID, func() { p.process(task) }); err != nil {
		logger.WithModule("AvatarWorkerPool").Warn("分身任务入队超时，回复被跳过",
			"userID", task.UserID, "convID", task.ConversationID)
		if p.service != nil && p.service.wsNotify != nil {
			p.service.wsNotify(task.UserID, "avatar_reply_skipped", map[string]interface{}{
				"conversation_id": task.ConversationID,
				"reason":          "queue_busy",
			})
		}
		return fmt.Errorf("分身任务入队超时，回复已跳过")
	}
	return nil
}

// process 处理分身任务。由 ReplyOrchestrator 的 worker 调用，
// 全局限流与按用户限流已在编排层完成，这里只处理分身专属逻辑。
func (p *AvatarWorkerPool) process(task AvatarTask) {
	logger.WithModule("AvatarWorkerPool").Info("开始处理分身任务", "userID", task.UserID, "convID", task.ConversationID, "triggerUserID", task.TriggerUserID)

	var session model.AvatarSession
	err := p.db.Where("user_id = ? AND conversation_id = ?", task.UserID, task.ConversationID).First(&session).Error
	if err == nil && session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		logger.WithModule("AvatarWorkerPool").Info("分身接管期内，跳过回复", "userID", task.UserID, "takeoverUntil", session.TakeoverUntil)
		return
	}

	// 加载一次配置，供生成回复（透传给 prepare 复用）与发送阶段（name/strategy）共用，避免重复查询
	var avatarConfig model.AvatarConfig
	if err := p.db.Where("user_id = ?", task.UserID).First(&avatarConfig).Error; err != nil {
		logger.WithModule("AvatarWorkerPool").Error("获取分身配置失败", "user", task.UserID, "error", err)
		return
	}

	reply, sources, err := p.service.GenerateReplyWithSources(task.UserID, task.ConversationID, task.TriggerMessage, &avatarConfig)
	if err != nil {
		logger.WithModule("AvatarWorkerPool").Error("分身回复生成失败", "user", task.UserID, "conv", task.ConversationID, "error", err)
		return
	}

	// 空/纯空白回复表示分身选择不回复（如知识范围外且配置为不回复）或 AI 不可用，
	// 直接跳过发送，避免落空白气泡
	if strings.TrimSpace(reply) == "" {
		logger.WithModule("AvatarWorkerPool").Info("分身选择不回复", "user", task.UserID, "conv", task.ConversationID)
		return
	}

	avatarCfgName := ""
	if avatarConfig.Name != "" {
		avatarCfgName = avatarConfig.Name
	}

	// 解析回复策略，应用仿真人延迟
	var replyStrategy model.AvatarReplyStrategy
	if avatarConfig.ReplyStrategyJSON != "" {
		_ = json.Unmarshal([]byte(avatarConfig.ReplyStrategyJSON), &replyStrategy)
	}
	// 发送逻辑抽成闭包，便于延迟发送时异步执行而不阻塞 worker
	send := func() {
		meta := avatarSendMeta{
			avatarCfgName:   avatarCfgName,
			disclaimerStyle: replyStrategy.DisclaimerStyle,
			sources:         sources,
		}
		// 群聊默认回群内（GroupReplyTarget=private 时回触发者私聊），私聊回原会话
		if task.IsGroupChat && replyStrategy.GroupReplyTarget == "private" {
			convService := NewConversationService(database.GetDB())
			conv, err := convService.CreateSingleConversation(task.UserID, task.TriggerUserID)
			if err != nil {
				logger.WithModule("AvatarWorkerPool").Error("创建私聊会话失败",
					"user", task.UserID, "trigger", task.TriggerUserID, "error", err)
				return
			}
			p.sendReply(task, conv.ID, reply, meta)
		} else {
			p.sendReply(task, task.ConversationID, reply, meta)
		}
		now := time.Now()
		p.db.Model(&session).Update("last_reply_at", now)
	}

	// 仿真人延迟：异步等待后发送，避免长 ReplyDelay 占住 worker（池仅 N 个）饿死后续任务
	if replyStrategy.ReplyDelay > 0 {
		logger.WithModule("AvatarWorkerPool").Info("分身回复延迟发送（异步）", "userID", task.UserID, "delaySec", replyStrategy.ReplyDelay)
		delay := time.Duration(replyStrategy.ReplyDelay) * time.Second
		select {
		case p.delaySem <- struct{}{}:
			go func() {
				defer func() { <-p.delaySem }()
				time.Sleep(delay)
				send()
			}()
		default:
			// 信号量满，降级为同步发送（避免 goroutine 无限堆积）
			logger.WithModule("AvatarWorkerPool").Warn("延迟 goroutine 达上限，降级同步发送", "userID", task.UserID)
			send()
		}
	} else {
		send()
	}
}

// sendReply 把分身回复作为新消息写入指定会话并广播。
// convID 即回复落点会话：群聊（GroupReplyTarget=private）回触发者私聊时用新建的私聊会话，
// 否则为原会话。分身侧元数据集中放在 meta，避免参数过多。
func (p *AvatarWorkerPool) sendReply(task AvatarTask, convID uint, reply string, meta avatarSendMeta) {
	// 命中知识来源时持久化到 Extra（JSON），与广播下发的 sources 一致，
	// 使刷新/REST 回放后「依据」徽章不丢（buildMessageResponse 从 Extra 读取）。
	sources := p.compactSources(meta.sources)
	msg := model.Message{
		ConversationID: convID,
		SenderID:       task.UserID,
		Type:           "text",
		Content:        reply,
		IsRead:         false,
		Origin:         "avatar",
	}
	if len(sources) > 0 {
		if b, err := json.Marshal(map[string]interface{}{"sources": sources}); err == nil {
			msg.Extra = string(b)
		}
	}

	if err := p.db.Create(&msg).Error; err != nil {
		logger.WithModule("AvatarWorkerPool").Error("保存分身消息失败", "conv", convID, "error", err)
		return
	}

	// 2. 预加载发送者信息
	p.db.Preload("Sender").First(&msg, msg.ID)

	// 3. 更新会话最后消息
	now := time.Now()
	p.db.Model(&model.Conversation{}).Where("id = ?", convID).Updates(map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	})

	// 4. 增加其他成员的未读数
	p.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", convID, task.UserID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

	// 5. 广播消息到会话
	responseData := map[string]interface{}{
		"id":               msg.ID,
		"conversation_id":  msg.ConversationID,
		"sender_id":        msg.SenderID,
		"type":             msg.Type,
		"content":          msg.Content,
		"is_read":          msg.IsRead,
		"created_at":       msg.CreatedAt,
		"is_avatar_reply":  msg.Origin == "avatar",
		"origin":           msg.Origin,
		"sender":           msg.Sender,
		"avatar_name":      meta.avatarCfgName,
		"disclaimer_style": meta.disclaimerStyle,
		"sources":          sources,
	}

	if ws.GlobalHub != nil {
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		jsonMsg, _ := json.Marshal(wsMsg)
		logger.WithModule("sendReply").Debug("Broadcasting",
			"conv", convID, "origin", msg.Origin, "sender_id", msg.SenderID, "sender_name", msg.Sender.Nickname)
		ws.GlobalHub.SendToConversation(convID, 0, jsonMsg)
	}

	logger.WithModule("AvatarWorkerPool").Info("分身回复已发送", "conv", convID, "msgID", msg.ID)
}

// compactSources 压缩待下发的来源：截断即时聊天无需的冗长 snippet，避免 WS 载荷膨胀；
// 无来源时返回 nil，保持旧响应兼容（前端不渲染「依据」）。
func (p *AvatarWorkerPool) compactSources(sources []KnowledgeSource) []KnowledgeSource {
	if len(sources) == 0 {
		return nil
	}
	out := make([]KnowledgeSource, 0, len(sources))
	seen := map[string]struct{}{}
	for _, s := range sources {
		// 去重（同一来源可能被多条命中），并截断 snippet
		key := s.Source + "|" + s.Title
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		if runes := []rune(s.Snippet); len(runes) > 80 {
			s.Snippet = string(runes[:80])
		}
		out = append(out, s)
	}
	return out
}
