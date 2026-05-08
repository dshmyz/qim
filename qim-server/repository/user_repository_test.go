package repository

import (
	"context"
	"testing"

	"qim-server/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&model.User{}, &model.UserRole{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}

	err := repo.Create(ctx, user)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}
	repo.Create(ctx, user)

	found, err := repo.FindByUsername(ctx, "testuser")
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)

	_, err = repo.FindByUsername(ctx, "notexist")
	assert.Error(t, err)
}

func TestUserRepository_Search(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	users := []*model.User{
		{Username: "zhangsan", Nickname: "张三", PasswordHash: "hash"},
		{Username: "lisi", Nickname: "李四", PasswordHash: "hash"},
		{Username: "wangwu", Nickname: "王五", PasswordHash: "hash"},
	}
	for _, u := range users {
		repo.Create(ctx, u)
	}

	results, err := repo.Search(ctx, "张", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "张三", results[0].Nickname)
}

func TestUserRepository_UpdateStatus(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Status:       "offline",
	}
	repo.Create(ctx, user)

	err := repo.UpdateStatus(ctx, user.ID, "online")
	assert.NoError(t, err)

	found, _ := repo.FindByID(ctx, user.ID)
	assert.Equal(t, "online", found.Status)
}

func TestUserRepository_FindByPhone(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Phone:        "13800138000",
	}
	repo.Create(ctx, user)

	found, err := repo.FindByPhone(ctx, "13800138000")
	assert.NoError(t, err)
	assert.Equal(t, user.Phone, found.Phone)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Email:        "test@example.com",
	}
	repo.Create(ctx, user)

	found, err := repo.FindByEmail(ctx, "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, user.Email, found.Email)
}
