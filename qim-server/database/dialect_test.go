package database

import (
	"strings"
	"testing"
)

func TestDialectCreatesUniqueIndexSQL(t *testing.T) {
	sqliteSQL := NewSQLiteDialect().CreateUniqueIndexSQL("idx_version_platform", "client_versions", []string{"app_type", "version"})
	if !strings.Contains(sqliteSQL, "CREATE UNIQUE INDEX IF NOT EXISTS") {
		t.Fatalf("sqlite unique index SQL lost UNIQUE/IF NOT EXISTS: %s", sqliteSQL)
	}

	mysqlSQL := NewMySQLDialect().CreateUniqueIndexSQL("idx_version_platform", "client_versions", []string{"app_type", "version"})
	if !strings.Contains(mysqlSQL, "CREATE UNIQUE INDEX") || strings.Contains(mysqlSQL, "IF NOT EXISTS") {
		t.Fatalf("mysql unique index SQL is invalid: %s", mysqlSQL)
	}
}

func TestDialectDropsIndexWithTableForMySQLOnly(t *testing.T) {
	sqliteSQL := NewSQLiteDialect().DropIndexSQL("idx_version_platform", "client_versions")
	if sqliteSQL != "DROP INDEX IF EXISTS `idx_version_platform`" {
		t.Fatalf("unexpected sqlite drop index SQL: %s", sqliteSQL)
	}

	mysqlSQL := NewMySQLDialect().DropIndexSQL("idx_version_platform", "client_versions")
	if mysqlSQL != "DROP INDEX `idx_version_platform` ON `client_versions`" {
		t.Fatalf("unexpected mysql drop index SQL: %s", mysqlSQL)
	}
}
