# Conversation Pagination Deduplication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove duplicate conversation rows and prevent unstable pagination from replaying a conversation.

**Architecture:** The chat store is the single merge seam: all list writes normalize IDs and incoming rows replace existing rows. The endpoint adds cursor pagination using the exact SQL ordering keys, while preserving the existing first-page response for compatibility.

**Tech Stack:** Vue 3, Pinia, TypeScript, Go, Gin, GORM, Vitest, Go test.

## Global Constraints

- Deduplicate only equal normalized IDs, never names or members.
- Retain one row per ID and prefer incoming server data.
- Cursor ordering: pinned DESC, activity DESC, conversation ID DESC.
- Keep first-page page/page_size behavior compatible.

---

### Task 1: Normalize all client-side list merges

**Files:**
- Modify: `qim-client/src/stores/chat.ts:105-139`
- Modify: `qim-client/src/composables/useConversation.ts:96-110,220-230`
- Modify: `qim-client/src/composables/useMainConversationLogic.ts:18-61`
- Modify: `qim-client/src/views/Main.vue:1090-1110`
- Test: `qim-client/tests/unit/stores/chat.test.ts`

**Interfaces:**
- Produces `mergeConversations(incoming: Conversation[]): void` on the chat store.
- Produces `mergeConversations(incoming: Conversation[]): void` from `useConversation`.
- `setConversations` also applies normalized-ID deduplication.

- [ ] **Step 1: Write failing tests**

```ts
it('deduplicates numeric and string forms of an ID, using incoming data', () => {
  const store = useChatStore()
  store.setConversations([
    { id: 7 as any, name: '旧数据', type: 'single' } as Conversation,
    { id: '7', name: '新数据', type: 'single' } as Conversation,
  ])
  expect(store.conversations).toEqual([{ id: '7', name: '新数据', type: 'single' }])
})

it('replaces an overlapping page row instead of appending it', () => {
  const store = useChatStore()
  store.setConversations([{ id: '1', name: '旧会话', type: 'single' } as Conversation])
  store.mergeConversations([{ id: '1', name: '新会话', type: 'single' } as Conversation, { id: '2', name: '下一页', type: 'group' } as Conversation])
  expect(store.conversations.map(c => c.id)).toEqual(['1', '2'])
  expect(store.conversations[0].name).toBe('新会话')
})
```

- [ ] **Step 2: Verify RED**

Run: `npm exec vitest run tests/unit/stores/chat.test.ts`

Expected: FAIL because `setConversations` assigns raw input and `mergeConversations` is absent.

- [ ] **Step 3: Implement the merge seam**

```ts
function mergeByConversationId(existing: Conversation[], incoming: Conversation[]) {
  const merged = new Map<string, Conversation>()
  for (const conversation of [...existing, ...incoming]) merged.set(String(conversation.id), conversation)
  return [...merged.values()]
}

function setConversations(convs: Conversation[]) { conversations.value = mergeByConversationId([], convs) }
function mergeConversations(incoming: Conversation[]) { conversations.value = mergeByConversationId(conversations.value, incoming) }
```

Expose `mergeConversations` from `useConversation`, pass it into `useMainConversationLogic` from `Main.vue`, and call it for paginated results instead of concatenating arrays.

- [ ] **Step 4: Verify GREEN**

Run: `npm exec vitest run tests/unit/stores/chat.test.ts tests/unit/composables/useConversationLogic.test.ts`

Expected: PASS with 0 failures.

### Task 2: Add cursor pagination to the endpoint

**Files:**
- Modify: `qim-server/handler/conversation_handler.go:19-83,346-356`
- Test: `qim-server/handler/handler_test.go`

**Interfaces:**
- Consumes optional base64 JSON cursor: `{ "pinned": bool, "activity": RFC3339Nano string, "id": uint }`.
- Produces `next_cursor` alongside current `has_more` response data.

- [ ] **Step 1: Write failing handler tests**

```go
func TestGetConversationsCursorDoesNotRepeatRowsAfterActivityChanges(t *testing.T) {
  // Request 2 of 3 rows, retain next_cursor, move the remaining row to the top,
  // then request with cursor and assert no ID overlaps the first response.
}

func TestGetConversationsCursorBreaksActivityTiesByConversationID(t *testing.T) {
  // Create 3 rows with the same activity time; cursor pages must return all IDs exactly once.
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./handler -run 'TestGetConversationsCursor' -count=1`

Expected: FAIL because `next_cursor` does not exist and `cursor` is ignored.

- [ ] **Step 3: Implement stable cursor query**

Add a private cursor encoder/decoder. Order both branches by:

```sql
ORDER BY is_pinned DESC, COALESCE(c.last_message_at, c.created_at) DESC, cm.conversation_id DESC
```

For cursor requests, replace OFFSET with the strict descending tuple predicate; fetch `pageSize + 1`, trim the extra row, and encode the final returned row as `next_cursor`. Keep the existing offset query for requests without `cursor`.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./handler -run 'TestGetConversations' -count=1`

Expected: PASS with 0 failures.

### Task 3: Use cursors for client load-more

**Files:**
- Modify: `qim-client/src/composables/useMainConversationLogic.ts:10-61`
- Test: `qim-client/tests/unit/composables/useConversationLogic.test.ts`

**Interfaces:**
- Consumes `next_cursor?: string` from the list response.
- Produces load-more requests containing `cursor=<encoded cursor>` when present.

- [ ] **Step 1: Write failing test**

```ts
it('uses next_cursor for the next incremental request', async () => {
  // Mock a first response with next_cursor: 'cursor-1', then load more.
  // Expect the second request URL to contain 'cursor=cursor-1'.
})
```

- [ ] **Step 2: Verify RED**

Run: `npm exec vitest run tests/unit/composables/useConversationLogic.test.ts`

Expected: FAIL because load-more only increments a numeric page.

- [ ] **Step 3: Implement cursor-aware loading**

Keep `nextCursor = ref<string | null>(null)`. Capture `response.data.next_cursor || null`; pass it to later `loadConversations` calls and add the `cursor` query only when non-empty. Retain numeric page queries for first loads and old servers without cursors.

- [ ] **Step 4: Verify end-to-end focused coverage**

Run: `npm exec vitest run tests/unit/stores/chat.test.ts tests/unit/composables/useConversationLogic.test.ts && go test ./handler -run 'TestGetConversations' -count=1`

Expected: every command exits 0.

- [ ] **Step 5: Commit**

Run: `git add qim-client/src/stores/chat.ts qim-client/src/composables/useConversation.ts qim-client/src/composables/useMainConversationLogic.ts qim-client/src/views/Main.vue qim-client/tests/unit/stores/chat.test.ts qim-client/tests/unit/composables/useConversationLogic.test.ts qim-server/handler/conversation_handler.go qim-server/handler/handler_test.go && git commit -m "fix: prevent duplicate paged conversations"`
