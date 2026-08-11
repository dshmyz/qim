package service

import (
	"context"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"gorm.io/gorm"
)

// embedFakeProvider 测试用 Provider：Embedding 返回确定性伪向量（fakeVec），
// 使 NoteVectorService 的向量化与检索在无外部 AI 时也能端到端命中。Chat 系列置空实现。
type embedFakeProvider struct{}

var _ ai.Provider = (*embedFakeProvider)(nil)

func (embedFakeProvider) Name() string { return "fake-embed" }
func (embedFakeProvider) Chat(messages []ai.Message) (string, error) {
	return "", nil
}
func (embedFakeProvider) ChatStream(messages []ai.Message, onChunk func(chunk ai.StreamChunk) error) error {
	return nil
}
func (embedFakeProvider) ChatStreamWithContext(ctx context.Context, messages []ai.Message, onChunk func(chunk ai.StreamChunk) error) error {
	return nil
}
func (embedFakeProvider) Embedding(text string) ([]float32, error) {
	return fakeVec(text), nil
}
func (embedFakeProvider) SupportsEmbedding() bool { return true }
func (embedFakeProvider) ChatWithTools(messages []ai.Message, tools []ai.ToolDef) (*ai.ChatResponse, error) {
	return nil, nil
}
func (embedFakeProvider) ChatStreamWithTools(ctx context.Context, messages []ai.Message, tools []ai.ToolDef, onChunk func(chunk ai.StreamChunk) error) error {
	return ai.ErrStreamingToolsNotSupported
}
func (embedFakeProvider) IsConfigured() bool { return true }
func (embedFakeProvider) WithModel(model string) ai.Provider {
	return embedFakeProvider{}
}

// newFakeNoteService 构造带真实内存 gracedb + 伪嵌入的 NoteService，用于端到端
// 验证「逐笔记 AI 可见性门控」：可见→向量化进集合可召回；不可见→移除向量不可召回。
func newFakeNoteService(t *testing.T) *NoteService {
	t.Helper()

	// 笔记本体数据库：SQLite 内存库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	if err := db.AutoMigrate(&model.Note{}); err != nil {
		t.Fatalf("迁移 Note 表失败: %v", err)
	}

	// 向量库：临时 gracedb + fakeEmbedder
	gdb, err := gracedb.Open(t.TempDir()+"/vec", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	t.Cleanup(func() { gdb.Close() })

	// 伪嵌入 AIService
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-embed", embedFakeProvider{})

	noteVecSvc := &NoteVectorService{vectorSvc: &VectorService{db: gdb}, aiService: aiSvc}
	noteSvc := NewNoteService(db)
	noteSvc.SetVectorService(noteVecSvc)
	return noteSvc
}

// waitNoteSearchHit 轮询 SearchNotes 直至命中状态达到 want。向量化走 SafeGo 异步，
// 且 gracedb 集合在首次写入前不存在（Search 返回 "gracedb: not found"），故检索报错时
// 视为「尚未就绪」继续重试，仅在超时后报告最后一次错误/超时——避免异步时序导致偶发失败。
func waitNoteSearchHit(t *testing.T, svc *NoteService, query string, want bool) {
	t.Helper()
	var lastErr error
	for i := 0; i < 200; i++ {
		results, err := svc.noteVectorSvc.SearchNotes(1, query, 3)
		if err != nil {
			lastErr = err
			time.Sleep(5 * time.Millisecond)
			continue
		}
		if (len(results) > 0) == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("SearchNotes 持续失败: %v (query=%q wantHit=%v)", lastErr, query, want)
	}
	t.Fatalf("等待命中结果超时: query=%q wantHit=%v", query, want)
}

// TestSetNoteAiAccessible 端到端验证逐笔记可见性门控：
//
//	true  → 笔记向量化进集合，SearchNotes 能召回
//	false → 移出集合，SearchNotes 召回为空
//	true  → 重新向量化，SearchNotes 又能召回
func TestSetNoteAiAccessible_GatesVectorSearch(t *testing.T) {
	svc := newFakeNoteService(t)

	// 创建后默认 AiAccessible=true，应已被向量化
	note := &model.Note{UserID: 1, Title: "旅游攻略", Content: "想去日本看樱花，住京都，五月出发。"}
	if err := svc.CreateNote(note); err != nil {
		t.Fatalf("CreateNote 失败: %v", err)
	}
	if !note.AiAccessible {
		t.Fatal("新建笔记 AiAccessible 应为 true")
	}

	waitNoteSearchHit(t, svc, "日本樱花", true) // 可见 → 可召回

	// 关闭可见性 → 向量移除 → 不可召回
	updated, err := svc.SetNoteAiAccessible(1, note.ID, false)
	if err != nil {
		t.Fatalf("SetNoteAiAccessible(false) 失败: %v", err)
	}
	if updated.AiAccessible {
		t.Fatal("关闭后 AiAccessible 应为 false")
	}
	waitNoteSearchHit(t, svc, "日本樱花", false)

	// 重新打开 → 重新向量化 → 可召回
	if _, err := svc.SetNoteAiAccessible(1, note.ID, true); err != nil {
		t.Fatalf("SetNoteAiAccessible(true) 失败: %v", err)
	}
	waitNoteSearchHit(t, svc, "日本樱花", true)
}

// TestSetNoteAiAccessible_NotFound 验证对不存在/无归属的笔记返回 gorm.ErrRecordNotFound。
func TestSetNoteAiAccessible_NotFound(t *testing.T) {
	svc := newFakeNoteService(t)

	note := &model.Note{UserID: 1, Title: "私有笔记", Content: "只给自己看的草稿"}
	if err := svc.CreateNote(note); err != nil {
		t.Fatalf("CreateNote 失败: %v", err)
	}

	// 他人无权访问
	if _, err := svc.SetNoteAiAccessible(2, note.ID, false); err != gorm.ErrRecordNotFound {
		t.Errorf("非本人操作应返回 ErrRecordNotFound, got %v", err)
	}
	// 不存在的笔记
	if _, err := svc.SetNoteAiAccessible(1, 99999, false); err != gorm.ErrRecordNotFound {
		t.Errorf("不存在的笔记应返回 ErrRecordNotFound, got %v", err)
	}
}

// TestUpdateNote_EmptyContentRemovesVectors 验证可见状态下清空内容也会移除向量，
// 避免旧内容残留在集合里被分身检索到。
func TestUpdateNote_EmptyContentRemovesVectors(t *testing.T) {
	svc := newFakeNoteService(t)

	note := &model.Note{UserID: 1, Title: "出游计划", Content: "初始内容：准备周三去郊外爬山露营。"}
	if err := svc.CreateNote(note); err != nil {
		t.Fatalf("CreateNote 失败: %v", err)
	}
	waitNoteSearchHit(t, svc, "爬山露营", true)

	// 清空内容（可见性不变）→ 向量应被移除
	note.Content = ""
	if err := svc.UpdateNote(note); err != nil {
		t.Fatalf("UpdateNote 失败: %v", err)
	}
	waitNoteSearchHit(t, svc, "爬山露营", false)
}

// TestUpdateNote_SyncsVectorOnVisibility 验证 UpdateNote 也会按可见性同步向量：
// 内容更新时可见仍可召回；改不可见后即刻不可召回。
func TestUpdateNote_SyncsVectorOnVisibility(t *testing.T) {
	svc := newFakeNoteService(t)

	note := &model.Note{UserID: 1, Title: "会议纪要", Content: "Q3 产品路线：重点做 AI 助手与笔记知识库。"}
	if err := svc.CreateNote(note); err != nil {
		t.Fatalf("CreateNote 失败: %v", err)
	}

	waitNoteSearchHit(t, svc, "产品路线", true)

	// 关闭可见性后更新内容（模拟用户在不可见状态下修改），向量应被移除
	note.AiAccessible = false
	note.Content = "已改：Q3 重点是客户服务。"
	if err := svc.UpdateNote(note); err != nil {
		t.Fatalf("UpdateNote 失败: %v", err)
	}
	waitNoteSearchHit(t, svc, "产品路线", false)

	// 重新打开并更新 → 重新向量化新内容 → 可召回新内容
	note.AiAccessible = true
	if err := svc.UpdateNote(note); err != nil {
		t.Fatalf("UpdateNote(open) 失败: %v", err)
	}
	waitNoteSearchHit(t, svc, "客户服务", true)
}
