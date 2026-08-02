package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserSettingTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.UserSetting{}))
	return db
}

// 未设置时返回默认值（不写入 DB）
func TestUserSettingService_GetOrDefault_NotExist(t *testing.T) {
	svc := NewUserSettingService(setupUserSettingTestDB(t))
	got, err := svc.GetOrDefault(uint(1), "quick_replies", `["收到"]`)
	require.NoError(t, err)
	assert.Equal(t, `["收到"]`, got)
	// 默认值不落库
	var cnt int64
	svc.db.Model(&model.UserSetting{}).Where("user_id = ? AND setting_key = ?", 1, "quick_replies").Count(&cnt)
	assert.Equal(t, int64(0), cnt)
}

// Upsert 后再读应返回写入值
func TestUserSettingService_Upsert_ThenGetOrDefault(t *testing.T) {
	svc := NewUserSettingService(setupUserSettingTestDB(t))
	require.NoError(t, svc.Upsert(uint(1), "quick_replies", `["收到","好的"]`))
	got, err := svc.GetOrDefault(uint(1), "quick_replies", `["默认"]`)
	require.NoError(t, err)
	assert.Equal(t, `["收到","好的"]`, got)
}

// 重复 Upsert 同 key 走更新，不新增行
func TestUserSettingService_Upsert_Twice_Updates(t *testing.T) {
	svc := NewUserSettingService(setupUserSettingTestDB(t))
	require.NoError(t, svc.Upsert(uint(1), "quick_replies", `["a"]`))
	require.NoError(t, svc.Upsert(uint(1), "quick_replies", `["a","b"]`))
	var cnt int64
	svc.db.Model(&model.UserSetting{}).Where("user_id = ? AND setting_key = ?", 1, "quick_replies").Count(&cnt)
	assert.Equal(t, int64(1), cnt)
	got, err := svc.GetOrDefault(uint(1), "quick_replies", "")
	require.NoError(t, err)
	assert.Equal(t, `["a","b"]`, got)
}

// 不同用户的同名 key 互不影响
func TestUserSettingService_Upsert_DifferentUsers(t *testing.T) {
	svc := NewUserSettingService(setupUserSettingTestDB(t))
	require.NoError(t, svc.Upsert(uint(1), "quick_replies", `["u1"]`))
	require.NoError(t, svc.Upsert(uint(2), "quick_replies", `["u2"]`))
	got1, _ := svc.GetOrDefault(uint(1), "quick_replies", "")
	got2, _ := svc.GetOrDefault(uint(2), "quick_replies", "")
	assert.Equal(t, `["u1"]`, got1)
	assert.Equal(t, `["u2"]`, got2)
}

// GetOrDefault 读不存在记录返回默认值，不报错
func TestUserSettingService_GetOrDefault_EmptyDefault(t *testing.T) {
	svc := NewUserSettingService(setupUserSettingTestDB(t))
	got, err := svc.GetOrDefault(uint(1), "missing_key", "")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

// Delete 删除自己的设置
func TestUserSettingService_Delete(t *testing.T) {
	svc := NewUserSettingService(setupUserSettingTestDB(t))
	require.NoError(t, svc.Upsert(uint(1), "quick_replies", `["a"]`))
	require.NoError(t, svc.Delete(uint(1), "quick_replies"))
	got, err := svc.GetOrDefault(uint(1), "quick_replies", "default")
	require.NoError(t, err)
	assert.Equal(t, "default", got)
}

// Delete 别人的设置不影响自己
func TestUserSettingService_Delete_DoesNotAffectOtherUser(t *testing.T) {
	svc := NewUserSettingService(setupUserSettingTestDB(t))
	require.NoError(t, svc.Upsert(uint(1), "quick_replies", `["u1"]`))
	require.NoError(t, svc.Upsert(uint(2), "quick_replies", `["u2"]`))
	require.NoError(t, svc.Delete(uint(1), "quick_replies"))
	got2, _ := svc.GetOrDefault(uint(2), "quick_replies", "")
	assert.Equal(t, `["u2"]`, got2)
}
