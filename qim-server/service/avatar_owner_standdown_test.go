package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupAvatarStanddownTestDB 内存库 + 消息/分身会话表，供 standDown 表测试。
func setupAvatarStanddownTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Message{}, &model.AvatarSession{}))
	return db
}

// TestAvatarStandDown 覆盖让位门三条件（单源权威判定）：
// 接管期内让位；主人触发后亲自发言让位；分身触发后已代答让位；
// 分身自身回复不算主人发言；触发前的历史状态不算；DB 异常/缺表 fail-open。
func TestAvatarStandDown(t *testing.T) {
	db := setupAvatarStanddownTestDB(t)
	p := &AvatarWorkerPool{db: db}

	convID := uint(100)
	ownerID := uint(3)
	otherID := uint(4)
	triggerAt := time.Now().Add(-time.Minute)

	task := AvatarTask{UserID: ownerID, ConversationID: convID}

	insertMsg := func(senderID uint, origin string, at time.Time) {
		require.NoError(t, db.Create(&model.Message{
			ConversationID: convID,
			SenderID:       senderID,
			Origin:         origin,
			Type:           "text",
			Content:        "msg",
			CreatedAt:      at,
		}).Error)
	}
	setTakeover := func(until *time.Time) {
		if until == nil {
			// 清掉 session 行，模拟合成默认开会话（无行）
			db.Where("1=1").Delete(&model.AvatarSession{})
			return
		}
		var cnt int64
		db.Model(&model.AvatarSession{}).Where("user_id = ? AND conversation_id = ?", ownerID, convID).Count(&cnt)
		if cnt == 0 {
			require.NoError(t, db.Create(&model.AvatarSession{
				ConversationID: convID, UserID: ownerID, AvatarEnabled: true, TakeoverUntil: until,
			}).Error)
		} else {
			require.NoError(t, db.Model(&model.AvatarSession{}).
				Where("user_id = ? AND conversation_id = ?", ownerID, convID).
				Update("takeover_until", until).Error)
		}
	}
	reset := func() {
		db.Where("1=1").Delete(&model.Message{})
	}

	// 空库 + 无 session 行：不让位（合成默认开会话基线）
	_, down, reason := p.standDown(task, triggerAt)
	assert.False(t, down, "空库基线不应让位")
	assert.Empty(t, reason)

	// ① takeover：接管期内让位；过期后放行
	future := triggerAt.Add(2 * time.Hour)
	setTakeover(&future)
	_, down, reason = p.standDown(task, triggerAt)
	assert.True(t, down, "接管期内应让位")
	assert.Equal(t, "接管期内", reason)

	past := triggerAt.Add(-time.Minute)
	setTakeover(&past)
	_, down, _ = p.standDown(task, triggerAt)
	assert.False(t, down, "接管已过期不应让位")
	setTakeover(nil)

	// ② ownerReplied：主人真实消息（legacy 空 origin）触发后 → 让位
	reset()
	insertMsg(ownerID, "", triggerAt.Add(time.Second))
	_, down, reason = p.standDown(task, triggerAt)
	assert.True(t, down, "主人触发后亲自发言应让位")
	assert.Equal(t, "主人已亲自发言", reason)

	// 只有分身自身回复（origin=avatar，挂在主人名下）→ 不算主人发言，但命中 ③ superseded
	reset()
	insertMsg(ownerID, "avatar", triggerAt.Add(time.Second))
	_, down, reason = p.standDown(task, triggerAt)
	assert.True(t, down, "分身触发后已代答应让位")
	assert.Equal(t, "分身已代答过", reason)

	// 机器人消息（origin=assistant）挂主人 sender 下属构造用例：三条件均不命中
	reset()
	insertMsg(ownerID, "assistant", triggerAt.Add(time.Second))
	_, down, reason = p.standDown(task, triggerAt)
	assert.False(t, down, "机器人消息既不算主人发言也不算分身代答")
	assert.Empty(t, reason)

	// 触发后只有他人消息 → 不让位
	reset()
	insertMsg(otherID, "", triggerAt.Add(time.Second))
	_, down, _ = p.standDown(task, triggerAt)
	assert.False(t, down)

	// 主人消息在触发前 → 不让位（历史发言）
	reset()
	insertMsg(ownerID, "user", triggerAt.Add(-time.Second))
	_, down, _ = p.standDown(task, triggerAt)
	assert.False(t, down)

	// nil DB 安全降级（不 panic、不让位）
	nilPool := &AvatarWorkerPool{}
	_, down, reason = nilPool.standDown(task, triggerAt)
	assert.False(t, down)
	assert.Empty(t, reason)
}
