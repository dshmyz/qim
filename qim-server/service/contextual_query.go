package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/dshmyz/qim/qim-server/model"
)

// 上下文感知检索（Contextual Retrieval）共享助手。
//
// 业界多轮 RAG 的已知缺口：向量检索的 query 若只用"最后一条消息"，追问（"那后来呢？"、
// "具体说说"）没有话题上下文，召回必然失败。这里把最近对话历史拼进 query，让 embedding
// 携带话题信息。纯字符串拼接、不调 LLM，代价可忽略；分身与群助手共用。

// truncateRunes 按 rune 截断字符串到 maxRunes，超长加省略号。中文等多字节字符按码点切，
// 避免从汉字中间截断产生非法 UTF-8 或乱码。空串/未超长原样返回。
func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes]) + "…"
}

// contextualQuery 构造上下文感知检索查询：把最近对话历史（时间正序 "sender: content" 行）
// 拼进 query，使多轮追问在向量检索时能借助话题上下文命中。recent 控制取最近多少行历史；
// 历史为空时退化为原 query。trigger 为空时用占位符（正常情况下由调用方保证非空）。
func contextualQuery(history, trigger string, recent int) string {
	lines := strings.Split(history, "\n")
	if len(lines) > recent && recent > 0 {
		lines = lines[len(lines)-recent:]
	}
	prefix := strings.TrimSpace(strings.Join(lines, "\n"))
	if prefix == "" {
		return trigger
	}
	if strings.TrimSpace(trigger) == "" {
		trigger = "当前提问"
	}
	return prefix + "\n当前提问：" + trigger
}

// fetchRecentHistoryForQuery 拉取会话最近若干条文本消息拼成 "sender: content" 行（时间正序），
// 供上下文感知检索构造查询。排除触发消息本身（content+sender 都匹配才排除，避免误删同名消息）。
// 单条内容按 truncateRunes 截断，防止把超长消息整条塞进 query 膨胀 embedding 输入。
func fetchRecentHistoryForQuery(db *gorm.DB, conversationID, senderID uint, originalContent string) string {
	if db == nil || conversationID == 0 {
		return ""
	}
	var messages []model.Message
	db.Where("conversation_id = ? AND type IN ?", conversationID, []string{"text", "markdown"}).
		Preload("Sender").
		Order("created_at DESC").
		Limit(6).
		Find(&messages)
	if len(messages) == 0 {
		return ""
	}
	// 反转为时间正序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	lines := make([]string, 0, len(messages))
	for _, m := range messages {
		if originalContent != "" && m.SenderID == senderID && m.Content == originalContent {
			continue
		}
		name := m.Sender.Nickname
		if name == "" {
			name = m.Sender.Username
		}
		lines = append(lines, fmt.Sprintf("%s: %s", name, truncateRunes(m.Content, 300)))
	}
	return strings.Join(lines, "\n")
}
