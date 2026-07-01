package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.ChannelSubscriber{},
	))
	return db
}

func countSubs(db *gorm.DB, channelID, userID uint) int64 {
	var n int64
	db.Model(&model.ChannelSubscriber{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).Count(&n)
	return n
}

// 真人用户创建后应自动订阅默认频道（AfterCreate 钩子）
func TestUserAfterCreate_SubscribesDefaultChannel(t *testing.T) {
	db := setupChannelTestDB(t)

	defaultCh := model.Channel{Name: "公告频道", Status: "active", IsDefault: true}
	require.NoError(t, db.Create(&defaultCh).Error)
	normalCh := model.Channel{Name: "普通频道", Status: "active", IsDefault: false}
	require.NoError(t, db.Create(&normalCh).Error)

	user := model.User{Username: "alice", Type: "user"}
	require.NoError(t, db.Create(&user).Error)

	assert.Equal(t, int64(1), countSubs(db, defaultCh.ID, user.ID), "真人用户应订阅默认频道")
	assert.Equal(t, int64(0), countSubs(db, normalCh.ID, user.ID), "不应订阅非默认频道")
}

// bot/system 用户不应自动订阅
func TestUserAfterCreate_SkipsNonHumanUsers(t *testing.T) {
	db := setupChannelTestDB(t)

	defaultCh := model.Channel{Name: "公告频道", Status: "active", IsDefault: true}
	require.NoError(t, db.Create(&defaultCh).Error)

	bot := model.User{Username: "bot_x", Type: "bot"}
	require.NoError(t, db.Create(&bot).Error)
	sys := model.User{Username: "system", Type: "system"}
	require.NoError(t, db.Create(&sys).Error)

	assert.Equal(t, int64(0), countSubs(db, defaultCh.ID, bot.ID), "bot 不应订阅")
	assert.Equal(t, int64(0), countSubs(db, defaultCh.ID, sys.ID), "system 不应订阅")
}

// 停用的默认频道不订阅
func TestUserAfterCreate_SkipsInactiveDefaultChannel(t *testing.T) {
	db := setupChannelTestDB(t)

	inactiveCh := model.Channel{Name: "停用频道", Status: "inactive", IsDefault: true}
	require.NoError(t, db.Create(&inactiveCh).Error)

	user := model.User{Username: "bob", Type: "user"}
	require.NoError(t, db.Create(&user).Error)

	assert.Equal(t, int64(0), countSubs(db, inactiveCh.ID, user.ID), "停用的默认频道不应订阅")
}
