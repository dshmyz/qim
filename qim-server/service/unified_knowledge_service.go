package service

import (
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

type UnifiedKnowledgeService struct {
	groupDocSvc    *GroupDocumentService
	legacyFallback *LegacyKnowledgeFallback
	vectorEnabled  bool
}

type LegacyKnowledgeFallback struct {
	SearchFunc func(query string, groupID uint, limit int) []KnowledgeSnippet
}

type KnowledgeSnippet struct {
	Title    string
	Content  string
	Score    float64
	Source   string
	Metadata map[string]string
}

// KnowledgeSource 群助手回复命中的知识来源最小结构（仅标题/相关度），随回复消息下发
// 供前端渲染「知识来源」折叠标签。与 Bot 命中笔记的 knowledge_sources 结构对齐，
// 只暴露标题与分数，不回传文档正文/片段，避免消息响应体过大或泄漏。
type KnowledgeSource struct {
	Title string  `json:"title"`
	Score float64 `json:"score"`
}

func NewUnifiedKnowledgeService(groupDocSvc *GroupDocumentService, fallback *LegacyKnowledgeFallback) *UnifiedKnowledgeService {
	return &UnifiedKnowledgeService{
		groupDocSvc:    groupDocSvc,
		legacyFallback: fallback,
		vectorEnabled:  true,
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
	// 同一文档被分成多个块，多个块可能同时命中检索。上下文串保留各块不同正文
	// （供模型参考文档不同段落），但「知识来源」徽章按标题去重，同文档只保留
	// 得分最高的一条，避免同一文档名重复展示好几条。
	sources := make([]KnowledgeSource, 0, len(snippets))
	bestScore := make(map[string]float64, len(snippets))
	for i, snip := range snippets {
		sourceTag := fmt.Sprintf("（语义检索，相关度: %.1f%%）", snip.Score*100)
		parts = append(parts, fmt.Sprintf("[%d] %s %s\n%s", i+1, snip.Title, sourceTag, snip.Content))
		title := snip.Title
		if title == "" {
			title = "未命名"
		}
		if prev, ok := bestScore[title]; !ok || snip.Score > prev {
			bestScore[title] = snip.Score
		}
	}
	for _, snip := range snippets {
		title := snip.Title
		if title == "" {
			title = "未命名"
		}
		if snip.Score == bestScore[title] {
			sources = append(sources, KnowledgeSource{Title: title, Score: snip.Score})
			delete(bestScore, title) // 只取第一条达到最高分的块
		}
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