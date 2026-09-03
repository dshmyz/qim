package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"gorm.io/gorm"
)

// pendingActionTTL 待确认动作的有效期：超时未确认即过期作废，
// 防止「几小时前的发送请求」被滞后确认后突袭发出。
const pendingActionTTL = 10 * time.Minute

// 待确认动作服务的哨兵错误。handler 据此映射 HTTP 语义（404/403/409/410）。
var (
	ErrPendingNotFound        = errors.New("待确认请求不存在")
	ErrPendingForbidden       = errors.New("无权操作该待确认请求")
	ErrPendingAlreadyHandled  = errors.New("该请求已被处理")
	ErrPendingExpired         = errors.New("待确认请求已过期")
	ErrPendingInvalidParams   = errors.New("待确认请求参数不完整")
	ErrPendingServiceNotReady = errors.New("待确认服务不可用")
)

// AIPendingActionService AI 敏感工具调用的待确认执行服务。
// 背景：侧边栏 AI 的 send_message 由 LLM 决定内容与目标会话，直接执行等于
// 「模型可代用户向任意所在会话发消息」（bot 1:1 路径已在白名单注释中显式排除该能力）。
// 本服务把执行拆成两段：工具执行时仅落 pending 记录（CreatePendingSend），
// 用户在客户端确认后才真正发出（ConfirmPendingSend）。
type AIPendingActionService struct {
	db      *gorm.DB
	msgSvc  *MessageService
}

// NewAIPendingActionService 创建待确认执行服务。msgSvc 为 nil 时确认端点不可用
// （仅注册了工具但未装配消息服务的场景，正常 DI 下不会出现）。
func NewAIPendingActionService(msgSvc *MessageService) *AIPendingActionService {
	if msgSvc == nil {
		return &AIPendingActionService{}
	}
	return &AIPendingActionService{db: msgSvc.db, msgSvc: msgSvc}
}

// CreatePendingSend 生成一条待确认的发送请求（不发送）。返回的记录含解析好的
// 目标会话展示名与内容预览，供 SSE pending 帧与确认条直接使用。
func (s *AIPendingActionService) CreatePendingSend(userID, contextConvID, targetConvID uint, content string) (*model.AIPendingAction, error) {
	if s.db == nil {
		return nil, ErrPendingServiceNotReady
	}
	if userID == 0 || targetConvID == 0 || content == "" {
		return nil, ErrPendingInvalidParams
	}

	// 顺带清理已过期记录（低频路径，直接删即可，避免表无限增长）。
	// sending 是瞬时态（同步 SendMessage 一次调用即离开），超过 TTL 的 sending 行
	// 必然是崩溃残留（消息已发但 confirmed 未落库），一并清理，否则永久卡死且无任何出口。
	s.db.Where("status IN (?, ?) AND expires_at < ?",
		model.AIPendingActionStatusPending, model.AIPendingActionStatusSending, time.Now()).
		Delete(&model.AIPendingAction{})

	record := &model.AIPendingAction{
		UserID:               userID,
		ConversationID:       contextConvID,
		TargetConversationID: targetConvID,
		TargetName:           s.resolveConversationName(userID, targetConvID),
		Content:              content,
		Status:               model.AIPendingActionStatusPending,
		ExpiresAt:            time.Now().Add(pendingActionTTL),
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("创建待确认请求失败: %w", err)
	}
	return record, nil
}

// ConfirmPendingSend 确认并执行待确认发送。
// 归属/状态/有效期三重校验后，经 MessageService.SendMessage 走正常用户发送路径
// （其内部含成员校验与敏感词校验，确认不是绕过）。
// 已被处理过的请求返回 ErrPendingAlreadyHandled 且附带当前记录，供前端回显终态。
// 并发防护：发送前先以条件 UPDATE 原子占有记录（pending→sending），
// 占有失败的并发请求按已处理回显，保证恰好一次发送。
func (s *AIPendingActionService) ConfirmPendingSend(userID, id uint) (*model.AIPendingAction, bool, error) {
	record, handled, err := s.loadOperable(userID, id)
	if err != nil {
		return record, handled, err
	}
	if s.msgSvc == nil {
		return record, false, ErrPendingServiceNotReady
	}

	claimRes := s.db.Model(&model.AIPendingAction{}).
		Where("id = ? AND status = ?", record.ID, model.AIPendingActionStatusPending).
		Updates(map[string]interface{}{"status": model.AIPendingActionStatusSending})
	if claimRes.Error != nil {
		return record, false, fmt.Errorf("锁定待确认请求失败: %w", claimRes.Error)
	}
	if claimRes.RowsAffected == 0 {
		// 并发确认/取消已抢先占有：重读当前状态回显终态
		if err := s.db.First(record, record.ID).Error; err != nil {
			return nil, true, ErrPendingAlreadyHandled
		}
		return record, true, ErrPendingAlreadyHandled
	}
	record.Status = model.AIPendingActionStatusSending
	msg, err := s.msgSvc.SendMessage(record.TargetConversationID, userID, "text", record.Content, nil)
	if err != nil {
		// 发送失败（如命中敏感词）回退 pending 态，用户可修改后重新发起或取消；
		// 回退仍按条件更新（万一此刻被取消，不覆盖终态）。
		s.db.Model(&model.AIPendingAction{}).
			Where("id = ? AND status = ?", record.ID, model.AIPendingActionStatusSending).
			Update("status", model.AIPendingActionStatusPending)
		record.Status = model.AIPendingActionStatusPending
		return record, false, fmt.Errorf("发送失败: %w", err)
	}

	res := s.db.Model(&model.AIPendingAction{}).
		Where("id = ? AND status = ?", record.ID, model.AIPendingActionStatusSending).
		Updates(map[string]interface{}{
			"status":     model.AIPendingActionStatusConfirmed,
			"message_id": msg.ID,
		})
	if res.Error != nil {
		return record, false, fmt.Errorf("更新待确认状态失败: %w", res.Error)
	}
	record.Status = model.AIPendingActionStatusConfirmed
	record.MessageID = msg.ID
	return record, false, nil
}

// CancelPendingSend 取消待确认发送。对已终态的记录同样返回 ErrPendingAlreadyHandled + 当前记录。
// 取消与确认竞争同一原子占有（pending→sending），确认正在发送时取消失败按已处理回显。
func (s *AIPendingActionService) CancelPendingSend(userID, id uint) (*model.AIPendingAction, bool, error) {
	record, handled, err := s.loadOperable(userID, id)
	if err != nil {
		return record, handled, err
	}

	res := s.db.Model(&model.AIPendingAction{}).
		Where("id = ? AND status = ?", record.ID, model.AIPendingActionStatusPending).
		Update("status", model.AIPendingActionStatusCancelled)
	if res.Error != nil {
		return record, false, fmt.Errorf("更新待确认状态失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		if err := s.db.First(record, record.ID).Error; err != nil {
			return nil, true, ErrPendingAlreadyHandled
		}
		return record, true, ErrPendingAlreadyHandled
	}
	record.Status = model.AIPendingActionStatusCancelled
	return record, false, nil
}

// loadOperable 加载一条处于 pending 且未过期的待确认记录。
// 返回 handled=true 表示记录已处于终态（confirmed/cancelled/expired），
// 此时附带记录本身返回、不再改动，调用方以 ErrPendingAlreadyHandled 回显。
func (s *AIPendingActionService) loadOperable(userID, id uint) (*model.AIPendingAction, bool, error) {
	if s.db == nil {
		return nil, false, ErrPendingServiceNotReady
	}
	var record model.AIPendingAction
	if err := s.db.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, ErrPendingNotFound
		}
		return nil, false, fmt.Errorf("查询待确认请求失败: %w", err)
	}
	if record.UserID != userID {
		return nil, false, ErrPendingForbidden
	}
	if record.Status != model.AIPendingActionStatusPending {
		return &record, true, ErrPendingAlreadyHandled
	}
	if time.Now().After(record.ExpiresAt) {
		record.Status = model.AIPendingActionStatusExpired
		s.db.Save(&record)
		return &record, true, ErrPendingExpired
	}
	return &record, false, nil
}

// resolveConversationName 尽力解析目标会话的展示名，仅供确认条展示（解析失败不阻断流程）。
// 群/讨论组 → Group.Name；bot 会话 → Bot.Name；单聊 → 对方昵称/用户名；其余回退会话 ID。
func (s *AIPendingActionService) resolveConversationName(userID, convID uint) string {
	fallback := fmt.Sprintf("会话 #%d", convID)

	var conv model.Conversation
	if err := s.db.First(&conv, convID).Error; err != nil {
		return fallback
	}

	switch conv.Type {
	case "group", "discussion":
		var group model.Group
		if err := s.db.Where("conversation_id = ?", convID).First(&group).Error; err == nil && group.Name != "" {
			return group.Name
		}
	case "bot":
		var bc model.BotConversation
		if err := s.db.Where("conversation_id = ?", convID).First(&bc).Error; err == nil {
			var bot model.Bot
			if err := s.db.First(&bot, bc.BotID).Error; err == nil && bot.Name != "" {
				return bot.Name
			}
		}
	case "single":
		var member model.ConversationMember
		if err := s.db.Where("conversation_id = ? AND user_id <> ?", convID, userID).First(&member).Error; err == nil {
			var user model.User
			if err := s.db.First(&user, member.UserID).Error; err == nil {
				if user.Nickname != "" {
					return user.Nickname
				}
				return user.Username
			}
		}
	}
	return fallback
}
