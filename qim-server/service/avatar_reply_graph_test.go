package service

import (
	"context"
	"testing"

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

// TestAvatarReplyGraphPrepare_MemoryThreshold
// 分身记忆参与范围外静默判定，但需过相关度阈值（≥0.5）：
// - 无关问题 → Recall 低分噪音 → hasKnowledge=false → 静默
// - 相关问题 → Recall 高分记忆 → hasKnowledge=true → 回复
func TestAvatarReplyGraphPrepare_MemoryThreshold(t *testing.T) {
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

	// case1: 无关问题 → Recall 低分（<0.5）→ 静默
	in1 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "hi"}
	require.NoError(t, g.prepare(context.Background(), in1, nil))
	assert.NotEmpty(t, in1.MemoryContext, "无关问题也可能 Recall 到记忆（低分）")
	assert.True(t, in1.SkipReply, "低分记忆（<0.5）不应绕过范围外静默")

	// case2: 相同内容查询 → Recall 高分 → 回复
	in2 := &AvatarReplyContext{UserID: 1, ConversationID: 99, Message: "项目截止日期是3月15日"}
	require.NoError(t, g.prepare(context.Background(), in2, nil))
	assert.NotEmpty(t, in2.MemoryContext, "相关问题应 Recall 到高分记忆")
	assert.False(t, in2.SkipReply, "高分记忆应绕过范围外静默——用户在问相关问题")
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

// TestAvatarReplyGraph_ExecuteWithImageSources 验证分身图片触发消息走多模态生成：
// 1) 生成的回复能经 fake provider 返回；2) 透传给模型的最后一条 user 消息携带图片 data URL。
// 图片路径忽略知识范围外静默（能看图就回，看不了由 worker 跳过），此处用 out-of-scope 配置验证仍回复。
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
	reply, _, err := g.ExecuteWithImageSources(context.Background(), 1, 1, `{"id":1,"url":"/files/x.png"}`, dataURL, "cat.png", &cfg)
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

	_, _, err := g.ExecuteWithImageSources(context.Background(), 1, 1, `{"id":1}`, "data:image/png;base64,dddd", "x.png", &cfg)
	// aiSvc 未配置 provider → GetCompletion 返回未配置错误，图片路径应如实上抛，由 worker 跳过
	assert.Error(t, err, "模型不可用时应返回错误而非静默假回复")
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
		&cfg,
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
		&cfg,
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
		&cfg,
	)
	assert.Error(t, err, "批量多模态模型不可用时应返回错误")
}
