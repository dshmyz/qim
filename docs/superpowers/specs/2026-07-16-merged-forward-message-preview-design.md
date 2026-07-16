# 合并转发消息摘要设计

## 目标

让合并转发卡片与引用消息使用同一套类型感知的摘要规则，避免文件、分享、小程序、资讯等结构化内容以原始 JSON 显示。

## 方案

- 新建 `utils/messagePreview.ts`，导出 `getMessagePreview(message)`，返回展示文本与语义类型。
- 文件解析复用现有 `name` / `fileName`、`size` / `fileSize` 与 URL 回退规则；输出“文件名 · 可读大小”。
- 图片输出“图片”；Markdown 先转为纯文本再截断；分享、小程序、资讯优先使用 JSON 内的名称或标题。
- 未知类型先识别文件 JSON；仍无法识别时输出“未知消息”，绝不回显整个 JSON。
- `MergedForwardMessage.vue` 使用该工具；`MessageItem.vue` 的引用消息摘要同步调用它，保留现有类型标签和跳转行为。

## 边界与验证

- 不改消息存储、接口与转发 JSON 格式。
- 解析失败必须安全降级，不抛出异常。
- 针对文件、分享、Markdown、未知 JSON 建立单元测试，并验证合并转发卡片不显示原始 JSON。
