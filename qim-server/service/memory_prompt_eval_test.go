package service

// 临时评测：记忆判定提示词「改造前 vs 改造后」真实 LLM 对照 + scope 标注抽查。
// 仅用于人工核对，不属于正式测试；会触发真实 LLM 调用与费用。
// 默认跳过；显式开启：MEM_EVAL=1 go test ./service/ -run TestMemoryPromptEval -v

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/config"
)

// beforeRememberBlock 旧版 evaluateRemember 追加的 JSON 指令（无挡位锚点、无数样本）。
// 与旧代码逐字一致：`importance: 1(极不重要) 到 5(极重要) 的整数档位`。
func beforeRememberBlock(msg string) string {
	return `判断以下对话内容是否包含值得记忆的长期信息。
值得记忆：个人偏好、重要决定、项目关键信息、约定事项。
不值得记忆：寒暄问候、确认/感谢短句、情绪化表达、日常流水（吃饭、天气、出行等琐碎日常）。

请以 JSON 返回，形如 {"remember": true, "importance": 3}。
- remember: 是否值得记忆
- importance: 1(极不重要) 到 5(极重要) 的整数档位

内容：` + msg
}

// afterVerdictPrompt 复用当前生产提示词（共享负面清单 + evaluateRemember 的新 JSON 指令）。
func afterVerdictPrompt() string {
	return `判断以下对话内容是否包含值得记忆的长期信息。
值得记忆：个人偏好、重要决定、项目关键信息、约定事项、答疑形成的可复用知识（步骤、配置、口径、规范）。
` + rememberVerdictNegativeClause
}

func memoryEvalCases() []string {
	return []string{
		"和张三约好周五晚上一起吃饭",        // conversation，应记
		"我答应王总监周四前提交Q3方案",     // conversation，应记
		"项目 Alpha 上线定在下周一",       // global，应记
		"团队决定后端统一改用 PostgreSQL",  // global 决定，应记
		"我喜欢美式咖啡，不加糖",          // global 偏好，应记
		"我习惯每天早上起床后先跑三公里",     // global 偏好，应记
		"这周末想去XX山自驾，要不要一起",     // 边界：邀约，倾向记
		"只要价格低于预算就立刻签合同",       // 关键决定，应记
		"接入钉钉机器人的方式是后台配 webhook，走 /api/v1/bot/webhook 接口，要先申请权限", // 答疑→global fact，应记
		"新版部署流程是先跑 migrate 再 build 再发版，记得先备份数据库", // 答疑知识，应记
		"这个 bug 我现在就给你处理，半小时后看结果", // 针对某人的临时答复，应不记
		"你问的那个接口我等下发你",         // 临时答复，未形成复用知识，应不记
		"我的微信密码是 Pin123456",      // 敏感凭据，应不记
		"帮我把这个文件放到桌面",          // 即时操作指令，应不记
		"今天周三，明天就是截止日了",        // 一次性相对时间，应不记
		"这破系统每次导出都卡死",          // 吐槽/主观宣泄（虽含实体），应不记
		"会议室我订到明天早上了",          // 一次性短效安排，应不记（importance≤2 联动）
		"今天中午吃了牛肉面",            // 不记：琐碎
		"外面下雨了，出门记得带伞",         // 不记：日常流水
		"嗯嗯知道了",                 // 不记：确认短句
		"哈哈笑死我了",                // 不记：情绪化
		"好的，谢谢你",                // 不记：感谢短句
		"今天天气不错",                // 不记：闲聊
		"那我们先这样，改天再聊",          // 不记：寒暄
	}
}

func TestMemoryPromptEval(t *testing.T) {
	if os.Getenv("MEM_EVAL") != "1" {
		t.Skip("真实 LLM 评测默认关闭，设置 MEM_EVAL=1 开启")
	}
	// 定位 qim-server 目录并切进去，让 config.Load() 读 config.yaml
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

	cfg := config.Load()
	svc := ai.NewAIService(&cfg.AI)
	if !svc.IsConfigured() {
		t.Skip("无可用 AI 供应商，跳过评测")
	}

	cases := memoryEvalCases()
	t.Log("== 判定(remember/importance)：改造前 vs 改造后 ==")
	for _, msg := range cases {
		// 改造前：直接构造完整 prompt 走 GetCompletion，再走现有解析
		beforeRaw, err1 := svc.GetCompletion(ai.TaskTypeAnalysis, []ai.Message{{Role: "user", Content: beforeRememberBlock(msg)}})
		before, _ := parseRememberVerdictJSON(beforeRaw)
		// 改造后：走 evaluateRemember 生产路径
		after, err2 := evaluateRemember(svc, afterVerdictPrompt(), msg)

		b := "ERR"
		if err1 == nil {
			b = formatVerdict(before)
		}
		a := "ERR"
		if err2 == nil {
			a = formatVerdict(after)
		}
		t.Logf("消息: %q\n  before: %s (err=%v)\n  after : %s (err=%v)", msg, b, err1, a, err2)
	}

	// 改造后 scope 抽查：仅对判定值得记的样本看 reflection 的 scope 标注
	t.Log("== scope 标注（改造后 reflection）==")
	for _, msg := range []string{
		"和张三约好周五晚上一起吃饭",
		"我答应王总监周四前提交Q3方案",
		"项目 Alpha 上线定在下周一",
		"团队决定后端统一改用 PostgreSQL",
		"我喜欢美式咖啡，不加糖",
		"这周末想去XX山自驾，要不要一起",
		"接入钉钉机器人的方式是后台配 webhook，走 /api/v1/bot/webhook 接口，要先申请权限",
	} {
		ref, ok := reflectStructure(svc, msg, nil, nil, nil)
		if !ok {
			t.Logf("消息: %q -> 反射失败", msg)
			continue
		}
		t.Logf("消息: %q -> scope=%q type=%q", msg, ref.Scope, ref.Type)
	}
}

func formatVerdict(v RememberVerdict) string {
	prefix := "记"
	if !v.ShouldRemember {
		prefix = "不记"
	}
	return prefix + " 重要性=" + trimFloat(v.Importance)
}

func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// TestMemoryMergeKindEval 真实 LLM 评测近义/冲突合并判定 inferMemoryMergeKind 的三分类准确率。
// 默认跳过；显式开启：MEM_EVAL=1 go test ./service/ -run TestMemoryMergeKindEval -v
func TestMemoryMergeKindEval(t *testing.T) {
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

	type pair struct {
		expected MemoryMergeKind
		new, old string
	}
	pairs := []pair{
		{MemoryMergeDuplicate, "我喜欢美式咖啡", "我习惯喝美式咖啡"},
		{MemoryMergeDuplicate, "项目截止日期是3月15日", "项目3月15日就截止了哈"},
		{MemoryMergeDuplicate, "周会定在每周五下午三点", "每周五下午三点开周会"},
		{MemoryMergeConflict, "项目改用 PostgreSQL 作为数据库", "项目使用 MySQL 作为数据库"},
		{MemoryMergeConflict, "截止日期推迟到3月20日", "项目截止日期是3月15日"},
		{MemoryMergeConflict, "这周六改去爬山了", "这周六约好去露营"},
		{MemoryMergeNew, "我喜欢喝美式咖啡", "我最近在学游泳"},
		{MemoryMergeNew, "项目改用 PostgreSQL", "团队决定每周五发版"},
		{MemoryMergeNew, "和张三约好周五吃饭", "项目使用 MySQL"},
	}

	kindName := map[MemoryMergeKind]string{MemoryMergeNew: "new", MemoryMergeConflict: "conflict", MemoryMergeDuplicate: "duplicate"}
	correct, total := 0, len(pairs)
	for _, p := range pairs {
		got, err := inferMemoryMergeKind(svc, p.new, p.old)
		mark := "✗"
		if err == nil && got == p.expected {
			mark = "✓"
			correct++
		}
		t.Logf("%s 期望=%-9s 实际=%-9s (err=%v)\n  新: %s\n  旧: %s", mark, kindName[p.expected], kindName[got], err, p.new, p.old)
	}
	t.Logf("准确率 %d/%d", correct, total)
	if correct < 7 {
		t.Fatalf("合并判定准确率过低 %d/%d", correct, total)
	}
}