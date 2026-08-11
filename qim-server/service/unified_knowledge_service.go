package service

import (
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

type UnifiedKnowledgeService struct {
	groupDocSvc     *GroupDocumentService
	legacyFallback  *LegacyKnowledgeFallback
	vectorEnabled   bool
	scoreThreshold  float64 // 知识来源分数门槛（0-1），低于此分数不注入 prompt 也不展示在徽章
}

type LegacyKnowledgeFallback struct {
	SearchFunc func(query string, groupID uint, limit int) []KnowledgeSnippet
}

type KnowledgeSnippet struct {
	Title    string
	Content  string
	Score    float64
	Source   string
	DocID    string            // 文档/记忆ID，供知识来源点击跳转
	Metadata map[string]string
}

// KnowledgeSource 群助手回复命中的知识来源最小结构，随回复消息下发
// 供前端渲染「知识来源」折叠标签。只暴露标题/分数/来源类型/文档ID，不回传正文。
type KnowledgeSource struct {
	Title  string  `json:"title"`
	Score  float64 `json:"score"`
	Source string  `json:"source,omitempty"` // "knowledge" | "notes" | "memory"
	ID     string  `json:"id,omitempty"`     // 文档/记忆ID，供前端点击跳转
}

// memoryResultsToSources 把群记忆 Recall 结果转为 KnowledgeSource，
// 保留原始分数并标记 source=memory，供前端「知识来源」徽章区分展示。
func memoryResultsToSources(results []SearchResult) []KnowledgeSource {
	if len(results) == 0 {
		return nil
	}
	out := make([]KnowledgeSource, 0, len(results))
	for _, r := range results {
		// 分数门槛：低于 0.6 的记忆召回结果视为噪音，不展示在知识来源徽章
		if r.Score < 0.6 {
			continue
		}
		title := r.Metadata["title"]
		if title == "" {
			// 群记忆没有 title 字段，用 content 前 20 个字符作为展示标题
			if r.Content != "" {
				runes := []rune(r.Content)
				if len(runes) > 20 {
					title = string(runes[:20]) + "..."
				} else {
					title = r.Content
				}
			} else {
				title = "未命名"
			}
		}
		out = append(out, KnowledgeSource{Title: title, Score: r.Score, Source: "memory", ID: r.DocID})
	}
	return out
}

func NewUnifiedKnowledgeService(groupDocSvc *GroupDocumentService, fallback *LegacyKnowledgeFallback, scoreThreshold float64) *UnifiedKnowledgeService {
	if scoreThreshold <= 0 {
		scoreThreshold = 0.6 // 默认值
	}
	return &UnifiedKnowledgeService{
		groupDocSvc:    groupDocSvc,
		legacyFallback: fallback,
		vectorEnabled:  true,
		scoreThreshold: scoreThreshold,
	}
}

func (s *UnifiedKnowledgeService) Search(query string, groupID uint, limit int) []KnowledgeSnippet {
	if s.groupDocSvc != nil && s.vectorEnabled {
		resp, err := s.groupDocSvc.SearchKnowledgeWithMode(groupID, query, limit, "", false)
		if err == nil && len(resp.Results) > 0 {
			snippets := make([]KnowledgeSnippet, 0, len(resp.Results))
			for _, r := range resp.Results {
				snippets = append(snippets, KnowledgeSnippet{
					Title:    r.Title,
					Content:  r.Snippet,
					Score:    r.Score,
					Source:   "auto",
					DocID:    r.KnowledgeID,
					Metadata: r.Metadata,
				})
			}
			logger.WithModule("UnifiedKnowledge").Info("检索命中", "source", "auto", "count", len(snippets))
			return snippets
		}

		if err != nil {
			logger.WithModule("UnifiedKnowledge").Error("语义检索失败，降级到兜底", "error", err)
		}
	}

	if s.legacyFallback != nil {
		results := s.legacyFallback.SearchFunc(query, groupID, limit)
		if len(results) > 0 {
			logger.WithModule("UnifiedKnowledge").Info("MySQL兜底命中", "count", len(results))
			return results
		}
	}

	return nil
}

func (s *UnifiedKnowledgeService) BuildContext(query string, groupID uint) string {
	ctx, _ := s.BuildContextWithSources(query, groupID, 3)
	return ctx
}

// BuildContextWithSources 一次检索同时产出注入提示词的上下文串与命中的知识来源
// （仅标题/相关度），避免同一查询重复检索两遍。来源不携带文档正文，防止消息响应体过大。
func (s *UnifiedKnowledgeService) BuildContextWithSources(query string, groupID uint, limit int) (string, []KnowledgeSource) {
	if s == nil {
		return "", nil
	}
	snippets := s.Search(query, groupID, limit)
	if len(snippets) == 0 {
		return "", nil
	}

	var parts []string
	parts = append(parts, "📚 群知识库相关内容：")
	// 同一文档被分成多个块，多个块可能同时命中检索。上下文串与「知识来源」徽章
	// 统一按标题去重：同文档只保留得分最高的那块正文注入提示词（避免浪费 context
	// window），徽章只展示一条（避免重复展示好几条）。
	sources := make([]KnowledgeSource, 0, len(snippets))
	seen := make(map[string]bool, len(snippets))
	idx := 0
	for _, snip := range snippets {
		// 分数门槛：低于阈值的召回结果视为不相关，不注入 prompt 也不展示在徽章
		if snip.Score < s.scoreThreshold {
			continue
		}
		title := snip.Title
		if title == "" {
			title = "未命名"
		}
		if seen[title] {
			continue
		}
		seen[title] = true
		idx++
		sourceTag := fmt.Sprintf("（语义检索，相关度: %.1f%%）", snip.Score*100)
		parts = append(parts, fmt.Sprintf("[%d] %s %s\n%s", idx, snip.Title, sourceTag, snip.Content))
		sources = append(sources, KnowledgeSource{
			Title: title, Score: snip.Score, Source: "knowledge",
			ID: snip.DocID, // 文档ID供前端点击跳转
		})
	}

	return strings.Join(parts, "\n\n"), sources
}

func (s *UnifiedKnowledgeService) SetVectorEnabled(enabled bool) {
	s.vectorEnabled = enabled
}

// SetGraphEnhanced 保留以兼容调用；gracedb 语义层下图谱增强已并入语义检索，此开关无效。
func (s *UnifiedKnowledgeService) SetGraphEnhanced(enabled bool) {
	_ = enabled
}