package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAIRouterTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.SystemConfig{},
		&model.AIProvider{},
	))
	return db
}

// 注入一个固定默认路由：未配置 DB 覆盖时使用
func defaultRouterForTest() *ai.RouterConfig {
	return &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat:  {Provider: "openai", Model: "gpt-4o"},
			ai.TaskTypeDigest: {Provider: "openai", Model: "gpt-4o-mini"},
		},
	}
}

// 无 DB 覆盖时，GetEffectiveRouter 应返回 config.yaml 默认路由
func TestAIRouter_GetEffectiveRouter_FallsBackToDefault(t *testing.T) {
	db := setupAIRouterTestDB(t)
	svc := NewAIRouterService(db)
	svc.SetDefaultRouterFunc(defaultRouterForTest)

	rc, providers, err := svc.GetEffectiveRouter()
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Equal(t, ai.TaskTypeChat, rc.DefaultTask)
	assert.Equal(t, "gpt-4o", rc.Routes[ai.TaskTypeChat].Model)
	assert.Empty(t, providers) // 未添加任何供应商

	has, err := svc.HasDBRouter()
	require.NoError(t, err)
	assert.False(t, has)
}

// Save → Get 往返：DB 覆盖生效，且默认路由被克隆不互相污染
func TestAIRouter_SaveThenGet_RoundTrip(t *testing.T) {
	db := setupAIRouterTestDB(t)
	svc := NewAIRouterService(db)
	svc.SetDefaultRouterFunc(defaultRouterForTest)

	// 添加一个启用的供应商，作为路由 provider 候选
	require.NoError(t, db.Create(&model.AIProvider{
		Name:    "volc",
		APIType: "openai",
		Models:  model.StringArray{"deepseek-v4-flash"},
		Enabled: true,
	}).Error)

	aiSvc := ai.NewAIService(&ai.AIConfig{})
	rc := &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "volc", Model: "deepseek-v4-flash"},
		},
	}
	require.NoError(t, svc.SaveRouter(aiSvc, rc))

	got, providers, err := svc.GetEffectiveRouter()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "volc", got.Routes[ai.TaskTypeChat].Provider)
	assert.Equal(t, "deepseek-v4-flash", got.Routes[ai.TaskTypeChat].Model)
	assert.Len(t, providers, 1)

	has, err := svc.HasDBRouter()
	require.NoError(t, err)
	assert.True(t, has)
}

// 路由引用了未配置的供应商，应当被拒绝
func TestAIRouter_SaveRouter_RejectsUnknownProvider(t *testing.T) {
	db := setupAIRouterTestDB(t)
	svc := NewAIRouterService(db)
	svc.SetDefaultRouterFunc(defaultRouterForTest)

	aiSvc := ai.NewAIService(&ai.AIConfig{})
	rc := &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "not-exist", Model: "gpt-4o"},
		},
	}
	err := svc.SaveRouter(aiSvc, rc)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownRouterProvider)

	// 校验失败不应写入 DB
	_, err = svc.GetDBRouter()
	require.NoError(t, err)
	has, err := svc.HasDBRouter()
	require.NoError(t, err)
	assert.False(t, has)
}

// 供应商启用了但 model 未登记，若其 models 非空则拒绝；models 为空允许自由输入
func TestAIRouter_SaveRouter_ModelValidation(t *testing.T) {
	db := setupAIRouterTestDB(t)
	svc := NewAIRouterService(db)
	svc.SetDefaultRouterFunc(defaultRouterForTest)

	require.NoError(t, db.Create(&model.AIProvider{
		Name:    "volc",
		APIType: "openai",
		Models:  model.StringArray{"deepseek-v4-flash"},
		Enabled: true,
	}).Error)

	aiSvc := ai.NewAIService(&ai.AIConfig{})

	// 未登记该 provider 的 model → 拒绝
	err := svc.SaveRouter(aiSvc, &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "volc", Model: "wrong-model"},
		},
	})
	require.Error(t, err)

	// 已登记的 model → 通过
	require.NoError(t, svc.SaveRouter(aiSvc, &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "volc", Model: "deepseek-v4-flash"},
		},
	}))
}

// ClearRouter 清除 DB 覆盖后，应回到 config.yaml 默认
func TestAIRouter_Clear_RevertsToDefault(t *testing.T) {
	db := setupAIRouterTestDB(t)
	svc := NewAIRouterService(db)
	svc.SetDefaultRouterFunc(defaultRouterForTest)

	require.NoError(t, db.Create(&model.AIProvider{
		Name:    "volc",
		APIType: "openai",
		Models:  model.StringArray{"deepseek-v4-flash"},
		Enabled: true,
	}).Error)

	aiSvc := ai.NewAIService(&ai.AIConfig{})
	require.NoError(t, svc.SaveRouter(aiSvc, &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "volc", Model: "deepseek-v4-flash"},
		},
	}))

	require.NoError(t, svc.ClearRouter(aiSvc))

	rc, _, err := svc.GetEffectiveRouter()
	require.NoError(t, err)
	// 回到 config.yaml 默认
	assert.Equal(t, "gpt-4o", rc.Routes[ai.TaskTypeChat].Model)

	has, err := svc.HasDBRouter()
	require.NoError(t, err)
	assert.False(t, has)
}
