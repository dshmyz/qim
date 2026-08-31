package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
)

// ai_tools_p1.go — P1 批次用户侧工具：日历（读/写）、文件搜索。
// 笔记/群文档检索已由 search_knowledge（UnifiedSearchGraph）覆盖，不另设工具；
// 审批属 admin 域（审批面板），不进用户侧工具面。
// 风险分级：list/search 只读直发；create_calendar_event 为本人低危写，直接执行
// （与 create_user_task 同级）；如未来出现高危写类工具，套 ConfirmTools 确认制。

// eventTimeLayouts 兼容模型可能输出的常见时间格式。
var eventTimeLayouts = []string{
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// ListCalendarEventsTool 查询用户日历日程。
type ListCalendarEventsTool struct {
	eventService *EventService
}

func NewListCalendarEventsTool(eventService *EventService) *ListCalendarEventsTool {
	return &ListCalendarEventsTool{eventService: eventService}
}

func (t *ListCalendarEventsTool) Name() string { return "list_calendar_events" }

func (t *ListCalendarEventsTool) Description() string {
	return "查询当前用户的日历日程列表，返回标题、开始/结束时间、是否全天。可用于回答「我有什么安排」。"
}

func (t *ListCalendarEventsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *ListCalendarEventsTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.eventService == nil {
		return nil, fmt.Errorf("event service not available")
	}
	var userID uint
	if ctx != nil {
		userID = ctx.UserID
	}
	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能查询日程")
	}

	// SQL 端 LIMIT：工具只需最近 50 条摘要 + 计数，全量物化浪费（用户上万条日程时
	// 每次调用都全表拉取再内存丢弃）
	const cap = 50
	events, total, err := t.eventService.GetEventsLimited(userID, cap)
	if err != nil {
		return nil, fmt.Errorf("查询日程失败: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(events))
	for _, e := range events {
		result = append(result, map[string]interface{}{
			"id":      e.ID,
			"title":   e.Title,
			"start":   e.Start.Format("2006-01-02 15:04"),
			"end":     e.End.Format("2006-01-02 15:04"),
			"all_day": e.AllDay,
		})
	}
	return map[string]interface{}{"events": result, "total": total, "returned": len(result)}, nil
}

// CreateCalendarEventTool 为用户创建日历事件（仅本人，低危写）。
type CreateCalendarEventTool struct {
	eventService *EventService
}

func NewCreateCalendarEventTool(eventService *EventService) *CreateCalendarEventTool {
	return &CreateCalendarEventTool{eventService: eventService}
}

func (t *CreateCalendarEventTool) Name() string { return "create_calendar_event" }

func (t *CreateCalendarEventTool) Description() string {
	return "为当前用户创建一条日历日程。需要标题和开始时间（格式 2006-01-02 15:04，全天日程可只传日期 2006-01-02）。"
}

func (t *CreateCalendarEventTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"title": map[string]interface{}{
			"type":        "string",
			"description": "日程标题",
			"required":    true,
		},
		"start": map[string]interface{}{
			"type":        "string",
			"description": "开始时间，格式 2006-01-02 15:04；全天日程可只传 2006-01-02",
			"required":    true,
		},
		"end": map[string]interface{}{
			"type":        "string",
			"description": "结束时间，格式同 start；可选，默认开始后 1 小时",
			"required":    false,
		},
		"all_day": map[string]interface{}{
			"type":        "boolean",
			"description": "是否全天日程，可选",
			"required":    false,
		},
	}
}

func (t *CreateCalendarEventTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.eventService == nil {
		return nil, fmt.Errorf("event service not available")
	}
	var userID uint
	if ctx != nil {
		userID = ctx.UserID
	}
	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能创建日程")
	}

	title, _ := params["title"].(string)
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	startStr, _ := params["start"].(string)
	start, err := parseFlexibleTime(startStr)
	if err != nil {
		return nil, fmt.Errorf("start 时间格式无效（应为 2006-01-02 15:04）: %w", err)
	}

	allDay, _ := params["all_day"].(bool)
	end := start.Add(time.Hour)
	if endStr, ok := params["end"].(string); ok && strings.TrimSpace(endStr) != "" {
		parsed, err := parseFlexibleTime(endStr)
		if err != nil {
			return nil, fmt.Errorf("end 时间格式无效: %w", err)
		}
		end = parsed
	} else if allDay {
		end = start.Add(24*time.Hour - time.Minute)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end 不能早于 start")
	}

	event := &model.Event{
		UserID: userID,
		Title:  title,
		Start:  start,
		End:    end,
		AllDay: allDay,
	}
	if err := t.eventService.CreateEvent(event); err != nil {
		return nil, fmt.Errorf("创建日程失败: %w", err)
	}
	return map[string]interface{}{
		"created": true,
		"id":      event.ID,
		"title":   event.Title,
		"start":   event.Start.Format("2006-01-02 15:04"),
		"end":     event.End.Format("2006-01-02 15:04"),
		"all_day": event.AllDay,
	}, nil
}

// parseFlexibleTime 按常见格式解析模型输出的时间字符串。
func parseFlexibleTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range eventTimeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间 %q", s)
}

// SearchFilesTool 关键词搜索用户个人文件。
type SearchFilesTool struct {
	fileService *FileService
}

func NewSearchFilesTool(fileService *FileService) *SearchFilesTool {
	return &SearchFilesTool{fileService: fileService}
}

func (t *SearchFilesTool) Name() string { return "search_files" }

func (t *SearchFilesTool) Description() string {
	return "按关键词搜索当前用户的个人文件，可按类型（image/video/audio）和是否收藏筛选。返回文件名、类型、大小、更新时间。"
}

func (t *SearchFilesTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"keyword": map[string]interface{}{
			"type":        "string",
			"description": "文件名关键词（可为空，配合 type/starred 筛选）",
			"required":    false,
		},
		"type": map[string]interface{}{
			"type":        "string",
			"description": "类型筛选：image、video、audio 或 mime 前缀，可选",
			"required":    false,
		},
		"starred": map[string]interface{}{
			"type":        "boolean",
			"description": "是否仅搜收藏文件，可选",
			"required":    false,
		},
	}
}

func (t *SearchFilesTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.fileService == nil {
		return nil, fmt.Errorf("file service not available")
	}
	var userID uint
	if ctx != nil {
		userID = ctx.UserID
	}
	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能搜索文件")
	}

	filters := map[string]string{}
	if kw, _ := params["keyword"].(string); strings.TrimSpace(kw) != "" {
		filters["search"] = strings.TrimSpace(kw)
	}
	if ft, _ := params["type"].(string); strings.TrimSpace(ft) != "" {
		filters["type"] = strings.TrimSpace(ft)
	}
	if starred, ok := params["starred"].(bool); ok {
		if starred {
			filters["starred"] = "true"
		} else {
			filters["starred"] = "false"
		}
	}

	const pageSize = 20
	files, total, err := t.fileService.GetFiles(userID, 1, pageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("搜索文件失败: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(files))
	for _, f := range files {
		result = append(result, map[string]interface{}{
			"id":         f.ID,
			"name":       f.Name,
			"mime_type":  f.MimeType,
			"size":       f.Size,
			"starred":    f.IsStarred,
			"updated_at": f.UpdatedAt.Format("2006-01-02 15:04"),
		})
	}
	return map[string]interface{}{"files": result, "total": total, "returned": len(result)}, nil
}

// RegisterP1Tools 注册 P1 批次工具（日历/文件）。
func RegisterP1Tools(registry *ai.ToolRegistry, eventSvc *EventService, fileSvc *FileService) {
	if eventSvc != nil {
		registry.RegisterTool(NewListCalendarEventsTool(eventSvc))
		registry.RegisterTool(NewCreateCalendarEventTool(eventSvc))
	}
	if fileSvc != nil {
		registry.RegisterTool(NewSearchFilesTool(fileSvc))
	}
}
