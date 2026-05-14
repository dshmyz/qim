package service

import (
	"context"
	"fmt"

	"qim-server/database"
	"qim-server/model"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type MessageRetriever struct {
	conversationID uint
	userID         uint
	topK           int
}

func NewMessageRetriever(conversationID, userID uint, topK int) *MessageRetriever {
	if topK <= 0 {
		topK = 10
	}
	return &MessageRetriever{
		conversationID: conversationID,
		userID:         userID,
		topK:           topK,
	}
}

func (r *MessageRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	topK := r.topK
	opt := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	if opt.TopK != nil {
		topK = *opt.TopK
	}

	db := database.GetDB()

	var messages []model.Message
	queryBuilder := db.Where("type = 'text'")

	if r.conversationID > 0 {
		queryBuilder = queryBuilder.Where("conversation_id = ?", r.conversationID)
	}

	if query != "" {
		queryBuilder = queryBuilder.Where("content LIKE ?", "%"+query+"%")
	}

	result := queryBuilder.Preload("Sender").
		Order("created_at DESC").
		Limit(topK).
		Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}

	docs := make([]*schema.Document, len(messages))
	for i, msg := range messages {
		senderName := ""
		if msg.Sender.ID != 0 {
			senderName = msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}
		}

		docs[i] = &schema.Document{
			ID:      fmt.Sprintf("msg_%d", msg.ID),
			Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content),
			MetaData: map[string]interface{}{
				"message_id":      msg.ID,
				"sender_id":       msg.SenderID,
				"sender_name":     senderName,
				"conversation_id": msg.ConversationID,
				"created_at":      msg.CreatedAt,
				"type":            "message",
			},
		}
	}

	return docs, nil
}
