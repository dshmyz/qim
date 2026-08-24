package service

// 回复质量评测闭环（方向1：LLM-as-judge）。
// 目的：先把"回复质量"变成可度量、可持续回归的指标，再支撑后续对提示词/召回/上下文的每次 A/B。
// 本文件是一个自包含的 judge 打分器 + 一组"优质 vs 劣质回答"对照样例，用来验证 judge 本身
// 能稳定区分"忠实基于依据的回答"与"幻觉/跑题的回答"。后续接入真实回复流水线时复用同一打分器。
//
// 仅人工核对用途，不属于正式测试；会触发真实 LLM 调用与费用。
// 显式开启：MEM_EVAL=1 go test ./service/ -run TestReplyQualityJudgeEval -v

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/config"
)

// ReplyQuality 该维度统一 LLM-as-judge 打分的结构（judge 产出的 JSON 对应）。
// 目标不是"绝对准确"，而是"方向正确且可对比"：同一样本下 judge 应能分辨好/坏回答。
type ReplyQuality struct {
	Faithfulness int    `json:"faithfulness"` // 忠实度 1-5：回复是否严格贴合提供的依据，不自行编造
	Relevance    int    `json:"relevance"`    // 相关度 1-5：回复是否针对用户问题
	Hallucinate  bool   `json:"hallucinate"`  // 是否出现幻觉（回答中出现依据里没有的事实）
	Composite    int    `json:"composite"`    // 综合分 1-5：三个维度的加权打分
	Reason       string `json:"reason"`       // judge 一句话理由，便于人工核对
}

// judgeReplyQuality 让 judge 模型按忠实度/相关度/幻觉/综合分给一条回复打分。
// 依据（context）即回复应"基于"的召回内容；这条 judge 打分可复用于任意回复，是评测循环的核心。
func judgeReplyQuality(svc *ai.AIService, question, context, reply string) (ReplyQuality, error) {
	judgePrompt := `你是回答质量评估器。下面给出一条用户问题、一条"回复所应依据的资料"，以及一条 AI 回复。
请评估该回复在多大程度上忠实且贴合地使用了依据资料来回答用户。

评估维度：
- faithfulness(1-5)：回复是否严格依据提供资料作答，不编造资料里没有的事实。
- relevance(1-5)：回复是否切题地回应了用户问题，而非答非所问。
- hallucinate(true/false)：回复中是否出现了依据资料里不存在的事实/数字/结论。
- composite(1-5)：综合分，重点惩罚"未依据资料却编造细节"的回复。

资料可能为空——资料为空时，回复只要不把编造内容说成是资料里的，忠实度就不应判低。

以 JSON 返回，形如 {"faithfulness":5,"relevance":5,"hallucinate":false,"composite":5,"reason":"一句话理由"}。

用户问题：` + question + `

依据资料：
` + context + `

AI 回复：
` + reply

	raw, err := svc.GetCompletion(ai.TaskTypeAnalysis, []ai.Message{{Role: "user", Content: judgePrompt}})
	if err != nil {
		return ReplyQuality{}, err
	}
	var q ReplyQuality
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		// 容错：可能带 markdown 代码围栏或前后残字，剥掉非 JSON 惰性再解一次
		s := raw
		if i := strings.Index(s, "{"); i >= 0 {
			s = s[i:]
		}
		if j := strings.LastIndex(s, "}"); j >= 0 {
			s = s[:j+1]
		}
		if err2 := json.Unmarshal([]byte(s), &q); err2 != nil {
			return ReplyQuality{}, err2
		}
	}
	return q, nil
}

type replyQualityCase struct {
	name  string
	q     string
	ctx   string
	reply string
	want  string // good / poor
}

// replyQualityCases 对照样例：同一问题+资料下给"忠实回答"和"幻觉/跑题回答"两种回复，
// 验证 judge 判断标准稳定——好回复综合分显著高于坏回复，坏回复幻觉被标出。
func replyQualityCases() []replyQualityCase {
	// 资料：假设从笔记/群知识/记忆召回到的内容
	const ctxProject = "项目 Alpha 计划下周一（8月31日）上线，后端已统一改用 PostgreSQL，上线由老王负责。"
	const ctxCoffee = "（资料为空，仅对话：用户偏好记忆）用户喜欢美式咖啡，不加糖。"
	const ctxWeather = "用户问今天天气，依据资料来自天气预报笔记：上海今天多云转阵雨，最高 26℃。"
	return []replyQualityCase{
		{
			name:  "忠实回答-项目",
			q:     "项目 Alpha 什么时候上线？谁负责？",
			ctx:   ctxProject,
			reply: "项目 Alpha 定在下周一（8月31日）上线，进展由老王负责。",
			want:  "good",
		},
		{
			name:  "幻觉回答-项目",
			q:     "项目 Alpha 什么时候上线？谁负责？",
			ctx:   ctxProject,
			reply: "项目 Alpha 将于下周一上线，由小李负责，本次上线还包含支付模块和消息推送两个新功能。",
			want:  "poor",
		},
		{
			name:  "跑题回答-项目",
			q:     "项目 Alpha 什么时候上线？",
			ctx:   ctxProject,
			reply: "我今天心情不错，天气挺好，改天一起吃饭吧。",
			want:  "poor",
		},
		{
			name:  "忠实回答-天气",
			q:     "今天上海天气怎么样？",
			ctx:   ctxWeather,
			reply: "今天上海多云转阵雨，最高 26℃，出门建议带伞。",
			want:  "good",
		},
		{
			name:  "幻觉回答-天气",
			q:     "今天上海天气怎么样？",
			ctx:   ctxWeather,
			reply: "今天上海晴到多云，最高 32℃，没什么降雨。",
			want:  "poor",
		},
	}
}

func TestReplyQualityJudgeEval(t *testing.T) {
	if os.Getenv("MEM_EVAL") != "1" {
		t.Skip("真实 LLM 评测默认关闭，设置 MEM_EVAL=1 开启")
	}
	wd, _ := os.Getwd()
	serverDir := wd
	for strings.HasSuffix(serverDir, "/service") || strings.HasSuffix(serverDir, "\\service") {
		serverDir = strings.TrimSuffix(serverDir, "/service")
		serverDir = strings.TrimSuffix(serverDir, "\\service")
		break
	}
	if _, err := os.Stat(serverDir + "/config.yaml"); err != nil {
		t.Skipf("未找到 config.yaml（%s），跳过真实 LLM 评测", serverDir+"/config.yaml")
	}
	_ = os.Chdir(serverDir)

	svc := ai.NewAIService(&config.Load().AI)
	if !svc.IsConfigured() {
		t.Skip("无可用 AI 供应商，跳过评测")
	}

	// 统计"好回答综合分>坏回答"的对照是否稳定成立，验证 judge 度量本身可信。
	consistent, goodTotal, goodSum, poorTotal, poorSum := 0, 0, 0, 0, 0
	for _, c := range replyQualityCases() {
		q, err := judgeReplyQuality(svc, c.q, c.ctx, c.reply)
		mark := "✗"
		if err == nil {
			if c.want == "good" {
				goodTotal++
				goodSum += q.Composite
				if q.Composite >= 4 && !q.Hallucinate {
					mark = "✓"
					consistent++
				}
			} else {
				poorTotal++
				poorSum += q.Composite
				// 坏回答看综合分低即可：跑题未必触发幻觉，但综合分应显著低于好回答。
				if q.Composite <= 2 {
					mark = "✓"
					consistent++
				}
			}
		}
		t.Logf("%s [%s] 期望=%s composite=%d faithful=%d relevant=%d hallucinate=%v (err=%v)\n   回复: %s",
			mark, c.name, c.want, q.Composite, q.Faithfulness, q.Relevance, q.Hallucinate, err, c.reply)
	}
	total := goodTotal + poorTotal
	avgGood, avgPoor := 0.0, 0.0
	if goodTotal > 0 {
		avgGood = float64(goodSum) / float64(goodTotal)
	}
	if poorTotal > 0 {
		avgPoor = float64(poorSum) / float64(poorTotal)
	}
	t.Logf("judge 对照一致率 %d/%d；好回复均值 composite=%.1f，坏回复均值 composite=%.1f",
		consistent, total, avgGood, avgPoor)
	if total > 0 && consistent < 4 {
		t.Fatalf("judge 稳定性不足：好/坏回答对照仅 %d/%d 成立", consistent, total)
	}
}

// baselineReplyCase 与 replyQualityCase 类似，多带 tag：同一问题可放多条不同回复打不同标签，
// 用于"现网(baseline) vs 改动后(alt/target)"的 A/B 对比。caseKey 问题+context 相同才算命中同一对比组。
type baselineReplyCase struct {
	Question string `json:"question"`
	Context  string `json:"context"`
	Reply    string `json:"reply"`
	Tag      string `json:"tag"`
}

// baselineGroupTagStats 一组标签的聚合得分。
type baselineGroupTagStats struct {
	Tag                 string
	N                   int
	Composite, Fid, Rel float64
	HallucinateRate     int // 出现幻觉的条数
}

// TestReplyQualityBaselineEval 读取 testdata/reply_quality_cases.json（人工从真实回复日志填充）：
// 对每条用 judgeReplyQuality 打分，按 tag 聚合出各维度均值；同一问题下存在 baseline 与 target 时
// 报告 target - baseline 的 delta。这是"回复质量可度量、可回归"的基线回路——后续改提示词/检索配置，
// 只需替换对应 tag 的 reply 再重跑即可对比是否变好。
// 默认跳过；显式开启：MEM_EVAL=1 go test ./service/ -run TestReplyQualityBaselineEval -v
func TestReplyQualityBaselineEval(t *testing.T) {
	if os.Getenv("MEM_EVAL") != "1" {
		t.Skip("真实 LLM 评测默认关闭，设置 MEM_EVAL=1 开启")
	}
	wd, _ := os.Getwd()
	serverDir := strings.TrimSuffix(wd, "/service")
	serverDir = strings.TrimSuffix(serverDir, "\\service")
	if _, err := os.Stat(serverDir + "/config.yaml"); err != nil {
		t.Skipf("未找到 config.yaml（%s），跳过真实 LLM 评测", serverDir+"/config.yaml")
	}
	_ = os.Chdir(serverDir)

	data, err := os.ReadFile(serverDir + "/testdata/reply_quality_cases.json")
	if err != nil {
		t.Skipf("无样例文件 %s（%v），跳过。可从真实回复日志填充此 JSON 开启基线评测", serverDir+"/testdata/reply_quality_cases.json", err)
	}
	var cases []baselineReplyCase
	if err := json.Unmarshal(data, &cases); err != nil || len(cases) == 0 {
		t.Skipf("样例 JSON 为空或无法解析：%v", err)
	}

	svc := ai.NewAIService(&config.Load().AI)
	if !svc.IsConfigured() {
		t.Skip("无可用 AI 供应商，跳过评测")
	}

	type entry struct {
		q baselineReplyCase
		s ReplyQuality
	}
	var scored []entry
	for _, c := range cases {
		s, err := judgeReplyQuality(svc, c.Question, c.Context, c.Reply)
		if err != nil {
			t.Logf("评分失败 [%s]: %v (reply=%q)", c.Tag, err, c.Reply)
			continue
		}
		scored = append(scored, entry{c, s})
		t.Logf("[%s] composite=%d faithful=%d relevant=%d hallucinate=%v\n   回复: %s", c.Tag, s.Composite, s.Faithfulness, s.Relevance, s.Hallucinate, c.Reply)
	}

	// 按 tag 聚合
	agg := map[string]*baselineGroupTagStats{}
	for _, e := range scored {
		st := agg[e.q.Tag]
		if st == nil {
			st = &baselineGroupTagStats{Tag: e.q.Tag}
			agg[e.q.Tag] = st
		}
		st.N++
		st.Composite += float64(e.s.Composite)
		st.Fid += float64(e.s.Faithfulness)
		st.Rel += float64(e.s.Relevance)
		if e.s.Hallucinate {
			st.HallucinateRate++
		}
	}
	for tag, st := range agg {
		n := float64(st.N)
		t.Logf("== 聚合 tag=%s: n=%d composite=%.2f faithfulness=%.2f relevance=%.2f hallucinate=%d/%d ==",
			tag, st.N, st.Composite/n, st.Fid/n, st.Rel/n, st.HallucinateRate, st.N)
	}

	// 同一问题出现多个 tag 时，输出 target/alt 相对 baseline 的 delta，作为 A/B 评判依据
	t.Log("== A/B delta（同问题下的 tag 差异，以 baseline 为基准）==")
	byKey := map[string]map[string]ReplyQuality{}
	for _, e := range scored {
		key := e.q.Question + "\n" + e.q.Context
		if byKey[key] == nil {
			byKey[key] = map[string]ReplyQuality{}
		}
		byKey[key][e.q.Tag] = e.s
	}
	for key, m := range byKey {
		if m["baseline"].Composite == 0 {
			continue // 无基准，无从对比
		}
		for tag, s := range m {
			if tag == "baseline" {
				continue
			}
			delta := float64(s.Composite) - float64(m["baseline"].Composite)
			t.Logf("  问题: %q\n    baseline composite=%d <- %s composite=%d (delta=%.0f)",
				strings.SplitN(key, "\n", 2)[0], m["baseline"].Composite, tag, s.Composite, delta)
		}
	}
}
