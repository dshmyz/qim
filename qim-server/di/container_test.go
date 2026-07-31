package di

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/stretchr/testify/assert"
)

func TestInitContainer(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Path: ":memory:",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
		},
	}
	database.Init(cfg)

	hub := ws.NewHub(database.GetDB(), cfg.JWT.Secret, "http")
	go hub.Run()

	container, err := InitContainer(cfg, hub)
	if err != nil {
		t.Fatalf("InitContainer returned error: %v", err)
	}

	if container == nil {
		t.Fatal("InitContainer returned nil")
	}

	if container.DB == nil {
		t.Error("DB should not be nil")
	}

	if container.UserService == nil {
		t.Error("UserService should not be nil")
	}

	if container.ConversationService == nil {
		t.Error("ConversationService should not be nil")
	}

	if container.MessageService == nil {
		t.Error("MessageService should not be nil")
	}

	if container.NotificationService == nil {
		t.Error("NotificationService should not be nil")
	}

	if container.EventService == nil {
		t.Error("EventService should not be nil")
	}

	if container.TaskService == nil {
		t.Error("TaskService should not be nil")
	}

	if container.FileService == nil {
		t.Error("FileService should not be nil")
	}

	if container.GroupService == nil {
		t.Error("GroupService should not be nil")
	}

	if container.AppService == nil {
		t.Error("AppService should not be nil")
	}

	if container.MiniAppService == nil {
		t.Error("MiniAppService should not be nil")
	}

	if container.NoteService == nil {
		t.Error("NoteService should not be nil")
	}

	if container.AdminService == nil {
		t.Error("AdminService should not be nil")
	}

	if container.RealtimeService == nil {
		t.Error("RealtimeService should not be nil")
	}

	if container.SensitiveWordService == nil {
		t.Error("SensitiveWordService should not be nil")
	}

	if container.AvatarService == nil {
		t.Error("AvatarService should not be nil")
	}

	if container.AvatarTriggerService == nil {
		t.Error("AvatarTriggerService should not depend on vector storage initialization")
	}

	if container.ApprovalService == nil {
		t.Error("ApprovalService should not be nil")
	}

	if container.WebSocketHub == nil {
		t.Error("WebSocketHub should not be nil")
	}

	if container.WebSocketHub != hub {
		t.Error("WebSocketHub should be the same instance passed to InitContainer")
	}

	if container.AuthMiddleware == nil {
		t.Error("AuthMiddleware should not be nil")
	}

	if GlobalContainer != container {
		t.Error("GlobalContainer should be set to the returned container")
	}
}

// TestInitContainer_S3MisconfigFailsFast 验证配置了 type=s3 但 S3 不可达时，
// InitContainer 应 fail-fast 返回 error，而非静默降级到 local（否则数据写本地而运维无感）。
func TestInitContainer_S3MisconfigFailsFast(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Type: "sqlite", Path: ":memory:"},
		JWT:      config.JWTConfig{Secret: "test-secret"},
		Storage: config.StorageConfig{
			Type: "s3",
			S3: config.S3StorageConfig{
				Endpoint:  "http://127.0.0.1:1", // 不可达端口
				AccessKey: "x",
				SecretKey: "x",
				Bucket:    "nonexistent",
				Region:    "us-east-1",
			},
		},
	}
	database.Init(cfg)

	_, err := InitContainer(cfg, nil)
	assert.Error(t, err, "type=s3 配置错误应 fail-fast 返回 error，不静默降级 local")
}
