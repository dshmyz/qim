package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"qim-server/ai"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type TextProcessIntent string

const (
	TextProcessTranslate TextProcessIntent = "translate"
	TextProcessRewrite   TextProcessIntent = "rewrite"
	TextProcessPolish    TextProcessIntent = "polish"
)

type TextProcessInput struct {
	Intent     TextProcessIntent
	Text       string
	TargetLang string
	SourceLang string
	Style      string
	Tone       string
	Language   string
}

type TextProcessOutput struct {
	Result string
}

type TextProcessGraph struct {
	runnable  compose.Runnable[*TextProcessInput, *TextProcessOutput]
	aiService *ai.AIService
	cache     *AICache
}

func NewTextProcessGraph(aiService *ai.AIService, cache *AICache) *TextProcessGraph {
	return &TextProcessGraph{
		aiService: aiService,
		cache:     cache,
	}
}

func (g *TextProcessGraph) Build() error {
	EnsureMergeRegistered(func(vs []*TextProcessInput) (*TextProcessInput, error) {
		return vs[0], nil
	})

	graph := compose.NewGraph[*TextProcessInput, *TextProcessOutput]()

	graph.AddLambdaNode("build_prompt", compose.InvokableLambda(g.buildPrompt))
	AddModelNode(graph, g.aiService)
	graph.AddLambdaNode("format", compose.InvokableLambda(g.format))

	graph.AddEdge(compose.START, "build_prompt")
	graph.AddEdge("build_prompt", "model")
	graph.AddEdge("model", "format")
	graph.AddEdge("format", compose.END)

	runnable, err := compileGraph(graph, "TextProcess")
	if err != nil {
		return fmt.Errorf("编译 TextProcess Graph 失败: %w", err)
	}
	g.runnable = runnable
	return nil
}

func (g *TextProcessGraph) Execute(ctx context.Context, input *TextProcessInput) (*TextProcessOutput, error) {
	cacheKey := g.generateCacheKey(input)
	return executeWithCache(ctx, g.cache, cacheKey, g.cacheTTL(input), g.runnable, input)
}

func (g *TextProcessGraph) cacheTTL(input *TextProcessInput) time.Duration {
	if input.Intent == TextProcessTranslate {
		return time.Hour * 24 * 365
	}
	return time.Hour * 24
}

func (g *TextProcessGraph) generateCacheKey(input *TextProcessInput) string {
	var parts []string
	parts = append(parts, string(input.Intent))
	parts = append(parts, input.Text)
	switch input.Intent {
	case TextProcessTranslate:
		parts = append(parts, input.SourceLang, input.TargetLang)
	case TextProcessRewrite:
		parts = append(parts, input.Style, input.Tone)
	case TextProcessPolish:
		parts = append(parts, input.Language)
	}
	return g.cache.GenerateKey(parts...)
}

func (g *TextProcessGraph) buildPrompt(ctx context.Context, input *TextProcessInput) ([]*schema.Message, error) {
	pb := NewPromptBuilder(g.roleForIntent(input)).AddRules(g.rulesForIntent(input.Intent)...)
	for k, v := range g.paramsForIntent(input) {
		pb.SetParam(k, v)
	}
	return pb.ToMessages(input.Text), nil
}

func (g *TextProcessGraph) format(ctx context.Context, msg *schema.Message) (*TextProcessOutput, error) {
	if msg == nil {
		return nil, fmt.Errorf("模型返回空消息")
	}
	return &TextProcessOutput{Result: strings.TrimSpace(msg.Content)}, nil
}

func (g *TextProcessGraph) roleForIntent(input *TextProcessInput) string {
	switch input.Intent {
	case TextProcessTranslate:
		return "你是 QIM 企业即时通讯系统的翻译助手。你的任务是准确、流畅地翻译文本。"
	case TextProcessRewrite:
		return "你是 QIM 企业即时通讯系统的改写助手。你的任务是改写文本使其更符合特定风格和语气。"
	case TextProcessPolish:
		return "你是 QIM 企业即时通讯系统的润色助手。你的任务是润色文本使其更加专业和流畅。"
	default:
		return "你是 QIM 企业即时通讯系统的文本处理助手。"
	}
}

func (g *TextProcessGraph) rulesForIntent(input TextProcessIntent) []string {
	switch input {
	case TextProcessTranslate:
		return []string{
			"1. 保持原文的语义和语气",
			"2. 使用目标语言的自然表达方式",
			"3. 保留专业术语和专有名词",
			"4. 保持原文的格式和结构",
		}
	case TextProcessRewrite:
		return []string{
			"1. 保持原文的核心意思",
			"2. 调整表达方式以符合指定风格",
			"3. 确保改写后的文本流畅自然",
			"4. 保持原文的格式和结构",
		}
	case TextProcessPolish:
		return []string{
			"1. 修正语法和拼写错误",
			"2. 优化句子结构和表达",
			"3. 保持原文的语义和语气",
			"4. 使文本更加简洁和专业",
		}
	default:
		return nil
	}
}

func (g *TextProcessGraph) paramsForIntent(input *TextProcessInput) map[string]string {
	params := make(map[string]string)
	switch input.Intent {
	case TextProcessTranslate:
		if input.SourceLang != "" {
			params["源语言"] = input.SourceLang
		}
		if input.TargetLang != "" {
			params["目标语言"] = input.TargetLang
		}
	case TextProcessRewrite:
		if input.Style != "" {
			params["风格"] = input.Style
		}
		if input.Tone != "" {
			params["语气"] = input.Tone
		}
	case TextProcessPolish:
		if input.Language != "" {
			params["语言"] = input.Language
		}
	}
	return params
}
