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
	snippets := s.Search(query, groupID, 3)
	if len(snippets) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "📚 群知识库相关内容：")
	for i, snip := range snippets {
		sourceTag := fmt.Sprintf("（语义检索，相关度: %.1f%%）", snip.Score*10)
		parts = append(parts, fmt.Sprintf("[%d] %s %s\n%s", i+1, snip.Title, sourceTag, snip.Content))
	}

	return strings.Join(parts, "\n\n")
}

func (s *UnifiedKnowledgeService) SetVectorEnabled(enabled bool) {
	s.vectorEnabled = enabled
}

// SetGraphEnhanced 保留以兼容调用；gracedb 语义层下图谱增强已并入语义检索，此开关无效。
func (s *UnifiedKnowledgeService) SetGraphEnhanced(enabled bool) {
	_ = enabled
}