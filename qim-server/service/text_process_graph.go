package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

var registerTextMergeOnce sync.Once

func NewTextProcessGraph(aiService *ai.AIService, cache *AICache) *TextProcessGraph {
	return &TextProcessGraph{
		aiService: aiService,
		cache:     cache,
	}
}

func (g *TextProcessGraph) Build() error {
	registerTextMergeOnce.Do(func() {
		compose.RegisterValuesMergeFunc(func(vs []*TextProcessInput) (*TextProcessInput, error) {
			return vs[0], nil
		})
	})

	graph := compose.NewGraph[*TextProcessInput, *TextProcessOutput]()

	graph.AddLambdaNode("build_prompt", g.createBuildPromptNode())
	graph.AddChatModelNode("model", NewEinoChatModel(g.aiService, 0))
	graph.AddLambdaNode("format", g.createFormatNode())

	graph.AddEdge(compose.START, "build_prompt")
	graph.AddEdge("build_prompt", "model")
	graph.AddEdge("model", "format")
	graph.AddEdge("format", compose.END)

	ctx := context.Background()
	runnable, err := graph.Compile(ctx, compose.WithGraphName("TextProcess"))
	if err != nil {
		return fmt.Errorf("编译 TextProcess Graph 失败: %w", err)
	}
	g.runnable = runnable
	return nil
}

func (g *TextProcessGraph) Execute(ctx context.Context, input *TextProcessInput) (*TextProcessOutput, error) {
	cacheKey := g.generateCacheKey(input)
	if cached, ok := g.cache.Get(cacheKey); ok {
		return &TextProcessOutput{Result: cached}, nil
	}

	if g.runnable == nil {
		return nil, fmt.Errorf("TextProcessGraph 未编译")
	}

	result, err := g.runnable.Invoke(ctx, input)
	if err != nil {
		return nil, err
	}

	ttl := time.Hour * 24
	if input.Intent == TextProcessTranslate {
		ttl = time.Hour * 24 * 365
	}
	g.cache.Set(cacheKey, result.Result, ttl)
	return result, nil
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

func (g *TextProcessGraph) createBuildPromptNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *TextProcessInput) ([]*schema.Message, error) {
		var result []*schema.Message

		systemPrompt := g.buildSystemPrompt(input)
		result = append(result, &schema.Message{Role: schema.System, Content: systemPrompt})

		userPrompt := g.buildUserPrompt(input)
		result = append(result, &schema.Message{Role: schema.User, Content: userPrompt})

		return result, nil
	})
}

func (g *TextProcessGraph) createFormatNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*TextProcessOutput, error) {
		if msg == nil {
			return nil, fmt.Errorf("模型返回空消息")
		}

		content := strings.TrimSpace(msg.Content)
		return &TextProcessOutput{Result: content}, nil
	})
}

func (g *TextProcessGraph) buildSystemPrompt(input *TextProcessInput) string {
	var sb strings.Builder

	switch input.Intent {
	case TextProcessTranslate:
		sb.WriteString("你是 QIM 企业即时通讯系统的翻译助手。你的任务是准确、流畅地翻译文本。\n\n")
		sb.WriteString("【翻译规则】\n")
		sb.WriteString("1. 保持原文的语义和语气\n")
		sb.WriteString("2. 使用目标语言的自然表达方式\n")
		sb.WriteString("3. 保留专业术语和专有名词\n")
		sb.WriteString("4. 保持原文的格式和结构\n")
		if input.SourceLang != "" {
			sb.WriteString(fmt.Sprintf("\n【源语言】%s\n", input.SourceLang))
		}
		if input.TargetLang != "" {
			sb.WriteString(fmt.Sprintf("【目标语言】%s\n", input.TargetLang))
		}

	case TextProcessRewrite:
		sb.WriteString("你是 QIM 企业即时通讯系统的改写助手。你的任务是改写文本使其更符合特定风格和语气。\n\n")
		sb.WriteString("【改写规则】\n")
		sb.WriteString("1. 保持原文的核心意思\n")
		sb.WriteString("2. 调整表达方式以符合指定风格\n")
		sb.WriteString("3. 确保改写后的文本流畅自然\n")
		sb.WriteString("4. 保持原文的格式和结构\n")
		if input.Style != "" {
			sb.WriteString(fmt.Sprintf("\n【风格】%s\n", input.Style))
		}
		if input.Tone != "" {
			sb.WriteString(fmt.Sprintf("【语气】%s\n", input.Tone))
		}

	case TextProcessPolish:
		sb.WriteString("你是 QIM 企业即时通讯系统的润色助手。你的任务是润色文本使其更加专业和流畅。\n\n")
		sb.WriteString("【润色规则】\n")
		sb.WriteString("1. 修正语法和拼写错误\n")
		sb.WriteString("2. 优化句子结构和表达\n")
		sb.WriteString("3. 保持原文的语义和语气\n")
		sb.WriteString("4. 使文本更加简洁和专业\n")
		if input.Language != "" {
			sb.WriteString(fmt.Sprintf("\n【语言】%s\n", input.Language))
		}
	}

	return sb.String()
}

func (g *TextProcessGraph) buildUserPrompt(input *TextProcessInput) string {
	var sb strings.Builder

	switch input.Intent {
	case TextProcessTranslate:
		sb.WriteString("请翻译以下文本：\n\n")
	case TextProcessRewrite:
		sb.WriteString("请改写以下文本：\n\n")
	case TextProcessPolish:
		sb.WriteString("请润色以下文本：\n\n")
	}

	sb.WriteString(input.Text)
	return sb.String()
}
