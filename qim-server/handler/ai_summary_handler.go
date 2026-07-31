package handler

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

var errSummaryAccessDenied = errors.New("summary conversation access denied")

// GenerateSummaryRequest 生成摘要请求
type GenerateSummaryRequest struct {
	ConversationID uint       `json:"conversation_id" binding:"required"`
	TimeRange      string     `json:"time_range"` // "1h", "today", "7d", "custom"
	StartTime      *time.Time `json:"start_time"`
	EndTime        *time.Time `json:"end_time"`
}

// summaryPrepareResult 包含摘要生成所需的数据和元信息
type summaryPrepareResult struct {
	Messages      []model.Message
	MessagesCount int
	StartTime     time.Time
	EndTime       time.Time
	GroupName     string
	ActiveMembers []string
}

func summaryRequestUserID(c *gin.Context) uint {
	raw, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	userID, ok := raw.(uint)
	if !ok {
		return 0
	}
	return userID
}

func resolveSummaryTimeRange(req *GenerateSummaryRequest) (time.Time, time.Time) {
	now := time.Now()
	var startTime, endTime time.Time
	endTime = now

	switch req.TimeRange {
	case "1h":
		startTime = endTime.Add(-1 * time.Hour)
	case "today":
		startTime = endTime.Truncate(24 * time.Hour)
	case "7d":
		startTime = endTime.Add(-7 * 24 * time.Hour)
	case "custom":
		if req.StartTime != nil && req.EndTime != nil {
			startTime = *req.StartTime
			endTime = *req.EndTime
		} else {
			startTime = endTime.Add(-24 * time.Hour)
		}
	default:
		startTime = endTime.Add(-24 * time.Hour)
	}
	return startTime, endTime
}

func prepareSummaryData(req *GenerateSummaryRequest, userID uint) (*summaryPrepareResult, error) {
	startTime, endTime := resolveSummaryTimeRange(req)

	db := database.GetDB()
	if userID == 0 {
		return nil, errSummaryAccessDenied
	}
	if err := ensureConversationAccess(db, req.ConversationID, userID); err != nil {
		return nil, errSummaryAccessDenied
	}

	var messages []model.Message
	db.Where("conversation_id = ? AND created_at >= ? AND created_at <= ?",
		req.ConversationID, startTime, endTime).
		Preload("Sender").
		Order("created_at ASC").
		Find(&messages)

	var group model.Group
	var groupName string
	if err := db.Where("conversation_id = ?", req.ConversationID).First(&group).Error; err == nil {
		groupName = group.Name
	}

	filtered := make([]model.Message, 0, len(messages))
	activeMap := make(map[string]bool)
	for _, msg := range messages {
		if msg.Type != "text" && msg.Type != "markdown" {
			continue
		}
		if msg.IsRecalled || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		filtered = append(filtered, msg)

		name := msg.Sender.Nickname
		if name == "" {
			name = msg.Sender.Username
		}
		if name != "" {
			activeMap[name] = true
		}
	}

	const maxMessages = 100
	if len(filtered) > maxMessages {
		filtered = filtered[len(filtered)-maxMessages:]
	}

	activeMembers := make([]string, 0, len(activeMap))
	for name := range activeMap {
		activeMembers = append(activeMembers, name)
	}
	sort.Strings(activeMembers)

	return &summaryPrepareResult{
		Messages:      filtered,
		MessagesCount: len(filtered),
		StartTime:     startTime,
		EndTime:       endTime,
		GroupName:     groupName,
		ActiveMembers: activeMembers,
	}, nil
}

func buildSummaryMessages(data *summaryPrepareResult) []ai.Message {
	var sb strings.Builder
	sb.WriteString("你是 QIM 企业即时通讯系统的对话摘要助手。请根据对话记录生成一份结构化摘要。\n\n")

	sb.WriteString("【输出格式】\n")
	sb.WriteString("请严格按以下四个部分输出（每个部分如果无内容可写“无”）：\n\n")
	sb.WriteString("### 一、讨论要点\n")
	sb.WriteString("- 列出对话中讨论过的核心议题（不超过 5 条）\n")
	sb.WriteString("- 每条用 1-2 句话概括\n\n")
	sb.WriteString("### 二、决策结论\n")
	sb.WriteString("- 列出已达成共识或已作出的决策\n")
	sb.WriteString("- 标注作出该结论的主要依据\n\n")
	sb.WriteString("### 三、待办事项\n")
	sb.WriteString("- 列出明确的行动项，每条包含：事项描述、负责人、建议截止时间（如有）\n")
	sb.WriteString("- 如果负责人不明确，写“待确认”\n\n")
	sb.WriteString("### 四、参与者\n")
	sb.WriteString("- 列出活跃参与讨论的人员及各自主要观点\n\n")

	sb.WriteString("【摘要规则】\n")
	sb.WriteString("1. 提取关键信息、决策和待办事项\n")
	sb.WriteString("2. 识别主要讨论话题\n")
	sb.WriteString("3. 使用简洁、客观的语言，避免冗余\n")
	sb.WriteString("4. 保持事实准确，不要添加主观评价\n")
	sb.WriteString("5. 输出使用 Markdown 格式\n\n")

	if data.GroupName != "" {
		sb.WriteString(fmt.Sprintf("【会话名称】%s\n", data.GroupName))
	}
	sb.WriteString(fmt.Sprintf("【时间范围】%s 至 %s\n",
		data.StartTime.Format("2006-01-02 15:04"),
		data.EndTime.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("【消息数量】%d 条\n", data.MessagesCount))
	if len(data.ActiveMembers) > 0 {
		sb.WriteString(fmt.Sprintf("【活跃参与者】%s\n", strings.Join(data.ActiveMembers, ", ")))
	}

	var conversationText strings.Builder
	conversationText.WriteString("以下是需要摘要的对话记录：\n\n")
	for _, msg := range data.Messages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		timestamp := msg.CreatedAt.Format("15:04")
		conversationText.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, senderName, msg.Content))
	}
	conversationText.WriteString("\n请严格按照上方结构化输出格式生成摘要。")

	return []ai.Message{
		{Role: "system", Content: sb.String()},
		{Role: "user", Content: conversationText.String()},
	}
}

// GenerateSummaryMeta 获取摘要元数据（不调用 AI）
func (h *AIHandler) GenerateSummaryMeta(c *gin.Context) {
	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	data, err := prepareSummaryData(&req, summaryRequestUserID(c))
	if err != nil {
		if errors.Is(err, errSummaryAccessDenied) {
			response.Forbidden(c, "无权访问该会话")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取摘要元数据失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"messages_count": data.MessagesCount,
			"time_range":     fmt.Sprintf("%s 至 %s", data.StartTime.Format("2006-01-02 15:04"), data.EndTime.Format("2006-01-02 15:04")),
			"group_name":     data.GroupName,
			"active_members": data.ActiveMembers,
		},
	})
}

// GenerateSummary 生成会话摘要（同步）
func (h *AIHandler) GenerateSummary(c *gin.Context) {
	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	data, err := prepareSummaryData(&req, summaryRequestUserID(c))
	if err != nil {
		if errors.Is(err, errSummaryAccessDenied) {
			response.Forbidden(c, "无权访问该会话")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "摘要生成失败: " + err.Error()})
		return
	}
	if data.MessagesCount == 0 {
		streamSSE(c, func(writeChunk func(string)) error {
			writeChunk("该时间段内没有可摘要的消息")
			return nil
		})
		return
	}

	messages := buildSummaryMessages(data)
	summary, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "摘要生成失败: " + err.Error()})
		return
	}

	summary = h.aiService.FilterOutput(summary, "ai_summary")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"summary":        summary,
			"messages_count": data.MessagesCount,
			"time_range":     fmt.Sprintf("%s 至 %s", data.StartTime.Format("2006-01-02 15:04"), data.EndTime.Format("2006-01-02 15:04")),
			"group_name":     data.GroupName,
			"active_members": data.ActiveMembers,
		},
	})
}

// GenerateSummaryStream 流式生成会话摘要（SSE）
// @Summary 流式生成会话摘要
// @Description 根据对话记录流式生成会话摘要，使用 SSE 返回
// @Tags AI
// @Accept json
// @Produce text/event-stream
// @Param request body GenerateSummaryRequest true "摘要请求"
// @Router /api/ai/summary/stream [post]
func (h *AIHandler) GenerateSummaryStream(c *gin.Context) {
	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	data, err := prepareSummaryData(&req, summaryRequestUserID(c))
	if err != nil {
		if errors.Is(err, errSummaryAccessDenied) {
			response.Forbidden(c, "无权访问该会话")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "摘要生成失败: " + err.Error()})
		return
	}
	if data.MessagesCount == 0 {
		streamSSE(c, func(writeChunk func(string)) error {
			writeChunk("该时间段内没有可摘要的消息")
			return nil
		})
		return
	}

	messages := buildSummaryMessages(data)

	streamSSE(c, func(writeChunk func(string)) error {
		return h.aiService.GetCompletionStream(ai.TaskTypeAnalysis, messages, func(chunk ai.StreamChunk) error {
			if chunk.Content != "" {
				writeChunk(chunk.Content)
			}
			return nil
		})
	})
}
