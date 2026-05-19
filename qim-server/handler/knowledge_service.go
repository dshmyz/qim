package handler

import (
	"fmt"
	"math"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/logger"
	"regexp"
	"strings"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	aiService *ai.AIService
}

// NewKnowledgeService 创建知识库服务
func NewKnowledgeService(aiService *ai.AIService) *KnowledgeService {
	return &KnowledgeService{
		aiService: aiService,
	}
}

// KnowledgeResult 知识检索结果
type KnowledgeResult struct {
	Note    model.Note `json:"note"`
	Score   float64    `json:"score"`
	Snippet string     `json:"snippet"`
}

// SearchKnowledge 搜索知识库
func (k *KnowledgeService) SearchKnowledge(query string, limit int) []KnowledgeResult {
	if k.aiService == nil || !k.aiService.IsConfigured() {
		return nil
	}

	db := database.GetDB()

	// 提取关键词
	keywords := k.extractKeywords(query)

	// 构建搜索条件
	var notes []model.Note
	db.Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%").
		Or("title LIKE ? OR content LIKE ?", "%"+keywords+"%", "%"+keywords+"%").
		Order("updated_at DESC").
		Limit(limit * 3).
		Find(&notes)

	if len(notes) == 0 {
		return nil
	}

	// 计算相关性得分
	var results []KnowledgeResult
	for _, note := range notes {
		score := k.calculateRelevance(query, &note)
		snippet := k.extractSnippet(note.Content, query, 100)

		results = append(results, KnowledgeResult{
			Note:    note,
			Score:   score,
			Snippet: snippet,
		})
	}

	// 按得分排序
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// extractKeywords 提取关键词（简单的中文分词）
func (k *KnowledgeService) extractKeywords(query string) string {
	// 移除标点符号
	re := regexp.MustCompile(`[^\w\x{4e00}-\x{9fa5}]`)
	return re.ReplaceAllString(query, " ")
}

// calculateRelevance 计算笔记与查询的相关性
func (k *KnowledgeService) calculateRelevance(query string, note *model.Note) float64 {
	score := 0.0

	queryLower := strings.ToLower(query)
	titleLower := strings.ToLower(note.Title)
	contentLower := strings.ToLower(note.Content)

	// 标题匹配权重高
	if strings.Contains(titleLower, queryLower) {
		score += 10.0
	}

	// 内容匹配
	if strings.Contains(contentLower, queryLower) {
		score += 5.0
	}

	// 关键词匹配
	keywords := strings.Fields(k.extractKeywords(query))
	for _, kw := range keywords {
		if strings.Contains(titleLower, kw) {
			score += 3.0
		}
		if strings.Contains(contentLower, kw) {
			score += 1.0
		}
	}

	// 时间衰减（越新得分越高）
	daysSinceUpdate := math.Max(1, float64(30-note.UpdatedAt.Day()))
	score *= (1 + 0.01*daysSinceUpdate)

	return score
}

// extractSnippet 提取包含查询关键词的片段
func (k *KnowledgeService) extractSnippet(content string, query string, maxLen int) string {
	content = strings.TrimSpace(content)
	if len(content) <= maxLen {
		return content
	}

	idx := strings.Index(content, query)
	if idx == -1 {
		return content[:maxLen] + "..."
	}

	start := idx - 30
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 30
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

// BuildKnowledgeContext 构建知识库上下文，用于注入到 AI prompt 中
func (k *KnowledgeService) BuildKnowledgeContext(query string) string {
	results := k.SearchKnowledge(query, 3)

	if len(results) == 0 {
		return ""
	}

	context := "📚 相关知识库内容：\n\n"
	for i, r := range results {
		context += fmt.Sprintf("[%d] %s (相关度: %.1f%%)\n", i+1, r.Note.Title, r.Score*10)
		context += fmt.Sprintf("   %s\n\n", r.Snippet)
	}

	return context
}

// AnswerWithKnowledge 基于知识库回答问题
func (k *KnowledgeService) AnswerWithKnowledge(query string, userID uint) (string, error) {
	if k.aiService == nil || !k.aiService.IsConfigured() {
		return "", fmt.Errorf("AI 服务未配置")
	}

	// 先搜索知识库
	knowledgeCtx := k.BuildKnowledgeContext(query)

	systemPrompt := `你是 QIM 企业即时通讯系统的智能助手。请根据以下信息回答问题。

回答规则：
- 优先使用知识库中的内容回答
- 如果知识库中有相关内容，基于该内容给出准确答案，并引用来源
- 如果知识库中没有相关内容，使用你的通用知识回答，但明确说明"以下回答基于通用知识，建议核实"
- 回答要简洁、专业、准确
- 使用中文回答`

	if knowledgeCtx != "" {
		systemPrompt += "\n\n" + knowledgeCtx
	}

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	answer, err := k.aiService.GetCompletion(ai.TaskTypeChat, messages)
	if err != nil {
		logger.WithModule("KnowledgeService").Error("AI 回答失败", "error", err)
		return "", err
	}

	return answer, nil
}

// GetKnowledgeStats 获取知识库统计信息
func (k *KnowledgeService) GetKnowledgeStats() map[string]interface{} {
	db := database.GetDB()

	var totalNotes int64
	db.Model(&model.Note{}).Count(&totalNotes)

	var activeNotes int64
	db.Model(&model.Note{}).Where("deleted_at IS NULL").Count(&activeNotes)

	var userCount int64
	db.Model(&model.Note{}).Distinct("user_id").Count(&userCount)

	return map[string]interface{}{
		"total_notes":  totalNotes,
		"active_notes": activeNotes,
		"user_count":   userCount,
	}
}
