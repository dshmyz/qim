package provider

import (
	"context"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	qsqlite "github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupLocalProviderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(qsqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	database.DB = db
	return db
}

func TestLocalProviderAllowsReservedPasswordForLocalUser(t *testing.T) {
	db := setupLocalProviderTestDB(t)
	password := "oauth_default_pass"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &model.User{Username: "local-reserved", PasswordHash: string(hash)}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := NewLocalProvider(true, 0).Authenticate(context.Background(), &Credentials{
		Username: user.Username,
		Password: password,
	})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected local user to authenticate with its valid password, got: %s", result.Message)
	}
}

func TestLocalProviderRejectsExternalUserWithoutLocalPassword(t *testing.T) {
	db := setupLocalProviderTestDB(t)
	user := &model.User{
		Username:     "oauth-user",
		PasswordHash: "!",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := NewLocalProvider(true, 0).Authenticate(context.Background(), &Credentials{
		Username: user.Username,
		Password: "any-password",
	})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if result.Success {
		t.Fatal("expected external auth user to be rejected by local authentication")
	}
}
