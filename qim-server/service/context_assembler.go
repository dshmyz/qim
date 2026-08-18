package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// ContextSourceType 上下文注入的来源类型。入口通过声明式 sources 列表指明「要注哪些源」，
// 取代各处散落的硬编码 if xx != nil { append } 块——新增知识源只需在 assembler 加一个分支
// + 一行 ContextSource 声明，所有接入入口自动可用。
type ContextSourceType string

const (
	// SourceHistory 会话最近消息。按 Key=conversationID scope，Output 为紧凑上下文块
	// （name: content 逐行），供侧边栏 current 模式注入。
	SourceHistory ContextSourceType = "history"
	// SourceNotes 用户笔记。按 Key=userID scope（内部走 NoteSearcher，天然按用户分集合，
	// 只能读到该用户自己的笔记）。命中源随 KnowledgeSources 下发供前端渲染「知识来源」。
	SourceNotes ContextSourceType = "notes"
	// SourceMemory 长期记忆。按 Key=userID scope。
	SourceMemory ContextSourceType = "memory"
	// SourceGroupDoc 群文档知识库。按 Key=groupID scope。
	SourceGroupDoc ContextSourceType = "group_doc"
)

// ContextSource 声明一个待注入的上下文源。
type ContextSource struct {
	Type ContextSourceType
	Key  uint // userID / conversationID / groupID，视 Type 而定
	TopK int
}

// ContextBundle 一次组装的结果：可直接 append 的消息块 + 命中的知识来源（供 Extra 持久化）。
type ContextBundle struct {
	Messages         []ai.Message
	KnowledgeSources []KnowledgeSource
}

// NoteSearcher 笔记检索接口（与 MessageService 共用同一接口，便于测试注入 mock）。
// 说明：SourceNotes 分支不直接复用 NewNoteRetriever（其依赖具体 *NoteVectorService），
// 而是走 NoteSearcher 接口——因为 MessageService 侧是以接口注入（di/container.go 注入
// *NoteVectorService 实现，测试注入 fake），用接口才能无缝接管现有 bot 笔记注入且不破坏单测。
type ContextAssembler struct {
	db           *gorm.DB
	noteSearcher NoteSearcher
	memorySvc    *AvatarMemoryService
	groupDocSvc  *GroupDocumentService
}

func NewContextAssembler(db *gorm.DB) *ContextAssembler {
	return &ContextAssembler{db: db}
}

// SetNoteSearcher 注入笔记检索服务。传 nil 即关闭该源（安全降级）。
func (a *ContextAssembler) SetNoteSearcher(searcher NoteSearcher) {
	a.noteSearcher = searcher
}

// SetGroupContextServices 注入长期记忆 + 群文档知识库服务。两者均可传 nil（对应源跳过）。
func (a *ContextAssembler) SetGroupContextServices(memorySvc *AvatarMemoryService, groupDocSvc *GroupDocumentService) {
	a.memorySvc = memorySvc
	a.groupDocSvc = groupDocSvc
}

// Assemble 按声明式 sources 并行检索各源并组装为可直接 append 的消息块。
// 每个源独立失败降级（一个源出错不影响其他），与现有「检索失败跳过」契约一致。
func (a *ContextAssembler) Assemble(ctx context.Context, query string, sources []ContextSource) *ContextBundle {
	bundle := &ContextBundle{}
	for _, src := range sources {
		switch src.Type {
		case SourceNotes:
			a.assembleNotes(ctx, query, src, bundle)
		case SourceMemory:
			a.assembleMemory(ctx, query, src, bundle)
		case SourceGroupDoc:
			a.assembleGroupDoc(ctx, query, src, bundle)
		case SourceHistory:
			a.assembleHistory(query, src, bundle)
		}
	}
	return bundle
}

// assembleNotes 关联入口：查询创建者笔记，产出 system 上下文消息 + KnowledgeSources。
// 文案与历史 bot 注入完全一致（行为零变）：标题/分数进 KnowledgeSources 供 Extra 透出，
// 不暴露笔记正文（避免响应体过大/泄漏）。
func (a *ContextAssembler) assembleNotes(ctx context.Context, query string, src ContextSource, bundle *ContextBundle) {
	if a.noteSearcher == nil {
		return
	}
	results, err := a.noteSearcher.SearchNotes(src.Key, query, src.TopK)
	if err != nil {
		logger.WithModule("ContextAssembler").Warn("笔记检索失败，跳过注入", "userID", src.Key, "error", err)
		return
	}
	if len(results) == 0 {
		return
	}

	parts := make([]string, 0, len(results))
	hitLogs := make([]string, 0, len(results))
	bundle.KnowledgeSources = make([]KnowledgeSource, 0, len(results))
	for _, r := range results {
		title := r.Metadata["title"]
		if title == "" {
			title = "未命名"
		}
		parts = append(parts, fmt.Sprintf("[笔记: %s]\n%s", title, r.Content))
		hitLogs = append(hitLogs, fmt.Sprintf("docID=%s title=%s score=%.4f", r.DocID, title, r.Score))
		bundle.KnowledgeSources = append(bundle.KnowledgeSources, KnowledgeSource{Title: title, Score: r.Score, Source: "notes", ID: r.DocID})
	}
	logger.WithModule("diag").Info("命中笔记",
		"userID", src.Key, "hits", len(results), "notes", strings.Join(hitLogs, " | "))

	bundle.Messages = append(bundle.Messages, ai.Message{
		Role: "system",
		Content: "以下是创建者的相关笔记，可作为回答参考（请基于笔记内容作答，" +
			"笔记未覆盖的问题按你的通用能力回答）：\n\n" +
			strings.Join(parts, "\n\n"),
	})
}

// assembleMemory 关联入口：查询用户长期记忆（未来源的示范分支——一行 ContextSource 即可接入）。
func (a *ContextAssembler) assembleMemory(ctx context.Context, query string, src ContextSource, bundle *ContextBundle) {
	if a.memorySvc == nil {
		return
	}
	retriever := NewMemoryRetriever(a.memorySvc, src.Key, src.TopK)
	docs, err := retriever.Retrieve(ctx, query)
	if err != nil || len(docs) == 0 {
		return
	}
	parts := contextDocsToParts("记忆", docs)
	bundle.Messages = append(bundle.Messages, ai.Message{
		Role:    "system",
		Content: "以下是用户长期记忆中可能与本次提问相关的内容，可作为回答参考：\n\n" + strings.Join(parts, "\n\n"),
	})
}

// assembleGroupDoc 关联入口：查询群文档知识库（未来源的示范分支）。
func (a *ContextAssembler) assembleGroupDoc(ctx context.Context, query string, src ContextSource, bundle *ContextBundle) {
	if a.groupDocSvc == nil {
		return
	}
	retriever := NewGroupDocRetriever(a.groupDocSvc, src.Key, src.TopK)
	docs, err := retriever.Retrieve(ctx, query)
	if err != nil || len(docs) == 0 {
		return
	}
	parts := contextDocsToParts("群文档", docs)
	bundle.Messages = append(bundle.Messages, ai.Message{
		Role:    "system",
		Content: "以下是群文档知识库中可能与本次提问相关的内容，可作为回答参考：\n\n" + strings.Join(parts, "\n\n"),
	})
}

// assembleHistory 关联入口：加载最近 TopK 条消息，产出紧凑上下文块（name: content 逐行，
// 时间正序）。语义与会话元信息（如群名）分离——群名属会话元信息，不在本源职责内，由调用方注入。
func (a *ContextAssembler) assembleHistory(query string, src ContextSource, bundle *ContextBundle) {
	if a.db == nil || src.Key == 0 {
		return
	}
	topK := src.TopK
	if topK <= 0 {
		topK = 20
	}

	var messages []model.Message
	a.db.Where("conversation_id = ? AND type IN ?", src.Key, []string{"text", "markdown"}).
		Preload("Sender").
		Order("created_at DESC").
		Limit(topK).
		Find(&messages)
	if len(messages) == 0 {
		return
	}
	// 反转为时间正序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		name := msg.Sender.Nickname
		if name == "" {
			name = msg.Sender.Username
		}
		lines = append(lines, fmt.Sprintf("%s: %s", name, msg.Content))
	}
	history := "【最近对话记录】\n" + strings.Join(lines, "\n") + "\n"

	bundle.Messages = append(bundle.Messages,
		ai.Message{Role: "user", Content: history},
		ai.Message{Role: "assistant", Content: "已了解当前会话上下文，请提问。"},
	)
}

// contextDocsToParts 把检索到的 schema.Document 转成带标题/内容的注入片段行。
func contextDocsToParts(kind string, docs []*schema.Document) []string {
	parts := make([]string, 0, len(docs))
	for _, d := range docs {
		title := ""
		if v, ok := d.MetaData["title"]; ok {
			title = fmt.Sprintf("%v", v)
		}
		if title == "" {
			title = "未命名"
		}
		parts = append(parts, fmt.Sprintf("[%s: %s]\n%s", kind, title, d.Content))
	}
	return parts
}
