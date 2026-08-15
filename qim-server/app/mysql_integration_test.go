package app_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/dshmyz/qim/qim-server/app"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// mysqlConfig 读取 MySQL 集成测试连接配置。
// 环境变量可覆盖，默认指向本机 docker 测试库（qim-mysql 容器）。
func mysqlConfig() config.DatabaseConfig {
	cfg := config.DatabaseConfig{
		Type:         "mysql",
		Host:         "127.0.0.1",
		Port:         3306,
		Username:     "root",
		Password:     "test",
		Database:     "qim_server",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		MaxLifetime:  3600,
	}
	if v := os.Getenv("QIM_TEST_MYSQL_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("QIM_TEST_MYSQL_USER"); v != "" {
		cfg.Username = v
	}
	if v := os.Getenv("QIM_TEST_MYSQL_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("QIM_TEST_MYSQL_DB"); v != "" {
		cfg.Database = v
	}
	return cfg
}

func mysqlDSN(cfg config.DatabaseConfig) string {
	return cfg.Username + ":" + cfg.Password + "@tcp(" + cfg.Host + ":" + itoa(cfg.Port) + ")/" + cfg.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func itoa(n int) string {
	if n == 0 {
		return "3306"
	}
	return fmt.Sprint(n)
}

var (
	ensureMigratedOnce sync.Once
	ensureMigratedErr  error
)

// mysqlTestDB 初始化 MySQL 连接。MySQL 不可达时跳过测试，不影响纯 SQLite 的本地测试。
func mysqlTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg := mysqlConfig()
	probe, err := gorm.Open(mysql.Open(mysqlDSN(cfg)), &gorm.Config{})
	if err != nil {
		t.Skipf("MySQL 不可达，跳过集成测试: %v", err)
	}
	sqlDB, _ := probe.DB()
	require.NoError(t, sqlDB.Ping())
	sqlDB.Close()

	db := database.Init(&config.Config{Database: cfg})
	require.NotNil(t, db)

	ensureMigratedOnce.Do(func() {
		ensureMigratedErr = app.MigrateDB(db)
	})
	require.NoError(t, ensureMigratedErr, "MigrateDB 在 MySQL 上失败")

	cleanMySQLTables(t, db, "message_read_receipts", "messages", "conversation_members", "conversations", "users")
	return db
}

func cleanMySQLTables(t *testing.T, db *gorm.DB, tables ...string) {
	t.Helper()
	for _, name := range tables {
		require.NoError(t, db.Exec("DELETE FROM `"+name+"`").Error, "清空表 %s 失败", name)
	}
}

// TestMySQL_MigrateDB_KeyTablesAndIndexes 验证迁移在真实 MySQL 上建表 + 建索引。
// 重点：FULLTEXT 索引（MATCH...AGAINST 依赖）、已读回执唯一索引（INSERT IGNORE 幂等依赖）、
// 消息分页复合索引。
func TestMySQL_MigrateDB_KeyTablesAndIndexes(t *testing.T) {
	db := mysqlTestDB(t)

	var cnt int64
	for _, table := range []string{"users", "conversations", "conversation_members", "messages", "message_read_receipts"} {
		db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", table).Scan(&cnt)
		require.Equal(t, int64(1), cnt, "表 %s 在 MySQL 上缺失", table)
	}

	indexExists := func(table, index string) bool {
		var n int64
		db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?", table, index).Scan(&n)
		return n > 0
	}

	require.True(t, indexExists("messages", "idx_messages_conversation_created_at"), "messages 分页复合索引缺失")
	require.True(t, indexExists("message_read_receipts", "idx_message_user_receipt"), "已读回执唯一索引缺失")
	require.True(t, indexExists("messages", "ft_messages_content"), "messages FULLTEXT 索引缺失")
	require.True(t, database.D.SupportsFulltext(), "MySQL 应支持 FULLTEXT")
}

// TestMySQL_MarkAsRead_InsertsReceipts 覆盖读回执的 MySQL INSERT IGNORE 方言分支：
// 写入 + 幂等 + 未读清零。
func TestMySQL_MarkAsRead_InsertsReceipts(t *testing.T) {
	db := mysqlTestDB(t)

	u1 := &model.User{Username: "u1_read", PasswordHash: "x"}
	u2 := &model.User{Username: "u2_read", PasswordHash: "x"}
	require.NoError(t, db.Create(u1).Error)
	require.NoError(t, db.Create(u2).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u1.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u2.ID, Role: "member"}).Error)

	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&model.Message{
			ConversationID: conv.ID,
			SenderID:       u1.ID,
			Type:           "text",
			Content:        "read receipt test msg",
		}).Error)
	}

	msgSvc := service.NewMessageService(db, nil, nil)
	require.NoError(t, msgSvc.MarkAsRead(conv.ID, u2.ID))

	var receipts int64
	require.NoError(t, db.Model(&model.MessageReadReceipt{}).Where("user_id = ?", u2.ID).Count(&receipts).Error)
	require.Equal(t, int64(3), receipts, "MySQL 读回执应写入 3 条")

	var member model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", conv.ID, u2.ID).First(&member).Error)
	require.Zero(t, member.UnreadCount, "未读计数应清零")

	// 幂等：重复已读不产生重复回执（INSERT IGNORE + 唯一索引）
	require.NoError(t, msgSvc.MarkAsRead(conv.ID, u2.ID))
	require.NoError(t, db.Model(&model.MessageReadReceipt{}).Where("user_id = ?", u2.ID).Count(&receipts).Error)
	require.Equal(t, int64(3), receipts, "重复已读不应产生重复回执")
}

// TestMySQL_SearchMessagesByFullText 覆盖全文搜索的 MySQL MATCH...AGAINST 分支，
// 确保走 FULLTEXT 而非 LIKE 降级。
func TestMySQL_SearchMessagesByFullText(t *testing.T) {
	db := mysqlTestDB(t)

	u := &model.User{Username: "u_search", PasswordHash: "x"}
	require.NoError(t, db.Create(u).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u.ID, Role: "member"}).Error)

	require.NoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: u.ID, Type: "text", Content: "hello golang gorm search"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: u.ID, Type: "text", Content: "unrelated content only"}).Error)

	msgSvc := service.NewMessageService(db, nil, nil)
	res, err := msgSvc.SearchMessagesByFullText(u.ID, "golang", nil, 20, 0)
	require.NoError(t, err, "MySQL MATCH...AGAINST 查询失败")
	require.Len(t, res, 1, "应命中 1 条含 golang 的消息")
	require.Contains(t, res[0].Content, "golang")
}

// TestMySQL_SendMessage_BasicFlow 覆盖发消息热路径在 MySQL 上的基本流程：
// 成员校验 → 落库 → 更新会话摘要 → 未读计数。
func TestMySQL_SendMessage_BasicFlow(t *testing.T) {
	db := mysqlTestDB(t)

	u1 := &model.User{Username: "u1_send", PasswordHash: "x"}
	u2 := &model.User{Username: "u2_send", PasswordHash: "x"}
	require.NoError(t, db.Create(u1).Error)
	require.NoError(t, db.Create(u2).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u1.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u2.ID, Role: "member"}).Error)

	msgSvc := service.NewMessageService(db, nil, nil)
	msg, err := msgSvc.SendMessage(conv.ID, u1.ID, "text", "hello from mysql test", nil)
	require.NoError(t, err, "MySQL 上发消息失败")
	require.NotZero(t, msg.ID)

	var convAfter model.Conversation
	require.NoError(t, db.First(&convAfter, conv.ID).Error)
	require.NotNil(t, convAfter.LastMessageID)
	require.Equal(t, msg.ID, *convAfter.LastMessageID, "会话摘要 last_message_id 应更新")

	var member model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", conv.ID, u2.ID).First(&member).Error)
	require.Equal(t, 1, member.UnreadCount, "对方未读计数应 +1")
}
