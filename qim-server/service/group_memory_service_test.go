package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
)

// GroupMemoryService 在向量库/AI 服务未配置时必须安全降级为 no-op，
// 不能让群助手回复路径因记忆服务缺失而报错。以下用 nil db / nil aiService 验证。

func newNilGroupMemoryService() *GroupMemoryService {
	return &GroupMemoryService{db: nil, aiService: nil}
}

func TestGroupMemoryService_NilDBSafeNoOp(t *testing.T) {
	s := newNilGroupMemoryService()

	assert.NoError(t, s.Remember(1, 2, "群决定：每周五例会", 3), "Remember 在 db=nil 时应 no-op 不报错")

	results, err := s.Recall(1, "例会", 5)
	assert.NoError(t, err)
	assert.Empty(t, results, "Recall 在 db=nil 时应返回空")

	memories, err := s.GetGroupMemories(1, 50)
	assert.NoError(t, err)
	assert.Empty(t, memories, "GetGroupMemories 在 db=nil 时应返回空")

	count, err := s.GetMemoryCount(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	assert.NoError(t, s.DeleteMemory(1, "any"), "DeleteMemory 在 db=nil 时应 no-op")

	deleted, err := s.ForgetAll(1)
	assert.NoError(t, err)
	assert.Equal(t, 0, deleted, "ForgetAll 在 db=nil 时应删 0 条")
}

func TestGroupMemoryService_ShouldRememberNilAI(t *testing.T) {
	s := &GroupMemoryService{db: nil, aiService: nil}
	should, err := s.ShouldRemember("群决定：下周三发布")
	assert.NoError(t, err)
	assert.False(t, should, "aiService=nil 时 ShouldRemember 应降级为 false（不记），避免误存")
}

// 隔离契约：群记忆的 namespace 必须与分身记忆（"avatar"）不同，
// 且 UserID 用 groupID 键--这条契约保证两套记忆不串。改动 namespace 需同步迁移数据。
func TestGroupMemoryService_NamespaceIsolation(t *testing.T) {
	assert.Equal(t, "group_assistant", groupMemoryNamespace, "群记忆 namespace 必须与分身(avatar)隔离")
	assert.NotEqual(t, "avatar", groupMemoryNamespace, "群记忆不得复用分身 namespace")
}

// 编译期保证 NewGroupMemoryService 在 vectorSvc=nil 时不 panic 且返回可用的 no-op 服务。
func TestNewGroupMemoryService_NilVectorSvc(t *testing.T) {
	s := NewGroupMemoryService(nil, new(ai.AIService))
	assert.NotNil(t, s)
	assert.Nil(t, s.db, "vectorSvc=nil 时 db 应为 nil（no-op 模式）")
}
