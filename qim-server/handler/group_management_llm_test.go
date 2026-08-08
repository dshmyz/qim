//go:build llm_integration

package handler

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestExecuteWithToolsRealLLM_KicksMember 真实 LLM 端到端验证「@AI 代管成员」：
// 以群主身份向 SmartReplyGraph.ExecuteWithTools 输入「把被踢者移出群」，
// 期望 LLM 返回 group_management(remove_member) tool call 并真实把成员从 DB 移除。
// 依赖 config.yaml 配置可用的 AI provider（qwen-flash）。
// 用 build tag llm_integration 隔离，默认 go test 不跑：go test -tags llm_integration -run TestExecuteWithToolsRealLLM
func TestExecuteWithToolsRealLLM_KicksMember(t *testing.T) {
	// config.Load() 读 cwd 下 config.yaml，go test 的 cwd 是包目录（handler/），
	// 切到 qim-server 根目录才能读到真实 AI 配置。
	if _, err := os.Stat("config.yaml"); err != nil {
		_, filename, _, _ := runtime.Caller(0)
		_ = os.Chdir(filepath.Dir(filepath.Dir(filename)))
	}
	cfg := config.Load()
	aiSvc := ai.NewAIService(&cfg.AI)
	if !aiSvc.IsConfigured() {
		t.Skip("AI 服务未配置，跳过真实 LLM 端到端测试")
	}
	toolRegistry := ai.NewToolRegistry(aiSvc)
	RegisterAdminTools(toolRegistry)
	aiSvc.SetToolRegistry(toolRegistry)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Conversation{}, &model.ConversationMember{}, &model.Group{}))
	database.DB = db

	owner := model.User{Username: "owner", Nickname: "群主"}
	victim := model.User{Username: "victim", Nickname: "被踢者"}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&victim).Error)
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: owner.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "owner", JoinedAt: time.Now()}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: victim.ID, Role: "member", JoinedAt: time.Now()}).Error)
	require.True(t, memberExists(db, conv.ID, victim.ID), "前置：被踢者应在群")

	graph := service.NewSmartReplyGraph(aiSvc, db, nil, nil, nil, nil)
	input := &service.SmartReplyContext{
		Message:         "把被踢者移出群",
		OriginalContent: "把被踢者移出群",
		UserID:          owner.ID,
		ConversationID:  conv.ID,
		IsAIMention:     true,
		AssistantName:   "AI助手",
	}

	reply, err := graph.ExecuteWithTools(context.Background(), input, service.ToolsetBuiltin)
	require.NoError(t, err)
	t.Logf("AI 回复: %q", reply)

	assert.False(t, memberExists(db, conv.ID, victim.ID),
		"被踢者应已被 AI 通过 group_management 工具移除；若仍在群，说明 LLM 未返回 tool call（模型能力/prompt 问题，非代码 bug）")
}
