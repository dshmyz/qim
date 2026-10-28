package handler

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunQualityGateAsyncRecordsAdvisory 旁路模式下正文先随流上屏，核验改后台跑：
// 必须异步记录 QualityAdvisory，且不阻塞调用方。
func TestRunQualityGateAsyncRecordsAdvisory(t *testing.T) {
	svc, _ := newQualityGateStubAI(t, func([]ai.Message) (string, error) {
		return failingVerdictJSON, nil
	})
	e := &SmartReplyEngine{aiService: svc, qualityGateEnabled: true, mentionGateBlock: false}
	input := &service.SmartReplyContext{Message: "问题", KnowledgeCtx: "知识库资料"}

	before := service.GlobalAIReplyMetrics.Snapshot()
	e.runQualityGateAsync(input, "一条含臆造事实的回复", nil, func() *model.Message { return nil })

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if service.GlobalAIReplyMetrics.Snapshot().QualityAdvisory > before.QualityAdvisory {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("异步核验未在超时内记录 advisory（before=%d）", before.QualityAdvisory)
}

// TestAnnotateQualityDoubtful 核验存疑标注落库：Extra 出现 quality，且不覆盖既有字段。
func TestAnnotateQualityDoubtful(t *testing.T) {
	db := setupHandlerTestDB(t)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })

	msg := &model.Message{
		ConversationID: 9, SenderID: 1, Type: "text", Content: "回复正文",
		Extra: `{"knowledge_sources":[{"title":"文档"}]}`,
	}
	require.NoError(t, db.Create(msg).Error)

	e := &SmartReplyEngine{messageSender: &WebSocketMessageSender{}} // hub 为 nil：广播 no-op
	e.annotateQualityDoubtful(msg, "回复包含资料中不存在的具体事实", 0.8)

	var saved model.Message
	require.NoError(t, db.First(&saved, msg.ID).Error)
	fields := service.ParseMessageExtraFields(saved.Extra)
	require.NotNil(t, fields)
	q, ok := fields["quality"].(map[string]interface{})
	require.True(t, ok, "quality 应随 Extra 落库: %s", saved.Extra)
	assert.Equal(t, "doubtful", q["status"])
	assert.Equal(t, "回复包含资料中不存在的具体事实", q["reason"])
	assert.Contains(t, fields, "knowledge_sources", "标注不得覆盖既有 Extra 字段")

	// 消息从未创建（getMsg 返回 nil）时安全跳过。
	e.annotateQualityDoubtful(nil, "原因", 0)
}

// TestRunQualityGateAsyncNoop 空正文/核验关闭时不触发后台调用。
func TestRunQualityGateAsyncNoop(t *testing.T) {
	called := false
	svc, _ := newQualityGateStubAI(t, func([]ai.Message) (string, error) {
		called = true
		return failingVerdictJSON, nil
	})
	input := &service.SmartReplyContext{Message: "问题", KnowledgeCtx: "资料"}

	(&SmartReplyEngine{aiService: svc, qualityGateEnabled: false}).runQualityGateAsync(input, "正文", nil, func() *model.Message { return nil })
	(&SmartReplyEngine{aiService: svc, qualityGateEnabled: true}).runQualityGateAsync(input, "   ", nil, func() *model.Message { return nil })

	time.Sleep(50 * time.Millisecond)
	if called {
		t.Fatal("核验关闭或正文为空时不应调用 judge")
	}
}
