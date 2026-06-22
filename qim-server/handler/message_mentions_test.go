package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
)

func TestBuildMessageResponse_ParsesMentionTokenForCurrentUser(t *testing.T) {
	message := model.Message{SenderID: 1, Content: "@{mention:2|Member} 请看"}

	response := buildMessageResponse(message, 2)

	assert.Equal(t, []uint{2}, response["mention_user_ids"])
	assert.True(t, response["is_at_mention"].(bool))
}
