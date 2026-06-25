# Conversation Member Join Timestamp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure every newly created `ConversationMember` receives a valid join timestamp so MySQL strict mode never receives `0000-00-00`.

**Architecture:** Make `JoinedAt` a GORM creation timestamp at the shared model boundary. This covers direct-message, bot, group, repository, and future creation paths without duplicating `time.Now()` in each caller. Add a model persistence regression test using the existing SQLite test database.

**Tech Stack:** Go 1.25, GORM, modernc SQLite, Testify.

---

### Task 1: Add the regression test

**Files:**
- Modify: `qim-server/handler/handler_test.go`
- Test: `qim-server/handler/handler_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestConversationMember_CreateSetsJoinedAt(t *testing.T) {
	db := setupHandlerTestDB(t)
	user := createTestUser(t, db)
	conv := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(conv).Error)

	member := &model.ConversationMember{ConversationID: conv.ID, UserID: user.ID, Role: "member"}
	require.NoError(t, db.Create(member).Error)
	assert.False(t, member.JoinedAt.IsZero())
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./handler -run '^TestConversationMember_CreateSetsJoinedAt$' -count=1`

Expected: FAIL because `JoinedAt` remains Go's zero time.

### Task 2: Set the timestamp at the model boundary

**Files:**
- Modify: `qim-server/model/model.go:173`

- [ ] **Step 1: Add GORM creation-time metadata**

```go
JoinedAt time.Time `json:"joined_at" gorm:"autoCreateTime"`
```

- [ ] **Step 2: Run the regression test to verify it passes**

Run: `go test ./handler -run '^TestConversationMember_CreateSetsJoinedAt$' -count=1`

Expected: PASS; the persisted member's `JoinedAt` is non-zero.

### Task 3: Quote the reserved groups table in group queries

**Files:**
- Modify: `qim-server/service/conversation_service.go:489-490`
- Modify: `qim-server/handler/group_handler.go:1296`
- Test: `qim-server/service/service_test.go`

- [ ] **Step 1: Write the failing service test**

```go
results, err := svc.SearchGroupsByName("项目", user.ID)
require.NoError(t, err)
require.Len(t, results, 1)
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./service -run '^TestConversationService_SearchGroupsByName_QuotesGroupsTable$' -count=1`

Expected: FAIL with a SQL syntax error near `groups`.

- [ ] **Step 3: Quote each `groups` table reference**

```go
s.db.Joins("JOIN `groups` ON `groups`.conversation_id = conversations.id")
```

```sql
INNER JOIN `groups` g ON g.conversation_id = cm.conversation_id
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./service -run '^TestConversationService_SearchGroupsByName_QuotesGroupsTable$' -count=1`

Expected: PASS with the matching group returned.

### Task 4: Verify the affected packages

**Files:**
- Verify: `qim-server/handler`

- [ ] **Step 1: Run all handler and service tests**

Run: `go test ./handler ./service -count=1`

Expected: PASS with no test failures.
