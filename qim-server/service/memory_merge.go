package service

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
)

// MemoryMergeKind 新记忆相对最相关旧记忆应如何处理。分身与群记忆共用同一套判定，
// 避免两套保存路径各自实现、行为不一致。
type MemoryMergeKind int

const (
	// MemoryMergeNew 新记忆与旧记忆是不同事实（或相似度未达门槛）→ 应新增一条。
	MemoryMergeNew MemoryMergeKind = iota
	// MemoryMergeConflict 新记忆与旧记忆讨论同一主题但结论/事实矛盾 → 用新的更新旧的（保留 memoryID）。
	MemoryMergeConflict
	// MemoryMergeDuplicate 新记忆是旧记忆的复述/确认（同事实不同措辞、结论一致）→
	// 不新增，仅刷新旧记忆（防止近义重复记忆挤占召回 TopK、口径不一让回复摇摆）。
	MemoryMergeDuplicate
)

var memoryKindRe = regexp.MustCompile(`"kind"\s*:\s*"([a-z]+)"`)

// memoryMergePrompt 构造"新记忆相对旧记忆应如何合并"的三分类提示。
// 同时承载原冲突检测与近义去重：冲突→更新，复述→合并，不同→新增，一次 LLM 调用决定两种行为。
func memoryMergePrompt(newMemo, oldMemo string) string {
	var b strings.Builder
	b.WriteString("判断下面两条记忆的关系，仅返回 JSON，形如 {\"kind\": \"new\"}。kind 取值：\n")
	b.WriteString("- new：不同事实（应作为新记忆分别存储）\n")
	b.WriteString("- conflict：同一件事，但结论或事实相互矛盾（应更新旧记忆）\n")
	b.WriteString("- duplicate：同一件事，只是换个说法或再次确认、结论一致（应合并旧记忆，不新增）\n")
	b.WriteString("参考：'我喜欢美式咖啡' vs '我习惯喝美式' 是 duplicate；'用MySQL' vs '改用PostgreSQL' 是 conflict；'喜欢美式' vs '怕吃辣' 是 new。\n\n")
	b.WriteString("新记忆：\n" + newMemo + "\n\n")
	b.WriteString("旧记忆：\n" + oldMemo + "\n")
	return b.String()
}

// inferMemoryMergeKind 用 LLM 判定新记忆相对最相关旧记忆的合并关系。
// aiService 为 nil 时返回 MemoryMergeNew（未知即新建，安全方向：不误并、不误删）。
// 解析失败同样按 MemoryMergeNew 处理，避免降级路径误更新旧记忆。
func inferMemoryMergeKind(aiService *ai.AIService, newMemo, oldMemo string) (MemoryMergeKind, error) {
	if aiService == nil {
		return MemoryMergeNew, nil
	}
	out, err := aiService.GetCompletion(ai.TaskTypeAnalysis, []ai.Message{{Role: "user", Content: memoryMergePrompt(newMemo, oldMemo)}})
	if err != nil {
		return MemoryMergeNew, err
	}
	m := memoryKindRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return MemoryMergeNew, nil
	}
	switch m[1] {
	case "conflict":
		return MemoryMergeConflict, nil
	case "duplicate":
		return MemoryMergeDuplicate, nil
	default:
		return MemoryMergeNew, nil
	}
}

// mergedImportance 计算"合并/更新旧记忆"时采用的重要度：取旧档位与新档位较大者，
// 防止复述一遍却把旧记忆的重要度(5)降为新档位(3)。
func mergedImportance(oldMeta map[string]string, new float64) float64 {
	merged := new
	if v, err := parseImportanceMeta(oldMeta["importance"]); err == nil && v > merged {
		merged = v
	}
	return merged
}

// parseImportanceMeta 解析 metadata 里以 "%.1f"（1-5 档位）存储的重要度字符串。
func parseImportanceMeta(s string) (float64, error) {
	if s == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(s, 64)
}