package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// TestBuildContextBlocks_Image 仅 Quoted.Kind=QuotedImage（图片成功读取）时，产出携带
// 图片 image_url 的上下文块，且 imageURLFromMessage 能从中提取出 data URL。
func TestBuildContextBlocks_Image(t *testing.T) {
	const dataURL = "data:image/png;base64,aaaa"
	blocks := buildContextBlocks(&SmartReplyContext{
		Quoted: &QuotedContext{
			Kind:     QuotedImage,
			Name:     "x.png",
			ImageURL: dataURL,
			Text:     "📷 你引用了一张图片「x.png」，请识别其内容并结合用户的问题回答。",
		},
	})

	roles, contents := flattenRoles(blocks)
	if !anyContains(contents, "你引用了一张图片") {
		t.Fatalf("应产出图片提示语，got contents=%v", contents)
	}
	if !anyContains(contents, "我已读到被引用的文件内容") {
		t.Fatalf("成功场景应产出「已读到」应答，got contents=%v", contents)
	}
	if anyContains(contents, "未能读取") {
		t.Fatalf("成功场景不应出现「未能读取」应答，got contents=%v", contents)
	}
	// 图片 data URL 应走在 MultiContent 的 image_url 部分而非平铺文本，防止 data URL 进入模型文本。
	var gotURL string
	for _, b := range blocks {
		if u := imageURLFromMessage(b); u != "" {
			gotURL = u
		}
	}
	if gotURL != dataURL {
		t.Fatalf("应从上下文块提取出图片 data URL，got %q", gotURL)
	}
	_ = roles
}

// TestBuildContextBlocks_ImageFailed 图片读取失败（Quoted.Kind=QuotedFailed）时，
// 只产出「未能读取」应答，不产出图片 data URL 块。
func TestBuildContextBlocks_ImageFailed(t *testing.T) {
	blocks := buildContextBlocks(&SmartReplyContext{
		Quoted: &QuotedContext{
			Kind: QuotedFailed,
			Name: "x.png",
			Text: "📷 你引用了一条图片消息「x.png」，但该图片当前无法读入上下文。",
		},
	})
	_, contents := flattenRoles(blocks)
	if !anyContains(contents, "未能读取") {
		t.Fatalf("失败场景应产出「未能读取」应答，got %v", contents)
	}
	for _, b := range blocks {
		if u := imageURLFromMessage(b); u != "" {
			t.Fatalf("失败场景不应有任何图片 data URL，got %q", u)
		}
	}
}

// TestEinoMessagesToAIMessages_Image 验证带图片的 schema.Message 能通过
// einoMessagesToAIMessages 把 MultiContent 里的 data URL 透传到 ai.Message.ImageURL。
func TestEinoMessagesToAIMessages_Image(t *testing.T) {
	const dataURL = "data:image/jpeg;base64,bbbb"
	in := []*schema.Message{
		{
			Role:    schema.User,
			Content: "📷 你引用了一张图片，请识别。",
			MultiContent: []schema.ChatMessagePart{
				{Type: schema.ChatMessagePartTypeText, Text: "📷 你引用了一张图片，请识别。"},
				{Type: schema.ChatMessagePartTypeImageURL, ImageURL: &schema.ChatMessageImageURL{URL: dataURL}},
			},
		},
		{Role: schema.User, Content: "普通文本消息"},
	}

	out := einoMessagesToAIMessages(in)
	require.Len(t, out, 2)
	assert.Equal(t, dataURL, out[0].ImageURL, "图片消息应透传 ImageURL")
	assert.Equal(t, "📷 你引用了一张图片，请识别。", out[0].Content, "文本 content 应保留")
	assert.Empty(t, out[1].ImageURL, "普通消息不应有 ImageURL")
	assert.Equal(t, "普通文本消息", out[1].Content)
}

// TestImageURLForContext 验证 GroupDocumentService 能把图片读取为 base64 data URL。
func TestImageURLForContext(t *testing.T) {
	db := setupFileServiceTestDB(t)
	accessor := newTestAccessor(t)
	svc := NewGroupDocumentService(db, accessor)

	ctx := context.Background()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00} // PNG signature
	sp, err := accessor.Put(ctx, "uploads/2026/08/pic.png", strings.NewReader(string(png)), int64(len(png)), "image/png")
	require.NoError(t, err)

	f := &model.File{UserID: 1, ScopeType: "user", ScopeID: 1, Name: "pic.png", OriginalName: "pic.png", StoragePath: sp, Size: int64(len(png)), MimeType: "image/png"}
	require.NoError(t, db.Create(f).Error)

	name, dataURL, err := svc.ImageURLForContext(f.ID)
	require.NoError(t, err)
	assert.Equal(t, "pic.png", name)
	assert.True(t, strings.HasPrefix(dataURL, "data:image/png;base64,"), "应为 png data URL，got %q", dataURL)

	// 两次读取结果应一致（确定性）
	_, dataURL2, err := svc.ImageURLForContext(f.ID)
	require.NoError(t, err)
	assert.Equal(t, dataURL, dataURL2)
}

// TestImageURLForContext_TooLarge 超大图片应返回哨兵错误 ErrQuotedImageTooLarge。
func TestImageURLForContext_TooLarge(t *testing.T) {
	db := setupFileServiceTestDB(t)
	accessor := newTestAccessor(t)
	svc := NewGroupDocumentService(db, accessor)

	ctx := context.Background()
	sp, err := accessor.Put(ctx, "uploads/2026/08/big.png", strings.NewReader(strings.Repeat("x", int(quoteMaxImageSize)+1)), quoteMaxImageSize+1, "image/png")
	require.NoError(t, err)

	f := &model.File{UserID: 1, ScopeType: "user", ScopeID: 1, Name: "big.png", OriginalName: "big.png", StoragePath: sp, Size: quoteMaxImageSize + 1, MimeType: "image/png"}
	require.NoError(t, db.Create(f).Error)

	_, _, err = svc.ImageURLForContext(f.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrQuotedImageTooLarge), "应返回 ErrQuotedImageTooLarge，got %v", err)
}

// TestImageURLForContext_Missing 图片记录查不到时返回错误。
func TestImageURLForContext_Missing(t *testing.T) {
	db := setupFileServiceTestDB(t)
	svc := NewGroupDocumentService(db, newTestAccessor(t))
	_, _, err := svc.ImageURLForContext(9999)
	require.Error(t, err)
}

// 编译期断言：GroupDocumentService 实现扩展后的 QuotedDocumentReader（含图片读取）。
var _ QuotedDocumentReader = (*GroupDocumentService)(nil)

// setupQuotedTestDB 构造 prepareInput 引用/自引用读取所需的 message + file + conversation 表（内存 SQLite）。
func setupQuotedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.Message{}, &model.File{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// newQuotedGraph 用静默日志的 db 会话构造 graph：prepareInput 补齐上下文的查询（groups/tasks/users）
// 在仅迁移 message/file/conversation 的测试库里必然未命中，属预期、按空上下文容忍，
// 静默避免把误报刷进测试输出。
func newQuotedGraph(db *gorm.DB, reader QuotedDocumentReader) *SmartReplyGraph {
	silent := db.Session(&gorm.Session{Logger: gormLogger.Default.LogMode(gormLogger.Silent)})
	return &SmartReplyGraph{db: silent, quotedFile: reader}
}

// TestPrepareInput_SelfQuoteImage 直接发图（不引用）自引用时，prepareInput 应像显式引用一样
// 走多模态：注入 base64 data URL、跳过知识检索，且措辞主语为"用户发送了"而非"你引用了"。
func TestPrepareInput_SelfQuoteImage(t *testing.T) {
	db := setupQuotedTestDB(t)
	require.NoError(t, db.Create(&model.Conversation{ID: 1, Type: "group"}).Error)
	accessor := newTestAccessor(t)
	docSvc := NewGroupDocumentService(db, accessor)

	ctx := context.Background()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00} // PNG signature
	sp, err := accessor.Put(ctx, "uploads/2026/08/self.png", strings.NewReader(string(png)), int64(len(png)), "image/png")
	require.NoError(t, err)

	f := &model.File{UserID: 1, ScopeType: "user", ScopeID: 1, Name: "self.png", OriginalName: "self.png", StoragePath: sp, Size: int64(len(png)), MimeType: "image/png"}
	require.NoError(t, db.Create(f).Error)

	msg := &model.Message{ConversationID: 1, SenderID: 1, Type: "image", Content: fmt.Sprintf(`{"url":"/static/self.png","id":%d,"name":"self.png"}`, f.ID)}
	require.NoError(t, db.Create(msg).Error)

	sg := newQuotedGraph(db, docSvc)
	input := &SmartReplyContext{
		Message:         "请识别用户发送的这张图片，并结合上下文回复。",
		UserID:          1,
		ConversationID:  1,
		IsAIMention:     true,
		QuotedMessageID: &msg.ID,
		IsSelfQuote:     true,
	}
	require.NoError(t, sg.prepareInput(input))

	require.NotNil(t, input.Quoted, "自引用图片应产生被引用上下文")
	assert.Equal(t, QuotedImage, input.Quoted.Kind)
	assert.True(t, strings.HasPrefix(input.Quoted.ImageURL, "data:image/png;base64,"), "应注入 base64 data URL，got %q", input.Quoted.ImageURL)
	assert.Contains(t, input.Quoted.Text, "用户发送了一张图片", "自引用措辞应为用户发送，got %q", input.Quoted.Text)
	assert.NotContains(t, input.Quoted.Text, "你引用了", "自引用不应出现引用措辞")
	assert.Empty(t, input.KnowledgeCtx, "图片自引用应跳过知识检索")
	assert.Empty(t, input.MemoryCtx, "图片自引用应跳过记忆检索")
}

// TestPrepareInput_SelfQuoteFile 直接发文件（不引用）自引用时，prepareInput 读取正文注入上下文，
// 措辞主语为"用户发送的文件"。
func TestPrepareInput_SelfQuoteFile(t *testing.T) {
	db := setupQuotedTestDB(t)
	require.NoError(t, db.Create(&model.Conversation{ID: 1, Type: "group"}).Error)
	accessor := newTestAccessor(t)
	docSvc := NewGroupDocumentService(db, accessor)

	ctx := context.Background()
	const body = "你好，这是用户直接发送的文件正文内容。"
	sp, err := accessor.Put(ctx, "uploads/2026/08/self.txt", strings.NewReader(body), int64(len(body)), "text/plain")
	require.NoError(t, err)

	f := &model.File{UserID: 1, ScopeType: "user", ScopeID: 1, Name: "self.txt", OriginalName: "self.txt", StoragePath: sp, Size: int64(len(body)), MimeType: "text/plain"}
	require.NoError(t, db.Create(f).Error)

	msg := &model.Message{ConversationID: 1, SenderID: 1, Type: "file", Content: fmt.Sprintf(`{"url":"/static/self.txt","id":%d,"name":"self.txt"}`, f.ID)}
	require.NoError(t, db.Create(msg).Error)

	sg := newQuotedGraph(db, docSvc)
	input := &SmartReplyContext{
		Message:         "请阅读用户发送的这份文件的内容，并结合上下文回复。",
		UserID:          1,
		ConversationID:  1,
		IsAIMention:     true,
		QuotedMessageID: &msg.ID,
		IsSelfQuote:     true,
	}
	require.NoError(t, sg.prepareInput(input))

	require.NotNil(t, input.Quoted, "自引用文件应产生被引用上下文")
	assert.Equal(t, QuotedFile, input.Quoted.Kind)
	assert.Contains(t, input.Quoted.Text, "用户发送的文件", "自引用措辞应为用户发送，got %q", input.Quoted.Text)
	assert.Contains(t, input.Quoted.Text, body, "应注入文件正文")
	assert.NotContains(t, input.Quoted.Text, "你引用了", "自引用不应出现引用措辞")
}
