package test

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupInitTestDataDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.Department{},
		&model.DepartmentEmployee{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Group{},
		&model.Message{},
		&model.Bot{},
		&model.BotConversation{},
	))

	return db
}

func TestInitTestDataSkipsBotUsersWhenCreatingBotConversations(t *testing.T) {
	db := setupInitTestDataDB(t)

	normalUsers := []model.User{
		{Username: "user1", PasswordHash: "hash", Nickname: "用户一", Type: "user"},
		{Username: "user2", PasswordHash: "hash", Nickname: "用户二", Type: "user"},
		{Username: "user3", PasswordHash: "hash", Nickname: "用户三", Type: "user"},
		{Username: "user4", PasswordHash: "hash", Nickname: "用户四", Type: "user"},
		{Username: "user5", PasswordHash: "hash", Nickname: "用户五", Type: "user"},
		{Username: "user6", PasswordHash: "hash", Nickname: "用户六", Type: "user"},
		{Username: "user7", PasswordHash: "hash", Nickname: "用户七", Type: "user"},
	}
	for i := range normalUsers {
		require.NoError(t, db.Create(&normalUsers[i]).Error)
	}

	strayBotUser := model.User{Username: "bot_stray", PasswordHash: "hash", Nickname: "不应建会话的机器人", Type: "bot"}
	require.NoError(t, db.Create(&strayBotUser).Error)

	systemBot := model.Bot{Name: "系统助手", Type: model.BotTypeSystem, IsActive: true}
	assistantBot := model.Bot{Name: "写作助手", Type: model.BotTypeAssistant, IsActive: true}
	require.NoError(t, db.Create(&systemBot).Error)
	require.NoError(t, db.Create(&assistantBot).Error)

	InitTestData(db)

	var botUserConvCount int64
	require.NoError(t, db.Model(&model.BotConversation{}).Where("user_id = ?", strayBotUser.ID).Count(&botUserConvCount).Error)
	assert.Equal(t, int64(0), botUserConvCount)

	var normalUserConvCount int64
	require.NoError(t, db.Model(&model.BotConversation{}).Where("user_id = ?", normalUsers[0].ID).Count(&normalUserConvCount).Error)
	assert.Greater(t, normalUserConvCount, int64(0))
}
