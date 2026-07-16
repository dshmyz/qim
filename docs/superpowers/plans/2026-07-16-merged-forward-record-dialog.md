# 合并转发聊天记录弹层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将合并转发卡片改为三条预览入口，并在独立只读弹层展示完整聊天记录。

**Architecture:** `MergedForwardMessage` 只显示标题、三条摘要和打开事件。新建 `MergedForwardRecordDialog` 接收既有 `MergedForwardPayload`，复用公共消息摘要工具按类型显示完整记录和时间分隔，不改变后端或转发快照。

**Tech Stack:** Vue 3、TypeScript、Vitest、现有 Font Awesome 与主题变量。

## Global Constraints

- 不改消息存储、接口与转发 JSON 格式。
- 不提供原消息跳转、回复、撤回或编辑；内容是转发时快照。
- 解析异常显示安全降级内容，不展示原始 JSON。
- 弹层可由遮罩、关闭按钮和 Escape 关闭，不影响原聊天消息状态。
- 不覆盖已有未提交用户改动。

---

### Task 1: 卡片预览和弹层组件

**Files:**
- Create: `qim-client/src/components/message/MergedForwardRecordDialog.vue`
- Modify: `qim-client/src/components/message/MergedForwardMessage.vue`
- Modify: `qim-client/tests/unit/components/MessageItem.test.ts`

**Interfaces:**
- Produces: `MergedForwardRecordDialog` props `payload: MergedForwardPayload | null`, `visible: boolean`, event `close`.
- Consumes: `getMessagePreview`, `MergedForwardPayload` and card click events.

- [ ] **Step 1: Write failing component tests**

```ts
it('shows only three previews and opens a complete record dialog', async () => {
  const wrapper = mount(MergedForwardMessage, { props: { content: makePayload(4) } })
  expect(wrapper.findAll('.merged-forward-preview').length).toBe(3)
  await wrapper.get('[data-testid="merged-forward-open"]').trigger('click')
  expect(wrapper.find('[data-testid="merged-forward-record-dialog"]').exists()).toBe(true)
  expect(wrapper.text()).toContain('聊天记录（共 4 条）')
})
```

- [ ] **Step 2: Verify red**

Run: `cd qim-client && npm exec vitest run tests/unit/components/MessageItem.test.ts`

Expected: FAIL because the card expands inline and no dialog exists.

- [ ] **Step 3: Implement card and dialog**

```vue
<!-- card -->
<button data-testid="merged-forward-open" type="button" @click="isRecordVisible = true">查看聊天记录</button>
<MergedForwardRecordDialog :visible="isRecordVisible" :payload="payload" @close="isRecordVisible = false" />
```

Render `payload.messages.slice(0, 3)` with sender and `getMessagePreview(message).label`; show `还有 N 条消息` for remaining items. In the dialog, render records in source order, use `getMessagePreview` for non-text messages, and insert a timestamp divider when the current item is more than 300000 milliseconds after the previous item. Add a close button, backdrop click handler, and keydown listener that closes only on Escape.

- [ ] **Step 4: Verify green**

Run: `cd qim-client && npm exec vitest run tests/unit/components/MessageItem.test.ts`

Expected: PASS with 0 failures.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/components/message/MergedForwardMessage.vue qim-client/src/components/message/MergedForwardRecordDialog.vue qim-client/tests/unit/components/MessageItem.test.ts
git commit -m "feat: view merged forwards in a record dialog"
```

### Task 2: 完整记录视觉与行为回归

**Files:**
- Modify: `qim-client/src/components/message/MergedForwardRecordDialog.vue`
- Modify: `qim-client/tests/unit/components/MessageItem.test.ts`

**Interfaces:**
- Consumes: `payload.messages` and `visible`.
- Produces: 可读的只读消息流、时间分隔、无效负载降级和 Escape 关闭。

- [ ] **Step 1: Write failing edge-case tests**

```ts
it('adds a time divider and closes the record dialog on Escape', async () => {
  const wrapper = mount(MergedForwardRecordDialog, { props: { visible: true, payload: makePayloadWithGap(301000) } })
  expect(wrapper.find('[data-testid="merged-forward-time-divider"]').exists()).toBe(true)
  await window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
  expect(wrapper.emitted('close')).toHaveLength(1)
})
```

- [ ] **Step 2: Verify red**

Run: `cd qim-client && npm exec vitest run tests/unit/components/MessageItem.test.ts`

Expected: FAIL because the dialog lacks time-divider and Escape behavior.

- [ ] **Step 3: Implement interaction and theme styles**

Use `Teleport` only if existing test setup supports it; otherwise render in component root with `position: fixed`. Style the overlay with `var(--modal-overlay)` fallback, constrain dialog to `min(720px, calc(100vw - 32px))`, make body scrollable, and give each record sender/content hierarchy. Guard missing payload with `聊天记录无法加载`.

- [ ] **Step 4: Verify focused suite and build**

Run: `cd qim-client && npm exec vitest run tests/unit/utils/messagePreview.test.ts tests/unit/components/MessageItem.test.ts && npm run build`

Expected: both commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/components/message/MergedForwardRecordDialog.vue qim-client/tests/unit/components/MessageItem.test.ts
git commit -m "style: polish merged forward record dialog"
```

