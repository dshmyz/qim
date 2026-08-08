package service

import (
	"context"

	"github.com/dshmyz/qim/qim-server/ai"
)

// fakeAvatarProvider 测试用 Provider 桩：按配置返回固定回复，用于在无需真实 LLM
// 的情况下验证分身触发/回复的意图判断与置信度门控等 fail-closed 逻辑。
type fakeAvatarProvider struct {
	reply string // Chat 返回的内容（通常是一段 JSON）
}

var _ ai.Provider = (*fakeAvatarProvider)(nil)

func (f *fakeAvatarProvider) Name() string { return "fake-avatar" }
func (f *fakeAvatarProvider) Chat(messages []ai.Message) (string, error) {
	return f.reply, nil
}
func (f *fakeAvatarProvider) ChatStream(messages []ai.Message, onChunk func(chunk ai.StreamChunk) error) error {
	return nil
}
func (f *fakeAvatarProvider) ChatStreamWithContext(ctx context.Context, messages []ai.Message, onChunk func(chunk ai.StreamChunk) error) error {
	return nil
}
func (f *fakeAvatarProvider) Embedding(text string) ([]float32, error) {
	return nil, nil
}
func (f *fakeAvatarProvider) ChatWithTools(messages []ai.Message, tools []ai.ToolDef) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: f.reply}, nil
}
func (f *fakeAvatarProvider) ChatStreamWithTools(context.Context, []ai.Message, []ai.ToolDef, func(ai.StreamChunk) error) error {
	return ai.ErrStreamingToolsNotSupported
}
func (f *fakeAvatarProvider) IsConfigured() bool { return true }
func (f *fakeAvatarProvider) WithModel(model string) ai.Provider {
	// 返回带 model 的副本（保持 reply 不变）
	return f
}

// newFakeAvatarAIService 构造一个注入了 fake provider 的 AIService，并确保路由能选中它。
func newFakeAvatarAIService(reply string) *ai.AIService {
	svc := ai.NewAIService(&ai.AIConfig{})
	svc.SetProviderForTesting("fake-avatar", &fakeAvatarProvider{reply: reply})
	return svc
}
