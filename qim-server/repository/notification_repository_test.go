package repository

import (
	"context"
	"testing"

	"qim-server/model"

	"github.com/stretchr/testify/assert"
	"qim-server/pkg/sqlite"
	"gorm.io/gorm"
)

func setupNotifTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&model.User{}, &model.Notification{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestNotificationRepository_Create(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif := &model.Notification{
		UserID:  user.ID,
		Type:    "system",
		Title:   "Test Notification",
		Content: "This is a test notification",
	}

	err := repo.Create(ctx, notif)
	assert.NoError(t, err)
	assert.NotZero(t, notif.ID)
}

func TestNotificationRepository_FindByUserID(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif1 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 1", Content: "Content 1"}
	notif2 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 2", Content: "Content 2"}
	db.Create(notif1)
	db.Create(notif2)

	results, err := repo.FindByUserID(ctx, user.ID, false)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	results, err = repo.FindByUserID(ctx, user.ID, true)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestNotificationRepository_FindByUserID_UnreadOnly(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif1 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 1", Content: "Content 1", Read: false}
	notif2 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 2", Content: "Content 2", Read: true}
	db.Create(notif1)
	db.Create(notif2)

	results, err := repo.FindByUserID(ctx, user.ID, true)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Notif 1", results[0].Title)
}

func TestNotificationRepository_MarkAsRead(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif := &model.Notification{UserID: user.ID, Type: "system", Title: "Test", Content: "Content", Read: false}
	db.Create(notif)

	err := repo.MarkAsRead(ctx, notif.ID)
	assert.NoError(t, err)

	var updated model.Notification
	db.First(&updated, notif.ID)
	assert.True(t, updated.Read)
	assert.NotNil(t, updated.ReadAt)
}

func TestNotificationRepository_MarkAllAsRead(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif1 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 1", Content: "Content 1", Read: false}
	notif2 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 2", Content: "Content 2", Read: false}
	db.Create(notif1)
	db.Create(notif2)

	err := repo.MarkAllAsRead(ctx, user.ID)
	assert.NoError(t, err)

	var updated []model.Notification
	db.Where("user_id = ?", user.ID).Find(&updated)
	for _, n := range updated {
		assert.True(t, n.Read)
	}
}

func TestNotificationRepository_CountUnread(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif1 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 1", Content: "Content 1", Read: false}
	notif2 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 2", Content: "Content 2", Read: false}
	notif3 := &model.Notification{UserID: user.ID, Type: "system", Title: "Notif 3", Content: "Content 3", Read: true}
	db.Create(notif1)
	db.Create(notif2)
	db.Create(notif3)

	count, err := repo.CountUnread(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestNotificationRepository_FindByID(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif := &model.Notification{UserID: user.ID, Type: "system", Title: "Test", Content: "Content"}
	db.Create(notif)

	found, err := repo.FindByID(ctx, notif.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Test", found.Title)

	_, err = repo.FindByID(ctx, 99999)
	assert.Error(t, err)
}

func TestNotificationRepository_Delete(t *testing.T) {
	db := setupNotifTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	notif := &model.Notification{UserID: user.ID, Type: "system", Title: "Test", Content: "Content"}
	db.Create(notif)

	err := repo.Delete(ctx, notif.ID)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, notif.ID)
	assert.Error(t, err)
}
