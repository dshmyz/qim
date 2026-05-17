package database

import (
	"fmt"
	"os"
	"qim-server/config"
	"qim-server/pkg/logger"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
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
		DB, err = gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{
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
	return DB
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
