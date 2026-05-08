package app

import (
	"encoding/json"
	"log"
	"qim-server/config"
	"qim-server/database"
	"qim-server/model"
	"qim-server/test"
	"qim-server/ws"

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
	hub := ws.NewHub()
	ws.GlobalHub = hub
	go hub.Run()

	// 初始化依赖注入容器
	InitContainer(cfg.JWT.Secret, hub)

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
		&model.UserAIConfig{},        // 用户AI配置
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

	migrateMiniApps(db)
	migrateGroupData(db)
	migrateFileSource(db)
	migrateNoteStyle(db)
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
	// 检查Group表是否为空
	var count int64
	db.Model(&model.Group{}).Count(&count)
	if count > 0 {
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
}

// migrateMiniApps 手动迁移 mini_apps 表，避免 AutoMigrate 在 SQLite 上产生 DDL 错误
func migrateMiniApps(db *gorm.DB) {
	if !db.Migrator().HasTable("mini_apps") {
		if err := db.Migrator().CreateTable(&model.MiniApp{}); err != nil {
			log.Fatal("创建 mini_apps 表失败:", err)
		}
		return
	}

	if !db.Migrator().HasColumn(&model.MiniApp{}, "permissions") {
		if err := db.Migrator().AddColumn(&model.MiniApp{}, "permissions"); err != nil {
			log.Fatal("为 mini_apps 表添加 permissions 字段失败:", err)
		}
	}
}

// migrateFileSource 迁移文件来源字段，将聊天消息中引用的文件标记为chat来源
func migrateFileSource(db *gorm.DB) {
	if !db.Migrator().HasTable("messages") || !db.Migrator().HasTable("files") {
		return
	}

	var messages []model.Message
	if err := db.Where("type IN ?", []string{"file", "image"}).Find(&messages).Error; err != nil {
		log.Printf("查询聊天文件消息失败: %v", err)
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
}

// migrateNoteStyle 为 notes 表添加 style 字段（兼容已存在的数据库）
func migrateNoteStyle(db *gorm.DB) {
	if !db.Migrator().HasTable("notes") {
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
			return
		}
		log.Printf("notes 表已添加 style 字段")
	}
}
