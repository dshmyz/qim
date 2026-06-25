# Textarea @ 提及补全 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让群聊与讨论组的 textarea 直接驱动 `@` 成员筛选、键盘选择和完整 token 替换。

**Architecture:** `mentions.ts` 提供纯函数解析 textarea 当前光标所在的 token。`ChatWindow.vue` 持有 token 范围、筛选词和 mention spans；`MessageInput.vue` 只展示候选与处理来自 textarea 的列表键盘事件。

**Tech Stack:** Vue 3 Composition API、TypeScript、Vitest、happy-dom。

---

## File structure

- Modify: `qim-client/src/utils/mentions.ts` — 提取有效的 `@token`。
- Modify: `qim-client/tests/unit/utils/mentions.test.ts` — 纯函数回归测试。
- Modify: `qim-client/src/components/chat/ChatWindow.vue` — 输入、token 范围、替换和 span。
- Modify: `qim-client/src/components/chat/ChatInputArea.vue` — 将查询 prop 传给输入组件。
- Modify: `qim-client/src/components/chat/MessageInput.vue` — 候选展示和 textarea 键盘路由。
- Create: `qim-client/tests/unit/components/MessageInput.test.ts` — @ 面板键盘交互测试。

### Task 1: Parse the active mention token

**Files:**

- Modify: `qim-client/src/utils/mentions.ts`
- Test: `qim-client/tests/unit/utils/mentions.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { findActiveMentionToken } from '@/utils/mentions'

it('extracts a valid @ query and full replacement range', () => {
  expect(findActiveMentionToken('请 @ali 看看', 6)).toEqual({ start: 2, end: 6, query: 'ali' })
})

it('rejects @ in email and URL text', () => {
  expect(findActiveMentionToken('mail a@b.com', 8)).toBeNull()
  expect(findActiveMentionToken('go get host/@v1', 17)).toBeNull()
})
```

- [ ] **Step 2: Run it and confirm RED**

Run: `npm test -- --run tests/unit/utils/mentions.test.ts`

Expected: FAIL because `findActiveMentionToken` does not exist.

- [ ] **Step 3: Add the minimal parser**

```ts
export interface ActiveMentionToken {
  start: number
  end: number
  query: string
}

export function findActiveMentionToken(text: string, cursor: number): ActiveMentionToken | null {
  if (cursor < 0 || cursor > text.length) return null
  let start = cursor - 1
  while (start >= 0 && !/\s/.test(text[start])) {
    if (text[start] === '@') break
    start--
  }
  if (start < 0 || text[start] !== '@' || (start > 0 && !/\s/.test(text[start - 1]))) return null
  let end = cursor
  while (end < text.length && !/\s/.test(text[end])) end++
  return { start, end, query: text.slice(start + 1, cursor) }
}
```

- [ ] **Step 4: Run it and confirm GREEN**

Run: `npm test -- --run tests/unit/utils/mentions.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/utils/mentions.ts qim-client/tests/unit/utils/mentions.test.ts
git commit -m "feat: parse active mention token"
```

### Task 2: Drive the candidates and replacement from textarea state

**Files:**

- Modify: `qim-client/src/components/chat/ChatWindow.vue:543-809`
- Modify: `qim-client/src/components/chat/ChatInputArea.vue:3-44`
- Test: `qim-client/tests/unit/components/MessageInput.test.ts`

- [ ] **Step 1: Write the failing replacement test**

Mount the input flow, set the model to `@ali`, select Alice, and assert:

```ts
expect(emitted['update:inputMessage']).toContainEqual(['@Alice '])
expect(emitted['update:inputMessage']).not.toContainEqual(['@Alice ali'])
```

- [ ] **Step 2: Run it and confirm RED**

Run: `npm test -- --run tests/unit/components/MessageInput.test.ts`

Expected: FAIL because textarea text does not supply the filter and selection replaces only `@`.

- [ ] **Step 3: Implement unified token state**

Import `findActiveMentionToken`; add `pendingAtEnd` and `atMembersQuery`; update on every textarea input:

```ts
const updateActiveMention = (text: string, cursor: number) => {
  const token = findActiveMentionToken(text, cursor)
  if (!token) {
    showAtMembersPanel.value = false
    pendingAtPosition.value = -1
    pendingAtEnd.value = -1
    atMembersQuery.value = ''
    return
  }
  pendingAtPosition.value = token.start
  pendingAtEnd.value = token.end
  atMembersQuery.value = token.query
  showAtMembersPanel.value = true
}
```

Pass `atMembersQuery` through `ChatInputArea`. Selection must use `value.slice(0, start) + insertText + value.slice(end)` and then clear both positions.

- [ ] **Step 4: Run it and confirm GREEN**

Run: `npm test -- --run tests/unit/components/MessageInput.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/components/chat/ChatWindow.vue qim-client/src/components/chat/ChatInputArea.vue qim-client/tests/unit/components/MessageInput.test.ts
git commit -m "feat: drive mention suggestions from textarea"
```

### Task 3: Keep textarea focus for candidate keyboard navigation

**Files:**

- Modify: `qim-client/src/components/chat/MessageInput.vue`
- Test: `qim-client/tests/unit/components/MessageInput.test.ts`

- [ ] **Step 1: Write the failing keyboard test**

```ts
await textarea.trigger('keydown', { key: 'ArrowDown' })
await textarea.trigger('keydown', { key: 'Enter' })
expect(wrapper.emitted('select-at-member')).toEqual([[members[0]]])
expect(wrapper.emitted('handle-keydown')).toBeUndefined()
```

Also assert Escape emits `close-at-members-panel`, and Enter with a closed panel emits normal `handle-keydown`.

- [ ] **Step 2: Run it and confirm RED**

Run: `npm test -- --run tests/unit/components/MessageInput.test.ts`

Expected: FAIL because Enter reaches the regular message-send handler.

- [ ] **Step 3: Implement a single query prop and textarea key router**

Remove the panel search `<input>`, its ref, local `atMembersSearchQuery`, and auto-focus watcher. Add an `atMembersQuery: string` prop. Use it for `filteredAtMembers`. Route textarea keys as follows:

```ts
const handleTextareaKeydown = (event: KeyboardEvent) => {
  if (props.showAtMembersPanel && ['ArrowDown', 'ArrowUp', 'Enter', 'Escape'].includes(event.key)) {
    handleAtMembersKeyDown(event)
    return
  }
  emit('handle-keydown', event)
}
```

Bind the textarea with `@keydown="handleTextareaKeydown"`. Reset `atMemberActiveIndex` to `-1` when `atMembersQuery` changes.

- [ ] **Step 4: Run focused tests and confirm GREEN**

Run: `npm test -- --run tests/unit/components/MessageInput.test.ts tests/unit/utils/mentions.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/components/chat/MessageInput.vue qim-client/tests/unit/components/MessageInput.test.ts
git commit -m "feat: navigate mention suggestions from textarea"
```

### Task 4: Verify the client change

**Files:** none expected.

- [ ] **Step 1: Run focused regression tests**

Run: `npm test -- --run tests/unit/utils/mentions.test.ts tests/unit/components/MessageInput.test.ts`

Expected: PASS with zero failures.

- [ ] **Step 2: Run type checking**

Run: `npm run type-check`

Expected: no new diagnostics from `MessageInput.vue`, `ChatInputArea.vue`, `ChatWindow.vue`, or `mentions.ts`; separately record unrelated repository diagnostics if present.

- [ ] **Step 3: Check the diff**

Run: `git diff --check && git diff -- qim-client/src/utils/mentions.ts qim-client/src/components/chat/ChatWindow.vue qim-client/src/components/chat/ChatInputArea.vue qim-client/src/components/chat/MessageInput.vue qim-client/tests/unit`

Expected: no whitespace errors and test-covered behavior for each design requirement.
