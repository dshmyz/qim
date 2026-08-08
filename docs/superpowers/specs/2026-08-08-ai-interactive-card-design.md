# AI 交互卡片协议设计

## 背景与目标

QIM 的群 @AI 已打通外部 MCP + ReAct 循环，能调用工具、流式反馈（`tool_call🔧` 卡片）。用户希望进一步让 AI **在聊天里发出可交互的卡片**——不止是"看完就结束"的文字，而是"点一下能确认、能填表、能跳详情、能触发下一步"的富交互消息。

目标是：**定义一套「AI 交互卡片」协议，后端为核、前端哑渲染**，使 AI 能稳定地发出多种交互卡片，且新增卡片类型不需要前端改动。

### 数据落地实证（决定行为原语的关键勘察）

勘察现有渲染链路（`CardMessage.vue` / `AIAnswerBubble.vue` / `useMarkdownRender` / `Main.vue` 的 `handleToolCall` / `BotMessagingService.UpdateMessageContent`）后确认：

| 已有能力 | 位置 | 结论 |
|---------|------|------|
| 卡片 = 消息 `content` 存 JSON | `CardMessage.vue`（`{title,text,buttons}`） | ✅ 已跑通 |
| 卡片点击幂等提交 | `useBotCardAction` + CardActionRecord | ✅ 已验证 |
| agent 回写卡片 content 更新状态 | `BotMessagingService.UpdateMessageContent` | ✅ 多状态卡通道已存在 |
| 卡片类型 → 事件 → 原位状态更新（running→ok/error） | `handleToolCall` + `tool_calls` 数组 | ✅ 同模式已跑通 |
| 卡片跳业务详情 | `TaskRefCard.vue`（点击 → `TaskDetailModal`） | ✅ 已存在 |

**结论**：QIM 已具备"消息=载体 + WS 通道 + 幂等提交 + 状态回写 + 内部跳转"**全部地基**。缺的不是卡片能力，而是**把卡片收拢成一套统一协议**——目前每种卡片单点实现，加新卡要重写。

### 参照：Claude / Trae 等智能体的交互模式

业界智能体与用户对话的交互抽象为 **5 种原子动作**：

| # | 模式 | 用户实际做的 |
|---|------|------------|
| 1 | 一句话确认（yes/no） | 确认 / 取消 |
| 2 | 从候选中选（单选/多选） | 选一个 |
| 3 | 补参数（表单） | 填几个字段 |
| 4 | 明确动作批准 | 批准 / 拒绝 |
| 5 | 进度 / 结果展示 | 观察状态 |

1、2、3、4 本质都是"用户交付一个决定或一段数据"，区别只在**交付方式**；5 是纯展示。因此行为原语**按"用户如何交付这个决定"划分，不按业务划分**。

## 已确认的决定

1. **行为原语**：`confirm` / `select` / `form` / `status` + `unknown` 兜底。前端只实现这 4 个行为组件 + 兜底。
2. **业务全押到 `ref.type + data`**，由后端解释，前端不感知业务。
3. **前端只认 3 种动作**：`redirect`（应用内跳详情）/ `submit`（交数据）/ `ack`（对话信号）。
4. **复用**：消息=卡片载体、WS 事件通道、`UpdateMessageContent` 状态回写、CardAction 幂等记录。新增范围仅为后端 `card` 小包 + 前端 `CardMessage` 内轻注册表。
5. **本次不引入**：前端通用卡片框架重构、文档/表格文件生成依赖（LibreOffice 类）、新消息总线/第三方基础设施。

## 设计

### 1. 统一信封（协议定稿）

每张卡 = 一条消息的 `content` 存 JSON：

```json
{
  "v": 1,
  "kind": "confirm" | "select" | "form" | "status",
  "title": "可选标题",
  "text": "可选说明文字",
  "ref": null | { "type": "task", "id": 123 },
  "data": null | { ... },
  "ui": { ... }
}
```

**分层职责**：
- `v` / `kind` / `title` / `text` —— 前端直接用
- `ref` —— 前端用来"跳转"（`ref.type + ref.id` 定位业务详情页）
- `data` —— 后端业务数据，**前端绝不解析、只随提交原样回传**，由后端解释
- `ui` —— 前端按 `kind` 渲染的依据

> 关键约定：**前端只理解 `kind` 和 `ui`，绝不解 `data`**。`data` 是后端与自身之间的私密通道，提交时原样带回。这是"前端哑渲染、业务全在后端"的落地。

### 2. 各 kind 的 `ui` 结构

**confirm**（确认/批准）
```json
{
  "kind": "confirm",
  "text": "确认创建待办「产品评审」？",
  "ref": { "type": "task" },
  "ui": { "confirmText": "确认创建", "cancelText": "取消", "tone": "primary|danger" }
}
```

**select**（从候选中选）
```json
{
  "kind": "select",
  "ref": { "type": "doc" },
  "ui": {
    "multiple": false,
    "options": [{ "value": "weekly", "label": "周报模板" },
                { "value": "meeting", "label": "会议纪要模板" }]
  }
}
```

**form**（补参数）
```json
{
  "kind": "form",
  "title": "创建待办",
  "ref": { "type": "task" },
  "ui": {
    "fields": [
      { "name": "title",    "label": "标题",   "type": "text",   "required": true },
      { "name": "assignee", "label": "负责人", "type": "text" },
      { "name": "due",      "label": "截止日", "type": "date" },
      { "name": "priority", "label": "优先级", "type": "select",
        "options": [{ "value": "high", "label": "高" }, { "value": "medium", "label": "中" }] }
    ],
    "submitText": "创建"
  }
}
```

**status**（只读状态展示）
```json
{
  "kind": "status",
  "title": "任务已创建",
  "ref": { "type": "task", "id": 123 },
  "ui": {
    "state": "success|info|warning|error",
    "detail": "已创建·负责人张三·截止明天",
    "actions": [{ "id": "open", "text": "查看详情", "action": "redirect" }]
  }
}
```

### 3. 动作按钮统一形态

按钮统一为：
```json
{ "id": "open", "text": "查看详情", "action": "redirect" }
```

前端只认识 **3 种动作**：
- `action: "redirect"` → 按 `ref.type + id` 应用内跳详情（纯导航，不回后端）
- `action: "submit"` → 收集输入后提交（form/select）
- `action: "ack"` → 纯回执，告知 AI"我知道了"，不触发业务跳转

> `redirect` 走 QIM 内部路由，须复用一个可靠的内部跳转函数（避免重蹈 id 类型不匹配的坑，见项目 [channel-id-type-mismatch] 教训）。

### 4. 完整链路（以 confirm → submit 建待办为例）

```
[1] 用户群聊 @AI："帮我把周五产品评审安排成任务"
[2] 后端 ai_service · ReAct 命中 create_group_task 工具
      · 先落库任务（拿到 taskId）
      · 生成 confirm 卡 payload 作为一条消息 content 写入会话
[3] 后端 → WS 推新消息 → 群内全体渲染 confirm 卡
[4] 用户点「确认创建」→ 前端 POST /api/v1/ai/cards/{message_id}/action
      body: { action_id:"confirm", data:{...} }
      （select/form 则为 { selections:[...] 或 values:{...} }）
[5] 后端 card-handler
      · 鉴权（scope 判定，见 §6）
      · 按 action_id 分派
      · 幂等（同一卡同一动作不重复执行）
      · 落库 / 执行业务 / 广播
      · 返回展示结果；若状态变则 WS 广播更新后的 payload
[6] 前端 · 发起者收到结果 → 卡变"已确认"或 redirect 跳详情
     · 群其他成员收到 WS 更新 → 卡变"已确认/已完成"
```

复用：消息=载体、WS 事件通道、`UpdateMessageContent` 状态回写、CardAction 幂等。**真正新增的只有第 4/5 跳**（动作回传 + card-handler）。

### 5. 新增后端模块（小而独立）

`card` 包（或 `ai/cards.go`）：
```
card/
├── types.go      # kind/枚举、payload 结构、动作请求结构
├── handler.go    # POST /cards/{message_id}/action 的 HTTP handler
└── registry.go   # (业务, 动作) → 执行器注册表
```

执行器按"业务"注册动作处理器：
```go
// 每个业务告诉 registry "confirm 这个动作点下去我要干什么"
register("task:confirm", func(req ActionReq) (*Result, error) {
    // 落库任务、广播、返回新 payload
})
```
- 新增一种业务卡 = 注册一个新的 handler，不碰通用链路。
- `data` 字段在 handler 内被后端解包，前端全程不碰。

### 6. 权限与身份（scope 影响半径模型）

`confirm`/`select`/`form` 卡在群里出现后，必须回答"谁有资格点、点了改什么范围"。

**核心洞察**：鉴权不是"能不能点"，而是"**点下去的影响半径**"。分三类在协议里显式声明：

| 权限型 | 谁能点 | 点了改什么 | 例子 |
|--------|-------|-----------|------|
| `self`（个人职权） | 仅创建者/触发人 | 仅改"我的"状态 | "确认建我的待办" |
| `group`（群组共享） | 群成员皆可 | 改"群的"共享状态，全体可见 | 投票选方案、确认采购 |
| `role`（角色/被@） | 特定角色或被@者 | 改指定对象状态 | 审批流，仅审批人可批 |

`ref` 不只指引跳转，还暗含影响半径。在 `ui` 里显式声明：
```json
{ "ui": { "scope": "self|group|role", "role": "approver" } }
```
`card-handler` 在分派前先按 scope 判定：`self`→查是否本人；`group`→查是否群成员；`role`→查角色。

**为什么必须显式**：会改共享状态的操作（群投票、确认消耗群资源）若被任意群成员一按就生效，是权限漏洞。`scope` 是把"按一下"和"按一下有没有权力"隔离开的闸。

### 7. 回环：卡片动作 → AI 下一步（连续交互）

现实对话是来回的：AI 发 confirm → 用户确认 → AI 再发 form → 用户填 → AI 发 status。后卡由前卡动作引发，需显式化回环。

**模型 A（采纳）：点击动作 = 注入一条用户消息**
- 用户点卡 = 向会话注入一条用户消息，其内容是"用户对某张卡做了某动作"（如"用户确认了 task:confirm"）。
- 后端把这次点击**作为一条新的用户消息喂进 AI 对话上下文**。
- AI 基于这次反馈自然生成下一条（form 卡 / status 卡）。
- **好处**：完全复用"用户消息 → AI 回复"管道，上下文连续性天然成立，无需另建"卡回调"机制。

**模型 B（否决）**：卡点击走后端 card-handler 独立回调，handler 内再手动触发新的 AI 生成。坏处：全新链路 + 手动维护"该点击对应哪个 AI 会话、带不带历史"，易与主对话脱节。

**回环里的权限传递（与 §6 交汇）**：
- 用户点卡注入的消息，**按点击者身份**进入 AI 上下文（张三点了 → "张三确认了"）。
- 这样 AI 后续生成的卡（form 填写人、status 归属）自然跟随"当前点击者张三"，权限从卡传递到下一轮 AI 上下文，保持一致。
- **必须钉死**：`card-handler` 注入消息时带点击者真实身份；否则若按原始创建者身份注入，会出现"王五点确认、王五资料被当成张三"的权限错位。回环越深，错得越离谱。

### 8. 前端改动（极小）

只在 `CardMessage.vue` 一个组件里做：
- 加一个 `kind → 子组件` 的轻注册表（`ConfirmCard / SelectCard / FormCard / StatusCard` + `UnknownCard`）。
- 加一次统一出口："点击 → 组装动作请求 → POST → 处理 redirect/submit/ack 结果"。
- **不加业务** —— 待办/文档/投票都是上述 4 种 kind，前端零业务代码。

```js
// 统一入口，只认得 kind
function renderCard(payload) {
  return renderers[payload.kind]?.(payload)
      ?? <FallbackCard data={payload}/>   // 未知 kind 兜底，不白屏
}
```

### 9. 错误 / 超时 / 幂等

- **幂等**：同 `message_id + action_id` 重复提交 → 返回已处理结果，不二次执行（沿用 CardActionRecord 模式）。
- **超时/失败**：card-handler 返回 error → 前端卡显示"失败 + 重试"（走 `status` 的 error 态或动作触发重试）。
- **降级**：`unknown` 兜底，未知 kind 显示原始 JSON + "此卡片暂不支持"，绝不白屏。

## 测试

- **后端（重点）**：
  - payload 解析/校验：无效 JSON、缺字段、未知 kind → 兜底不崩
  - 幂等：同 `message_id+action_id` 二次提交 → 不重复执行
  - 鉴权 scope：`self/group/role` 三种权限型各自的越权拒绝
  - 执行器：每种 kind 的 handler 单测（confirm 落库、form 校验必填、select 单选/多选）
  - 广播：状态变更 → WS 事件带新 payload 正确
- **前端（轻量）**：`CardMessage` 注册表 4 种 kind + unknown 渲染正确；动作出口组装/结果处理。复用 Vitest + Playwright。

## 涉及文件

- 后端（新增/改动）：
  - `qim-server/ai/cards.go` 或 `qim-server/service/card/`（types + handler + registry）
  - `qim-server/app/routes.go`（注册 `POST /v1/ai/cards/{message_id}/action`）
  - `qim-server/handler/ai_handler.go`（卡片消息类型/渲染分支接入）
- 前端：
  - `qim-client/src/components/message/CardMessage.vue`（kind 注册表 + 动作出口）
  - `qim-client/src/components/message/` 下新增 `ConfirmCard/SelectCard/FormCard/StatusCard/UnknownCard.vue`

## 落地顺序（MVP 切片）

**第 1 步（MVP）——把"建待办闭环"打通**
- 后端 `card` 最小骨架（types + handler + 单测）
- 前端 `CardMessage` 加注册表 + `confirm` 卡
- 目标：`confirm` 一条路走通"建任务 → 确认 → 落库 → 状态更新"；`status` 卡顺带（落库后要展示结果）；`unknown` 兜底第 1 步就带

**第 2 步——补 `form` 卡（输入表单）**
- 覆盖"负责人/截止/优先级"这类补参数场景；后端加 `form` 执行器

**第 3 步——补 `select` 卡 + `redirect` 跳转**
- 覆盖"从候选选"（模板、方案、投票）；`redirect` 接上 QIM 任务/文档详情页可靠跳转

## 范围外（明确不做）

- 不做通用前端卡片框架的重构，只在现有 `CardMessage` 内加注册表。
- 不做文档/表格文件生成（LibreOffice 类）；文档卡产物落到群文档/笔记，用 MarkdownRenderer 渲染，不引入文件生成依赖。
- 不改现有 bot 按钮卡链路，兼容为主（`title/text` 类旧卡照常渲染）。
- 不引入新基础设施，复用消息/WS/CardAction，不加消息总线、不引第三方。

## 开放项（需后续定夺）

1. **`redirect` 首批接哪些业务页**：任务详情已有，文档/群会话是否一起。
2. **卡片权限粒度细化**：`scope` 的三类是否够用；哪些卡允许全体点（确认/投票）、哪些仅创建者点（改状态），在 registry 里明确。
3. **`data` 的敏感度**：卡片 content 会随消息持久化，`data` 不应带不该长期存储的临时信息（如内部 prompt 片段）；只放展示/回传所需最小字段。
