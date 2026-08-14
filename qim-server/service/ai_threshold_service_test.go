package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAiThresholdService_GetFloat_Default(t *testing.T) {
	svc := NewAiThresholdService(nil) // nil DB → 全部走默认值
	assert.Equal(t, 0.6, svc.GetFloat("ai.knowledge_score_threshold", 0.6))
	assert.Equal(t, 0.5, svc.GetFloat("ai.memory_recall_threshold", 0.5))
	assert.Equal(t, 0.7, svc.GetFloat("ai.conflict_detection_threshold", 0.7))
	assert.Equal(t, 20.0, svc.GetFloat("ai.context_history_limit", 20))
	assert.Equal(t, 0.99, svc.GetFloat("ai.unknown_key", 0.99), "未知 key 应返回指定默认值")
}

func TestAiThresholdService_GetInt(t *testing.T) {
	svc := NewAiThresholdService(nil)
	assert.Equal(t, 20, svc.GetInt("ai.context_history_limit", 20))
	assert.Equal(t, 5, svc.GetInt("ai.recent_ai_messages_limit", 5))
}

func TestAiThresholdService_GetAll(t *testing.T) {
	svc := NewAiThresholdService(nil)
	all := svc.GetAll()
	require.Len(t, all, len(config.Thresholds))
	assert.Equal(t, 0.3, all["ai.knowledge_score_threshold"])
	assert.Equal(t, 0.3, all["ai.memory_source_threshold"])
	assert.Equal(t, 0.5, all["ai.memory_recall_threshold"])
	assert.Equal(t, 0.7, all["ai.conflict_detection_threshold"])
	assert.Equal(t, 20.0, all["ai.context_history_limit"])
	assert.Equal(t, 5.0, all["ai.recent_ai_messages_limit"])
}

func TestAiThresholdService_BatchUpdate_Validation(t *testing.T) {
	svc := NewAiThresholdService(nil) // nil DB → BatchUpdate 会失败，但校验在 DB 操作前

	// 未知 key 应拒绝
	err := svc.BatchUpdate(map[string]interface{}{
		"ai.unknown_key": 0.5,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未知的阈值 key")

	// 超出范围应拒绝
	err = svc.BatchUpdate(map[string]interface{}{
		"ai.knowledge_score_threshold": 1.5, // 超出 [0, 1]
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "超出范围")

	// 合法值但 nil DB → DB 操作失败
	err = svc.BatchUpdate(map[string]interface{}{
		"ai.knowledge_score_threshold": 0.8,
	})
	assert.Error(t, err, "nil DB 应返回错误")
}
