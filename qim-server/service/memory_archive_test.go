package service

import (
	"testing"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/store"
	"github.com/dshmyz/gracedb/pkg/types"
)

// TestLazyArchive_SoftHidesWeakIdle verifies the lazy-archive helper soft-hides
// weak, idle memories (Archived=true) while strong ones survive, exactly the
// "soft forget" we want for an ever-growing memory store:
//   - weak memory archived → hidden from default search/list enumeration
//   - strong memory kept → still returned
//
// Uses aggressive opts (MaxIdle=0, higher threshold) via the injectable
// implementation so the test doesn't need to backdate CreatedAt; the production
// wrapper passes conservative defaults. Mirrors gracedb's own archive test.
func TestLazyArchive_SoftHidesWeakIdle(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	resetArchiveCooldown()

	// 强记忆：高重要度，不应被归档
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: "m_strong", UserID: "1", Scope: "user", Namespace: "avatar",
		Content: "重要引用", Importance: 0.9,
	}); err != nil {
		t.Fatal(err)
	}
	// 弱记忆：低重要度，应被归档
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: "m_weak", UserID: "1", Scope: "user", Namespace: "avatar",
		Content: "随手记的琐事", Importance: 0.1,
	}); err != nil {
		t.Fatal(err)
	}

	// 通过可选桩触发一次归档：无闲置下限（MaxIdle=0），阈值 0.5（高于弱记忆的强度、
	// 低于强记忆的强度），这样弱记忆立刻符合、强记忆不触发。
	archived, err := lazyArchiveWeakMemoriesOpts(db, "1", "avatar", store.ArchiveOptions{
		StrengthThreshold: 0.5,
		MaxIdle:           0,
	})
	if err != nil {
		t.Fatalf("lazyArchiveWeakMemoriesOpts 失败: %v", err)
	}
	if archived != 1 {
		t.Errorf("archived = %d, want 1（仅弱记忆被 soft-hide）", archived)
	}

	// 归档后：弱记忆应从默认检索消失，强记忆仍在。
	records, err := (&AvatarMemoryService{db: db}).GetUserMemories(1, 100)
	if err != nil {
		t.Fatalf("GetUserMemories 失败: %v", err)
	}
	got := map[string]bool{}
	for _, r := range records {
		got[r.Content] = true
	}
	if !got["重要引用"] {
		t.Error("强记忆不应被归档，仍应在列表中")
	}
	if got["随手记的琐事"] {
		t.Error("弱记忆应被归档（soft-hide）并从默认列表消失")
	}
}

// TestLazyArchive_CooldownThrottles verifies a second immediate call within the
// cooldown window does not re-run the archive (returns 0, no re-scan/write),
// so repeatedly opening the panel / building the graph won't hot-path the scan.
func TestLazyArchive_CooldownThrottles(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	resetArchiveCooldown()

	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: "m_weak", UserID: "1", Scope: "user", Namespace: "avatar",
		Content: "琐事", Importance: 0.1,
	}); err != nil {
		t.Fatal(err)
	}

	// 第一次：正常执行（会归档一条）
	first, err := lazyArchiveWeakMemoriesOpts(db, "1", "avatar", store.ArchiveOptions{
		StrengthThreshold: 0.5, MaxIdle: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 立即再调一次：处于冷却期内，应被节流为 0（不重新扫描/写入）
	second, err := lazyArchiveWeakMemoriesOpts(db, "1", "avatar", store.ArchiveOptions{
		StrengthThreshold: 0.5, MaxIdle: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 {
		t.Fatalf("first = %d, want 1", first)
	}
	if second != 0 {
		t.Errorf("second = %d, want 0（冷却期内应节流）", second)
	}
}
