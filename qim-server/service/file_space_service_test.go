package service

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupFileSpaceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Group{},
		&model.File{},
		&model.Folder{},
	))
	return db
}

func createFileSpaceUser(t *testing.T, db *gorm.DB, username string) *model.User {
	t.Helper()
	user := &model.User{Username: username, PasswordHash: "hash"}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createFileSpaceGroup(t *testing.T, db *gorm.DB, ownerID uint) (*model.Group, *model.ConversationMember) {
	t.Helper()
	conversation := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conversation).Error)
	group := &model.Group{ConversationID: conversation.ID, GroupType: "group", Name: "测试群", CreatorID: ownerID}
	require.NoError(t, db.Create(group).Error)
	owner := &model.ConversationMember{ConversationID: conversation.ID, UserID: ownerID, Role: "owner"}
	require.NoError(t, db.Create(owner).Error)
	return group, owner
}

func createUserFile(t *testing.T, db *gorm.DB, userID uint, name string) *model.File {
	t.Helper()
	file := &model.File{
		UserID:       userID,
		ScopeType:    "user",
		ScopeID:      userID,
		Name:         name,
		OriginalName: name,
		Size:         42,
		MimeType:     "application/pdf",
		StoragePath:  "uploads/" + name,
		Checksum:     "checksum-" + name,
		Source:       "upload",
	}
	require.NoError(t, db.Create(file).Error)
	return file
}

func TestFileSpaceShareReferenceDoesNotDeleteOriginal(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	group, _ := createFileSpaceGroup(t, db, admin.ID)
	spaces := NewFileSpaceService(db)

	source := createUserFile(t, db, admin.ID, "chat.pdf")
	shared, err := spaces.ShareReference(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, source.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "shared_reference", shared.Source)
	require.Equal(t, strconv.FormatUint(uint64(source.ID), 10), shared.SourceID)
	require.Equal(t, source.StoragePath, shared.StoragePath)
	require.Equal(t, source.Checksum, shared.Checksum)

	require.NoError(t, spaces.Delete(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, shared.ID))
	require.NoError(t, db.First(&source, source.ID).Error)
}

func TestFileSpaceRejectsMemberFolderManagement(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	member := createFileSpaceUser(t, db, "member")
	group, _ := createFileSpaceGroup(t, db, admin.ID)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ConversationID, UserID: member.ID, Role: "member"}).Error)
	spaces := NewFileSpaceService(db)

	_, err := spaces.CreateFolder(ctx, member.ID, FileSpace{Type: "group", ID: group.ID}, "会议纪要", nil)
	require.ErrorIs(t, err, ErrFileSpaceForbidden)
}

func TestFileSpaceRejectsManagementFromAnotherGroup(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	owner := createFileSpaceUser(t, db, "owner")
	otherOwner := createFileSpaceUser(t, db, "other-owner")
	group, _ := createFileSpaceGroup(t, db, owner.ID)
	otherGroup, _ := createFileSpaceGroup(t, db, otherOwner.ID)
	spaces := NewFileSpaceService(db)

	_, err := spaces.CreateFolder(ctx, otherOwner.ID, FileSpace{Type: "group", ID: group.ID}, "越权文件夹", nil)
	require.ErrorIs(t, err, ErrFileSpaceForbidden)
	_, err = spaces.List(ctx, otherOwner.ID, FileSpace{Type: "group", ID: group.ID}, FileSpaceQuery{})
	require.ErrorIs(t, err, ErrFileSpaceForbidden)

	// A group owner may only manage their own group space.
	_, err = spaces.CreateFolder(ctx, owner.ID, FileSpace{Type: "group", ID: otherGroup.ID}, "越权文件夹", nil)
	require.ErrorIs(t, err, ErrFileSpaceForbidden)
}

func TestFileSpaceRejectsFolderFromAnotherSpace(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	group, _ := createFileSpaceGroup(t, db, admin.ID)
	spaces := NewFileSpaceService(db)
	personalFolder, err := spaces.CreateFolder(ctx, admin.ID, FileSpace{Type: "user", ID: admin.ID}, "私人", nil)
	require.NoError(t, err)

	_, err = spaces.CreateFolder(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, "错误父目录", &personalFolder.ID)
	require.ErrorIs(t, err, ErrFileSpaceForbidden)

	file := createUserFile(t, db, admin.ID, "chat.pdf")
	err = spaces.Move(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, []uint{file.ID}, nil)
	require.ErrorIs(t, err, ErrFileSpaceForbidden)
}

func TestFileSpaceRejectsCyclicFolderParent(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	spaces := NewFileSpaceService(db)
	space := FileSpace{Type: "user", ID: admin.ID}
	first, err := spaces.CreateFolder(ctx, admin.ID, space, "一", nil)
	require.NoError(t, err)
	second, err := spaces.CreateFolder(ctx, admin.ID, space, "二", &first.ID)
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.Folder{}).Where("id = ?", first.ID).Update("parent_id", second.ID).Error)

	_, err = spaces.CreateFolder(ctx, admin.ID, space, "三", &first.ID)
	require.ErrorIs(t, err, ErrFileSpaceInvalid)
}

func TestFileSpaceAllowsNestedFolderParent(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	spaces := NewFileSpaceService(db)
	space := FileSpace{Type: "user", ID: admin.ID}
	first, err := spaces.CreateFolder(ctx, admin.ID, space, "一", nil)
	require.NoError(t, err)
	second, err := spaces.CreateFolder(ctx, admin.ID, space, "二", &first.ID)
	require.NoError(t, err)

	third, err := spaces.CreateFolder(ctx, admin.ID, space, "三", &second.ID)
	require.NoError(t, err)
	require.Equal(t, second.ID, *third.ParentID)
}

func TestFileSpaceMemberCanListGroupFiles(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	member := createFileSpaceUser(t, db, "member")
	group, _ := createFileSpaceGroup(t, db, admin.ID)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ConversationID, UserID: member.ID, Role: "member"}).Error)
	spaces := NewFileSpaceService(db)
	source := createUserFile(t, db, admin.ID, "chat.pdf")
	_, err := spaces.ShareReference(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, source.ID, nil)
	require.NoError(t, err)

	items, err := spaces.List(ctx, member.ID, FileSpace{Type: "group", ID: group.ID}, FileSpaceQuery{})
	require.NoError(t, err)
	require.Len(t, items.Files, 1)
}

func TestFileSpaceDeleteKeepsStorageWhileAReferenceIsActive(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	group, _ := createFileSpaceGroup(t, db, admin.ID)
	spaces := NewFileSpaceService(db)
	accessor := newTestAccessor(t)
	spaces.SetStorageAccessor(accessor)
	source := createUserFile(t, db, admin.ID, "chat.pdf")
	storagePath, err := accessor.Put(ctx, "uploads/chat.pdf", bytes.NewReader([]byte("pdf")), 3, "application/pdf")
	require.NoError(t, err)
	require.NoError(t, db.Model(source).Update("storage_path", storagePath).Error)
	source.StoragePath = storagePath

	_, err = spaces.ShareReference(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, source.ID, nil)
	require.NoError(t, err)
	require.NoError(t, spaces.Delete(ctx, admin.ID, FileSpace{Type: "user", ID: admin.ID}, source.ID))

	reader, err := accessor.GetByPath(ctx, storagePath)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
}

func TestFileSpaceDeleteRemovesUnreferencedStorage(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	spaces := NewFileSpaceService(db)
	accessor := newTestAccessor(t)
	spaces.SetStorageAccessor(accessor)
	source := createUserFile(t, db, admin.ID, "orphan.pdf")
	storagePath, err := accessor.Put(ctx, "uploads/orphan.pdf", bytes.NewReader([]byte("pdf")), 3, "application/pdf")
	require.NoError(t, err)
	require.NoError(t, db.Model(source).Update("storage_path", storagePath).Error)

	require.NoError(t, spaces.Delete(ctx, admin.ID, FileSpace{Type: "user", ID: admin.ID}, source.ID))
	_, err = accessor.GetByPath(ctx, storagePath)
	require.Error(t, err)
}

func TestFileServiceLegacyOperationsExcludeGroupScopedRecords(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	admin := createFileSpaceUser(t, db, "admin")
	group, _ := createFileSpaceGroup(t, db, admin.ID)
	service := NewFileService(db)

	personal := createUserFile(t, db, admin.ID, "personal.pdf")
	groupFile := &model.File{
		UserID:       admin.ID,
		ScopeType:    "group",
		ScopeID:      group.ID,
		Name:         "group.pdf",
		OriginalName: "group.pdf",
		MimeType:     "application/pdf",
		StoragePath:  "uploads/group.pdf",
	}
	require.NoError(t, db.Create(groupFile).Error)

	files, total, err := service.GetFiles(admin.ID, 1, 50, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, files, 1)
	require.Equal(t, personal.ID, files[0].ID)

	_, err = service.GetFile(admin.ID, groupFile.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	deleted, err := service.BatchDelete(admin.ID, []uint{groupFile.ID})
	require.NoError(t, err)
	require.Zero(t, deleted)
	require.NoError(t, db.First(&groupFile, groupFile.ID).Error)
}

func TestFileSpaceDeleteFinalReferenceCleansStorageAfterOriginalIsDeleted(t *testing.T) {
	db := setupFileSpaceTestDB(t)
	ctx := context.Background()
	admin := createFileSpaceUser(t, db, "admin")
	group, _ := createFileSpaceGroup(t, db, admin.ID)
	spaces := NewFileSpaceService(db)
	accessor := newTestAccessor(t)
	spaces.SetStorageAccessor(accessor)
	source := createUserFile(t, db, admin.ID, "chat.pdf")
	storagePath, err := accessor.Put(ctx, "uploads/chat.pdf", bytes.NewReader([]byte("pdf")), 3, "application/pdf")
	require.NoError(t, err)
	require.NoError(t, db.Model(source).Update("storage_path", storagePath).Error)
	source.StoragePath = storagePath

	shared, err := spaces.ShareReference(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, source.ID, nil)
	require.NoError(t, err)
	require.NoError(t, spaces.Delete(ctx, admin.ID, FileSpace{Type: "user", ID: admin.ID}, source.ID))
	require.NoError(t, spaces.Delete(ctx, admin.ID, FileSpace{Type: "group", ID: group.ID}, shared.ID))

	_, err = accessor.GetByPath(ctx, storagePath)
	require.Error(t, err)
}
