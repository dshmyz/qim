package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "管理待办",
	}
	cmd.AddCommand(newTaskListCmd(), newTaskCreateCmd(), newTaskUpdateCmd())
	return cmd
}

func newTaskListCmd() *cobra.Command {
	var status string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出待办",
		Run: func(cmd *cobra.Command, args []string) {
			respBody, err := userGet("/api/v1/tasks")
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data []struct {
					ID       uint64  `json:"id"`
					Title    string  `json:"title"`
					Status   string  `json:"status"`
					Priority string  `json:"priority"`
					DueDate  *string `json:"due_date"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			tasks := resp.Data
			if status != "" {
				var filtered []struct {
					ID       uint64  `json:"id"`
					Title    string  `json:"title"`
					Status   string  `json:"status"`
					Priority string  `json:"priority"`
					DueDate  *string `json:"due_date"`
				}
				for _, t := range tasks {
					if t.Status == status {
						filtered = append(filtered, t)
					}
				}
				tasks = filtered
			}
			if len(tasks) > limit {
				tasks = tasks[:limit]
			}
			if len(tasks) == 0 {
				fmt.Println("（无待办）")
				return
			}
			for _, t := range tasks {
				due := "-"
				if t.DueDate != nil && *t.DueDate != "" {
					due = (*t.DueDate)[:10]
				}
				fmt.Printf("#%-4d [%s] %s  (priority=%s, due=%s)\n", t.ID, t.Status, t.Title, t.Priority, due)
			}
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "筛选状态: todo|doing|done")
	cmd.Flags().IntVar(&limit, "limit", 50, "最多返回条数")
	return cmd
}

func newTaskCreateCmd() *cobra.Command {
	var title, due, priority, desc string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "建待办",
		Run: func(cmd *cobra.Command, args []string) {
			if title == "" {
				die("--title 必填")
			}
			body := map[string]any{"title": title, "priority": priority}
			if due != "" {
				body["due_date"] = due
			}
			if desc != "" {
				body["description"] = desc
			}
			respBody, err := userPost("/api/v1/tasks", body)
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
			outRaw(respBody, "✅ 待办已创建 (ID: %d)\n", resp.Data.ID)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "待办标题（必填）")
	cmd.Flags().StringVar(&due, "due", "", "截止日期 YYYY-MM-DD（可选）")
	cmd.Flags().StringVar(&priority, "priority", "medium", "优先级: low|medium|high")
	cmd.Flags().StringVar(&desc, "desc", "", "描述（可选）")
	return cmd
}

func newTaskUpdateCmd() *cobra.Command {
	var status, priority, title, due string
	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "改待办",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if !isNumeric(id) {
				die("待办 ID 必须是数字: %s", id)
			}
			body := map[string]any{}
			if status != "" {
				body["status"] = status
			}
			if priority != "" {
				body["priority"] = priority
			}
			if title != "" {
				body["title"] = title
			}
			if due != "" {
				body["due_date"] = due
			}
			if len(body) == 0 {
				die("至少指定一个要修改的字段")
			}
			respBody, err := userPut("/api/v1/tasks/"+id, body)
			if err != nil {
				die("%v", err)
			}
			outRaw(respBody, "✅ 待办 #%s 已更新\n", id)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "新状态: todo|doing|done")
	cmd.Flags().StringVar(&priority, "priority", "", "新优先级: low|medium|high")
	cmd.Flags().StringVar(&title, "title", "", "新标题")
	cmd.Flags().StringVar(&due, "due", "", "新截止日期 YYYY-MM-DD")
	return cmd
}
