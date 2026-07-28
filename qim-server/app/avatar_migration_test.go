package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigrateDB_BackfillsMissingAvatarColumns reproduces the runtime 500
// "no such column: activate_by_default": a legacy avatar_configs table (created
// by an older schema, so it trips AutoMigrate's "already exists" path and gets
// skipped) must still get newly-added columns back-filled by MigrateDB.
func TestMigrateDB_BackfillsMissingAvatarColumns(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Exec(`
		CREATE TABLE avatar_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT,
			enabled NUMERIC,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE UNIQUE INDEX idx_avatar_configs_user_id ON avatar_configs(user_id);
	`).Error)

	// 旧表确实没有这些后续新增的列
	require.False(t, hasSQLiteColumn(t, db, "avatar_configs", "activate_by_default"))
	require.False(t, hasSQLiteColumn(t, db, "avatar_configs", "knowledge_scope_json"))
	require.False(t, hasSQLiteColumn(t, db, "avatar_configs", "takeover_cooldown"))

	require.NoError(t, MigrateDB(db))

	// 迁移后必须补齐
	require.True(t, hasSQLiteColumn(t, db, "avatar_configs", "activate_by_default"), "activate_by_default 未补齐")
	require.True(t, hasSQLiteColumn(t, db, "avatar_configs", "custom_persona_addon"), "custom_persona_addon 未补齐")
	require.True(t, hasSQLiteColumn(t, db, "avatar_configs", "knowledge_scope_json"), "knowledge_scope_json 未补齐")
	require.True(t, hasSQLiteColumn(t, db, "avatar_configs", "trigger_rules_json"), "trigger_rules_json 未补齐")
	require.True(t, hasSQLiteColumn(t, db, "avatar_configs", "reply_strategy_json"), "reply_strategy_json 未补齐")
	require.True(t, hasSQLiteColumn(t, db, "avatar_configs", "takeover_cooldown"), "takeover_cooldown 未补齐")
	require.True(t, hasSQLiteColumn(t, db, "avatar_configs", "use_system_config"), "use_system_config 未补齐")
}
