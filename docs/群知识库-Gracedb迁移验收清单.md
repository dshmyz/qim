# 群知识库 gracedb 迁移 · 端到端验收清单

> 目标：用一个**真实 PDF** 在群里走完整条链路，证明「cortexdb → gracedb」迁移后：
> ① 文档处理不 panic / 不维度报错；② 群助手提问能召回；③ 精确关键词也能召回（混合检索）；
> ④ 前端相似度 % 显示不回归。
>
> 需要：**真实火山方舟 API key（config.yaml 已配 doubao-embedding-vision）+ 一个真实可解析的 PDF**（非 `%%EOF` 伪 PDF）。

---

## 0. 前置准备

```bash
cd qim-server
# 确认配置：embedding 模型已指向火山方舟 + 向量库路径为 gracedb 目录
grep -A2 'embedding' config.yaml          # 应看到 doubao-embedding-vision
grep -A2 'vector:' config.yaml            # 应看到 ./data/gracedb
```

- 确认 `data/vector.db`（旧 cortexdb）已删除，`data/gracedb` 为全新空目录（或本地无残留）。
- 启动服务：`go run main.go`（观察启动日志应有 `VectorService 初始化成功 path=./data/gracedb`）。
- 用带群管理权限的账号登录客户端，进入目标群聊。

---

## 1. 上传真实 PDF 到群并绑定知识库

### 前置：把 PDF 上传到群文件空间
在群聊 → 文件 → 上传一个**真实 PDF**（内容建议包含：一段概念描述 + 一个明显可搜索的精确编号/人名，便于后面验证两个召回维度）。

### 绑定：群 AI 设置 → 知识库 → 绑定文档
1. 打开群 AI 设置（GroupAIPanel）→「知识库 / AIKnowledgeSettings」。
2. 在「绑定文档」下拉中把刚上传的 PDF **手动绑定**（注意：选项来自群文件空间，若下拉为空见下方常见问题 A）。

### 触发处理
绑定后文档应出现状态徽标，或手动点「处理」/「重试」：
- 预期：`pending → processing → completed`
- **通过标准**：状态停到 **completed**，`chunk_count > 0`，无 `failed`。

### 落库校验（后端命令）
```bash
# 看群记忆/文档状态
# 若失败，直接查该文档的处理错误信息（响应里的 process_error）
curl -s "http://localhost:8080/api/v1/groups/<groupID>/ai-documents?token=<JWT>"
```
状态字段：`pending / processing / completed / failed`。

---

## 2. 验证「处理 → 已完成」（核心：不再 panic / 维度报错）

**用例 2a · 观察服务端日志**：处理期间按 `Ctrl+C` 前抓日志，确认：
```bash
grep -E 'panic|index out of range|维度|HNSW|VectorService|文档处理完成' <运行日志>
```
- 预期：**无 panic**、无 `index out of range [1024] with length 1024` 这类 cortexdb 维度崩溃
- 预期出现：`文档处理完成 doc_id=… chunks=N`

**用例 2b · 管理后台看向量数据**：打开 admin → 向量数据管理（VectorData.vue）
- 预期：左侧 collections 出现 `group_<id>`，count = 该文档切片数
- 点开能看到每条切片的 content / metadata（group_id / doc_id / title）

---

## 3. 验证群助手提问 → 语义召回

**用例 3a · 语义命中**：在群里 @AI，问出一个与文档内容**同义改写**的问题（不用文档原句）。
- 例：文档写「年假需提前三天走审批流程」，就问「我想请假怎么申请？有什么流程」
- **通过标准**：AI 回复里带了文档内容 / 前端「知识来源」折叠标签出现，命中文档

**用例 3b · 精确关键词命中（混合检索的核心收益）**：@AI 问文档里的**精确编号 / 人名 / 唯一串**。
- 例：文档有 `PRD-2024-001`，就问「PRD-2024-001 是干嘛的」
- **通过标准**：能命中该段落。这是纯语义容易漏、FTS 词法补上的场景——如果命中，说明**混合检索确实生效**。

---

## 4. 验证前端相似度 % 显示不回归

- 在「知识来源」标签展开，看每个来源的相似度百分比。
- **通过标准**：显示的百分比是合理范围（如 60%~95%），**不是** 1%~3% 这种 RRF 小分数
- 说明：迁移后的 `searchHybrid` 用 RRF 排序、但**展示分数保留语义余弦分（0-1）**；若看到 1-3% 说明展示分回退逻辑没生效，需回查 `hybridDisplayScores`。

---

## 5. 回归：笔记 RAG（受影响路径）

> 群/笔记知识检索共用 `SearchKnowledgeWithMode`。抽查一条不回归。

- 分身/笔记里绑定过知识库的用户，用笔记语义检索问一个之前能命中的问题。
- **通过标准**：仍能召回（确认混合检索没有破坏原本的语义召回路径）。

---

## 常见问题排查

| 现象 | 可能原因 | 处理 |
|------|---------|------|
| **A. 绑定文档下拉为空** | 群 AI 可绑定文档来自「群文件空间」(`scope_type='conversation'`)，文件没进群文件夹 | 把 PDF 上传/移动到群文件夹，或用 `all=1` 忽略层级 |
| **B. 状态停在 failed，process_error「维度报错/not found」** | gracedb 集合未确保创建 | 确认走的是 `ensureGracedbCollection`（Upsert 前建集合）；旧 `data/vector.db` 是否已清 |
| **C. 状态停在 processing 超时** | embedding 调用火山接口慢/超时 / 网络 | 看服务端日志 `切片向量化失败` 具体 error；确认 key 有效 |
| **D. 文档解析失败「内容为空」** | PDF 是伪 PDF（缺 `%%EOF`）或扫描件无文本层 | 换成带文本层的真实 PDF |
| **E. 提问完全答不上/无知识来源** | 未绑定成功（回到用例 1）/ 知识库没被挂到群 AI | 确认绑定后重新触发处理，再看日志 |

---

## 通过标准汇总

- [ ] 文档 `completed`，无 panic / 维度报错（用例 2a/2b）
- [ ] 管理后台能看到 `group_<id>` 集合与切片（用例 2b）
- [ ] 语义同义改写问题能命中（用例 3a）
- [ ] 精确编号/关键词能命中（用例 3b，混合检索生效）
- [ ] 前端相似度 % 显示在合理区间（用例 4）
- [ ] 笔记 RAG 无明显回归（用例 5）

---

> 这份清单聚焦「迁移后群知识链路是否真的可用」。跑完若某条不通过，把对应的 `process_error` / 服务端日志片段贴回来，我据此定位是迁移回归还是环境问题。
