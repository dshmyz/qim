package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

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

	systemPrompt := aiprompt.CurrentTimeLine() + "\n\n" + `你是一个笔记分析助手。分析以下笔记内容，返回 JSON 格式结果：
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

// noteFormatMaxRunes AI 格式化时参与整理的笔记内容上限（字符数）。
// 与 AIService.filterContent 的输入上限（10000 rune）对齐：handler 先按此截断，
// 使截断点由我们控制（整字符、不夹带「内容已截断」后缀），AI 层不会再二次截断。
// 超出部分截断，返回 truncated 标记供前端提示。
const noteFormatMaxRunes = 10000

// FormatNote AI 格式化笔记内容为规范 Markdown：清理页眉页脚/页号/错乱换行，
// 规范化标题/列表/表格/代码块，保留原文语义。前端确认预览后调用 PUT 覆盖保存。
// @Summary AI 格式化笔记内容
// @Description 将笔记内容整理为规范 Markdown；超长笔记只处理前 10000 字符并返回 truncated 标记
// @Tags 笔记
// @Accept json
// @Produce json
// @Param id path int true "笔记ID"
// @Success 200 {object} response.Response{data=object{content=string,truncated=bool}} "格式化结果"
// @Failure 400 {object} response.Response "无效ID/内容疑似乱码"
// @Failure 404 {object} response.Response "笔记不存在"
// @Failure 503 {object} response.Response "AI 服务未配置"
// @Router /notes/{id}/format [post]
func FormatNote(c *gin.Context) {
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

	// 乱码输入拒绝：内容本身已损坏（U+FFFD 占比超阈值），格式化无法修复，
	// 且会把乱码语义扩散进整篇输出。
	if service.IsMojibakeText(note.Content) {
		response.BadRequest(c, "笔记内容疑似乱码，无法格式化，请检查内容是否损坏")
		return
	}

	content := note.Content
	truncated := false
	if runes := []rune(content); len(runes) > noteFormatMaxRunes {
		content = string(runes[:noteFormatMaxRunes])
		truncated = true
	}

	systemPrompt := aiprompt.CurrentTimeLine() + "\n\n" + `你是一个笔记排版助手。把用户提供的原始笔记内容整理为规范的 Markdown 格式，规则：
1. 清理干扰：页眉页脚、页码、目录残留、重复行；修复 PDF/DOCX 提取产生的错乱换行（把被拆断的句子重新连起来）
2. 结构化：识别并规范标题层级（#/##/###）、无序/有序列表、任务列表、引用；检测到的表格转为 Markdown 表格
3. 保留语义：原文内容、专有名词、数字、引用出处一律保留，不得增删改写
4. 输出要求：只输出整理后的 Markdown 正文，不要任何解释、不要代码围栏包裹、不要"以下是"之类的开场白`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: content},
	}

	result, err := aiSvc.GetCompletion(ai.TaskTypeAnalysis, messages)
	if err != nil {
		response.InternalServerError(c, "AI 格式化失败")
		return
	}

	// 防御：AI 输出同样过乱码检查，异常时不让结果落库
	if service.IsMojibakeText(result) {
		response.InternalServerError(c, "AI 返回内容异常，请重试")
		return
	}

	response.Success(c, gin.H{
		"content":   strings.TrimSpace(result),
		"truncated": truncated,
	})
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
