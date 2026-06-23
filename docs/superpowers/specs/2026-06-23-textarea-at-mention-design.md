# Textarea @ 提及补全

## 目标

在群聊和讨论组中，用户在消息 textarea 输入 `@` 后继续输入的字符应实时筛选成员；键盘焦点始终留在 textarea。选择候选项后，完整的 `@查询词` 必须被替换为提及文本。

## 状态边界

`ChatWindow` 是唯一的提及 token 状态拥有者：它从 textarea 文本与光标位置解析有效 token，维护 token 的起止位置、筛选词和 `mentionSpans`。

`MessageInput` 只呈现由父层提供的筛选词和候选人，并在 textarea 仍聚焦时处理候选列表的键盘导航。它不再拥有独立的提及搜索输入或搜索状态。

## 行为

1. 有效 token 的 `@` 必须位于文本开头或空白字符之后；邮箱和 URL 中的 `@` 不触发。
2. token 从 `@` 延伸到下一个空白字符或文本末尾；光标在 token 内时，`@` 后至光标前的字符为筛选词。
3. 输入、删除或移动光标后重新解析 token。没有有效 token 时关闭候选面板。
4. 面板打开时，ArrowUp、ArrowDown、Enter 和 Escape 在 textarea 中控制候选列表；Enter 不发送消息。
5. 选择成员或“所有人”时，替换整个 token，并在插入文本的尾随空格后恢复 textarea 焦点和光标。
6. 记录的 `MentionSpan` 精确覆盖插入后的 `@姓名`，不包括尾随空格。

## 测试

- 纯 token 解析：有效边界、邮箱/URL、中文和英文筛选词、多个 token、光标位于 token 中间。
- 完整替换：`@ali` 选择 Alice 后成为 `@Alice `，不会保留 `ali`。
- 组件交互：面板打开时 textarea 的方向键与 Enter 操作候选项且不会触发发送。
