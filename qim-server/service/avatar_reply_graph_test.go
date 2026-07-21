package service

import (
	"context"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvatarReplyGraphPrepareOutOfScopeSkip(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))

	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{"knowledgeDocs":true}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	// noteSvc/groupDocSvc/memorySvc 全 nil → 三处知识皆空，模拟"命中知识范围外"
	g := &AvatarReplyGraph{db: db}

	// case1: 配了知识来源但无命中 + out-of-scope=false → 跳过 LLM
	in := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in))
	assert.True(t, in.SkipReply, "知识范围外且配置为不回复应跳过 LLM")

	// case2: out-of-scope=true → 不跳过
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":true}`).Error)
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in2))
	assert.False(t, in2.SkipReply, "ReplyOutOfScope=true 时应回复")

	// case3: 未配知识来源（纯人设分身）→ 不跳过，保持原有行为
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("knowledge_scope_json", `{}`).Error)
	in3 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in3))
	assert.False(t, in3.SkipReply, "未配置知识来源的纯人设分身应正常回复")
}
