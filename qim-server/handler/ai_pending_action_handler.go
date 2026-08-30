package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

// pendingSendFromResult 从确认制工具的返回值中提取待确认发送载荷（委托 service 单一实现）。
func pendingSendFromResult(result interface{}) *ai.PendingSend {
	return service.PendingSendFromResult(result)
}

// pendingActionUserID 从 gin 上下文取认证用户 ID，与 ai_handler 同口径。
func pendingActionUserID(c *gin.Context) uint {
	userIDAny, _ := c.Get("user_id")
	userID, _ := userIDAny.(uint)
	return userID
}

// pendingActionResponse 统一回包：already_handled=true 时携带终态记录供前端回显
// （幂等语义与 bot 卡片动作一致，重复点击不报错只回当前状态）。
func pendingActionResponse(c *gin.Context, record *model.AIPendingAction, alreadyHandled bool) {
	response.Success(c, gin.H{
		"already_handled": alreadyHandled,
		"status":          record.Status,
		"id":              record.ID,
		"target_name":     record.TargetName,
		"message_id":      record.MessageID,
	})
}

// mapPendingActionError 把服务层哨兵错误映射为 HTTP 语义。
// 返回 false 表示已写出响应。
func mapPendingActionError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, service.ErrPendingNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, service.ErrPendingForbidden):
		response.Forbidden(c, err.Error())
	case errors.Is(err, service.ErrPendingExpired):
		response.Error(c, http.StatusGone, http.StatusGone, err.Error())
	case errors.Is(err, service.ErrPendingAlreadyHandled):
		// 理论上不会走到（loadOperable 已把记录带出），防御性兜底
		response.Conflict(c, err.Error())
	default:
		response.BadRequest(c, err.Error())
	}
	return false
}

// logPendingAction 记录用户操作审计（服务未装配时静默跳过）。
func logPendingAction(c *gin.Context, action string) {
	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "ai", action)
	}
}

// ConfirmPendingAction POST /ai/pending-actions/:id/confirm
// 用户确认侧边栏 AI 的待确认发送，真正执行消息发送（走正常用户发送路径，非绕过）。
func (h *AIHandler) ConfirmPendingAction(c *gin.Context) {
	if h.pendingActions == nil {
		response.InternalServerError(c, "待确认服务不可用")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "参数错误")
		return
	}
	userID := pendingActionUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "需要登录")
		return
	}

	record, handled, err := h.pendingActions.ConfirmPendingSend(userID, uint(id))
	if err != nil {
		if handled && record != nil {
			// 已处理/已过期：携带终态回显，帮助前端同步状态
			logger.WithModule("AIHandler").Info("pending send 已终态", "id", id, "status", record.Status)
			pendingActionResponse(c, record, handled)
			return
		}
		// 发送失败（敏感词/成员校验等）保留 pending 态并把原因带回给用户
		mapPendingActionError(c, err)
		return
	}

	logPendingAction(c, "ai_pending_send_confirmed")
	pendingActionResponse(c, record, handled)
}

// CancelPendingAction POST /ai/pending-actions/:id/cancel
// 用户取消侧边栏 AI 的待确认发送。
func (h *AIHandler) CancelPendingAction(c *gin.Context) {
	if h.pendingActions == nil {
		response.InternalServerError(c, "待确认服务不可用")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "参数错误")
		return
	}
	userID := pendingActionUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "需要登录")
		return
	}

	record, handled, err := h.pendingActions.CancelPendingSend(userID, uint(id))
	if err != nil {
		if record != nil {
			pendingActionResponse(c, record, handled)
			return
		}
		mapPendingActionError(c, err)
		return
	}

	logPendingAction(c, "ai_pending_send_cancelled")
	pendingActionResponse(c, record, handled)
}


// SetAIMessageFeedback POST /ai/feedback
// 用户对单条 AI 回复的 👍/👎 反馈：rating 1/-1 为设置，0 为撤销。
// 主观反馈接入 ai_reply_metrics 质量闭环，作为自动度量的用户侧信号源。
func (h *AIHandler) SetAIMessageFeedback(c *gin.Context) {
	if h.feedback == nil {
		response.InternalServerError(c, "反馈服务不可用")
		return
	}
	userID := pendingActionUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "需要登录")
		return
	}
	var req struct {
		MessageID uint `json:"message_id"`
		Rating    int  `json:"rating"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.MessageID == 0 {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := h.feedback.SetFeedback(userID, req.MessageID, req.Rating); err != nil {
		switch {
		case errors.Is(err, service.ErrFeedbackNotFound):
			response.NotFound(c, err.Error())
		case errors.Is(err, service.ErrFeedbackForbidden), errors.Is(err, service.ErrFeedbackInvalidParam):
			response.BadRequest(c, err.Error())
		default:
			response.InternalServerError(c, "保存反馈失败")
		}
		return
	}
	logPendingAction(c, "ai_message_feedback")
	response.Success(c, gin.H{"message_id": req.MessageID, "rating": req.Rating})
}

// GetAIMessageFeedback GET /ai/feedback/:messageId
// 查询当前用户对某条消息的反馈（客户端恢复选中态）。
func (h *AIHandler) GetAIMessageFeedback(c *gin.Context) {
	if h.feedback == nil {
		response.InternalServerError(c, "反馈服务不可用")
		return
	}
	userID := pendingActionUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "需要登录")
		return
	}
	messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 64)
	if err != nil || messageID == 0 {
		response.BadRequest(c, "参数错误")
		return
	}
	rating, err := h.feedback.GetFeedback(userID, uint(messageID))
	if err != nil {
		response.InternalServerError(c, "查询反馈失败")
		return
	}
	response.Success(c, gin.H{"message_id": messageID, "rating": rating})
}
