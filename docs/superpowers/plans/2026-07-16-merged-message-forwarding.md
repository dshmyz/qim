# 多条消息合并转发 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让用户勾选多条聊天消息，并以一张可展开的聊天记录卡片转发到已有的联系人或群组。

**Architecture:** 新建纯 TypeScript 工具负责快照与安全解析。ChatWindow 独占多选状态，沿用既有 `forwardMessage` 全局事件进入分享弹窗；useShareLogic 只序列化并发送。MessageItem 委托独立卡片组件渲染 `merged_forward`，服务端按普通消息持久化和广播。

**Tech Stack:** Vue 3、TypeScript、Pinia、Vitest、既有 REST/WebSocket 消息通道。

## Global Constraints

- 不新增表、不迁移数据库；`messages.content` 保存版本化 JSON 快照。
- 第一版仅支持文本、Markdown、图片和文件摘要；不提供原消息跳转、嵌套转发或实时追溯。
- 选择顺序与消息列表一致，撤回消息不可选。
- 非法内容安全降级为“聊天记录无法加载”。
- 不覆盖工作区已有的 `menus.css`、`FileManagementApp.vue`、`MessageContextMenu.vue`、`MainContextMenus.vue` 未提交改动。

---

### Task 1: 定义快照协议与卡片

**Files:**
- Create: `qim-client/src/utils/mergedForward.ts`
- Create: `qim-client/src/components/message/MergedForwardMessage.vue`
- Create: `qim-client/tests/unit/utils/mergedForward.test.ts`
- Modify: `qim-client/src/types/index.ts:34-72`
- Modify: `qim-client/src/components/message/MessageItem.vue:93-174,216-225`
- Modify: `qim-client/tests/unit/components/MessageItem.test.ts`

**Interfaces:**
- Produces: `MergedForwardPayload`, `createMergedForwardPayload(messages: Message[]): MergedForwardPayload`, and `parseMergedForwardPayload(content: string): MergedForwardPayload | null`.
- Consumes: `Message.id/type/content/sender.name/timestamp`.

- [ ] **Step 1: Write the failing utility test**

```ts
it('keeps selected messages in list order and rejects invalid payloads', () => {
  const payload = createMergedForwardPayload([
    { id: '2', type: 'image', content: '/a.png', sender: { name: '乙' }, timestamp: 2 },
    { id: '1', type: 'text', content: '你好', sender: { name: '甲' }, timestamp: 1 },
  ] as any)
  expect(payload.messages.map(item => item.id)).toEqual(['2', '1'])
  expect(parseMergedForwardPayload('{invalid')).toBeNull()
})
```

- [ ] **Step 2: Verify it is red**

Run: `cd qim-client && npm exec vitest run tests/unit/utils/mergedForward.test.ts`

Expected: FAIL because `@/utils/mergedForward` does not exist.

- [ ] **Step 3: Implement the protocol**

```ts
export type MergedForwardItem = {
  id: string; type: string; content: string; senderName: string; timestamp: number
}
export type MergedForwardPayload = {
  version: 1; title: '聊天记录'; messages: MergedForwardItem[]
}
export const createMergedForwardPayload = (messages: Message[]): MergedForwardPayload => ({
  version: 1, title: '聊天记录',
  messages: messages.map(message => ({
    id: String(message.id), type: message.type, content: message.content,
    senderName: message.sender?.name || '未知用户', timestamp: Number(message.timestamp),
  })),
})
export const parseMergedForwardPayload = (content: string): MergedForwardPayload | null => {
  try {
    const value = JSON.parse(content)
    return value?.version === 1 && value?.title === '聊天记录' && Array.isArray(value?.messages) ? value : null
  } catch { return null }
}
```

Add `'merged_forward'` to the `Message.type` union. Render a card headed `聊天记录（{{ payload?.messages.length ?? 0 }}条）`; toggle its `expanded` ref from the button `data-testid="merged-forward-toggle"`. Display text content directly, `[图片]` for images, the parsed file name for files, and `聊天记录无法加载` when parsing returns null. In MessageItem add:

```vue
<MergedForwardMessage
  v-else-if="message.type === 'merged_forward'"
  :content="message.content"
  :is-self="isSelf"
/>
```

- [ ] **Step 4: Verify focused tests are green**

Run: `cd qim-client && npm exec vitest run tests/unit/utils/mergedForward.test.ts tests/unit/components/MessageItem.test.ts`

Expected: PASS with 0 failures.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/utils/mergedForward.ts qim-client/src/components/message/MergedForwardMessage.vue qim-client/src/types/index.ts qim-client/src/components/message/MessageItem.vue qim-client/tests/unit/utils/mergedForward.test.ts qim-client/tests/unit/components/MessageItem.test.ts
git commit -m "feat: render merged forward messages"
```

### Task 2: 将消息数组发送为合并转发

**Files:**
- Modify: `qim-client/src/composables/useShareLogic.ts:1-219`
- Modify: `qim-client/tests/unit/composables/useShareLogic.test.ts`

**Interfaces:**
- Consumes: `shareData.value` as a single message or `Message[]`.
- Produces: one `{ type: 'merged_forward', content: JSON.stringify(payload) }` per selected destination.

- [ ] **Step 1: Write the failing share test**

```ts
it('sends one merged forward message for multiple source messages', async () => {
  ;(request as any)
    .mockResolvedValueOnce({ code: 0, data: { id: 10 } })
    .mockResolvedValueOnce({ code: 0, data: { id: 99 } })
  const logic = useShareLogic(ref([
    { id: '1', type: 'text', content: '第一条', sender: { name: '甲' }, timestamp: 1 },
    { id: '2', type: 'text', content: '第二条', sender: { name: '乙' }, timestamp: 2 },
  ]), ref('message'), ref([]), ref([]), ref([]), ref(null), vi.fn(), vi.fn(), vi.fn())
  await logic.handleShareConfirm({ users: ['2'], groups: [] })
  expect(request).toHaveBeenLastCalledWith('/api/v1/conversations/10/messages', expect.objectContaining({
    body: expect.stringContaining('"type":"merged_forward"'),
  }))
})
```

- [ ] **Step 2: Verify it is red**

Run: `cd qim-client && npm exec vitest run tests/unit/composables/useShareLogic.test.ts`

Expected: FAIL because the current code dereferences `shareData.value.type`.

- [ ] **Step 3: Normalize data and send a single card**

```ts
const forwardedMessages = Array.isArray(shareData.value) ? shareData.value : [shareData.value]
const messageData = forwardedMessages.length > 1
  ? { type: 'merged_forward', content: JSON.stringify(createMergedForwardPayload(forwardedMessages)) }
  : buildForwardedMessage(forwardedMessages[0])
```

Extract the present single-message type branches to `buildForwardedMessage(message)`. Use the same `messageData` and local store shape in the user and group loops so a card appears immediately for either destination.

- [ ] **Step 4: Verify tests are green**

Run: `cd qim-client && npm exec vitest run tests/unit/composables/useShareLogic.test.ts tests/unit/composables/useShareLogicFile.test.ts`

Expected: PASS with 0 failures.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/composables/useShareLogic.ts qim-client/tests/unit/composables/useShareLogic.test.ts
git commit -m "feat: send merged message forwards"
```

### Task 3: 选择消息并接入现有分享弹窗

**Files:**
- Modify: `qim-client/src/components/chat/MessageContextMenu.vue:38-63,87-107,166-175`
- Modify: `qim-client/src/components/chat/OverlayManager.vue:20-31,135-150`
- Modify: `qim-client/src/components/chat/ChatWindow.vue:31-57,144-178,585-602,1726-1744,2469-2510`
- Modify: `qim-client/src/components/chat/ChatBody.vue:1-25,48-83`
- Modify: `qim-client/src/components/chat/MessageListView.vue:24-51,69-85`
- Create: `qim-client/tests/unit/components/ChatWindowMergedForward.test.ts`

**Interfaces:**
- Produces: `window.dispatchEvent(new CustomEvent('forwardMessage', { detail: { messages } }))` when two or more messages are selected.
- Consumes: the existing single-message `forwardMessage` event and Main's `openShareModal('message', data)`.

- [ ] **Step 1: Write the failing interaction test**

```ts
it('emits selected messages in list order when forwarding', async () => {
  const wrapper = mount(ChatWindow, { props: makeChatWindowProps({
    messages: [makeMessage('1', '第一条'), makeMessage('2', '第二条')],
  }) })
  const received: any[] = []
  window.addEventListener('forwardMessage', event => received.push((event as CustomEvent).detail), { once: true })
  await (wrapper.vm as any).startMessageSelection()
  await (wrapper.vm as any).toggleMessageSelection('2')
  await (wrapper.vm as any).toggleMessageSelection('1')
  await (wrapper.vm as any).forwardSelectedMessages()
  expect(received[0].messages.map((message: any) => message.id)).toEqual(['1', '2'])
})
```

- [ ] **Step 2: Verify it is red**

Run: `cd qim-client && npm exec vitest run tests/unit/components/ChatWindowMergedForward.test.ts`

Expected: FAIL because the selection methods and multi-message payload do not exist.

- [ ] **Step 3: Add selection state and event plumbing**

```ts
const isMessageSelectionMode = ref(false)
const selectedMessageIds = ref(new Set<string>())
const selectedMessages = computed(() =>
  props.messages.filter(message => selectedMessageIds.value.has(String(message.id)))
)
const toggleMessageSelection = (id: string) => {
  const ids = new Set(selectedMessageIds.value)
  ids.has(id) ? ids.delete(id) : ids.add(id)
  selectedMessageIds.value = ids
}
const forwardSelectedMessages = () => {
  if (selectedMessages.value.length < 2) return
  window.dispatchEvent(new CustomEvent('forwardMessage', { detail: { messages: selectedMessages.value } }))
  isMessageSelectionMode.value = false
  selectedMessageIds.value = new Set()
}
```

Add a `select-messages` event next to the current context-menu forward event and forward it through OverlayManager. The context-menu action initializes selection with the clicked message. Pass `selection-mode` and `selected-message-ids` through ChatBody and MessageListView; render a checkbox for each non-recalled MessageItem with `data-testid="message-select-<id>"`. Add an action bar showing `已选择 N 条`, `取消`, and a disabled-until-two `合并转发` button. In Main's event listener retain `detail.message` behavior and add `detail.messages` to call `openShareModal('message', detail.messages)`.

- [ ] **Step 4: Verify interaction and regression tests are green**

Run: `cd qim-client && npm exec vitest run tests/unit/components/ChatWindowMergedForward.test.ts tests/unit/components/MessageListView.test.ts tests/unit/composables/useShareLogic.test.ts`

Expected: PASS with 0 failures.

- [ ] **Step 5: Run final validation and commit**

Run: `cd qim-client && npm run test:unit && npm run typecheck && npm run build`

Expected: every command exits 0.

```bash
git add qim-client/src/components/chat/ChatWindow.vue qim-client/src/components/chat/ChatBody.vue qim-client/src/components/chat/MessageListView.vue qim-client/src/components/chat/OverlayManager.vue qim-client/src/components/chat/MessageContextMenu.vue qim-client/src/views/Main.vue qim-client/tests/unit/components/ChatWindowMergedForward.test.ts
git commit -m "feat: select multiple messages for forwarding"
```

