package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newNotesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "note",
		Short: "管理笔记（长期记忆）",
	}
	cmd.AddCommand(newNoteListCmd(), newNoteGetCmd(), newNoteCreateCmd(), newNoteUpdateCmd())
	return cmd
}

func newNoteListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出正式笔记（排除便签）",
		Run: func(cmd *cobra.Command, args []string) {
			respBody, err := userGet("/api/v1/notes")
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data []struct {
					ID        uint64 `json:"id"`
					Title     string `json:"title"`
					Content   string `json:"content"`
					Type      string `json:"type"`
					Tags      string `json:"tags"`
					UpdatedAt string `json:"updated_at"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			var notes []struct {
				ID        uint64 `json:"id"`
				Title     string `json:"title"`
				Content   string `json:"content"`
				Type      string `json:"type"`
				Tags      string `json:"tags"`
				UpdatedAt string `json:"updated_at"`
			}
			for _, n := range resp.Data {
				if n.Type != "" && n.Type != "note" {
					continue // 只展示正式笔记，排除便签(type=sticky)
				}
				notes = append(notes, n)
			}
			if len(notes) > limit {
				notes = notes[:limit]
			}
			if len(notes) == 0 {
				fmt.Println("（无笔记）")
				return
			}
			for _, n := range notes {
				title := n.Title
				if title == "" {
					title = "(无标题)"
				}
				updated := n.UpdatedAt
				if len(updated) > 10 {
					updated = updated[:10]
				}
				tags := strings.Trim(jsonTags(n.Tags), "[]\"")
				tagStr := ""
				if tags != "" {
					tagStr = "  tags=" + tags
				}
				fmt.Printf("#%-4d %s  (%s)%s\n", n.ID, title, updated, tagStr)
			}
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "最多返回条数")
	return cmd
}

func newNoteGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [id]",
		Short: "获取单条笔记全文",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if !isNumeric(id) {
				die("笔记 ID 必须是数字: %s", id)
			}
			respBody, err := userGet("/api/v1/notes/" + id)
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data struct {
					ID      uint64 `json:"id"`
					Title   string `json:"title"`
					Content string `json:"content"`
					Tags    string `json:"tags"`
					Summary string `json:"summary"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			n := resp.Data
			fmt.Printf("标题: %s\n", n.Title)
			if n.Summary != "" {
				fmt.Printf("摘要: %s\n", n.Summary)
			}
			fmt.Printf("标签: %s\n", jsonTags(n.Tags))
			fmt.Println("---")
			fmt.Println(n.Content)
		},
	}
	return cmd
}

func newNoteCreateCmd() *cobra.Command {
	var title, content string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "建笔记",
		Run: func(cmd *cobra.Command, args []string) {
			if title == "" {
				die("--title 必填")
			}
			body := map[string]any{"title": title, "content": content}
			respBody, err := userPost("/api/v1/notes", body)
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
			outRaw(respBody, "✅ 笔记已创建 (ID: %d)\n", resp.Data.ID)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "笔记标题（必填）")
	cmd.Flags().StringVar(&content, "content", "", "笔记正文（可选）")
	return cmd
}

func newNoteUpdateCmd() *cobra.Command {
	var title, content string
	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "改笔记（整体覆盖标题/正文）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if !isNumeric(id) {
				die("笔记 ID 必须是数字: %s", id)
			}
			if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("content") {
				die("至少提供 --title 或 --content 之一")
			}
			// 后端 PUT 要求 title/content 一并提交，先拉取现有笔记合并
			currentBody, err := userGet("/api/v1/notes/" + id)
			if err != nil {
				die("获取笔记失败: %v", err)
			}
			var curResp struct {
				Data struct {
					Title   string `json:"title"`
					Content string `json:"content"`
				} `json:"data"`
			}
			if err := json.Unmarshal(currentBody, &curResp); err != nil {
				die("解析笔记失败: %v", err)
			}
			t, c := curResp.Data.Title, curResp.Data.Content
			if cmd.Flags().Changed("title") {
				t = title
			}
			if cmd.Flags().Changed("content") {
				c = content
			}
			respBody, err := userPut("/api/v1/notes/"+id, map[string]any{"title": t, "content": c})
			if err != nil {
				die("%v", err)
			}
			outRaw(respBody, "✅ 笔记 #%s 已更新\n", id)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "新标题")
	cmd.Flags().StringVar(&content, "content", "", "新正文")
	return cmd
}

// jsonTags 把后端返回的 JSON 数组字符串 tags 转成可读形式（去除引号与括号）。
func jsonTags(s string) string {
	if s == "" || s == "[]" {
		return "（无）"
	}
	return strings.TrimSpace(s)
}
