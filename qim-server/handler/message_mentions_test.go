package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
)

func TestBuildMessageResponse_ParsesMentionTokenForCurrentUser(t *testing.T) {
	message := model.Message{SenderID: 1, Content: "@{mention:2|Member} 请看"}

	response := buildMessageResponse(message, 2, []uint{1, 2})

	assert.Equal(t, []uint{2}, response["mention_user_ids"])
	assert.True(t, response["is_at_mention"].(bool))
}

func TestBuildMessageResponse_ReportsBotSenderType(t *testing.T) {
	message := model.Message{
		SenderID: 1,
		Content:  "AI reply",
		Sender: model.User{
			ID:       1,
			Nickname: "Helper",
			Type:     "bot",
		},
	}

	response := buildMessageResponse(message, 2, []uint{1, 2})

	assert.Equal(t, "bot", response["sender_type"])
	assert.True(t, response["is_ai_message"].(bool))
}

// TestBuildMessageResponse_CarriesExtraForRecallEdit 回归：撤回后重新编辑依赖 Extra 中的
// original_content 回填输入框。历史拉取走 buildMessageResponse，若丢失 Extra，切窗口/重启后
// 重新拉取的消息将无法回填（曾出现「撤回编辑 → 切窗口 → 回来不能回填」）。
func TestBuildMessageResponse_CarriesExtraForRecallEdit(t *testing.T) {
	message := model.Message{
		ID:       261,
		SenderID: 4,
		Content:  "[消息已撤回]",
		Extra:    `{"original_content":"明天开会记得带上方案"}`,
	}

	response := buildMessageResponse(message, 4, []uint{4})

	assert.Equal(t, `{"original_content":"明天开会记得带上方案"}`, response["extra"])

	// 无 Extra 时恒为 ""，与 WS 侧原始模型 json tag 行为一致，前端可稳定访问该字段
	response2 := buildMessageResponse(model.Message{ID: 300, SenderID: 4, Content: "hi"}, 4, []uint{4})
	assert.Equal(t, "", response2["extra"])
}
