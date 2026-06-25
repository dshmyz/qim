package app

import (
	"path/filepath"
	"testing"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := database.Init(&config.Config{
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Path: filepath.Join(t.TempDir(), "qim-test.db"),
		},
	})
	SetDB(db)
	return db
}

func hasTable(t *testing.T, db *gorm.DB, name string) bool {
	t.Helper()
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count)
	return count > 0
}

func hasSQLiteColumn(t *testing.T, db *gorm.DB, table, column string) bool {
	t.Helper()
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&count)
	return count > 0
}

func TestMigrateDB_CreatesCoreTables(t *testing.T) {
	db := newTestDB(t)
	MigrateDB(db)

	core := []string{
		"users", "conversations", "messages", "user_roles",
		"groups", "notifications", "channels",
	}
	for _, name := range core {
		if !hasTable(t, db, name) {
			t.Errorf("MigrateDB 后核心表 %s 缺失", name)
		}
	}
}

func TestMigrateDB_CreatesMissingModels(t *testing.T) {
	db := newTestDB(t)
	MigrateDB(db)

	if !hasTable(t, db, "avatar_tool_bindings") {
		t.Error("AvatarToolBinding 表缺失")
	}
	if !hasTable(t, db, "document_process_statuses") {
		t.Error("DocumentProcessStatus 表缺失")
	}
}

func TestMigrateDB_CreatesMessagesOriginColumn(t *testing.T) {
	db := newTestDB(t)
	MigrateDB(db)

	if !hasSQLiteColumn(t, db, "messages", "origin") {
		t.Fatal("MigrateDB 后 messages.origin 字段缺失")
	}
}

func TestInitAdminUser_CreatesAdminWithRole(t *testing.T) {
	db := newTestDB(t)
	MigrateDB(db)
	initAdminUser()

	var user model.User
	if err := db.Where("type = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("管理员用户未创建: %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("管理员用户名预期 admin，得到 %s", user.Username)
	}

	var role model.UserRole
	if err := db.Where("user_id = ? AND role = ?", user.ID, "system_admin").First(&role).Error; err != nil {
		t.Fatalf("管理员 system_admin 角色缺失: %v", err)
	}
}
