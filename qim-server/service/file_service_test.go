package service

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type deleteCountingAccessor struct {
	StorageAccessor
	mu          sync.Mutex
	deleteCalls []string
}

func (a *deleteCountingAccessor) DeleteByPath(ctx context.Context, storagePath string) error {
	a.mu.Lock()
	a.deleteCalls = append(a.deleteCalls, storagePath)
	a.mu.Unlock()
	return a.StorageAccessor.DeleteByPath(ctx, storagePath)
}

func (a *deleteCountingAccessor) DeleteCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.deleteCalls)
}

func (a *deleteCountingAccessor) GetByPath(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	return a.StorageAccessor.GetByPath(ctx, storagePath)
}

func setupFileServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.File{}, &model.Folder{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestFileService_BatchDelete_DeletesPhysicalFiles 验证批量删除同时清理物理文件，
// 而非只删 DB 记录留下孤儿文件（local/S3 均适用）。
func TestFileService_BatchDelete_DeletesPhysicalFiles(t *testing.T) {
	db := setupFileServiceTestDB(t)
	user := &model.User{Username: "u", PasswordHash: "h"}
	assert.NoError(t, db.Create(user).Error)

	accessor := newTestAccessor(t)
	svc := NewFileService(db)
	svc.SetStorageAccessor(accessor)

	ctx := context.Background()
	sp1, err := accessor.Put(ctx, "uploads/2026/07/a.txt", strings.NewReader("aaa"), 3, "text/plain")
	assert.NoError(t, err)
	sp2, err := accessor.Put(ctx, "uploads/2026/07/b.txt", strings.NewReader("bbb"), 3, "text/plain")
	assert.NoError(t, err)

	f1 := &model.File{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, Name: "a.txt", OriginalName: "a.txt", StoragePath: sp1, Size: 3, MimeType: "text/plain"}
	f2 := &model.File{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, Name: "b.txt", OriginalName: "b.txt", StoragePath: sp2, Size: 3, MimeType: "text/plain"}
	assert.NoError(t, db.Create(f1).Error)
	assert.NoError(t, db.Create(f2).Error)

	exists := func(sp string) bool {
		rc, err := accessor.GetByPath(ctx, sp)
		if err != nil {
			return false
		}
		rc.Close()
		return true
	}
	assert.True(t, exists(sp1), "删除前文件1应存在")
	assert.True(t, exists(sp2), "删除前文件2应存在")

	n, err := svc.BatchDelete(user.ID, []uint{f1.ID, f2.ID})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), n)

	assert.False(t, exists(sp1), "批量删除后文件1物理文件应被删除")
	assert.False(t, exists(sp2), "批量删除后文件2物理文件应被删除")

	var count int64
	db.Model(&model.File{}).Where("id IN ?", []uint{f1.ID, f2.ID}).Count(&count)
	assert.Equal(t, int64(0), count, "DB 记录应被删除")
}

// TestFileService_BatchDelete_OnlyAffectsOwnedFiles 验证批量删除只删本人文件，
// 他人文件不受影响（物理文件与 DB 记录均保留）。
func TestFileService_BatchDelete_OnlyAffectsOwnedFiles(t *testing.T) {
	db := setupFileServiceTestDB(t)
	me := &model.User{Username: "me", PasswordHash: "h"}
	other := &model.User{Username: "other", PasswordHash: "h"}
	assert.NoError(t, db.Create(me).Error)
	assert.NoError(t, db.Create(other).Error)

	accessor := newTestAccessor(t)
	svc := NewFileService(db)
	svc.SetStorageAccessor(accessor)

	ctx := context.Background()
	// other 的文件，其 ID 恰好在删除请求里（越权尝试）
	spOther, _ := accessor.Put(ctx, "uploads/2026/07/other.txt", strings.NewReader("ooo"), 3, "text/plain")
	otherFile := &model.File{UserID: other.ID, ScopeType: "user", ScopeID: other.ID, Name: "other.txt", OriginalName: "other.txt", StoragePath: spOther, Size: 3, MimeType: "text/plain"}
	assert.NoError(t, db.Create(otherFile).Error)

	// me 请求删除 other 的文件 ID
	n, err := svc.BatchDelete(me.ID, []uint{otherFile.ID})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n, "不应删他人文件")

	// other 的物理文件与 DB 记录应保留
	rc, err := accessor.GetByPath(ctx, spOther)
	assert.NoError(t, err, "他人物理文件应保留")
	if rc != nil {
		rc.Close()
	}
	var count int64
	db.Model(&model.File{}).Where("id = ?", otherFile.ID).Count(&count)
	assert.Equal(t, int64(1), count, "他人 DB 记录应保留")
}

func TestFileServiceDeleteFileDeletesUnreferencedStorage(t *testing.T) {
	db := setupFileServiceTestDB(t)
	user := &model.User{Username: "delete-one", PasswordHash: "h"}
	assert.NoError(t, db.Create(user).Error)

	base := newTestAccessor(t)
	accessor := &deleteCountingAccessor{StorageAccessor: base}
	svc := NewFileService(db)
	svc.SetStorageAccessor(accessor)

	ctx := context.Background()
	storagePath, err := base.Put(ctx, "uploads/delete-one.txt", strings.NewReader("one"), 3, "text/plain")
	assert.NoError(t, err)
	file := &model.File{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, Name: "delete-one.txt", OriginalName: "delete-one.txt", StoragePath: storagePath, Size: 3, MimeType: "text/plain"}
	assert.NoError(t, db.Create(file).Error)

	assert.NoError(t, svc.DeleteFile(user.ID, file.ID))
	assert.Equal(t, 1, accessor.DeleteCount())
	_, err = accessor.GetByPath(ctx, storagePath)
	assert.Error(t, err)
}

func TestFileServiceDeleteFolderFilesCleansOnlyUnreferencedStorage(t *testing.T) {
	db := setupFileServiceTestDB(t)
	user := &model.User{Username: "folder-delete", PasswordHash: "h"}
	assert.NoError(t, db.Create(user).Error)
	folder := &model.Folder{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, Name: "folder"}
	assert.NoError(t, db.Create(folder).Error)

	base := newTestAccessor(t)
	accessor := &deleteCountingAccessor{StorageAccessor: base}
	svc := NewFileService(db)
	svc.SetStorageAccessor(accessor)
	ctx := context.Background()

	orphanPath, err := base.Put(ctx, "uploads/folder-orphan.txt", strings.NewReader("orphan"), 6, "text/plain")
	assert.NoError(t, err)
	sharedPath, err := base.Put(ctx, "uploads/folder-shared.txt", strings.NewReader("shared"), 6, "text/plain")
	assert.NoError(t, err)
	orphan := &model.File{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, FolderID: &folder.ID, Name: "orphan.txt", StoragePath: orphanPath}
	shared := &model.File{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, FolderID: &folder.ID, Name: "shared.txt", StoragePath: sharedPath}
	activeReference := &model.File{UserID: user.ID, ScopeType: "group", ScopeID: 99, Name: "shared-ref.txt", StoragePath: sharedPath, Source: "shared_reference"}
	assert.NoError(t, db.Create(orphan).Error)
	assert.NoError(t, db.Create(shared).Error)
	assert.NoError(t, db.Create(activeReference).Error)

	assert.NoError(t, svc.DeleteFolderFiles(user.ID, folder.ID))
	_, err = accessor.GetByPath(ctx, orphanPath)
	assert.Error(t, err)
	reader, err := accessor.GetByPath(ctx, sharedPath)
	assert.NoError(t, err)
	assert.NoError(t, reader.Close())
	assert.Equal(t, 1, accessor.DeleteCount())
}

func TestFileServiceDeleteFolderRecursiveCleansChildStorage(t *testing.T) {
	db := setupFileServiceTestDB(t)
	user := &model.User{Username: "folder-recursive", PasswordHash: "h"}
	assert.NoError(t, db.Create(user).Error)
	parent := &model.Folder{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, Name: "parent"}
	assert.NoError(t, db.Create(parent).Error)
	child := &model.Folder{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, Name: "child", ParentID: &parent.ID}
	assert.NoError(t, db.Create(child).Error)

	base := newTestAccessor(t)
	accessor := &deleteCountingAccessor{StorageAccessor: base}
	svc := NewFileService(db)
	svc.SetStorageAccessor(accessor)
	ctx := context.Background()
	storagePath, err := base.Put(ctx, "uploads/recursive.txt", strings.NewReader("child"), 5, "text/plain")
	assert.NoError(t, err)
	file := &model.File{UserID: user.ID, ScopeType: "user", ScopeID: user.ID, FolderID: &child.ID, Name: "child.txt", StoragePath: storagePath}
	assert.NoError(t, db.Create(file).Error)

	assert.NoError(t, svc.DeleteFolderRecursive(user.ID, parent.ID))
	_, err = accessor.GetByPath(ctx, storagePath)
	assert.Error(t, err)
	assert.Equal(t, 1, accessor.DeleteCount())
}
