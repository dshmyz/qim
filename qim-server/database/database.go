package database

import (
	"fmt"
	"log"
	"time"
	"qim-server/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
			Logger: logger.Default.LogMode(logger.Warn),
		})
	} else {
		DB, err = gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
	}

	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("获取数据库实例失败:", err)
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

	log.Printf("数据库连接成功，连接池配置: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%ds", maxOpenConns, maxIdleConns, maxLifetime)
	return DB
}

func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Println("获取数据库实例失败:", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Println("数据库关闭失败:", err)
	}
}

func GetDB() *gorm.DB {
	return DB
}
