package service

import (
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncateRunes(t *testing.T) {
	// 短文本原样返回
	assert.Equal(t, "你好", truncateRunes("你好", 10))
	// 超长按 rune 截断并加省略号，中文不被切碎
	assert.Equal(t, "你好…", truncateRunes("你好世界", 2))
	assert.Equal(t, "你好世…", truncateRunes("你好世界", 3))
	// 恰好等于上限不截断
	assert.Equal(t, "abcd", truncateRunes("abcd", 4))
	// 空串
	assert.Equal(t, "", truncateRunes("", 10))
	// maxRunes<=0 视为不截断
	assert.Equal(t, "abc", truncateRunes("abc", 0))
	// 多字节：保证截断后仍是合法 UTF-8（rune 级，不会从汉字中间断）
	out := truncateRunes("中文字符串测试", 3)
	assert.Equal(t, "中文字…", out)
}

func TestContextualQuery(t *testing.T) {
	history := "Alice: 上周的项目方案\nBob: 我们决定用 MySQL\nAlice: 那后来呢"
	// 基本：历史 + 当前提问
	q := contextualQuery(history, "具体说说", 4)
	assert.Contains(t, q, "我们决定用 MySQL", "历史应拼进 query")
	assert.Contains(t, q, "当前提问：具体说说", "当前提问应拼进 query")

	// recent 上限：只取最近 N 行历史
	q2 := contextualQuery(history, "具体说说", 1)
	assert.NotContains(t, q2, "Alice: 上周的项目方案", "超过 recent 的历史行应被丢弃")
	assert.Contains(t, q2, "那后来呢", "最近一行历史应保留")

	// 历史为空 → 退化为原 query
	assert.Equal(t, "具体说说", contextualQuery("", "具体说说", 4))
	assert.Equal(t, "具体说说", contextualQuery("   \n  ", "具体说说", 4))

	// trigger 为空 + 有历史 → 用占位符
	q3 := contextualQuery("Alice: hi", "", 4)
	assert.Contains(t, q3, "当前提问：")
}

func TestFetchRecentHistoryForQuery(t *testing.T) {
	// setupServiceTestDB 已 AutoMigrate User/Message 等表
	db := setupServiceTestDB(t)

	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Nickname: "Alice", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "bob", Nickname: "Bob", PasswordHash: "h"}).Error)

	// 3 条文本 + 1 条图片（应被过滤，只取文本）
	msgs := []model.Message{
		{ConversationID: 7, SenderID: 1, Type: "text", Content: "第一条"},
		{ConversationID: 7, SenderID: 2, Type: "text", Content: "第二条"},
		{ConversationID: 7, SenderID: 1, Type: "text", Content: "触发消息"},
		{ConversationID: 7, SenderID: 2, Type: "image", Content: "图片消息"},
	}
	for i := range msgs {
		require.NoError(t, db.Create(&msgs[i]).Error)
	}

	// 排除触发消息（content+sender 匹配）
	hist, err := fetchRecentHistoryForQuery(db, 7, 1, "触发消息")
	require.NoError(t, err)
	assert.NotContains(t, hist, "触发消息", "触发消息本身应被排除，避免 query 自引用")
	assert.Contains(t, hist, "Alice: 第一条")
	assert.Contains(t, hist, "Bob: 第二条")
	assert.NotContains(t, hist, "图片消息", "非 text/markdown 消息不应进入历史")

	// 无 dedupe 参数时保留全部文本
	hist2, err := fetchRecentHistoryForQuery(db, 7, 1, "")
	require.NoError(t, err)
	assert.Contains(t, hist2, "触发消息")

	// 时间正序：第一条在第二条之前
	assert.True(t, strings.Index(hist2, "第一条") < strings.Index(hist2, "第二条"), "历史应按时间正序")
}

// TestFetchRecentHistoryForQuery_ReturnsErrorOnDBFailure 数据库查询失败时必须显式返回 error，
// 不能静默吞掉后返回空历史——否则上下文感知检索在 DB 故障时会"假装没有历史"继续工作，
// 多轮追问召回必然失败且无从排查。
func TestFetchRecentHistoryForQuery_ReturnsErrorOnDBFailure(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Nickname: "Alice", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: 7, SenderID: 1, Type: "text", Content: "第一条"}).Error)

	// 关闭底层连接使后续查询必然报错
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = fetchRecentHistoryForQuery(db, 7, 1, "")
	require.Error(t, err, "DB 查询失败时应显式返回 error，而非静默返回空历史")
}
