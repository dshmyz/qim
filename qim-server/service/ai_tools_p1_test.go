package service

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseT(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := parseFlexibleTime(s)
	require.NoError(t, err)
	return tm
}

// newP1ToolFixture 造用户 + 日历事件 + 个人文件。
func newP1ToolFixture(t *testing.T) (*EventService, *FileService, *model.User, *gorm.DB) {
	t.Helper()
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Event{}, &model.File{}))

	user := &model.User{Username: "dora", Nickname: "朵拉"}
	require.NoError(t, db.Create(user).Error)
	other := &model.User{Username: "eve", Nickname: "伊芙"}
	require.NoError(t, db.Create(other).Error)

	eventSvc := NewEventService(db)
	require.NoError(t, eventSvc.CreateEvent(&model.Event{UserID: user.ID, Title: "周会", Start: parseT(t, "2026-09-01 10:00"), End: parseT(t, "2026-09-01 11:00")}))
	require.NoError(t, eventSvc.CreateEvent(&model.Event{UserID: user.ID, Title: "发版", Start: parseT(t, "2026-09-02 20:00"), End: parseT(t, "2026-09-02 21:00")}))
	require.NoError(t, eventSvc.CreateEvent(&model.Event{UserID: other.ID, Title: "别人的日程", Start: parseT(t, "2026-09-01 09:00"), End: parseT(t, "2026-09-01 10:00")}))

	fileSvc := NewFileService(db)
	require.NoError(t, fileSvc.CreateFile(&model.File{UserID: user.ID, Name: "季度报表.xlsx", OriginalName: "季度报表.xlsx", Size: 1024, MimeType: "application/vnd.ms-excel", ScopeType: "user", ScopeID: user.ID}))
	require.NoError(t, fileSvc.CreateFile(&model.File{UserID: user.ID, Name: "团建照片.png", OriginalName: "团建照片.png", Size: 204800, MimeType: "image/png", IsStarred: true, ScopeType: "user", ScopeID: user.ID}))
	// 群空间文件：CreateFile 会强制 scope=user，需直插模拟群空间记录
	require.NoError(t, db.Create(&model.File{UserID: user.ID, Name: "群空间文件.png", OriginalName: "群空间文件.png", Size: 1, MimeType: "image/png", ScopeType: "group", ScopeID: 999}).Error)

	return eventSvc, fileSvc, user, db
}

func TestListCalendarEventsTool(t *testing.T) {
	eventSvc, _, user, _ := newP1ToolFixture(t)
	tool := NewListCalendarEventsTool(eventSvc)

	result, err := tool.Execute(map[string]interface{}{}, &ai.CallerContext{UserID: user.ID})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.EqualValues(t, 2, m["total"], "只应看到自己的日程")

	events := m["events"].([]map[string]interface{})
	byTitle := map[string]map[string]interface{}{}
	for _, e := range events {
		byTitle[e["title"].(string)] = e
	}
	require.Len(t, byTitle, 2)
	assert.Equal(t, "2026-09-01 10:00", byTitle["周会"]["start"])
	assert.Equal(t, "2026-09-02 20:00", byTitle["发版"]["start"])
}

func TestCreateCalendarEventTool(t *testing.T) {
	eventSvc, _, user, _ := newP1ToolFixture(t)
	tool := NewCreateCalendarEventTool(eventSvc)
	ctx := &ai.CallerContext{UserID: user.ID}

	// 标准格式
	result, err := tool.Execute(map[string]interface{}{
		"title": "项目评审",
		"start": "2026-09-03 14:00",
		"end":   "2026-09-03 15:30",
	}, ctx)
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, true, m["created"])
	assert.Equal(t, "2026-09-03 14:00", m["start"])
	assert.Equal(t, "2026-09-03 15:30", m["end"])

	// 全天日程：只传日期，end 默认覆盖全天
	result, err = tool.Execute(map[string]interface{}{
		"title":   "外勤",
		"start":   "2026-09-04",
		"all_day": true,
	}, ctx)
	require.NoError(t, err)
	m = result.(map[string]interface{})
	assert.Equal(t, true, m["all_day"])
	assert.Equal(t, "2026-09-04 23:59", m["end"])

	// 非法时间
	_, err = tool.Execute(map[string]interface{}{"title": "坏数据", "start": "明天下午"}, ctx)
	assert.Error(t, err)

	// 缺标题
	_, err = tool.Execute(map[string]interface{}{"start": "2026-09-03 14:00"}, ctx)
	assert.Error(t, err)

	// end 早于 start
	_, err = tool.Execute(map[string]interface{}{"title": "倒挂", "start": "2026-09-03 14:00", "end": "2026-09-03 13:00"}, ctx)
	assert.Error(t, err)

	// 落库校验
	events, err := eventSvc.GetEvents(user.ID)
	require.NoError(t, err)
	assert.Len(t, events, 4) // 2 fixture + 2 created
}

func TestSearchFilesTool(t *testing.T) {
	_, fileSvc, user, _ := newP1ToolFixture(t)
	tool := NewSearchFilesTool(fileSvc)
	ctx := &ai.CallerContext{UserID: user.ID}

	// 关键词
	result, err := tool.Execute(map[string]interface{}{"keyword": "报表"}, ctx)
	require.NoError(t, err)
	m := result.(map[string]interface{})
	files := m["files"].([]map[string]interface{})
	require.Len(t, files, 1)
	assert.Equal(t, "季度报表.xlsx", files[0]["name"])

	// 类型筛选（群空间文件不应出现）
	result, err = tool.Execute(map[string]interface{}{"type": "image"}, ctx)
	require.NoError(t, err)
	m = result.(map[string]interface{})
	files = m["files"].([]map[string]interface{})
	require.Len(t, files, 1)
	assert.Equal(t, "团建照片.png", files[0]["name"])

	// 收藏筛选
	result, err = tool.Execute(map[string]interface{}{"starred": true}, ctx)
	require.NoError(t, err)
	m = result.(map[string]interface{})
	assert.EqualValues(t, 1, m["total"])

	// 无匹配
	result, err = tool.Execute(map[string]interface{}{"keyword": "不存在的文件"}, ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 0, result.(map[string]interface{})["total"])
}
