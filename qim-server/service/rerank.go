package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// KnowledgeReranker 对知识检索片段做相关性判定，过滤误召回。
//
// 与装配层（BuildContextWithSources）解耦：装配层只负责拼装 prompt 上下文与徽章，
// 不关心"这条知识要不要"。相关性判定是独立的 AI 能力，通过本接口注入，nil 时
// BuildContextWithSources 走纯装配路径（不做任何限制，行为与无 rerank 时一致）。
//
// 语义：Relevant 在注入前用用户提问判定，拦截分数虚高但无关的召回（如"兼职"撞出
// "马克思主义论文"），避免污染 prompt。徽章展示其通过的真实来源（不追"答案实引"，
// 避免为精确引用付出额外 LLM 调用与流式链路复杂度）。
//
// 关键语义：第三返回 confident=false 表示"拿不准"。调用方须"宁可留不可错杀"，
// 对拿不准的片段保留而不是丢弃——避免 LLM 只扫了几十字摘要就误杀相关文档。
type KnowledgeReranker interface {
	// Relevant 判定 snip 是否与 query 相关（注入前，query=用户提问）。
	//   relevant=true  → 相关，保留
	//   relevant=false, confident=true → 明确不相关，可过滤
	//   relevant=false, confident=false → 拿不准/判定失败，调用方保留
	// 返回的 err 非 nil 时，调用方应按"拿不准"处理（保留），不阻塞检索流程。
	Relevant(ctx context.Context, query string, snip KnowledgeSnippet) (relevant bool, confident bool, err error)
}

// LLMReranker 用 LLM 做相关性判定的实现。一次调用校验单条片段，输出 JSON 判定。
// 使用 TaskTypeAnalysis 路由到轻量模型；调用失败或解析失败时返回"拿不准"，由调用方谨慎保留。
type LLMReranker struct {
	ai *ai.AIService
	// judge 可覆写的判定函数，测试注入 mock；默认走 ai.GetCompletion。
	judge func(ctx context.Context, query, title, content string) (relevant bool, confident bool, err error)
}

// NewLLMReranker 构建基于 LLM 的相关性判定器。ai 为 nil 时返回 nil，
// 调用方会得到"拿不准"的分级语义（走保留路径），无需额外判错。
func NewLLMReranker(aiSvc *ai.AIService) *LLMReranker {
	if aiSvc == nil {
		return nil
	}
	r := &LLMReranker{ai: aiSvc}
	r.judge = r.defaultJudge
	return r
}

func (r *LLMReranker) Relevant(ctx context.Context, query string, snip KnowledgeSnippet) (bool, bool, error) {
	if r == nil || r.judge == nil {
		return false, false, nil // 无判定能力 → 拿不准，调用方保留
	}
	return r.judge(ctx, query, snip.Title, snip.Content)
}

// defaultJudge 构造判定 prompt，让 LLM 对单条片段输出 {"relevant": bool, "confident": bool}。
// 语义上鼓励"拿不准就 confident=false"，把误杀决策交给调用方。
func (r *LLMReranker) defaultJudge(ctx context.Context, query, title, content string) (bool, bool, error) {
	if title == "" {
		title = "未命名"
	}
	if n := len([]rune(content)); n > 200 {
		content = string([]rune(content)[:200]) + "..."
	}

	prompt := `你是一个知识检索相关性判断器。判断下面这条检索结果是否与用户问题真正相关。
只返回 JSON，格式：{"relevant": true|false, "confident": true|false}
- relevant：这条知识是否真正回答了用户问题。
- confident：你是否能明确下结论；若拿不准（如信息不足、语义模糊），必须设为 false。
不要返回 JSON 以外的任何内容。

用户问题：` + query + `

检索结果标题：` + title + `

检索结果内容：` + content

	messages := []ai.Message{{Role: "user", Content: prompt}}
	resp, err := r.ai.GetCompletion(ai.TaskTypeAnalysis, messages)
	if err != nil {
		logger.WithModule("UnifiedKnowledge").Warn("LLM 相关性判定失败", "error", err)
		return false, false, err // 拿不准，调用方保留
	}

	relevant, confident, ok := r.parseVerdict(resp)
	if !ok {
		logger.WithModule("UnifiedKnowledge").Warn("LLM 相关性判定解析失败，按拿不准处理", "raw", resp)
		return false, false, nil
	}
	return relevant, confident, nil
}

// verdict 从 LLM 返回的 JSON 判定中承载 {布尔判定, confident}。
type verdict struct {
	// 两个字段必须都被显式解析出；任缺一个即视为"拿不准"（ok=false），
	// 避免"判定键缺失但被当成 false 而误杀"。
	Relevant  *bool `json:"relevant"`
	Used      *bool `json:"used"`
	Confident *bool `json:"confident"`
}

// parseVerdict 解析 LLM 返回的判定 JSON。成功时 ok=true 且返回 relevant（used 同字段复用）与 confident。
// 关键约定：仅当两个布尔字段都被显式解析出时才视为合法（ok=true）；
// 任一缺失或整体解析失败一律 ok=false → 调用方按"拿不准"保留，绝不因缺字段而误杀。
func (r *LLMReranker) parseVerdict(resp string) (relevant bool, confident bool, ok bool) {
	obj := extractJSONObject(resp) // 已有跨结构复用的抽取函数：磨掉 markdown/前言
	if obj == "" {
		return false, false, false
	}
	var v verdict
	if err := json.Unmarshal([]byte(obj), &v); err != nil {
		return false, false, false
	}
	// 取 relevant 或 used 之一作为判定值；confident 必须有。
	decided := v.Relevant
	if decided == nil {
		decided = v.Used
	}
	if decided == nil || v.Confident == nil {
		return false, false, false // 或缺判定字段或缺 confident → 拿不准
	}
	return *decided, *v.Confident, true
}

// extractBoolField 从文本中查找形如 "key": true / "key": false 的字段值。
// 兼容纯文本兜底（parseVerdict 已优先走 JSON）。本函数不用于最终判定，
// 保留给极少数非 JSON 输出的降级识别；缺字段返回 false。
func extractBoolField(text, key string) bool {
	re := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*(true|false)`, key))
	m := re.FindStringSubmatch(text)
	if m == nil {
		return false
	}
	return m[1] == "true"
}
