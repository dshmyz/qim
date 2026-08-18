package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAppTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.App{}))
	return db
}

// TestDeleteApp_OwnerOnly 非管理员只能删除自己的应用，删除他人应用或不存在应用返回 gorm.ErrRecordNotFound
func TestDeleteApp_OwnerOnly(t *testing.T) {
	db := setupAppTestDB(t)
	svc := NewAppService(db)

	require.NoError(t, db.Create(&model.App{UserID: 1, Name: "我的应用", URL: "https://a.com"}).Error)
	require.NoError(t, db.Create(&model.App{UserID: 2, Name: "别人的应用", URL: "https://b.com"}).Error)

	// 删除自己的应用成功
	require.NoError(t, svc.DeleteApp(1, 1, false))
	var count int64
	db.Model(&model.App{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// 删除别人的应用 → ErrRecordNotFound，数据保留
	err := svc.DeleteApp(2, 1, false)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	db.Model(&model.App{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// 不存在的应用 → ErrRecordNotFound
	err = svc.DeleteApp(999, 1, false)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestDeleteApp_AdminCanDeleteAny 管理员可删除他人的个人应用与全局应用
func TestDeleteApp_AdminCanDeleteAny(t *testing.T) {
	db := setupAppTestDB(t)
	svc := NewAppService(db)

	require.NoError(t, db.Create(&model.App{UserID: 2, Name: "别人的应用", URL: "https://b.com"}).Error)
	require.NoError(t, db.Create(&model.App{UserID: 3, Name: "全局应用", URL: "https://g.com", IsGlobal: true}).Error)

	require.NoError(t, svc.DeleteApp(1, 1, true))
	require.NoError(t, svc.DeleteApp(2, 1, true))
	var count int64
	db.Model(&model.App{}).Count(&count)
	assert.Equal(t, int64(0), count)
}
