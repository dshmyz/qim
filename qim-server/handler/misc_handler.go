package handler

import (
	"encoding/json"
	"strconv"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
)

func GetBots(c *gin.Context) {
	db := database.GetDB()
	userID, _ := c.Get("user_id")
	var uid uint
	if v, ok := userID.(uint); ok {
		uid = v
	}

	var bots []model.Bot
	// 可见范围：系统 bot + 模板 bot 全局可见；自定义 bot 仅创建者本人可见（与 BotUsableByUser 一致）。
	db.Where(
		"is_active = ? AND (creator_id = 0 OR is_template = ? OR creator_id = ?)",
		true, true, uid,
	).Find(&bots)

	response.Success(c, bots)
}

func GetSystemMessages(c *gin.Context) {
	pageStr := c.Query("page")
	pageSizeStr := c.Query("pageSize")

	page := 1
	pageSize := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	offset := (page - 1) * pageSize

	db := database.GetDB()
	var systemMessages []model.SystemMessage
	var total int64

	db.Model(&model.SystemMessage{}).Count(&total)
	db.Preload("Sender").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&systemMessages)

	response.Success(c, gin.H{
		"list":  systemMessages,
		"total": total,
		"page":  page,
	})
}

func CreateSystemMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title          string `json:"title" binding:"required"`
		Content        string `json:"content" binding:"required"`
		TargetType     string `json:"target_type"`
		TargetID       *uint  `json:"target_id"`
		TargetIDs      []uint `json:"target_ids"`
		TargetVersion  string `json:"target_version"`  // target_type=version：定向客户端版本
		TargetPlatform string `json:"target_platform"` // target_type=version：定向平台（空=全部）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 定向版本必须指定版本号：空值会让 GetVersionUsers("") 退化为"全部在线用户"而 BroadcastToVersion("")
	// 只命中未知版本桶，DB 通知集合与实时推送目标分歧。仅前端校验不够，须在服务端兜底。
	if req.TargetType == "version" && req.TargetVersion == "" {
		response.BadRequest(c, "target_type=version 时必须提供 target_version")
		return
	}

	db := database.GetDB()

	systemMessage := model.SystemMessage{
		Title:          req.Title,
		Content:        req.Content,
		SenderID:       userID.(uint),
		Status:         "active",
		TargetType:     req.TargetType,
		TargetID:       req.TargetID,
		TargetVersion:  req.TargetVersion,
		TargetPlatform: req.TargetPlatform,
	}

	if err := db.Create(&systemMessage).Error; err != nil {
		response.InternalServerError(c, "创建系统消息失败")
		return
	}

	db.Preload("Sender").First(&systemMessage, systemMessage.ID)

	var usersToNotify []uint

	switch req.TargetType {
	case "all":
		var allUsers []model.User
		db.Find(&allUsers)
		for _, u := range allUsers {
			usersToNotify = append(usersToNotify, u.ID)
		}
	case "department":
		if req.TargetID != nil {
			var deptEmployees []model.DepartmentEmployee
			db.Where("department_id = ?", *req.TargetID).Find(&deptEmployees)
			for _, de := range deptEmployees {
				usersToNotify = append(usersToNotify, de.UserID)
			}
		}
		// 支持多选部门
		if len(req.TargetIDs) > 0 {
			for _, deptID := range req.TargetIDs {
				var deptEmployees []model.DepartmentEmployee
				db.Where("department_id = ?", deptID).Find(&deptEmployees)
				for _, de := range deptEmployees {
					usersToNotify = append(usersToNotify, de.UserID)
				}
			}
		}
	case "group":
		if req.TargetID != nil {
			var conversation model.Conversation
			if err := db.Where("id = ?", *req.TargetID).First(&conversation).Error; err == nil {
				var members []model.ConversationMember
				db.Where("conversation_id = ?", conversation.ID).Find(&members)
				for _, m := range members {
					usersToNotify = append(usersToNotify, m.UserID)
				}
			}
		}
	case "user":
		if req.TargetID != nil {
			usersToNotify = append(usersToNotify, *req.TargetID)
		}
		// 支持多选用户
		usersToNotify = append(usersToNotify, req.TargetIDs...)
	case "version":
		// 定向版本（可选平台）的在线用户：版本只在在线连接上可知，离线用户无法按版本寻址。
		// 老客户端未上报版本时归入"未知版本"桶，选该桶可给老客户端发升级通知。
		// 已知限制：GetVersionUsers 只枚举本节点的在线连接；多节点部署下连接在其他节点的
		// 同版本用户不在名单内（既无通知记录也无推送）。单节点默认部署不受影响。
		if ws.GlobalHub != nil {
			for _, u := range ws.GlobalHub.GetVersionUsers(req.TargetVersion) {
				if req.TargetPlatform != "" && u.Platform != req.TargetPlatform {
					continue
				}
				usersToNotify = append(usersToNotify, u.UserID)
			}
		}
	default:
		usersToNotify = append(usersToNotify, userID.(uint))
	}

	// 批量创建通知用事务包裹，避免部分失败导致某些用户收不到通知。
	// WS 推送放事务外：事务回滚时不推送；事务成功后统一推送，保证不推"已回滚的通知"。
	//
	// 去重：多选部门时同一员工可能在多个部门出现；多选用户时 TargetID 可能与 TargetIDs
	// 重复。不去重会让同一用户收到多条相同通知（DB 多条记录 + WS 多次推送）。
	seen := make(map[uint]struct{}, len(usersToNotify))
	deduped := make([]uint, 0, len(usersToNotify))
	for _, uid := range usersToNotify {
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		deduped = append(deduped, uid)
	}
	usersToNotify = deduped

	notifications := make([]model.Notification, 0, len(usersToNotify))
	for _, notifyUserID := range usersToNotify {
		notifications = append(notifications, model.Notification{
			UserID:  notifyUserID,
			Type:    "system_message",
			Title:   req.Title,
			Content: req.Content,
		})
	}

	if len(notifications) > 0 {
		if err := db.CreateInBatches(notifications, 100).Error; err != nil {
			response.InternalServerError(c, "创建通知失败")
			return
		}
	}

	// 事务成功后再推送 WS
	// 逐用户推送各自的通知记录（user_id/id 与 DB 行一致，客户端已读/删除按 id+user_id 归属校验），
	// version 定向同样走此路径：接收者已按版本筛选，通知是用户级记录，无需设备级分发。
	if ws.GlobalHub != nil {
		for _, n := range notifications {
			notificationMsg := ws.WSMessage{
				Type: "notification",
				Data: n,
			}
			jsonMsg, _ := json.Marshal(notificationMsg)
			ws.GlobalHub.SendToUser(n.UserID, jsonMsg)
		}
	}

	response.Success(c, systemMessage)
}

func UpdateSystemMessage(c *gin.Context) {
	messageIDStr := c.Param("id")

	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var systemMessage model.SystemMessage
	if err := db.First(&systemMessage, uint(messageID)).Error; err != nil {
		response.NotFound(c, "消息不存在")
		return
	}

	systemMessage.Status = req.Status
	if err := db.Save(&systemMessage).Error; err != nil {
		response.InternalServerError(c, "更新消息状态失败")
		return
	}

	response.Success(c, systemMessage)
}

// DeleteSystemMessage 删除系统消息（仅 system_admin）
func DeleteSystemMessage(c *gin.Context) {
	messageIDStr := c.Param("id")

	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	db := database.GetDB()

	var systemMessage model.SystemMessage
	if err := db.First(&systemMessage, uint(messageID)).Error; err != nil {
		response.NotFound(c, "消息不存在")
		return
	}

	if err := db.Delete(&systemMessage).Error; err != nil {
		response.InternalServerError(c, "删除消息失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

func BroadcastMessage(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
		Origin  string `json:"origin"` // 发送方节点 ID：用于丢弃回环，防跨节点消息风暴
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if ws.GlobalHub != nil {
		// 节点中继接收：仅投递本节点客户端，不再转发其他节点
		// （此前写 GlobalHub.Broadcast 会再次进 asyncBroadcast → broadcastToOtherNodes，
		//  多节点下消息在节点间无限转发形成风暴）
		ws.GlobalHub.DeliverBroadcastFromNode(req.Origin, []byte(req.Message))
	}

	response.SuccessWithMessage(c, "消息广播成功", nil)
}

func SendToUserMessage(c *gin.Context) {
	var req struct {
		UserID  uint   `json:"user_id"`
		Message string `json:"message"`
		Origin  string `json:"origin"` // 发送方节点 ID：用于丢弃回环
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if ws.GlobalHub != nil {
		// 节点中继接收：仅投递本节点该用户，不再转发其他节点
		// （此前直接 SendToUser 会再次 sendToUserToOtherNodes 回传发送方，双向无限循环）
		ws.GlobalHub.DeliverToUserFromNode(req.Origin, req.UserID, []byte(req.Message))
	}

	response.SuccessWithMessage(c, "消息发送成功", nil)
}

// BroadcastChatMessage 以系统账号(type='system')为发送方，向用户(默认全员，可指定)的单聊会话
// 发送一条普通私聊消息。与 CreateSystemMessage(通知红点，进通知中心) 不同，这里写入真实的
// conversations + messages，消息会出现在目标用户的「最近会话」列表中，带未读计数与 WS 推送。
//
// 复用 ConversationService.CreateSingleConversation(幂等：已有会话则复用) 与
// MessageService.SendMessage(内部处理敏感词、未读、WS 广播、会话显示恢复)。
// 单聊触发 SmartReplyEngine 时会在 conv.Type=="single" 处提前返回，不会产生 AI 自动回复。
//
// 角色门槛：system_admin（与系统消息更新/删除一致）。
func BroadcastChatMessage(c *gin.Context) {
	var req struct {
		Content        string `json:"content" binding:"required"`
		TargetUserIDs  []uint `json:"target_user_ids"`  // 为空表示全员；非空则仅发给这些用户
		ExcludeUserIDs []uint `json:"exclude_user_ids"` // 可选：从目标中排除（如排除系统/机器人）
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		response.BadRequest(c, "参数错误：content 必填")
		return
	}

	app := di.GlobalContainer
	msgSvc := app.MessageService
	db := database.GetDB()

	systemUserID := msgSvc.GetSystemUserID()
	if systemUserID == 0 {
		response.InternalServerError(c, "系统账号(type=system)不存在")
		return
	}

	exclude := make(map[uint]bool, len(req.ExcludeUserIDs))
	for _, id := range req.ExcludeUserIDs {
		exclude[id] = true
	}

	// 收集目标用户
	var targetUsers []model.User
	if len(req.TargetUserIDs) > 0 {
		db.Where("id IN ?", req.TargetUserIDs).Find(&targetUsers)
	} else {
		// 全员：默认排除系统账号本身，避免给自己发消息
		db.Where("type != ?", "system").Find(&targetUsers)
	}

	// 异步发送：POST 立即返回 job_id，前端轮询真实成败——避免大用户量群发超过前端超时
	// 而管理员不知道是否已发送（worker 池并行显著缩短总耗时）。
	var total int
	for _, u := range targetUsers {
		if !exclude[u.ID] {
			total++
		}
	}
	job := newBroadcastChatJob(total, req.Content)
	// 响应字段在启动 worker 前读取（此时 Status 恒为 running、Total 已定），避免与并发 worker 读竞争
	jobID, jobTotal, jobStatus := job.ID, job.Total, job.Status
	go runBroadcastChatJob(job, targetUsers, exclude, systemUserID)

	response.Success(c, gin.H{
		"job_id": jobID,
		"total":  jobTotal,
		"status": jobStatus,
	})
}
