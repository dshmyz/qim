package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCleanupStaleStreamingMessages_ConvertsStaleToMarkdown 僵尸流式消息超时收尾为 markdown。
func TestCleanupStaleStreamingMessages_ConvertsStaleToMarkdown(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	// stale：updated_at 早于 10 分钟前
	stale := model.Message{Type: "streaming", Content: "部分内容", UpdatedAt: time.Now().Add(-11 * time.Minute)}
	require.NoError(t, db.Create(&stale).Error)

	n, err := CleanupStaleStreamingMessages(db, 10*time.Minute)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	var got model.Message
	require.NoError(t, db.First(&got, stale.ID).Error)
	assert.Equal(t, "markdown", got.Type, "僵尸流式应转 markdown")
	assert.Equal(t, "部分内容", got.Content, "已累积 content 保留")
}

// TestCleanupStaleStreamingMessages_KeepsActiveStream 活跃流（updated_at 近）不误杀。
func TestCleanupStaleStreamingMessages_KeepsActiveStream(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	fresh := model.Message{Type: "streaming", Content: "正在生成", UpdatedAt: time.Now().Add(-30 * time.Second)}
	require.NoError(t, db.Create(&fresh).Error)
	markdown := model.Message{Type: "markdown", Content: "已完成", UpdatedAt: time.Now().Add(-1 * time.Hour)}
	require.NoError(t, db.Create(&markdown).Error)

	n, err := CleanupStaleStreamingMessages(db, 10*time.Minute)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "活跃流与非流式都不应被改")

	var s model.Message
	require.NoError(t, db.First(&s, fresh.ID).Error)
	assert.Equal(t, "streaming", s.Type, "活跃流保持 streaming")
	var m model.Message
	require.NoError(t, db.First(&m, markdown.ID).Error)
	assert.Equal(t, "markdown", m.Type, "已完成的 markdown 不受影响")
}
