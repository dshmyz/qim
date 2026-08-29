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

// pendingSendFromResult 从确认制工具的返回值中提取待确认发送载荷。
// 结果约定见 service.SendMessageTool 确认分支（status=pending_confirmation + pending 载荷）。
func pendingSendFromResult(result interface{}) *ai.PendingSend {
	m, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}
	if status, _ := m["status"].(string); status != "pending_confirmation" {
		return nil
	}
	info, ok := m["pending"].(ai.PendingSend)
	if !ok || info.ID == 0 {
		return nil
	}
	return &info
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
