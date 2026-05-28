package app

import (
	"fmt"
	"os"
	"qim-server/auth"
	"qim-server/config"
	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/logger"
	"qim-server/test"
	"qim-server/ws"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Global database instance
var globalDB *gorm.DB

// SetDB sets the global database instance
func SetDB(db *gorm.DB) {
	globalDB = db
}

// GetDB returns the global database instance
func GetDB() *gorm.DB {
	return globalDB
}

// initSystemUser 初始化系统用户
func initSystemUser() {
	db := database.GetDB()

	if !db.Migrator().HasTable(&model.User{}) {
		logger.WithModule("Init").Error("users 表不存在，跳过系统用户初始化。")
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.User{}).Where("type = ?", "system").Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}

		systemUser := model.User{
			Username: "system",
			Nickname: "系统",
			Avatar:   "/system.png",
			Type:     "system",
		}
		if err := tx.Create(&systemUser).Error; err != nil {
			return err
		}
		logger.WithModule("Init").Info("创建系统用户成功", "id", systemUser.ID)
		return nil
	})

	if err != nil {
		logger.WithModule("Init").Error("创建系统用户失败", "error", err)
	}
}

// initAdminUser 初始化管理员用户
func initAdminUser() {
	db := database.GetDB()

	// 检查 users 表是否存在
	if !db.Migrator().HasTable(&model.User{}) {
		logger.WithModule("Init").Error("users 表不存在，跳过管理员初始化。请检查数据库连接和迁移是否成功。")
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var count int64
		// 检查所有用户（包括已删除的），确保管理员用户存在
		if err := tx.Unscoped().Model(&model.User{}).Where("type = ?", "admin").Count(&count).Error; err != nil {
			return fmt.Errorf("查询管理员用户失败: %w", err)
		}

		if count > 0 {
			// 如果存在管理员用户，检查是否被软删除，如果是则恢复
			var existingAdmin model.User
			if err := tx.Unscoped().Where("type = ?", "admin").First(&existingAdmin).Error; err != nil {
				return err
			}
			if existingAdmin.DeletedAt.Valid {
				// 恢复被软删除的管理员用户
				if err := tx.Unscoped().Model(&existingAdmin).Update("deleted_at", nil).Error; err != nil {
					return err
				}
				logger.WithModule("Init").Info("恢复管理员用户", "id", existingAdmin.ID, "username", existingAdmin.Username)
			}
			return nil
		}

		adminUsername := os.Getenv("QIM_ADMIN_USERNAME")
		if adminUsername == "" {
			adminUsername = "admin"
		}

		adminPassword := os.Getenv("QIM_ADMIN_PASSWORD")
		if adminPassword == "" {
			adminPassword = "admin123"
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}

		adminUser := model.User{
			Username:     adminUsername,
			PasswordHash: string(hashedPassword),
			Nickname:     "管理员",
			Avatar:       "admin.png",
			Type:         "admin",
			Status:       "offline",
		}
		if err := tx.Create(&adminUser).Error; err != nil {
			return err
		}

		adminRole := model.UserRole{
			UserID: adminUser.ID,
			Role:   "system_admin",
		}
		if err := tx.Create(&adminRole).Error; err != nil {
			return err
		}

		logger.WithModule("Init").Info("创建管理员用户成功", "id", adminUser.ID, "username", adminUsername)
		return nil
	})

	if err != nil {
		logger.WithModule("Init").Error("创建管理员用户失败", "error", err)
	}
}

// seedBotTemplates 初始化 Bot 模板（系统助手、AI助手）
func seedBotTemplates(db *gorm.DB) {
	if isMigrationCompleted(db, "seed_bot_templates") {
		return
	}

	if !db.Migrator().HasTable("bots") {
		markMigrationCompleted(db, "seed_bot_templates")
		return
	}

	var count int64
	db.Model(&model.Bot{}).Where("type IN ?", []string{"system", "ai"}).Count(&count)
	if count > 0 {
		markMigrationCompleted(db, "seed_bot_templates")
		return
	}

	systemBot := model.Bot{
		Name:        "系统助手",
		Avatar:      "",
		Description: "提供系统相关的帮助和信息",
		Type:        "system",
		Config:      `{"responses":{"greeting":"你好！我是系统助手，有什么可以帮你的吗？","help":"我可以帮助你了解系统功能，解答常见问题。"}}`,
		IsActive:    true,
	}
	if err := db.Create(&systemBot).Error; err != nil {
		logger.WithModule("Init").Error("创建系统助手 Bot 失败", "error", err)
	}

	aiBot := model.Bot{
		Name:        "AI助手",
		Avatar:      "",
		Description: "基于大模型的智能助手，能回答各种问题",
		Type:        "ai",
		Config:      `{"api_key":"your-api-key", "model":"gpt-3.5-turbo", "temperature":0.7}`,
		IsActive:    true,
	}
	if err := db.Create(&aiBot).Error; err != nil {
		logger.WithModule("Init").Error("创建AI助手 Bot 失败", "error", err)
		return
	}

	logger.WithModule("Init").Info("Bot 模板初始化完成")
	markMigrationCompleted(db, "seed_bot_templates")
}

// InitApp 初始化应用
func InitApp() (*config.Config, *gorm.DB, *ws.Hub) {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db := database.Init(cfg)

	// 设置全局数据库实例
	SetDB(db)

	// 自动迁移表
	MigrateDB(db)

	// ========== 预置数据 ==========
	if cfg.DataInit.PresetData {
		// 初始化系统用户（无论什么环境都需要）
		initSystemUser()

		// 初始化管理员用户（无论什么环境都需要）
		initAdminUser()
	}

	// ========== Bot 模板 ==========
	if cfg.DataInit.BotTemplates {
		seedBotTemplates(db)
	}

	// ========== 演示小程序 ==========
	if cfg.DataInit.DemoMiniApps {
		test.SeedDemoMiniApps(db)
	}

	// ========== 测试数据 ==========
	if cfg.DataInit.TestData {
		// 添加测试数据
		test.AddTestData()

		// 初始化测试数据
		test.InitTestData(db)
	}

	// 初始化WebSocket Hub
	hub := ws.NewHub(database.GetDB(), cfg.Database.Type)
	ws.GlobalHub = hub
	go hub.Run()

	// 初始化依赖注入容器
	InitContainer(cfg, hub)

	// 初始化认证链
	auth.InitAuthChain()

	return cfg, db, hub
}

// MigrateDB 自动迁移数据库表
func MigrateDB(db *gorm.DB) {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Department{},
		&model.DepartmentEmployee{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Message{},
		&model.File{},
		&model.Folder{},
		&model.Note{},
		&model.ConversationSession{},
		&model.MessageReadReceipt{},
		&model.Bot{},
		&model.BotConversation{},
		&model.Event{},
		&model.SystemMessage{},
		&model.App{},
		&model.Notification{},
		&model.UserRole{},
		&model.Channel{},
		&model.ChannelSubscriber{},
		&model.ChannelMessage{},
		&model.ChannelMessageLike{},
		&model.ChannelMessageComment{},
		&model.ShortLink{},
		&model.Task{},
		&model.RealtimeSession{},     // 实时会话
		&model.RealtimeParticipant{}, // 实时参与者
		&model.AIConfig{},            // AI配置
		&model.Group{},               // 群聊
		&model.GroupDocument{},       // 群文档
		&model.SensitiveWord{},       // 敏感词
		&model.SystemConfig{},        // 系统配置
		&model.OperationLog{},        // 操作日志
		&model.ClientVersion{},       // 客户端版本
		&model.Blacklist{},           // 黑名单
		&model.AIProvider{},          // AI提供商
		&model.AvatarConfig{},        // 分身配置
		&model.AvatarSession{},       // 分身会话状态
		&model.AvatarLearnTask{},     // 分身学习任务
		&model.FileChunk{},           // 文件分片
		&model.UploadTask{},          // 上传任务
		&model.AuthProvider{},        // 认证提供者
		&model.ExternalUserMapping{}, // 外部用户映射
		&model.OrgSyncConfig{},       // 组织架构同步配置
		&model.OrgSyncLog{},          // 组织架构同步日志
		&model.UserFeedback{},        // 用户反馈
		&model.CrashLog{},            // 崩溃日志
	); err != nil {
		// GORM AutoMigrate 在表已存在时会报错，但这不影响程序运行
		// 只有在真正的迁移失败时才记录错误
		logger.WithModule("Migrate").Error("数据库迁移失败", "error", err)
		logger.WithModule("Migrate").Error("如果使用 MySQL，请检查：1) 数据库是否已创建 2) 连接账号是否有建表权限 3) 数据库名是否正确")
	}

	addIndexes(db)
	seedBuiltInApps(db)
	seedFileUploadConfig(db)
}

// isMigrationCompleted 检查指定的迁移版本是否已完成
func isMigrationCompleted(db *gorm.DB, migrationName string) bool {
	var config model.SystemConfig
	err := db.Where("key = ?", "migration:"+migrationName).First(&config).Error
	return err == nil
}

// markMigrationCompleted 标记指定的迁移版本为已完成
func markMigrationCompleted(db *gorm.DB, migrationName string) {
	config := model.SystemConfig{
		Key:   "migration:" + migrationName,
		Value: time.Now().Format(time.RFC3339),
		Type:  "string",
		Desc:  "迁移版本: " + migrationName,
	}
	db.Where("key = ?", "migration:"+migrationName).FirstOrCreate(&config)
	logger.WithModule("Migration").Info("标记迁移为已完成", "name", migrationName)
}

// seedBuiltInApps 初始化默认内置应用
func seedBuiltInApps(db *gorm.DB) {
	if isMigrationCompleted(db, "seed_built_in_apps") {
		return
	}

	if !db.Migrator().HasTable("apps") {
		return
	}

	var count int64
	db.Model(&model.App{}).Where("is_global = ?", true).Count(&count)
	if count > 0 {
		markMigrationCompleted(db, "seed_built_in_apps")
		return
	}

	now := time.Now()
	defaultApps := []model.App{
		{UserID: 1, Name: "日历", Code: "calendar", Icon: "fas fa-calendar", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "文件管理", Code: "file_manager", Icon: "fas fa-folder", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "任务管理", Code: "task_manager", Icon: "fas fa-check-square", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "便签", Code: "sticky_notes", Icon: "fas fa-sticky-note", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "笔记", Code: "notes", Icon: "fas fa-book", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "短链接管理", Code: "short_link", Icon: "fas fa-link", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
	}

	for _, app := range defaultApps {
		if err := db.Create(&app).Error; err != nil {
			logger.WithModule("Migrate").Error("创建内置应用失败", "name", app.Name, "error", err)
		}
	}

	logger.WithModule("Migrate").Info("内置应用种子数据初始化完成", "count", len(defaultApps))
	markMigrationCompleted(db, "seed_built_in_apps")
}

// seedFileUploadConfig 初始化文件上传配置（大小限制、允许的文件类型）
func seedFileUploadConfig(db *gorm.DB) {
	defaultConfigs := []model.SystemConfig{
		{Key: "file_upload:max_size", Value: "52428800", Type: "number", Desc: "文件上传最大大小（字节），默认 50MB"},
		{Key: "file_upload:allowed_extensions", Value: `[".jpg",".jpeg",".png",".gif",".bmp",".webp",".pdf",".doc",".docx",".xls",".xlsx",".ppt",".pptx",".txt",".md",".csv",".zip",".rar",".7z",".mp3",".wav",".mp4",".avi",".mov"]`, Type: "json", Desc: "允许上传的文件扩展名列表"},
	}
	for _, cfg := range defaultConfigs {
		db.Where("key = ?", cfg.Key).FirstOrCreate(&cfg)
	}
	logger.WithModule("Migrate").Info("文件上传配置初始化完成")
}

// addIndexes 添加性能优化索引，确保索引已存在则跳过创建
func addIndexes(db *gorm.DB) {
	cfg := config.Load()
	isMySQL := cfg.Database.Type == "mysql"

	// 1. messages(conversation_id, created_at) 复合索引
	if !db.Migrator().HasIndex(&model.Message{}, "idx_messages_conversation_created_at") {
		if isMySQL {
			db.Exec("CREATE INDEX idx_messages_conversation_created_at ON messages(conversation_id, created_at)")
		} else {
			db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_conversation_created_at ON messages(conversation_id, created_at)")
		}
		logger.WithModule("Index").Info("添加 messages(conversation_id, created_at) 复合索引")
	}

	// 2. groups(name) 索引
	if !db.Migrator().HasIndex(&model.Group{}, "idx_groups_name") {
		if isMySQL {
			db.Exec("CREATE INDEX idx_groups_name ON groups(name)")
		} else {
			db.Exec("CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name)")
		}
		logger.WithModule("Index").Info("添加 groups(name) 索引")
	}

	// 3. notifications(user_id, read, created_at) 复合索引
	if !db.Migrator().HasIndex(&model.Notification{}, "idx_notifications_user_read_created_at") {
		if isMySQL {
			db.Exec("CREATE INDEX idx_notifications_user_read_created_at ON notifications(user_id, `read`, created_at)")
		} else {
			db.Exec("CREATE INDEX IF NOT EXISTS idx_notifications_user_read_created_at ON notifications(user_id, `read`, created_at)")
		}
		logger.WithModule("Index").Info("添加 notifications(user_id, read, created_at) 复合索引")
	}

	// 4. 消息全文搜索索引
	if isMySQL {
		// MySQL: 使用 FULLTEXT INDEX
		if !hasFulltextIndex(db, "messages", "ft_messages_content") {
			db.Exec("ALTER TABLE messages ADD FULLTEXT INDEX ft_messages_content (content)")
			logger.WithModule("Index").Info("添加 messages FULLTEXT 全文索引")
		}
	} else {
		// SQLite: 使用 FTS5 虚拟表
		if isFTS5Available(db) && !hasFTS5Table(db, "messages_fts5") {
			err := db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts5 USING fts5(content, conversation_id, created_at, tokenize='unicode61')").Error
			if err == nil {
				db.Exec("INSERT INTO messages_fts5(content, conversation_id, created_at) SELECT content, conversation_id, created_at FROM messages")
				logger.WithModule("Index").Info("创建 messages FTS5 全文搜索虚拟表")
			} else {
				logger.WithModule("Index").Warn("创建 FTS5 虚拟表失败，将使用 LIKE 搜索", "error", err)
			}
		}
	}
}

// hasFulltextIndex 检查 MySQL 表是否存在指定名称的 FULLTEXT 索引
func hasFulltextIndex(db *gorm.DB, tableName, indexName string) bool {
	var count int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = ? 
		AND INDEX_NAME = ?`, tableName, indexName).Scan(&count)
	return count > 0
}

// hasFTS5Table 检查 SQLite 是否存在 FTS5 虚拟表
func hasFTS5Table(db *gorm.DB, tableName string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", tableName).Scan(&count)
	return count > 0
}

// isFTS5Available 检查 SQLite 是否支持 FTS5
func isFTS5Available(db *gorm.DB) bool {
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

// createFulltextIndexMySQL 在 MySQL 上创建全文索引
func createFulltextIndexMySQL(db *gorm.DB, tableName, indexName string, columns []string) bool {
	var exists bool
	db.Raw(`SELECT EXISTS(SELECT 1 FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = ? 
		AND INDEX_NAME = ?)`, tableName, indexName).Scan(&exists)

	if exists {
		return false
	}

	cols := strings.Join(columns, ", ")
	db.Exec(fmt.Sprintf("ALTER TABLE %s ADD FULLTEXT INDEX %s (%s)", tableName, indexName, cols))
	return true
}
