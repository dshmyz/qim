package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/auth"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/test"
	"github.com/dshmyz/qim/qim-server/ws"

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

	if !tableExists(db, "users") {
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
	if !tableExists(db, "users") {
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

// seedBotTemplates 初始化 Bot 模板（系统助手、AI助手、业务模板）
func seedBotTemplates(db *gorm.DB) {
	if isMigrationCompleted(db, "seed_bot_templates") {
		return
	}

	if !tableExists(db, "bots") {
		markMigrationCompleted(db, "seed_bot_templates")
		return
	}

	var count int64
	db.Model(&model.Bot{}).Where("type IN ?", []string{model.BotTypeSystem, model.BotTypeAssistant}).Count(&count)
	if count > 0 {
		markMigrationCompleted(db, "seed_bot_templates")
		return
	}

	templates := []model.Bot{
		{
			Name:        "系统助手",
			Avatar:      "",
			Description: "提供系统相关的帮助和信息",
			Type:        model.BotTypeSystem,
			Config:      `{"responses":{"greeting":"你好！我是系统助手，有什么可以帮你的吗？","help":"我可以帮助你了解系统功能，解答常见问题。"}}`,
			IsActive:    true,
			IsTemplate:  true,
		},
		{
			Name:        "AI助手",
			Avatar:      "",
			Description: "基于大模型的智能助手，能回答各种问题",
			Type:        model.BotTypeAssistant,
			Config:      `{"system_prompt":"你是一个有用的AI助手，能够帮助用户回答各种问题、完成任务。","use_system_config":true}`,
			IsActive:    true,
			IsTemplate:  true,
		},
		{
			Name:        "代码助手",
			Avatar:      "",
			Description: "编程专家，帮助编写、审查、优化代码",
			Type:        model.BotTypeAssistant,
			Config:      `{"system_prompt":"你是一个经验丰富的编程助手，擅长多种编程语言。你能帮助用户编写高质量代码、进行代码审查、解决编程问题、优化性能。请提供清晰的代码示例和详细解释。","use_system_config":true}`,
			IsActive:    true,
			IsTemplate:  true,
		},
		{
			Name:        "翻译助手",
			Avatar:      "",
			Description: "多语言翻译专家，提供准确的翻译服务",
			Type:        model.BotTypeAssistant,
			Config:      `{"system_prompt":"你是一个专业的翻译助手，精通多种语言之间的翻译。请提供准确、流畅、符合语境的翻译结果。如果原文有歧义，请说明并提供多种翻译选项。","use_system_config":true}`,
			IsActive:    true,
			IsTemplate:  true,
		},
		{
			Name:        "写作助手",
			Avatar:      "",
			Description: "写作专家，帮助撰写文章、文案、报告等内容",
			Type:        model.BotTypeAssistant,
			Config:      `{"system_prompt":"你是一个专业的写作助手，能够帮助用户撰写各类文章、文案、报告、邮件等。请根据用户需求提供结构清晰、语言流畅、风格合适的内容。","use_system_config":true}`,
			IsActive:    true,
			IsTemplate:  true,
		},
	}

	for _, tpl := range templates {
		if err := db.Create(&tpl).Error; err != nil {
			logger.WithModule("Init").Error("创建 Bot 模板失败", "name", tpl.Name, "error", err)
		}
	}

	logger.WithModule("Init").Info("Bot 模板初始化完成", "count", len(templates))
	markMigrationCompleted(db, "seed_bot_templates")
}

// seedBusinessBotTemplates 补充业务 Bot 模板（用于已运行过旧版本迁移的数据库）
func seedBusinessBotTemplates(db *gorm.DB) {
	if isMigrationCompleted(db, "seed_business_bot_templates") {
		return
	}

	if !tableExists(db, "bots") {
		markMigrationCompleted(db, "seed_business_bot_templates")
		return
	}

	businessTemplates := []model.Bot{
		{
			Name:        "代码助手",
			Avatar:      "",
			Description: "编程专家，帮助编写、审查、优化代码",
			Type:        model.BotTypeAssistant,
			Config:      `{"system_prompt":"你是一个经验丰富的编程助手，擅长多种编程语言。你能帮助用户编写高质量代码、进行代码审查、解决编程问题、优化性能。请提供清晰的代码示例和详细解释。","use_system_config":true}`,
			IsActive:    true,
			IsTemplate:  true,
		},
		{
			Name:        "翻译助手",
			Avatar:      "",
			Description: "多语言翻译专家，提供准确的翻译服务",
			Type:        model.BotTypeAssistant,
			Config:      `{"system_prompt":"你是一个专业的翻译助手，精通多种语言之间的翻译。请提供准确、流畅、符合语境的翻译结果。如果原文有歧义，请说明并提供多种翻译选项。","use_system_config":true}`,
			IsActive:    true,
			IsTemplate:  true,
		},
		{
			Name:        "写作助手",
			Avatar:      "",
			Description: "写作专家，帮助撰写文章、文案、报告等内容",
			Type:        model.BotTypeAssistant,
			Config:      `{"system_prompt":"你是一个专业的写作助手，能够帮助用户撰写各类文章、文案、报告、邮件等。请根据用户需求提供结构清晰、语言流畅、风格合适的内容。","use_system_config":true}`,
			IsActive:    true,
			IsTemplate:  true,
		},
	}

	seeded := 0
	for _, tpl := range businessTemplates {
		var existing model.Bot
		err := db.Where("name = ? AND is_template = ?", tpl.Name, true).First(&existing).Error
		if err != nil {
			if err := db.Create(&tpl).Error; err != nil {
				logger.WithModule("Init").Error("创建 Bot 模板失败", "name", tpl.Name, "error", err)
			} else {
				seeded++
			}
		}
	}

	if seeded > 0 {
		logger.WithModule("Init").Info("补充业务 Bot 模板完成", "seeded", seeded)
	}
	markMigrationCompleted(db, "seed_business_bot_templates")
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

		// 初始化内置小程序（无论什么环境都需要）
		seedMiniApps(db)

		// 初始化系统默认频道（所有真人用户自动订阅）
		seedDefaultChannel(db)
	}

	// ========== Bot 模板 ==========
	if cfg.DataInit.BotTemplates {
		seedBotTemplates(db)
		seedBusinessBotTemplates(db)
	}

	// ========== 测试数据（迁移之后） ==========
	if cfg.DataInit.TestData {
		test.AddTestData()
		test.InitTestData(db)
	}

	// 初始化WebSocket Hub
	hub := ws.NewHub(database.GetDB(), cfg.JWT.Secret)
	ws.GlobalHub = hub
	go hub.Run()

	// 初始化依赖注入容器
	InitContainer(cfg, hub)

	// 初始化认证链
	auth.InitAuthChain()

	return cfg, db, hub
}

// tableExists 检查表是否存在，委托给方言实现
func tableExists(db *gorm.DB, tableName string) bool {
	return database.D.TableExists(db, tableName)
}

// MigrateDB 自动迁移数据库表（分步迁移策略）
func MigrateDB(db *gorm.DB) {
	// ========== 第一阶段：基础表（无外键依赖） ==========
	baseModels := []interface{}{
		&model.User{},
		&model.Department{},
		&model.File{},
		&model.Folder{},
		&model.Note{},
		&model.Bot{},
		&model.AIUsageLog{},
		&model.Event{},
		&model.AlertRule{},
		&model.AlertHistory{},
		&model.SensitiveWord{},
		&model.SystemConfig{},
		&model.OperationLog{},
		&model.ClientVersion{},
		&model.Blacklist{},
		&model.AIProvider{},
		&model.MiniApp{},
		&model.FileChunk{},
		&model.UploadTask{},
		&model.AuthProvider{},
		&model.OrgSyncConfig{},
		&model.OrgSyncLog{},
		&model.UserFeedback{},
		&model.CrashLog{},
		&model.Approval{},
		&model.ApprovalConfig{},
	}

	// ========== 第二阶段：关联表（有外键依赖） ==========
	relatedModels := []interface{}{
		&model.DepartmentEmployee{},    // 依赖 User, Department
		&model.Conversation{},          // 独立表
		&model.ConversationMember{},    // 依赖 User, Conversation
		&model.Message{},               // 依赖 User, Conversation
		&model.ConversationSession{},   // 依赖 User, Conversation
		&model.MessageReadReceipt{},    // 依赖 User, Message
		&model.BotConversation{},       // 依赖 Bot, User, Conversation
		&model.SystemMessage{},         // 依赖 User
		&model.App{},                   // 依赖 User
		&model.Notification{},          // 依赖 User
		&model.UserRole{},              // 依赖 User
		&model.Channel{},               // 依赖 User
		&model.ChannelSubscriber{},     // 依赖 User, Channel
		&model.ChannelMessage{},        // 依赖 User, Channel
		&model.ChannelMessageLike{},    // 依赖 User, ChannelMessage
		&model.ChannelMessageComment{}, // 依赖 User, ChannelMessage
		&model.ShortLink{},             // 依赖 User
		&model.Task{},                  // 依赖 User
		&model.RealtimeSession{},       // 独立表
		&model.RealtimeParticipant{},   // 依赖 RealtimeSession
		&model.AIConfig{},              // 依赖 User
		&model.Group{},                 // 依赖 Conversation
		&model.GroupDocument{},         // 依赖 Group, File
		&model.AvatarConfig{},          // 依赖 User
		&model.AvatarSession{},         // 依赖 User, AvatarConfig
		&model.AvatarToolBinding{},     // 依赖 AvatarConfig
		&model.AvatarLearnTask{},       // 依赖 User, AvatarConfig
		&model.DocumentProcessStatus{}, // 依赖 GroupDocument
	}

	// 分阶段迁移
	migrateModels(db, baseModels, "基础表")
	migrateModels(db, relatedModels, "关联表")

	addIndexes(db)
	seedBuiltInApps(db)
	seedFileUploadConfig(db)
	seedApprovalConfigs(db)
}

// migrateModels 迁移一组模型
func migrateModels(db *gorm.DB, models []interface{}, stage string) {
	logger.WithModule("Migrate").Info(fmt.Sprintf("开始迁移 %s", stage))

	migrated := 0
	skipped := 0
	failed := 0

	for _, m := range models {
		modelName := fmt.Sprintf("%T", m)

		err := db.AutoMigrate(m)

		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "already exists") {
				if !db.Migrator().HasTable(m) {
					if createErr := db.Migrator().CreateTable(m); createErr != nil {
						failed++
						logger.WithModule("Migrate").Error("迁移表失败", "model", modelName, "error", createErr)
					} else {
						migrated++
						logger.WithModule("Migrate").Info("表迁移成功", "model", modelName)
					}
				} else {
					skipped++
					logger.WithModule("Migrate").Info("表已存在，跳过", "model", modelName)
				}
			} else {
				failed++
				logger.WithModule("Migrate").Error("迁移表失败", "model", modelName, "error", err)
			}
		} else {
			migrated++
			logger.WithModule("Migrate").Info("表迁移成功", "model", modelName)
		}
	}

	logger.WithModule("Migrate").Info(fmt.Sprintf("%s迁移统计", stage),
		"migrated", migrated,
		"skipped", skipped,
		"failed", failed,
		"total", len(models))

	if failed > 0 {
		logger.WithModule("Migrate").Error(fmt.Sprintf("%s部分表迁移失败，请检查错误日志", stage))
	}
}

// isMigrationCompleted 检查指定的迁移版本是否已完成
func isMigrationCompleted(db *gorm.DB, migrationName string) bool {
	var config model.SystemConfig
	err := db.Where("config_key = ?", "migration:"+migrationName).First(&config).Error
	return err == nil
}

// markMigrationCompleted 标记指定的迁移版本为已完成
func markMigrationCompleted(db *gorm.DB, migrationName string) {
	config := model.SystemConfig{
		ConfigKey: "migration:" + migrationName,
		Value:     time.Now().Format(time.RFC3339),
		Type:      "string",
		Desc:      "迁移版本: " + migrationName,
	}
	db.Where("config_key = ?", "migration:"+migrationName).FirstOrCreate(&config)
	logger.WithModule("Migration").Info("标记迁移为已完成", "name", migrationName)
}

// seedMiniApps 初始化内置小程序
func seedMiniApps(db *gorm.DB) {
	if !tableExists(db, "mini_apps") {
		return
	}

	var count int64
	db.Model(&model.MiniApp{}).Count(&count)
	if count > 0 {
		return
	}

	logger.WithModule("Init").Info("初始化内置小程序数据...")

	miniApps := []model.MiniApp{
		{AppID: "calculator", Name: "计算器", Description: "简单易用的计算器", Path: "/miniapps/calculator/index.html", Status: "active"},
		{AppID: "sticky-notes", Name: "便签", Description: "快速记录想法和灵感", Path: "/miniapps/sticky-notes/index.html", Status: "active"},
		{AppID: "todo", Name: "待办事项", Description: "任务管理工具", Path: "/miniapps/todo/index.html", Status: "active"},
		{AppID: "json-formatter", Name: "JSON 格式化", Description: "JSON 格式化和压缩工具", Path: "/miniapps/json-formatter/index.html", Status: "active"},
		{AppID: "timestamp-converter", Name: "时间戳转换", Description: "时间戳与日期时间互转", Path: "/miniapps/timestamp-converter/index.html", Status: "active"},
		{AppID: "base64-converter", Name: "Base64 编解码", Description: "Base64 编码和解码工具", Path: "/miniapps/base64-converter/index.html", Status: "active"},
		{AppID: "unit-converter", Name: "单位转换", Description: "多种单位之间的转换", Path: "/miniapps/unit-converter/index.html", Status: "active"},
		{AppID: "password-generator", Name: "密码生成器", Description: "生成强密码", Path: "/miniapps/password-generator/index.html", Status: "active"},
	}

	for _, app := range miniApps {
		if err := db.Create(&app).Error; err != nil {
			logger.WithModule("Init").Error("创建内置小程序失败", "app_id", app.AppID, "error", err)
		}
	}

	logger.WithModule("Init").Info("内置小程序数据初始化完成", "count", len(miniApps))
}

// seedDefaultChannel 初始化系统默认频道（所有真人用户自动订阅）
func seedDefaultChannel(db *gorm.DB) {
	if isMigrationCompleted(db, "seed_default_channel") {
		return
	}
	if !tableExists(db, "channels") {
		return
	}

	var count int64
	db.Model(&model.Channel{}).Where("is_default = ?", true).Count(&count)
	if count == 0 {
		var creatorID uint
		var sysUser model.User
		if err := db.Where("type = ?", "system").First(&sysUser).Error; err == nil {
			creatorID = sysUser.ID
		}

		channel := model.Channel{
			Name:              "公告频道",
			Description:       "系统默认频道，所有成员自动订阅",
			CreatorID:         creatorID,
			Status:            "active",
			PublishPermission: "creator_only",
			CommentPermission: "all_subscribers",
			IsDefault:         true,
		}
		if err := db.Create(&channel).Error; err != nil {
			logger.WithModule("Init").Error("创建默认频道失败", "error", err)
			return
		}
		logger.WithModule("Init").Info("创建默认频道成功", "id", channel.ID)
	}

	markMigrationCompleted(db, "seed_default_channel")
}

// seedBuiltInApps 初始化默认内置应用
func seedBuiltInApps(db *gorm.DB) {
	if isMigrationCompleted(db, "seed_built_in_apps") {
		return
	}

	if !tableExists(db, "apps") {
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
		{UserID: 1, Name: "日历", Code: "calendar", Icon: "fas fa-calendar", Category: "main", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "文件管理", Code: "file_manager", Icon: "fas fa-folder", Category: "main", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "任务管理", Code: "task_manager", Icon: "fas fa-check-square", Category: "main", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "便签", Code: "sticky_notes", Icon: "fas fa-sticky-note", Category: "main", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "笔记", Code: "notes", Icon: "fas fa-book", Category: "main", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "短链接管理", Code: "short_link", Icon: "fas fa-link", Category: "tool", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
		{UserID: 1, Name: "智能助手", Code: "ai_assistant", Icon: "fas fa-robot", Category: "main", Status: "active", IsGlobal: true, OpenType: "in-app", CreatedAt: now, UpdatedAt: now},
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
		{ConfigKey: "file_upload:max_size", Value: "524288000", Type: "number", Desc: "文件上传最大大小（字节），默认 500MB"},
		{ConfigKey: "file_upload:enable_type_check", Value: "false", Type: "boolean", Desc: "是否启用文件类型检查，默认不启用（允许所有类型）"},
		{ConfigKey: "file_upload:allowed_extensions", Value: `[".jpg",".jpeg",".png",".gif",".bmp",".webp",".pdf",".doc",".docx",".xls",".xlsx",".ppt",".pptx",".txt",".md",".csv",".zip",".rar",".7z",".mp3",".wav",".mp4",".avi",".mov",".exe",".msi",".dmg",".pkg",".AppImage",".deb",".rpm"]`, Type: "json", Desc: "允许上传的文件扩展名列表（仅当启用类型检查时生效）"},
	}
	for _, cfg := range defaultConfigs {
		db.Where("config_key = ?", cfg.ConfigKey).FirstOrCreate(&cfg)
	}
	logger.WithModule("Migrate").Info("文件上传配置初始化完成")
}

// seedApprovalConfigs 初始化审批配置
func seedApprovalConfigs(db *gorm.DB) {
	if isMigrationCompleted(db, "seed_approval_configs") {
		return
	}

	if !tableExists(db, "approval_configs") {
		return
	}

	var count int64
	db.Model(&model.ApprovalConfig{}).Count(&count)
	if count > 0 {
		markMigrationCompleted(db, "seed_approval_configs")
		return
	}

	now := time.Now()
	defaultConfigs := []model.ApprovalConfig{
		{Type: "avatar", Enabled: true, Description: "分身功能审批", CreatedAt: now, UpdatedAt: now},
		{Type: "bot", Enabled: true, Description: "机器人创建审批", CreatedAt: now, UpdatedAt: now},
		{Type: "channel", Enabled: true, Description: "频道创建审批", CreatedAt: now, UpdatedAt: now},
		{Type: "group_ai", Enabled: true, Description: "群聊AI助手审批", CreatedAt: now, UpdatedAt: now},
	}

	for _, config := range defaultConfigs {
		if err := db.Create(&config).Error; err != nil {
			logger.WithModule("Migrate").Error("创建审批配置失败", "type", config.Type, "error", err)
		}
	}

	logger.WithModule("Migrate").Info("审批配置种子数据初始化完成", "count", len(defaultConfigs))
	markMigrationCompleted(db, "seed_approval_configs")
}

// addIndexes 添加性能优化索引，确保索引已存在则跳过创建
func addIndexes(db *gorm.DB) {
	// 1. messages(conversation_id, created_at) 复合索引
	if !db.Migrator().HasIndex(&model.Message{}, "idx_messages_conversation_created_at") {
		db.Exec(database.D.CreateIndexSQL("idx_messages_conversation_created_at", "messages", []string{"conversation_id", "created_at"}))
		logger.WithModule("Index").Info("添加 messages(conversation_id, created_at) 复合索引")
	}

	// 2. groups(name) 索引
	if !db.Migrator().HasIndex(&model.Group{}, "idx_groups_name") {
		db.Exec(database.D.CreateIndexSQL("idx_groups_name", "groups", []string{"name"}))
		logger.WithModule("Index").Info("添加 groups(name) 索引")
	}

	// 3. notifications(user_id, read, created_at) 复合索引
	if !db.Migrator().HasIndex(&model.Notification{}, "idx_notifications_user_read_created_at") {
		db.Exec(database.D.CreateIndexSQL("idx_notifications_user_read_created_at", "notifications", []string{"user_id", "read", "created_at"}))
		logger.WithModule("Index").Info("添加 notifications(user_id, read, created_at) 复合索引")
	}

	// 4. 消息全文搜索索引
	if database.D.SupportsFulltext() {
		if !database.D.HasFulltextIndex(db, "messages", "ft_messages_content") {
			database.D.CreateFulltextIndex(db, "messages", "ft_messages_content", []string{"content"})
			logger.WithModule("Index").Info("添加 messages FULLTEXT 全文索引")
		}
	} else if database.D.SupportsFTS5() {
		// SQLite：使用 FTS5 虚拟表
		if !tableExists(db, "messages_fts5") {
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
