package service

import (
	"context"
	"log"

	"qim-server/ai"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type EinoChatModel struct {
	aiService *ai.AIService
	userID    uint
}

func NewEinoChatModel(aiService *ai.AIService, userID uint) *EinoChatModel {
	return &EinoChatModel{
		aiService: aiService,
		userID:    userID,
	}
}

func (m *EinoChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	aiMessages := einoMessagesToAIMessages(input)

	var reply string
	var err error

	callerCtx := &ai.CallerContext{UserID: m.userID}
	reply, err = m.aiService.GetCompletionWithTools(aiMessages, callerCtx)
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

	// 打印发送给模型的 Prompt，方便排查拦截原因
	for _, msg := range aiMessages {
		log.Printf("[EinoChatModel] [%s]: %s", msg.Role, msg.Content)
	}

	sr, sw := schema.Pipe[*schema.Message](0)

	go func() {
		defer sw.Close()

		err := m.aiService.GetCompletionStream(aiMessages, func(chunk ai.StreamChunk) error {
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
	}
	return result
}

var _ model.ToolCallingChatModel = (*EinoChatModel)(nil)
