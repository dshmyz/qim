package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	// @ token 的解析契约只有这一处定义（`@{mention:<id>|<name>}`）。CLI 虽是纯 HTTP 客户端、
	// 不耦合 service/handler，但 mention 是零依赖的契约包：自己再抄一份正则迟早分叉，
	// 判断"这条消息是不是 @ 我"必须与服务端完全同源。
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/spf13/cobra"
)

// agentHistoryLimit 每个会话随水位一起落盘的最近消息条数，供 agent 理解上下文
// （"那后来呢"这类追问没有上文就答不了）。只记录本进程观察到的消息——冷启动瞬间
// 的历史不补，随轮询自然热起来，避免为取最近 N 条去翻几十页历史接口。
const agentHistoryLimit = 10

// agentMaxAttempts 单条消息的 agent 调用尝试次数（进程内计数）。超过即放弃并推进水位：
// pull 模式没有服务端 outbox 兜底，卡在同一条上会让整个会话永久停摆，比丢一条更糟。
const agentMaxAttempts = 3

// agentAcceptDefault 默认响应的消息类型。
// image/file 的 content 是资源引用而非可读文本，未做下载解析，故不入列；
// card_action 是用户点了本 bot 发出的卡片，必须响应（等价于 webhook 的 bot.card_action 事件）。
const agentAcceptDefault = "text,markdown,card_action"

// agentDefaultPrompt 默认提示词模板。占位符经 promptReplacer 替换；agent 的输出即回复正文，
// 因此明确要求"不要加前缀/不要解释"，否则群里会出现「回复：…」这类机器腔。
const agentDefaultPrompt = `你在即时通讯中收到一条新消息，请给出要回复的内容。

会话：{thread}{group}
发送者：{sender}（user_id={sender_id}）
时间：{time}
消息类型：{type}
最近上下文（如有）：
{history}

消息内容：
{content}

要求：直接输出回复正文（可用 Markdown）。不要加「回复：」「以下是回答」等前缀，
不要解释你在做什么，不要复述问题。若这条消息无需回应，输出空内容。`

// agentLoopOptions 一次轮询循环的全部运行参数。超过四个参数，故收敛为对象而非散参。
type agentLoopOptions struct {
	agentCmd     string        // 必填：shell 命令，prompt 走 stdin，stdout 即回复
	interval     time.Duration // 轮询间隔
	agentTimeout time.Duration // 单次 agent 调用超时
	threads      []uint64      // 空 = 自动发现全部已入群会话
	selfID       uint64        // 0 = 通过 GET /bot/me 自发现
	statePath    string        // 水位线落盘路径
	replyType    string        // 回复消息类型
	acceptTypes  map[string]bool
	mentionOnly  bool // true = 群里只响应 @ 本 bot 的消息
	replay       bool // true = 冷启动也处理历史（危险，默认关）
	once         bool // 只扫一轮就退出
	dryRun       bool // 只打印，不调 agent、不回写
	verbose      bool
	promptTpl    string
}

// agentBriefState 会话水位 + 最近上下文快照。带 msgID 是为了渲染上下文时能精确排除
// 当前这条（当前消息已在 {content} 里，再进 {history} 就重复了）。
type agentBriefState struct {
	MsgID  uint64 `json:"msg_id"`
	Sender string `json:"sender"`
	Text   string `json:"text"`
}

// agentLoopState 磁盘上的循环状态：thread id → 已处理到的最大 message id 与最近上下文。
type agentLoopState struct {
	LastID  map[uint64]uint64            `json:"last_id"`
	History map[uint64][]agentBriefState `json:"history,omitempty"`
}

// agentRunner 持有一次循环的配置与可变状态。
type agentRunner struct {
	cfg      agentLoopOptions
	state    agentLoopState
	attempts map[uint64]int    // key = message id
	groups   map[uint64]string // 群聊会话：id → 群名（也是 mention-only 的 isGroup 依据）
	dmNames  map[uint64]string // 1:1 私聊会话：id → 对端昵称（mention-only 不约束）
	selfID   uint64
	selfName string
}

func newAgentCmd() *cobra.Command {
	var (
		agentCmd     string
		interval     time.Duration
		agentTimeout time.Duration
		threadsRaw   string
		selfID       uint64
		statePath    string
		replyType    string
		accept       string
		mentionOnly  bool
		replay       bool
		once         bool
		dryRun       bool
		verbose      bool
		promptTpl    string
	)

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "桌面智能体接入（pull 模式：无 webhook 也能收发）",
	}
	loop := &cobra.Command{
		Use:   "loop",
		Short: "轮询会话新消息 → 调用本地智能体 → 回写到同一会话",
		Long: fmt.Sprintf(`%[1]s agent loop — 无公网、无 webhook 时的桌面智能体接入。

QIM 侧配置：机器人 → 模式选「外部 Agent」、webhook_url 留空（纯 pull 模式）、
签发令牌，再把机器人拉进群。本命令负责"醒来"：

  webhook_url 为空时 QIM 不主动投递，谁来拉消息？就是本进程——常驻轮询 + 水位线落盘，
  重启不重放、不重复回复。

用法：
  %[1]s config set --server http://localhost:8080 --token qbot_...
  %[1]s agent loop --command 'claude -p --output-format text'

智能体通过 stdin 收到提示词（原文里的 @ token 已还原成 @姓名），stdout 即回复正文。
prompt 不会被拼进命令行，因此不会被消息内容注入。

群内默认只响应 @ 本机器人的消息；1:1 私聊（不在群列表中的会话）每条都处理，
无需任何开关。--mention-only=false 可让群内也每条都回。

开机自启（轮询模式必须常驻，否则无人应答）：
  macOS  launchd plist 执行 %[1]s agent loop …
  Linux   systemd --user unit 的 ExecStart 同上`, CliName),
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			if strings.TrimSpace(agentCmd) == "" {
				die("--command 必填：智能体的非交互调用命令（prompt 走 stdin）")
			}
			acceptTypes := map[string]bool{}
			for _, t := range strings.Split(accept, ",") {
				if t = strings.TrimSpace(t); t != "" {
					acceptTypes[t] = true
				}
			}
			if len(acceptTypes) == 0 {
				die("--accept 不能为空")
			}
			var threads []uint64
			if threadsRaw != "" {
				for _, p := range strings.Split(threadsRaw, ",") {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					n, err := strconv.ParseUint(p, 10, 64)
					if err != nil {
						die("--thread 需为会话 ID（数字，逗号分隔）: %s", p)
					}
					threads = append(threads, n)
				}
			}
			tpl := promptTpl
			if tpl == "" {
				tpl = agentDefaultPrompt
			}
			opts := agentLoopOptions{
				agentCmd:     agentCmd,
				interval:     interval,
				agentTimeout: agentTimeout,
				threads:      threads,
				selfID:       selfID,
				statePath:    statePath,
				replyType:    replyType,
				acceptTypes:  acceptTypes,
				mentionOnly:  mentionOnly,
				replay:       replay,
				once:         once,
				dryRun:       dryRun,
				verbose:      verbose,
				promptTpl:    tpl,
			}
			runAgentLoop(opts)
		},
	}

	f := loop.Flags()
	f.StringVar(&agentCmd, "command", "", "智能体非交互命令（必填；prompt 从 stdin 读，stdout 为回复）")
	f.DurationVar(&interval, "interval", 2*time.Second, "轮询间隔")
	f.DurationVar(&agentTimeout, "agent-timeout", 120*time.Second, "单次智能体调用超时")
	f.StringVar(&threadsRaw, "thread", "", "只轮询这些会话（逗号分隔的 conversation_id）；空=自动发现全部群聊+私聊")
	f.Uint64Var(&selfID, "self-id", 0, "本机器人的虚拟用户 id；0=通过 GET /bot/me 自动发现")
	f.StringVar(&statePath, "state", "", "水位线文件路径（默认 ~/.qim/agent-loop-state.json）")
	f.StringVar(&replyType, "reply-type", "markdown", "回复消息类型: text|markdown")
	f.StringVar(&accept, "accept", agentAcceptDefault, "响应的消息类型（逗号分隔）")
	f.BoolVar(&mentionOnly, "mention-only", true, "群内只响应 @ 本机器人的消息（1:1 私聊不受限，恒每条回复；false=群内也全量回复）")
	f.BoolVar(&replay, "replay", false, "冷启动也处理已有历史消息（默认跳到最新，避免开进程就刷屏）")
	f.BoolVar(&once, "once", false, "只扫一轮就退出（调试用）")
	f.BoolVar(&dryRun, "dry-run", false, "只打印将要发送的内容，不调智能体也不回写")
	f.BoolVar(&verbose, "verbose", false, "输出轮询与跳过明细（stderr）")
	f.StringVar(&promptTpl, "prompt", "", "自定义提示词模板（占位符 {thread}{group}{sender}{sender_id}{time}{type}{content}{history}）")

	cmd.AddCommand(loop)
	return cmd
}

func runAgentLoop(opts agentLoopOptions) {
	cfg := mustConfig()
	if opts.interval <= 0 {
		opts.interval = 2 * time.Second
	}
	if opts.statePath == "" {
		opts.statePath = defaultAgentStatePath()
	}

	r := &agentRunner{cfg: opts, attempts: map[uint64]int{}, groups: map[uint64]string{}, dmNames: map[uint64]string{}}
	if err := r.loadState(); err != nil {
		// 状态读坏不致命：按冷启动对待，最多导致本次从最新开始，不重复回复
		logf("warn: 读取状态文件失败，按冷启动处理: %v", err)
	}
	if err := r.discover(cfg); err != nil {
		die("%v", err)
	}

	if opts.mentionOnly && r.selfID == 0 {
		die("无法确定本机器人身份（--self-id 未给，GET /bot/me 也失败），--mention-only 需要它判断是否 @ 我")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logf("轮询启动：会话 %d 个，间隔 %v，mention-only=%v，状态 %s",
		len(r.threadsToPoll()), opts.interval, opts.mentionOnly, opts.statePath)
	if opts.mentionOnly && len(opts.threads) > 0 {
		logf("提示: --thread 指定的会话若不在群列表中，按 1:1 私聊处理（每条都回复）")
	}

	lastDiscover := time.Now()
	lastHeartbeat := time.Time{} // 零值：启动后立即心跳一次
	for {
		// 运行期定时重发现：新建群/私聊最迟一个间隔后进入轮询，被踢出群即退出轮询。
		// 显式 --thread 时轮询范围由用户固定，无需刷新。内部失败仅 warn，不打断轮询。
		if len(r.cfg.threads) == 0 && time.Since(lastDiscover) >= agentRediscoverInterval {
			if err := r.refreshConversations(cfg, false); err != nil {
				logf("warn: 刷新会话列表失败: %v", err)
			}
			lastDiscover = time.Now()
		}
		// 心跳：向服务端声明本 bot 驱动进程在运行（bot 列表/运维视图展示在线状态）。
		// 失败静默——下个周期再试，服务端 90s TTL 自动判离。
		if time.Since(lastHeartbeat) >= agentHeartbeatInterval {
			if err := sendHeartbeat(); err != nil && r.cfg.verbose {
				logf("warn: 心跳失败: %v", err)
			}
			lastHeartbeat = time.Now()
		}
		for _, tid := range r.threadsToPoll() {
			if ctx.Err() != nil {
				break
			}
			if err := r.pollThread(ctx, cfg, tid); err != nil && ctx.Err() == nil {
				logf("warn: 会话 %d 轮询失败: %v", tid, err)
			}
		}
		if err := r.saveState(); err != nil {
			logf("warn: 保存状态失败: %v", err)
		}
		if opts.once {
			return
		}
		select {
		case <-ctx.Done():
			if err := r.saveState(); err != nil {
				logf("warn: 退出前保存状态失败: %v", err)
			}
			logf("已停止")
			return
		case <-time.After(opts.interval):
		}
	}
}

// threadsToPoll 返回本轮要扫的会话（显式指定优先，否则自动发现的全部群聊+私聊）。
func (r *agentRunner) threadsToPoll() []uint64 {
	if len(r.cfg.threads) > 0 {
		return r.cfg.threads
	}
	out := make([]uint64, 0, len(r.groups)+len(r.dmNames))
	for id := range r.groups {
		out = append(out, id)
	}
	for id := range r.dmNames {
		out = append(out, id)
	}
	return out
}

// agentRediscoverInterval 运行期重新发现会话的间隔：新建群/私聊最迟一个间隔后进入轮询。
const agentRediscoverInterval = 60 * time.Second

// agentHeartbeatInterval 心跳间隔：服务端 TTL 90s，30s 一次容忍丢包。
const agentHeartbeatInterval = 30 * time.Second

// discover 启动时全量发现：身份 + 会话列表（群聊 + 1:1 私聊，GET /bot/conversations）。
// 群聊进 groups（mention 过滤与群名展示），私聊进 dmNames（每条都回）。显式 --thread
// 时列表仅用于展示与归属提示，但"没有任何会话"不再是致命错误。
func (r *agentRunner) discover(cfg config) error {
	// 身份：判断 @ 是否指向自己，必须有稳定 id
	if r.cfg.selfID > 0 {
		r.selfID = r.cfg.selfID
	} else if id, name, err := fetchBotIdentity(cfg); err == nil {
		r.selfID, r.selfName = id, name
	} else if r.cfg.mentionOnly && len(r.cfg.threads) == 0 {
		return fmt.Errorf("获取机器人身份失败: %w", err)
	} else {
		logf("warn: 获取机器人身份失败（%v），mention 过滤不可用", err)
	}
	return r.refreshConversations(cfg, true)
}

// refreshConversations 拉取会话列表并整体替换 groups/dmNames（成功时以服务端为准，
// bot 被踢出群/会话被删即从轮询中移除）。fatal=true 用于启动（拉取失败且无显式 --thread、
// 或零会话为致命错误）；false 用于运行期定时刷新（任何失败仅 warn，沿用既有表，不因
// 瞬时故障丢轮询范围）。
//
// 旧版 server 兼容：/bot/conversations 仅在 404（路由不存在）时降级 /bot/groups 仅发现
// 群聊；其他错误（瞬时 5xx 等）不走降级——误归因为"版本旧"会误导排查，且运行期会永久
// 关闭私聊发现。
//
// 运行期（fatal=false）新发现的私聊会话把水位预置 0：私聊创建后的历史即「loop 在跑但
// 尚未轮询该会话期间」的未答消息，应当处理；新发现的群不做预置，沿用 pollThread 冷启动
// 跳到最新——回放整群旧 @ 会刷屏。
func (r *agentRunner) refreshConversations(cfg config, fatal bool) error {
	body, err := httpGet(cfg, cfg.ServerURL+"/api/v1/bot/conversations")
	if err != nil {
		if !strings.Contains(err.Error(), "HTTP 404") {
			if fatal {
				if len(r.cfg.threads) > 0 {
					logf("warn: 获取会话列表失败（%v），仅轮询 --thread 指定的会话；"+
						"这些会话将按 1:1 处理（每条都回复、不检查 @），群聊请确保会话列表可获取", err)
					return nil
				}
				return fmt.Errorf("获取会话列表失败: %w", err)
			}
			logf("warn: 刷新会话列表失败（%v），沿用上次发现的会话", err)
			return nil
		}
		// 404：旧版 server，降级 /bot/groups（仅群聊）。降级也失败才按致命处理。
		fbBody, fbErr := httpGet(cfg, cfg.ServerURL+"/api/v1/bot/groups")
		if fbErr != nil {
			if fatal {
				if len(r.cfg.threads) > 0 {
					logf("warn: 获取会话列表失败（%v），仅轮询 --thread 指定的会话；"+
						"这些会话将按 1:1 处理（每条都回复、不检查 @），群聊请确保会话列表可获取", err)
					return nil
				}
				return fmt.Errorf("获取会话列表失败: %w", err)
			}
			logf("warn: 刷新会话列表失败（%v），沿用上次发现的会话", err)
			return nil
		}
		if !fatal {
			logf("warn: server 未提供 /bot/conversations（版本较旧），私聊自动发现不可用，仅轮询群聊；升级 server 后恢复")
		}
		r.importLegacyGroups(fbBody)
		if fatal && len(r.groups) == 0 && len(r.cfg.threads) == 0 {
			return fmt.Errorf("机器人还没有任何会话：把机器人拉进群，或在客户端与它开私聊后再运行")
		}
		return nil
	}
	var resp struct {
		Data []struct {
			ConversationID uint64 `json:"conversation_id"`
			ConvType       string `json:"conv_type"`
			Name           string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		if fatal {
			return fmt.Errorf("解析会话列表失败: %w", err)
		}
		logf("warn: 解析会话列表失败（%v），沿用上次发现的会话", err)
		return nil
	}
	newGroups := map[uint64]string{}
	newDMs := map[uint64]string{}
	for _, c := range resp.Data {
		if c.ConvType == "bot" {
			newDMs[c.ConversationID] = c.Name
		} else {
			newGroups[c.ConversationID] = c.Name
		}
	}
	r.groups, r.dmNames = newGroups, newDMs
	if !fatal {
		// 运行期新发现的私聊：预置水位 0，未答的既有消息会被处理（见函数注释）
		for id := range r.dmNames {
			if _, seen := r.state.LastID[id]; !seen {
				r.state.LastID[id] = 0
			}
		}
		if len(r.groups) == 0 && len(r.dmNames) == 0 {
			logf("warn: 机器人当前没有任何会话")
		}
		return nil
	}
	if len(r.groups) == 0 && len(r.dmNames) == 0 && len(r.cfg.threads) == 0 {
		return fmt.Errorf("机器人还没有任何会话：把机器人拉进群，或在客户端与它开私聊后再运行")
	}
	return nil
}

// importLegacyGroups 解析旧版 /bot/groups 响应（仅群聊），整体替换 groups 表
// （与 refreshConversations 的替换语义一致：不在列表里的群退出轮询）。
func (r *agentRunner) importLegacyGroups(body []byte) {
	var resp struct {
		Data []struct {
			ConversationID uint64 `json:"conversation_id"`
			GroupName      string `json:"group_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		logf("warn: 解析降级群列表失败: %v", err)
		return
	}
	groups := make(map[uint64]string, len(resp.Data))
	for _, g := range resp.Data {
		groups[g.ConversationID] = g.GroupName
	}
	r.groups = groups
}

// pollThread 拉取一个会话的增量消息并逐条处理。
// 水位只在一条消息**处理完成**（回过 / 判定跳过 / 重试耗尽放弃）后推进，因此
// agent 失败时下轮会从同一条重来，不会因中间出错而漏消息。
func (r *agentRunner) pollThread(ctx context.Context, cfg config, threadID uint64) error {
	key := strconv.FormatUint(threadID, 10)
	if _, seen := r.state.LastID[threadID]; !seen {
		// 冷启动：把水位设为当前最新，只处理此后的新消息。
		// --replay 时保持 0，从最早一页开始（会把历史 @ 全部回一遍，务必慎用）。
		if r.state.LastID == nil {
			r.state.LastID = map[uint64]uint64{}
		}
		if r.cfg.replay {
			r.state.LastID[threadID] = 0
		} else {
			r.state.LastID[threadID] = latestMessageID(key)
			if r.cfg.verbose {
				logf("会话 %d 冷启动，跳到最新水位 %d", threadID, r.state.LastID[threadID])
			}
		}
	}

	msgs, err := fetchMessages(key, r.state.LastID[threadID], 100)
	if err != nil {
		return err
	}
	if len(msgs) > 0 {
		// 拉取即读：仅私聊上报——否则私聊里用户消息永远「未读」，已读回执形同虚设。
		// 群聊不上报：bot 读群消息的「已读」没有人际意义，写了还会污染已读名单
		//（服务端展示层虽已过滤 bot 回执，源头不写更干净）。失败仅 warn，不打断消息处理。
		if _, isGroup := r.groups[threadID]; !isGroup {
			if err := markThreadRead(threadID); err != nil && r.cfg.verbose {
				logf("warn: 会话%d 上报已读失败: %v", threadID, err)
			}
		}
	}
	for _, m := range msgs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if m.ID <= r.state.LastID[threadID] {
			continue
		}
		r.observe(threadID, m)

		if !r.shouldReply(threadID, m) {
			if r.cfg.verbose {
				logf("跳过 会话%d msg=%d type=%s sender=%s", threadID, m.ID, m.Type, m.SenderNickname)
			}
			r.advance(threadID, m.ID)
			continue
		}

		if err := r.handleOne(ctx, cfg, threadID, m); err != nil {
			r.attempts[m.ID]++
			if r.attempts[m.ID] >= agentMaxAttempts {
				// 放弃并推进：pull 模式没有服务端 outbox 兜底，卡在同一条上会让
				// 整个会话永久停摆，比丢一条更糟。失败原因留在 stderr 供排查。
				logf("error: 会话%d msg=%d 放弃（已尝试 %d 次）: %v",
					threadID, m.ID, r.attempts[m.ID], err)
				delete(r.attempts, m.ID)
				r.advance(threadID, m.ID)
				continue
			}
			// 水位不动，中止本批：同会话内保持顺序，下轮从这条重试
			_ = r.saveState()
			return fmt.Errorf("msg=%d 第 %d 次失败: %w", m.ID, r.attempts[m.ID], err)
		}
		delete(r.attempts, m.ID)
		r.advance(threadID, m.ID)
	}
	return nil
}

// advance 推进会话水位并落盘。
func (r *agentRunner) advance(threadID, msgID uint64) {
	if r.state.LastID == nil {
		r.state.LastID = map[uint64]uint64{}
	}
	r.state.LastID[threadID] = msgID
	_ = r.saveState()
}

// observe 把本轮看到的消息追加进该会话的滚动上下文（含被跳过的），维持最近 N 条。
func (r *agentRunner) observe(threadID uint64, m message) {
	if r.state.History == nil {
		r.state.History = map[uint64][]agentBriefState{}
	}
	text := strings.TrimSpace(m.Content)
	if text == "" {
		return
	}
	// @ token 还原成可读文本，agent 才知道谁被点了名
	text = mention.Decode(text)
	if runes := []rune(text); len(runes) > 200 {
		text = string(runes[:200]) + "…"
	}
	sender := m.SenderNickname
	if sender == "" {
		sender = "user#" + strconv.FormatUint(m.SenderID, 10)
	}
	h := append(r.state.History[threadID], agentBriefState{
		MsgID: m.ID, Sender: sender, Text: text,
	})
	if len(h) > agentHistoryLimit {
		h = h[len(h)-agentHistoryLimit:]
	}
	r.state.History[threadID] = h
}

// shouldReply 决定这条消息是否交给智能体。
// mention-only 只约束群聊（threadID 在 groups 表）：群外会话（1:1 私聊）的消息天然
// 是对着 bot 说的、不含 @ token，若同样过滤将永远沉默，故每条都回。
// threadID 不在 groups 表即视为群外——含群列表拉取失败的降级路径（仅显式 --thread
// 时才会走到那里，用户已显式圈定会话范围）。
func (r *agentRunner) shouldReply(threadID uint64, m message) bool {
	if !r.cfg.acceptTypes[m.Type] {
		return false
	}
	// 自己或别的 bot 发的：既不自问自答，也顺带避免两个 agent 互相触发成环
	if m.SenderType == "bot" || m.Origin == "bot" {
		return false
	}
	if r.selfID != 0 && m.SenderID == r.selfID {
		return false
	}
	if _, isGroup := r.groups[threadID]; !isGroup {
		return true
	}
	if !r.cfg.mentionOnly {
		return true
	}
	// @all 不响应：target 是 "all" 而非具体 id，IsMentioned 天然不命中
	return mention.IsMentioned(mention.Parse(m.Content), uint(r.selfID))
}

// handleOne 组提示词 → 调智能体 → 回写到同一会话（conversation_id 必须等于 thread_id，
// 只传 to_user_id 会另开一条单聊，回不到群里）。
//
// 回复走流式占位：先建一条 type=streaming 消息（客户端立即渲染 typing 气泡），
// 智能体产出后一次 delta 收尾；判定无需回复或调用出错则 abort 丢弃占位——pull 模式的
// 智能体调用常耗时十几秒，无占位时这段静默与「bot 挂了」在用户眼里无法区分。
// 占位建失败不阻断回复，降级为旧的整条发送。
func (r *agentRunner) handleOne(ctx context.Context, cfg config, threadID uint64, m message) error {
	prompt := r.buildPrompt(threadID, m)
	if r.cfg.verbose {
		logf("→ 会话%d msg=%d 交给智能体（%d 字）", threadID, m.ID, len(prompt))
	}
	if r.cfg.dryRun {
		fmt.Printf("---- dry-run 会话%d msg=%d ----\n%s\n", threadID, m.ID, prompt)
		return nil
	}

	placeholder, perr := sendMessageToConversation(threadID, "", "streaming", 0)
	if perr != nil {
		logf("warn: 会话%d 建流式占位失败（%v），降级整条回复", threadID, perr)
	}
	// abortPlaceholder 占位未正常收尾时丢弃，避免留下永久 typing 气泡
	//（服务端另有 CleanupStaleStreamingMessages 兜底，这里主动清更及时）。
	abortPlaceholder := func() {
		if perr == nil {
			if aerr := streamChunkAbort(placeholder); aerr != nil && r.cfg.verbose {
				logf("warn: 丢弃占位 %d 失败: %v", placeholder, aerr)
			}
		}
	}

	reply, err := runAgent(ctx, r.cfg.agentCmd, prompt, r.cfg.agentTimeout)
	if err != nil {
		abortPlaceholder()
		return err
	}
	if reply == "" {
		if r.cfg.verbose {
			logf("← 智能体未产出内容，丢弃占位 msg=%d", m.ID)
		}
		abortPlaceholder()
		return nil
	}
	if perr == nil {
		// 一次 delta 带全量内容并 finish：占位气泡转为最终 markdown
		if err := streamChunk(placeholder, reply, true); err != nil {
			abortPlaceholder()
			return fmt.Errorf("流式收尾失败: %w", err)
		}
	} else if _, err := sendMessageToConversation(threadID, reply, r.cfg.replyType, 0); err != nil {
		return fmt.Errorf("回写失败: %w", err)
	}
	if r.cfg.verbose {
		logf("← 已回复 会话%d msg=%d（%d 字）", threadID, m.ID, len(reply))
	}
	return nil
}

// buildPrompt 渲染提示词模板。所有值经占位符替换，不进入 shell 命令行。
func (r *agentRunner) buildPrompt(threadID uint64, m message) string {
	group := ""
	if name := r.groups[threadID]; name != "" {
		group = fmt.Sprintf("（群「%s」）", name)
	}
	history := r.formatHistory(threadID, m.ID)
	return strings.NewReplacer(
		"{thread}", strconv.FormatUint(threadID, 10),
		"{group}", group,
		"{sender}", m.SenderNickname,
		"{sender_id}", strconv.FormatUint(m.SenderID, 10),
		"{time}", m.CreatedAt,
		"{type}", m.Type,
		"{content}", mention.Decode(m.Content),
		"{history}", history,
	).Replace(r.cfg.promptTpl)
}

// formatHistory 输出最近上下文，排除当前这条（它已由 {content} 提供，重复会让
// 智能体把同一句读两遍）。
func (r *agentRunner) formatHistory(threadID, currentID uint64) string {
	h := r.state.History[threadID]
	lines := make([]string, 0, len(h))
	for _, b := range h {
		if b.MsgID == currentID {
			continue
		}
		lines = append(lines, b.Sender+": "+b.Text)
	}
	if len(lines) == 0 {
		return "（暂无）"
	}
	return strings.Join(lines, "\n")
}

// runAgent 执行智能体命令：prompt 走 stdin（不经命令行，杜绝内容注入），stdout 即回复。
// stderr 直连本进程 stderr，智能体的进度/思考能实时看到。
// timeout <= 0 表示不限时。
func runAgent(ctx context.Context, agentCmd, prompt string, timeout time.Duration) (string, error) {
	runCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, "sh", "-c", agentCmd)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stderr = os.Stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("智能体超时（%v）", timeout)
		}
		return "", fmt.Errorf("智能体执行失败: %w", err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// fetchBotIdentity 读取本机器人的 id 与显示名。
func fetchBotIdentity(cfg config) (uint64, string, error) {
	body, err := httpGet(cfg, cfg.ServerURL+"/api/v1/bot/me")
	if err != nil {
		return 0, "", err
	}
	var resp struct {
		Data struct {
			ID            uint64 `json:"id"`
			Name          string `json:"name"`
			VirtualUserID uint64 `json:"virtual_user_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, "", fmt.Errorf("解析身份响应失败: %w", err)
	}
	if resp.Data.VirtualUserID == 0 {
		return 0, "", fmt.Errorf("响应缺少 virtual_user_id: %s", truncate(string(body), 200))
	}
	return resp.Data.VirtualUserID, resp.Data.Name, nil
}

func defaultAgentStatePath() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, configDir, "agent-loop-state.json")
	}
	return filepath.Join(configDir, "agent-loop-state.json")
}

func (r *agentRunner) loadState() error {
	b, err := os.ReadFile(r.cfg.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.state = agentLoopState{LastID: map[uint64]uint64{}, History: map[uint64][]agentBriefState{}}
			return nil
		}
		return err
	}
	var st agentLoopState
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	if st.LastID == nil {
		st.LastID = map[uint64]uint64{}
	}
	if st.History == nil {
		st.History = map[uint64][]agentBriefState{}
	}
	r.state = st
	return nil
}

// saveState 原子写（临时文件 + rename），避免进程被杀时留下半份状态导致水位回退重放。
func (r *agentRunner) saveState() error {
	if r.cfg.statePath == "" {
		return nil
	}
	dir := filepath.Dir(r.cfg.statePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r.state, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".agent-loop-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, r.cfg.statePath)
}

// logf 统一走 stderr：stdout 留给 --output / dry-run 的可解析输出。
func logf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "["+CliName+" agent] "+format+"\n", a...)
}

// markThreadRead 上报会话已读（服务端以 bot 虚拟用户写回执）：拉取即读，
// 否则私聊里用户发出的消息永远「未读」。失败由调用方按 warn 处理。
func markThreadRead(threadID uint64) error {
	cfg := mustConfig()
	body, _ := json.Marshal(map[string]any{"thread_id": threadID})
	_, err := httpPost(cfg, cfg.ServerURL+"/api/v1/bot/read", body)
	return err
}

// sendHeartbeat 向服务端上报本 bot 在线（服务端 TTL 90s）。
func sendHeartbeat() error {
	cfg := mustConfig()
	_, err := httpPost(cfg, cfg.ServerURL+"/api/v1/bot/heartbeat", []byte("{}"))
	return err
}
