package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSendMessage_SensitiveCheck_MarkdownType 验证 SendMessage 对 markdown 类型消息也执行敏感词检查。
// markdown 类型含敏感词应被拦截，不含敏感词应正常发送。
func TestSendMessage_SensitiveCheck_MarkdownType(t *testing.T) {
	db := setupServiceTestDB(t)

	// 准备：创建用户与会话（参考 TestMessageService_SendMessage 的内联构造方式）
	user1 := &model.User{Username: "u-markdown-1", PasswordHash: "hash", Nickname: "Markdown Sender"}
	user2 := &model.User{Username: "u-markdown-2", PasswordHash: "hash", Nickname: "Markdown Receiver"}
	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)

	conv, err := NewConversationService(db).CreateSingleConversation(user1.ID, user2.ID)
	require.NoError(t, err)

	// 直接构造 MessageService，注入敏感词缓存，避免触发 db 查询 sensitive_words 表。
	// sensitiveWordLoaded=true 使 CheckSensitiveContent 直接使用缓存，不调用 loadSensitiveWords。
	svc := &MessageService{
		db:                  db,
		sensitiveWordCache:  []model.SensitiveWord{{Word: "禁用词", Enabled: true}},
		sensitiveWordLoaded: true,
	}

	// markdown 类型消息含敏感词，应被拦截
	_, err = svc.SendMessage(conv.ID, user1.ID, "markdown", "```go\n// 禁用词\nfmt.Println(1)\n```", nil)
	assert.ErrorIs(t, err, ErrSensitiveWordBlocked)

	// markdown 类型消息不含敏感词，正常发送
	msg, err := svc.SendMessage(conv.ID, user1.ID, "markdown", "```go\nfmt.Println(1)\n```", nil)
	require.NoError(t, err)
	assert.Equal(t, "markdown", msg.Type)
	assert.Equal(t, "```go\nfmt.Println(1)\n```", msg.Content)
}

// TestRefreshSensitiveWordCache_Success 验证正常情况下 RefreshSensitiveWordCache 返回 nil，
// 且仅将 enabled=true 的敏感词载入缓存。
// 对应修复 S2：缓存刷新失败不能再被静默吞掉，函数必须返回 error。
func TestRefreshSensitiveWordCache_Success(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SensitiveWord{}))

	// 启用的敏感词应进入缓存
	require.NoError(t, db.Create(&model.SensitiveWord{Word: "敏感1", Level: "medium", Enabled: true}).Error)
	// 停用的敏感词不应进入缓存。
	// 注意：model.SensitiveWord 的 Enabled 字段带 gorm:"default:true"，
	// 用结构体 Create 时 false 是零值会被 GORM 替换为默认值 true，故用 map 显式写入 false。
	require.NoError(t, db.Model(&model.SensitiveWord{}).Create(map[string]interface{}{
		"word": "停用词", "level": "medium", "enabled": false,
	}).Error)
	require.NoError(t, db.Create(&model.SensitiveWord{Word: "敏感2", Level: "high", Enabled: true}).Error)

	svc := &MessageService{db: db}
	err := svc.RefreshSensitiveWordCache()
	require.NoError(t, err)

	svc.sensitiveWordCacheMu.RLock()
	loaded := svc.sensitiveWordLoaded
	cache := svc.sensitiveWordCache
	svc.sensitiveWordCacheMu.RUnlock()

	assert.True(t, loaded, "缓存应标记为已加载")
	assert.Len(t, cache, 2, "仅启用的敏感词进入缓存")

	// 验证 CheckSensitiveContent 能命中新刷新的缓存
	contains, words := svc.CheckSensitiveContent("包含敏感1的内容")
	assert.True(t, contains)
	assert.Contains(t, words, "敏感1")
}

// TestRefreshSensitiveWordCache_DBError 验证 DB 查询失败时 RefreshSensitiveWordCache 返回 error，
// 而不是静默吞掉（历史 bug：CRUD 成功但缓存不一致，管理员却看到成功）。
func TestRefreshSensitiveWordCache_DBError(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SensitiveWord{}))

	svc := &MessageService{db: db}

	// 删除表使后续查询失败，模拟 DB 异常
	require.NoError(t, db.Migrator().DropTable(&model.SensitiveWord{}))

	err := svc.RefreshSensitiveWordCache()
	assert.Error(t, err, "DB 查询失败时应返回 error，不能静默吞掉")
}
