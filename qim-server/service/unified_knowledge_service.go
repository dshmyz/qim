package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// rerankJudgeTimeout 知识检索 LLM 相关性校验阶段的总超时上限。判定串行逐条进行，
// 用该预算兜底模型逐步挂起/网络变慢，避免多条累加拖死整条回复；超时后保留未判片段。
const rerankJudgeTimeout = 15 * time.Second

type UnifiedKnowledgeService struct {
	groupDocSvc    *GroupDocumentService
	legacyFallback *LegacyKnowledgeFallback
	vectorEnabled  bool
	thresholdSvc   *AiThresholdService // 阈值读取服务（nil 时用默认值）
	aiService      *ai.AIService       // 保留引用以构造默认 reranker；nil 时不构造
	// reranker 相关性二次判定；nil 时走纯装配路径（不做任何限制）。
	// 与装配层解耦：装配层不关心"要不要这条"，只负责格式化/去重/阈值。
	reranker KnowledgeReranker
	// rerankJudgeTimeout LLM 相关性校验阶段的总超时预算（0 时用包级默认 15s）。
	// 提取为字段便于测试注入短超时，快速锁定"预算耗尽→保留未判片段"语义。
	rerankJudgeTimeout time.Duration
}

type LegacyKnowledgeFallback struct {
	SearchFunc func(query string, groupID uint, limit int) []KnowledgeSnippet
}

type KnowledgeSnippet struct {
	Title    string
	Content  string
	Score    float64
	Source   string
	DocID    string // 文档/记忆ID，供知识来源点击跳转
	Metadata map[string]string
}

// KnowledgeSource 群助手回复命中的知识来源最小结构，随回复消息下发
// 供前端渲染「知识来源」折叠标签。只暴露标题/分数/来源类型/文档ID，不回传正文。
type KnowledgeSource struct {
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Source  string  `json:"source,omitempty"`  // "knowledge" | "notes" | "memory"
	ID      string  `json:"id,omitempty"`      // 文档/记忆ID，供前端点击跳转
	Snippet string  `json:"snippet,omitempty"` // 命中正文摘要，供前端悬停/点击展示（知识库/记忆来源共用）；超长截断至 maxSnippetLen 防撑爆 UI。
}

// maxSnippetLen 摘要最大长度（ runes ）。超过时截取后加省略号，防止超长记忆/命中正文
// 撑爆浏览器 tooltip / toast / 消息 Extra 体积。120 字够展示一到两句话的实质内容。
const maxSnippetLen = 120

// clipSnippet 截断正文至 maxSnippetLen，超长加省略号。空串原样返回（不改语义）。
func clipSnippet(s string) string {
	if runes := []rune(s); len(runes) > maxSnippetLen {
		return string(runes[:maxSnippetLen]) + "…"
	}
	return s
}

// memoryResultsToSources 把群记忆 Recall 结果转为 KnowledgeSource，
// 保留原始分数并标记 source=memory，供前端「知识来源」徽章区分展示。
// threshold 为分数门槛：低于该分的召回结果视为噪音，不展示。
func memoryResultsToSources(results []SearchResult, threshold float64) []KnowledgeSource {
	if len(results) == 0 {
		return nil
	}
	// 降序排序：与群助手 selectTopKnowledge 语义一致，保证展示顺序 = 相关度顺序；
	// 召回超过期望条数时，靠排序 + 下方 threshold 硬下限拦纯噪音，避免低分真实命中被误杀。
	sorted := append([]SearchResult(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })

	out := make([]KnowledgeSource, 0, len(sorted))
	for _, r := range sorted {
		if r.Score < threshold {
			continue // 硬下限：拦住纯噪音记忆；真实低分命中（> threshold）保留
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
		out = append(out, KnowledgeSource{Title: title, Score: r.Score, Source: "memory", ID: r.DocID, Snippet: clipSnippet(r.Content)})
	}
	return out
}

func NewUnifiedKnowledgeService(groupDocSvc *GroupDocumentService, fallback *LegacyKnowledgeFallback, thresholdSvc *AiThresholdService, aiService *ai.AIService) *UnifiedKnowledgeService {
	return &UnifiedKnowledgeService{
		groupDocSvc:        groupDocSvc,
		legacyFallback:     fallback,
		vectorEnabled:      true,
		thresholdSvc:       thresholdSvc,
		aiService:          aiService,
		reranker:           NewLLMReranker(aiService),
		rerankJudgeTimeout: rerankJudgeTimeout, // 默认 15s 总预算
	}
}

// SetReranker 替换相关性判定器（测试注入 mock 或切换实现用）。传 nil 可关闭判定，走纯装配路径。
func (s *UnifiedKnowledgeService) SetReranker(r KnowledgeReranker) {
	if s == nil {
		return
	}
	s.reranker = r
}

func (s *UnifiedKnowledgeService) Search(query string, groupID uint, limit int) []KnowledgeSnippet {
	if s.groupDocSvc != nil && s.vectorEnabled {
		// 多拉候选以便相对排序后仍能取满 limit 条：单靠绝对分数门槛会误丢"唯一命中但
		// 词面相关度低"的真实文档（如长文件名课程 PDF），排序取前 N 更符合"这批里最相关"
		// 的语义。这里多取 limit*3，由 BuildContextWithSources 收敛到最终 limit 条。
		fetchLimit := limit * 3
		resp, err := s.groupDocSvc.SearchKnowledgeWithMode(groupID, query, fetchLimit, "", false)
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
			// 诊断：记录本轮检索 query 与命中的标题，用于定位"知识用了但徽章缺失"。
			_titles := make([]string, 0, len(snippets))
			for _, sn := range snippets {
				_titles = append(_titles, sn.Title)
			}
			logger.WithModule("UnifiedKnowledge").Info("检索命中", "query", query, "source", "auto", "count", len(snippets), "titles", _titles)
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

	// 相关性二次判定：过滤误召回。这里用的是宽进严出前的"宁可留不可错杀"语义——
	// 只对"明确不相关"的片段做过滤，拿不准/判定失败的保留。
	snippets = s.filterByReranker(query, snippets)
	if len(snippets) == 0 {
		return "", nil
	}

	// 硬下限：低于此分数的召回视为纯噪音，直接丢弃。与"相对排序取前 limit 条"配合：
	// 唯一命中但词面相关度偏低（如长文件名课程 PDF）的文档，只要高于硬下限即保留，
	// 不再受旧版绝对门槛（0.6）误杀。默认 0.3 仅拦真实噪音。
	floor := 0.3
	if s.thresholdSvc != nil {
		floor = s.thresholdSvc.GetFloat("ai.knowledge_score_threshold", 0.3)
	}

	picked := s.selectTopKnowledge(floor, limit, snippets)

	var parts []string
	parts = append(parts, "📚 群知识库相关内容：")
	// picked 已由 selectTopKnowledge 完成：硬下限去噪、同标题去重（保留最高分块）、
	// 分数降序、截取前 limit 条。这里只负责拼注入串与徽章。
	sources := make([]KnowledgeSource, 0, len(picked))
	for idx, snip := range picked {
		title := snip.Title
		if title == "" {
			title = "未命名"
		}
		sourceTag := fmt.Sprintf("（语义检索，相关度: %.1f%%）", snip.Score*100)
		parts = append(parts, fmt.Sprintf("[%d] %s %s\n%s", idx+1, snip.Title, sourceTag, snip.Content))
		sources = append(sources, KnowledgeSource{
			Title: title, Score: snip.Score, Source: "knowledge",
			ID:      snip.DocID,                // 文档ID供前端点击跳转
			Snippet: clipSnippet(snip.Content), // 命中正文摘要，供前端「来源小字/悬停」展示；超长截断防 UI 撑爆
		})
	}

	// 诊断：本 query 最终产出到徽章的知识来源数量。为 0 即"知识上下文存在但徽章空"的信号，
	// 借此与"检索未命中 / 被 reranker 滤掉 / 全部低于门槛"区分开。
	logger.WithModule("UnifiedKnowledge").Info("上下文与徽章产出",
		"query", query, "snippet_count", len(snippets), "badge_sources", len(sources))

	return strings.Join(parts, "\n\n"), sources
}

// selectTopKnowledge 从检索候选里选出注入 prompt 与徽章的知识：硬下限去噪 → 按标题去重
// （同文档多块保留最高分块）→ 分数降序 → 截取前 limit 条。纯函数，可独立单测。
// 相对排序而非绝对门槛：保证"唯一命中但词面相关度偏低"的真实文档也能进 top-N，
// 同时用硬下限拦住明确无关的召回。
func (s *UnifiedKnowledgeService) selectTopKnowledge(floor float64, limit int, snippets []KnowledgeSnippet) []KnowledgeSnippet {
	// 去重：同文档多块保留最高分块。以 DocID 为主键（DocID 为空时回退到 title），
	// 确保不同文档但同标题（如都叫"背景"）的内容都被保留，不被误折叠。
	byDoc := make(map[string]KnowledgeSnippet)
	for _, snip := range snippets {
		if snip.Score < floor {
			continue // 纯噪音，丢弃
		}
		key := snip.DocID
		if key == "" {
			key = snip.Title
			if key == "" {
				key = "未命名"
			}
		}
		cur, ok := byDoc[key]
		if !ok || snip.Score > cur.Score {
			byDoc[key] = snip
		}
	}
	out := make([]KnowledgeSnippet, 0, len(byDoc))
	for _, snip := range byDoc {
		out = append(out, snip)
	}
	// 分数降序稳定排序
	for i := 1; i < len(out); i++ {
		for j := 0; j < len(out)-i; j++ {
			if out[j].Score < out[j+1].Score {
				out[j], out[j+1] = out[j+1], out[j]
			}
		}
	}
	// 截取前 limit 条
	n := limit
	if n <= 0 || n > len(out) {
		n = len(out)
	}
	return out[:n]
}

// filterByReranker 依相关性判定结果过滤片段。判定器 nil 或判定"拿不准/失败"时保留，
// 仅"明确不相关"（relevant=false 且 confident=true）被过滤。并发安全：逐条独立判定，
// 结果按原序稳定输出。ai.knowledge_llm_rerank=0 时跳过判定（保留全部），供后台一键关闭。
func (s *UnifiedKnowledgeService) filterByReranker(query string, snippets []KnowledgeSnippet) []KnowledgeSnippet {
	if s == nil || s.reranker == nil || len(snippets) == 0 {
		return snippets
	}
	if s.thresholdSvc != nil && s.thresholdSvc.GetInt("ai.knowledge_llm_rerank", 1) == 0 {
		logger.WithModule("UnifiedKnowledge").Info("知识检索：LLM 校验已关闭，走纯阈值模式", "query", query, "count", len(snippets))
		return snippets
	}

	// 校验阶段整体加超时上限。LLM 判定是串行的（每条一次调用，最多 limit*3 条），
	// 单次调用虽受 provider 的 HTTP 超时兜底（~120s），多条累加会拖住整条回复。预算耗尽即
	// 停止发起新判定并保留剩余未判片段（宁可多留不可阻塞），GetCompletion 非流式暂未透传 ctx，
	// 故在逐条间的 ctx.Err() 检查处提前退出。预算取字段值，0 时回退包级默认 15s。
	timeout := s.rerankJudgeTimeout
	if timeout <= 0 {
		timeout = rerankJudgeTimeout
	}
	judgeCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	filtered := make([]KnowledgeSnippet, 0, len(snippets))
	for i, snip := range snippets {
		if judgeCtx.Err() != nil {
			logger.WithModule("UnifiedKnowledge").Warn("知识检索：LLM 校验超时，停止判定并保留剩余片段",
				"query", query, "judged", i, "count", len(snippets))
			filtered = append(filtered, snippets[i:]...)
			break
		}
		relevant, confident, err := s.reranker.Relevant(judgeCtx, query, snip)
		if err != nil || !confident {
			// 判定失败或拿不准 → 保留，避免误杀相关文档
			filtered = append(filtered, snip)
			continue
		}
		if relevant {
			filtered = append(filtered, snip)
			continue
		}
		// 明确不相关 → 过滤
		logger.WithModule("UnifiedKnowledge").Info("知识检索：LLM 判定不相关，已过滤",
			"query", query, "title", snip.Title, "score", snip.Score)
	}

	if len(filtered) != len(snippets) {
		logger.WithModule("UnifiedKnowledge").Info("知识检索：LLM 校验完成",
			"query", query, "before", len(snippets), "after", len(filtered))
	}
	return filtered
}

func (s *UnifiedKnowledgeService) SetVectorEnabled(enabled bool) {
	s.vectorEnabled = enabled
}

// SetGraphEnhanced 保留以兼容调用；gracedb 语义层下图谱增强已并入语义检索，此开关无效。
func (s *UnifiedKnowledgeService) SetGraphEnhanced(enabled bool) {
	_ = enabled
}
