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

// ===== 测试覆盖缺口补齐 =====
// 1. 自选模型失败回退系统默认路径
// 2. 群助手知识节点上下文感知检索（图级流程的直接单测）
// 3. 笔记混合检索的语义腿失败 → FTS-only 降级

// TestAvatarReplyGraph_CustomProviderFallbackToSystem 自选模型配置在生成期失败
// （provider 不存在 → CreateProviderByName 直接报错）时应回退系统默认并正常回复，
// 兑现「回退系统默认…不阻断回复」契约。
func TestAvatarReplyGraph_CustomProviderFallbackToSystem(t *testing.T) {
	utils.InitEncryptionKey("")
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}, &model.AIConfig{}))
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u", PasswordHash: "h"}).Error)

	// 自选模型配置指向不存在的 provider → 生成期必然失败 → 触发系统回退
	encKey, err := utils.EncryptAPIKey("sk-test-123")
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.AIConfig{
		ID: 10, UserID: 1, ConfigName: "坏配置", Provider: "nonexistent-provider",
		ModelName: "x", BaseURL: "http://x", APIKeyEncrypted: encKey, AIEnabled: true,
	}).Error)
	cfg := model.AvatarConfig{
		UserID: 1, Enabled: true, Name: "分身",
		UseSystemConfig:    false,
		ModelConfigID:      uptr(10),
		KnowledgeScopeJSON: `{}`,
		ReplyStrategyJSON:  `{"replyOutOfScope":true}`,
	}
	require.NoError(t, db.Create(&cfg).Error)

	capProv := &capturingAvatarProvider{}
	capProv.reply = "系统模型回复"
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-avatar", capProv)
	g := NewAvatarReplyGraph(aiSvc, db, nil, nil, nil)

	reply, _, err := g.ExecuteWithSources(context.Background(), 1, 99, "你好", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "系统模型回复", reply, "自选模型创建失败时应回退系统默认并正常回复")
	require.NotNil(t, capProv.lastMessages, "回退后应调用系统 provider")
}

// chatOnlyProvider 只支持 Chat、不支持 Embedding 的 provider 桩：让 aiService.Embed 报错，
// 用于验证笔记混合检索的「语义腿失败 → 降级词法腿」路径。
type chatOnlyProvider struct {
	*fakeAvatarProvider
}

func (chatOnlyProvider) SupportsEmbedding() bool { return false }

// TestNoteSearchHybrid_FTSOnlyDegradation 笔记混合检索的降级路径：
// 语义腿失败（Embed 报错）时，FTS 词法腿仍应命中，且分数经 hybridDisplayScores
// 还原到不低于知识硬下限的展示语义。
func TestNoteSearchHybrid_FTSOnlyDegradation(t *testing.T) {
	gdb, err := gracedb.Open(t.TempDir()+"/vec", gracedb.WithEmbedder(fakeEmbedder{}))
	require.NoError(t, err)
	defer gdb.Close()
	vecSvc := &VectorService{db: gdb}

	// 播种：embedFakeProvider 正常嵌入，写入笔记
	seedAI := ai.NewAIService(&ai.AIConfig{})
	seedAI.SetProviderForTesting("fake-embed", embedFakeProvider{})
	seedSvc := &NoteVectorService{vectorSvc: vecSvc, aiService: seedAI}
	require.NoError(t, seedSvc.VectorizeNote(1, 1, "项目A进度", "项目A截止日期是3月15日，负责人张三，目前进度正常"))

	// 查询：chatOnlyProvider 无 Embedding → 语义腿失败 → 应降级到 FTS-only
	brokenAI := ai.NewAIService(&ai.AIConfig{})
	brokenAI.SetProviderForTesting("fake-chat-only", chatOnlyProvider{&fakeAvatarProvider{reply: ""}})
	querySvc := &NoteVectorService{vectorSvc: vecSvc, aiService: brokenAI}

	results, err := querySvc.SearchNotes(1, "项目A截止日期是3月15日", 3)
	require.NoError(t, err)
	require.NotEmpty(t, results, "语义腿失败时 FTS-only 应仍能命中")
	for _, r := range results {
		assert.GreaterOrEqual(t, r.Score, 0.3, "FTS-only 命中分数应还原到不低于知识硬下限")
	}
}

// TestSmartReplyKnowledgeNode_ContextualRetrieval 群助手上下文感知检索的直接单测：
// 追问消息（"那后来呢？"）本身不含话题，retrieveGroupKnowledge（createKnowledgeNode 共用）
// 把最近历史拼进 query 后应命中群知识库文档并产出知识来源。
// （compose.Lambda 无公开 Invoke，经抽取出的方法直接测同一检索代码；图级流程此前仅靠
// handler 集成测试兜底。）
func TestSmartReplyKnowledgeNode_ContextualRetrieval(t *testing.T) {
	db := setupServiceTestDB(t) // 已含 User/Conversation/Group/Message 表
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Nickname: "Alice", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "bob", Nickname: "Bob", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Conversation{ID: 7, Type: "group"}).Error)
	require.NoError(t, db.Create(&model.Group{ID: 77, ConversationID: 7, Name: "项目群"}).Error)
	// 最近历史确立话题：项目A 的讨论（追问消息本身不含话题词）
	msgs := []model.Message{
		{ConversationID: 7, SenderID: 1, Type: "text", Content: "项目A什么时候截止？"},
		{ConversationID: 7, SenderID: 2, Type: "text", Content: "3月15日"},
	}
	for i := range msgs {
		require.NoError(t, db.Create(&msgs[i]).Error)
	}

	// 群知识库：gracedb + fakeEmbedder + embedFakeProvider，写入一篇项目文档
	gdb, err := gracedb.Open(t.TempDir()+"/vec", gracedb.WithEmbedder(fakeEmbedder{}))
	require.NoError(t, err)
	defer gdb.Close()
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-embed", embedFakeProvider{})
	vecSvc := &VectorService{db: gdb}
	require.NoError(t, ensureGracedbCollection(gdb, "group_77"))
	emb, err := aiSvc.Embed("项目A截止日期是3月15日，负责人张三，目前进度正常")
	require.NoError(t, err)
	require.NoError(t, vecSvc.AddVector(context.Background(), "group_77", "doc1_chunk0", emb,
		"项目A截止日期是3月15日，负责人张三，目前进度正常",
		map[string]string{"title": "项目A进度", "group_id": "77"}))

	groupDocSvc := NewGroupDocumentService(db, nil)
	groupDocSvc.SetVectorServices(vecSvc, aiSvc)
	uk := NewUnifiedKnowledgeService(groupDocSvc, nil, nil, aiSvc)

	g := &SmartReplyGraph{db: db, unifiedKnowledge: uk}
	input := &SmartReplyContext{
		ConversationID: 7,
		UserID:         1,
		Message:        "那后来呢？", // 追问：裸 query 无法命中话题，靠上下文感知检索
		Group:          &model.Group{ID: 77, ConversationID: 7, Name: "项目群"},
	}

	g.retrieveGroupKnowledge(input)
	require.NotEmpty(t, input.KnowledgeCtx, "上下文感知检索应命中群知识库文档")
	require.NotEmpty(t, input.KnowledgeSources, "命中后应产出知识来源（徽章数据）")
	assert.Equal(t, "项目A进度", input.KnowledgeSources[0].Title, "应命中项目A文档")

	// SkipKnowledge=true 时应跳过检索（保持零值，不产出知识上下文）
	input2 := &SmartReplyContext{SkipKnowledge: true, Group: &model.Group{ID: 77}}
	g.retrieveGroupKnowledge(input2)
	assert.Empty(t, input2.KnowledgeCtx, "SkipKnowledge 时应跳过知识检索")
}
