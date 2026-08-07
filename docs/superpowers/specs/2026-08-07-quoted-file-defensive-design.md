# 群 AI 被引用文件内容注入 — 防御性设计

日期:2026-08-07
状态:已确认(方案 B)

## 背景

群里用户发一个文件,引用它让群 AI 助手提取内容。后端在 `@AI` 提到且引用了文件消息时,尝试解析该文件正文注入 AI 上下文,让 AI 依据内容回答。

之前已落地整条链路(透传完整 `*model.Message` → `SmartReplyContext.QuotedMessageID` → `ExtractTextForContext` 解析正文 → `buildHistoryMessages` 注入),并逐步补充了失败分支。本次是**防御性收尾**:理清极端边界,消除内部矛盾,确保"大模型不自行判断、代码层区分情况"。

## 设计目标

1. **不依赖大模型识别错误场景**:每个边界都在 Go 层识别,转成明确的提示语。大模型只负责把提示语口语化成自然回复,不负责"猜"。
2. **诚实告知**:读不出时 AI 明确说"读不了"+给出建议,而不是假装读到或瞎编。
3. **消除矛盾**:修掉当前 `buildHistoryMessages` 中"失败时仍提示『我已读到被引用的文件内容』"的自相矛盾应答。

## 核心改动:拆分成功/失败上下文

### 1. 数据模型 `SmartReplyContext`(smart_reply_graph.go)

新增字段,与现有 `QuotedFileCtx` 互斥:

```go
// QuotedFileCtx 被引用文件【成功解析出的正文】(供 AI 依据内容回答),为空表示无正文。
QuotedFileCtx string

// QuotedFileFailed 被引用文件【读取失败/读不了】的说明(供 AI 如实告知用户),为空表示无失败。
// 与 QuotedFileCtx 互斥:成功时仅 QuotedFileCtx 非空,失败时仅 QuotedFileFailed 非空。
QuotedFileFailed string
```

### 2. 赋值逻辑 prepareInput

所有分支归类。成功→`QuotedFileCtx`;其余全部失败→`QuotedFileFailed`:

| 场景 | 字段 |
|------|------|
| 解析成功且正文非空 | `QuotedFileCtx`(注入正文,截断 4000 字符) |
| 解析成功但正文为空 | `QuotedFileFailed`(内容为空/无法提取文字) |
| 体积 > 20MB | `QuotedFileFailed`(过大,建议拆分) |
| 类型不支持 / 解析失败 / 存储读取失败 | `QuotedFileFailed`(转格式或传群知识库) |
| fileID 缺失 / Content 非 JSON | `QuotedFileFailed`(缺少可解析的文件信息) |
| 引用图片 / 视频 / 语音消息 | `QuotedFileFailed`(该类型无法解析) |

### 3. 应答 buildHistoryMessages

将"上下文注入"段(KnowledgeCtx / MemoryCtx / QuotedFileCtx / QuotedFileFailed)抽成纯函数 `buildContextBlocks`,去掉固定应答,按字段分支:

```go
if input.QuotedFileCtx != "" {
    result = append(..., &schema.Message{Role: User, Content: input.QuotedFileCtx})
    result = append(..., &schema.Message{Role: Assistant, Content: "我已读到被引用的文件内容。"})
}
if input.QuotedFileFailed != "" {
    result = append(..., &schema.Message{Role: User, Content: input.QuotedFileFailed})
    result = append(..., &schema.Message{Role: Assistant, Content: "我未能读取该文件,将如实向用户说明原因。"})
}
```

成功说"已读到",失败说"未能读取",消除歧义。抽纯函数是为让该段无 DB、可独立单测。

## 边界清单(已全部显式化,无静默假答)

- [x] 文件记录查不到(db.First 失败)
- [x] 存储未初始化(store == nil)
- [x] 体积 > 20MB(读入前按 file.Size 拦截)
- [x] 存储路径读不到(GetByPath 失败)
- [x] 临时文件创建 / 写入失败
- [x] 解析失败 / 类型不支持
- [x] 解析成功但正文为空
- [x] 引用文件但拿不到 fileID
- [x] 引用的是图片 / 视频 / 语音消息

有意保持静默的:引用的是普通文本消息(非文件相关),不打扰 AI 正常回答——这不算破坏场景。

## 错误处理

无需新增错误类型。失败统一走 `QuotedFileFailed`,成功正文走 `QuotedFileCtx`,两者互斥。

## 测试

将 `buildHistoryMessages` 里"上下文注入"段(KnowledgeCtx / MemoryCtx / QuotedFileCtx / QuotedFileFailed)抽成纯函数 `buildContextBlocks(input *SmartReplyContext) []*schema.Message`,该段不依赖 DB(历史消息查询在独立的后段),因此可无 DB 单测。

对 `buildContextBlocks` 断言:
1. 仅 `QuotedFileCtx` 非空 → 产出「我已读到被引用的文件内容。」应答,无「未能读取」应答
2. 仅 `QuotedFileFailed` 非空 → 产出「我未能读取该文件,将如实向用户说明原因。」应答,无「我已读到」应答
3. 两者皆空 → 无引用上下文消息
4. 两者互斥:不会同时注入两套上下文(由 prepareInput 赋值逻辑保证)

测试文件:`qim-server/service/smart_reply_graph_test.go`(新增)

## 范围

- 仅本次焦点:被引用文件内容注入的防御与一致性。
- 不做全链路白盒复盘、不做自动转换/分段读取等额外兜底(用户已确认保持"诚实告知"策略)。

## 涉及文件

- `qim-server/service/smart_reply_graph.go`(核心)
- `qim-server/service/group_document_service.go`(20MB 护栏,已具备,不改)
- 测试:新增 `smart_reply_graph_test.go`(如有)或并入现有 test 文件
