# 群 AI 被引用图片多模态接入

日期:2026-08-07
状态:已落地

## 背景

群里用户发一张图片,引用它让群 AI 助手识别内容。后端在 `@AI` 提到且引用了图片消息时,把图片原始字节转成 base64 data URL 注入 AI 上下文,让支持视觉的模型真正"看图"回答,而不是沿用"无法解析该类型"的兜底话术。

## 设计目标

1. **让 AI 真正看图**:被引用图片走多模态路径,注入 base64 data URL 给视觉模型,由模型识别内容后回答。
2. **复用现有模型路由**:不硬编码 openai/anthropic,而是复用 `ModelRouter` 的 `TaskTypeVision`。管理员可在 Router 页配置视觉模型;未配置时由路由回退到默认任务,保证多模型/自定义配置下能通。
3. **诚实降级**:图片读不出(过大/读取失败/缺 id)时沿用 `QuotedFileFailed` 提示语,让 AI 如实说"看不到图",不假装读到。
4. **不扩散范围**:仅流式路径(`ExecuteStream`)走视觉任务;工具指令路径(`ExecuteWithTools`)保持 chat,因为管理指令(踢人/禁言)与看图无关。

## 数据链路

```
被引用图片消息(Content={"url":<storagePath>,"id":<fileID>})
  → prepareInput Type=="image" 分支
      → GroupDocumentService.ImageURLForContext(fileID)   // 读字节 → base64 data URL
      → 成功: SmartReplyContext.Quoted = &QuotedContext{Kind: QuotedImage, Name, Text, ImageURL}
      失败: SmartReplyContext.Quoted = &QuotedContext{Kind: QuotedFailed, Name, Text}   // "看不了图"提示语
  → buildContextBlocks / buildQuotedContextMessage
      → 图片块经 schema.Message.MultiContent 携带 image_url data URL
  → ExecuteStream: Quoted.Kind==QuotedImage → ai.TaskTypeVision (路由回退); 其他 → TaskTypeChat
  → einoMessagesToAIMessages / imageURLFromMessage
      → MultiContent 的 data URL 提取到 ai.Message.ImageURL
  → ai.Message.MarshalJSON 转 OpenAI image_url 数组 → 视觉模型识别
```

## 核心改动

### 1. GroupDocumentService.ImageURLForContext (group_document_service.go)

复用 `ExtractTextForContext` 的读取骨架(fileID → `File.StoragePath` → `store.GetByPath` → 字节),新差异:

- 返回 `base64 data URL` 而非解析文本。
- 大小护栏沿用 5MB(`quoteMaxImageSize`,`ImageURLForContext` 内按 `file.Size` 先拦截入内存,读入后二次校验实际字节数),与 `TranslateImage` 的 `maxImageTranslateSize` 一致(图片以 base64 注入有 ~1.33x 膨胀)。
- 返回哨兵错误 `ErrQuotedImageTooLarge`,与 `ErrQuotedFileTooLarge` 同模式,供 prepareInput 区分"图片过大"与"读取失败"。

### 2. QuotedDocumentReader 接口扩展 (smart_reply_graph.go)

```go
type QuotedDocumentReader interface {
    ExtractTextForContext(fileID uint) (name string, text string, err error)
    ImageURLForContext(fileID uint) (name string, dataURL string, err error)
}
```

`*GroupDocumentService` 已实现,编译期断言。接口新增方法后,`g.quotedFile` 直接可调用;注入 nil 时整段(含图片)不执行。

### 3. SmartReplyContext：三个字符串字段重构为 Quoted 判别联合

原 `QuotedFileCtx` / `QuotedFileFailed` / `QuotedImageURL` 三个字符串字段把同一对"成功/失败"语义散在三处，且图片成功时需 `QuotedFileCtx`（提示词）与 `QuotedImageURL`（数据）并存、又都和 `QuotedFileFailed` 互斥，互斥关系靠注释维系。改为判别联合：

```go
type QuotedKind string

const (
	QuotedNone   QuotedKind = ""       // 无被引用对象
	QuotedFile   QuotedKind = "file"   // 成功读到文件正文
	QuotedImage  QuotedKind = "image"  // 成功读到图片（多模态）
	QuotedFailed QuotedKind = "failed" // 读取失败/类型不支持/过大/缺信息
)

type QuotedContext struct {
	Kind     QuotedKind
	Name     string // 可读名称（文件名等）
	Text     string // 注入 prompt 的成句内容（含提示语）
	ImageURL string // image: base64 data URL
}
```

`SmartReplyContext` 只保留一个字段：

```go
// Quoted 被引用对象（文件正文 / 图片 / 读取失败）的上下文注入，nil 表示无被引用内容。
Quoted *QuotedContext
```

成功/失败成单一 `Quoted.Kind` 分支，`buildContextBlocks` 与 `ExecuteStream` 用 switch 判别；新增媒体类型（audio/video）无需再加字段。

### 4. prepareInput 图片分支

原 `image/video/audio` 合并分支拆开:`image` 走多模态,`video/audio` 维持"无法解析"兜底。

| 图片场景 | 字段 |
|---------|------|
| parseQuotedFileID 拿不到 id | `QuotedFileFailed`(缺少图片信息) |
| ImageURLForContext 成功 | `QuotedImageURL` + `QuotedFileCtx`(提示语) |
| 图片 > 5MB | `QuotedFileFailed`(过大,建议压缩重发) |
| 读取失败/存储失败 | `QuotedFileFailed`(看不到图) |

### 5. 消息链路携带图片 + vision 路由

- `buildContextBlocks` → `buildQuotedContextMessage`:图片时用户消息块带 `schema.Message.MultiContent`(text 部分 + `ChatMessagePartTypeImageURL` 部分,`ChatMessageImageURL.URL` 为 data URL)。data URL **不**平铺进 Content,防止大段 base64 污染模型文本。
- `ExecuteStream`(smart_reply_graph.go):`input.QuotedImageURL != ""` 时用 `ai.TaskTypeVision` 建 `EinoChatModel`,否则 `TaskTypeChat`;未配置视觉任务时 `ModelRouter.SelectProvider` 回退默认任务。
- `ExecuteWithTools` 保持 `TaskTypeChat`(管理指令不看图)。
- `einoMessagesToAIMessages`(eino_chat_model.go):新增 `imageURLFromMessage` 从 `MultiContent` 提取 data URL 落到 `ai.Message.ImageURL`,由 `ai.Message.MarshalJSON` 转 OpenAI `image_url` 数组格式;无图片部分返回空串,不影响原有文本消息。

## 边界清单

- [x] 图片记录查不到(db.First 失败)→ 读失败兜底
- [x] 存储未初始化(store == nil)→ 读失败兜底
- [x] 图片 > 5MB(读入前按 file.Size 拦截 + 读入后二次校验)→ ErrQuotedImageTooLarge 提示语
- [x] 存储路径读不到(GetByPath 失败)→ 读失败兜底
- [x] 引用图片但拿不到 fileID → 缺少图片信息提示语
- [x] 视频 / 语音保持"无法解析"兜底
- [x] 视觉任务未配置 → ModelRouter 回退默认任务,不硬编码厂商

有意保持静默:引用普通文本消息(非文件/图片),不打扰 AI 正常回答。

## 测试

新增 `qim-server/service/smart_reply_image_test.go`(无 DB / 内存 DB):

1. `TestBuildContextBlocks_Image` / `TestBuildContextBlocks_ImageFailed` — 图片成功块带 image_url data URL、失败块无图,各自配相应应答。
2. `TestEinoMessagesToAIMessages_Image` — `MultiContent` data URL 透传到 `ai.Message.ImageURL`,普通文本消息不受影响。
3. `TestImageURLForContext` — 内存 sqlite + 临时目录存储,图片读成 data URL 且确定性。
4. `TestImageURLForContext_TooLarge` — 超 5MB 返回 `ErrQuotedImageTooLarge`。
5. `TestImageURLForContext_Missing` — 记录查不到返回错误。
6. 编译期断言 `GroupDocumentService` 实现扩展后的 `QuotedDocumentReader`。

## 范围

- 仅本次焦点:被引用图片多模态接入 + 诚实降级。
- 不做自动缩放/上传压缩等额外兜底;图片过大保持"诚实告知"策略(用户已确认)。
- 不触碰文档正文注入(QuotedFileCtx)既有行为。

## 涉及文件

- `qim-server/service/group_document_service.go`(ImageURLForContext + 常量/哨兵错误 + mime/http imports)
- `qim-server/service/smart_reply_graph.go`(接口扩展 / QuotedImageURL / prepareInput 图片分支 / buildContextBlocks / buildQuotedContextMessage / ExecuteStream vision 路由)
- `qim-server/service/eino_chat_model.go`(imageURLFromMessage + einoMessagesToAIMessages 透传)
- 测试:新增 `qim-server/service/smart_reply_image_test.go`
