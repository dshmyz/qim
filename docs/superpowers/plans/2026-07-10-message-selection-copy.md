# Message Selection Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Copy the full selection present when the message context menu opens, without relying on the browser selection after the menu is clicked.

**Architecture:** `ChatWindow` keeps one local selection snapshot for the currently open message menu. It passes that text explicitly to `copyMessage`; the action function uses a non-empty supplied text first and otherwise preserves its message-copy behavior. The prior module-global selection state is removed.

**Tech Stack:** Vue 3, TypeScript, Vitest.

## Global Constraints

- Any non-empty selection is copied in full, including selections across messages.
- No selection copies the clicked message.
- Do not change Electron clipboard permissions or image-copy behavior.

---

### Task 1: Snapshot selection in the menu owner

**Files:**
- Modify: `qim-client/src/components/chat/ChatWindow.vue:161,577,1686-1689,2448-2477`
- Modify: `qim-client/src/composables/useMessageActions.ts:28-35,297-314`
- Modify: `qim-client/src/components/chat/MessageListView.vue:33,63`
- Test: `qim-client/tests/unit/composables/useMessageActions.test.ts`

**Interfaces:**
- Consumes: `copyMessage(message: Message, selectedText?: string): Promise<void>`.
- Produces: A local `messageMenuSelection` snapshot that is passed to `copyMessage` when the context-menu copy event fires.

- [ ] **Step 1: Write the failing regression test**

```ts
it('uses the supplied menu-open selection after the live selection changes', async () => {
  const { copyMessage } = useMessageActions(ref('http://localhost:3000'), ref({ id: 1 }))
  mockSelectionText = ''

  await copyMessage(makeMessage({ content: '当前消息内容' }), '跨消息选中的全部内容')

  expect(mockWriteText).toHaveBeenCalledWith('跨消息选中的全部内容')
})
```

- [ ] **Step 2: Verify the test is red**

Run: `npm exec vitest run tests/unit/composables/useMessageActions.test.ts`

Expected: FAIL because `copyMessage` currently accepts only the message and ignores the passed selection.

- [ ] **Step 3: Implement the local snapshot**

```ts
const messageMenuSelection = ref('')

const showMessageContextMenu = (event: MouseEvent, message: Message) => {
  messageMenuSelection.value = window.getSelection?.().toString() ?? ''
  // existing positioning and selectedMessage setup
}

const closeMessageContextMenu = () => {
  showMessageContextMenuFlag.value = false
  selectedMessage.value = null
  messageMenuSelection.value = ''
  document.removeEventListener('click', closeMessageContextMenu)
}
```

```ts
const copyMessage = async (message: Message, selectedText?: string) => {
  if (selectedText) {
    await navigator.clipboard.writeText(selectedText)
    QMessage.success('已复制')
    return
  }
  // existing message copy behavior
}
```

Remove `recordSelectionBeforeContextMenu`, its module-global state, and the `MessageListView` right-mouse listener.

- [ ] **Step 4: Verify the focused tests are green**

Run: `npm exec vitest run tests/unit/composables/useMessageActions.test.ts tests/unit/electron-permissions.test.ts`

Expected: PASS with 0 failed tests.

- [ ] **Step 5: Commit the implementation**

```bash
git add qim-client/src/components/chat/ChatWindow.vue qim-client/src/components/chat/MessageListView.vue qim-client/src/composables/useMessageActions.ts qim-client/tests/unit/composables/useMessageActions.test.ts
git commit -m "fix: preserve selected text for message copy"
```
