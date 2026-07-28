# 群聊助手群级记忆

## 目标

给群聊助手一个正经的、**群级（按 group_id 键）**的长期记忆，与分身记忆彻底隔离；同时切断当前"群助手蹭发送者分身记忆"导致的私聊->群可见回复的跨上下文泄露路径。

## 决策（已与用户确认）

- **存储内容**：原始择要消息--三层门控（便宜预筛 -> 向量去重 -> LLM 质量门）通过后存原文，与分身 `AvatarMemoryService` 范式一致。
- **前端**：后端 + GroupAIPanel 最小管理 UI（列表 / 单删 / 清空）。
- **存储后端**：复用 CortexDB（`VectorService.GetDB()`），`Scope=MemoryScopeUser, Namespace="group_assistant", UserID=<groupID>`。分身用 `Namespace="avatar"` + `UserID=<userID>`，两者靠 namespace 天然隔离，不混。

## 后端改动

### 1. 新建 `service/group_memory_service.go`（镜像 AvatarMemoryService，按 groupID 键）

- `Remember(groupID, conversationID, content)`：`SaveMemory` Scope=User / Namespace="group_assistant" / UserID=groupID，metadata: conversation_id、remembered_at。
- `Recall(groupID, query, topK) ([]SearchResult, error)`：同 scope/namespace 检索。
- `ShouldRemember(message) (bool, error)`：LLM 门控，**群特定 prompt**（值得记：群决定、约定、项目关键信息、群偏好；闲聊/简短回复不记）。
- `GetGroupMemories(groupID, limit) ([]MemoryRecord, error)`、`DeleteMemory(docID)`、`ForgetAll(groupID)`（按 namespace+userID 清空）、`GetMemoryCount(groupID)`。
- 依赖：`VectorService`（GetDB）+ `AIService`。构造函数 `NewGroupMemoryService(vectorSvc, aiService)`。

### 2. 写入路径 `handler/smart_reply_handler.go`

- 新增 `maybeRememberGroupMessage(groupID, conversationID, content)`，镜像 `maybeRememberSenderMessage` 三层门控（`looksMemorable` -> `Recall` 去重 score>0.85 -> `ShouldRemember`），通过则 `groupMemorySvc.Remember(groupID, ...)`。
- 在 `HandleMessage` 群分支调用（拿到 group + aiConfig 后，**仅 `aiConfig.Enabled` 时**）。异步 `SafeGo`，不阻塞主流程。
- 门控只看群 AI 是否启用，**不依赖发送者是否开分身**。

### 3. 读取路径 `service/smart_reply_graph.go`（切断分身借用）

- `SmartReplyGraph` 新增字段 `groupMemorySvc *GroupMemoryService`，**替换**图中现有 `memorySvc *AvatarMemoryService` 的角色（图的记忆来源改为群级）。
- `createMemoryNode`：改按 `input.Group.ID` 调 `groupMemorySvc.Recall`（`input.Group` 由 prepare 节点已加载），label 改为 `💡 群聊记忆：…`；`input.Group==nil` 时置空。
- `ExecuteStream` 内联记忆段（~215-226 行）同步改为群记忆召回。
- `NewSmartReplyGraph` 签名：`memorySvc *AvatarMemoryService` 参数替换为 `groupMemorySvc *GroupMemoryService`。
- 图里不再出现 `AvatarMemoryService.Recall`。

### 4. legacy 回退路径

- `generateAndSendReplyLegacy` 当前 `e.memorySvc.Recall(userID,…)`：改为先查 group（conversation->group），再 `groupMemorySvc.Recall(groupID,…)`；查不到群或 `groupMemorySvc==nil` 则置空。与图路径一致，不再读发送者分身记忆。

### 5. 引擎接线 `handler/smart_reply_handler.go`

- `SmartReplyEngine` 新增 `groupMemorySvc *GroupMemoryService` + `SetGroupMemoryService`。
- `InitSmartReplyGraph` 传 `e.groupMemorySvc`（替代原 memorySvc 入参）。
- `e.memorySvc`（AvatarMemoryService）**保留**，仅 `maybeRememberSenderMessage`（分身写）继续用。

### 6. DI / 路由

- `di/container.go`：Container 加字段 `GroupMemoryService`；`InitContainer` 里 `groupMemorySvc = service.NewGroupMemoryService(vectorSvc, aiService)`（与 AvatarMemoryService 并列，共用 vectorSvc）；赋值到 Container。
- `app/routes.go`：`if gms := di.GlobalContainer.GroupMemoryService; gms != nil { handler.SetGroupMemoryService(gms) }`。

### 7. 管理 API（新建 `handler/group_memory_handler.go` + routes）

群 AI 路由下新增，**:id 为 conversation_id（与现有 ai-settings 路由一致）**，handler 内解析 group；权限同 ai-settings（群主/管理员）：

- `GET    /groups/:id/group-memories`        -> 列表（最近 N 条）
- `DELETE /groups/:id/group-memories/:memory_id` -> 删单条
- `DELETE /groups/:id/group-memories`        -> 清空本群
- `POST   /groups/:id/group-memories/search` -> 搜索（给 UI 即时搜，可选）

## 前端改动

- `GroupAIPanel.vue` 新增 "群记忆" tab（或 section）：
  - 列表（调 GET）：显示记忆条目（内容 + 时间）+ 单条删除。
  - "清空全部"按钮（二次确认）。
  - 可选搜索框（调 search）。
- `types/ai.ts` 加 `GroupMemory` 类型。复用现有 `request` 封装。

## 测试

- `service/group_memory_service_test.go`：测可纯测部分（`ShouldRemember` 的 prompt 构造、`looksMemorable` 复用、scope/namespace/key 构造正确性）；Recall/Remember 依赖向量库，按 avatar 测试范式处理。
- `handler` 群记忆管理端点：权限 + 基本流程（参照 avatar memory 测试范式）。
- 现有 `DecideGroupAIReply` 触发测试不受影响（未动）。

## 不在本次范围

- 群删除时自动清记忆（后续可挂 group delete 钩子；本次提供清空 API 手动）。
- 记忆容量上限 / TTL（交给 CortexDB 自身管理）。
- 跨群记忆共享（群级隔离正是目的，不做）。
- LLM 总结成"群事实"（v2 增强，本次存原文）。

## 影响面

- **分身**：完全不动（avatar 写/读路径照旧）。
- **群助手**：读路径从"蹭发送者分身记忆"换成"群级记忆"；写路径新增群级写入。行为变化：群助手回复上下文来源改变；私聊->群泄露路径被切断。
- **触发逻辑**：`DecideGroupAIReply` / `HandleMessage` 触发分支不动。
- **性能**：每条群消息多一次异步三层门控（与分身 `maybeRememberSenderMessage` 同量级，已有先例）。

## 验证

- `go build ./...` / `go vet` / `go test ./service ./handler` 全绿。
- 手动：群助手回复后，群记忆列表出现条目；清空后列表空；私聊消息不进群记忆；分身记忆列表不受群记忆影响。
