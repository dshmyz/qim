# 消息搜索改进实现计划

日期：2026-10-13
依据 spec：`docs/superpowers/specs/2026-10-13-message-search-design.md`

## 现状关键锚点（实现前必读）

- `service/message_service.go:31` `useFulltextForMessageKeyword`：前人已发现 MySQL 默认全文解析器对中文不可靠，做了"ASCII 才走 FULLTEXT、中文一律 LIKE"的绕行补丁。本计划用 ngram 修复根因后，该函数退役，由单源函数取代。
- `service/message_service.go:1460` `SearchMessages`：跨会话搜索，假 total。
- `service/message_service.go:1399` `GetMessagesByFilter`：消息管理器的会话内搜索（带成员校验、真 total）。
- `service/admin_service.go:195` `AdminService.SearchMessages`：管理后台 LIKE 搜索。
- `service/message_service.go:1911` `SearchMessagesByFullText`：死代码（仅 `app/mysql_integration_test.go:164` 引用）。
- `app/init.go:1364-1375`：messages FULLTEXT 索引迁移 + SQLite FTS5 虚拟表建表回填（烂尾）。
- `database/dialect.go`：`SupportsFulltext()` 探测（建临时表加 FULLTEXT 索引）。
- `handler/message_handler.go:1366` `SearchMessages` handler：`"total": len(responseMessages)` 假分页。
- 前端：
  - `qim-client/src/views/Main.vue:2440` `handleSearch`：Sidebar 搜索现走 `/api/v1/users/search`（用户/群），另有本地会话列表过滤（:2420 `filteredConversations`）；防抖 300ms 已存在（:2501）。
  - `qim-client/src/components/layout/Sidebar.vue`：搜索框 + 搜索结果浮层（`SearchResultItem` 目前只有用户/群形态）。
  - `qim-client/src/components/chat/ChatWindow.vue:2010` 起：跳转定位流程（拉前后各 20 条 → mergeMessages → `scrollAndHighlightMessage`），现仅由消息管理器点击触发。
  - `qim-client/src/stores/chat.ts`：会话/消息 store，`pendingJump` 的落点。
  - 死代码：`components/ai/AISearchInput.vue`、`components/ai/AISearchResults.vue`、`components/chat/MessageSearch.vue`（三者均无挂载点）；`composables/useAIActions.ts:220` `searchMessages`（仅死组件用）。

## 阶段 1：后端 — 匹配条件单源

### 任务 1.1 新增 `database.ContentMatch`

新文件 `qim-server/database/content_match.go`：

```go
// ContentMatch 返回"消息内容含关键词"的 WHERE 片段与参数（匹配单源）。
// ngram 全文可用且关键词 ≥2 rune → MATCH...AGAINST(BOOLEAN MODE)；
// 否则（单字、中文但库无 ngram、SQLite/TiDB）→ LIKE 降级。
func ContentMatch(keyword string) (string, []any)
```

- 条件：`D.SupportsFulltext()` 为真（阶段 1.2 后其语义即"ngram 全文可用"）且 `len([]rune(keyword)) >= 2` → `"MATCH(messages.content) AGAINST(? IN BOOLEAN MODE)", [keyword]`；否则 `"messages.content LIKE ?", ["%"+keyword+"%"]`。
- 空关键词由调用方自行跳过（三处调用方现状都是 `if keyword != ""`）。
- 删除 `service/message_service.go:33` `useFulltextForMessageKeyword`。

### 任务 1.2 三处调用方接入单源

- `MessageService.SearchMessages`（:1475-1483）：删除现有 MATCH/LIKE 分支，改用 `database.ContentMatch`。
- `MessageService.GetMessagesByFilter`（:1423-1431）：同上（原 `MATCH(content)` 无前缀，统一为 `messages.content` 前缀——该查询无 join，前缀安全）。
- `AdminService.SearchMessages`（admin_service.go:203-205）：同上。

### 验收

- `go build ./...` 通过；`go test ./service/... ./database/...` 通过。
- grep 确认 `useFulltextForMessageKeyword` 无残留、三处调用方不再手写 MATCH/LIKE。

## 阶段 2：后端 — 索引迁移与探测

### 任务 2.1 ngram 探测（`database/dialect.go`）

- `SupportsFulltext()` 的探测建表改用 `WITH PARSER ngram` 建探测索引：成功 → 支持（此后 ContentMatch 走 MATCH）；失败 → 降级 LIKE，与现状行为兼容。

### 任务 2.2 索引迁移（`app/init.go:1364` 附近）

- MySQL 分支：先 `SHOW CREATE TABLE messages` 检查现有 FULLTEXT 索引定义是否含 `WITH PARSER ngram`；不含 → drop 旧索引，`ALTER TABLE messages ADD FULLTEXT INDEX idx_messages_content (content) WITH PARSER ngram` 重建。
- SQLite 分支：删除 FTS5 建表回填，改为 `DROP TABLE IF EXISTS messages_fts5`（启动一次即可，下版本移除该 DROP）。

### 任务 2.3 删除死代码

- 删 `service/message_service.go:1911` `SearchMessagesByFullText`。
- `app/mysql_integration_test.go:162` `TestMySQL_SearchMessagesByFullText` 改造为 `TestMySQL_SearchMessages`：用 `SearchMessages` 覆盖 ngram MATCH 分支，**必须加中文关键词用例**（如建含"项目排期讨论"的消息、搜"排期"能命中）。

### 验收

- 集成测试（本地 MySQL 可用时）通过：中文关键词命中；单字关键词 LIKE 兜底命中。
- SQLite dev 环境：`go run main.go` 启动无 FTS5 建表日志，`messages_fts5` 表消失。

## 阶段 3：后端 — `/messages/search` 真分页

### 任务 3.1 service 签名

- `MessageService.SearchMessages` 返回改为 `([]model.Message, int64, error)`；在现有查询上先 `Count` 再分页 `Find`（两步同条件，参照 `GetMessagesByFilter` :1440-1444 写法）。

### 任务 3.2 handler

- `handler/message_handler.go:1403` 接新签名；`"total"` 用真 total。

### 任务 3.3 测试

- `handler` 层：`SearchMessages` 真分页测试（两条数据 pageSize=1，total=2、第二页 1 条），沿用现有 `handler_test.go` 模式。
- 单测（SQLite）顺带覆盖 ContentMatch LIKE 分支与单字降级。

### 验收

- `go test ./handler/... ./service/...` 通过。

## 阶段 4：前端 — 死代码清理

- 删 `components/ai/AISearchInput.vue`、`components/ai/AISearchResults.vue`（删前 grep 确认仍无引用）。
- 删 `components/chat/MessageSearch.vue`（同上；它与 AISearch 系列同属未挂载的搜索残留）。
- 删 `useAIActions.ts` 的 `searchMessages`（确认仅 AISearchInput 使用后）。
- `api/message.ts:116` `searchMessages` **保留**（阶段 5 复用）。
- `/api/v1/ai/search` 后端**不动**。

### 验收

- `npm run build` 通过；grep 无残留 import；`npm run test` 现有用例不红。

## 阶段 5：前端 — 微信式全局搜索

### 任务 5.1 消息命中结果（Sidebar + Main.vue）

- `api/message.ts` `searchMessages` 按新响应核对（total 为数字），需要则微调返回类型。
- `Main.vue`：
  - 现有防抖 watch（:2501）内，除 `handleSearch`（users/search）外并行调 `messageApi.searchMessages({ keyword, page: 1, pageSize: 20 })`，存 `messageHits = ref([])`；失败静默降级为空（不阻塞用户/群结果）。
  - 传 `messageHits` 进 Sidebar。
- `Sidebar.vue`：搜索结果浮层新增"聊天记录"区块（位于用户/群结果之后）：按 `conversation_id` 分组，组头 = 会话名 + 命中数（会话名从结果的 conversation 数据取，单聊对端名按 `other_member` 兜底——见单聊 members 常为空的已知坑）；条目 = 发送人 + 时间 + 高亮片段。
  - 高亮纯前端 `<mark>`（大小写不敏感替换关键词）。
  - 非文本类型命中（image/file 等）显示类型标签，不渲染 content。
  - 无命中不显示该区块。
- 组件拆分：新建 `components/chat/MessageHitItem.vue`（单条命中渲染）+ 分组容器逻辑放 Sidebar 或独立 `MessageHitsPanel.vue`，避免 Sidebar 膨胀。

### 任务 5.2 跳转定位（chat store pendingJump）

- `stores/chat.ts` 新增：
  ```ts
  const pendingJump = ref<{ conversationId: number, messageId: number } | null>(null)
  function setPendingJump(conversationId: number, messageId: number)
  function consumePendingJump(conversationId: number) // 匹配本会话则返回 messageId 并清空，否则 null
  ```
- `ChatWindow.vue`：把 :2010 起的跳转流程抽成 `jumpToMessage(conversationId, messageId)`（消息管理器点击改为调用它）；新增 watch——`chatStore.consumePendingJump(props.conversation.id)` 有值时触发 `jumpToMessage`。会话切换竞态沿用现有 `conversationIdAtStart` 检查。
- `Main.vue`：消息命中点击 → `chatStore.setPendingJump(convId, msgId)` → 走现有会话切换路径（openChat/handleGroupChatSelect）→ 清搜索。

### 任务 5.3 交互细节

- `Ctrl/Cmd+K` 聚焦 Sidebar 搜索框（Main.vue 挂全局 keydown，注意与现有快捷键去重）。
- 搜索中 spinner；空态文案"无相关聊天记录"。

### 验收（前端测试）

- vitest 新增：
  - MessageHitItem：文本命中渲染+高亮、非文本命中显示类型标签。
  - chat store：pendingJump 置位/按会话消费/不匹配会话返回 null 且保留。
  - Main.vue 搜索防抖后 messageHits 被填充（mock api）。
- `npm run test` 通过。

## 阶段 6：整体验证

1. `cd qim-server && go build ./... && go test ./...`
2. `cd qim-client && npm run test && npm run build`
3. dev 环境手工验证清单（SQLite）：
   - 侧栏搜索中文关键词 → 命中分区块、点击跳转定位高亮；
   - 单字关键词可搜；
   - 消息管理器中文搜索（走 filter 接口）恢复命中；
   - 清空关键词回到会话列表。
4. （可选，有 MySQL 时）启动迁移日志确认 ngram 索引重建、集成测试通过。

## 依赖与顺序

阶段 1 → 2 → 3 顺序执行（同仓后端，改动相互咬合）；阶段 4 独立可先行；阶段 5 依赖阶段 3 的真分页 API；阶段 6 收尾。
