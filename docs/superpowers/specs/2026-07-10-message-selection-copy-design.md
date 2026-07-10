# Message Selection Copy Design

## Goal

Make the chat context menu copy the complete text selected at the moment the menu opens, including selections that span multiple messages.

## Design

The chat window owns the context-menu interaction state. When it opens the message context menu, it snapshots `window.getSelection()?.toString().trim()` into the existing menu state. The copy action receives that snapshot and writes it directly when non-empty; otherwise it copies the clicked message's decoded text.

The snapshot is never re-read after the menu opens. This removes dependence on browser selection changes caused by clicking the custom menu, and does not introduce module-level mutable state.

## Constraints

- Any non-empty selection is copied in full, regardless of which messages it spans.
- No selection copies the clicked message.
- The change does not alter clipboard permissions or image-copy behavior.

## Verification

Add a regression test that changes the live selection after opening the menu and verifies the originally snapshotted selection is copied. Retain coverage for the no-selection message-copy path.
