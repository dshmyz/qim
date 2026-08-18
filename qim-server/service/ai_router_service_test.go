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
			ai.TaskTypeChat:   {Provider: "openai", Model: "gpt-4o"},
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
	_, saveErr := svc.SaveRouter(aiSvc, rc)
	require.NoError(t, saveErr)

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

// 路由引用了未配置的供应商：该路由被跳过并返回警告，其余有效路由正常保存。
func TestAIRouter_SaveRouter_SkipsUnknownProviderWithWarning(t *testing.T) {
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
	rc := &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat:   {Provider: "volc", Model: "deepseek-v4-flash"},
			ai.TaskTypeDigest: {Provider: "not-exist", Model: "gpt-4o"},
		},
	}
	warnings, err := svc.SaveRouter(aiSvc, rc)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "not-exist")

	// 有效路由已保存，无效路由被跳过
	got, _, err := svc.GetEffectiveRouter()
	require.NoError(t, err)
	assert.Equal(t, "volc", got.Routes[ai.TaskTypeChat].Provider)
	_, hasInvalid := got.Routes[ai.TaskTypeDigest]
	assert.False(t, hasInvalid, "无效路由不应被写入")
}

// 全部路由都引用未配置供应商时，不应覆盖既有配置
func TestAIRouter_SaveRouter_AllUnknown_NoWrite(t *testing.T) {
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
	_, err := svc.SaveRouter(aiSvc, rc)
	require.Error(t, err)

	has, err := svc.HasDBRouter()
	require.NoError(t, err)
	assert.False(t, has)
}

// 纯 config.yaml 部署（DB 无供应商）时，运行时池子回退 config.yaml 供应商，
// 路由引用该 provider 应被放行（运行时能用则保存应通过），而非误报“未配置”。
func TestAIRouter_SaveRouter_AcceptsConfigProviderWhenDBEmpty(t *testing.T) {
	db := setupAIRouterTestDB(t)
	svc := NewAIRouterService(db)
	svc.SetDefaultRouterFunc(defaultRouterForTest)

	// DB 不登记任何供应商；AIService 仅靠 config.yaml 的 openai 配置初始化池子
	aiSvc := ai.NewAIService(&ai.AIConfig{OpenAI: ai.OpenAIConfig{APIKey: "test-key", Model: "deepseek-v4-flash", BaseURL: "https://example.com"}})
	assert.Contains(t, aiSvc.ProviderNames(), "openai", "config.yaml openai 应在运行时池中")

	rc := &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "openai", Model: "deepseek-v4-flash"},
		},
	}
	// 修复前：openai 不在 DB -> ErrUnknownRouterProvider；修复后：池子含 openai -> 放行
	_, saveErr := svc.SaveRouter(aiSvc, rc)
	require.NoError(t, saveErr)

	got, _, err := svc.GetEffectiveRouter()
	require.NoError(t, err)
	assert.Equal(t, "openai", got.Routes[ai.TaskTypeChat].Provider)
}

// model 不再做硬校验：已登记/未登记（自定义输入）均可保存，模型由 provider 运行时校验。
func TestAIRouter_SaveRouter_ModelFreeInput(t *testing.T) {
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

	// 已登记的 model → 通过
	warnings, err := svc.SaveRouter(aiSvc, &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "volc", Model: "deepseek-v4-flash"},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, warnings)

	// 未登记（自定义）model → 也通过（运行时由 provider 校验）
	warnings, err = svc.SaveRouter(aiSvc, &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "volc", Model: "my-custom-model"},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, warnings)
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
	_, saveErr := svc.SaveRouter(aiSvc, &ai.RouterConfig{
		DefaultTask: ai.TaskTypeChat,
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeChat: {Provider: "volc", Model: "deepseek-v4-flash"},
		},
	})
	require.NoError(t, saveErr)

	require.NoError(t, svc.ClearRouter(aiSvc))

	rc, _, err := svc.GetEffectiveRouter()
	require.NoError(t, err)
	// 回到 config.yaml 默认
	assert.Equal(t, "gpt-4o", rc.Routes[ai.TaskTypeChat].Model)

	has, err := svc.HasDBRouter()
	require.NoError(t, err)
	assert.False(t, has)
}
