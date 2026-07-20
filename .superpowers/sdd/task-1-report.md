# Task 1 Report: Authorize Chat Attachment References by Group Message

## Status

Implemented; commit ID is recorded below after commit creation.

## Changes

- The group-reference handler now requires `message_id` and `file_id`, resolves the URL conversation to the internal group ID, and delegates to `ShareMessageAttachment`.
- `ShareMessageAttachment` authorizes group-file management, verifies the target group and a `file` message in that exact group conversation, parses its attachment JSON, and rejects a mismatched file ID as forbidden.
- The former exported arbitrary cross-space `ShareReference` operation was removed. The remaining private helper is reachable only after message authorization.
- The chat client retains both the selected message ID and parsed attachment ID. The group-files panel submits both as `message_id` and `file_id`.
- Added handler and service regressions for the valid target-group attachment, a message from another group, and a mismatched file ID. Added a focused panel test for the client request parameters.

## TDD Evidence

1. `go test ./handler -run TestGroupFileHandlerShareMessageAttachment -count=1 -v` failed before implementation: a message from another group received HTTP 200 rather than the required HTTP 403.
2. `go test ./service -run TestFileSpaceShareMessageAttachmentRequiresTargetGroupFileMessage -count=1 -v` failed before implementation because `ShareMessageAttachment` did not exist.
3. `npm test -- tests/unit/group-files-panel.test.ts` failed before the client bridge update: `shareReference` was called without the selected message ID.

## Verification

- `cd qim-server && go test ./handler ./service -run 'TestGroupFileHandlerShareMessageAttachment|TestFileSpace' -count=1` — passed (`handler` and `service`).
- `cd qim-server && go vet ./handler ./service` — passed.
- `cd qim-client && npm test -- tests/unit/group-files-panel.test.ts` — passed: 5 tests.
- `cd qim-client && npm run typecheck` — remains nonzero because of pre-existing errors across unrelated files; the output included existing errors in `src/api/avatar.ts`, `src/api/message.ts`, AI components, and many other components. It did not report this task's changed group-file API/panel code. npm also emitted existing unsupported-project-config warnings.
- A targeted ESLint run is unavailable: the repository has no locally installed/configured ESLint (`npx eslint` attempted to use ESLint 10 and reported that no `eslint.config.*` exists).

## Commit

`fix(groups): authorize chat file references by message` (this commit includes the report).

## Concerns

- The full client typecheck is not a clean repository baseline. Focused client behavior and all scoped Go verification pass.
