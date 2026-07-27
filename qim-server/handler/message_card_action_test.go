package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCardActionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Message{}, &model.CardActionRecord{}))
	database.DB = db
	return db
}

// TestBuildCardActionMap_ReturnsClickedActions 只返当前用户已点的卡片消息 action_id。
func TestBuildCardActionMap_ReturnsClickedActions(t *testing.T) {
	db := setupCardActionTestDB(t)

	card1 := model.Message{Type: "card", Content: "{}"}
	card2 := model.Message{Type: "card", Content: "{}"}
	card3 := model.Message{Type: "card", Content: "{}"}
	textMsg := model.Message{Type: "text", Content: "hi"}
	require.NoError(t, db.Create(&card1).Error)
	require.NoError(t, db.Create(&card2).Error)
	require.NoError(t, db.Create(&card3).Error)
	require.NoError(t, db.Create(&textMsg).Error)

	// 用户 5 点了 card1(confirm) 和 card3(cancel)；card2 未点
	require.NoError(t, db.Create(&model.CardActionRecord{MessageID: card1.ID, UserID: 5, ActionID: "confirm", BotID: 1}).Error)
	require.NoError(t, db.Create(&model.CardActionRecord{MessageID: card3.ID, UserID: 5, ActionID: "cancel", BotID: 1}).Error)
	// 另一用户对 card1 的记录不应计入用户 5
	require.NoError(t, db.Create(&model.CardActionRecord{MessageID: card1.ID, UserID: 9, ActionID: "confirm", BotID: 1}).Error)

	msgs := []model.Message{card1, card2, card3, textMsg}
	m := buildCardActionMap(msgs, 5)
	assert.Equal(t, "confirm", m[card1.ID], "card1 被用户5点了 confirm")
	assert.Equal(t, "cancel", m[card3.ID], "card3 被用户5点了 cancel")
	_, has2 := m[card2.ID]
	assert.False(t, has2, "card2 未被点，不应出现")
	assert.Len(t, m, 2, "仅 2 张卡片被当前用户点过")
}

// TestBuildCardActionMap_NoCardsReturnsEmpty 无卡片消息时返回空 map（不查库）。
func TestBuildCardActionMap_NoCardsReturnsEmpty(t *testing.T) {
	setupCardActionTestDB(t)
	msgs := []model.Message{{Type: "text"}, {Type: "markdown"}}
	m := buildCardActionMap(msgs, 5)
	assert.NotNil(t, m)
	assert.Empty(t, m)
}

// 引用 sqlite 包避免未使用（与其它 handler 测试一致）
var _ = sqlite.Open
