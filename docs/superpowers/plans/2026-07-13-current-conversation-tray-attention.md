# Current Conversation Tray Attention Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Flash native attention for an eligible message in the selected conversation whenever the QIM window is not actively being viewed.

**Architecture:** Electron's main process exposes one read-only IPC query for native window activity. The renderer combines that result with conversation identity and streaming state through a small pure decision function, while leaving notification, do-not-disturb, mention override, mute, sound, and desktop-notification rules unchanged.

**Tech Stack:** Electron 42 IPC, Vue 3, TypeScript, Vitest 4

## Global Constraints

- Do not flash for a non-streaming message in the selected conversation only when the main window is focused.
- Flash for a non-streaming message in the selected conversation when the main window is unfocused, minimized, or hidden.
- Preserve existing notification, do-not-disturb, mention override, mute, sound, and desktop-notification behavior.
- Treat a failed native window-state query as inactive so an eligible alert is not lost.
- Do not modify unread-count behavior.

---

### Task 1: Expose native main-window activity

**Files:**
- Modify: `qim-client/electron/main.js:960-980`
- Modify: `qim-client/electron/preload.cjs:135-143`
- Modify: `qim-client/src/types/electron.d.ts:31-35`
- Modify: `qim-client/tests/unit/tray-attention.test.ts`

**Interfaces:**
- Consumes: Electron `mainWindow.isDestroyed()`, `isVisible()`, `isMinimized()`, and `isFocused()`.
- Produces: `window.electron.windowState.isActive(): Promise<boolean>` backed by `is-main-window-active`.

- [ ] **Step 1: Write failing bridge tests**

Extend `qim-client/tests/unit/tray-attention.test.ts`:

```ts
const preloadProcess = readFileSync(resolve(__dirname, '../../electron/preload.cjs'), 'utf8')
const electronTypes = readFileSync(resolve(__dirname, '../../src/types/electron.d.ts'), 'utf8')

it('reports the main window as active only when it is viewable and focused', () => {
  expect(mainProcess).toContain("ipcMain.handle('is-main-window-active'")
  expect(mainProcess).toMatch(/!mainWindow\.isDestroyed\(\)[\s\S]*?mainWindow\.isVisible\(\)[\s\S]*?!mainWindow\.isMinimized\(\)[\s\S]*?mainWindow\.isFocused\(\)/)
})

it('exposes main-window activity through the preload bridge and types', () => {
  expect(preloadProcess).toContain("ipcRenderer.invoke('is-main-window-active')")
  expect(electronTypes).toContain('windowState: {')
  expect(electronTypes).toContain('isActive: () => Promise<boolean>')
})
```

- [ ] **Step 2: Run the bridge tests and verify red**

Run: `cd qim-client && npx vitest run tests/unit/tray-attention.test.ts`

Expected: the two new tests fail because the IPC handler and preload API do not exist.

- [ ] **Step 3: Implement the native query and bridge**

Add inside `registerIPC()` in `qim-client/electron/main.js`:

```js
ipcMain.handle('is-main-window-active', () => {
  return Boolean(
    mainWindow &&
    !mainWindow.isDestroyed() &&
    mainWindow.isVisible() &&
    !mainWindow.isMinimized() &&
    mainWindow.isFocused()
  )
})
```

Expose it in `qim-client/electron/preload.cjs`:

```js
windowState: {
  isActive: () => ipcRenderer.invoke('is-main-window-active')
},
```

Add to `ElectronAPI` in `qim-client/src/types/electron.d.ts`:

```ts
windowState: {
  isActive: () => Promise<boolean>
}
```

- [ ] **Step 4: Run the bridge tests and verify green**

Run: `cd qim-client && npx vitest run tests/unit/tray-attention.test.ts`

Expected: all tray-attention tests pass.

- [ ] **Step 5: Commit the native bridge**

```bash
git add qim-client/electron/main.js qim-client/electron/preload.cjs qim-client/src/types/electron.d.ts qim-client/tests/unit/tray-attention.test.ts
git commit -m "feat: expose main window attention state"
```

---

### Task 2: Use window activity in the new-message decision

**Files:**
- Create: `qim-client/src/utils/messageAttention.ts`
- Create: `qim-client/tests/unit/utils/messageAttention.test.ts`
- Modify: `qim-client/src/views/Main.vue:1818-1873`

**Interfaces:**
- Consumes: `window.electron.windowState.isActive(): Promise<boolean>`.
- Produces: `shouldRequestMessageAttention(input: MessageAttentionInput): boolean`.

- [ ] **Step 1: Write the failing decision tests**

Create `qim-client/tests/unit/utils/messageAttention.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { shouldRequestMessageAttention } from '../../../src/utils/messageAttention'

describe('shouldRequestMessageAttention', () => {
  it.each([
    { name: 'focused selected conversation', current: true, streaming: false, active: true, expected: false },
    { name: 'unfocused selected conversation', current: true, streaming: false, active: false, expected: true },
    { name: 'focused other conversation', current: false, streaming: false, active: true, expected: true },
    { name: 'unfocused other conversation', current: false, streaming: false, active: false, expected: true },
    { name: 'streaming message', current: false, streaming: true, active: false, expected: false },
  ])('$name => $expected', ({ current, streaming, active, expected }) => {
    expect(shouldRequestMessageAttention({
      isCurrentConversation: current,
      isStreaming: streaming,
      isWindowActive: active,
    })).toBe(expected)
  })
})
```

- [ ] **Step 2: Run the decision tests and verify red**

Run: `cd qim-client && npx vitest run tests/unit/utils/messageAttention.test.ts`

Expected: FAIL because `src/utils/messageAttention.ts` does not exist.

- [ ] **Step 3: Implement the pure function**

Create `qim-client/src/utils/messageAttention.ts`:

```ts
export interface MessageAttentionInput {
  isCurrentConversation: boolean
  isStreaming: boolean
  isWindowActive: boolean
}

export function shouldRequestMessageAttention({
  isCurrentConversation,
  isStreaming,
  isWindowActive,
}: MessageAttentionInput): boolean {
  if (isStreaming) return false
  return !isCurrentConversation || !isWindowActive
}
```

- [ ] **Step 4: Run the decision tests and verify green**

Run: `cd qim-client && npx vitest run tests/unit/utils/messageAttention.test.ts`

Expected: all parameterized tests pass.

- [ ] **Step 5: Integrate the decision into `handleNewMessage`**

Import the helper in `qim-client/src/views/Main.vue`, query native activity with an inactive fallback, and replace the outer `!isCurrentConv && !newMessage.isStreaming` condition:

```ts
import { shouldRequestMessageAttention } from '../utils/messageAttention'

let isWindowActive = false
if (window.electron?.windowState?.isActive) {
  try {
    isWindowActive = await window.electron.windowState.isActive()
  } catch (error) {
    logger.warn('读取主窗口状态失败，按非活动窗口处理:', error)
  }
}

const shouldRequestAttention = shouldRequestMessageAttention({
  isCurrentConversation: isCurrentConv,
  isStreaming: Boolean(newMessage.isStreaming),
  isWindowActive,
})

if (shouldRequestAttention) {
  // Keep the existing alert-settings block here unchanged.
}
```

Do not move conversation loading, `chatStore.receiveMessage`, scrolling, or unread updates into the alert block.

- [ ] **Step 6: Run focused tests and typecheck**

```bash
cd qim-client && npx vitest run tests/unit/tray-attention.test.ts tests/unit/utils/messageAttention.test.ts
cd qim-client && npm run typecheck
```

Expected: both test files pass and typecheck exits successfully. Record unrelated pre-existing typecheck failures separately.

- [ ] **Step 7: Commit renderer behavior**

```bash
git add qim-client/src/utils/messageAttention.ts qim-client/tests/unit/utils/messageAttention.test.ts qim-client/src/views/Main.vue
git commit -m "fix: flash for current chat while window is inactive"
```

---

### Task 3: Final verification

**Files:**
- Verify only; no planned source changes.

**Interfaces:**
- Consumes: Tasks 1 and 2.
- Produces: verification evidence for the complete fix.

- [ ] **Step 1: Run the complete focused suite**

Run: `cd qim-client && npx vitest run tests/unit/tray-attention.test.ts tests/unit/utils/messageAttention.test.ts tests/unit/stores/chat.test.ts`

Expected: all selected test files pass.

- [ ] **Step 2: Check whitespace and accidental edits**

Run: `git diff --check HEAD~2..HEAD` and `git status --short`.

Expected: no whitespace errors; the user's existing `classic-emoji.ts` and `emoji.ts` modifications remain untouched.

- [ ] **Step 3: Perform the Electron smoke test**

Run `cd qim-client && npm run electron:dev`, then verify:

1. Selected conversation plus focused QIM: no flash.
2. Selected conversation plus another focused application: flash.
3. Refocus QIM: flashing stops.
4. Selected conversation plus hidden or minimized QIM: flash.
5. Other conversation plus focused QIM: flash.

Expected: all five behaviors pass, and mute or do-not-disturb still suppresses alerts.
