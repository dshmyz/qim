package database

import (
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"gorm.io/gorm"
)

// Dialect 封装数据库方言差异，将 MySQL / SQLite 的语法差异内聚到一处
type Dialect interface {
	// Type 返回数据库类型标识："mysql" 或 "sqlite"
	Type() string

	// TableExists 检查表是否存在
	TableExists(db *gorm.DB, tableName string) bool

	// CreateIndexSQL 生成创建索引的 SQL（MySQL 不使用 IF NOT EXISTS，SQLite 使用）
	CreateIndexSQL(indexName, table string, columns []string) string

	// SupportsFulltext 返回运行时探测到的全文索引能力
	SupportsFulltext() bool

	// SupportsFTS5 返回 SQLite FTS5 是否可用（MySQL 始终返回 false）
	SupportsFTS5() bool

	// CreateFulltextIndex 创建全文索引
	CreateFulltextIndex(db *gorm.DB, table, indexName string, columns []string)

	// HasFulltextIndex 检查全文索引是否存在
	HasFulltextIndex(db *gorm.DB, table, indexName string) bool

	// InitFulltext 初始化全文搜索索引（如 MySQL 的 FULLTEXT 或 SQLite 的 FTS5 虚拟表）
	InitFulltext(db *gorm.DB)
}

// D 当前运行时的方言实例，在 Init() 中初始化
var D Dialect

// NewSQLiteDialect 返回 SQLite 方言实例（供测试及外部调用）
func NewSQLiteDialect() Dialect {
	return &sqliteDialect{}
}

// NewMySQLDialect 返回 MySQL 方言实例（供测试及外部调用）
func NewMySQLDialect() Dialect {
	return &mysqlDialect{}
}

// mysqlDialect MySQL / TiDB / OceanBase 等兼容 MySQL 协议的方言
type mysqlDialect struct {
	supportsFulltext bool
}

func (d *mysqlDialect) Type() string { return "mysql" }

func (d *mysqlDialect) TableExists(db *gorm.DB, tableName string) bool {
	var count int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = ?`, tableName).Scan(&count)
	return count > 0
}

func (d *mysqlDialect) CreateIndexSQL(indexName, table string, columns []string) string {
	cols := make([]string, len(columns))
	for i, c := range columns {
		cols[i] = "`" + c + "`"
	}
	return fmt.Sprintf("CREATE INDEX `%s` ON `%s`(%s)", indexName, table, strings.Join(cols, ", "))
}

func (d *mysqlDialect) SupportsFulltext() bool { return d.supportsFulltext }

func (d *mysqlDialect) SupportsFTS5() bool { return false }

func (d *mysqlDialect) CreateFulltextIndex(db *gorm.DB, table, indexName string, columns []string) {
	if d.HasFulltextIndex(db, table, indexName) {
		return
	}
	cols := strings.Join(columns, ", ")
	db.Exec(fmt.Sprintf("ALTER TABLE %s ADD FULLTEXT INDEX %s (%s)", table, indexName, cols))
}

func (d *mysqlDialect) HasFulltextIndex(db *gorm.DB, table, indexName string) bool {
	var count int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = ?
		AND INDEX_NAME = ?`, table, indexName).Scan(&count)
	return count > 0
}

func (d *mysqlDialect) InitFulltext(db *gorm.DB) {
	d.supportsFulltext = false

	if err := db.Exec("CREATE TEMPORARY TABLE IF NOT EXISTS __probe_ft (content TEXT)").Error; err != nil {
		logger.WithModule("Database").Warn("能力探测失败：无法创建临时表", "error", err)
		return
	}
	defer db.Exec("DROP TEMPORARY TABLE IF EXISTS __probe_ft")

	if err := db.Exec("ALTER TABLE __probe_ft ADD FULLTEXT INDEX __probe_ft_idx (content)").Error; err != nil {
		logger.WithModule("Database").Info("FULLTEXT 索引不可用，将使用 LIKE 搜索", "error", err)
		return
	}
	db.Exec("DROP INDEX __probe_ft_idx ON __probe_ft")

	d.supportsFulltext = true
	logger.WithModule("Database").Info("数据库支持 FULLTEXT 全文索引")
}

// sqliteDialect SQLite 方言
type sqliteDialect struct {
	supportsFTS5 bool
}

func (d *sqliteDialect) Type() string { return "sqlite" }

func (d *sqliteDialect) TableExists(db *gorm.DB, tableName string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	return count > 0
}

func (d *sqliteDialect) CreateIndexSQL(indexName, table string, columns []string) string {
	cols := make([]string, len(columns))
	for i, c := range columns {
		cols[i] = "`" + c + "`"
	}
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS `%s` ON `%s`(%s)", indexName, table, strings.Join(cols, ", "))
}

func (d *sqliteDialect) SupportsFulltext() bool { return false }

func (d *sqliteDialect) SupportsFTS5() bool { return d.supportsFTS5 }

func (d *sqliteDialect) CreateFulltextIndex(_ *gorm.DB, _, _ string, _ []string) {
	// SQLite 使用 FTS5 虚拟表，不使用传统索引
}

func (d *sqliteDialect) HasFulltextIndex(_ *gorm.DB, _, _ string) bool {
	return false
}

func (d *sqliteDialect) InitFulltext(db *gorm.DB) {
	// 此处仅探测 FTS5 能力，实际 FTS5 虚拟表的创建在 addIndexes 迁移阶段
	d.supportsFTS5 = d.isFTS5Available(db)
	if !d.supportsFTS5 {
		logger.WithModule("Database").Info("SQLite FTS5 不可用，将使用 LIKE 搜索")
		return
	}
	logger.WithModule("Database").Info("SQLite FTS5 可用")
}

// isFTS5Available 检查 SQLite 是否支持 FTS5
func (d *sqliteDialect) isFTS5Available(db *gorm.DB) bool {
	var result int
	err := db.Raw("SELECT 1 FROM sqlite_master WHERE name = 'fts5' AND type = 'table'").Scan(&result).Error
	if err == nil && result == 1 {
		return true
	}

	err = db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS __fts5_test USING fts5(content)").Error
	if err == nil {
		db.Exec("DROP TABLE __fts5_test")
		return true
	}
	return false
}
