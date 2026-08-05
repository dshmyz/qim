package service

import (
	"context"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/utils"
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

	// case1: 无知识/记忆命中 + out-of-scope=false，且 AI 不可用 → fail-closed 静默
	in := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in, nil))
	assert.True(t, in.SkipReply, "知识范围外且 AI 无法判断针对性时应静默")

	// case2: out-of-scope=true → 不跳过
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":true}`).Error)
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in2, nil))
	assert.False(t, in2.SkipReply, "ReplyOutOfScope=true 时应回复")

	// case3: 未配知识来源（纯人设分身），out-of-scope=false，AI 不可用 → fail-closed 静默。
	// 若配置了可用的 AI，则由 needReplyForOutOfScope 判断消息是否有针对性（有针对性才放行）。
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("knowledge_scope_json", `{}`).Error)
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":false}`).Error)
	in3 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in3, nil))
	assert.True(t, in3.SkipReply, "无知识来源且 AI 不可用时范围外静默")

	// case4: 仅开 Tasks（无 docs/notes/knowledge 命中）+ out-of-scope=false → 滑入范围外判定。
	// AI 不可用 → fail-closed 静默。
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("knowledge_scope_json", `{"tasks":true}`).Error)
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":false}`).Error)
	in4 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in4, nil))
	assert.True(t, in4.SkipReply, "无知识/记忆命中且 AI 不可用时，Tasks 不构成范围内依据，范围外静默")
}

func TestAvatarReplyGraphPrepareTaskContext(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}, &model.Task{}))

	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Task{UserID: 1, Title: "准备周报", Priority: "high", Status: "todo"}).Error)
	require.NoError(t, db.Create(&model.Task{UserID: 1, Title: "已完成项", Status: "done"}).Error) // 应被排除
	require.NoError(t, db.Create(&model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		KnowledgeScopeJSON: `{"tasks":true}`,
		// replyOutOfScope=true：本测试只验证任务上下文注入，不验证范围外静默门控
		// （门控与任务的关系由 TestAvatarReplyGraphTaskDoesNotBypassOutOfScopeGate 单独覆盖）
		ReplyStrategyJSON: `{"replyOutOfScope":true}`,
	}).Error)

	g := &AvatarReplyGraph{db: db}
	in := &AvatarReplyContext{UserID: 1, ConversationID: 0, Message: "我这周有啥事"}
	require.NoError(t, g.prepare(context.Background(), in, nil))

	assert.Contains(t, in.TaskContext, "准备周报", "Tasks 开启时应把未完成任务注入 prompt")
	assert.NotContains(t, in.TaskContext, "已完成项", "done 状态的任务不应进入上下文")
	assert.False(t, in.SkipReply, "replyOutOfScope=true 时不应被范围外静默门控跳过")
}

// TestAvatarReplyGraphTaskDoesNotBypassOutOfScopeGate 验证任务只作附加知识注入、不构成
// 「范围内」依据：即便存在任务，范围外静默门控依然生效（避免"有任务就什么都回"的乱回复）。
func TestAvatarReplyGraphTaskDoesNotBypassOutOfScopeGate(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}, &model.Task{}))

	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Task{UserID: 1, Title: "准备周报", Priority: "high", Status: "todo"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		KnowledgeScopeJSON: `{"tasks":true}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	// 有任务注入，但无笔记/群知识/记忆命中 + AI 不可用 → 仍应范围外静默（Tasks 不旁路门控）
	g := &AvatarReplyGraph{db: db}
	in := &AvatarReplyContext{UserID: 1, ConversationID: 0, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in, nil))
	assert.NotEmpty(t, in.TaskContext, "任务应注入上下文")
	assert.True(t, in.SkipReply, "存在任务不应让范围外静默失效（避免乱回复）")
}

func TestAvatarReplyGraphResolveCustomModel(t *testing.T) {
	utils.InitEncryptionKey()
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}, &model.AIConfig{}))

	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)

	encKey, err := utils.EncryptAPIKey("sk-test-123")
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.AIConfig{
		ID:              10,
		UserID:          1,
		ConfigName:      "我的DeepSeek",
		Provider:        "deepseek",
		ModelName:       "deepseek-chat",
		BaseURL:         "https://api.deepseek.com/v1",
		APIKeyEncrypted: encKey,
	}).Error)

	// aiConfigSvc 仅用于持有 db 引用；resolveCustomProvider 直接查 db，
	// 以 nil aiConfigSvc 独立单测（不含 decrypt 之外的服务逻辑）
	g := &AvatarReplyGraph{db: db}
	g2 := &AvatarReplyGraph{db: db, aiConfigSvc: NewAIConfigService(db, ai.NewProviderFactory())}

	// case1: 默认 UseSystemConfig=true → 不解析自选模型
	in1 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: true, ModelConfigID: nil}}
	assert.Nil(t, in1.CustomProvider, "UseSystemConfig 时应走系统配置")

	// case2: 关闭 UseSystemConfig + 有效的 ModelConfigID → 解析出自选 provider
	in2 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	require.NoError(t, g2.prepare(context.Background(), in2, &in2.Config))
	require.NotNil(t, in2.CustomProvider, "应命中自选模型")
	assert.Equal(t, "deepseek", in2.CustomProvider.ProviderName)
	assert.Equal(t, "deepseek-chat", in2.CustomProvider.Config.Model)
	assert.Equal(t, "sk-test-123", in2.CustomProvider.Config.APIKey, "APIKey 应被正确解密")

	// case3: 关闭 UseSystemConfig 但 ModelConfigID 指向不存在的配置 → 回退系统默认（nil）
	in3 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(999)}}
	require.NoError(t, g2.prepare(context.Background(), in3, &in3.Config))
	assert.Nil(t, in3.CustomProvider, "配置不存在时应静默回退系统默认")

	// case4: 未注入 aiConfigSvc（如测试态构造）→ 回退系统默认
	in4 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	g.aiConfigSvc = nil
	require.NoError(t, g.prepare(context.Background(), in4, &in4.Config))
	assert.Nil(t, in4.CustomProvider, "aiConfigSvc 未注入时不应 panic，应回退系统默认")

	// case5: 自选配置被禁用（AIEnabled=false）→ 回退系统默认，不应用被禁的模型
	require.NoError(t, db.Model(&model.AIConfig{}).Where("id = ?", 10).Update("ai_enabled", false).Error)
	in5 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	require.NoError(t, g2.prepare(context.Background(), in5, &in5.Config))
	assert.Nil(t, in5.CustomProvider, "被禁用的自选配置不应用于回复，应回退系统默认")
	require.NoError(t, db.Model(&model.AIConfig{}).Where("id = ?", 10).Update("ai_enabled", true).Error)

	// case6: 用户保存的生成参数（MaxTokens/Temperature）应透传到临时 provider
	in6 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	require.NoError(t, g2.prepare(context.Background(), in6, &in6.Config))
	require.NotNil(t, in6.CustomProvider, "应命中自选模型")
	assert.Equal(t, 1000, in6.CustomProvider.Config.ExtraParams["max_tokens"], "应透传已保存的 MaxTokens")
	assert.Equal(t, float64(0.7), in6.CustomProvider.Config.ExtraParams["temperature"], "应透传已保存的 Temperature")
}

func uptr(v uint) *uint { return &v }

// TestAvatarReplyGraphNeedReplyForOutOfScope 覆盖「乱回复」的另一道防线：知识范围外
// 且配置为不回复时，用 LLM 判断消息是否有针对性。有针对性 → 放行；无针对性 → 静默；
// LLM 返回非法内容 → fail-closed 静默。
func TestAvatarReplyGraphNeedReplyForOutOfScope(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)

	// 无知识命中 + ReplyOutOfScope=false → 进入 needReplyForOutOfScope。
	// 配置在所有分用例间一致（判定结果只取决于 fake LLM 的返回）。
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		KnowledgeScopeJSON: `{}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	cases := []struct {
		name     string
		reply    string // fake LLM 对针对性判断的返回
		wantSkip bool
	}{
		{"有针对性放行", `{"should_reply":true}`, false},
		{"无针对性静默", `{"should_reply":false}`, true},
		{"LLM返回非法JSON静默", `无法回复`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := &AvatarReplyGraph{db: db, aiService: newFakeAvatarAIService(tc.reply)}
			in := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
			require.NoError(t, g.prepare(context.Background(), in, &cfg))
			assert.Equal(t, tc.wantSkip, in.SkipReply, "case=%s: SkipReply 判定不符", tc.name)
		})
	}
}
