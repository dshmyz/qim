package service

import (
	"context"
	"log"

	"github.com/dshmyz/qim/qim-server/ai"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type EinoChatModel struct {
	aiService *ai.AIService
	taskType  ai.TaskType
	userID    uint
	useTools  bool
}

func NewEinoChatModel(aiService *ai.AIService, taskType ai.TaskType, userID uint) *EinoChatModel {
	return &EinoChatModel{
		aiService: aiService,
		taskType:  taskType,
		userID:    userID,
		useTools:  true,
	}
}

func NewEinoChatModelNoTools(aiService *ai.AIService, taskType ai.TaskType, userID uint) *EinoChatModel {
	return &EinoChatModel{
		aiService: aiService,
		taskType:  taskType,
		userID:    userID,
		useTools:  false,
	}
}

func (m *EinoChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	aiMessages := einoMessagesToAIMessages(input)

	var reply string
	var err error

	uid := m.userID
	if v, ok := UserIDFromCtx(ctx); ok {
		uid = v
	}
	callerCtx := &ai.CallerContext{UserID: uid}
	if m.useTools {
		reply, err = m.aiService.GetCompletionWithTools(m.taskType, aiMessages, callerCtx)
	} else {
		reply, err = m.aiService.GetCompletion(m.taskType, aiMessages)
	}
	if err != nil {
		return nil, err
	}

	return &schema.Message{
		Role:    schema.Assistant,
		Content: reply,
	}, nil
}

func (m *EinoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	aiMessages := einoMessagesToAIMessages(input)

	sr, sw := schema.Pipe[*schema.Message](0)

	go func() {
		defer sw.Close()

		err := m.aiService.GetCompletionStreamWithContext(ctx, m.taskType, aiMessages, func(chunk ai.StreamChunk) error {
			msg := &schema.Message{
				Role:    schema.Assistant,
				Content: chunk.Content,
			}
			sw.Send(msg, nil)
			return nil
		})

		if err != nil {
			log.Printf("[EinoChatModel] Stream error: %v", err)
			sw.Send(nil, err)
		}
	}()

	return sr, nil
}

func (m *EinoChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func einoMessagesToAIMessages(messages []*schema.Message) []ai.Message {
	result := make([]ai.Message, len(messages))
	for i, msg := range messages {
		role := string(msg.Role)
		result[i] = ai.Message{
			Role:    role,
			Content: msg.Content,
		}
		// 多模态：从 MultiContent 提取图片 URL（群 AI 被引用图片以 data URL 注入），
		// 落到 ai.Message.ImageURL 后由 MarshalJSON 转成 OpenAI image_url 数组格式。
		if imgURL := imageURLFromMessage(msg); imgURL != "" {
			result[i].ImageURL = imgURL
		}
	}
	return result
}

// imageURLFromMessage 从 eino schema.Message 的 MultiContent 提取首个 image_url 部分的数据 URL；
// 无图片部分返回空串。
func imageURLFromMessage(msg *schema.Message) string {
	for _, part := range msg.MultiContent {
		if part.Type == schema.ChatMessagePartTypeImageURL && part.ImageURL != nil && part.ImageURL.URL != "" {
			return part.ImageURL.URL
		}
	}
	return ""
}

var _ model.ToolCallingChatModel = (*EinoChatModel)(nil)

// userIDCtxKey 用于通过 context.Context 把当前提问用户的 ID 传给
// 编译期写死 userID=0 的 graph model 节点（见 SmartReplyGraph.Execute），
// 使 GroupManagementTool 等工具执行时 callerCtx.UserID 为真实用户、
// isSystemAdmin 校验生效，避免 userID=0 绕过权限。
type userIDCtxKey struct{}

// UserIDToCtx 把 userID 注入 ctx。
func UserIDToCtx(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, userID)
}

// UserIDFromCtx 从 ctx 取出 userID。
func UserIDFromCtx(ctx context.Context) (uint, bool) {
	uid, ok := ctx.Value(userIDCtxKey{}).(uint)
	return uid, ok
}
