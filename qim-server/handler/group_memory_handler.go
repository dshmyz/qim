package handler

import (
	"errors"
	"strconv"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

// resolveGroupForMemory 解析 :id（conversation_id）对应的群并做成员/权限校验。
// 权限与 UpdateGroupAISettings 一致：必须是成员；正式群（GroupType=="group"）仅群主/管理员可管理群记忆。
func resolveGroupForMemory(c *gin.Context) (*model.Group, bool) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return nil, false
	}

	groupSvc := di.GlobalContainer.GroupService
	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return nil, false
	}

	convSvc := di.GlobalContainer.ConversationService
	conv, err := convSvc.GetConversation(uint(convID))
	if err != nil {
		response.NotFound(c, "会话不存在")
		return nil, false
	}
	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能管理群聊的群记忆")
		return nil, false
	}

	member, err := convSvc.GetMember(uint(convID), userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是成员")
		return nil, false
	}
	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以管理群记忆")
		return nil, false
	}
	return group, true
}

// GetGroupMemories 列出本群群记忆条目。
func GetGroupMemories(c *gin.Context) {
	memorySvc := di.GlobalContainer.GroupMemoryService
	if memorySvc == nil {
		response.Success(c, []interface{}{})
		return
	}
	group, ok := resolveGroupForMemory(c)
	if !ok {
		return
	}
	memories, err := memorySvc.GetGroupMemories(group.ID, 50)
	if err != nil {
		response.InternalServerError(c, "获取群记忆失败")
		return
	}
	response.Success(c, memories)
}

// DeleteGroupMemory 删除单条群记忆。
func DeleteGroupMemory(c *gin.Context) {
	memorySvc := di.GlobalContainer.GroupMemoryService
	if memorySvc == nil {
		response.BadRequest(c, "群记忆服务未启用")
		return
	}
	group, ok := resolveGroupForMemory(c)
	if !ok {
		return
	}
	memoryID := c.Param("memory_id")
	if memoryID == "" {
		response.BadRequest(c, "缺少记忆ID")
		return
	}
	if err := memorySvc.DeleteMemory(group.ID, memoryID); err != nil {
		if errors.Is(err, service.ErrMemoryNotFound) {
			response.NotFound(c, "记忆不存在")
			return
		}
		response.InternalServerError(c, "删除群记忆失败")
		return
	}
	response.SuccessWithMessage(c, "已删除", nil)
}

// ClearGroupMemories 清空本群全部群记忆。
func ClearGroupMemories(c *gin.Context) {
	memorySvc := di.GlobalContainer.GroupMemoryService
	if memorySvc == nil {
		response.BadRequest(c, "群记忆服务未启用")
		return
	}
	group, ok := resolveGroupForMemory(c)
	if !ok {
		return
	}
	deleted, err := memorySvc.ForgetAll(group.ID)
	if err != nil {
		response.InternalServerError(c, "清空群记忆失败")
		return
	}
	response.SuccessWithMessage(c, "已清空", gin.H{"deleted": deleted})
}

// SearchGroupMemories 按关键词检索本群群记忆。
func SearchGroupMemories(c *gin.Context) {
	memorySvc := di.GlobalContainer.GroupMemoryService
	if memorySvc == nil {
		response.Success(c, []interface{}{})
		return
	}
	group, ok := resolveGroupForMemory(c)
	if !ok {
		return
	}
	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if req.TopK <= 0 || req.TopK > 20 {
		req.TopK = 5
	}
	results, err := memorySvc.Recall(group.ID, req.Query, req.TopK)
	if err != nil {
		response.InternalServerError(c, "检索群记忆失败")
		return
	}
	response.Success(c, results)
}
