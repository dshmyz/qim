package app

import (
	"encoding/json"
	"fmt"
	"log"
	"qim-server/config"
	"qim-server/database"
	"qim-server/model"
	"qim-server/test"
	"qim-server/ws"
	"strings"
	"time"

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
		log.Printf("[Init] 创建系统用户成功: ID=%d", systemUser.ID)
		return nil
	})

	if err != nil {
		log.Printf("[Init] 创建系统用户失败: %v", err)
	}
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

	// 仅在非生产环境初始化测试数据
	if cfg.Server.Mode != "release" {
		// 添加测试数据
		test.AddTestData()

		// 初始化测试数据
		test.InitTestData(db)
	}

	// 初始化系统用户（无论什么环境都需要）
	initSystemUser()

	// 初始化WebSocket Hub
	hub := ws.NewHub(database.GetDB(), cfg.Database.Type)
	ws.GlobalHub = hub
	go hub.Run()

	// 初始化依赖注入容器
	InitContainer(cfg, hub)

	return cfg, db, hub
}

// MigrateDB 自动迁移数据库表
func MigrateDB(db *gorm.DB) {
	cleanupDuplicateReadReceipts(db)

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
	); err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 添加性能优化索引
	addIndexes(db)

	migrateMiniApps(db)
	migrateGroupData(db)
	migrateFileSource(db)
	migrateNoteStyle(db)
	migrateAIConfigs(db)
	migrateUserAIConfigs(db)
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
	log.Printf("[Migration] 标记迁移 %s 为已完成", migrationName)
}

// cleanupDuplicateReadReceipts 清理消息已读回执中的重复记录
func cleanupDuplicateReadReceipts(db *gorm.DB) {
	if !db.Migrator().HasTable("message_read_receipts") {
		return
	}

	db.Exec(`
		DELETE FROM message_read_receipts
		WHERE id NOT IN (
			SELECT MIN(id)
			FROM message_read_receipts
			GROUP BY message_id, user_id
		)
	`)
	log.Printf("已清理消息已读回执的重复记录")
}

// migrateGroupData 迁移群聊数据到Group表
func migrateGroupData(db *gorm.DB) {
	// 检查迁移版本是否已完成
	if isMigrationCompleted(db, "migrate_group_data") {
		return
	}

	// 检查Group表是否为空
	var count int64
	db.Model(&model.Group{}).Count(&count)
	if count > 0 {
		markMigrationCompleted(db, "migrate_group_data")
		return // 已有数据，跳过迁移
	}

	// 查询所有群聊和讨论组会话
	var conversations []model.Conversation
	db.Where("type = ? OR type = ?", "group", "discussion").Find(&conversations)

	// 迁移数据到Group表
	for _, conv := range conversations {
		group := model.Group{
			ConversationID:   conv.ID,
			GroupType:        conv.Type, // 使用原Type字段值作为GroupType
			Name:             "",
			Avatar:           "",
			CreatorID:        0,
			Announcement:     "",
			InvitePermission: "owner_admin",
			CreatedAt:        conv.CreatedAt,
			UpdatedAt:        conv.UpdatedAt,
		}
		db.Create(&group)
	}
	log.Printf("群聊数据迁移完成，共迁移 %d 个群聊", len(conversations))

	markMigrationCompleted(db, "migrate_group_data")
}

// migrateMiniApps 手动迁移 mini_apps 表，避免 AutoMigrate 在 SQLite 上产生 DDL 错误
func migrateMiniApps(db *gorm.DB) {
	// 检查迁移版本是否已完成
	if isMigrationCompleted(db, "migrate_mini_apps") {
		return
	}

	if !db.Migrator().HasTable("mini_apps") {
		if err := db.Migrator().CreateTable(&model.MiniApp{}); err != nil {
			log.Fatal("创建 mini_apps 表失败:", err)
		}
		markMigrationCompleted(db, "migrate_mini_apps")
		return
	}

	if !db.Migrator().HasColumn(&model.MiniApp{}, "permissions") {
		if err := db.Migrator().AddColumn(&model.MiniApp{}, "permissions"); err != nil {
			log.Fatal("为 mini_apps 表添加 permissions 字段失败:", err)
		}
	}

	markMigrationCompleted(db, "migrate_mini_apps")
}

// migrateFileSource 迁移文件来源字段，将聊天消息中引用的文件标记为chat来源
func migrateFileSource(db *gorm.DB) {
	// 检查迁移版本是否已完成
	if isMigrationCompleted(db, "migrate_file_source") {
		return
	}

	if !db.Migrator().HasTable("messages") || !db.Migrator().HasTable("files") {
		markMigrationCompleted(db, "migrate_file_source")
		return
	}

	var messages []model.Message
	if err := db.Where("type IN ?", []string{"file", "image"}).Find(&messages).Error; err != nil {
		log.Printf("查询聊天文件消息失败: %v", err)
		markMigrationCompleted(db, "migrate_file_source")
		return
	}

	updated := 0
	for _, msg := range messages {
		var fileData struct {
			URL string `json:"url"`
			ID  uint   `json:"id"`
		}
		if err := json.Unmarshal([]byte(msg.Content), &fileData); err != nil {
			continue
		}
		if fileData.ID > 0 {
			result := db.Model(&model.File{}).Where("id = ? AND source = ?", fileData.ID, "upload").Update("source", "chat")
			if result.RowsAffected > 0 {
				updated++
			}
		}
	}

	if updated > 0 {
		log.Printf("文件来源迁移完成，共更新 %d 个聊天文件", updated)
	}

	markMigrationCompleted(db, "migrate_file_source")
}

// migrateNoteStyle 为 notes 表添加 style 字段（兼容已存在的数据库）
func migrateNoteStyle(db *gorm.DB) {
	// 检查迁移版本是否已完成
	if isMigrationCompleted(db, "migrate_note_style") {
		return
	}

	if !db.Migrator().HasTable("notes") {
		markMigrationCompleted(db, "migrate_note_style")
		return
	}

	// Migrate color from color column to style JSON if color column exists
	if db.Migrator().HasColumn(&model.Note{}, "color") {
		// Get all notes with color values
		var notes []map[string]interface{}
		db.Table("notes").Select("id, color").Find(&notes)
		for _, note := range notes {
			if color, ok := note["color"].(string); color != "" && ok {
				// Get current style
				var styleStr string
				db.Table("notes").Where("id = ?", note["id"]).Pluck("style", &styleStr)
				if len(styleStr) == 0 {
					styleStr = "{}"
				}
				var styleMap map[string]interface{}
				if err := json.Unmarshal([]byte(styleStr), &styleMap); err == nil {
					styleMap["color"] = color
					if styleBytes, err := json.Marshal(styleMap); err == nil {
						db.Table("notes").Where("id = ?", note["id"]).Update("style", string(styleBytes))
					}
				}
			}
		}
		db.Migrator().DropColumn(&model.Note{}, "color")
		log.Printf("已将 notes 表的 color 字段迁移到 style JSON 中")
	}

	if !db.Migrator().HasColumn(&model.Note{}, "style") {
		if err := db.Migrator().AddColumn(&model.Note{}, "style"); err != nil {
			log.Printf("为 notes 表添加 style 字段失败: %v", err)
			markMigrationCompleted(db, "migrate_note_style")
			return
		}
		log.Printf("notes 表已添加 style 字段")
	}

	markMigrationCompleted(db, "migrate_note_style")
}

// migrateAIConfigs 迁移 AI 配置数据，将旧的冗余字段转换为新的 JSON 格式
func migrateAIConfigs(db *gorm.DB) {
	// 检查迁移版本是否已完成
	if isMigrationCompleted(db, "migrate_ai_configs") {
		return
	}

	if !db.Migrator().HasTable("ai_configs") {
		return
	}

	if !db.Migrator().HasColumn(&model.AIConfig{}, "config_json") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "config_json"); err != nil {
			log.Printf("为 ai_configs 表添加 config_json 字段失败: %v", err)
			return
		}
	}

	if db.Migrator().HasColumn(&model.AIConfig{}, "openai_api_key") {
		var aiConfigs []map[string]interface{}
		db.Table("ai_configs").Find(&aiConfigs)

		migrated := 0
		for _, cfg := range aiConfigs {
			provider, _ := cfg["provider"].(string)
			if provider == "" {
				provider = "openai"
			}

			var configJSON map[string]interface{}
			configJSON = map[string]interface{}{
				"provider": provider,
			}

			switch provider {
			case "openai":
				if apiKey, ok := cfg["openai_api_key"].(string); ok && apiKey != "" {
					configJSON["api_key"] = apiKey
				}
				if model, ok := cfg["openai_model"].(string); ok && model != "" {
					configJSON["model"] = model
				}
				if baseURL, ok := cfg["openai_base_url"].(string); ok && baseURL != "" {
					configJSON["base_url"] = baseURL
				}
			case "baidu":
				if apiKey, ok := cfg["baidu_api_key"].(string); ok && apiKey != "" {
					configJSON["api_key"] = apiKey
				}
				if secretKey, ok := cfg["baidu_secret_key"].(string); ok && secretKey != "" {
					configJSON["secret_key"] = secretKey
				}
				if model, ok := cfg["baidu_model"].(string); ok && model != "" {
					configJSON["model"] = model
				}
				if baseURL, ok := cfg["baidu_base_url"].(string); ok && baseURL != "" {
					configJSON["base_url"] = baseURL
				}
			case "alibaba":
				if apiKey, ok := cfg["alibaba_api_key"].(string); ok && apiKey != "" {
					configJSON["api_key"] = apiKey
				}
				if model, ok := cfg["alibaba_model"].(string); ok && model != "" {
					configJSON["model"] = model
				}
				if baseURL, ok := cfg["alibaba_base_url"].(string); ok && baseURL != "" {
					configJSON["base_url"] = baseURL
				}
			case "tencent":
				if secretID, ok := cfg["tencent_secret_id"].(string); ok && secretID != "" {
					configJSON["secret_id"] = secretID
				}
				if secretKey, ok := cfg["tencent_secret_key"].(string); ok && secretKey != "" {
					configJSON["secret_key"] = secretKey
				}
				if model, ok := cfg["tencent_model"].(string); ok && model != "" {
					configJSON["model"] = model
				}
				if baseURL, ok := cfg["tencent_base_url"].(string); ok && baseURL != "" {
					configJSON["base_url"] = baseURL
				}
			case "bytedance":
				if apiKey, ok := cfg["bytedance_api_key"].(string); ok && apiKey != "" {
					configJSON["api_key"] = apiKey
				}
				if model, ok := cfg["bytedance_model"].(string); ok && model != "" {
					configJSON["model"] = model
				}
				if baseURL, ok := cfg["bytedance_base_url"].(string); ok && baseURL != "" {
					configJSON["base_url"] = baseURL
				}
			case "anthropic":
				if apiKey, ok := cfg["anthropic_api_key"].(string); ok && apiKey != "" {
					configJSON["api_key"] = apiKey
				}
				if model, ok := cfg["anthropic_model"].(string); ok && model != "" {
					configJSON["model"] = model
				}
				if baseURL, ok := cfg["anthropic_base_url"].(string); ok && baseURL != "" {
					configJSON["base_url"] = baseURL
				}
			}

			if len(configJSON) > 1 {
				if configBytes, err := json.Marshal(configJSON); err == nil {
					db.Table("ai_configs").Where("id = ?", cfg["id"]).Update("config_json", string(configBytes))
					migrated++
				}
			}
		}

		if migrated > 0 {
			log.Printf("AI 配置数据迁移完成，共迁移 %d 条记录", migrated)
		}

		db.Migrator().DropColumn(&model.AIConfig{}, "openai_api_key")
		db.Migrator().DropColumn(&model.AIConfig{}, "openai_model")
		db.Migrator().DropColumn(&model.AIConfig{}, "openai_base_url")
		db.Migrator().DropColumn(&model.AIConfig{}, "baidu_api_key")
		db.Migrator().DropColumn(&model.AIConfig{}, "baidu_secret_key")
		db.Migrator().DropColumn(&model.AIConfig{}, "baidu_model")
		db.Migrator().DropColumn(&model.AIConfig{}, "baidu_base_url")
		db.Migrator().DropColumn(&model.AIConfig{}, "alibaba_api_key")
		db.Migrator().DropColumn(&model.AIConfig{}, "alibaba_model")
		db.Migrator().DropColumn(&model.AIConfig{}, "alibaba_base_url")
		db.Migrator().DropColumn(&model.AIConfig{}, "tencent_secret_id")
		db.Migrator().DropColumn(&model.AIConfig{}, "tencent_secret_key")
		db.Migrator().DropColumn(&model.AIConfig{}, "tencent_model")
		db.Migrator().DropColumn(&model.AIConfig{}, "tencent_base_url")
		db.Migrator().DropColumn(&model.AIConfig{}, "bytedance_api_key")
		db.Migrator().DropColumn(&model.AIConfig{}, "bytedance_model")
		db.Migrator().DropColumn(&model.AIConfig{}, "bytedance_base_url")
		db.Migrator().DropColumn(&model.AIConfig{}, "anthropic_api_key")
		db.Migrator().DropColumn(&model.AIConfig{}, "anthropic_model")
		db.Migrator().DropColumn(&model.AIConfig{}, "anthropic_base_url")
		log.Printf("已删除 ai_configs 表的旧供应商字段")
	}

	markMigrationCompleted(db, "migrate_ai_configs")
}

// migrateUserAIConfigs 将 user_ai_configs 表的数据迁移到 ai_configs 表，然后删除旧表
func migrateUserAIConfigs(db *gorm.DB) {
	// 检查迁移版本是否已完成
	if isMigrationCompleted(db, "migrate_user_ai_configs") {
		return
	}

	if !db.Migrator().HasTable("user_ai_configs") {
		return
	}

	if !db.Migrator().HasColumn(&model.AIConfig{}, "config_name") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "config_name"); err != nil {
			log.Printf("为 ai_configs 表添加 config_name 字段失败: %v", err)
			return
		}
	}
	if !db.Migrator().HasColumn(&model.AIConfig{}, "is_default") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "is_default"); err != nil {
			log.Printf("为 ai_configs 表添加 is_default 字段失败: %v", err)
			return
		}
	}
	if !db.Migrator().HasColumn(&model.AIConfig{}, "api_key_encrypted") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "api_key_encrypted"); err != nil {
			log.Printf("为 ai_configs 表添加 api_key_encrypted 字段失败: %v", err)
			return
		}
	}
	if !db.Migrator().HasColumn(&model.AIConfig{}, "model_name") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "model_name"); err != nil {
			log.Printf("为 ai_configs 表添加 model_name 字段失败: %v", err)
			return
		}
	}
	if !db.Migrator().HasColumn(&model.AIConfig{}, "base_url") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "base_url"); err != nil {
			log.Printf("为 ai_configs 表添加 base_url 字段失败: %v", err)
			return
		}
	}
	if !db.Migrator().HasColumn(&model.AIConfig{}, "is_verified") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "is_verified"); err != nil {
			log.Printf("为 ai_configs 表添加 is_verified 字段失败: %v", err)
			return
		}
	}
	if !db.Migrator().HasColumn(&model.AIConfig{}, "last_tested_at") {
		if err := db.Migrator().AddColumn(&model.AIConfig{}, "last_tested_at"); err != nil {
			log.Printf("为 ai_configs 表添加 last_tested_at 字段失败: %v", err)
			return
		}
	}

	var userConfigs []map[string]interface{}
	db.Table("user_ai_configs").Find(&userConfigs)

	migrated := 0
	for _, cfg := range userConfigs {
		userID, _ := cfg["user_id"].(uint64)
		configName, _ := cfg["config_name"].(string)
		provider, _ := cfg["provider"].(string)
		apiKeyEncrypted, _ := cfg["api_key_encrypted"].(string)
		modelName, _ := cfg["model_name"].(string)
		baseURL, _ := cfg["base_url"].(string)
		temperature, _ := cfg["temperature"].(float64)
		maxTokens, _ := cfg["max_tokens"].(int64)
		isVerified, _ := cfg["is_verified"].(bool)

		var lastTestedAt interface{}
		if v, ok := cfg["last_tested_at"]; ok && v != nil {
			lastTestedAt = v
		}

		createdAt, _ := cfg["created_at"].(time.Time)
		updatedAt, _ := cfg["updated_at"].(time.Time)

		newConfig := model.AIConfig{
			UserID:          uint(userID),
			ConfigName:      configName,
			IsDefault:       false,
			Provider:        provider,
			APIKeyEncrypted: apiKeyEncrypted,
			ModelName:       modelName,
			BaseURL:         baseURL,
			Temperature:     temperature,
			MaxTokens:       int(maxTokens),
			IsVerified:      isVerified,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
		}

		if lastTestedAt != nil {
			if t, ok := lastTestedAt.(time.Time); ok {
				newConfig.LastTestedAt = &t
			}
		}

		if err := db.Create(&newConfig).Error; err == nil {
			migrated++
		}
	}

	if migrated > 0 {
		log.Printf("User AI 配置数据迁移完成，共迁移 %d 条记录", migrated)
	}

	db.Migrator().DropTable("user_ai_configs")
	log.Printf("已删除 user_ai_configs 表")

	markMigrationCompleted(db, "migrate_user_ai_configs")
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
		log.Printf("[Index] 添加 messages(conversation_id, created_at) 复合索引")
	}

	// 2. groups(name) 索引
	if !db.Migrator().HasIndex(&model.Group{}, "idx_groups_name") {
		if isMySQL {
			db.Exec("CREATE INDEX idx_groups_name ON groups(name)")
		} else {
			db.Exec("CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name)")
		}
		log.Printf("[Index] 添加 groups(name) 索引")
	}

	// 3. notifications(user_id, read, created_at) 复合索引
	if !db.Migrator().HasIndex(&model.Notification{}, "idx_notifications_user_read_created_at") {
		if isMySQL {
			db.Exec("CREATE INDEX idx_notifications_user_read_created_at ON notifications(user_id, `read`, created_at)")
		} else {
			db.Exec("CREATE INDEX IF NOT EXISTS idx_notifications_user_read_created_at ON notifications(user_id, `read`, created_at)")
		}
		log.Printf("[Index] 添加 notifications(user_id, read, created_at) 复合索引")
	}

	// 4. 消息全文搜索索引
	if isMySQL {
		// MySQL: 使用 FULLTEXT INDEX
		if !hasFulltextIndex(db, "messages", "ft_messages_content") {
			db.Exec("ALTER TABLE messages ADD FULLTEXT INDEX ft_messages_content (content)")
			log.Printf("[Index] 添加 messages FULLTEXT 全文索引")
		}
	} else {
		// SQLite: 使用 FTS5 虚拟表
		if !hasFTS5Table(db, "messages_fts5") {
			db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts5 USING fts5(content, conversation_id, created_at, tokenize='unicode61')")
			// 同步现有数据到 FTS5
			db.Exec("INSERT INTO messages_fts5(content, conversation_id, created_at) SELECT content, conversation_id, created_at FROM messages")
			log.Printf("[Index] 创建 messages FTS5 全文搜索虚拟表")
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
