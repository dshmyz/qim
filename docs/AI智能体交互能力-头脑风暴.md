# AI 智能体交互能力头脑风暴

> 背景：QIM 作为企业 IM，已有分身（avatar）、群 AI 助手、AI 检索/摘要、向量检索、审批服务、MCP 工具绑定等底座。本文围绕"在通讯工具里能跟智能体怎么交互"发散 8 个方向，对照市面能力，标注 QIM 现状与缺口，并给出重点推荐与难度评估。
>
> 状态图例：🟢 已有 ｜ 🟡 部分/有底子 ｜ 🔴 缺口

## 一、市面通讯工具 AI/智能体能力速览

| 产品 | 智能体交互形态 |
|---|---|
| 飞书/Lark | 智能伙伴（群/文档/多维表格里嵌智能体，按行处理数据）、妙记会议纪要、智能体作为群成员有角色、开放平台 AI bot |
| Teams | Copilot（聊天/会议摘要、catch up、代起草）、Copilot Studio agent 部署进聊天可"代表你行动"、Loop 实时协作组件 |
| Slack | Slack AI（频道/线程摘要、搜索问答、每日 recap）、Agentforce agent 以"成员"身份进频道被 @、Workflow + AI 节点、Canvas |
| Discord/Telegram | bot 生态、嵌入式 mini-app（Telegram Bot API 2.0、Discord Activities SDK） |
| Zoom | AI Companion 会议内实时辅助、会后行动项提取 |

**共性趋势**：智能体从"被你问"变成"主动找你"、从"回答"变成"替你做"、从"一对一"变成"群里带角色、可被审计"。

---

## 二、8 个交互方向

### 1. 在哪聊（会话入口）

- 🟢 群里 @ 群 AI 助手
- 🟡 跟自己的分身 1:1 DM（现在分身是"替你回别人"，反向"你跟你的分身聊"没做出来）
- 🔴 侧边栏 meta 对话：不离开当前会话，侧边开一个"问 AI 关于这个会话"的窗（"总结这个线程""帮我起草回复那条"），回复只给自己看
- 🔴 输入框内联指令：`/总结` `/翻译` `/排期` 直接在任意输入框触发，结果以卡片回填
- 🔴 hover "问 AI"：悬停在单条消息/文件上，一键"解释这段""这文件要点"

### 2. 让智能体干活（委派/行动）

- 🟡 异步任务委派：`@分身 帮我把这周的群文件整理成清单发给我`，agent 跑完回报。QIM 有 MCP 工具绑定 + 任务/文件/日历，agent 真能执行
- 🔴 事件触发型 agent workflow：新文件入群->自动摘要入库；新成员入群->agent 推 onboarding；关键词命中->建任务。把 anomaly_detector + 群级记忆串成"事件->动作"
- 🔴 批量委派：`给这三个群各发一份通知并收集回复`，agent 扇出执行
- 🔴 会议中实时 agent：realtime/屏幕共享已有，加一个"会议 copilot"——实时纪要、捕捉 action item、会后落进任务

### 3. 智能体主动找你（proactive）

- 🟡 离线摘要推送："你不在时群里有 3 件要紧事"+ 链接跳转。群级记忆 + 消息检索能做
- 🟢 异常告警（anomaly_detector 已有），但 🔴 缺"智能体用自然语言解释这条异常 + 建议处置"
- 🔴 跟进提醒：agent 监测"你答应别人但没做"的消息，主动催你
- 🔴 每日站会/digest bot：定时把任务/未读/会议生成早报推给你

### 4. 智能体知道什么（知识/记忆）

- 🟢 分身记忆可查可删
- 🔴 跨源 workspace Q&A：一份问答横跨聊天+文档+笔记+任务（Notion AI connectors 打法；现笔记/群文档各自向量检索）
- 🔴 记忆可解释/可编辑：分身为什么记得这个？重要度多少？让用户改、标"忘掉这类"。现在只能删
- 🔴 群级知识库自动维护：群 AI 自己把群里的决策/文档沉淀成结构化知识，而非只检索

### 5. 多智能体协作

- 🔴 专家 agent 花名册：HR 助手、IT helpdesk、法务助手——各是有专属知识范围的分身，按群/按 @ 选取。企业 IM 差异化点
- 🔴 agent 间 handoff / 路由：一个"调度 agent"判断该交给哪个专家，或你的分身把任务丢给群 AI
- 🔴 双 agent 辩论/会诊：让两个专家就一个问题讨论给你看（适合决策类场景）

### 6. 人在环里（控制/治理）

- 🟢 审批服务 + 分身接管已有（很好的种子）
- 🔴 审批队列 UI：把"agent 待执行的动作"做成可视化队列（代发消息、代建任务、代删文件），逐条批准/批量批准/改了再批
- 🔴 透明度面板："分身为什么回/没回这条"（trigger-check 已有，做成用户可见侧栏）
- 🔴 agent 行为审计 + 撤销：所有 agent 动作可回溯、可撤回（企业合规刚需）

### 7. 富交互（卡片/嵌入式）

- 🔴 agent 渲染交互卡片：agent 回一张带按钮/表单的卡（"约这个时间？""选 A 还是 B"），人一键确认。飞书/Slack 杀手锏，QIM 缺
- 🔴 嵌入式 mini-app / canvas：agent 在聊天里开一个共享实时文档/白板，双方共同编辑（Loop/Canvas 打法）
- 🔴 有脸有嗓的分身：视频通话里分身用 TTS+虚拟形象替你出席简单会议

### 8. 跨系统（外部）

- 🟢 avatar 绑 MCP 工具（agent 能调工具）
- 🔴 agent 作为 MCP client 调外部系统：Jira/CRM/邮箱/日历——把企业其它系统接进 agent
- 🔴 BYO agent：外部智能体注册进 QIM（Slack Agentforce 打法），第三方能力即插即用
- 🔴 跨渠道代理：agent 帮你盯着外部邮件/IM，重要的事桥接进来

---

## 三、重点推荐 5 个

1. **侧边栏 meta 对话**（"问 AI 关于这个会话"）——最高频、最快出价值，补 QIM 最大缺口
2. **异步任务委派 + 审批队列**——把已有的分身+MCP+审批串成"agent 真的替你干活且可控"，企业 IM 区别于消费级 IM 的核心
3. **agent 交互卡片**——把"问答"升级成"行动闭环"，体验跃迁最明显
4. **专家 agent 花名册 + 群内带角色**——企业差异化（HR/IT/法务），分身框架天然能扩展成多角色
5. **proactive digest + 跟进提醒**——让 agent 从被动变主动，留存感最强；群级记忆/anomaly 已铺好一半

**归类**：1、3 是"让交互更好"（体验层，见效快）；2、4 是"让 agent 真能干活"（能力层，企业护城河）；5 是"让 agent 主动"（留存层）。

---

## 四、难度评估（粗估，一个熟悉 QIM 的人）

| # | 功能 | 难度 | MVP | 完整版 | 已有底子 | 真正难的点 |
|---|---|---|---|---|---|---|
| 1 | 侧边栏 meta 对话 | 中 | 1-2周 | ~3周 | 消息检索、摘要/搜索 handler、流式、OverlayManager 面板系统 | 上下文组装策略（取多少消息、要不要向量召回、怎么截断不爆 token） |
| 3 | 交互卡片 | 中 | ~2周 | 3-4周 | Element Plus、MessageListView、动作 API | 卡片 schema + 动作回调路由 + 权限 |
| 5 | digest + 跟进 | 中->大 | 2-3周（仅 digest） | 5-6周 | 群级记忆、anomaly_detector、notification_handler | 跟进提醒要识别"用户答应了什么"且持续跟踪"做了没" |
| 4 | 专家 agent 花名册 | 中 | ~3周 | 4-5周 | 分身框架本身是可配置 agent 实例 | 多 agent 路由、按角色装专属知识、provisioning 后台 |
| 2 | 异步委派 + 审批队列 | 大 | 2-3周（窄版） | 6-8周 | ApprovalService、MCP 工具、动作 API | 要造一个多步 agent 执行 runtime（意图->计划->分步执行->卡审批->恢复） |

**关键判断**：只有 #2 是"造新运行时"，其余四个本质是"把已有零件拼起来"。#5 的 digest 与跟进要分开看（digest 中等，跟进接近大）。

---

## 五、建议起步顺序

- **见效优先**：1 -> 3 -> 5(digest) -> 4 -> 2
  - 1、3 最快出可感知价值，且为后面铺路（卡片是 #2 审批队列的交互载体，meta 对话是 #4 专家的容器之一）
- **押差异化优先**：4 -> 2（窄版）-> 3 -> 1 -> 5
  - 直接攻企业护城河，但前期见效慢、风险高

> 待验证项：#1 的“上下文组装”与 #2 的“runtime 要不要造”目前靠推断，真要落地前值得先摸对应代码确认。

---

## 六、对外交互方式（与 Claude Code / OpenCode 等编码智能体）

### MCP 现状（已调查）
- `ai/mcp.go` 的 `MCPServer` 是**自定义 HTTP**（`GET /tools`、`POST /execute`），不是标准 MCP 协议（JSON-RPC over stdio / Streamable HTTP），Claude Code/OpenCode 不能直接消费
- 但 `MCPTool` 接口（Name/Description/Parameters/Execute）形状对得上，`CallerContext` 提供权限模型，**registry 可复用**
- 已注册工具：运维 + 知识（KnowledgeSearch/Save/MemorySearch via `UnifiedMCPBridge`）+ 群管理；**缺 IM 类工具**（send/get/search messages、create_task、request_approval）
- `handleExecuteTool` 传 `nil` ctx，对外暴露需接 token -> CallerContext 鉴权
- 附带发现：`pkg/scheduler`（robfig/cron/v3）已有统一调度器，#5 proactive 不缺定时框架，难度可降回中等

### 其它方式（不止 MCP）
| 方式 | 方向 | 适合 |
|---|---|---|
| MCP | pull / 结构化 | 原生工具发现 |
| CLI（agent 走 Bash 调 `qim`） | pull / 非结构化 | 任何 agent、人机共用、最易调试 |
| REST API + 通用 HTTP 工具 | pull | 最通用、跨语言 |
| WS 命令通道 | 双向 | 低延迟、复用现有 WS |
| Webhook | push | 事件驱动回调 |
| Claude Code 专属：Hooks / Slash commands / Skills / Subagent / CLAUDE.md | 宿主集成 | 深度绑 CC 体验 |
| A2A 协议 | agent 间 | 前向，生态早 |

**关键洞察**：Claude Code/OpenCode 本来就靠 Bash 驱动外部工具（git/gh/kubectl），所以一个设计良好的 `qim` CLI **同时就是 agent 接口**，最低摩擦且人机共用。

### 推荐组合拳
```
agent/人 ─┬─ Bash ─> qim CLI ─┐
          └─ MCP adapter ─────┴─> 统一 REST 后端
QIM 事件 ─> Webhook/WS ─> agent（push）
+ CLAUDE.md / skill 教化 agent 用 CLI
```
CLI 做底座（先做、人能用），MCP 做结构化层（适配层薄、后加），Webhook/WS 做 push。三者共享同一后端，非互斥。

### MCP 适配工作量
- 标准 MCP 适配器（stdio + Streamable HTTP）+ auth：~1 周
- IM 工具（send/get/search messages、create_task、request_approval）包装现有 service：1-2 周
- `subscribe_events` 反向通道：~1 周
- MVP 合计 3-4 周

---

## 七、Agent↔用户消息闭环（主线方案）

把前面散点收拢成一个完整闭环：**agent 能主动找用户、用户能在熟悉的 IM 里回、回复能回到 agent**。本质是 Bot API + webhook 模型，QIM 已有一半底子。

### 核心模型
```
agent ──POST 消息──> QIM ──推用户──> 用户在 QIM 看到
用户在 QIM 回复 ──> QIM 存消息 + 检测“agent 会话的回复”
                ──webhook/WS 回调──> agent（带 thread + 上下文）
```

### 要做的几块
1. **Agent 身份（bot 账号）**：bot/app 账号类型 + token；消息标 `origin=agent`/`sender_type=bot`；客户端显示 bot 徽标。复用 messages `origin` 字段
2. **出站 agent -> QIM**：`POST /api/v1/bot/messages`（bot token 鉴权），带 `thread_id` + 可选 `metadata`（任务上下文，回复时原样带回）
3. **会话落点**：首次推送自动建 agent↔user DM，用户会话列表见 agent。复用 avatar `groupReplyTarget:'private'` + sessions
4. **入站路由（核心新东西）**：用户回复进库走正常流程；QIM 检测会话有 agent 参与方 -> webhook POST 或 WS 推 `{type:'agent_reply'}`，带 thread_id + metadata。**会话即线程**，任务级 correlation 由 message metadata 交给 agent 自管
5. **结构化回复**：agent 发卡片带按钮，点击走 action callback 路由回 agent。审批/选择场景刚需
6. **客户端 UX**：agent 作为联系人 + bot 徽标 + 输入状态；卡片嵌消息内；支持静音
7. **治理**：token + scope、rate limit（复用 middleware）、用户 mute/block、audit（复用 operation_log_handler）
8. **注册后台**：注册 agent 拿 token、配 webhook、配权限

### 与前面点的关系
| 早先的点 | 在此闭环中的角色 |
|---|---|
| 交互卡片（#3） | 结构化回复载体 |
| 审批队列（#2） | 特例：审批卡 -> 用户批准 -> 路由回 agent 执行 |
| proactive digest（#5） | agent 主动推的一种内容 |
| MCP 适配器 | 给 agent 提供“发消息”工具，复用同一条出站 API |

### MVP 切法
1. 纯文本往返：bot 身份 + 出站 + webhook 回调 + 基础治理（2-3 周）— ✅ **已落地（2026-07-23）**，方案见 [落地MVP第一步](./AI智能体交互能力-落地MVP第一步.md)
2. 加卡片按钮：结构化回复（+1 周）- ✅ **已落地（2026-07-23）**，方案见 [.claude/plans/bot-interactive-cards-step2.md](../.claude/plans/bot-interactive-cards-step2.md)
3. 加 WS 实时通道：低延迟在线往返（+1 周）- ✅ **已落地（2026-07-24）**：流式回复给用户（agent `SendOutbound(msg_type:"streaming")` 建流式消息 + 多次 `POST /bot/messages/:id/stream` 累加 content_delta + `finish:true` 收尾转 markdown，经 `message_updated` WS 推送，客户端气泡由 StreamingMessage 转 markdown），方案见 [.claude/plans/bot-streaming-step3.md](../.claude/plans/bot-streaming-step3.md)
4. 端到端打通 agent 闭环（CLI pull 底座）- ✅ **已落地（2026-07-24）**：`qim` CLI（`messages list/poll`、`send`、`stream-stdin`）+ `GET /bot/messages` pull 端点 + 客户端 token/webhook 配置 UI（BotConfigDialog）+ agent-loop.sh 示例，真 server 实测 messages/send/stream 全通，方案见 [.claude/plans/qim-cli-agent-loop-step4.md](../.claude/plans/qim-cli-agent-loop-step4.md)
5. 加后台注册/管理

### 复用盘点
messages `origin`、avatar sessions + private reply、WS `{type,data}`、ApprovalService、operation_log_handler、middleware 限流。**真正新写的核心只有 bot 身份 + 入站回复路由**，其余拼现有零件。
