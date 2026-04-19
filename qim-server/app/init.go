package app

import (
	"log"
	"qim-server/config"
	"qim-server/database"
	"qim-server/model"
	"qim-server/test"
	"qim-server/ws"

	"gorm.io/gorm"
)

// InitApp 初始化应用
func InitApp() (*config.Config, *gorm.DB, *ws.Hub) {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db := database.Init(cfg)

	// 自动迁移表
	MigrateDB(db)

	// 添加测试数据
	test.AddTestData()

	// 初始化测试数据
	test.InitTestData(db)

	// 初始化WebSocket Hub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	go hub.Run()

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
		&model.MiniApp{},
		&model.App{},
		&model.Notification{},
		&model.UserRole{},
		&model.Channel{},
		&model.ChannelSubscriber{},
		&model.ChannelMessage{},
		&model.ShortLink{},
	); err != nil {
		log.Fatal("数据库迁移失败:", err)
	}
}
