# QIM AI 能力深化设计方案

## 元信息

- 日期: 2026-05-14
- 状态: 待审查
- 基于: 现有 Eino v0.8.13 基础设施

---

## 一、背景与问题诊断

### 1.1 项目 AI 功能现状

QIM 项目中 AI 功能分为三个层次：

| 类别 | 功能 | 实现方式 | 深度评分 |
|---|---|---|---|
| 真实 AI + 有业务价值 | AI 对话、会话摘要 | LLM API 调用，有结构化 prompt | ★★★★☆ |
| 真实 AI + 浅层包装 | 翻译、改写、润色、搜索重排 | System prompt → LLM | ★★★☆☆ |
| **根本不用 AI（冒充）** | 运维工具（5个）、MCP 工具（4个） | `strings.Contains()` 硬编码 | ☆☆☆☆☆ |

### 1.2 核心问题

1. **40% 的 AI 功能是假的** — 运维工具和 MCP 工具完全不用 AI，用硬编码 if-else + 假数据
2. **已用 AI 的功能大多是浅层 prompt wrapper** — 没有 RAG、缓存、质量保障
3. **AI 是显性存在，而非无感渗透** — 用户必须主动进入 AI 面板才能使用，不像 Slack/Notion 那样渗透在日常体验中

### 1.3 现有 Eino 基础设施（优势）

项目已集成 Eino v0.8.13，具备完善的 AI 编排基础：

| 已有组件 | 用途 |
|---|---|
| `EinoChatModel` | AIService → Eino ChatModel 适配器，支持流式 |
| `SmartReplyGraph` | 智能回复 Graph：prepare → knowledge/memory/history(并行) → merge → model → format |
| `AvatarReplyGraph` | 数字分身 Graph：prepare → ChatTemplate → model → format |
| `MemoryRetriever` | Eino Retriever，封装 AvatarMemoryService |
| `GroupDocRetriever` | Eino Retriever，封装群文档搜索 |
| `NoteRetriever` | Eino Retriever，封装笔记向量搜索 |
| 7 个 Provider | OpenAI、Anthropic、阿里、腾讯、字节、百度、DeepSeek |

---

## 二、设计目标

1. **修复假 AI** — 运维工具和 MCP 工具接入真正的 LLM
2. **基于 Eino Graph 深化** — 用 Eino Graph 编排所有 AI 场景，不自己造 Pipeline
3. **新增高价值场景** — 智能消息速览、Unified Search（多源统一搜索）
4. **AI 无感渗透** — 让 AI 融入日常 IM 体验，而非独立的"AI 面板"

---

## 三、架构设计

### 3.1 整体架构

```
                         ┌──────────────────────────────────────┐
                         │       Eino compose.Graph 层           │
                         │                                      │
                         │  SmartReplyGraph (已有)               │
                         │  AvatarReplyGraph (已有)              │
                         │  OpsGraph          (新增)            │
                         │  SmartDigestGraph  (新增)            │
                         │  UnifiedSearchGraph(新增)            │
                         │                                      │
                         │  ┌─────────┐ ┌──────────┐ ┌───────┐ │
                         │  │Retriever│ │ChatModel │ │Lambda │ │
                         │  │(4个)    │ │(EinoChat)│ │(节点) │ │
                         │  └─────────┘ └──────────┘ └───────┘ │
                         └──────────────────────────────────────┘
                                        │
                                        ▼
                         ┌──────────────────────────────────────┐
                         │           ai.AIService               │
                         │  ┌──────────┐ ┌───────┐ ┌─────────┐ │
                         │  │ Provider │ │MCPSvr │ │ Embed   │ │
                         │  │ Factory  │ │(改造) │ │         │ │
                         │  └──────────┘ └───────┘ └─────────┘ │
                         └──────────────────────────────────────┘
```

### 3.2 核心原则

- **所有 AI 流程走 Eino Graph** — Handler 不直接调 AIService，而是调 Graph
- **复用不重复** — 4 个 Retriever（3 个已有 + 1 个新增） + 2 个已有 Graph + EinoChatModel 全部复用
- **一切走 LLM** — 运维工具不再硬编码
- **Native Function Calling** — 用 Provider 原生 function calling API 替代 prompt engineering

---

## 四、核心改动

### 4.1 ai_service.go 改造 — Native Function Calling

**改造前：** 把工具列表拼成 system prompt 文本，让 AI 手动返回 JSON 格式的工具调用。

**改造后：**

Provider 接口新增：

```go
type ToolDef struct {
    Name        string
    Description string
    Parameters  map[string]interface{}
}

type ChatResponse struct {
    Content   string
    ToolCalls []ToolCall
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]interface{}
}

// Provider 接口新增方法
type Provider interface {
    // ... 已有方法
    ChatWithTools(messages []Message, tools []ToolDef) (*ChatResponse, error) // 新增
}
```

`GetCompletionWithTools` 改为：

```go
func (s *AIService) GetCompletionWithTools(messages []Message, tools []ToolDef, callerCtx *CallerContext) (string, error) {
    resp, err := s.provider.ChatWithTools(messages, tools)
    if err != nil { return "", err }
    if len(resp.ToolCalls) == 0 { return resp.Content, nil }

    for _, tc := range resp.ToolCalls {
        result, _ := s.executeToolCall(tc, callerCtx)
        messages = append(messages,
            Message{Role: "assistant", Content: "", ToolCalls: []ToolCall{tc}},
            Message{Role: "tool", Content: result, ToolCallID: tc.ID},
        )
    }

    finalResp, _ := s.provider.ChatWithTools(messages, tools)
    return finalResp.Content, nil
}
```

各 Provider（OpenAI、Anthropic 等）实现 `ChatWithTools`。

### 4.2 ops_tools.go 改造 — 5 个运维工具接入 LLM

**改造原则：** `OpsTool` 接口不变，Handler 层不动，只改 `Execute()` 内部。

**改造前：**
```go
func (t *IntelligentTroubleshootingTool) Execute(params) {
    symptom := params["symptom"].(string)
    if strings.Contains(symptom, "CPU") { return "检查进程" }
    // ... 硬编码 if-else
}
```

**改造后：**
```go
type IntelligentTroubleshootingTool struct {
    aiService *AIService
}

func (t *IntelligentTroubleshootingTool) Execute(params) (interface{}, error) {
    systemPrompt := `你是一个资深运维工程师。请分析故障症状并给出解决方案。
严格按 JSON 格式输出：{ "analysis": "...", "possible_causes": [...], "solutions": [...], "commands": [...], "urgency": "..." }`

    messages := []Message{
        {Role: "system", Content: systemPrompt},
        {Role: "user", Content: fmt.Sprintf("症状：%s\n日志：%s", params["symptom"], params["logs"])},
    }

    result, err := t.aiService.GetCompletion(messages)
    // 解析 JSON 返回
}
```

5 个工具全部统一用此模式。

### 4.3 mcp.go 改造 — 4 个 MCP 工具从假数据到真实/LLM

**保留：** 工具注册/启禁用/执行框架、`MCPServer`、`MCPTool` 接口
**改造：** `ServerMonitorTool`、`LogAnalyzerTool`、`ProcessManagerTool`、`NetworkTools` 的 `Execute` 方法从返回硬编码假数据改为调 LLM（或标记为示例工具，按需真实实现）。

### 4.4 OpsGraph（新增）— Eino Graph 编排运维场景

```go
// service/ops_graph.go

type OpsInput struct {
    Intent string // troubleshoot|command|log_analysis|alert|knowledge
    Params map[string]interface{}
    UserID uint
}

type OpsGraph struct {
    graphs    map[string]compose.Runnable[OpsInput, map[string]interface{}]
    aiService *ai.AIService
}

func (g *OpsGraph) buildTroubleshootGraph() compose.Runnable[...] {
    graph := compose.NewGraph[...]()
    graph.AddRetrieverNode("knowledge", opsKnowledgeRetriever)
    graph.AddChatModelNode("analyzer", NewEinoChatModel(g.aiService, 0))
    graph.AddLambdaNode("format", formatOpsResult)
    // START → knowledge → analyzer → format → END
    return graph.Compile(ctx)
}
```

5 个 Graph 共享底层 Retriever 和 ChatModel，只 prompt 和输出格式不同。

### 4.5 SmartDigestGraph（新增）— 智能消息速览

**触发时机：** 用户登录/切换群聊时自动请求，WebSocket 推送新消息时增量更新。

**数据流：**

```
START → prepare(加载未读消息+用户画像)
         ↓
    ┌────┼────┐
    ↓    ↓    ↓
classify extract  rank
(意图分类)(@我)(相关性)
    ↓    ↓    ↓
    └────┼────┘
         ↓
     merge(汇聚)
         ↓
   summarize(LLM生成摘要)
         ↓
    format(结构化输出)
         ↓
       END
```

**输出格式：**

```json
{
  "generated_at": "2026-05-14 14:30",
  "unread_count": 127,
  "categories": [
    {
      "type": "mention",
      "priority": "high",
      "items": [{
        "summary": "张三问你用户模块的数据库方案确定了吗",
        "sender": "张三",
        "message_id": 12345,
        "group_name": "技术讨论群",
        "suggested_action": "回复"
      }]
    },
    {
      "type": "related",
      "priority": "medium",
      "items": [{
        "summary": "李四和王五在讨论 API 鉴权方案，涉及你上周提交的 JWT 实现",
        "key_speakers": ["李四", "王五"],
        "group_name": "后端开发群",
        "message_count": 23
      }]
    },
    {
      "type": "hot_topic",
      "priority": "low",
      "items": [{
        "topic": "Q2 技术规划讨论",
        "group_name": "全员群",
        "message_count": 50,
        "one_line": "主要讨论了下季度的技术栈升级和人力分配"
      }]
    },
    {
      "type": "urgent",
      "priority": "critical",
      "items": [{
        "summary": "生产环境 CPU 告警，运维已处理",
        "group_name": "运维告警群",
        "resolved": true
      }]
    }
  ]
}
```

**缓存策略：** 按 `user_id + conversation_id` 缓存。新消息到达时通过 WebSocket 通知或消息入库时清理缓存键。

### 4.6 UnifiedSearchGraph（新增）— 多源统一搜索

**数据流：**

```
START → query_rewrite(LLM改写查询)
         ↓
    ┌────┼────┬────┐
    ↓    ↓    ↓    ↓
messages notes docs memory
(消息)  (笔记)(文档)(记忆)
 全部复用已有 Retriever！
    ↓    ↓    ↓    ↓
    └────┼────┴────┘
         ↓
    rerank(去重+LLM排序)
         ↓
 synthesize(LLM综合回答+引用)
         ↓
    format(结构化输出)
         ↓
       END
```

**Retriever 复用：**

| 节点 | Retriever | 状态 |
|---|---|---|
| messages | MessageRetriever | **新增**，封装消息搜索 |
| notes | NoteRetriever | **复用已有** |
| group_docs | GroupDocRetriever | **复用已有** |
| memory | MemoryRetriever | **复用已有** |

**输出格式：**

```json
{
  "query": "用户模块的鉴权方案是什么",
  "answer": "根据已检索到的信息，用户模块的鉴权方案采用 JWT + OAuth2...",
  "sources": [
    {
      "type": "group_doc",
      "title": "用户模块技术方案 v2.3",
      "relevance": 0.95,
      "excerpt": "...JWT token 有效期 2 小时...",
      "added_by": "张三",
      "date": "2026-04-20"
    },
    {
      "type": "message",
      "sender": "李四",
      "content": "鉴权那块我们最终定了用 JWT + OAuth2",
      "group": "技术讨论群",
      "date": "2026-05-10",
      "relevance": 0.82
    },
    {
      "type": "note",
      "title": "API 鉴权备忘",
      "excerpt": "JWT secret 配置在环境变量 JWT_SECRET",
      "relevance": 0.71
    }
  ]
}
```

### 4.7 AI 缓存（新增）— 翻译/摘要/速览缓存

```go
// service/ai_cache.go

type AICache struct {
    store sync.Map // 可替换为 Redis
}

func (c *AICache) GetOrCompute(key string, compute func() (string, error), ttl time.Duration) (string, error)
```

**缓存策略：**

| 场景 | 缓存键 | TTL |
|---|---|---|
| 翻译 | `sha256(text + source + target)` | 永久 |
| 会话摘要 | `sha256(conversation_id + time_range)` | 1 小时 |
| 改写/润色 | `sha256(text + style + tone)` | 24 小时 |
| 消息速览 | `sha256(user_id + conversation_id)` | 新消息到达时失效 |
| 对话/搜索 | 不缓存 | — |

---

### 4.8 老代码迁移策略

**原则：不是全部迁移，而是有价值的迁移。**

| 老代码 | 当前调用方式 | 迁移决策 | 理由 |
|---|---|---|---|
| 通用对话 `GetCompletion` | Handler → AIService | ❌ 不迁移 | 太底层太通用，Graph 是它的上层消费者而非替代者 |
| 流式对话 `GetCompletionStream` | Handler → AIService | ❌ 不迁移 | 同上 |
| 会话摘要 | Handler 手动拼 prompt → AIService | ✅ 迁移到 SummaryGraph | 加缓存、格式校验、上下文注入 |
| 翻译/改写/润色 | Handler system prompt → AIService | ✅ 迁移到 TextProcessGraph | 核心价值是缓存，避免重复调 LLM |
| 语义搜索 | Handler DB 查询 → AIService 排序 | ✅ 替换为 UnifiedSearchGraph | 多源检索 + RAG |
| 运维 5 个路由 | Handler → new Tool → Execute | ✅ 替换为 OpsGraph | 接入真正 LLM |

#### 4.8.1 SummaryGraph（新增）

```
prepare(加载消息+用户上下文)
       ↓
build_messages(拼 system prompt)
       ↓
model(LLM 生成摘要)
       ↓
validate(格式校验 — 检查四段结构)
       ↓
cache(写入缓存)
       ↓
END
```

比现有直接调 AIService 多了：
- 格式校验节点（检查摘要是否包含核心话题/重要决策/待办事项/关键发言）
- 缓存节点（同一会话+时间段不重复生成）

#### 4.8.2 TextProcessGraph（新增）

翻译/改写/润色共用一个 Graph，通过 intent 参数区分：

```
cache_check(查缓存)
       ↓
  ┌────┴────┐
  ↓         ↓
命中      未命中
  ↓         ↓
直接返回   model(LLM)
             ↓
        cache_write(写缓存)
             ↓
           END
```

核心价值是缓存——相同的文本+语言，永久不再调 LLM。

#### 4.8.3 为什么老代码不删

| 层级 | 文件 | 为什么不删 |
|---|---|---|
| **底层引擎** | `ai/ai_service.go` | Eino Graph 的 ChatModel 节点（EinoChatModel）内部调 AIService。删了 Graph 就没法工作。AIService 是引擎，Graph 是编排层。 |
| **HTTP 层** | `handler/*.go` | Handler 承担路由注册、参数校验、响应格式化职责，这些保留。只把"手动拼 prompt → 调 AIService"换成"调 Graph"。 |
| **工具接口** | `ai/ops_tools.go` | `OpsTool` 接口被 MCPServer 依赖，接口不变，只改 Execute 内部实现。 |
| **Provider 接口** | `ai/provider*.go` | 接口扩展（新增 ChatWithTools），已有方法（Chat、ChatStream、Embedding）不动。 |

**真正"删除"的是 Handler 内部的手动拼 prompt 逻辑，但这算重构，不是删文件。**

---

## 五、文件改动清单

### 改造文件（3 个）

| 文件 | 改动内容 |
|---|---|
| `ai/ai_service.go` | 新增 `ChatWithTools`，改造 `GetCompletionWithTools` 用 native function calling |
| `ai/provider.go` | Provider 接口新增 `ChatWithTools`、`ToolDef`、`ChatResponse`、`ToolCall` 类型 |
| `ai/ops_tools.go` | 5 个工具的 `Execute()` 从硬编码改为调 AIService |

### 新增文件（7 个）

| 文件 | 内容 |
|---|---|
| `service/ops_graph.go` | Eino Graph 编排 5 个运维场景 |
| `service/summary_graph.go` | 会话摘要 Graph（迁移自 handler，加缓存+校验） |
| `service/text_process_graph.go` | 翻译/改写/润色 Graph（核心是缓存） |
| `service/smart_digest_graph.go` | 智能消息速览 Eino Graph |
| `service/unified_search_graph.go` | 多源统一搜索 Eino Graph |
| `service/message_retriever.go` | 消息检索 Retriever（Eino 接口） |
| `service/ai_cache.go` | 翻译/摘要/速览结果缓存 |

### 同时改造（Provider 层，约 7 个文件）

| 文件 | 改动内容 |
|---|---|
| `ai/provider_openai.go` | 实现 `ChatWithTools` |
| `ai/provider_anthropic.go` | 实现 `ChatWithTools` |
| `ai/provider_alibaba.go` 等 | 实现 `ChatWithTools`（或降级为 prompt engineering） |

### 不改动文件（约 15 个）

7 个 Provider 的其他方法、3 个 Retriever、2 个已有 Graph、EinoChatModel、所有类型/配置/Handler 文件。

### Handler 层微调

Handler 从直接调 AIService 改为调对应的 Graph（保留路由注册、参数校验、响应格式化）：

| Handler 文件 | 改动 |
|---|---|
| `ai_handler.go` 运维路由 | → 调 OpsGraph |
| `ai_handler.go` 通用对话 | ❌ 不动（太底层） |
| `ai_summary_handler.go` | → 调 SummaryGraph |
| `ai_text_handler.go` | → 调 TextProcessGraph |
| `ai_search_handler.go` | → 调 UnifiedSearchGraph |
| 新增 Digest 路由 | → 调 SmartDigestGraph |

---

## 六、实施顺序

| 阶段 | 内容 | 依赖 |
|---|---|---|
| **Phase 1**: 基础改造 | ai_service.go + Provider 接口新增 ChatWithTools | 无 |
| **Phase 2**: 修复假 AI | ops_tools.go 改造（5个工具） + mcp.go 改造（4个工具） | Phase 1 |
| **Phase 3**: 运维 Graph | service/ops_graph.go | Phase 1, 2 |
| **Phase 4**: 缓存 | service/ai_cache.go | 无 |
| **Phase 5**: 迁移老功能 | SummaryGraph + TextProcessGraph | Phase 1, 4 |
| **Phase 6**: 新增场景 | SmartDigestGraph + UnifiedSearchGraph + MessageRetriever | Phase 1, 4, 复用已有 Retriever |
| **Phase 7**: Handler 适配 | Handler 调 Graph 而非直接调 AIService | 各 Phase 完成后 |