# 群文件空间加固 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让群管理员能够安全保存本群成员的聊天附件，并消除群文件的会话串写、迁移可见性、引用清理和请求竞态问题。

**Architecture:** 以群会话消息作为聊天附件转存的唯一授权证据；`file_id` 只能在服务器确认其属于目标群消息后被引用。文件记录仍复用 `files/folders` 的 scope 字段，不增加表；同一存储路径的引用创建与删除由进程内路径锁和数据库事务协调。

**Tech Stack:** Go、Gin、GORM、SQLite、Vue 3、TypeScript、Vitest、testify。

## Global Constraints

- 不新增数据库表。
- 不按 `source=chat` 或任意文件 ID 放行跨空间引用。
- 个人文件上传后 attach 失败必须保留，不能隐式删除。
- 所有新增行为先写失败测试，再写最小实现。
- 不触碰工作区中无关的骨架屏、产品名和消息选择改动。

---

### Task 1: 以群消息归属授权聊天附件转存

**Files:**
- Modify: `qim-server/handler/group_file_handler.go:190-208`
- Modify: `qim-server/service/file_space_service.go:221-261`
- Modify: `qim-server/handler/group_file_handler_test.go:20-55, 140-220`
- Modify: `qim-server/service/file_space_service_test.go`
- Modify: `qim-client/src/api/groupFiles.ts:74-78`
- Modify: `qim-client/src/components/chat/ChatWindow.vue:1571-1584`

**Interfaces:**
- Consumes: `model.Message{ID, ConversationID, Type, Content}` and `model.Group{ID, ConversationID}`.
- Produces: `FileSpaceService.ShareMessageAttachment(ctx, actorID, groupID, messageID, fileID, folderID) (*model.File, error)`.
- Produces: `POST /api/v1/groups/:id/files/references` JSON `{ "message_id": uint, "file_id": uint, "folder_id": *uint }`.

- [ ] **Step 1: Write the failing handler tests**

Add `model.Message{}` to the setup migration. Create a personal file owned by the member and a `file` message in the target group's conversation with JSON content `{"id": <fileID>}`. Assert the owner receives HTTP 200 when posting both IDs. Add two denial cases: same file in another group conversation and a mismatched `file_id`; both must be HTTP 403.

```go
response := requestAsGroupFileUser(t, router, owner.ID, http.MethodPost, path,
  fmt.Sprintf(`{"message_id":%d,"file_id":%d}`, message.ID, upload.ID))
require.Equal(t, http.StatusOK, response.Code)
```

- [ ] **Step 2: Run the handler test red**

Run: `go test ./handler -run TestGroupFileHandlerShareMessageAttachment -count=1 -v`

Expected: FAIL because the handler does not accept `message_id` or verify message ownership.

- [ ] **Step 3: Implement the server-side verification**

In `ShareGroupFileReference`, bind `MessageID`. Resolve the URL group to `group.ID`, then call `ShareMessageAttachment`. In that service method:

```go
if err := s.authorize(ctx, actorID, FileSpace{Type: "group", ID: groupID}, FileSpaceActionManage); err != nil { return nil, err }
var group model.Group
if err := s.db.WithContext(ctx).Where("id = ?", groupID).First(&group).Error; err != nil { return nil, ErrFileSpaceForbidden }
var message model.Message
if err := s.db.WithContext(ctx).Where("id = ? AND conversation_id = ? AND type = ?", messageID, group.ConversationID, "file").First(&message).Error; err != nil { return nil, ErrFileSpaceForbidden }
```

Parse `message.Content` into a local `{ ID uint \`json:"id"\` }` struct; reject invalid JSON or unequal IDs. Load the source file by ID, validate target folder, then create the group-scoped reference. Keep generic `ShareReference` only for explicitly authorized same-space callers, or make it unexported if no caller remains.

- [ ] **Step 4: Pass server tests**

Run: `go test ./handler ./service -run 'TestGroupFileHandlerShareMessageAttachment|TestFileSpace' -count=1`

Expected: PASS; owner/admin can save only a file attachment contained in the selected group conversation.

- [ ] **Step 5: Update the client request**

Store both selected message ID and parsed file ID in `ChatWindow`; change `groupFiles.shareReference` to submit both. Keep JSON parse errors as a user-visible `无法识别聊天附件` response and do not open the panel.

- [ ] **Step 6: Commit**

```bash
git add qim-server/handler/group_file_handler.go qim-server/service/file_space_service.go qim-server/handler/group_file_handler_test.go qim-server/service/file_space_service_test.go qim-client/src/api/groupFiles.ts qim-client/src/components/chat/ChatWindow.vue
git commit -m "fix(groups): authorize chat file references by message"
```

### Task 2: 关闭会话串写和列表响应竞态

**Files:**
- Modify: `qim-client/src/components/chat/ChatWindow.vue:340-341, 567-589`
- Modify: `qim-client/src/components/groups/GroupFilesPanel.vue:123-153`
- Modify: `qim-client/tests/unit/group-files-panel.test.ts`
- Modify: `qim-client/tests/unit/components/ChatWindowMergedForward.test.ts` or create `qim-client/tests/unit/components/ChatWindowGroupFiles.test.ts`

**Interfaces:**
- Produces: `closeGroupFiles()` executed whenever `props.conversation.id` changes.
- Produces: `latestListRequest` monotonic token; only matching requests mutate `files`, `folders`, `page`, `pageSize`, and `total`.

- [ ] **Step 1: Write failing component tests**

Create deferred `groupFiles.list` promises. Change `groupId` from `1` to `2`, resolve group 2 first and group 1 second, and assert displayed data remains group 2. Mount `ChatWindow` with a visible group file panel/reference, change its conversation ID, and assert the `GroupFilesPanel` is absent.

- [ ] **Step 2: Run tests red**

Run: `npx vitest run tests/unit/group-files-panel.test.ts tests/unit/components/ChatWindowGroupFiles.test.ts`

Expected: FAIL because the old response mutates current state and conversation changes retain the panel state.

- [ ] **Step 3: Implement minimal state isolation**

At the top of `loadFiles`, increment a local `requestVersion` ref and capture it. After awaiting `groupFiles.list`, return unless captured version still equals the ref. In the existing conversation-ID watcher call `closeGroupFiles()` before loading the new draft.

- [ ] **Step 4: Run tests green**

Run: `npx vitest run tests/unit/group-files-panel.test.ts tests/unit/components/ChatWindowGroupFiles.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/components/chat/ChatWindow.vue qim-client/src/components/groups/GroupFilesPanel.vue qim-client/tests/unit/group-files-panel.test.ts qim-client/tests/unit/components/ChatWindowGroupFiles.test.ts
git commit -m "fix(groups): isolate file panel state by conversation"
```

### Task 3: 统一引用的删除与存储清理

**Files:**
- Create: `qim-server/service/storage_path_lock.go`
- Modify: `qim-server/service/file_service.go:194-214, 400-403`
- Modify: `qim-server/service/file_space_service.go:199-218, 221-261`
- Modify: `qim-server/service/file_service_test.go`
- Modify: `qim-server/service/file_space_service_test.go`

**Interfaces:**
- Produces: `withStoragePathLock(path string, fn func() error) error`, backed by a package-level keyed mutex map and releasing the key after `fn` returns.
- Produces: `FileService.DeleteFile` that loads the record, soft-deletes it, then calls `deleteStoragePathIfUnreferenced` under the path lock.

- [ ] **Step 1: Write failing lifecycle tests**

Add a `DeleteFile` test with a fake `StorageAccessor`: delete an unreferenced personal file and assert exactly one `DeleteByPath` call. Add a concurrent test that blocks `DeleteByPath`, starts a reference creation for the same path, then releases deletion; assert the new reference remains downloadable and the accessor was not asked to remove an active path.

- [ ] **Step 2: Run tests red**

Run: `go test ./service -run 'TestFileServiceDeleteFile|TestFileSpace.*Concurrent' -count=1 -v`

Expected: FAIL because `DeleteFile` only soft-deletes and concurrent operations have no shared lock.

- [ ] **Step 3: Implement one shared deletion flow**

Use `withStoragePathLock` around all operations that create a reference or delete a record with a `StoragePath`. Inside a GORM transaction, delete/create the record and count active records for the same path. Call physical `DeleteByPath` only after the transaction reports no active reference and while still holding the path lock. Make `FileSpaceService.ShareMessageAttachment` and any remaining reference creator acquire the same lock before insert.

- [ ] **Step 4: Run tests green**

Run: `go test ./service -run 'TestFileServiceDeleteFile|TestFileSpace' -count=1`

Expected: PASS; all deletion APIs clean unreferenced storage, but never remove a path used by a concurrently created reference.

- [ ] **Step 5: Commit**

```bash
git add qim-server/service/storage_path_lock.go qim-server/service/file_service.go qim-server/service/file_space_service.go qim-server/service/file_service_test.go qim-server/service/file_space_service_test.go
git commit -m "fix(files): serialize shared storage cleanup"
```

### Task 4: 让文件空间迁移失败可见且可恢复

**Files:**
- Modify: `qim-server/app/init.go:286-300, 347-424`
- Modify: `qim-server/app/init_test.go`
- Modify: `qim-server/app/file_space_migration_test.go`

**Interfaces:**
- Produces: `func MigrateDB(db *gorm.DB) error`.
- Consumes: `InitApp`, which terminates initialization with `log.Fatalf` if `MigrateDB` returns an error.

- [ ] **Step 1: Write the failing migration error test**

Use a GORM callback or a closed/invalid test table to make the `files` update in `MigrateFileSpaces` return an error. Assert `MigrateDB(db)` returns that error rather than continuing silently. Keep the existing `user/0` regression to prove retry stays idempotent.

- [ ] **Step 2: Run tests red**

Run: `go test ./app -run 'TestMigrateDB.*FileSpace' -count=1 -v`

Expected: FAIL because `MigrateDB` has no error result.

- [ ] **Step 3: Propagate errors**

Change `MigrateDB` to return `error`; return an error when compatibility columns, model migration, or `MigrateFileSpaces` fails. Update `InitApp` to `log.Fatalf("数据库迁移失败: %v", err)` before seed operations, and update existing tests to `require.NoError(t, MigrateDB(db))`.

- [ ] **Step 4: Run tests green**

Run: `go test ./app -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add qim-server/app/init.go qim-server/app/init_test.go qim-server/app/file_space_migration_test.go
git commit -m "fix(files): fail startup when scope migration fails"
```

### Task 5: 完成上传失败反馈和暗色可读性

**Files:**
- Modify: `qim-client/src/components/groups/GroupFilesPanel.vue:216-232`
- Modify: `qim-client/src/components/ai/AIMessageBadge.vue`
- Modify: `qim-client/tests/unit/group-files-panel.test.ts`
- Create: `qim-client/tests/unit/components/AIMessageBadge.test.ts`

**Interfaces:**
- Produces: attach failure toast `上传成功，文件已保留在文件箱，可重试加入群文件`.
- Produces: explicit `color` on `.ai-message-badge` and an explicit dark-mode override.

- [ ] **Step 1: Write failing tests**

Mock `fileApi.uploadFile` success and `groupFiles.attach` rejection. Assert the user-facing error includes `已保留在文件箱`. Mount `AIMessageBadge` and assert its scoped CSS contains both a foreground `color` and `prefers-color-scheme: dark` rule.

- [ ] **Step 2: Run tests red**

Run: `npx vitest run tests/unit/group-files-panel.test.ts tests/unit/components/AIMessageBadge.test.ts`

Expected: FAIL because generic upload failure text is used and badge CSS has no explicit foreground color.

- [ ] **Step 3: Implement minimum feedback and contrast**

Split upload and attach errors in `handleUpload`: upload failures retain `上传群文件失败`; attach failures show the retained-file message. Set a readable default badge text color and a lighter dark-mode variant without changing layout.

- [ ] **Step 4: Run client verification**

Run: `npx vitest run tests/unit/group-files-panel.test.ts tests/unit/components/AIMessageBadge.test.ts && npm run build`

Expected: tests PASS and Vite build exits 0; record existing chunk/CSS warnings separately if emitted.

- [ ] **Step 5: Commit**

```bash
git add qim-client/src/components/groups/GroupFilesPanel.vue qim-client/src/components/ai/AIMessageBadge.vue qim-client/tests/unit/group-files-panel.test.ts qim-client/tests/unit/components/AIMessageBadge.test.ts
git commit -m "fix(groups): clarify attach failure and badge contrast"
```

## Plan self-review

- Spec coverage: Task 1 covers safe administrator attachment transfer; Task 2 covers session and list races; Task 3 covers storage cleanup; Task 4 covers migration failure; Task 5 covers upload feedback, JSON client behavior remains in Task 1, and dark contrast.
- Scope: no table, repository-wide refactor, or generic source-based authorization is introduced.
- Type consistency: the only new API request uses `message_id`, `file_id`, and optional `folder_id` consistently across handler, service and client.
