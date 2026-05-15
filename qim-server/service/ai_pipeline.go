package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"qim-server/ai"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// compileGraph compiles an Eino graph with a one-time value merge registration.
// Each concrete graph should call this in its Build() method with its own node setup.
func compileGraph[Input, Output any](graph *compose.Graph[Input, Output], name string) (compose.Runnable[Input, Output], error) {
	runnable, err := graph.Compile(context.Background(), compose.WithGraphName(name))
	if err != nil {
		return nil, fmt.Errorf("编译 %s Graph 失败: %w", name, err)
	}
	return runnable, nil
}

// registerMergeFunc ensures RegisterValuesMergeFunc is called at most once globally.
var registerMergeOnce sync.Once

// EnsureMergeRegistered registers a values merge function for the given type.
// Call this once per concrete Input type in the graph's Build() method.
func EnsureMergeRegistered[T any](mergeFunc func([]*T) (*T, error)) {
	registerMergeOnce.Do(func() {
		// Since sync.Once only fires once, we use a generic wrapper approach.
		// In practice, the first registered merge func wins.
		// For different types, Eino handles them separately.
		compose.RegisterValuesMergeFunc(mergeFunc)
	})
}

// executeWithCache runs the graph with automatic cache check/compute/set.
func executeWithCache[Input any, Output any](
	ctx context.Context,
	cache *AICache,
	cacheKey string,
	cacheTTL time.Duration,
	runnable compose.Runnable[Input, Output],
	input Input,
) (Output, error) {
	var zeroOutput Output
	if cache != nil && cacheKey != "" {
		if cached, ok := cache.Get(cacheKey); ok {
			if err := json.Unmarshal([]byte(cached), &zeroOutput); err == nil {
				return zeroOutput, nil
			}
		}
	}

	if runnable == nil {
		return zeroOutput, fmt.Errorf("Graph 未编译")
	}

	result, err := runnable.Invoke(ctx, input)
	if err != nil {
		return zeroOutput, err
	}

	if cache != nil && cacheKey != "" {
		data, _ := json.Marshal(result)
		cache.Set(cacheKey, string(data), cacheTTL)
	}

	return result, nil
}

// simpleExecuteWithCache for graphs with string output (e.g., AvatarReplyGraph returns string directly).
func simpleExecuteWithCache[Input any](
	ctx context.Context,
	cache *AICache,
	cacheKey string,
	cacheTTL time.Duration,
	runnable compose.Runnable[Input, string],
	input Input,
) (string, error) {
	if cache != nil && cacheKey != "" {
		if cached, ok := cache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	if runnable == nil {
		return "", fmt.Errorf("Graph 未编译")
	}

	result, err := runnable.Invoke(ctx, input)
	if err != nil {
		return "", err
	}

	if cache != nil && cacheKey != "" {
		cache.Set(cacheKey, result, cacheTTL)
	}

	return result, nil
}

// AddModelNode adds a chat model node to the graph.
func AddModelNode[Input, Output any](graph *compose.Graph[Input, Output], aiService *ai.AIService) {
	graph.AddChatModelNode("model", NewEinoChatModel(aiService))
}

// PipelineState holds shared data between pipeline nodes.
type PipelineState struct {
	Vars map[string]any
}

func NewPipelineState() *PipelineState {
	return &PipelineState{Vars: make(map[string]any)}
}

// PipelineNode is a step in the AI pipeline.
// Each node receives the current state and can read/write vars.
type PipelineNode interface {
	// NodeType returns the Eino node type for this step.
	NodeType() string
}

// LambdaNode wraps a lambda function as a pipeline node.
type LambdaNode[In, Out any] struct {
	name string
	fn   any // compose.Lambda function
}

// PromptBuilder helps construct system/user prompt pairs.
type PromptBuilder struct {
	role       string
	rules      []string
	params     map[string]string
	userPrefix string
}

func NewPromptBuilder(role string) *PromptBuilder {
	return &PromptBuilder{
		role:   role,
		params: make(map[string]string),
	}
}

func (b *PromptBuilder) AddRule(rule string) *PromptBuilder {
	b.rules = append(b.rules, rule)
	return b
}

func (b *PromptBuilder) AddRules(rules ...string) *PromptBuilder {
	b.rules = append(b.rules, rules...)
	return b
}

func (b *PromptBuilder) SetParam(key, value string) *PromptBuilder {
	b.params[key] = value
	return b
}

func (b *PromptBuilder) SetUserPrefix(prefix string) *PromptBuilder {
	b.userPrefix = prefix
	return b
}

func (b *PromptBuilder) BuildSystem() string {
	var sb strings.Builder
	sb.WriteString(b.role)
	sb.WriteString("\n\n")

	if len(b.rules) > 0 {
		sb.WriteString("【规则】\n")
		for _, r := range b.rules {
			sb.WriteString(r)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	for k, v := range b.params {
		sb.WriteString(fmt.Sprintf("【%s】%s\n", k, v))
	}

	return sb.String()
}

func (b *PromptBuilder) BuildUser(content string) string {
	var sb strings.Builder
	if b.userPrefix != "" {
		sb.WriteString(b.userPrefix)
		sb.WriteString("\n\n")
	}
	sb.WriteString(content)
	return sb.String()
}

func (b *PromptBuilder) ToMessages(userContent string) []*schema.Message {
	return []*schema.Message{
		{Role: schema.System, Content: b.BuildSystem()},
		{Role: schema.User, Content: b.BuildUser(userContent)},
	}
}
