package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/mention"
)

const agentTestSelfID = uint64(77)
const agentTestThread = uint64(42)

// agentStubServer 模拟 QIM 的 Bot API：身份、全会话列表、增量拉取、出站回写、流式分段。
type agentStubServer struct {
	mu                 sync.Mutex
	inbox              []message        // 模拟会话里的消息
	posted             []map[string]any // POST /bot/messages 的请求体
	streams            []map[string]any // POST /bot/messages/:id/stream 的请求体
	reads              []map[string]any // POST /bot/read 的请求体
	meCalled           bool
	legacyOnly         bool             // true=不提供 /bot/conversations（模拟旧版 server），只服务 /bot/groups
	extraConversations []map[string]any // 追加进 /bot/conversations 响应的额外会话（测运行期新发现）
}

func (s *agentStubServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON := func(v any) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": v})
		}
		switch {
		case r.URL.Path == "/api/v1/bot/me":
			s.mu.Lock()
			s.meCalled = true
			s.mu.Unlock()
			writeJSON(map[string]any{
				"id": 1, "name": "小助手", "virtual_user_id": agentTestSelfID, "is_active": true,
			})
		case r.URL.Path == "/api/v1/bot/conversations" && !s.legacyOnly:
			convs := []map[string]any{
				{"conversation_id": agentTestThread, "conv_type": "group", "name": "测试群"},
			}
			convs = append(convs, s.extraConversations...)
			writeJSON(convs)
		case r.URL.Path == "/api/v1/bot/groups":
			writeJSON([]map[string]any{{"conversation_id": agentTestThread, "group_name": "测试群"}})
		case r.URL.Path == "/api/v1/bot/messages" && r.Method == http.MethodGet:
			after := uint64(0)
			if v := r.URL.Query().Get("after_id"); v != "" {
				if n, err := strconv.ParseUint(v, 10, 64); err == nil {
					after = n
				}
			}
			out := []message{}
			s.mu.Lock()
			for _, m := range s.inbox {
				if m.ID > after {
					out = append(out, m)
				}
			}
			s.mu.Unlock()
			writeJSON(map[string]any{"messages": out})
		case r.URL.Path == "/api/v1/bot/messages" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			s.mu.Lock()
			s.posted = append(s.posted, body)
			s.mu.Unlock()
			writeJSON(map[string]any{"message_id": 999, "conversation_id": body["conversation_id"]})
		case strings.HasPrefix(r.URL.Path, "/api/v1/bot/messages/") && strings.HasSuffix(r.URL.Path, "/stream") && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			s.mu.Lock()
			s.streams = append(s.streams, body)
			s.mu.Unlock()
			writeJSON(map[string]any{})
		case r.URL.Path == "/api/v1/bot/read" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			s.mu.Lock()
			s.reads = append(s.reads, body)
			s.mu.Unlock()
			writeJSON(map[string]any{})
		case r.URL.Path == "/api/v1/bot/heartbeat" && r.Method == http.MethodPost:
			writeJSON(map[string]any{})
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	})
}

func startAgentStub(t *testing.T) *agentStubServer {
	t.Helper()
	s := &agentStubServer{}
	srv := httptest.NewServer(s.handler())
	t.Cleanup(srv.Close)
	writeTestConfig(t, srv.URL, "qbot_test_token")
	return s
}

// newAgentRunnerForTest 组装一个只依赖内存状态的 runner。
func newAgentRunnerForTest(opts agentLoopOptions) *agentRunner {
	return &agentRunner{
		cfg:      opts,
		attempts: map[uint64]int{},
		groups:   map[uint64]string{agentTestThread: "测试群"},
		dmNames:  map[uint64]string{},
		state: agentLoopState{
			LastID:  map[uint64]uint64{},
			History: map[uint64][]agentBriefState{},
		},
		selfID:   agentTestSelfID,
		selfName: "小助手",
	}
}

func baseLoopOpts(statePath string) agentLoopOptions {
	return agentLoopOptions{
		agentCmd:     "cat", // prompt 走 stdin，原样回显为回复——便于断言提示词内容
		agentTimeout: time.Minute,
		statePath:    statePath,
		replyType:    "markdown",
		acceptTypes:  map[string]bool{"text": true, "markdown": true, "card_action": true},
		mentionOnly:  true,
		once:         true,
		promptTpl:    agentDefaultPrompt,
	}
}

// TestAgentLoop_ReplyToMentionEndToEnd 走完整链路：拉到群里 @ 本机器人的消息 →
// 交给 agent（cat）→ 回写到**同一会话**（conversation_id 必须等于 thread_id，
// 只传 to_user_id 会另开单聊回不到群里）→ 水位推进落盘。
func TestAgentLoop_ReplyToMentionEndToEnd(t *testing.T) {
	stub := startAgentStub(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	// 预置水位：跳过冷启动的 latestMessageID 翻页
	writeState(t, statePath, map[uint64]uint64{agentTestThread: 100})

	stub.inbox = []message{{
		ID: 101, ConversationID: agentTestThread, SenderID: 5,
		SenderType: "user", SenderNickname: "alice",
		Content: mention.Encode(uint(agentTestSelfID), "小助手") + " 你好在吗",
		Type:    "text", CreatedAt: "2026-10-15T10:00:00Z",
	}}

	runAgentLoop(baseLoopOpts(statePath))

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if !stub.meCalled {
		t.Errorf("未传 --self-id 时应通过 GET /bot/me 自发现身份")
	}
	// 流式占位：一次 POST（type=streaming，conversation_id 必须等于 thread_id）
	if len(stub.posted) != 1 {
		t.Fatalf("应恰有一次占位 POST，实际 %d 条：%+v", len(stub.posted), stub.posted)
	}
	body := stub.posted[0]
	if got := body["conversation_id"]; got != float64(agentTestThread) {
		t.Errorf("占位会话错误: conversation_id=%v，应为 %d（否则回不到群里）", got, agentTestThread)
	}
	if body["msg_type"] != "streaming" {
		t.Errorf("msg_type=%v，应为 streaming（客户端渲染 typing 气泡）", body["msg_type"])
	}
	// 流式收尾：agent（cat）回显 prompt，应作为一次 delta + finish 送达
	if len(stub.streams) != 1 {
		t.Fatalf("应恰有一次流式收尾，实际 %d 条：%+v", len(stub.streams), stub.streams)
	}
	content, _ := stub.streams[0]["content_delta"].(string)
	if !strings.Contains(content, "你好在吗") {
		t.Errorf("提示词未携带消息正文: %q", content)
	}
	if !strings.Contains(content, "@小助手") {
		t.Errorf("@ token 应还原成可读文本，实际: %q", content)
	}
	if !strings.Contains(content, "测试群") {
		t.Errorf("提示词应带群名: %q", content)
	}
	if stub.streams[0]["finish"] != true {
		t.Errorf("收尾应 finish=true，实际 %+v", stub.streams[0])
	}
	// 拉取即读仅限私聊：本测试轮询的是群会话，不应有已读上报
	if len(stub.reads) != 0 {
		t.Fatalf("群聊不应上报已读，实际 %d 条：%+v", len(stub.reads), stub.reads)
	}

	st := readState(t, statePath)
	if st.LastID[agentTestThread] != 101 {
		t.Errorf("水位应推进到 101，实际 %d", st.LastID[agentTestThread])
	}
	if len(st.History[agentTestThread]) != 1 {
		t.Errorf("应记录 1 条上下文，实际 %d", len(st.History[agentTestThread]))
	}
}

// TestAgentLoop_AbortsPlaceholderOnEmptyReply 智能体判定无需回复（stdout 为空）时，
// 流式占位应以 abort 丢弃——否则留下一条永久 typing/空气泡。
func TestAgentLoop_AbortsPlaceholderOnEmptyReply(t *testing.T) {
	stub := startAgentStub(t)
	statePath := filepath.Join(t.TempDir(), "state.json")
	writeState(t, statePath, map[uint64]uint64{agentTestThread: 100})

	stub.inbox = []message{{
		ID: 101, ConversationID: agentTestThread, SenderID: 5,
		SenderType: "user", SenderNickname: "alice",
		Content:    mention.Encode(uint(agentTestSelfID), "小助手") + " 在吗",
		Type:       "text", CreatedAt: "2026-10-19T10:00:00Z",
	}}

	opts := baseLoopOpts(statePath)
	opts.agentCmd = "true" // 退出码 0、无输出 → 判定无需回复
	runAgentLoop(opts)

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.posted) != 1 {
		t.Fatalf("应有占位 POST，实际 %d 条", len(stub.posted))
	}
	if len(stub.streams) != 1 {
		t.Fatalf("应恰有一次 abort，实际 %d 条：%+v", len(stub.streams), stub.streams)
	}
	if stub.streams[0]["abort"] != true {
		t.Errorf("空回复应 abort=true，实际 %+v", stub.streams[0])
	}
}

// TestAgentLoop_DMReportsRead 私聊拉取即读：处理过消息的私聊会话应上报已读
//（用户侧已读回执依赖它）；群聊不上报（bot 读群消息的已读没有人际意义）。
func TestAgentLoop_DMReportsRead(t *testing.T) {
	stub := startAgentStub(t)
	statePath := filepath.Join(t.TempDir(), "state.json")
	writeState(t, statePath, map[uint64]uint64{777: 100})

	stub.inbox = []message{{
		ID: 101, ConversationID: 777, SenderID: 5,
		SenderType: "user", SenderNickname: "alice",
		Content: "在吗", Type: "text", CreatedAt: "2026-10-19T10:00:00Z",
	}}

	opts := baseLoopOpts(statePath)
	opts.mentionOnly = false          // 私聊无 @ token
	opts.threads = []uint64{777}      // 显式 --thread 指向私聊（不在群列表中）
	runAgentLoop(opts)

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.reads) != 1 {
		t.Fatalf("私聊应恰有一次已读上报，实际 %d 条：%+v", len(stub.reads), stub.reads)
	}
	if got := stub.reads[0]["thread_id"]; got != float64(777) {
		t.Errorf("已读上报会话错误: thread_id=%v，应为 777", got)
	}
}

// TestAgentLoop_IgnoresNonMentionAndSelf 验证过滤：群里没人 @ 它就不吭声（否则刷屏），
// 自己/别的 bot 发的、系统消息一概不处理（防自问自答与 agent 互相触发成环）。
func TestAgentLoop_IgnoresNonMentionAndSelf(t *testing.T) {
	cases := []struct {
		name string
		msg  message
	}{
		{"没 @ 任何人的普通消息", message{ID: 201, SenderID: 5, SenderType: "user", Content: "今天天气不错", Type: "text"}},
		{"@ 了别人", message{ID: 202, SenderID: 5, SenderType: "user", Content: mention.Encode(9, "bob") + " 看下", Type: "text"}},
		{"@所有人", message{ID: 203, SenderID: 5, SenderType: "user", Content: mention.EncodeAll() + " 开会了", Type: "text"}},
		{"自己发的", message{ID: 204, SenderID: agentTestSelfID, SenderType: "bot", Content: mention.Encode(uint(agentTestSelfID), "小助手") + " 我说的", Type: "text"}},
		{"别的 bot 发的", message{ID: 205, SenderID: 88, SenderType: "bot", Origin: "bot", Content: "hi", Type: "text"}},
		{"系统消息", message{ID: 206, SenderID: 0, SenderType: "system", Content: "未配置回调地址", Type: "system"}},
		{"不在响应类型内", message{ID: 207, SenderID: 5, SenderType: "user", Content: mention.Encode(uint(agentTestSelfID), "小助手") + " 图", Type: "image"}},
	}
	r := newAgentRunnerForTest(baseLoopOpts(""))
	for _, c := range cases {
		if r.shouldReply(agentTestThread, c.msg) {
			t.Errorf("%s: 不应触发回复", c.name)
		}
	}

	hit := message{ID: 208, SenderID: 5, SenderType: "user",
		Content: mention.Encode(uint(agentTestSelfID), "小助手") + " 你好", Type: "text"}
	if !r.shouldReply(agentTestThread, hit) {
		t.Errorf("@ 本机器人的文本消息应触发回复")
	}
}

// TestAgentLoop_SingleChatRepliesWithoutMention 验证 mention-only 只约束群聊：
// 群外会话（1:1 私聊）的消息天然是对着 bot 说的，不带 @ token 也应每条都回。
// （threadID 不在 groups 表 = 非群聊会话。）
func TestAgentLoop_SingleChatRepliesWithoutMention(t *testing.T) {
	r := newAgentRunnerForTest(baseLoopOpts(""))
	dm := message{ID: 301, SenderID: 5, SenderType: "user", Content: "在吗", Type: "text"}

	const dmThread = 777 // 不在 newAgentRunnerForTest 预置的 groups 表里
	if !r.shouldReply(dmThread, dm) {
		t.Errorf("单聊消息不带 @ 也应触发回复")
	}
	// 同样内容在群里（mention-only=true）不应触发
	if r.shouldReply(agentTestThread, message{ID: 302, SenderID: 5, SenderType: "user", Content: "在吗", Type: "text"}) {
		t.Errorf("群内未 @ 的消息不应触发回复")
	}
	// mention-only=false 时群内每条都回（原语义保留）
	r.cfg.mentionOnly = false
	if !r.shouldReply(agentTestThread, message{ID: 303, SenderID: 5, SenderType: "user", Content: "在吗", Type: "text"}) {
		t.Errorf("mention-only=false 时群内普通消息也应触发回复")
	}
}

// TestAgentDiscover_LegacyServerFallback 旧版 server（无 /bot/conversations）时 discover
// 应降级 /bot/groups 仅发现群聊，而不是启动即死；群会话仍进 groups 表（mention 过滤有效）。
func TestAgentDiscover_LegacyServerFallback(t *testing.T) {
	s := &agentStubServer{legacyOnly: true}
	srv := httptest.NewServer(s.handler())
	t.Cleanup(srv.Close)
	writeTestConfig(t, srv.URL, "qbot_test_token")

	r := newAgentRunnerForTest(baseLoopOpts(""))
	r.groups = map[uint64]string{} // 清空预置，验证由 discover 填充
	if err := r.discover(mustConfig()); err != nil {
		t.Fatalf("旧版 server 下 discover 不应失败: %v", err)
	}
	if _, ok := r.groups[agentTestThread]; !ok {
		t.Errorf("降级路径应通过 /bot/groups 发现群聊，实际 groups=%v", r.groups)
	}
	if len(r.dmNames) != 0 {
		t.Errorf("降级路径无私聊发现，dmNames 应为空，实际 %v", r.dmNames)
	}
}

// TestAgentRefresh_RuntimeDiscoversNewDM 运行期刷新：服务端新出现的私聊会话被并入轮询
// 并预置水位 0（未答的既有消息会被处理）；群列表整体替换（不在列表中的群退出轮询）；
// 既有会话的水位不受刷新影响（不重放）。
func TestAgentRefresh_RuntimeDiscoversNewDM(t *testing.T) {
	const dmThread = 777
	s := &agentStubServer{extraConversations: []map[string]any{
		{"conversation_id": float64(dmThread), "conv_type": "bot", "name": "爱丽丝"},
	}}
	srv := httptest.NewServer(s.handler())
	t.Cleanup(srv.Close)
	writeTestConfig(t, srv.URL, "qbot_test_token")

	r := newAgentRunnerForTest(baseLoopOpts(""))
	r.groups = map[uint64]string{agentTestThread: "旧名"}
	r.state.LastID[agentTestThread] = 555 // 既有水位
	const staleGroup = 888
	r.groups[staleGroup] = "已被移出" // 不在服务端列表里，刷新后应消失

	if err := r.refreshConversations(mustConfig(), false); err != nil {
		t.Fatalf("运行期刷新不应失败: %v", err)
	}
	if name, ok := r.dmNames[dmThread]; !ok || name != "爱丽丝" {
		t.Errorf("新私聊应被发现，dmNames=%v", r.dmNames)
	}
	if w, ok := r.state.LastID[dmThread]; !ok || w != 0 {
		t.Errorf("运行期新发现的私聊应预置水位 0，实际 %v", r.state.LastID[dmThread])
	}
	if _, ok := r.groups[staleGroup]; ok {
		t.Errorf("不在服务端列表中的群应被整体替换移除，groups=%v", r.groups)
	}
	if r.state.LastID[agentTestThread] != 555 {
		t.Errorf("既有会话水位不应被刷新影响，实际 %d", r.state.LastID[agentTestThread])
	}
}

// TestAgentLoop_HistoryExcludesCurrent 验证滚动上下文：维持最近 N 条、超限裁剪，
// 且渲染时排除当前这条——当前消息已在 {content}，再进 {history} 会让 agent 把同一句读两遍。
func TestAgentLoop_HistoryExcludesCurrent(t *testing.T) {
	r := newAgentRunnerForTest(baseLoopOpts(""))
	total := agentHistoryLimit + 5
	for i := 1; i <= total; i++ {
		r.observe(agentTestThread, message{
			ID: uint64(i), SenderID: 5, SenderNickname: "alice",
			Content: "msg" + strconv.Itoa(i),
		})
	}
	h := r.state.History[agentTestThread]
	if len(h) != agentHistoryLimit {
		t.Fatalf("上下文应裁到 %d 条，实际 %d", agentHistoryLimit, len(h))
	}
	if h[len(h)-1].MsgID != uint64(total) {
		t.Errorf("应保留最新一条，最后一条 id=%d", h[len(h)-1].MsgID)
	}

	got := r.formatHistory(agentTestThread, uint64(total))
	if n := strings.Count(got, "alice: "); n != agentHistoryLimit-1 {
		t.Errorf("渲染应排除当前消息：%d 行，期望 %d", n, agentHistoryLimit-1)
	}
	if strings.Contains(got, "msg"+strconv.Itoa(total)) {
		t.Errorf("当前消息不应出现在上下文里: %q", got)
	}
	if !strings.Contains(got, "msg"+strconv.Itoa(total-1)) {
		t.Errorf("上一条消息应出现在上下文里: %q", got)
	}
}

// TestAgentLoop_StateAtomicSave 状态文件必须原子写：进程被杀时不能留半份 JSON，
// 否则水位回退会把已回复的消息再回一遍。
func TestAgentLoop_StateAtomicSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")
	r := newAgentRunnerForTest(baseLoopOpts(path))
	r.state.LastID[agentTestThread] = 101

	if err := r.saveState(); err != nil {
		t.Fatalf("saveState 失败: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".agent-loop-") {
			t.Errorf("残留临时文件 %s", e.Name())
		}
	}

	r2 := newAgentRunnerForTest(baseLoopOpts(path))
	if err := r2.loadState(); err != nil {
		t.Fatalf("loadState 失败: %v", err)
	}
	if r2.state.LastID[agentTestThread] != 101 {
		t.Errorf("水位读回错误: %d", r2.state.LastID[agentTestThread])
	}
}

// writeState 预置水位文件（绕开冷启动的 latestMessageID 翻页）。
func writeState(t *testing.T, path string, last map[uint64]uint64) {
	t.Helper()
	b, err := json.Marshal(agentLoopState{
		LastID:  last,
		History: map[uint64][]agentBriefState{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readState(t *testing.T, path string) agentLoopState {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var st agentLoopState
	if err := json.Unmarshal(b, &st); err != nil {
		t.Fatal(err)
	}
	return st
}

// TestRunAgent_PromptViaStdin 核心安全约束：提示词只经 stdin 进入智能体，绝不拼进命令行
// ——否则群消息内容就成了 shell 注入面。
func TestRunAgent_PromptViaStdin(t *testing.T) {
	canary := filepath.Join(t.TempDir(), "pwned")
	prompt := `内容：$(touch ` + canary + `); rm -rf ~ 以及 ` + "`id`"
	out, err := runAgent(t.Context(), "cat", prompt, time.Minute)
	if err != nil {
		t.Fatalf("runAgent 失败: %v", err)
	}
	if out != prompt {
		t.Errorf("stdin 应原样传递\n got: %q\nwant: %q", out, prompt)
	}
	// prompt 里带了 $(touch …)，只有被拼进 shell 命令行才会落盘
	if _, statErr := os.Stat(canary); statErr == nil {
		os.Remove(canary)
		t.Errorf("提示词被当作 shell 执行了")
	}
}
