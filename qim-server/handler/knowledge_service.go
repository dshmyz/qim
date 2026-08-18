package handler

import (
	"fmt"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"math"
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
