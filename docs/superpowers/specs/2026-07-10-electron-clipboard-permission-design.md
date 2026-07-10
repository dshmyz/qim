# Electron Clipboard Permission Design

## Goal

Restore right-click message copying in the desktop client without changing the existing renderer-side message formatting or menu event flow.

## Scope

- Allow Electron's `clipboard-sanitized-write` permission alongside the existing `media` permission.
- Add a regression test that prevents the clipboard write permission from being removed from the main-process allowlist.
- Do not grant clipboard read access, alter message content handling, or refactor calls to Electron IPC in this change.

## Design

`ensureMediaPermissions` already installs the single permission request handler for the default session. It will use an explicit allowlist containing `media` and `clipboard-sanitized-write`; every other permission remains denied. This preserves the current least-privilege behavior while permitting `navigator.clipboard.writeText()` initiated by the chat renderer.

The regression test will inspect the main-process source and assert that the allowlist includes the clipboard write permission and that unknown permissions remain denied. It uses the same source-inspection convention as existing Electron configuration tests, avoiding a real Electron window in unit tests.

## Error Handling

The renderer's existing error handling remains unchanged. Once the permission request succeeds, the existing success/failure UI behavior continues to apply. No clipboard read permission is introduced.

## Verification

1. The new focused unit test fails before the allowlist includes `clipboard-sanitized-write`.
2. It passes after the minimal main-process change.
3. The message action unit suite continues to pass.
4. Type checking validates the client source after the configuration update.
