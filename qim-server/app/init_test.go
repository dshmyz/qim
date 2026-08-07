package app

import (
	"path/filepath"
	"testing"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, MigrateDB(db))

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
	require.NoError(t, MigrateDB(db))

	if !hasTable(t, db, "avatar_tool_bindings") {
		t.Error("AvatarToolBinding 表缺失")
	}
	if !hasTable(t, db, "document_process_statuses") {
		t.Error("DocumentProcessStatus 表缺失")
	}
}

func TestMigrateDB_CreatesMessagesOriginColumn(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, MigrateDB(db))

	if !hasSQLiteColumn(t, db, "messages", "origin") {
		t.Fatal("MigrateDB 后 messages.origin 字段缺失")
	}
}

func TestMigrateDB_AddsAccountStatusToExistingUsers(t *testing.T) {
	db := newTestDB(t)
	if err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username VARCHAR(50) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			status VARCHAR(20) DEFAULT 'offline',
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("创建旧版 users 表失败: %v", err)
	}

	require.NoError(t, MigrateDB(db))

	if !hasSQLiteColumn(t, db, "users", "account_status") {
		t.Fatal("MigrateDB 后 users.account_status 字段缺失")
	}
}

// AIProvider 显式表名：新装库应创建 ai_providers，而非 GORM 默认误拆的 a_iprovider(s)。
func TestMigrateDB_AICreatesTableName(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, MigrateDB(db))

	require.True(t, hasTable(t, db, "ai_providers"), "AIProvider 应建在 ai_providers 表")
	require.False(t, hasTable(t, db, "a_iprovider"), "不应存在 a_iprovider 误拆表")
	require.False(t, hasTable(t, db, "a_iproviders"), "不应存在 a_iproviders 误拆表")
}

// BotConversation 收敛：历史库带 user_id 列 + 索引 + 存量数据时，迁移应
// 删 user_id 列与索引，且 (bot_id, conversation_id) 数据完整保留。
// AutoMigrate 不删列，靠 migrateCompatibilityColumns 手动 DropIndex + DropColumn。
func TestMigrateDB_DropsBotConversationUserID(t *testing.T) {
	db := newTestDB(t)
	// 先建一张带 user_id 的旧版 bot_conversations，模拟收敛前的历史库
	require.NoError(t, db.Exec(`
		CREATE TABLE bot_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id integer NOT NULL,
			user_id integer NOT NULL,
			conversation_id integer NOT NULL,
			created_at datetime
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_bot_conversations_user_id ON bot_conversations(user_id)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_bot_conversations_bot_id ON bot_conversations(bot_id)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_bot_conversations_conversation_id ON bot_conversations(conversation_id)`).Error)

	// 存量数据：3 组 bot↔会话（user_id 各不相同，收敛后不再区分）
	for i := 1; i <= 3; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO bot_conversations (bot_id, user_id, conversation_id) VALUES (?, ?, ?)`,
			i, 100+i, i*7,
		).Error)
	}

	require.NoError(t, MigrateDB(db))

	// user_id 列与索引已删除
	require.False(t, hasSQLiteColumn(t, db, "bot_conversations", "user_id"), "user_id 列应被删除")
	require.False(t, hasTable(t, db, "idx_bot_conversations_user_id"), "user_id 索引应被删除")
	// bot_id/conversation_id 列与数据保留
	require.True(t, hasSQLiteColumn(t, db, "bot_conversations", "bot_id"), "bot_id 列应保留")
	require.True(t, hasSQLiteColumn(t, db, "bot_conversations", "conversation_id"), "conversation_id 列应保留")

	var count int64
	require.NoError(t, db.Table("bot_conversations").Count(&count).Error)
	require.Equal(t, int64(3), count, "存量数据应完整保留")
}

// 历史误拆表 a_iproviders 上已有数据时，迁移应改名到 ai_providers 且数据保留。
func TestMigrateDB_RenamesLegacyAITable(t *testing.T) {
	db := newTestDB(t)

	if err := db.Exec(`
		CREATE TABLE a_iproviders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100) NOT NULL,
			provider VARCHAR(50) NOT NULL,
			api_type VARCHAR(20) NOT NULL,
			endpoint VARCHAR(500),
			api_key VARCHAR(500),
			models TEXT,
			enabled BOOLEAN DEFAULT 1,
			status VARCHAR(20) DEFAULT 'connected',
			priority INTEGER DEFAULT 0,
			config TEXT,
			last_test_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("创建旧版 a_iproviders 表失败: %v", err)
	}
	require.NoError(t, db.Exec(
		`INSERT INTO a_iproviders (name, provider, api_type, models) VALUES (?, ?, ?, ?)`,
		"默认OpenAI", "openai", "openai", `["gpt-4o"]`,
	).Error)

	require.NoError(t, MigrateDB(db))

	require.True(t, hasTable(t, db, "ai_providers"), "旧表应改名为 ai_providers")
	require.False(t, hasTable(t, db, "a_iproviders"), "旧表 a_iproviders 应已移除")

	var provider model.AIProvider
	require.NoError(t, db.First(&provider).Error, "供应商数据应保留")
	require.Equal(t, "默认OpenAI", provider.Name)
	require.Equal(t, []string{"gpt-4o"}, []string(provider.Models))
}

// 用户实测 MySQL 出现的是单数 a_iprovider，同样应改名保留数据。
func TestMigrateDB_RenamesLegacyAITableSingular(t *testing.T) {
	db := newTestDB(t)

	require.NoError(t, db.Exec(`
		CREATE TABLE a_iprovider (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100) NOT NULL,
			provider VARCHAR(50) NOT NULL,
			api_type VARCHAR(20) NOT NULL,
			models TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO a_iprovider (name, provider, api_type) VALUES (?, ?, ?)`,
		"Claude测试", "anthropic", "claude",
	).Error)

	require.NoError(t, MigrateDB(db))

	require.True(t, hasTable(t, db, "ai_providers"))
	require.False(t, hasTable(t, db, "a_iprovider"))

	var provider model.AIProvider
	require.NoError(t, db.First(&provider).Error, "供应商数据应保留")
	require.Equal(t, "Claude测试", provider.Name)
}

func TestInitAdminUser_CreatesAdminWithRole(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, MigrateDB(db))
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
