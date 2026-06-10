package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type NoteTagsRequest struct {
	Tags []string `json:"tags"`
}

type NoteSummaryRequest struct {
	Summary string `json:"summary"`
}

type AIAnalyzeResponse struct {
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	ActionItems []string `json:"action_items"`
}

func AnalyzeNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的笔记ID")
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		response.NotFound(c, "笔记不存在")
		return
	}

	aiSvc := di.GlobalContainer.AIService
	if aiSvc == nil || !aiSvc.IsConfigured() {
		response.Error(c, http.StatusServiceUnavailable, 503, "AI 服务未配置")
		return
	}

	systemPrompt := `你是一个笔记分析助手。分析以下笔记内容，返回 JSON 格式结果：
1. summary: 笔记摘要（100字以内）
2. tags: 推荐标签（最多5个，简洁明了）
3. action_items: 提取的行动项（如果有，最多5个）

只返回 JSON，格式：{"summary": "...", "tags": ["标签1", "标签2"], "action_items": ["行动项1"]}`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: note.Content},
	}

	result, err := aiSvc.GetCompletion(ai.TaskTypeAnalysis, messages)
	if err != nil {
		response.InternalServerError(c, "AI 分析失败")
		return
	}

	var analyzeResult AIAnalyzeResponse
	jsonStr := result
	if idx := findJSONStart(result); idx >= 0 {
		jsonStr = result[idx:]
		if endIdx := findJSONEnd(jsonStr); endIdx >= 0 {
			jsonStr = jsonStr[:endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &analyzeResult); err != nil {
		analyzeResult = AIAnalyzeResponse{
			Summary:     result[:min(100, len(result))],
			Tags:        []string{},
			ActionItems: []string{},
		}
	}

	response.Success(c, analyzeResult)
}

func ExportNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的笔记ID")
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		response.NotFound(c, "笔记不存在")
		return
	}

	content := fmt.Sprintf("# %s\n\n%s", note.Title, note.Content)

	filename := fmt.Sprintf("%s.md", note.Title)
	c.Header("Content-Disposition", sanitizeFilename(filename))
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.String(http.StatusOK, content)
}

func UpdateNoteTags(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的笔记ID")
		return
	}

	var req NoteTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tagsJSON, _ := json.Marshal(req.Tags)

	db := database.GetDB()
	if err := db.Model(&model.Note{}).Where("id = ? AND user_id = ?", uint(noteID), userID).Update("tags", string(tagsJSON)).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

func UpdateNoteSummary(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的笔记ID")
		return
	}

	var req NoteSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	if err := db.Model(&model.Note{}).Where("id = ? AND user_id = ?", uint(noteID), userID).Update("summary", req.Summary).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' || c == '[' {
			return i
		}
	}
	return -1
}

func findJSONEnd(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '}' || s[i] == ']' {
			return i
		}
	}
	return -1
}
