# 合并转发消息摘要 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为合并转发和引用消息提供统一、类型感知且不泄露 JSON 的摘要展示。

**Architecture:** 新建纯函数 `messagePreview.ts`，将消息类型和内容规范化为 `label` 与 `kind`；组件只负责图标及布局。合并转发卡片和 MessageItem 的引用预览调用同一函数，维持既有存储和接口不变。

**Tech Stack:** Vue 3、TypeScript、Vitest。

## Global Constraints

- 不改消息存储、接口与转发 JSON 格式。
- 解析失败必须安全降级，不抛出异常。
- 文件、分享、Markdown、未知 JSON 均不可直接显示原始 JSON。
- 不覆盖现有未提交的用户改动。

---

### Task 1: 创建统一消息摘要工具并接入卡片

**Files:**
- Create: `qim-client/src/utils/messagePreview.ts`
- Create: `qim-client/tests/unit/utils/messagePreview.test.ts`
- Modify: `qim-client/src/components/message/MergedForwardMessage.vue:1-55`
- Modify: `qim-client/tests/unit/components/MessageItem.test.ts`

**Interfaces:**
- Produces: `getMessagePreview({ type, content }): { kind: 'text' | 'image' | 'file' | 'share' | 'miniApp' | 'news' | 'unknown'; label: string }`.
- Consumes: message `type` and serialized `content`.

- [ ] **Step 1: Write the failing utility tests**

```ts
it('formats structured message content without exposing JSON', () => {
  expect(getMessagePreview({ type: 'file', content: JSON.stringify({ name: '方案.pdf', size: 1024 }) }))
    .toEqual({ kind: 'file', label: '方案.pdf · 1 KB' })
  expect(getMessagePreview({ type: 'share', content: JSON.stringify({ name: '设计说明' }) }))
    .toEqual({ kind: 'share', label: '分享：设计说明' })
  expect(getMessagePreview({ type: 'unknown', content: JSON.stringify({ raw: true }) }))
    .toEqual({ kind: 'unknown', label: '未知消息' })
})
```

- [ ] **Step 2: Verify red**

Run: `cd qim-client && npm exec vitest run tests/unit/utils/messagePreview.test.ts`

Expected: FAIL because the module does not exist.

- [ ] **Step 3: Implement the pure formatter**

```ts
export type MessagePreview = { kind: 'text' | 'image' | 'file' | 'share' | 'miniApp' | 'news' | 'unknown'; label: string }

export const getMessagePreview = ({ type, content }: { type: string; content: string }): MessagePreview => {
  const data = safeParse(content)
  if (type === 'image') return { kind: 'image', label: '图片' }
  if (type === 'file' || isFilePayload(data)) return { kind: 'file', label: formatFile(data, content) }
  if (type === 'share') return { kind: 'share', label: `分享：${data?.name || '内容'}` }
  if (type === 'miniApp' || type === 'mini-app') return { kind: 'miniApp', label: `小程序：${data?.name || '未命名'}` }
  if (type === 'news') return { kind: 'news', label: data?.title || '资讯' }
  if (type === 'markdown') return { kind: 'text', label: stripMarkdown(content) || '无内容' }
  return data ? { kind: 'unknown', label: '未知消息' } : { kind: 'text', label: content || '无内容' }
}
```

Use finite numeric size to render B/KB/MB; `safeParse` returns null on invalid JSON; `stripMarkdown` removes code markers, heading/list prefixes and link URLs before truncating to 120 characters. In `MergedForwardMessage.vue`, replace `fileName` and `messagePreview` with `getMessagePreview(message)`; map its `kind` to existing Font Awesome icons and render `preview.label`.

- [ ] **Step 4: Verify green**

Run: `cd qim-client && npm exec vitest run tests/unit/utils/messagePreview.test.ts tests/unit/components/MessageItem.test.ts`

Expected: PASS with 0 failures.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/utils/messagePreview.ts qim-client/tests/unit/utils/messagePreview.test.ts qim-client/src/components/message/MergedForwardMessage.vue qim-client/tests/unit/components/MessageItem.test.ts
git commit -m "feat: format merged forward previews"
```

### Task 2: 复用摘要工具到引用消息预览

**Files:**
- Modify: `qim-client/src/components/message/MessageItem.vue:39-83,305-327`
- Modify: `qim-client/tests/unit/components/MessageItem.test.ts`

**Interfaces:**
- Consumes: `getMessagePreview({ type, content })`.
- Produces: 引用消息展示类型标签加已格式化摘要，不回显文件或分享 JSON。

- [ ] **Step 1: Write the failing component test**

```ts
it('uses the shared preview for a quoted file message', () => {
  const wrapper = mount(MessageItem, {
    props: { message: makeMessage({ quotedMessage: { id: 'q1', type: 'file', content: JSON.stringify({ name: '方案.pdf', size: 1024 }), sender: { name: '甲' } } }) },
  })
  expect(wrapper.text()).toContain('方案.pdf · 1 KB')
  expect(wrapper.text()).not.toContain('{"name"')
})
```

- [ ] **Step 2: Verify red**

Run: `cd qim-client && npm exec vitest run tests/unit/components/MessageItem.test.ts`

Expected: FAIL because the quoted preview has its own per-type formatting.

- [ ] **Step 3: Replace duplicated branches**

Import `getMessagePreview`, compute `quotedPreview` from `message.quotedMessage`, and render:

```vue
<div class="quoted-message-preview-content">
  [{{ quotedPreview.kind === 'image' ? '图片' : quotedPreview.kind === 'file' ? '文件' : quotedPreview.kind === 'share' ? '分享' : quotedPreview.kind === 'miniApp' ? '小程序' : quotedPreview.kind === 'news' ? '资讯' : '消息' }}]
  {{ quotedPreview.label }}
</div>
```

Keep the existing click-to-scroll event and sender header unchanged. Remove only `getFileName` and `isFileContent` when no longer referenced.

- [ ] **Step 4: Verify green and build**

Run: `cd qim-client && npm exec vitest run tests/unit/utils/messagePreview.test.ts tests/unit/components/MessageItem.test.ts && npm run build`

Expected: both commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/components/message/MessageItem.vue qim-client/tests/unit/components/MessageItem.test.ts
git commit -m "refactor: share message preview formatting"
```

