package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/dshmyz/qim/qim-server/model"
)

func main() {
	// 配置日志
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if len(os.Args) < 2 {
		fmt.Println("用法: go run tools/test_migrate.go <mysql_dsn>")
		fmt.Println("示例: go run tools/test_migrate.go 'user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local'")
		os.Exit(1)
	}

	dsn := os.Args[1]

	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		logrus.Fatalf("连接数据库失败: %v", err)
	}

	logrus.Info("成功连接到 MySQL 数据库")

	// 所有模型列表
	models := []interface{}{
		&model.User{},
		&model.Department{},
		&model.DepartmentEmployee{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Message{},
		&model.MessageReadReceipt{},
		&model.File{},
		&model.Folder{},
		&model.Note{},
		&model.Bot{},
		&model.BotConversation{},
		&model.AIUsageLog{},
		&model.Event{},
		&model.SystemMessage{},
		&model.App{},
		&model.Notification{},
		&model.UserRole{},
		&model.AlertRule{},
		&model.AlertHistory{},
		&model.Channel{},
		&model.ChannelSubscriber{},
		&model.ChannelMessage{},
		&model.ChannelMessageLike{},
		&model.ChannelMessageComment{},
		&model.ShortLink{},
		&model.Task{},
		&model.RealtimeSession{},
		&model.RealtimeParticipant{},
		&model.AIConfig{},
		&model.Group{},
		&model.GroupDocument{},
		&model.SensitiveWord{},
		&model.SystemConfig{},
		&model.OperationLog{},
		&model.ClientVersion{},
		&model.Blacklist{},
		&model.AIProvider{},
		&model.MiniApp{},
		&model.AvatarConfig{},
		&model.AvatarSession{},
		&model.AvatarLearnTask{},
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

	// 逐表迁移，记录每个表的创建情况
	successCount := 0
	failCount := 0
	failedModels := []string{}

	for _, m := range models {
		modelName := fmt.Sprintf("%T", m)
		logrus.Infof("正在迁移表: %s", modelName)

		if err := db.AutoMigrate(m); err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "already exists") {
				logrus.Infof("表已存在，跳过: %s", modelName)
				successCount++
			} else {
				logrus.Errorf("迁移表失败 %s: %v", modelName, err)
				failCount++
				failedModels = append(failedModels, fmt.Sprintf("%s: %v", modelName, err))
			}
		} else {
			logrus.Infof("迁移表成功: %s", modelName)
			successCount++
		}
	}

	// 统计结果
	logrus.Info("========== 迁移结果统计 ==========")
	logrus.Infof("总数: %d", len(models))
	logrus.Infof("成功: %d", successCount)
	logrus.Infof("失败: %d", failCount)

	if failCount > 0 {
		logrus.Error("========== 失败的表 ==========")
		for _, fm := range failedModels {
			logrus.Error(fm)
		}
		os.Exit(1)
	} else {
		logrus.Info("========== 所有表迁移成功 ==========")
	}
}