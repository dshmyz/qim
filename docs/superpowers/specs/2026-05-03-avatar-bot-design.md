# 用户分身（Avatar）功能设计

## 概述

用户可以开启 AI 分身代替自己回复消息。分身自动学习用户的说话风格，在用户设定的规则下自动回复，回复时标注"AI 代回复"。用户发消息即接管，分身自动暂停。

**核心设计决策：分身不在群聊中直接发话。** 群聊中的分身触发后，以私聊方式回复对方。这样从根本上避免了多分身并发导致的 LLM 限流和 WebSocket 广播风暴。

## 回复模式

| 会话类型 | 分身行为 | 说明 |
|---------|---------|------|
| **私聊** | 直接在当前对话中回复 | 消息带"AI 代回复"标记 |
| **群聊** | 不在群里发话，改为私聊回复触发者 | 群聊保持干净，对方通过私聊收到分身回复 |

### 群聊分身回复流程

```
群里有人 @用户 或触发关键词
  → 分身检测到触发
  → 不在群聊中发送消息
  → 自动创建/找到与触发者的私聊会话
  → 在私聊中发送：
    「[群聊 XXX 中 @你] 张三的分身代为回复：
     {分身回复内容}
     ——此消息由 AI 分身自动生成」
  → 同时给用户本人推送通知：
    「你的分身在群聊 XXX 中代你回复了李四」
```

## 定位

| 维度 | 群组 AI 助手 | 独立 Bot | 分身（Avatar） |
|------|-------------|---------|---------------|
| 身份 | 群的公共助手 | 独立机器人 | 用户的数字替身 |
| 控制权 | 群管理员 | 创建者 | 用户本人 |
| 回复身份 | "AI助手" | "XX机器人" | "张三（AI代回复）" |
| 人设来源 | 群管理员配置 | 创建者配置 | 自动学习用户风格 |
| 知识来源 | 群知识库 | 系统提示词 | 用户历史+知识库+笔记 |
| 交互模式 | 嵌入群聊 | 独立对话窗口 | 私聊直接回复 / 群聊私聊回复 |
| 核心价值 | 群效率提升 | 定制化 AI 能力 | 用户不在时的代理 |

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 隐私：分身可能泄露敏感信息 | 学习阶段过滤敏感内容；回复前敏感信息检测；用户可预览/编辑人设；明确告知数据范围 |
| 误导：他人以为在和人对话 | 强制标注"AI 代回复"；消息样式有视觉区分；私聊回复中明确标注来源群聊 |
| 失控：分身说出用户不会说的话 | 用户发消息即接管；可选半自动模式（预览确认）；分身消息可撤回；全局紧急关闭开关 |
| 滥用：变相刷屏 | 分身不在群聊发话，私聊回复天然限流；防刷屏间隔；全局限流策略 |
| 学习不足：新用户效果差 | 消息不足时提示；允许手动补充提示词；提供风格预览 |
| 性能：多用户同时触发 | 全局限流 + Worker Pool + 配置缓存（详见性能防护章节） |

## 数据模型

```typescript
interface AvatarConfig {
  id: number
  userId: number
  name: string
  enabled: boolean

  autoLearnedPersona: string
  customPersonaAddon: string
  personaVersion: number
  lastLearnedAt: string | null

  knowledgeScope: {
    conversationHistory: boolean
    knowledgeDocs: boolean
    notes: boolean
    tasks: boolean
  }

  triggerRules: {
    mode: 'offline' | 'keyword' | 'mention' | 'all' | 'custom'
    keywords: string[]
    timeRanges: TimeRange[]
    excludedConversations: number[]
  }

  replyStrategy: {
    maxReplyLength: 'short' | 'medium' | 'long'
    replyDelay: number
    confidenceThreshold: number
    disclaimerStyle: 'badge' | 'footer' | 'both'
  }

  modelConfigId: number | null
  useSystemConfig: boolean

  takeoverCooldown: number

  createdAt: string
  updatedAt: string
}

interface TimeRange {
  dayOfWeek: number[]
  startHour: number
  endHour: number
}

interface AvatarSession {
  conversationId: number
  avatarEnabled: boolean
  takeoverUntil: string | null
  lastReplyAt: string | null
}
```

## API 设计

```
GET    /api/v1/avatar/config              获取分身配置
POST   /api/v1/avatar/config              创建分身配置
PUT    /api/v1/avatar/config              更新分身配置
DELETE /api/v1/avatar/config              删除分身配置

POST   /api/v1/avatar/learn-persona       触发风格学习（异步）
GET    /api/v1/avatar/learn-status         查询学习进度
GET    /api/v1/avatar/learned-persona      获取学习结果

GET    /api/v1/avatar/sessions             获取会话分身状态
PUT    /api/v1/avatar/sessions/:convId     开启/关闭会话分身
POST   /api/v1/avatar/sessions/:convId/takeover  手动接管

POST   /api/v1/avatar/generate-reply      生成分身回复（不发送）
POST   /api/v1/avatar/send-reply          发送分身回复
POST   /api/v1/avatar/preview             风格预览
```

## 消息流

### 私聊场景

```
收到新消息（私聊）
  → 该会话是否开启分身？ 否 → 正常处理
  → 是否在接管冷却期？ 是 → 正常处理
  → 是否满足触发规则？ 否 → 正常处理
  → 置信度检查 低于阈值 → 通知用户"有消息需要你回复"
  → 满足条件
    → 延迟 replyDelay 秒
    → 后端：加载人设 + 知识上下文 → LLM 生成 → 敏感检测
    → 在当前私聊中发送（is_avatar_reply: true）
    → 前端：显示"AI 代回复"标记 + 视觉区分
```

### 群聊场景

```
收到新消息（群聊）
  → 该群是否有人开启分身？ 否 → 正常处理
  → 消息是否触发某用户的分身规则？（@用户 / 关键词 / 离线）
  → 否 → 正常处理
  → 是
    → 延迟 replyDelay 秒
    → 后端：加载人设 + 知识上下文 → LLM 生成 → 敏感检测
    → 不在群聊中发送
    → 找到/创建 分身用户 ↔ 触发者 的私聊会话
    → 在私聊中发送：
      「[群聊 XXX 中 @你] {用户名}的分身代为回复：{内容}」
    → 通知分身用户：「你的分身在群聊 XXX 中代你回复了 {触发者}」
```

## 性能防护

### 问题分析

分身每次回复需要 1 次 LLM API 调用（1-5s）+ 3-5 次 DB 查询 + WebSocket 推送。多人同时触发时，LLM API 限流是硬瓶颈。

**群聊私聊回复模式已从根本上缓解了最严重的性能问题**（避免了 N 个分身 × M 个群成员的广播风暴），但仍需以下防护：

### 1. 全局 LLM 限流

```go
type AvatarRateLimiter struct {
    globalLimiter *rate.Limiter  // 全局 RPM 限制
    userLimiters  sync.Map       // userId -> *rate.Limiter（用户级上限）
}

// 配置
const (
    AvatarGlobalRPM     = 30     // 全局每分钟最多 30 次分身 LLM 调用
    AvatarUserRPM       = 10     // 每用户每分钟最多 10 次
    AvatarUserDaily     = 100    // 每用户每天最多 100 次
)
```

- 系统默认模型共享 API Key，全局 RPM 受上游限制
- 用户自配模型（自己的 API Key）不受全局限制，但仍受用户级限制
- 超限时排队等待，非丢弃（避免用户感知丢失消息）

### 2. Worker Pool

```go
type AvatarWorkerPool struct {
    queue   chan AvatarTask     // 缓冲队列，容量 100
    workers int                 // 并发 worker 数 = 5
}

// 启动时初始化
func NewAvatarWorkerPool() *AvatarWorkerPool {
    p := &AvatarWorkerPool{
        queue:   make(chan AvatarTask, 100),
        workers: 5,
    }
    for i := 0; i < p.workers; i++ {
        go p.run()
    }
    return p
}
```

- 最多 5 个并发 LLM 请求，避免打爆上游 API
- 队列缓冲 100 个任务，超出后返回 429 Too Many Requests
- 每个 worker 串行处理：加载配置 → 构建 prompt → 调用 LLM → 发送回复

### 3. 配置缓存

```go
type AvatarConfigCache struct {
    configs sync.Map       // userId -> *AvatarConfig
    ttl     time.Duration  // 5 分钟
}

// 配置变更时主动失效
func (c *AvatarConfigCache) Invalidate(userId int64) {
    c.configs.Delete(userId)
}
```

- 分身配置缓存 5 分钟，避免每次回复查 DB
- 人设变更、设置更新时主动失效
- 知识上下文（会话历史）不缓存，保证实时性

### 4. 防刷屏间隔

继承群组 AI 的防刷屏机制，并加强：

- 同一用户分身对同一触发者，5 分钟内最多回复 1 次
- 同一用户分身全局，1 分钟内最多回复 3 次
- 用户接管后，冷却期内分身完全沉默

### 5. 降级策略

| 条件 | 降级行为 |
|------|---------|
| LLM 队列满（>100） | 返回 429，通知用户"分身暂时忙碌" |
| LLM 调用超时（>10s） | 放弃本次回复，通知用户"分身回复失败" |
| LLM 连续失败 3 次 | 暂停该用户分身 5 分钟，通知用户 |
| 系统负载高（CPU>80%） | 拒绝新的分身请求，仅处理已排队的 |

### 性能影响量化

| 场景 | 无防护 | 有防护后 |
|------|--------|---------|
| 10 人群 3 分身（群聊触发） | 3 并发 LLM + 30 WS | 3 串行 LLM + 3 私聊 WS |
| 100 人群 20 分身（群聊触发） | 20 并发 LLM + 2000 WS | 5 串行 LLM + 5 私聊 WS（其余排队） |
| 私聊 1v1 分身 | 1 LLM + 1 WS | 1 LLM + 1 WS（无变化） |

**群聊私聊回复模式 + Worker Pool + 全局限流后，最坏情况从 O(分身数×群人数) 降到 O(worker数×1)，性能完全可控。**

## 前端组件

新增：

```
src/components/avatar/
  AvatarSettingsPanel.vue          分身主设置面板
    AvatarBasicSettings.vue        开关、名称、模型选择
    AvatarPersonaSettings.vue      学习状态、预览、手动微调
    AvatarTriggerSettings.vue      触发规则
    AvatarKnowledgeSettings.vue    知识范围勾选
    AvatarReplySettings.vue        回复策略
  AvatarSessionToggle.vue          会话级分身开关（聊天头部）
  AvatarTakeoverBanner.vue         接管状态横幅
  AvatarReplyBadge.vue             "AI 代回复"消息标记
  AvatarGroupReplyNotice.vue       群聊中"你的分身已私聊回复"通知
```

修改：

```
ChatWindow.vue       添加 AvatarSessionToggle
MessageItem.vue      检测 is_avatar_reply，渲染 AvatarReplyBadge
Main.vue             handleNewMessage 处理分身消息 + 群聊分身通知
UserDetailPanel.vue  添加"分身设置"入口
```

## 关键交互

### 首次开启

1. 进入分身设置 → 检测历史消息数量
2. 消息充足 → 自动触发风格学习 → 显示进度
3. 学习完成 → 展示人设描述 → 用户预览/编辑 → 确认
4. 选择模型配置 → 设置触发规则 → 开启
5. 在具体会话中点击开关启用

### 私聊分身回复展示

- 消息右上角"AI 代回复"徽章（蓝色标签）
- 消息背景色略有区分（淡蓝色底）
- 发送者本人看到"你的分身已回复"提示

### 群聊分身回复展示

- 群聊中不显示分身消息
- 用户本人收到系统通知：「你的分身在群聊 XXX 中代你回复了李四」
- 点击通知跳转到与李四的私聊，查看分身回复内容
- 对方（李四）在私聊中收到分身回复，带"AI 代回复"标记

### 用户接管

- 用户发消息 → 分身暂停
- 横幅："分身已暂停，将在 N 分钟后恢复"
- 可点击"立即恢复"或"继续暂停"

## 与现有 AI 助手改进的关联

分身系统的实现应同步解决以下已有问题：

- P0: BotChatView 接通 AI 对话 API（分身的回复 API 可复用）
- P1: 统一 API 调用方式（分身 API 全部走 useRequest）
- P1: 类型定义统一到 types/（分身类型放在 types/avatar.ts）
- P1: 用共享组件替换 alert/confirm（分身设置面板使用项目统一弹窗）
