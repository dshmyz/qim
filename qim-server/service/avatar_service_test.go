package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
)

func TestAvatarServiceSetAIServiceRebuildsReplyGraph(t *testing.T) {
	db := setupServiceTestDB(t)
	initialAI := ai.NewAIService(&ai.AIConfig{})
	updatedAI := ai.NewAIService(&ai.AIConfig{})
	avatarService := NewAvatarService(db, initialAI)

	avatarService.SetAIService(updatedAI)

	assert.Same(t, updatedAI, avatarService.aiService)
	assert.Same(t, updatedAI, avatarService.replyGraph.aiService)
}
