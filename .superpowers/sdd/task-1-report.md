# Task 1 report: snapshot protocol and card

## Changes

- Added `MergedForwardPayload` and `MergedForwardItem`, plus payload creation and parsing helpers.
- Added the `merged_forward` message type and routed it through `MessageItem`.
- Added an expandable merged-forward card that shows the record count, text, image markers, file names, and an invalid-payload fallback.
- Added utility and message-item unit coverage.

## TDD evidence

### RED

Command:

```bash
cd qim-client && npm exec vitest run tests/unit/utils/mergedForward.test.ts
```

Result: failed as expected because `@/utils/mergedForward` did not exist (`Failed to resolve import`).

The message-item routing test was also run before implementation and failed as expected because no `.merged-forward-stub` was rendered.

### GREEN

Command:

```bash
cd qim-client && npm exec vitest run tests/unit/utils/mergedForward.test.ts tests/unit/components/MessageItem.test.ts
```

Result: passed: 2 test files, 6 tests.

## Verification

- Re-ran the focused GREEN command after self-review: 2 test files and 6 tests passed.
- Ran `npm exec vitest run` once. It failed outside this task in `FileManagementPreview.test.ts` (8 failures from missing active Pinia) and reported 21 existing `ScreenShareSimple` media-stream environment errors.
- Ran `npm exec vue-tsc --noEmit`; it reports numerous existing project-wide type errors unrelated to the new merged-forward files.
- `git diff --check` passed.

## Files

- `qim-client/src/utils/mergedForward.ts`
- `qim-client/src/components/message/MergedForwardMessage.vue`
- `qim-client/src/types/index.ts`
- `qim-client/src/components/message/MessageItem.vue`
- `qim-client/tests/unit/utils/mergedForward.test.ts`
- `qim-client/tests/unit/components/MessageItem.test.ts`

## Concerns

The focused task tests pass. The full frontend suite and project typecheck are currently blocked by unrelated pre-existing failures; no unrelated files were changed for this task.

## Review follow-up: malformed snapshot structure

### Changes

- `parseMergedForwardPayload` now validates every message item before returning a payload: `id`, `type`, `content`, and `senderName` must be strings, and `timestamp` must be a finite number.
- A malformed item such as `messages: [null]` is rejected, so `MergedForwardMessage` takes its existing `聊天记录无法加载` fallback instead of allowing expansion to dereference missing fields.
- Added a parser regression test and a card fallback test covering `messages: [null]`.

### TDD evidence

#### RED

Command:

```bash
cd qim-client && npm exec vitest run tests/unit/utils/mergedForward.test.ts tests/unit/components/MessageItem.test.ts
```

Output: failed as expected with 2 failed tests and 6 passed tests. The parser returned the malformed object instead of `null`; the card rendered `聊天记录（1条）展开` instead of `聊天记录无法加载`.

#### GREEN

Command:

```bash
cd qim-client && npm exec vitest run tests/unit/utils/mergedForward.test.ts tests/unit/components/MessageItem.test.ts
```

Output: passed: 2 test files, 8 tests, 0 failures.

### Self-review

Command:

```bash
git diff --check -- qim-client/src/utils/mergedForward.ts qim-client/tests/unit/utils/mergedForward.test.ts qim-client/tests/unit/components/MessageItem.test.ts .superpowers/sdd/task-1-report.md
```

Output: passed with no whitespace errors. The review confirmed the guard rejects null, primitive, missing-field, wrong-type, and non-finite-timestamp entries without changing the valid snapshot path.
