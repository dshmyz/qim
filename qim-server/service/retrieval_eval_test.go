package service

import (
	"testing"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 检索评测 harness：在固定种子语料上度量笔记检索的 hit@1 / hit@3 / MRR。
//
// 作用：
//   - 回归守卫：任何检索改动（混合召回/RRF/阈值/宽窄）不得跌破种子语料的命中下限，
//     让"调阈值"从拍脑袋变成可度量的比较。
//   - 真实数据评测：把 seedNotes/golden 换成真实笔记与期望命中（或扩展为读 QIM_EVAL_GOLDEN
//     JSON），运行本测试即可看当前检索在真实语料上的命中率，据此调 ai.knowledge_score_threshold
//     等阈值。
//
// 指标：hit@k = 期望命中出现在 Top-k；MRR = 期望命中的倒数排名均值。
func TestRetrievalEval_NotesHybrid(t *testing.T) {
	// 种子语料：主题分明的笔记（向量与词法特征都易区分，保证评测基线稳定）
	seedNotes := []struct{ title, content string }{
		{"项目A进度", "项目A截止日期是3月15日，负责人张三，目前进度正常"},
		{"东京旅行攻略", "东京五日行程：浅草寺、秋叶原、箱根温泉，预算1.5万"},
		{"MySQL规范", "MySQL 使用规范：索引命名、慢查询优化、主从配置"},
		{"产品评审纪要", "产品评审：首页改版通过，移动端适配下周五完成"},
		{"年会策划", "年度年会定在12月20日，地点海淀区某酒店，预算3万"},
	}
	// golden 集：query → 期望命中的笔记标题
	golden := []struct{ query, want string }{
		{"项目什么时候截止", "项目A进度"},
		{"东京玩什么", "东京旅行攻略"},
		{"MySQL 索引怎么写", "MySQL规范"},
		{"评审结论是什么", "产品评审纪要"},
		{"年会定在哪天", "年会策划"},
	}

	// 真实检索链路：gracedb + fakeEmbedder + 伪嵌入 provider（与生产 NoteVectorService 同构）
	gdb, err := gracedb.Open(t.TempDir()+"/eval", gracedb.WithEmbedder(fakeEmbedder{}))
	require.NoError(t, err)
	defer gdb.Close()
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("fake-embed", embedFakeProvider{})
	noteVecSvc := &NoteVectorService{vectorSvc: &VectorService{db: gdb}, aiService: aiSvc}
	for i, n := range seedNotes {
		require.NoError(t, noteVecSvc.VectorizeNote(1, uint(i+1), n.title, n.content))
	}

	// 评测：每个 query 检索 Top-3，统计 hit@1 / hit@3 / MRR
	hits1, hits3 := 0, 0
	mrr := 0.0
	for _, g := range golden {
		results, err := noteVecSvc.SearchNotes(1, g.query, 3)
		require.NoError(t, err)
		titles := make([]string, 0, len(results))
		for _, r := range results {
			titles = append(titles, r.Metadata["title"])
		}
		rank := -1
		for i, ti := range titles {
			if ti == g.want {
				rank = i
				break
			}
		}
		if rank >= 0 {
			hits3++
			mrr += 1.0 / float64(rank+1)
			if rank == 0 {
				hits1++
			}
		}
		t.Logf("query=%q want=%q top3=%v hit@1=%v", g.query, g.want, titles, rank == 0)
	}
	rate1 := float64(hits1) / float64(len(golden))
	rate3 := float64(hits3) / float64(len(golden))
	mrrAvg := mrr / float64(len(golden))
	t.Logf("评测结果: hit@1=%.2f (%d/%d) hit@3=%.2f (%d/%d) MRR=%.3f",
		rate1, hits1, len(golden), rate3, hits3, len(golden), mrrAvg)

	// 回归守卫：主题分明的种子语料上 hit@3 应全中；hit@1 至少 3/5
	assert.Equal(t, len(golden), hits3, "种子语料上 hit@3 应全中（检索回归守卫，改动不得跌破）")
	assert.GreaterOrEqual(t, hits1, len(golden)-2, "hit@1 至少 3/5（宽松守卫）")
}
