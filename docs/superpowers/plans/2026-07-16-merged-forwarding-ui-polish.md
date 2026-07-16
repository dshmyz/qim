# 合并转发界面精修 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化合并转发卡片、多选勾选和底部操作栏的视觉层级、响应式布局及键盘可用性。

**Architecture:** 保持现有 Vue 组件与消息数据接口不变。样式收敛在 `MergedForwardMessage.vue`、`ChatWindow.vue` 和 `MessageListView.vue`，以现有 CSS 变量实现主题适配；只添加必要的数据属性供组件测试断言。

**Tech Stack:** Vue 3、TypeScript、Scoped CSS、Vitest、Vite。

## Global Constraints

- 只修改呈现与交互样式；不改消息 JSON、接口、选择或发送逻辑。
- 所有颜色来自既有 CSS 变量，并为键盘 focus 提供清晰轮廓。
- 合并转发卡片在 640px 以下保持可读和可点击；多选操作栏在窄屏紧凑排列。
- 不覆盖工作区已有未提交的 `menus.css`、`FileManagementApp.vue`、`MessageContextMenu.vue`、`MainContextMenus.vue` 改动。

---

### Task 1: 精修合并转发卡片

**Files:**
- Modify: `qim-client/src/components/message/MergedForwardMessage.vue:1-99`
- Modify: `qim-client/tests/unit/components/MessageItem.test.ts`

**Interfaces:**
- Consumes: 既有 `content`、`isSelf` 与 `merged-forward-toggle` 测试标识。
- Produces: 带图标、摘要层级、键盘焦点态和窄屏规则的可展开卡片。

- [ ] **Step 1: 写入失败的样式结构测试**

```ts
it('marks the merged-forward card as responsive and its toggle as accessible', () => {
  const source = readFileSync(resolve(__dirname, '../../../src/components/message/MergedForwardMessage.vue'), 'utf8')
  expect(source).toContain('class="merged-forward-icon"')
  expect(source).toContain('aria-expanded="expanded"')
  expect(source).toContain('@media (max-width: 640px)')
  expect(source).toContain(':focus-visible')
})
```

- [ ] **Step 2: 验证测试为红**

Run: `cd qim-client && npm exec vitest run tests/unit/components/MessageItem.test.ts`

Expected: FAIL because the visual hierarchy and accessibility markers do not exist.

- [ ] **Step 3: 实现卡片视觉层级**

```vue
<div class="merged-forward-header">
  <span class="merged-forward-icon" aria-hidden="true"><i class="fas fa-comments"></i></span>
  <span class="merged-forward-title">聊天记录（{{ payload.messages.length }}条）</span>
  <button :aria-expanded="expanded" data-testid="merged-forward-toggle" type="button" @click="expanded = !expanded">
    {{ expanded ? '收起' : '展开' }} <i :class="expanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'" aria-hidden="true"></i>
  </button>
</div>
```

将卡片宽度改为 `min(360px, 100%)`，加入柔和阴影、标题与摘要层级、两行截断、`:focus-visible` 轮廓，并在 `@media (max-width: 640px)` 中降低内边距与圆角。图片和文件摘要分别加 `fa-image` 和 `fa-file` 图标，文字消息保留原文摘要。

- [ ] **Step 4: 验证测试通过**

Run: `cd qim-client && npm exec vitest run tests/unit/components/MessageItem.test.ts`

Expected: PASS with 0 failures.

- [ ] **Step 5: 提交**

```bash
git add qim-client/src/components/message/MergedForwardMessage.vue qim-client/tests/unit/components/MessageItem.test.ts
git commit -m "style: polish merged forward card"
```

### Task 2: 精修多选控件与操作栏

**Files:**
- Modify: `qim-client/src/components/chat/ChatWindow.vue:73-77,2684-2701`
- Modify: `qim-client/src/components/chat/MessageListView.vue:43-51,332-339`
- Modify: `qim-client/tests/unit/components/ChatWindowMergedForward.test.ts`

**Interfaces:**
- Consumes: 现有 `selectedMessages.length`、`cancelMessageSelection` 与 `forwardSelectedMessages`。
- Produces: 使用 `message-selection-actions`、`message-selection-control` 的主次操作视觉和无障碍标签，不改变选择事件。

- [ ] **Step 1: 写入失败的样式结构测试**

```ts
it('keeps a labelled selection toolbar with primary and secondary actions', () => {
  const source = readFileSync(resolve(__dirname, '../../../src/components/chat/ChatWindow.vue'), 'utf8')
  expect(source).toContain('aria-label="多选消息操作"')
  expect(source).toContain('class="message-selection-cancel"')
  expect(source).toContain('class="message-selection-forward"')
  expect(source).toContain('@media (max-width: 640px)')
})
```

- [ ] **Step 2: 验证测试为红**

Run: `cd qim-client && npm exec vitest run tests/unit/components/ChatWindowMergedForward.test.ts`

Expected: FAIL because toolbar semantics and visual class names do not exist.

- [ ] **Step 3: 实现工具栏与勾选视觉**

```vue
<div v-if="isMessageSelectionMode" class="message-selection-actions" aria-label="多选消息操作" role="toolbar">
  <span class="message-selection-count">已选择 {{ selectedMessages.length }} 条</span>
  <button class="message-selection-cancel" type="button" @click="cancelMessageSelection">取消</button>
  <button class="message-selection-forward" type="button" :disabled="selectedMessages.length < 2" @click="forwardSelectedMessages">合并转发</button>
</div>
```

为 `.message-selection-actions` 使用半透明 `var(--card-bg)`、顶部阴影和安全间距；取消按钮使用边框次操作，转发按钮使用 `var(--primary-color)` 与明确禁用态。将 checkbox 包进圆形控件，选中时使用主题主色、阴影与 44px 最小点击区域；在 640px 以下让计数独占第一行、两个按钮均分第二行。

- [ ] **Step 4: 验证测试通过**

Run: `cd qim-client && npm exec vitest run tests/unit/components/ChatWindowMergedForward.test.ts tests/unit/components/MessageListView.test.ts`

Expected: PASS with 0 failures.

- [ ] **Step 5: 完整验证并提交**

Run: `cd qim-client && npm exec vitest run tests/unit/components/MessageItem.test.ts tests/unit/components/ChatWindowMergedForward.test.ts tests/unit/components/MessageListView.test.ts && npm run build`

Expected: all commands exit 0.

```bash
git add qim-client/src/components/chat/ChatWindow.vue qim-client/src/components/chat/MessageListView.vue qim-client/tests/unit/components/ChatWindowMergedForward.test.ts
git commit -m "style: refine message selection controls"
```

