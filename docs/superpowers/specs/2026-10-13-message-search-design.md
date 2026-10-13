# 消息搜索改进设计（中文全文 + 全局搜索 + 链路收敛）

日期：2026-10-13
状态：已确认（brainstorming 产出）

## 1. 背景与现状问题

当前消息搜索存在四条并存且分叉的路径：

1. **REST `/api/v1/messages/search`**（`handler.SearchMessages` → `service.MessageService.SearchMessages`）：跨会话关键词搜索，MySQL 走 FULLTEXT `MATCH...AGAINST`，SQLite 降级 LIKE。客户端 UI 无任何调用方，仅 qim-mcp 在用。
2. **消息管理器**（`MessageManager.vue`）：会话内搜索，走 `getMessagesByFilter` 的 `search` 参数，是用户实际使用的入口。
3. **AI 语义搜索 `/api/v1/ai/search`**：统一检索图（gracedb 向量 + LLM 重排），但前端组件 `AISearchInput.vue` 未被任何页面挂载，属于死代码。
4. **管理后台搜索**：纯 LIKE。

核心问题：

- **中文分词失效（真根因）**：MySQL FULLTEXT 索引未指定 `WITH PARSER ngram`，默认解析器把整段中文当一个词，中文关键词基本搜不到；SQLite FTS5 用 `unicode61` 分词器，同样不切中文。产品是中文 IM，此问题直接影响主路径可用性。
- **FTS5 索引烂尾**：`messages_fts5` 虚拟表只在启动时回填一次，之后新消息永远不进索引；`SearchMessagesByFullText` 无任何生产调用方（仅集成测试引用）。
- **假分页**：`/messages/search` 返回的 `total` 是 `len(list)`，跨会话翻页不成立。
- **缺全局搜索入口**：跨会话搜索有后端、无前端。

## 2. 已确认的产品决策

- **关键词为主**：用户搜索框就是关键词搜索（快、准、可跳转定位原消息）。语义检索只服务 AI 助手/群助手的知识召回，不进搜索 UI。后端 `/api/v1/ai/search` 保留（AI 域另有消费者）。
- **全局搜索 + 定位**：新增跨会话搜索入口，点击结果跳转原会话并滚动定位到原消息。
- **方案选型**：修好 SQL 原生全文检索（MySQL ngram），不引入外置搜索引擎、不引入内嵌全文引擎依赖。

## 3. 后端设计

### 3.1 索引层（数据库迁移，`app/init.go` 索引迁移处）

- MySQL：messages 的 FULLTEXT 索引重建为 `WITH PARSER ngram`。检测方式：`SHOW CREATE TABLE messages` 判断现有 FULLTEXT 索引定义是否已含 `WITH PARSER ngram`；不含则 drop 旧索引、重建。一次性迁移，大表重建成本可接受。
- 探测逻辑升级（`database/dialect.go`）：`SupportsFulltext()` 的探测建表改用 `WITH PARSER ngram` 建探测索引——探测成功才认为中文全文可用，失败自动降级 LIKE（兼容未带 ngram 的 MySQL 变体）。
- SQLite：启动迁移中 `DROP TABLE IF EXISTS messages_fts5`，删除烂尾虚拟表。

### 3.2 匹配条件单源

新增共享构造函数"关键词 → WHERE 条件"，逻辑：

- 关键词 ≥ 2 字符 且 库支持 ngram 全文 → `MATCH(content) AGAINST(? IN BOOLEAN MODE)`
- 其他情况（单字查询、库不支持全文）→ `LIKE '%kw%'`

单字降级对调用方透明（ngram_token_size=2 导致单字 MATCH 无结果，必须兜底）。

三个调用方统一接入：

- `MessageService.SearchMessages`（跨会话搜索）
- `GetMessagesByFilter` 的 `search` 参数（消息管理器）
- `AdminService.SearchMessages`（管理后台）

从此匹配逻辑只改一处。

### 3.3 API 修复

`/api/v1/messages/search`：`total` 改为真正的 COUNT 查询，跨会话翻页成立。响应结构其余不变（继续走 `BuildMessageResponse`）。

### 3.4 死代码清理（范围内顺手做）

- 删 `service.MessageService.SearchMessagesByFullText`（`message_service.go:1911`），集成测试 `TestMySQL_SearchMessagesByFullText` 改造为覆盖 `SearchMessages` 的 ngram MATCH 分支。
- 删 `app/init.go` 的 FTS5 建表回填逻辑。
- 删未挂载的 `AISearchInput.vue`；`AISearchResults.vue` 若仅被其引用则一并删。
- 客户端 `api/message.ts` 的 `searchMessages` 保留，全局搜索直接复用。
- `useAIActions.searchMessages` 若仅服务死组件则一并删除。

### 3.5 权限

沿用现有 `conversation_members` join 过滤（只能搜到自己所在会话的消息），不新做权限层。

## 4. 前端设计（微信式交互）

### 4.1 入口

左侧会话列表顶部嵌入常驻搜索框（`Ctrl/Cmd+K` 快捷聚焦）。

### 4.2 结果视图（左侧面板内切换，非弹窗）

```
┌─────────────────────┬──────────────────────────┐
│ [🔍 搜索…]          │                          │
├─────────────────────┤      聊天窗口               │
│ （输入关键词后：）     │  （点击结果后跳转定位        │
│ 项目群          12   │    该消息并高亮）           │
│  └ 张三: 我们讨论…██ │                          │
│  └ 李四: 明天开会██  │                          │
│ 王五（单聊）      3   │                          │
│  └ 王五: 方案里提到… │                          │
└─────────────────────┴──────────────────────────┘
```

- 输入防抖 300ms，调 `/messages/search`（不传 conv_id）。
- 结果按会话分组：组头 = 会话名 + 命中数；条目 = 发送人 + 时间 + 高亮片段。
- 高亮纯前端 `<mark>`；图片/文件等非文本命中显示类型标签而非内容。
- 分页两级：首屏一次全局请求（默认 pageSize）铺出各会话分组；组内"查看更多"用 API 既有的 `conv_id` 参数按该会话翻页加载。
- 状态：搜索中 spinner；无结果空态；清空关键词回到会话列表；Esc 收起；重开保留上次关键词。

### 4.3 跳转定位

- chat store 新增一次性 `pendingJump = { conversationId, messageId }` 状态。
- 点击结果 → chatStore 切换会话 → ChatWindow watch 到本会话有待跳转目标 → 复用**现有**流程（拉取目标消息前后各 20 条 → `mergeMessages` → `scrollAndHighlightMessage`，`ChatWindow.vue:2010` 起）→ 完成后清除 pendingJump。
- 不新造跳转逻辑，仅扩展现有流程的触发源。

### 4.4 消息管理器不动

会话内搜索保持现有交互，后端匹配条件换成单源函数后自动获得中文全文能力，前端零改动。

## 5. 错误处理

- 单字关键词：单源函数内自动降级 LIKE，前端无感知。
- ngram 不可用（探测失败）：启动迁移 warning + LIKE 降级，搜索功能完整可用。
- 搜索请求失败：左侧面板显示错误提示，不破坏会话列表状态。
- 目标消息已被删除：定位流程已有"未找到目标消息"警告分支，沿用。

## 6. 测试

- 后端单测：单源匹配函数（LIKE 分支、单字降级分支）；`SearchMessages` handler 真分页（现有 `handler_test.go` 模式）。
- 后端集成：`mysql_integration_test.go` 改造为覆盖 `SearchMessages` 的 ngram MATCH 分支（含中文关键词用例）。
- 前端单测（vitest）：左侧搜索面板组件——防抖触发、分组渲染、点击 emit 正确跳转目标、状态切换（列表⇄结果）；chat store 的 pendingJump 置位/消费。

## 7. 不做的事（YAGNI）

- 不引入外置搜索引擎（Meilisearch 等）或内嵌全文引擎（bleve 等）。
- 语义/向量搜索不进用户搜索产品面（保留在 AI 助手域）。
- 不做统一搜索面板（联系人/群/文件混合搜索）——后续如需要另立 spec。
- 管理后台 UI 不改（仅后端匹配条件接入单源）。
