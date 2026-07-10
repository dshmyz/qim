# Electron Clipboard Permission Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore desktop right-click message copying by permitting Electron's clipboard write permission.

**Architecture:** The default Electron session retains one explicit permission allowlist. Add `clipboard-sanitized-write` to the existing `media` entry so renderer `navigator.clipboard.writeText()` calls can complete, while all other permission requests remain denied. A source-level main-process unit test locks that contract down.

**Tech Stack:** Electron 42, JavaScript main process, Vitest.

## Global Constraints

- Allow only `media` and `clipboard-sanitized-write`; do not grant clipboard read access.
- Do not alter renderer-side message formatting or context-menu event wiring.
- Preserve denial of unknown permissions.

---

### Task 1: Permit clipboard writes in the Electron session

**Files:**
- Create: `qim-client/tests/unit/electron-permissions.test.ts`
- Modify: `qim-client/electron/main.js:1204-1211`

**Interfaces:**
- Consumes: Electron's `session.defaultSession.setPermissionRequestHandler` callback signature `(webContents, permission, callback)`.
- Produces: A permission allowlist that grants `media` and `clipboard-sanitized-write`, and denies every other permission.

- [ ] **Step 1: Write the failing test**

```ts
import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')

describe('Electron permission policy', () => {
  it('allows sanitized clipboard writes while leaving other permissions denied', () => {
    expect(mainProcess).toContain("['media', 'clipboard-sanitized-write'].includes(permission)")
    expect(mainProcess).toContain('callback(false)')
  })
})
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `npm exec vitest run tests/unit/electron-permissions.test.ts`

Expected: FAIL because `clipboard-sanitized-write` is absent from the permission allowlist.

- [ ] **Step 3: Implement the minimal allowlist change**

```js
session.defaultSession.setPermissionRequestHandler((_webContents, permission, callback) => {
  if (['media', 'clipboard-sanitized-write'].includes(permission)) {
    callback(true)
  } else {
    callback(false)
  }
})
```

- [ ] **Step 4: Run focused regression tests**

Run: `npm exec vitest run tests/unit/electron-permissions.test.ts tests/unit/composables/useMessageActions.test.ts`

Expected: PASS with 0 failed tests.

- [ ] **Step 5: Run static verification**

Run: `npm run typecheck && npm run lint -- --quiet`

Expected: both commands exit 0.

- [ ] **Step 6: Commit the implementation**

```bash
git add qim-client/electron/main.js qim-client/tests/unit/electron-permissions.test.ts
git commit -m "fix: allow desktop clipboard writes"
```
