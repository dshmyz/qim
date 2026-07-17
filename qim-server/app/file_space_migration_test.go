package app

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrateFileSpacesBackfillsLegacyUserRecords(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Exec(`
		CREATE TABLE files (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, scope_type TEXT,
			scope_id INTEGER NOT NULL DEFAULT 0, name TEXT, size INTEGER, storage_path TEXT,
			updated_at DATETIME, deleted_at DATETIME
		);
		CREATE TABLE folders (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, scope_type TEXT,
			scope_id INTEGER NOT NULL DEFAULT 0, name TEXT, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO files (id, user_id, scope_type, scope_id, name, size, storage_path) VALUES
			(1, 7, '', 0, 'empty-file.txt', 1, 'files/empty'),
			(2, 8, NULL, 0, 'null-file.txt', 1, 'files/null'),
			(3, 9, 'user', 0, 'unscoped-user-file.txt', 1, 'files/current');
		INSERT INTO folders (id, user_id, scope_type, scope_id, name) VALUES
			(1, 7, '', 0, 'empty-folder'),
			(2, 8, NULL, 0, 'null-folder'),
			(3, 9, 'user', 0, 'unscoped-user-folder')
	`).Error)

	require.NoError(t, MigrateFileSpaces(db))
	assertScopes(t, db, []uint{1, 2}, "user", []uint{7, 8})
	assertScopes(t, db, []uint{3}, "user", []uint{0})

	// A second execution must leave the already-backfilled rows unchanged.
	require.NoError(t, MigrateFileSpaces(db))
	assertScopes(t, db, []uint{1, 2}, "user", []uint{7, 8})
}

func assertScopes(t *testing.T, db *gorm.DB, ids []uint, scopeType string, scopeIDs []uint) {
	t.Helper()
	for i, id := range ids {
		var file model.File
		require.NoError(t, db.First(&file, id).Error)
		require.Equal(t, scopeType, file.ScopeType)
		require.Equal(t, scopeIDs[i], file.ScopeID)

		var folder model.Folder
		require.NoError(t, db.First(&folder, id).Error)
		require.Equal(t, scopeType, folder.ScopeType)
		require.Equal(t, scopeIDs[i], folder.ScopeID)
	}
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
	require.NoError(t, db.Model(&model.File{}).Where("id = ?", file.ID).Update("scope_type", "").Error)
	require.NoError(t, db.Model(&model.Folder{}).Where("id = ?", folder.ID).Update("scope_type", "").Error)

	MigrateDB(db)

	require.NoError(t, db.First(&file, file.ID).Error)
	require.NoError(t, db.First(&folder, folder.ID).Error)
	require.Equal(t, "user", file.ScopeType)
	require.Equal(t, uint(8), file.ScopeID)
	require.Equal(t, "user", folder.ScopeType)
	require.Equal(t, uint(8), folder.ScopeID)
}
