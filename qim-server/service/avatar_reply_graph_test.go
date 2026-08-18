package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/gracedb/pkg/gracedb"
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

	// case1: 无知识/记忆命中 + out-of-scope=false → 硬静默（不再依赖 AI，一律 SkipReply）
	in := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in, nil))
	assert.True(t, in.SkipReply, "知识范围外且无知识命中时应硬静默")

	// case2: out-of-scope=true → 不跳过
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":true}`).Error)
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in2, nil))
	assert.False(t, in2.SkipReply, "ReplyOutOfScope=true 时应回复")

	// case3: 未配知识来源（纯人设分身），out-of-scope=false → 硬静默（不依赖 AI）。
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("knowledge_scope_json", `{}`).Error)
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":false}`).Error)
	in3 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in3, nil))
	assert.True(t, in3.SkipReply, "无知识来源时范围外硬静默")

	// case4: 仅开 Tasks（无 docs/notes/knowledge 命中）+ out-of-scope=false → 范围外硬静默。
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("knowledge_scope_json", `{"tasks":true}`).Error)
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":false}`).Error)
	in4 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "在吗"}
	require.NoError(t, g.prepare(context.Background(), in4, nil))
	assert.True(t, in4.SkipReply, "无知识/记忆命中时，Tasks 不构成范围内依据，范围外硬静默")
}

// TestAvatarReplyGraphPrepare_MemoryInjection
// 记忆只按召回门槛（默认 0.5）过滤注入：低分噪音记忆不进 prompt（避免"被记忆干扰"），
// 高分记忆注入作辅助上下文；但记忆不再单独放行范围外——笔记/群知识为空时一律静默。
func TestAvatarReplyGraphPrepare_MemoryInjection(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{"knowledgeDocs":false,"notes":false}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	// 用 in-memory gracedb 创建分身记忆服务，写入一条记忆
	gdb, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	require.NoError(t, err)
	defer gdb.Close()
	vecSvc := &VectorService{db: gdb}
	memSvc := NewAvatarMemoryService(vecSvc, nil)
	require.NoError(t, memSvc.Remember(1, 99, "项目截止日期是3月15日，目前进度正常", 4))

	g := &AvatarReplyGraph{db: db, memorySvc: memSvc}

	// case1: 无关问题 → Recall 低分（<0.5）→ 不注入 + 静默
	in1 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "hi"}
	require.NoError(t, g.prepare(context.Background(), in1, nil))
	assert.Empty(t, in1.MemoryContext, "低分记忆（<0.5）不应注入 prompt")
	assert.True(t, in1.SkipReply, "低分记忆被滤掉后无知识命中，范围外静默")

	// case2: 相同内容查询 → Recall 高分 → 注入作辅助上下文，但不再放行范围外
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "项目截止日期是3月15日"}
	require.NoError(t, g.prepare(context.Background(), in2, nil))
	assert.NotEmpty(t, in2.MemoryContext, "相关问题应 Recall 到高分记忆并注入")
	assert.True(t, in2.SkipReply, "记忆命中不再放行范围外——笔记/群知识为空时应静默")
}

// TestAvatarReplyGraphPrepare_MemoryToggle 验证分身记忆开关：
// Memory=false 时跳过召回（高分也不注入）；nil/未设置时按默认启用（高分注入）。
// 无论开关如何，记忆都不再单独放行范围外（笔记/群知识为空时一律 SkipReply）。
func TestAvatarReplyGraphPrepare_MemoryToggle(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{"memory":false}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	gdb, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	require.NoError(t, err)
	defer gdb.Close()
	vecSvc := &VectorService{db: gdb}
	memSvc := NewAvatarMemoryService(vecSvc, nil)
	require.NoError(t, memSvc.Remember(1, 99, "项目截止日期是3月15日，目前进度正常", 4))

	g := &AvatarReplyGraph{db: db, memorySvc: memSvc}

	// case1: Memory=false → 高分查询也不注入记忆
	in1 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "项目截止日期是3月15日"}
	require.NoError(t, g.prepare(context.Background(), in1, &cfg))
	assert.Empty(t, in1.MemoryContext, "Memory=false 时应跳过记忆召回，高分也不注入")
	assert.True(t, in1.SkipReply, "记忆关闭且无笔记/群知识，范围外静默")

	// case2: Memory 未设置（nil）→ 默认启用，高分记忆注入（但记忆不再单独放行）
	cfg2 := model.AvatarConfig{UserID: 1, Enabled: true, Name: "分身",
		KnowledgeScopeJSON: `{}`, ReplyStrategyJSON: `{"replyOutOfScope":false}`}
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "项目截止日期是3月15日"}
	require.NoError(t, g.prepare(context.Background(), in2, &cfg2))
	assert.NotEmpty(t, in2.MemoryContext, "Memory 未设置时应默认启用并注入高分记忆")
	assert.True(t, in2.SkipReply, "记忆命中不再放行范围外——无笔记/群知识时应静默")
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

	// resolveCustomProvider 仅依赖 db（自选模型解析已收敛到共享 resolveUserAIConfigProvider，
	// 不再依赖注入的 AIConfigService）。用裸 graph（不注入任何服务）即可独立验证解析。
	g := &AvatarReplyGraph{db: db}

	// case1: 默认 UseSystemConfig=true → 不解析自选模型
	in1 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: true, ModelConfigID: nil}}
	assert.Nil(t, in1.CustomProvider, "UseSystemConfig 时应走系统配置")

	// case2: 关闭 UseSystemConfig + 有效的 ModelConfigID → 解析出自选 provider
	in2 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	require.NoError(t, g.prepare(context.Background(), in2, &in2.Config))
	require.NotNil(t, in2.CustomProvider, "应命中自选模型")
	assert.Equal(t, "deepseek", in2.CustomProvider.ProviderName)
	assert.Equal(t, "deepseek-chat", in2.CustomProvider.Config.Model)
	assert.Equal(t, "sk-test-123", in2.CustomProvider.Config.APIKey, "APIKey 应被正确解密")

	// case3: 关闭 UseSystemConfig 但 ModelConfigID 指向不存在的配置 → 回退系统默认（nil）
	in3 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(999)}}
	require.NoError(t, g.prepare(context.Background(), in3, &in3.Config))
	assert.Nil(t, in3.CustomProvider, "配置不存在时应静默回退系统默认")

	// case4: 裸 graph 仅靠 db 即可解析（曾错误依赖 AIConfigService，去除该死依赖后仍应命中）
	in4 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	require.NoError(t, g.prepare(context.Background(), in4, &in4.Config))
	require.NotNil(t, in4.CustomProvider, "仅靠 db 即可解析，不应再依赖注入服务")

	// case5: 自选配置被禁用（AIEnabled=false）→ 回退系统默认，不应用被禁的模型
	require.NoError(t, db.Model(&model.AIConfig{}).Where("id = ?", 10).Update("ai_enabled", false).Error)
	in5 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	require.NoError(t, g.prepare(context.Background(), in5, &in5.Config))
	assert.Nil(t, in5.CustomProvider, "被禁用的自选配置不应用于回复，应回退系统默认")
	require.NoError(t, db.Model(&model.AIConfig{}).Where("id = ?", 10).Update("ai_enabled", true).Error)

	// case6: 用户保存的生成参数（MaxTokens/Temperature）应透传到临时 provider
	in6 := &AvatarReplyContext{UserID: 1, Message: "hi", Config: model.AvatarConfig{UserID: 1, UseSystemConfig: false, ModelConfigID: uptr(10)}}
	require.NoError(t, g.prepare(context.Background(), in6, &in6.Config))
	require.NotNil(t, in6.CustomProvider, "应命中自选模型")
	assert.Equal(t, 1000, in6.CustomProvider.Config.ExtraParams["max_tokens"], "应透传已保存的 MaxTokens")
	assert.Equal(t, float64(0.7), in6.CustomProvider.Config.ExtraParams["temperature"], "应透传已保存的 Temperature")
}

func uptr(v uint) *uint { return &v }

// TestAvatarReplyGraphNeedReplyForOutOfScope 覆盖「乱回复」的关键防线：知识范围外
// 且配置为不回复时，直接硬静默（原行为会用 LLM 二次判断"是否有针对性"并可能放行闲聊，
// 导致用户以为设置没生效）。改为硬门控后，无任何知识命中即 SkipReply，不依赖 LLM 返回。
func TestAvatarReplyGraphNeedReplyForOutOfScope(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)

	// 无知识命中 + ReplyOutOfScope=false → 硬静默。LLM 返回何种内容都不再影响判定。
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		KnowledgeScopeJSON: `{}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	cases := []struct {
		name     string
		reply    string // fake LLM 的返回（硬门控下不再被消费）
		wantSkip bool
	}{
		// 即便 LLM 说"有针对性"，无知识命中仍硬静默（不再放行闲聊/问候）
		{"无知识命中一律硬静默", `{"should_reply":true}`, true},
		{"无知识命中一律硬静默(LLM 说无需)", `{"should_reply":false}`, true},
		{"LLM 异常也静默", `无法回复`, true},
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

// TestAvatarReplyGraphPrepare_KnowledgeFloor 验证分身路径接入知识硬下限：
// ai.knowledge_score_threshold（默认 0.3）过滤低分笔记命中——低分命中不注入、
// 不进"依据"徽章，也不构成"范围内"放行（配合范围外静默 → SkipReply）。
func TestAvatarReplyGraphPrepare_KnowledgeFloor(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}, &model.SystemConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{"notes":true}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	// 真实笔记向量服务：gracedb + fakeEmbedder + 伪嵌入 provider（与 newFakeNoteService 同构）
	gdb, err := gracedb.Open(t.TempDir()+"/vec", gracedb.WithEmbedder(fakeEmbedder{}))
	require.NoError(t, err)
	defer gdb.Close()
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-embed", embedFakeProvider{})
	noteVecSvc := &NoteVectorService{vectorSvc: &VectorService{db: gdb}, aiService: aiSvc}
	require.NoError(t, noteVecSvc.VectorizeNote(1, 1, "项目文档", "项目截止日期是3月15日，目前进度正常"))
	got, err := noteVecSvc.SearchNotes(1, "项目截止日期是3月15日", 3)
	require.NoError(t, err)
	require.NotEmpty(t, got, "笔记应可召回")

	// case1: 默认下限 0.3 → 命中进上下文，构成范围内 → 不静默
	g1 := &AvatarReplyGraph{db: db, noteSvc: noteVecSvc}
	in1 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "项目截止日期是3月15日"}
	require.NoError(t, g1.prepare(context.Background(), in1, &cfg))
	assert.NotEmpty(t, in1.NoteContext, "过下限的笔记应注入")
	assert.False(t, in1.SkipReply, "笔记命中应构成范围内，不静默")

	// case2: 硬下限拉到超过任何余弦分 → 同一条笔记被滤掉 → 不注入不进徽章且范围外静默
	// （直接种 SystemConfig 行绕过 BatchUpdate 的范围校验，值取 1.5 保证高于任意余弦相似度）
	require.NoError(t, db.Create(&model.SystemConfig{ConfigKey: "ai.knowledge_score_threshold", Value: "1.5", Type: "number"}).Error)
	g2 := &AvatarReplyGraph{db: db, noteSvc: noteVecSvc, thresholdSvc: NewAiThresholdService(db)}
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "项目截止日期是3月15日"}
	require.NoError(t, g2.prepare(context.Background(), in2, &cfg))
	assert.Empty(t, in2.NoteContext, "低于硬下限的笔记不应注入")
	assert.Empty(t, in2.Sources, "低于硬下限的命中不进“依据”徽章")
	assert.True(t, in2.SkipReply, "笔记被硬下限滤掉后无知识命中，范围外静默")
}

// TestAvatarReplyGraph_RenderScopePolicy 验证范围策略与 replyOutOfScope/BypassScope 联动：
// 严格（范围外静默）→ 渲染"严格基于上方资料"；宽松（范围外也回/草稿·图片路径）→ 保留"给出你的理解"。
func TestAvatarReplyGraph_RenderScopePolicy(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	g := NewAvatarReplyGraph(newFakeAvatarAIService("x"), db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	renderAll := func(in *AvatarReplyContext) string {
		t.Helper()
		msgs, err := g.renderPrompt(context.Background(), in)
		require.NoError(t, err)
		var sb strings.Builder
		for _, m := range msgs {
			sb.WriteString(m.Content)
		}
		return sb.String()
	}

	// strict：ReplyOutOfScope=false 且非宽松路径 → 严格策略，不含"给出你的理解"
	in := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "你好"}
	require.NoError(t, g.prepare(context.Background(), in, &cfg))
	joined := renderAll(in)
	assert.Contains(t, joined, "严格基于上方", "范围外静默时应渲染严格策略")
	assert.NotContains(t, joined, "给出你的理解", "严格策略不应允许自由发挥")
	assert.Contains(t, joined, "没有相关资料", "严格策略应指导资料不足时明说")

	// relaxed：ReplyOutOfScope=true → 宽松策略
	require.NoError(t, db.Model(&model.AvatarConfig{}).Where("user_id = ?", 1).
		Update("reply_strategy_json", `{"replyOutOfScope":true}`).Error)
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "你好"}
	require.NoError(t, g.prepare(context.Background(), in2, nil))
	joined2 := renderAll(in2)
	assert.Contains(t, joined2, "给出你的理解", "范围外也回时应渲染宽松策略")
	assert.NotContains(t, joined2, "严格基于上方", "宽松策略不应渲染严格文本")

	// bypass：ReplyOutOfScope=false 但 BypassScope=true（草稿/图片路径）→ 宽松
	in3 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "你好", BypassScope: true}
	require.NoError(t, g.prepare(context.Background(), in3, &cfg))
	joined3 := renderAll(in3)
	assert.Contains(t, joined3, "给出你的理解", "草稿/图片路径应渲染宽松策略")
}

func TestBuildCustomProviderExtraParams(t *testing.T) {
	// 零值透传语义（修复 #3）：
	// - max_tokens=0 在某些 provider 会被 API 拒绝或解释为无限制，且无确定性语义 → 跳过
	// - temperature=0 是用户显式设置的确定性输出（AIConfig.Temperature DB 默认 0.7，读到 0 必为用户故意），
	//   必须透传，不能被跳过成 provider 默认 0.7
	p := buildCustomProviderExtraParams(0, 0)
	_, hasMax := p["max_tokens"]
	assert.False(t, hasMax, "max_tokens=0 不应透传")
	assert.Equal(t, float64(0), p["temperature"], "temperature=0 应透传（确定性输出）")

	// 正常值透传
	p = buildCustomProviderExtraParams(1000, 0.7)
	assert.Equal(t, 1000, p["max_tokens"])
	assert.Equal(t, 0.7, p["temperature"])

	// 仅 max_tokens 有效
	p = buildCustomProviderExtraParams(2000, 0)
	assert.Equal(t, 2000, p["max_tokens"])
	assert.Equal(t, float64(0), p["temperature"], "temperature=0 应透传")

	// 仅 temperature 有效
	p = buildCustomProviderExtraParams(0, 0.5)
	_, hasMax = p["max_tokens"]
	assert.False(t, hasMax, "max_tokens=0 不应透传")
	assert.Equal(t, 0.5, p["temperature"])
}

// TestAvatarReplyGraph_MemoryRecallThreshold
// 分身记忆召回相关度门槛默认 0.5；注入阈值服务后经 GetFloat 读取（nil DB 回退默认值 0.5）。
func TestAvatarReplyGraph_MemoryRecallThreshold(t *testing.T) {
	g := &AvatarReplyGraph{}
	if got := g.memoryRecallThreshold(); got != 0.5 {
		t.Errorf("默认记忆召回门槛应为 0.5，got %v", got)
	}

	g.SetThresholdService(NewAiThresholdService(nil))
	if got := g.memoryRecallThreshold(); got != 0.5 {
		t.Errorf("注入阈值服务后应读回默认 0.5，got %v", got)
	}
}

// capturingAvatarProvider 测试用 Provider 桩：在复用 fakeAvatarProvider 的默认回复基础上，
// 捕获最近一次 Chat 收到的 ai.Message 列表，供断言多模态 ImageURL 是否透传给模型。
type capturingAvatarProvider struct {
	fakeAvatarProvider
	lastMessages []ai.Message
}

var _ ai.Provider = (*capturingAvatarProvider)(nil)

func (c *capturingAvatarProvider) Chat(messages []ai.Message) (string, error) {
	c.lastMessages = messages
	return c.reply, nil
}

// streamingAvatarProvider 测试用流式 Provider 桩：在复用 capturingAvatarProvider 的
// 整段回复与消息捕获基础上，ChatStreamWithContext 把固定回复逐字拆块发出
// （EinoChatModel.Stream 走的是 ChatStreamWithContext），供流式草稿路径
// （ExecuteStreamWithImageSources）断言流式拼装结果与图片 data URL 透传。
type streamingAvatarProvider struct {
	capturingAvatarProvider
}

var _ ai.Provider = (*streamingAvatarProvider)(nil)

func (c *streamingAvatarProvider) ChatStreamWithContext(ctx context.Context, messages []ai.Message, onChunk func(chunk ai.StreamChunk) error) error {
	c.lastMessages = messages
	for _, r := range []rune(c.reply) {
		if err := onChunk(ai.StreamChunk{Content: string(r)}); err != nil {
			return err
		}
	}
	return nil
}

// TestAvatarReplyGraph_ExecuteWithImageSources 验证分身图片触发消息走多模态生成：
// 1) 生成的回复能经 fake provider 返回；2) 透传给模型的最后一条 user 消息携带图片 data URL。
// 该路径是分身对图片的「自动回复」（worker 触发，非用户主动）：即便 ReplyOutOfScope=false
// 也能看图回（图片路径不因范围外静默而跳过），但 system prompt 必须走严格范围策略，
// 不得因 BypassScope 放宽到"资料不足可自由发挥"。
func TestAvatarReplyGraph_ExecuteWithImageSources(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{}`, // 无知识来源，验证图片路径不被范围外静默挡住
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	capProv := &capturingAvatarProvider{}
	capProv.reply = "识别到图片：一只猫" // 继承 fakeAvatarProvider.reply 字段
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-avatar", capProv)

	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	const dataURL = "data:image/png;base64,cccc"
	// 自动图片回复：bypassScope=false，范围外策略照常生效（严格）
	reply, _, err := g.ExecuteWithImageSources(context.Background(), 1, 1, `{"id":1,"url":"/files/x.png"}`, dataURL, "cat.png", &cfg, false)
	require.NoError(t, err)
	assert.Equal(t, "识别到图片：一只猫", reply, "分身应返回多模态图片识别的回复")

	// 取透传给模型的最后一条 user 消息，断言携带图片 data URL
	var lastUser ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			lastUser = m
		}
	}
	assert.Equal(t, dataURL, lastUser.ImageURL, "分身图片路径应把 base64 data URL 作为 ImageURL 交给模型")
	assert.Contains(t, lastUser.Content, "cat.png", "分身图片路径应提示模型识别该图片")

	// 自动图片回复不置 BypassScope：system prompt 必须是严格范围策略
	var system ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "system" {
			system = m
		}
	}
	require.NotEmpty(t, system.Content, "应存在 system prompt")
	assert.Contains(t, system.Content, "回答必须严格基于上方提供的资料", "自动图片回复应走严格范围策略")
	assert.NotContains(t, system.Content, "请明确说明资料不足并给出你的理解", "自动图片回复不应被 BypassScope 放宽")
}

// TestAvatarReplyGraph_ExecuteBatchWithImagesSources_AutoStrictScope 分身对「合并窗口内连发
// 的一批消息」（含图片）的自动批量回复同样不置 BypassScope：ReplyOutOfScope=false 时
// system prompt 走严格范围策略，与用户主动触发的草稿路径（宽松）区分开。
func TestAvatarReplyGraph_ExecuteBatchWithImagesSources_AutoStrictScope(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	capProv := &capturingAvatarProvider{}
	capProv.reply = "识别到一批消息"
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-avatar", capProv)

	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	const dataURL = "data:image/png;base64,cccc"
	reply, _, err := g.ExecuteBatchWithImagesSources(context.Background(), 1, 1,
		[]string{"1.在吗"}, []string{dataURL}, []string{"a.png"}, &cfg, false)
	require.NoError(t, err)
	assert.Equal(t, "识别到一批消息", reply, "分身应返回批量消息的合并回复")

	var system ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "system" {
			system = m
		}
	}
	require.NotEmpty(t, system.Content, "应存在 system prompt")
	assert.Contains(t, system.Content, "回答必须严格基于上方提供的资料", "自动批量图片回复应走严格范围策略")
	assert.NotContains(t, system.Content, "请明确说明资料不足并给出你的理解", "自动批量图片回复不应被 BypassScope 放宽")
}

// TestExecuteWithImageSources_FailNoSilentFalse 图片生成失败时（模型不支持视觉/调用报错）应返回错误，
// 由调用方（worker）按「尽力而为失败则跳过」降级，而非产出假回复。
func TestExecuteWithImageSources_ModelError(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{UserID: 1, Enabled: true, Name: "分身"}
	require.NoError(t, db.Create(&cfg).Error)

	aiSvc := ai.NewAIService(&ai.AIConfig{})
	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	_, _, err := g.ExecuteWithImageSources(context.Background(), 1, 1, `{"id":1}`, "data:image/png;base64,dddd", "x.png", &cfg, false)
	// aiSvc 未配置 provider → GetCompletion 返回未配置错误，图片路径应如实上抛，由 worker 跳过
	assert.Error(t, err, "模型不可用时应返回错误而非静默假回复")
}

// TestAvatarReplyGraph_ExecuteStreamWithImageSources 验证分身「帮我回复」草稿模式基于图片流式生成：
// 1) 流式逐块拼出的草稿与整段回复一致；2) 透传给模型的最后一条 user 消息携带图片 data URL 与文件名提示。
// 草稿模式忽略知识范围外静默（用户主动要草稿），此处用 out-of-scope 配置验证仍回复。
func TestAvatarReplyGraph_ExecuteStreamWithImageSources(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{}`, // 无知识来源，验证图片草稿路径不被范围外静默挡住
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	capProv := &streamingAvatarProvider{}
	capProv.reply = "识别到图片：一只猫"
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-avatar", capProv)

	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	const dataURL = "data:image/png;base64,cccc"
	stream, err := g.ExecuteStreamWithImageSources(context.Background(), 1, 1,
		"对方发来了一张图片，请起草一条回复。", dataURL, "cat.png", nil, &cfg)
	require.NoError(t, err)
	defer stream.Close()

	// 消费流式 reader，逐块拼出完整草稿（与 handler streamCompletionFromReader 同构）
	var sb strings.Builder
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			t.Fatalf("流式接收失败: %v", recvErr)
		}
		if msg != nil {
			sb.WriteString(msg.Content)
		}
	}
	assert.Equal(t, "识别到图片：一只猫", sb.String(), "流式草稿应逐块拼出与整段一致的回复")

	// 取透传给模型的最后一条 user 消息，断言携带图片 data URL
	var lastUser ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			lastUser = m
		}
	}
	assert.Equal(t, dataURL, lastUser.ImageURL, "分身流式图片路径应把 base64 data URL 作为 ImageURL 交给模型")
	assert.Contains(t, lastUser.Content, "cat.png", "分身流式图片路径应提示模型识别该图片")
}

// TestAvatarReplyGraph_ExecuteStream_HistoryAnchored 验证草稿模式的历史锚定：
// 右键历史消息（非会话最新一条）派生草稿时，对话历史只取目标消息「之前」的内容，
// 不含目标之后的消息——否则模型会在"对方说：旧消息"与历史里的后续对话中左右矛盾、答非所问。
func TestAvatarReplyGraph_ExecuteStream_HistoryAnchored(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "qim", Nickname: "QIM", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "bob", Nickname: "Bob", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{}`, // 无知识来源，草稿模式忽略范围外静默（用户主动要草稿）
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	// 会话时间线（最后一条是目标之后的后续对话，锚定后不应进入历史）：
	//   Bob: 早上好 → QIM: 早，资料发你 → Bob: 看下这份报价【目标】→ QIM: 收到，明天回复你
	base := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	msgs := []model.Message{
		{ConversationID: 1, SenderID: 2, Type: "text", Content: "早上好", CreatedAt: base.Add(0 * time.Minute)},
		{ConversationID: 1, SenderID: 1, Type: "text", Content: "早，资料发你", CreatedAt: base.Add(1 * time.Minute)},
		{ConversationID: 1, SenderID: 2, Type: "text", Content: "看下这份报价", CreatedAt: base.Add(2 * time.Minute)},
		{ConversationID: 1, SenderID: 1, Type: "text", Content: "收到，明天回复你", CreatedAt: base.Add(3 * time.Minute)},
	}
	for i := range msgs {
		require.NoError(t, db.Create(&msgs[i]).Error)
	}
	target := msgs[2] // 看下这份报价

	capProv := &streamingAvatarProvider{}
	capProv.reply = "好的，报价我看了明天答复你"
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-avatar", capProv)

	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	// 目标不是最新一条，把历史锚定到目标的 CreatedAt 派生草稿
	stream, err := g.ExecuteStream(context.Background(), 1, 1, target.Content, &target.CreatedAt, &cfg)
	require.NoError(t, err)
	defer stream.Close()
	for {
		_, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			t.Fatalf("流式接收失败: %v", recvErr)
		}
	}

	// 取透传给模型的 user 消息（消息块结构：知识/历史各自成块 + 最后"对方说"），
	// 拼合全部 user 消息后断言历史已锚定到目标
	var joined string
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			joined += m.Content
		}
	}
	require.NotEmpty(t, joined, "应有 user 消息携带 prompt")
	assert.Contains(t, joined, "对方说：看下这份报价", "草稿应以目标消息为回复对象")
	assert.Contains(t, joined, "Bob: 早上好", "目标之前的历史应保留")
	assert.Contains(t, joined, "QIM: 早，资料发你", "目标之前的历史应保留")
	assert.NotContains(t, joined, "收到，明天回复你", "目标之后的消息不应混入历史（否则模型答非所问）")
}

// TestAvatarReplyGraph_ExecuteBatchWithImagesSources 验证分身「合并窗口连发一批消息」走批量多模态生成：
// 1) 批内多条文本按序拼入 prompt；2) 批内多张图片全部作为 data URL 注入（ai.Message.ImageURLs）；
// 3) 生成回复能经 fake provider 返回。批内无文本（全图）时也应正常生成。
func TestAvatarReplyGraph_ExecuteBatchWithImagesSources(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{}`, // 无知识来源，验证批量路径不被范围外静默挡住
		ReplyStrategyJSON:  `{"replyOutOfScope":false}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	capProv := &capturingAvatarProvider{}
	capProv.reply = "识别到两张图：猫和狗"
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-avatar", capProv)

	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	const img1 = "data:image/png;base64,aaa"
	const img2 = "data:image/png;base64,bbb"
	reply, _, err := g.ExecuteBatchWithImagesSources(
		context.Background(), 1, 1,
		[]string{"第一句", "第二句"}, []string{img1, img2}, []string{"猫.png", "狗.png"},
		&cfg, false, // 自动批量回复：不置 BypassScope
	)
	require.NoError(t, err)
	assert.Equal(t, "识别到两张图：猫和狗", reply, "分身批量路径应返回多模态合并回复")

	// 取透传给模型的最后一条 user 消息，断言携带两张图片及全部批内文本
	var lastUser ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			lastUser = m
		}
	}
	assert.Len(t, lastUser.ImageURLs, 2, "批量路径应把多张图作为 ImageURLs 交给模型")
	assert.Equal(t, []string{img1, img2}, lastUser.ImageURLs, "多图应按序透传")
	assert.Equal(t, img1, lastUser.ImageURL, "多图时首图仍回填 ImageURL（向后兼容）")
	assert.Contains(t, lastUser.Content, "第一句", "批内第一条文本应进入 prompt")
	assert.Contains(t, lastUser.Content, "第二句", "批内第二条文本应进入 prompt")

	// 批内全为图片（无文本）：占位提示 + 图片注入，仍应正常生成
	capProv.reply = "看图回复"
	reply2, _, err2 := g.ExecuteBatchWithImagesSources(
		context.Background(), 1, 1,
		[]string{"对方发来了一张/多张图片，请识别其内容并结合对话回复。"}, []string{img1}, []string{"猫.png"},
		&cfg, false,
	)
	require.NoError(t, err2)
	assert.Equal(t, "看图回复", reply2, "批内全图也应走多模态生成")
}

// TestAvatarReplyGraph_ExecuteBatch_ModelError 批量多模态生成模型不可用时应返回错误（由 worker
// 按「尽力而为失败则跳过」整批跳过），而非静默产出假回复。
func TestAvatarReplyGraph_ExecuteBatch_ModelError(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{UserID: 1, Enabled: true, Name: "分身"}
	require.NoError(t, db.Create(&cfg).Error)

	aiSvc := ai.NewAIService(&ai.AIConfig{}) // 未配置 provider → GetCompletion 报错
	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)
	require.NoError(t, g.BuildGraph())

	_, _, err := g.ExecuteBatchWithImagesSources(
		context.Background(), 1, 1,
		[]string{"第一句"}, []string{"data:image/png;base64,aaa"}, []string{"x.png"},
		&cfg, false,
	)
	assert.Error(t, err, "批量多模态模型不可用时应返回错误")
}

// TestSelectTopByScore 宽召回后的收敛逻辑：过滤低于下限的噪音 → 分数降序 → 取前 K。
// 分身笔记/群知识注入共用，防止宽召回后把整库知识塞进 prompt。
func TestSelectTopByScore(t *testing.T) {
	snips := []KnowledgeSnippet{
		{Title: "A", Score: 0.9},
		{Title: "B", Score: 0.5},
		{Title: "C", Score: 0.2}, // 低于下限 → 过滤
		{Title: "D", Score: 0.7},
		{Title: "E", Score: 0.95},
	}
	got := selectTopByScore(snips, 0.3, 3)
	require.Len(t, got, 3, "应取前 3")
	assert.Equal(t, "E", got[0].Title, "应分数降序")
	assert.Equal(t, "A", got[1].Title)
	assert.Equal(t, "D", got[2].Title)
	assert.NotContains(t, []string{"C"}, got[0].Title+"", "低于下限的应被过滤")

	// k 大于候选数 → 全取（floor 只滤掉 0.2 的 C，剩 4 条）
	got2 := selectTopByScore(snips, 0.3, 10)
	assert.Len(t, got2, 4, "低于下限的被过滤后仅剩 4 条")

	// 全低于下限 → 空
	assert.Empty(t, selectTopByScore(snips, 0.99, 3))

	// 空输入
	assert.Empty(t, selectTopByScore(nil, 0.3, 3))
}

// TestAvatarReplyGraph_ExecuteWithSources 验证自动回复主路径（非图片）：
// 走 renderPrompt 消息块拼装 + GetCompletion（不再经 Eino 编译图）；
// 命中范围外静默时不调用模型、返回空回复。
func TestAvatarReplyGraph_ExecuteWithSources(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "qim", Nickname: "QIM", PasswordHash: "h"}).Error)
	cfg := model.AvatarConfig{
		UserID:             1,
		Enabled:            true,
		Name:               "分身",
		KnowledgeScopeJSON: `{"knowledgeDocs":true,"notes":true}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":true}`, // 范围外也回，验证消息块送达模型
	}
	require.NoError(t, db.Create(&cfg).Error)

	capProv := &capturingAvatarProvider{}
	capProv.reply = "回复内容"
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-avatar", capProv)

	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)

	// case1: 正常回复，消息块送达模型
	reply, sources, err := g.ExecuteWithSources(context.Background(), 1, 99, "你好", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "回复内容", reply)
	assert.Nil(t, sources, "无知识命中时来源为空")

	var joined string
	for _, m := range capProv.lastMessages {
		joined += m.Role + ": " + m.Content + "\n"
	}
	assert.Contains(t, joined, "system: 你是QIM的AI分身", "system 消息应拼装")
	assert.Contains(t, joined, "对方说：你好", "最后一条 user 消息应为提问")
	// 无知识/历史命中时，不应出现上下文块及其确认对
	assert.NotContains(t, joined, "收到笔记信息", "无笔记命中时不应有笔记块")
	assert.NotContains(t, joined, "已了解对话历史。", "无历史时不应有历史块确认对")

	// case2: 范围外静默 → 不调用模型
	cfg2 := model.AvatarConfig{UserID: 1, Enabled: true, Name: "分身",
		KnowledgeScopeJSON: `{}`, ReplyStrategyJSON: `{"replyOutOfScope":false}`}
	capProv.lastMessages = nil
	reply2, sources2, err2 := g.ExecuteWithSources(context.Background(), 1, 99, "在吗", &cfg2)
	require.NoError(t, err2)
	assert.Empty(t, reply2, "范围外静默应返回空回复")
	assert.Empty(t, sources2)
	assert.Nil(t, capProv.lastMessages, "范围外静默不应调用模型")
}

// TestAvatarMemoryEnabled 记忆开关判定的单测：nil（存量未设置）默认启用；
// AvatarConfig 解析 JSON 后与 scope 方法一致；损坏 JSON 按启用处理（不阻断）。
func TestAvatarMemoryEnabled(t *testing.T) {
	// scope 级
	assert.True(t, model.AvatarKnowledgeScope{}.MemoryEnabled(), "Memory nil 默认启用")
	f := false
	assert.False(t, model.AvatarKnowledgeScope{Memory: &f}.MemoryEnabled(), "显式 false 关闭")
	tr := true
	assert.True(t, model.AvatarKnowledgeScope{Memory: &tr}.MemoryEnabled(), "显式 true 启用")

	// config 级（解析 JSON）
	assert.True(t, model.AvatarConfig{}.MemoryEnabled(), "空配置默认启用")
	assert.False(t, model.AvatarConfig{KnowledgeScopeJSON: `{"memory":false}`}.MemoryEnabled())
	assert.True(t, model.AvatarConfig{KnowledgeScopeJSON: `{"memory":true}`}.MemoryEnabled())
	assert.True(t, model.AvatarConfig{KnowledgeScopeJSON: `{broken`}.MemoryEnabled(), "损坏 JSON 按启用处理")
}

// TestGetConversationHistory_ReturnsErrorOnDBFailure 历史查询失败时必须显式返回 error，
// 由调用方（prepare）降级为空历史并记日志——不能静默吞掉，否则 DB 故障时分身
// 会"假装没有历史"继续回复，多轮追问断链且无从排查。
func TestGetConversationHistory_ReturnsErrorOnDBFailure(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "qim", Nickname: "QIM", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: 7, SenderID: 1, Type: "text", Content: "第一条"}).Error)

	// 关闭底层连接使后续查询必然报错
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	g := &AvatarReplyGraph{db: db}
	_, err = g.getConversationHistory(7, 10, "在吗", nil)
	require.Error(t, err, "历史查询失败时应显式返回 error，而非静默返回空历史")
}
