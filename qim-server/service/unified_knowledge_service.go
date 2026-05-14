package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

type UnifiedKnowledgeService struct {
	groupDocSvc    *GroupDocumentService
	legacyFallback *LegacyKnowledgeFallback
	vectorEnabled  bool
	graphEnhanced  bool
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
		graphEnhanced:  false,
	}
}

func (s *UnifiedKnowledgeService) Search(query string, groupID uint, limit int) []KnowledgeSnippet {
	if s.groupDocSvc != nil && s.vectorEnabled {
		resp, err := s.groupDocSvc.SearchKnowledgeWithMode(groupID, query, limit, cortexdb.RetrievalModeAuto, s.graphEnhanced)
		if err == nil && len(resp.Results) > 0 {
			sourceTag := "auto"
			if s.graphEnhanced {
				sourceTag = "graph"
			}
			snippets := make([]KnowledgeSnippet, 0, len(resp.Results))
			for _, r := range resp.Results {
				snippets = append(snippets, KnowledgeSnippet{
					Title:    r.Title,
					Content:  r.Snippet,
					Score:    r.Score,
					Source:   sourceTag,
					Metadata: r.Metadata,
				})
			}
			log.Printf("[UnifiedKnowledge] %s检索命中 %d 条", sourceTag, len(snippets))
			return snippets
		}

		if err != nil {
			log.Printf("[UnifiedKnowledge] 语义检索失败，降级到词法: %v", err)
			return s.lexicalFallback(query, groupID, limit)
		}
	}

	if s.legacyFallback != nil {
		results := s.legacyFallback.SearchFunc(query, groupID, limit)
		if len(results) > 0 {
			log.Printf("[UnifiedKnowledge] MySQL 兜底命中 %d 条", len(results))
			return results
		}
	}

	return nil
}

func (s *UnifiedKnowledgeService) lexicalFallback(query string, groupID uint, limit int) []KnowledgeSnippet {
	if s.groupDocSvc == nil {
		return nil
	}

	resp, err := s.groupDocSvc.SearchKnowledgeWithMode(groupID, query, limit, cortexdb.RetrievalModeLexical, false)
	if err != nil || len(resp.Results) == 0 {
		return nil
	}

	log.Printf("[UnifiedKnowledge] 词法检索兜底命中 %d 条", len(resp.Results))
	snippets := make([]KnowledgeSnippet, 0, len(resp.Results))
	for _, r := range resp.Results {
		snippets = append(snippets, KnowledgeSnippet{
			Title:    r.Title,
			Content:  r.Snippet,
			Score:    r.Score,
			Source:   "lexical",
			Metadata: r.Metadata,
		})
	}
	return snippets
}

func (s *UnifiedKnowledgeService) BuildContext(query string, groupID uint) string {
	snippets := s.Search(query, groupID, 3)
	if len(snippets) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "📚 群知识库相关内容：")
	for i, snip := range snippets {
		sourceTag := ""
		switch snip.Source {
		case "auto":
			sourceTag = fmt.Sprintf("（语义检索，相关度: %.1f%%）", snip.Score*10)
		case "graph":
			sourceTag = fmt.Sprintf("（图谱增强检索，相关度: %.1f%%）", snip.Score*10)
		default:
			sourceTag = fmt.Sprintf("（词法检索，相关度: %.1f%%）", snip.Score*10)
		}
		parts = append(parts, fmt.Sprintf("[%d] %s %s\n%s", i+1, snip.Title, sourceTag, snip.Content))
	}

	return strings.Join(parts, "\n\n")
}

func (s *UnifiedKnowledgeService) SetVectorEnabled(enabled bool) {
	s.vectorEnabled = enabled
}

func (s *UnifiedKnowledgeService) SetGraphEnhanced(enabled bool) {
	s.graphEnhanced = enabled
}
