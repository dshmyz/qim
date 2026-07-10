# Conversation Pagination Deduplication Design

## Goal

Prevent duplicate conversations in the recent-conversation list while preserving correct ordering as new messages arrive during pagination.

## Scope

- Deduplicate all client-side conversation list writes by normalized conversation ID.
- Replace offset pagination for `GET /api/v1/conversations` with an opaque cursor for incremental loading.
- Keep the first-page request backward compatible while the client migrates to cursor pagination.
- Do not merge two different conversation IDs merely because their names or members look alike.

## Client Design

The chat store owns a single `mergeConversations(existing, incoming)` module. It normalizes IDs with `String(id)`, retains one entry per ID, and replaces an existing entry with the newer incoming representation. `setConversations` and paginated loading use this module rather than raw array assignment or concatenation.

This is the seam for list integrity: all callers receive a deduplicated list without knowing whether the source was first load, refresh, pagination, WebSocket, or create-conversation response. It immediately removes duplicate-ID rows even if an old server returns overlapping pages.

## Server Design

The list endpoint orders records by `is_pinned DESC`, `COALESCE(last_message_at, created_at) DESC`, then `conversation_id DESC`. Its response returns `next_cursor` encoding the final row's three ordering fields. A cursor request uses lexicographic “strictly after” predicates rather than `OFFSET`, so inserting or updating conversations before the current page cannot replay a row on a later page.

For a transition period, requests without `cursor` retain the existing `page` and `page_size` parameters and response shape. Cursor-aware clients include `cursor` on subsequent loads and use `next_cursor` to determine whether more data exists. The server returns both `next_cursor` and `has_more`.

## Data Integrity

The existing unique index on `(conversation_id, user_id)` remains the membership invariant. A separate follow-up audit will identify actual duplicate single conversations (different conversation IDs for one user pair); it is not merged by the list deduplication logic.

## Verification

- Client tests prove overlapping pages collapse to one normalized ID and incoming data wins.
- Server tests prove stable ordering for tied timestamps and cursor paging does not repeat rows after an intervening activity update.
- Existing first-page API tests remain valid.
