# Current Conversation Tray Attention Design

## Problem

The client suppresses message attention whenever a message belongs to the selected conversation. The selected conversation remains unchanged when the Electron window is hidden, minimized, or loses focus, so messages for that conversation do not flash the tray even though the user is no longer looking at QIM.

## Desired Behavior

- Do not flash for a non-streaming message in the selected conversation only when the main window is focused.
- Flash for a non-streaming message in the selected conversation when the main window is unfocused, minimized, or hidden.
- Continue flashing for eligible messages in other conversations.
- Preserve the existing global notification, do-not-disturb, mention override, and conversation mute rules.
- Stop external attention when the main window receives focus.

## Design

Electron's main process remains the source of truth for native window state. Add an IPC handler that reports whether the main window exists, is visible, is not minimized, and is focused. Expose it through the preload bridge as an asynchronous window-attention query.

Before deciding whether to alert for a new message, the renderer queries that state. A message needs attention when it is not in the selected conversation, or when the user is not actively viewing the selected conversation. The existing alert settings are then applied without change.

If the window-state query fails, treat the window as not actively viewed. Missing an alert is more harmful than an extra alert while native state is unavailable.

## Test Coverage

Add focused tests for the alert decision and Electron bridge:

- Focused window plus selected conversation does not flash.
- Unfocused window plus selected conversation flashes.
- Hidden or minimized window plus selected conversation flashes.
- Focused window plus another conversation flashes.
- Streaming, muted, globally disabled, and do-not-disturb messages remain suppressed according to existing rules.
- The main process reports inactive for a missing, destroyed, hidden, minimized, or unfocused window.

## Scope

This change only corrects tray and taskbar attention eligibility. It does not change unread counts, desktop-notification permissions, notification content, or sound settings.
