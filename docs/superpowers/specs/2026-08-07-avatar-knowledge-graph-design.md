# 分身「知识图谱」设计

## 背景与目标

群助手的「知识图谱」页已上线（展示群知识文档片段簇）。用户在问：**分身的设置界面是否也要加知识图谱？**

查了分身的数据落地后结论：**要加，但不是照搬群的。**

- 群的"知识"是单一文档库，图谱 = 把片段铺开画（片段簇，无边）。
- 分身的"知识"是**异构的多种来源**：分身记忆、个人文档、笔记、任务。
- 分身记忆反射（`reflectConsolidated`）在每条记忆 metadata 里已提取 `knowledge_memory_entities` / `knowledge_memory_themes`（`[]string`）——这是**真·知识图谱的原材料**（实体/主题节点 + 记忆共现关系），比群图谱（片段簇）更有价值。

### 数据落地实证（决定来源取舍的关键）

勘察分身知识在回复时的实际取数（`avatar_reply_graph.go` 的 `Prepare`）后确认：

| 来源 | 真实数据落地 | 是否有可画图谱的向量数据 |
|------|------------|------------------------|
| **记忆** | gracedb `avatar` namespace，metadata 含 `knowledge_memory_entities`/`themes` | ✅ 有（实体/主题） |
| **笔记** | gracedb `user_notes_{userID}` 集合 | ✅ 有（片段） |
| **文档 (knowledgeDocs)** | 其实是**所在会话「群知识库」**（`groupDocSvc.SearchKnowledge(group.ID,…)`），非个人文档集合 | ⚠️ 非个人图谱 |
| **任务 (tasks)** | 仅关系库 `model.Task`（按 user_id 查列表），**从不向量化** | ❌ 无向量数据 |

结论：**任务没有图谱数据、文档本质上属于群库而非"我的文档"**。为诚实与聚焦，v1 知识图谱只做「记忆 + 笔记」两个来源，不做文档/任务（避免空图/误导）。

目标：在分身设置面板新增一个独立的「知识图谱」一级 tab，用「记忆」和「笔记」两个来源分别可视化分身的知识。

## 已确认的决定

1. **图谱形态**：记忆实体图谱——节点=分身记忆反射出的主题/实体，边=共同出现在同一记忆中的实体对。笔记走片段簇。
2. **放置位置**：分身设置面板新增独立一级 tab「知识图谱」（与「普通设置」「高级设置」并列）。
3. **来源组织**：顶部来源选择器（**记忆 / 笔记**）单选，每次渲染**一类来源**，不与其它来源混叠。文档/任务因无个人向量数据，v1 不做。

## 设计

### 1. 图形构成（按来源子图）

| 来源 | 数据落地 | 渲染形态 |
|------|---------|---------|
| **记忆** | gracedb `avatar` namespace；每条记忆 metadata 含 `knowledge_memory_entities`/`themes` | **实体图谱**：节点=主题/实体 + count，边=记忆共同提及的实体对 + 共现权重。这是真·知识图谱 |
| **笔记** | gracedb `user_notes_{userID}` 集合 | **片段簇**：节点=笔记按标题切的片段，无边 |

来源选择器原生就支持扩展；将来若任务/个人文档补了向量化存储，只需 `source` 增加枚举与对应集合即可，不动结构。

### 2. 后端（服务端建图，一个端点）

新增 `GET /api/v1/avatar/knowledge-graph?source=memory|note&max_nodes=N`，由 JWT `user_id` 驱动（天然归属校验、无 IDOR）：

- **source=memory**：`AvatarMemoryService.GetUserMemories(userID, limit)` 枚举该用户全部 avatar 记忆 → 扫描 metadata 的 `knowledge_memory_entities` / `knowledge_memory_themes` → 聚合：
  - `nodes`：`{id, name, type:"entity"|"theme", count}`（同一实体/主题出现于多少记忆）
  - `edges`：`{source, target, weight}`（两个实体在同一记忆中共同出现，weight=共现记忆数）
  - `memories`：`{id, content}`（供点节点展示"包含此主题的记忆"）
- **source=note**：`VectorService.GetByCollection(ctx, fmt.Sprintf("user_notes_%d", userID), maxNodes)` 枚举笔记片段 → `nodes`（片段 + 来源元数据，如 note_id/title/type），`edges` 为空。

响应统一信封 `{nodes, edges, total_nodes, total_edges}`，与群图谱 `GetGroupKnowledgeGraph` 同构；`source=memory` 额外多返回 `memories: []{id, content}` 用于点节点的记忆列表展示（该字段仅 memory 来源有）。

### 3. 前端（独立一级 tab + 自包含组件）

- `AvatarSettingsPanel.vue`：`mainTabs` 增加 `{ key:'graph', label:'知识图谱', icon:'fas fa-project-diagram' }`；`activeMainTab` 类型加 `'graph'`；模板加 `v-else-if="activeMainTab === 'graph'"` 分支渲染新组件 `<AvatarGraph />`。
- **复用策略**：`AvatarGraph.vue` **自包含重写**（参照群 `AIGraph.vue` 的画布力导向渲染模式），不强行 DRY 合并到群 AIGraph。原因：分的实体图（节点+边+权重）与群片段簇形态不同，且分身面板上下文差异大；自包含降低改动风险、遵循 scope 纪律。
- `AvatarGraph.vue` 职责：
  - 顶部来源选择器（记忆/笔记），切换即拉取对应 `source`。
  - 拉取 `${serverUrl}/api/v1/avatar/knowledge-graph?source=X`。
  - 力导向渲染：记忆来源画节点+边（边宽/透明度随 weight），笔记来源画无边界点。
  - 点击节点：记忆来源显示"包含此主题的记忆"列表；笔记来源显示内容预览。
- **沿用已修的空白屏经验**：`loading` 置 false → `await nextTick()` → 再 `renderGraph`，确保图容器（`graphRef`）真正挂载后再画，避免空白。
- `serverUrl` 来源与其它 avatar 组件一致（`localStorage` serverUrl 或 `API_BASE_URL`）。

### 4. 错误处理

- 集合不存在 / 无记忆：返回空数组，前端显示"该来源暂无内容"空态。
- 向量服务未初始化：返回空图而非 500。
- 前端 fetch 失败：捕获 + 空态，不白屏。

### 5. 测试

- Go：聚合函数单测（构造含 entities 的假记忆，断言 nodes/edges 聚合正确；空输入返回空）。沿用既有 `AvatarMemoryService` 测试风格。
- 前端：`vue-tsc --noEmit` 0 error；`npm run build` 通过。
- 手工验证：分身设置 → 知识图谱 tab → 记忆/笔记两来源各切一遍，确认记忆出实体网、笔记出片段簇、点节点有详情。

## 涉及文件

- 后端（新增/改动）：
  - `qim-server/service/avatar_memory_service.go`（新增建图聚合方法）
  - `qim-server/service/avatar_memory_service_test.go`（聚合单测）
  - `qim-server/handler/avatar_handler.go`（新增 `GetAvatarKnowledgeGraph`）
  - `qim-server/app/routes.go`（注册 `authed.GET("/avatar/knowledge-graph", ...)`）
- 前端：
  - `qim-client/src/components/avatar/AvatarGraph.vue`（新组件）
  - `qim-client/src/components/avatar/AvatarSettingsPanel.vue`（加第三个一级 tab + 分支）

## 范围外（明确不做，v1 不折叠）

- **不做文档/任务两个来源**：任务从未向量化（只有关系库列表），文档其实是所在群的群知识库而非个人文档集合，均无个人向量图谱数据。来源选择器已留扩展位。
- 不把笔记片段通过 embedding 相似度挂接到记忆实体上（"相似度合并一张大图"本轮不做）；记忆/笔记各自独立成子图。
- 不画真实实体-关系知识图谱（如 RDF 三元组）；只用记忆反射已有的 entities/themes 做共现关系。
- 不把知识图谱做成可编辑/可增删。
