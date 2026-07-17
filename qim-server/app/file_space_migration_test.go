package app

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/require"
)

func TestMigrateFileSpacesBackfillsLegacyUserRecords(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.File{}, &model.Folder{}))

	file := model.File{UserID: 7, Name: "legacy.txt", Size: 1, StoragePath: "files/legacy"}
	folder := model.Folder{UserID: 7, Name: "旧文件夹"}
	require.NoError(t, db.Create(&file).Error)
	require.NoError(t, db.Create(&folder).Error)

	require.NoError(t, MigrateFileSpaces(db))
	require.NoError(t, db.First(&file, file.ID).Error)
	require.Equal(t, "user", file.ScopeType)
	require.Equal(t, uint(7), file.ScopeID)
	require.NoError(t, db.First(&folder, folder.ID).Error)
	require.Equal(t, "user", folder.ScopeType)
	require.Equal(t, uint(7), folder.ScopeID)
}

func TestMigrateFileSpacesLeavesExistingScopesUntouched(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.File{}, &model.Folder{}))

	file := model.File{UserID: 7, Name: "shared.txt", Size: 1, StoragePath: "files/shared", ScopeType: "group", ScopeID: 19}
	folder := model.Folder{UserID: 7, Name: "共享文件夹", ScopeType: "group", ScopeID: 19}
	require.NoError(t, db.Create(&file).Error)
	require.NoError(t, db.Create(&folder).Error)

	require.NoError(t, MigrateFileSpaces(db))
	require.NoError(t, db.First(&file, file.ID).Error)
	require.Equal(t, "group", file.ScopeType)
	require.Equal(t, uint(19), file.ScopeID)
	require.NoError(t, db.First(&folder, folder.ID).Error)
	require.Equal(t, "group", folder.ScopeType)
	require.Equal(t, uint(19), folder.ScopeID)
}

func TestMigrateDBBackfillsLegacyFileSpaces(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.File{}, &model.Folder{}))
	file := model.File{UserID: 8, Name: "legacy.txt", Size: 1, StoragePath: "files/legacy"}
	folder := model.Folder{UserID: 8, Name: "旧文件夹"}
	require.NoError(t, db.Create(&file).Error)
	require.NoError(t, db.Create(&folder).Error)

	MigrateDB(db)

	require.NoError(t, db.First(&file, file.ID).Error)
	require.NoError(t, db.First(&folder, folder.ID).Error)
	require.Equal(t, "user", file.ScopeType)
	require.Equal(t, uint(8), file.ScopeID)
	require.Equal(t, "user", folder.ScopeType)
	require.Equal(t, uint(8), folder.ScopeID)
}
