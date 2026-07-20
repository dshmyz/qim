package sync

import (
	"context"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/scheduler"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupOrgSyncTestDB 构建内存 SQLite + OrgSyncConfig 表，供 LoadOrgSyncJobs 测试用
func setupOrgSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.OrgSyncConfig{}))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// LoadOrgSyncJobs 从 DB 加载启用的 OrgSyncConfig 并注册到 pkg/scheduler
// 期望返回 JobName 集合（便于测试断言）
func Test_LoadOrgSyncJobs_RegistersAllEnabledConfigs(t *testing.T) {
	db := setupOrgSyncTestDB(t)
	// 准备 3 条配置：2 启用 + 1 禁用
	// 注：model.OrgSyncConfig.Enabled 有 gorm:"default:true"，零值会触发默认值
	// 用 map 显式插入避免被默认值覆盖
	require.NoError(t, db.Model(&model.OrgSyncConfig{}).Create(map[string]interface{}{
		"name": "ldap-sync", "enabled": true, "sync_type": "ldap", "schedule": "0 2 * * *",
	}).Error)
	require.NoError(t, db.Model(&model.OrgSyncConfig{}).Create(map[string]interface{}{
		"name": "oauth-sync", "enabled": true, "sync_type": "oauth", "schedule": "@hourly",
	}).Error)
	require.NoError(t, db.Model(&model.OrgSyncConfig{}).Create(map[string]interface{}{
		"name": "disabled-sync", "enabled": false, "sync_type": "ldap", "schedule": "0 3 * * *",
	}).Error)

	sched := scheduler.New()
	engine := NewEngine()

	err := LoadOrgSyncJobs(sched, engine, db)
	require.NoError(t, err)

	names := sched.ListNames()
	assert.Contains(t, names, "orgsync-ldap-sync")
	assert.Contains(t, names, "orgsync-oauth-sync")
	assert.NotContains(t, names, "orgsync-disabled-sync", "禁用的 config 不应被注册")
	assert.Len(t, names, 2)
}

// 空格错误或非法 cron 的 config 应跳过，不阻塞其他 config 注册
func Test_LoadOrgSyncJobs_SkipsInvalidCronExpr(t *testing.T) {
	db := setupOrgSyncTestDB(t)
	require.NoError(t, db.Create(&model.OrgSyncConfig{
		Name: "bad-cron", Enabled: true, SyncType: "ldap", Schedule: "not a cron",
	}).Error)
	require.NoError(t, db.Create(&model.OrgSyncConfig{
		Name: "good-cron", Enabled: true, SyncType: "ldap", Schedule: "0 4 * * *",
	}).Error)

	sched := scheduler.New()
	engine := NewEngine()

	err := LoadOrgSyncJobs(sched, engine, db)
	require.NoError(t, err, "非法表达式应被跳过，不应让整个加载失败")

	names := sched.ListNames()
	assert.NotContains(t, names, "orgsync-bad-cron", "非法 cron 的 config 应被跳过")
	assert.Contains(t, names, "orgsync-good-cron", "合法 cron 的 config 应正常注册")
}

// ReloadOrgSyncJobs 移除旧 Job 后重新加载
func Test_ReloadOrgSyncJobs_RemovesOldAndReloadsNew(t *testing.T) {
	db := setupOrgSyncTestDB(t)
	require.NoError(t, db.Create(&model.OrgSyncConfig{
		Name: "keep-me", Enabled: true, SyncType: "ldap", Schedule: "0 2 * * *",
	}).Error)

	sched := scheduler.New()
	engine := NewEngine()

	require.NoError(t, LoadOrgSyncJobs(sched, engine, db))
	require.Contains(t, sched.ListNames(), "orgsync-keep-me")

	// 新增一条 + 删除原配置
	require.NoError(t, db.Where("name = ?", "keep-me").Delete(&model.OrgSyncConfig{}).Error)
	require.NoError(t, db.Create(&model.OrgSyncConfig{
		Name: "new-config", Enabled: true, SyncType: "ldap", Schedule: "0 5 * * *",
	}).Error)

	require.NoError(t, ReloadOrgSyncJobs(sched, engine, db))
	names := sched.ListNames()
	assert.NotContains(t, names, "orgsync-keep-me", "已删除的 config 的 Job 应被移除")
	assert.Contains(t, names, "orgsync-new-config", "新增 config 的 Job 应被注册")
}

// LoadOrgSyncJobs 在 Schedule 为空字符串时应跳过该 config
func Test_LoadOrgSyncJobs_SkipsEmptySchedule(t *testing.T) {
	db := setupOrgSyncTestDB(t)
	require.NoError(t, db.Create(&model.OrgSyncConfig{
		Name: "no-schedule", Enabled: true, SyncType: "ldap", Schedule: "",
	}).Error)

	sched := scheduler.New()
	engine := NewEngine()

	require.NoError(t, LoadOrgSyncJobs(sched, engine, db))
	assert.Empty(t, sched.ListNames(), "空 Schedule 应被跳过")
}

// 确保注册的 Job callback 真的会调用 syncFn（不会立刻执行）
func Test_LoadOrgSyncJobs_JobCallback_CallsSyncFn(t *testing.T) {
	db := setupOrgSyncTestDB(t)
	require.NoError(t, db.Create(&model.OrgSyncConfig{
		Name: "quick-trigger", Enabled: true, SyncType: "ldap", Schedule: "@every 1s",
	}).Error)

	sched := scheduler.New()
	called := make(chan struct{}, 1)
	syncFn := func(ctx context.Context, cfg *model.OrgSyncConfig) {
		select {
		case called <- struct{}{}:
		default:
		}
	}

	require.NoError(t, LoadOrgSyncJobsWithSyncFn(sched, db, syncFn))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched.Start(ctx)
	defer sched.Stop()

	select {
	case <-called:
		// 触发了
	case <-time.After(3 * time.Second):
		t.Fatal("Job callback 未被触发")
	}
}
