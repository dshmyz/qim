package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newEventCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "管理日历事件",
	}
	cmd.AddCommand(newEventListCmd(), newEventCreateCmd(), newEventUpdateCmd())
	return cmd
}

func newEventListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出日历事件",
		Run: func(cmd *cobra.Command, args []string) {
			respBody, err := userGet("/api/v1/events")
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data []struct {
					ID       uint64 `json:"id"`
					Title    string `json:"title"`
					Start    string `json:"start"`
					End      string `json:"end"`
					Reminder int    `json:"reminder"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			events := resp.Data
			if len(events) > limit {
				events = events[:limit]
			}
			if len(events) == 0 {
				fmt.Println("（无事件）")
				return
			}
			for _, e := range events {
				remind := ""
				if e.Reminder > 0 {
					remind = fmt.Sprintf(" (提醒提前%d分钟)", e.Reminder)
				}
				start := e.Start
				if len(start) > 16 {
					start = start[:16]
				}
				end := e.End
				if len(end) > 16 {
					end = end[:16]
				}
				fmt.Printf("#%-4d %s  %s ~ %s%s\n", e.ID, e.Title, start, end, remind)
			}
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "最多返回条数")
	return cmd
}

func newEventCreateCmd() *cobra.Command {
	var title, start, end, desc string
	var reminder int
	cmd := &cobra.Command{
		Use:   "create",
		Short: "建日历事件",
		Run: func(cmd *cobra.Command, args []string) {
			if title == "" || start == "" || end == "" {
				die("--title/--start/--end 必填")
			}
			startRFC, err := localToRFC3339(start)
			if err != nil {
				die("--start 格式错误: %v（示例: \"2026-08-01 14:00\"）", err)
			}
			endRFC, err := localToRFC3339(end)
			if err != nil {
				die("--end 格式错误: %v", err)
			}
			body := map[string]any{
				"title":    title,
				"start":    startRFC,
				"end":      endRFC,
				"reminder": reminder,
			}
			if desc != "" {
				body["description"] = desc
			}
			respBody, err := userPost("/api/v1/events", body)
			if err != nil {
				die("%v", err)
			}
			var resp struct {
				Data struct {
					ID uint64 `json:"id"`
				} `json:"data"`
			}
			_ = json.Unmarshal(respBody, &resp)
			if resp.Data.ID == 0 {
				fmt.Fprintln(os.Stderr, "创建失败:", string(respBody))
				os.Exit(1)
			}
			outRaw(respBody, "✅ 事件已创建 (ID: %d)\n", resp.Data.ID)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "事件标题（必填）")
	cmd.Flags().StringVar(&start, "start", "", "开始时间，如 \"2026-08-01 14:00\"（必填，本地时间）")
	cmd.Flags().StringVar(&end, "end", "", "结束时间，如 \"2026-08-01 15:00\"（必填，本地时间）")
	cmd.Flags().IntVar(&reminder, "reminder", 0, "提前提醒分钟数（0=不提醒）")
	cmd.Flags().StringVar(&desc, "desc", "", "描述（可选）")
	return cmd
}

func newEventUpdateCmd() *cobra.Command {
	var title, start, end, desc string
	var reminder int
	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "改事件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if !isNumeric(id) {
				die("事件 ID 必须是数字: %s", id)
			}
			// API 要求 title/start/end 全部必填，先拉取现有事件合并
			currentBody, err := userGet("/api/v1/events/" + id)
			if err != nil {
				die("获取事件失败: %v", err)
			}
			var curResp struct {
				Data struct {
					Title       string `json:"title"`
					Description string `json:"description"`
					Start       string `json:"start"`
					End         string `json:"end"`
					Reminder    int    `json:"reminder"`
				} `json:"data"`
			}
			if err := json.Unmarshal(currentBody, &curResp); err != nil {
				die("解析事件失败: %v", err)
			}

			ev := curResp.Data
			body := map[string]any{
				"title":    ev.Title,
				"start":    ev.Start,
				"end":      ev.End,
				"reminder": ev.Reminder,
			}
			if ev.Description != "" {
				body["description"] = ev.Description
			}
			if title != "" {
				body["title"] = title
			}
			if start != "" {
				s, err := localToRFC3339(start)
				if err != nil {
					die("--start 格式错误: %v", err)
				}
				body["start"] = s
			}
			if end != "" {
				e, err := localToRFC3339(end)
				if err != nil {
					die("--end 格式错误: %v", err)
				}
				body["end"] = e
			}
			if reminder >= 0 {
				body["reminder"] = reminder
			}
			if desc != "" {
				body["description"] = desc
			}
			respBody, err := userPut("/api/v1/events/"+id, body)
			if err != nil {
				die("%v", err)
			}
			outRaw(respBody, "✅ 事件 #%s 已更新\n", id)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "新标题")
	cmd.Flags().StringVar(&start, "start", "", "新开始时间")
	cmd.Flags().StringVar(&end, "end", "", "新结束时间")
	cmd.Flags().IntVar(&reminder, "reminder", -1, "新提醒分钟数（-1=不改）")
	cmd.Flags().StringVar(&desc, "desc", "", "新描述")
	return cmd
}

// localToRFC3339 把 "2006-01-02 15:04" 本地时间字符串转为 RFC3339（带本地时区偏移）。
func localToRFC3339(s string) (string, error) {
	layouts := []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02T15:04", time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("无法解析 %q", s)
}
