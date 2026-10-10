package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ── 统一检索合并去重 ──

// TestDedupSources 验证检索来源合并去重：DocID 命中（跨渠道同一文档）与
// 归一化内容命中（用户重复发送的相同消息/无 ID 来源撞车）任一重复即丢弃，
// 保序保留首个。
func TestDedupSources(t *testing.T) {
	sources := []SearchSource{
		{Type: "message", DocID: "msg_1", Content: "[张三]: 周三开会"},
		{Type: "memory", DocID: "mem_9", Content: "[张三]: 周三开会"},   // 内容撞车（跨渠道）
		{Type: "message", DocID: "msg_1", Content: "[张三]: 周三开会"}, // DocID 撞车
		{Type: "note", DocID: "note_2", Content: "  会议纪要  "},        // 无重复但带空白
		{Type: "note", DocID: "note_3", Content: "会议纪要"},            // 归一化后与上一条相同
		{Type: "message", DocID: "msg_2", Content: "[李四]: 收到"},
	}

	got := dedupSources(sources)
	assert.Len(t, got, 3, "应保留 3 条：msg_1、note_2、msg_2")
	assert.Equal(t, "msg_1", got[0].DocID)
	assert.Equal(t, "note_2", got[1].DocID)
	assert.Equal(t, "msg_2", got[2].DocID)
}

// ── ListTasksTool 出口去重 ──

// TestListTasksTool_DedupExit 验证任务列表工具出口去重：同名+同截止的重复行
// （历史数据/其他写入方残留）不再透传给模型，返回条数与 count 一致。
func TestListTasksTool_DedupExit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))

	due := time.Date(2030, 8, 18, 23, 59, 0, 0, time.UTC)
	tasks := []model.Task{
		{UserID: 7, Title: "跟进项目预算审批进度", DueDate: &due, Priority: "high", Status: "in_progress"},
		{UserID: 7, Title: "跟进项目预算审批进度", DueDate: &due, Priority: "medium", Status: "in_progress"}, // 重复（优先级不同）
		{UserID: 7, Title: "跟进项目预算审批进度", DueDate: nil, Priority: "high", Status: "todo"},             // 无截止，非重复
		{UserID: 7, Title: "跟进项目预算审批进度", DueDate: nil, Priority: "high", Status: "todo"},             // 无截止重复
		{UserID: 7, Title: "准备项目启动会会议议程", DueDate: &due, Priority: "medium", Status: "done"},
	}
	for i := range tasks {
		require.NoError(t, db.Create(&tasks[i]).Error)
	}

	tool := NewListTasksTool(NewTaskService(db))
	result, err := tool.Execute(map[string]interface{}{}, &ai.CallerContext{UserID: 7})
	require.NoError(t, err)

	res := result.(map[string]interface{})
	list := res["tasks"].([]map[string]interface{})
	assert.Len(t, list, 3, "5 行含 2 行重复，去重后应返回 3 条")
	assert.Equal(t, 3, res["count"])

	// 保留的是每组首条（高优先级那条在前——按主键序）
	titles := make(map[string]int)
	for _, t := range list {
		titles[t["title"].(string)]++
	}
	assert.Equal(t, 2, titles["跟进项目预算审批进度"], "同名不同截止的两条都在")
	assert.Equal(t, 1, titles["准备项目启动会会议议程"])
}
