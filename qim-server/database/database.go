package database

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	qsqlite "github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.Config) *gorm.DB {
	var err error

	if cfg.Database.Type == "mysql" {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Database.Username,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Database,
		)
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gormLogger.Default.LogMode(gormLogger.Warn),
		})
	} else {
		// 确保数据库目录存在且可写
		dbDir := filepath.Dir(cfg.Database.Path)
		if dbDir != "." && dbDir != "/" {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				logger.WithModule("Database").Error("创建数据库目录失败", "path", dbDir, "error", err)
				os.Exit(1)
			}
		}

		// 纯 Go SQLite，禁用 mmap + WAL 避免信创/容器权限问题
		dsn := cfg.Database.Path + "?_pragma=mmap_size(0)&_pragma=journal_mode(DELETE)&_pragma=busy_timeout(5000)"
		DB, err = gorm.Open(qsqlite.Open(dsn), &gorm.Config{
			Logger: gormLogger.Default.LogMode(gormLogger.Warn),
		})
	}

	if err != nil {
		logger.WithModule("Database").Error("数据库连接失败", "error", err)
		os.Exit(1)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logger.WithModule("Database").Error("获取数据库实例失败", "error", err)
		os.Exit(1)
	}

	// SQLite 是进程内数据库，设置 MaxOpenConns=1 避免 WAL 锁竞争
	if cfg.Database.Type == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(0)
		logger.WithModule("Database").Info("SQLite 数据库连接成功（单连接模式）")
	} else {
		maxOpenConns := cfg.Database.MaxOpenConns
		if maxOpenConns <= 0 {
			maxOpenConns = 100
		}
		maxIdleConns := cfg.Database.MaxIdleConns
		if maxIdleConns <= 0 {
			maxIdleConns = 10
		}
		maxLifetime := cfg.Database.MaxLifetime
		if maxLifetime <= 0 {
			maxLifetime = 3600
		}

		sqlDB.SetMaxOpenConns(maxOpenConns)
		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)

		logger.WithModule("Database").Info("数据库连接成功", "maxOpen", maxOpenConns, "maxIdle", maxIdleConns, "maxLifetime", maxLifetime)
	}

	// 自动迁移数据库表结构
	// 最小迁移：补全 Event 表的 reminder_sent 列（原 autoMigrate 被注释后从未运行）
	if err := DB.AutoMigrate(&model.Event{}); err != nil {
		logger.WithModule("Database").Error("Event 表迁移失败", "error", err)
	}

	return DB
}

func autoMigrate() {
	err := DB.AutoMigrate(
		&model.User{},
		&model.Department{},
		&model.DepartmentEmployee{},
		&model.Conversation{},
		&model.Group{},
		&model.GroupDocument{},
		&model.ConversationMember{},
		&model.Message{},
		&model.MessageReadReceipt{},
		&model.File{},
		&model.Folder{},
		&model.Note{},
		&model.ConversationSession{},
		&model.Bot{},
		&model.AIUsageLog{},
		&model.BotConversation{},
		&model.Event{},
		&model.UserRole{},
		&model.SystemMessage{},
		&model.Task{},
		&model.MiniApp{},
		&model.App{},
		&model.Notification{},
		&model.Channel{},
		&model.ChannelSubscriber{},
		&model.ChannelMessage{},
		&model.ChannelMessageLike{},
		&model.ChannelMessageComment{},
		&model.ShortLink{},
		&model.AIProvider{},
		&model.AIConfig{},
		&model.SensitiveWord{},
		&model.SystemConfig{},
		&model.OperationLog{},
		&model.ClientVersion{},
		&model.Blacklist{},
		&model.AuthProvider{},
		&model.OrgSyncConfig{},
		&model.OrgSyncLog{},
		&model.AlertRule{},
		&model.AlertHistory{},
		&model.CrashLog{},
		&model.UserFeedback{},
	)
	if err != nil {
		logger.WithModule("Database").Error("数据库自动迁移失败", "error", err)
	} else {
		logger.WithModule("Database").Info("数据库自动迁移成功")
	}
}

func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		logger.WithModule("Database").Error("获取数据库实例失败", "error", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		logger.WithModule("Database").Error("数据库关闭失败", "error", err)
	}
}

func GetDB() *gorm.DB {
	return DB
}
