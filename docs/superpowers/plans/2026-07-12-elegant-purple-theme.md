# Elegant Purple Theme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace QIM's saturated elegant-purple palette with the approved cool, low-saturation Moonlight Lavender palette across global tokens, component overrides, and theme previews.

**Architecture:** Keep the existing `data-theme="elegant-purple"` contract and CSS-variable architecture. Define the canonical palette in `themes.css`, make component-specific rules consume those variables, and protect the palette with a focused source-level Vitest regression test.

**Tech Stack:** Vue 3, TypeScript, CSS custom properties, Vitest 4, Vite 5

## Global Constraints

- Core palette: primary `#75629A`, accent `#8B78B4`, deep purple `#665486`, pale purple `#A497C6`, content `#F8F7FB`, list `#FCFBFE`, hover `#EFEBF6`, border `#E5E0EF`, primary text `#342E43`, secondary text `#746D81`.
- Preserve the `elegant-purple` theme id and existing component/layout behavior.
- Do not modify other themes, typography, layout, or interactions.
- Main and secondary text on their normal backgrounds must meet WCAG AA contrast for regular text.
- Preserve unrelated working-tree changes.

---

### Task 1: Lock the approved palette with a regression test

**Files:**
- Create: `qim-client/tests/unit/elegant-purple-theme.test.ts`
- Test: `qim-client/tests/unit/elegant-purple-theme.test.ts`

**Interfaces:**
- Consumes: source files containing the existing `elegant-purple` CSS contract.
- Produces: a regression test that asserts the canonical tokens, component variable usage, and preview colors.

- [ ] **Step 1: Write the failing source-level test**

```ts
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const root = resolve(__dirname, '../..')
const read = (path: string) => readFileSync(resolve(root, path), 'utf8')

describe('elegant purple theme', () => {
  it('defines the approved Moonlight Lavender tokens', () => {
    const css = read('src/assets/styles/themes.css')
    const block = css.match(/\[data-theme="elegant-purple"\]\s*\{([\s\S]*?)\n\}/)?.[1]

    expect(block).toBeTruthy()
    expect(block).toContain('--primary-color: #75629a;')
    expect(block).toContain('--accent-color: #8b78b4;')
    expect(block).toContain('--active-color: #665486;')
    expect(block).toContain('--content-bg: #f8f7fb;')
    expect(block).toContain('--list-bg: #fcfbfe;')
    expect(block).toContain('--hover-color: #efebf6;')
    expect(block).toContain('--border-color: #e5e0ef;')
    expect(block).toContain('--text-color: #342e43;')
    expect(block).toContain('--text-secondary: #746d81;')
  })

  it('uses theme variables in elegant-purple component overrides', () => {
    expect(read('src/components/message/MessageItem.vue')).not.toContain('rgba(139, 92, 246')
    expect(read('src/components/chat/QuotedMessagePreview.vue')).toContain('border-left-color: var(--accent-color)')
    expect(read('src/components/modals/UserProfile.vue')).not.toContain('#7e22ce')
  })

  it('shows the approved palette in every elegant-purple preview', () => {
    for (const path of [
      'src/assets/styles/layout.css',
      'src/views/Main.css',
      'src/components/menus/MainContextMenus.vue',
      'src/components/settings/SettingsPanel.vue',
    ]) {
      expect(read(path)).toMatch(/\.elegant-purple-theme\s*\{[^}]*#75629a/is)
    }
  })
})
```

- [ ] **Step 2: Run the focused test and confirm it fails**

Run: `cd qim-client && npm run test:unit -- tests/unit/elegant-purple-theme.test.ts`

Expected: FAIL because the current theme still defines `#7e22ce`, `#8b5cf6`, and `#6b4c9a`.

- [ ] **Step 3: Commit the failing test**

```bash
git add qim-client/tests/unit/elegant-purple-theme.test.ts
git commit -m "test: lock elegant purple theme palette"
```

### Task 2: Replace the canonical theme and short-link tokens

**Files:**
- Modify: `qim-client/src/assets/styles/themes.css:307-357`
- Modify: `qim-client/src/assets/styles/themes.css:758-778`
- Test: `qim-client/tests/unit/elegant-purple-theme.test.ts`

**Interfaces:**
- Consumes: the existing `elegant-purple` CSS custom-property names.
- Produces: the approved values for all components that consume global theme variables.

- [ ] **Step 1: Replace the global token block**

Set the existing properties to the approved values and add the missing content variable:

```css
[data-theme="elegant-purple"] {
  --primary-color: #75629a;
  --primary-light: #efebf6;
  --secondary-color: #f8f7fb;
  --text-color: #342e43;
  --border-color: #e5e0ef;
  --hover-color: #efebf6;
  --active-color: #665486;
  --accent-color: #8b78b4;
  --text-secondary: #746d81;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #efebf6;
  --panel-bg: #ffffff;
  --list-bg: #fcfbfe;
  --card-bg: #ffffff;
  --header-panel-bg: #efebf6;
  --content-bg: #f8f7fb;
  --right-content-bg: #ffffff;
  --right-content-header-bg: #ffffff;
  --input-bg: #ffffff;
  --modal-bg: #ffffff;
  --btn-bg: #efebf6;
  --side-options-bg: linear-gradient(135deg, #a497c6 0%, #8b78b4 50%, #665486 100%);
}
```

Use cool-purple shadows: `rgba(64, 53, 91, 0.07)` for small and `rgba(64, 53, 91, 0.12)`/`rgba(64, 53, 91, 0.08)` for medium shadow layers.

- [ ] **Step 2: Align the title-bar gradient**

```css
[data-theme="elegant-purple"] .window-controls-left {
  background: linear-gradient(135deg, #a497c6 0%, #8b78b4 50%, #665486 100%);
}
```

- [ ] **Step 3: Align the short-link local variables**

Use `#f8f7fb`, `#ffffff`, `#e5e0ef`, `#efebf6`, `#75629a`, `#342e43`, `#746d81`, `#a497c6`, `#665486`, and `rgba(117, 98, 154, 0.18)` in the corresponding existing short-link properties.

- [ ] **Step 4: Run the focused test**

Run: `cd qim-client && npm run test:unit -- tests/unit/elegant-purple-theme.test.ts`

Expected: token assertions PASS; component and preview assertions remain FAIL until Tasks 3 and 4.

- [ ] **Step 5: Commit the canonical palette**

```bash
git add qim-client/src/assets/styles/themes.css
git commit -m "style: apply moonlight lavender theme tokens"
```

### Task 3: Remove saturated component overrides

**Files:**
- Modify: `qim-client/src/components/message/MessageItem.vue:583-599`
- Modify: `qim-client/src/components/chat/QuotedMessagePreview.vue:216-219`
- Modify: `qim-client/src/components/modals/UserProfile.vue:274-292`
- Review: `qim-client/src/components/chat/MessageStatus.vue:167-169`
- Review: `qim-client/src/components/ai/AIMessageBadge.vue:49-56`
- Test: `qim-client/tests/unit/elegant-purple-theme.test.ts`

**Interfaces:**
- Consumes: `--primary-color`, `--active-color`, `--accent-color`, `--text-color`, `--border-color`, and `--hover-color` from Task 2.
- Produces: theme-consistent message, quote, profile, status, and AI badge states without old hard-coded purple values.

- [ ] **Step 1: Use a clear primary message bubble**

```css
[data-theme="elegant-purple"] .message-item.self .message-bubble,
[data-theme="elegant-purple"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #ffffff;
  border: none;
}

[data-theme="elegant-purple"] .message-item.self .recalled-message {
  background: color-mix(in srgb, var(--primary-color), transparent 12%) !important;
  color: rgba(255, 255, 255, 0.88) !important;
}
```

- [ ] **Step 2: Replace quote and profile fallbacks**

Use `border-left-color: var(--accent-color);` for quoted messages. In `UserProfile.vue`, retain the existing variable references but replace fallbacks with `#e5e0ef`, `#342e43`, `#75629a`, and `#665486`.

- [ ] **Step 3: Review status and AI badge rules**

Confirm the read state still uses semantic `--success-color`. Keep the shared AI badge rule unchanged unless it prevents the badge from inheriting the new theme variables; do not broaden this task into redesigning other themes.

- [ ] **Step 4: Run the focused test**

Run: `cd qim-client && npm run test:unit -- tests/unit/elegant-purple-theme.test.ts`

Expected: component assertions PASS; preview assertions remain FAIL.

- [ ] **Step 5: Commit component alignment**

```bash
git add qim-client/src/components/message/MessageItem.vue qim-client/src/components/chat/QuotedMessagePreview.vue qim-client/src/components/modals/UserProfile.vue
git commit -m "style: align purple theme component states"
```

### Task 4: Align every theme preview swatch

**Files:**
- Modify: `qim-client/src/assets/styles/layout.css:208-211,263-266`
- Modify: `qim-client/src/views/Main.css:389-392`
- Modify: `qim-client/src/components/menus/MainContextMenus.vue:333`
- Modify: `qim-client/src/components/settings/SettingsPanel.vue:761`
- Test: `qim-client/tests/unit/elegant-purple-theme.test.ts`

**Interfaces:**
- Consumes: approved primary `#75629A` and deep purple `#665486`.
- Produces: consistent preview swatches in all theme selectors.

- [ ] **Step 1: Replace each preview rule**

```css
.elegant-purple-theme {
  background: #75629a;
  border: 1px solid #665486;
}
```

For one-line component preview rules that have no border today, set only `background: #75629a;` to preserve their current shape and spacing.

- [ ] **Step 2: Run the complete regression test**

Run: `cd qim-client && npm run test:unit -- tests/unit/elegant-purple-theme.test.ts`

Expected: all three tests PASS.

- [ ] **Step 3: Commit preview alignment**

```bash
git add qim-client/src/assets/styles/layout.css qim-client/src/views/Main.css qim-client/src/components/menus/MainContextMenus.vue qim-client/src/components/settings/SettingsPanel.vue
git commit -m "style: refresh elegant purple previews"
```

### Task 5: Verify accessibility, behavior, and build output

**Files:**
- Modify only if verification exposes a theme-scoped defect in files already listed above.
- Test: `qim-client/tests/unit/elegant-purple-theme.test.ts`

**Interfaces:**
- Consumes: completed theme implementation from Tasks 1-4.
- Produces: verification evidence that the palette is readable, test-protected, and buildable.

- [ ] **Step 1: Calculate required contrast ratios**

Run a small Node expression or browser contrast checker for:

```text
#342E43 on #F8F7FB: must be >= 4.5:1
#746D81 on #F8F7FB: must be >= 4.5:1
#FFFFFF on #75629A: must be >= 4.5:1
```

Expected: all three pairs meet WCAG AA. If a pair fails, darken only the foreground token while preserving the approved palette direction, then update the regression test and design-token documentation together.

- [ ] **Step 2: Search for obsolete purple values in theme-specific rules**

Run:

```bash
rg -n "#7e22ce|#8b5cf6|#7c3aed|#6b4c9a|#5b21b6|#6b21a8|139, 92, 246" qim-client/src
```

Expected: no matches belonging to `elegant-purple` rules; unrelated colors outside this theme are left unchanged.

- [ ] **Step 3: Run focused and full static verification**

```bash
cd qim-client
npm run test:unit -- tests/unit/elegant-purple-theme.test.ts
npm run typecheck
npm run build
```

Expected: focused test PASS, typecheck exits 0, production build exits 0.

- [ ] **Step 4: Visually inspect the running client**

Run `cd qim-client && npm run dev`, switch to “高雅紫”, and inspect the sidebar, conversation list, chat bubbles, quote preview, profile dialog, settings theme preview, and short-link page at desktop width.

Expected: the UI matches the Moonlight Lavender direction, no saturated legacy purple remains, focus/hover/selected states remain distinguishable, and no layout changes are visible.

- [ ] **Step 5: Commit verification-only corrections if needed**

```bash
git add <only-the-theme-files-corrected-during-verification>
git commit -m "fix: polish elegant purple theme contrast"
```

Skip this commit when verification requires no corrections.
