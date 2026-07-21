# Product Name Build Metadata Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `APP_CONFIG.productName` use `qim-client/package.json`'s `build.productName` value.

**Architecture:** Vite and Vitest inject the same `__APP_PRODUCT_NAME__` build constant. `appConfig.ts` consumes it while its public getter API remains unchanged.

**Tech Stack:** Vite, Vitest, TypeScript, Vue.

## Global Constraints

- Use `build.productName` as the single renderer source of product display name.
- Preserve `getProductName(): string` and all existing call sites.

---

### Task 1: Inject and consume the Electron product name

**Files:**
- Create: `qim-client/tests/unit/appConfig.test.ts`
- Modify: `qim-client/vite.config.ts:10-16`
- Modify: `qim-client/vitest.config.ts:11-17`
- Modify: `qim-client/src/vite-env.d.ts:1-8`
- Modify: `qim-client/src/config/appConfig.ts:1-15`

**Interfaces:**
- Consumes: `pkg.build.productName: string` from `qim-client/package.json`.
- Produces: `getProductName(): string`, returning the injected `__APP_PRODUCT_NAME__` value.

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, it } from 'vitest'
import { APP_CONFIG, getProductName } from '../../src/config/appConfig'

describe('appConfig', () => {
  it('uses Electron build productName as the display name', () => {
    expect(APP_CONFIG.productName).toBe('青雀 QIM')
    expect(getProductName()).toBe('青雀 QIM')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test -- tests/unit/appConfig.test.ts`

Expected: FAIL because the current injected `__APP_NAME__` value is `qim`.

- [ ] **Step 3: Write the minimal implementation**

```ts
// vite.config.ts and vitest.config.ts
__APP_PRODUCT_NAME__: JSON.stringify(extra.productName),

// vite-env.d.ts
declare const __APP_PRODUCT_NAME__: string

// appConfig.ts
const productName = __APP_PRODUCT_NAME__
```

- [ ] **Step 4: Run focused test and type check**

Run: `npm test -- tests/unit/appConfig.test.ts && npm run typecheck`

Expected: both commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add qim-client/vite.config.ts qim-client/vitest.config.ts qim-client/src/vite-env.d.ts qim-client/src/config/appConfig.ts qim-client/tests/unit/appConfig.test.ts
git commit -m "fix: use build product name in renderer"
```
