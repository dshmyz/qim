package qapi

// 合并转发与分享消息的线协议（对齐 qim-client）：
//
//	merged_forward: {"version":1,"title":"...","messages":[
//	    {"id":"...","type":"text","content":"...","senderName":"...","timestamp":123}]}
//	share: {"type":"file|note|sticky|message","id":"...","name":"...",
//	        "content":"...","originalContent":"...","originalMessage":{...}}
//
// 网页端转发非文本单条消息（图/文件/小程序/卡片）保持原类型原 content，
// 文本走 "[转发] " 前缀，多选合并走 merged_forward。这里的解析只覆盖
// 原生端要渲染与再转发的字段，未知的 JSON 字段忽略。

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MergedForwardItem 是合并转发里的一条原始消息（content 保持线格式原样）。
type MergedForwardItem struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Content    string `json:"content"`
	SenderName string `json:"senderName"`
	Timestamp  int64  `json:"timestamp"`
}

// MergedForward 是 merged_forward 消息的 content 解析结果。
type MergedForward struct {
	Version  int                 `json:"version"`
	Title    string              `json:"title"`
	Messages []MergedForwardItem `json:"messages"`
}

// ParseMergedForward 解析并校验 merged_forward content，形状不对返回 nil
// （对齐网页端 parseMergedForwardPayload：version 必须 1、title 非空、
// messages 每条字段齐全）。
func ParseMergedForward(content string) *MergedForward {
	var mf MergedForward
	if err := json.Unmarshal([]byte(content), &mf); err != nil {
		return nil
	}
	if mf.Version != 1 || strings.TrimSpace(mf.Title) == "" || len(mf.Messages) == 0 {
		return nil
	}
	for _, it := range mf.Messages {
		if it.ID == "" || it.Type == "" || it.SenderName == "" || it.Timestamp <= 0 {
			return nil
		}
	}
	return &mf
}

// BuildMergedForwardPayload 构造合并转发 content（发送侧）。
func BuildMergedForwardPayload(title string, items []MergedForwardItem) string {
	b, _ := json.Marshal(MergedForward{Version: 1, Title: strings.TrimSpace(title), Messages: items})
	return string(b)
}

// MergedForwardLabel 是合并转发消息在列表/气泡里的短标签。
func MergedForwardLabel(content string) string {
	if mf := ParseMergedForward(content); mf != nil {
		return fmt.Sprintf("[聊天记录] %s（%d条）", mf.Title, len(mf.Messages))
	}
	return "[聊天记录]"
}

// ItemPreview 把合并转发里的一条消息转成人话（对齐网页端 getMessagePreview）。
func ItemPreview(it MergedForwardItem) string {
	switch it.Type {
	case "text":
		if text, _ := DecodeMentions(it.Content); text != "" {
			return text
		}
		return "[文本]"
	case "image":
		return "[图片]"
	case "file":
		return "[文件] " + it.Content // content 多为文件名或 attachment JSON，先给原文
	case "video":
		return "[视频]"
	case "audio":
		return "[语音]"
	case "markdown":
		return "[富文本]"
	case "share":
		return "[分享] " + ShareCardName(it.Content)
	case "merged_forward":
		return "[聊天记录]"
	}
	if it.Type != "" {
		return "[" + it.Type + "]"
	}
	return "[消息]"
}

// ShareCard 是 share 消息的 content 解析结果。
type ShareCard struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	// Original 是原文（转发消息时是原始 content，便签分享时可能缺失）。
	Original string `json:"originalContent"`
}

// ParseShareCard 解析 share content，形状不对返回 nil。
func ParseShareCard(content string) *ShareCard {
	var sc ShareCard
	if err := json.Unmarshal([]byte(content), &sc); err != nil {
		return nil
	}
	if sc.Name == "" && sc.Content == "" {
		return nil
	}
	return &sc
}

// ShareCardName 从 share content 里抠显示名（列表预览用），失败返回空。
func ShareCardName(content string) string {
	if sc := ParseShareCard(content); sc != nil {
		return sc.Name
	}
	return ""
}

// ShareKindLabel 是分享类型的中文标签。
func ShareKindLabel(t string) string {
	switch t {
	case "file":
		return "文件"
	case "note":
		return "笔记"
	case "sticky":
		return "便签"
	case "message":
		return "消息"
	}
	if t == "" {
		return "内容"
	}
	return t
}
