package model

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// valueHolder 用于验证 StringArray 作为 GORM 字段能正常存取（覆盖多元素数组）。
type valueHolder struct {
	ID     uint        `gorm:"primarykey"`
	Models StringArray `gorm:"type:text"`
}

func setupStringArrayTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&valueHolder{}))
	return db
}

// 多元素 StringArray 应能正常 INSERT + Scan。
// 回归：Value() 若不返回 driver.Value，GORM 会把 []string 当作 SQL 元组，
// 触发 SQLite 的「row value misused」，多元素插入即失败。
func TestStringArray_MultiElementRoundTrip(t *testing.T) {
	db := setupStringArrayTestDB(t)

	original := StringArray{"gpt-4o", "gpt-4o-mini", "deepseek-v4-flash"}
	require.NoError(t, db.Create(&valueHolder{Models: original}).Error)

	var got valueHolder
	require.NoError(t, db.First(&got).Error)
	require.Equal(t, original, got.Models)
	require.Len(t, got.Models, 3)
}

// 单元素同样正常。
func TestStringArray_SingleElementRoundTrip(t *testing.T) {
	db := setupStringArrayTestDB(t)

	original := StringArray{"gpt-4o"}
	require.NoError(t, db.Create(&valueHolder{Models: original}).Error)

	var got valueHolder
	require.NoError(t, db.First(&got).Error)
	require.Equal(t, original, got.Models)
}

// nil 应存成空数组 "[]"，读回为空切片（长度 0），而非报错。
func TestStringArray_NilValue(t *testing.T) {
	db := setupStringArrayTestDB(t)
	require.NoError(t, db.Create(&valueHolder{}).Error)

	var got valueHolder
	require.NoError(t, db.First(&got).Error)
	require.NotNil(t, got.Models)
	require.Len(t, got.Models, 0)
}
