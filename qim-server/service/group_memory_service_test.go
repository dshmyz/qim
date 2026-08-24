package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/types"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
)

// GroupMemoryService 在向量库/AI 服务未配置时必须安全降级为 no-op，
// 不能让群助手回复路径因记忆服务缺失而报错。以下用 nil db / nil aiService 验证。

func newNilGroupMemoryService() *GroupMemoryService {
	return &GroupMemoryService{db: nil, aiService: nil}
}

func TestGroupMemoryService_NilDBSafeNoOp(t *testing.T) {
	s := newNilGroupMemoryService()

	assert.NoError(t, s.Remember(1, 2, "群决定：每周五例会", 3), "Remember 在 db=nil 时应 no-op 不报错")

	results, err := s.Recall(1, "例会", 5)
	assert.NoError(t, err)
	assert.Empty(t, results, "Recall 在 db=nil 时应返回空")

	memories, err := s.GetGroupMemories(1, 50)
	assert.NoError(t, err)
	assert.Empty(t, memories, "GetGroupMemories 在 db=nil 时应返回空")

	count, err := s.GetMemoryCount(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	assert.NoError(t, s.DeleteMemory(1, "any"), "DeleteMemory 在 db=nil 时应 no-op")

	deleted, err := s.ForgetAll(1)
	assert.NoError(t, err)
	assert.Equal(t, 0, deleted, "ForgetAll 在 db=nil 时应删 0 条")
}

func TestGroupMemoryService_ShouldRememberNilAI(t *testing.T) {
	s := &GroupMemoryService{db: nil, aiService: nil}
	should, err := s.ShouldRemember("群决定：下周三发布")
	assert.NoError(t, err)
	assert.False(t, should, "aiService=nil 时 ShouldRemember 应降级为 false（不记），避免误存")
}

// 隔离契约：群记忆的 namespace 必须与分身记忆（"avatar"）不同，
// 且 UserID 用 groupID 键--这条契约保证两套记忆不串。改动 namespace 需同步迁移数据。
func TestGroupMemoryService_NamespaceIsolation(t *testing.T) {
	assert.Equal(t, "group_assistant", groupMemoryNamespace, "群记忆 namespace 必须与分身(avatar)隔离")
	assert.NotEqual(t, "avatar", groupMemoryNamespace, "群记忆不得复用分身 namespace")
}

// 编译期保证 NewGroupMemoryService 在 vectorSvc=nil 时不 panic 且返回可用的 no-op 服务。
func TestNewGroupMemoryService_NilVectorSvc(t *testing.T) {
	s := NewGroupMemoryService(nil, new(ai.AIService))
	assert.NotNil(t, s)
	assert.Nil(t, s.db, "vectorSvc=nil 时 db 应为 nil（no-op 模式）")
}

// TestGroupMemoryService_Consolidate_DuplicateRefreshesNotAppends 验证群记忆近义复述（R1 duplicate）：
// 同一决定被再次确认/换个说法时，不新增重复群记忆，仅刷新旧记忆重要度，防近义重复占位。
func TestGroupMemoryService_Consolidate_DuplicateRefreshesNotAppends(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()

	svc := &GroupMemoryService{db: db, aiService: nil}
	svc.SetMergeCheck(func(_, _ string) (MemoryMergeKind, error) { return MemoryMergeDuplicate, nil })

	oldID := "groupmem_5_1"
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: oldID, UserID: "5", Scope: "user", Namespace: groupMemoryNamespace,
		Content: "项目截止日期是3月15日", Importance: 0.4,
		Metadata: map[string]interface{}{"importance": "5.0"},
	}); err != nil {
		t.Fatalf("预置群记忆失败: %v", err)
	}

	ref := MemoryReflection{Summary: "项目截止日期是3月15日（已在群里再次确认）", Importance: 2}
	memories := []SearchResult{
		{Content: "项目截止日期是3月15日", Score: 0.85, DocID: oldID, Metadata: map[string]string{"importance": "5.0"}},
	}

	// 传入发言归属（senderID/senderName）验证 P2：落库应记录"谁说的"。
	// 注意：换一个 sender 且近义复述——R1 判定为 dup，应保留首提者（不改变原 metadata）。
	ok, err := svc.saveConsolidatedGroupMemory(5, 1, 999, "小王", ref, memories)
	if err != nil {
		t.Fatalf("saveConsolidatedGroupMemory 失败: %v", err)
	}
	if !ok {
		t.Fatal("近义复述应返回 ok=true（刷新了旧群记忆）")
	}

	if count := countMemories(db, "5", groupMemoryNamespace); count != 1 {
		t.Fatalf("近义复述应合并而非新增，期望 1 条，got %d", count)
	}
	rec, err := loadMemoryRecord(db, "5", groupMemoryNamespace, oldID)
	if err != nil {
		t.Fatalf("读取群记忆失败: %v", err)
	}
	if !strings.Contains(rec.Content, "3月15日") {
		t.Fatalf("近义复述不应改动旧内容，got %q", rec.Content)
	}
	if rec.Importance != 1.0 {
		t.Fatalf("群记忆重要度应刷新为 max(2,5)=5→1.0，got %v", rec.Importance)
	}
}

// TestGroupMemoryService_Consolidate_StoresSender 验证 P2：新增群记忆时落库 sender_id/sender_name。
func TestGroupMemoryService_Consolidate_StoresSender(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()

	svc := &GroupMemoryService{db: db, aiService: nil}
	ref := MemoryReflection{Summary: "团队决定下周一起用新部署流程", Importance: 4}

	ok, err := svc.saveConsolidatedGroupMemory(5, 1, 42, "李四", ref, nil)
	if err != nil {
		t.Fatalf("saveConsolidatedGroupMemory 失败: %v", err)
	}
	if !ok {
		t.Fatal("新记忆应成功落库")
	}

	recs, err := db.SearchMemory(types.MemorySearchRequest{UserID: "5", Scope: "user", Namespace: groupMemoryNamespace, TopK: 10})
	if err != nil {
		t.Fatalf("查询群记忆失败: %v", err)
	}
	if got := len(recs.Results); got != 1 {
		t.Fatalf("期望写入 1 条，got %d", got)
	}
	md := recs.Results[0].Memory.Metadata
	if md["sender_id"] != "42" {
		t.Fatalf("sender_id 期望 42，got %v", md["sender_id"])
	}
	if md["sender_name"] != "李四" {
		t.Fatalf("sender_name 期望 李四，got %v", md["sender_name"])
	}
}

// TestGroupMemory_UpdateMemory_CorrectsAndCrossGroupDenied 验证群记忆显式纠正接口：
// 本群可纠正记忆内容；其他群不能纠正（防跨群越权）。
func TestGroupMemory_UpdateMemory_CorrectsAndCrossGroupDenied(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	svc := &GroupMemoryService{db: db, aiService: nil}

	// group9 存一条记忆（群记忆 namespace 是 group_assistant，UserID 是 groupID）
	memID := "groupmem_9_1"
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: memID, UserID: "9", Scope: "user", Namespace: groupMemoryNamespace,
		Content: "项目截止日期是3月15日", Importance: 0.8,
	}); err != nil {
		t.Fatalf("预置群记忆失败: %v", err)
	}

	// 本群 group9 纠正成功
	if err := svc.UpdateMemory(9, memID, "项目截止日期已改为3月20日"); err != nil {
		t.Fatalf("本群 UpdateMemory 应成功: %v", err)
	}
	rec, err := loadMemoryRecord(db, "9", groupMemoryNamespace, memID)
	if err != nil {
		t.Fatalf("读取纠正后的群记忆失败: %v", err)
	}
	if !strings.Contains(rec.Content, "3月20日") {
		t.Fatalf("群记忆内容应更新为 3月20日，got %q", rec.Content)
	}

	// 其他群 group8 尝试纠正 group9 的记忆 → 拒绝
	err = svc.UpdateMemory(8, memID, "跨群篡改")
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("跨群纠正应返回 ErrMemoryNotFound，got %v", err)
	}
}

// TestGroupMemoryService_ConflictThreshold
// 群记忆冲突检测门槛默认 0.3；注入阈值服务后经 GetFloat 读取（nil DB 回退默认值 0.3）。
func TestGroupMemoryService_ConflictThreshold(t *testing.T) {
	svc := &GroupMemoryService{}
	if got := svc.conflictThreshold(); got != 0.3 {
		t.Errorf("默认冲突门槛应为 0.3，got %v", got)
	}

	svc.SetThresholdService(NewAiThresholdService(nil))
	if got := svc.conflictThreshold(); got != 0.3 {
		t.Errorf("注入阈值服务后应读回默认 0.3，got %v", got)
	}
}
