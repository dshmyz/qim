# 群文件空间 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不新增表的前提下，将现有文件和文件夹泛化为可复用的文件空间，并交付带角色权限的群文件共享中心。

**Architecture:** `files` 与 `folders` 通过 `scope_type` / `scope_id` 表达个人或群归属；`group_documents` 继续只服务 AI 知识库。`FileSpaceService` 是唯一外部 seam，隐藏范围、成员角色、目录有效性、引用文件生命周期和存储清理逻辑；个人文件处理器逐步转为 user scope 的调用者，群文件路由作为 group scope 的调用者。

**Tech Stack:** Go、Gin、GORM、SQLite/MySQL、Vue 3、TypeScript、Vitest。

## Global Constraints

- 不创建新表；仅迁移现有 `files` 与 `folders` 的新增列和索引。
- `group_documents` 不承载普通群文件，也不得触发普通群文件的 AI 解析或向量化。
- 所有群成员可上传、下载；仅群主/管理员可建目录、移动、删除。
- 聊天附件仅由管理员手动转存；转存复用二进制对象，删除群引用不能影响原附件。
- 旧个人文件和文件夹迁移后必须保持既有 URL 和行为。

---

## File Structure

- `qim-server/model/model.go`：为 `File` / `Folder` 增加 scope 字段和引用来源约定。
- `qim-server/app/init.go`：执行 AutoMigrate 后回填旧个人文件空间并创建复合索引。
- `qim-server/service/file_space_service.go`：深的文件空间 module，封装权限、目录和引用逻辑。
- `qim-server/service/file_space_service_test.go`：测试 scope 隔离、群角色、目录和转存删除。
- `qim-server/handler/group_file_handler.go`：只把 HTTP 请求映射到文件空间接口。
- `qim-server/app/routes.go`：注册群文件路由。
- `qim-client/src/api/groupFiles.ts`：群文件请求适配器。
- `qim-client/src/components/groups/GroupFilesPanel.vue`：目录、列表、搜索、上传与管理动作。
- `qim-client/src/components/shared/GroupDetail.vue`、`qim-client/src/components/chat/ChatWindow.vue`：打开同一群文件中心的两个入口。
- `qim-client/src/components/chat/MessageContextMenu.vue`：管理员“保存到群文件”入口。

### Task 1: 迁移 scope 数据模型并保持个人文件兼容

**Files:**
- Modify: `qim-server/model/model.go:227-260`
- Modify: `qim-server/app/init.go:346-430`
- Create: `qim-server/app/file_space_migration_test.go`

**Interfaces:**
- Produces `model.File.ScopeType string`, `ScopeID uint`, `Source string`, `SourceID string`。
- Produces `model.Folder.ScopeType string`, `ScopeID uint`；个人文件夹的 `UserID` 保留为创建者。
- Produces `MigrateFileSpaces(db *gorm.DB) error`，可重复执行且只回填 scope 为空的历史记录。

- [ ] **Step 1: 写失败的迁移测试**

```go
func TestMigrateFileSpacesBackfillsLegacyUserRecords(t *testing.T) {
  db := newTestDB(t)
  file := model.File{UserID: 7, Name: "legacy.txt", Size: 1, StoragePath: "files/legacy"}
  folder := model.Folder{UserID: 7, Name: "旧文件夹"}
  require.NoError(t, db.Create(&file).Error)
  require.NoError(t, db.Create(&folder).Error)
  require.NoError(t, MigrateFileSpaces(db))
  require.NoError(t, db.First(&file, file.ID).Error)
  require.Equal(t, "user", file.ScopeType)
  require.Equal(t, uint(7), file.ScopeID)
  require.NoError(t, db.First(&folder, folder.ID).Error)
  require.Equal(t, "user", folder.ScopeType)
  require.Equal(t, uint(7), folder.ScopeID)
}
```

- [ ] **Step 2: 验证测试失败**

运行：`cd qim-server && go test ./app -run TestMigrateFileSpacesBackfillsLegacyUserRecords -v`

预期：FAIL，因为 scope 字段和迁移函数尚不存在。

- [ ] **Step 3: 写最小迁移实现**

```go
type File struct {
  // 保留已有字段
  ScopeType string `json:"scope_type" gorm:"size:20;not null;default:'user';index:idx_file_scope_folder,priority:1"`
  ScopeID uint `json:"scope_id" gorm:"not null;default:0;index:idx_file_scope_folder,priority:2"`
}

func MigrateFileSpaces(db *gorm.DB) error {
  if err := db.Where("scope_type = '' OR scope_type IS NULL").
    Model(&model.File{}).Updates(map[string]interface{}{"scope_type": "user", "scope_id": gorm.Expr("user_id")}).Error; err != nil { return err }
  return db.Where("scope_type = '' OR scope_type IS NULL").
    Model(&model.Folder{}).Updates(map[string]interface{}{"scope_type": "user", "scope_id": gorm.Expr("user_id")}).Error
}
```

在 `MigrateDB` 的模型迁移后调用 `MigrateFileSpaces`，并为 folders 建 `(scope_type, scope_id, parent_id)` 索引。

- [ ] **Step 4: 验证迁移与现有文件测试**

运行：`cd qim-server && go test ./app -run TestMigrateFileSpacesBackfillsLegacyUserRecords -v && go test ./service -run 'Test.*Folder|Test.*File' -v`

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add qim-server/model/model.go qim-server/app/init.go qim-server/app/file_space_migration_test.go
git commit -m "feat(files): add reusable file space scope"
```

### Task 2: 实现深的文件空间 module

**Files:**
- Create: `qim-server/service/file_space_service.go`
- Create: `qim-server/service/file_space_service_test.go`
- Modify: `qim-server/service/file_service.go`

**Interfaces:**
- Consumes `FileSpace{Type string; ID uint}`，有效 type 为 `user`、`group`。
- Produces `List(ctx, actorID, space, query)`, `CreateFolder(ctx, actorID, space, name, parentID)`, `Move(ctx, actorID, space, fileIDs, folderID)`, `Delete(ctx, actorID, space, fileID)`, `ShareReference(ctx, actorID, groupID, sourceFileID, folderID)`。
- 对 group scope：成员可 list/upload/download；只有 owner/admin 可调用后三个管理操作。

- [ ] **Step 1: 写失败的权限与引用测试**

```go
func TestFileSpaceShareReferenceDoesNotDeleteOriginal(t *testing.T) {
  source := createUserFile(t, db, 11, "chat.pdf")
  shared, err := spaces.ShareReference(ctx, adminID, FileSpace{Type: "group", ID: groupID}, source.ID, nil)
  require.NoError(t, err)
  require.Equal(t, "shared_reference", shared.Source)
  require.Equal(t, strconv.FormatUint(uint64(source.ID), 10), shared.SourceID)
  require.NoError(t, spaces.Delete(ctx, adminID, FileSpace{Type: "group", ID: groupID}, shared.ID))
  require.NoError(t, db.First(&source, source.ID).Error)
}

func TestFileSpaceRejectsMemberFolderManagement(t *testing.T) {
  _, err := spaces.CreateFolder(ctx, memberID, FileSpace{Type: "group", ID: groupID}, "会议纪要", nil)
  require.ErrorIs(t, err, ErrFileSpaceForbidden)
}
```

- [ ] **Step 2: 验证测试失败**

运行：`cd qim-server && go test ./service -run 'TestFileSpace(ShareReference|RejectsMember)' -v`

预期：FAIL，因为 `FileSpaceService` 尚不存在。

- [ ] **Step 3: 实现小 interface、深 implementation**

```go
type FileSpace struct { Type string; ID uint }

func (s *FileSpaceService) authorize(ctx context.Context, actorID uint, space FileSpace, action FileSpaceAction) error
func (s *FileSpaceService) CreateFolder(ctx context.Context, actorID uint, space FileSpace, name string, parentID *uint) (*model.Folder, error)
func (s *FileSpaceService) ShareReference(ctx context.Context, actorID uint, space FileSpace, sourceFileID uint, folderID *uint) (*model.File, error)
```

`authorize` 在 group scope 查询 `ConversationMember` 与 `Group`：upload/download 允许成员，manage 只允许 `owner`/`admin`。`ShareReference` 复制元数据并复用 `StoragePath`/`Checksum`，写 `Source="shared_reference"` 和原 ID；`Delete` 只软删引用记录，物理删除只允许非引用资源且确认无其他同路径有效记录。

- [ ] **Step 4: 验证 module**

运行：`cd qim-server && go test ./service -run TestFileSpace -v`

预期：PASS，包含跨群拒绝、跨空间目录拒绝与循环目录拒绝测试。

- [ ] **Step 5: 提交**

```bash
git add qim-server/service/file_space_service.go qim-server/service/file_space_service_test.go qim-server/service/file_service.go
git commit -m "feat(files): centralize scoped file permissions"
```

### Task 3: 暴露群文件 HTTP 路由

**Files:**
- Create: `qim-server/handler/group_file_handler.go`
- Create: `qim-server/handler/group_file_handler_test.go`
- Modify: `qim-server/app/routes.go:430-445`

**Interfaces:**
- `GET /api/v1/groups/:id/files?page=&page_size=&folder_id=&search=` 返回分页群文件。
- `POST /api/v1/groups/:id/folders`、`PATCH /api/v1/groups/:id/files/:file_id`、`DELETE /api/v1/groups/:id/files/:file_id` 为管理员操作。
- `POST /api/v1/groups/:id/files/references` 接收 `{ "file_id": 12, "folder_id": 3 }`，为管理员转存聊天附件。

- [ ] **Step 1: 写失败的 handler 测试**

```go
func TestGroupFileHandlerForbidsMemberFolderCreation(t *testing.T) {
  response := requestAsMember(t, router, http.MethodPost, "/api/v1/groups/10/folders", `{"name":"规范"}`)
  require.Equal(t, http.StatusForbidden, response.Code)
}

func TestGroupFileHandlerListsOnlyRequestedGroupScope(t *testing.T) {
  response := requestAsMember(t, router, http.MethodGet, "/api/v1/groups/10/files", "")
  require.JSONEq(t, `{"code":0,"data":{"total":1}}`, response.Body.String())
}
```

- [ ] **Step 2: 验证测试失败**

运行：`cd qim-server && go test ./handler -run TestGroupFileHandler -v`

预期：FAIL，因为路由和 handler 未注册。

- [ ] **Step 3: 写薄 handler**

每个 handler 只解析参数、取得 `user_id`、构造 `FileSpace{Type:"group", ID:group.ID}` 并调用 `FileSpaceService`。不得在 handler 写 GORM 查询、群角色判断或存储清理逻辑。

- [ ] **Step 4: 验证路由**

运行：`cd qim-server && go test ./handler -run TestGroupFileHandler -v`

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add qim-server/handler/group_file_handler.go qim-server/handler/group_file_handler_test.go qim-server/app/routes.go
git commit -m "feat(groups): expose shared file space routes"
```

### Task 4: 群文件中心与双入口

**Files:**
- Create: `qim-client/src/api/groupFiles.ts`
- Create: `qim-client/src/components/groups/GroupFilesPanel.vue`
- Create: `qim-client/tests/unit/group-files-panel.test.ts`
- Modify: `qim-client/src/components/shared/GroupDetail.vue`
- Modify: `qim-client/src/components/chat/ChatWindow.vue`
- Modify: `qim-client/src/components/chat/MessageContextMenu.vue`

**Interfaces:**
- `GroupFilesPanel` 输入 `groupId`、`canManage`，发出 `close`。
- `groupFiles.list(groupId, filters)`、`createFolder`、`move`、`remove`、`shareReference` 使用 Task 3 路由。

- [ ] **Step 1: 写失败的组件测试**

```ts
it('shows upload and download to every member but directory management only to managers', async () => {
  const member = mount(GroupFilesPanel, { props: { groupId: 1, canManage: false } })
  expect(member.text()).toContain('上传文件')
  expect(member.text()).not.toContain('新建文件夹')
  const manager = mount(GroupFilesPanel, { props: { groupId: 1, canManage: true } })
  expect(manager.text()).toContain('新建文件夹')
})
```

- [ ] **Step 2: 验证测试失败**

运行：`cd qim-client && npx vitest run tests/unit/group-files-panel.test.ts`

预期：FAIL，因为面板不存在。

- [ ] **Step 3: 实现文件中心和入口**

面板显示目录树、搜索框、分页列表（名称、上传者、时间、大小）及上传动作。`GroupDetail` 添加“群文件”按钮；`ChatWindow` 顶部添加文件夹图标；两者使用同一个打开状态和 `GroupFilesPanel`。只有 `canManage` 为真时显示建文件夹、移动、删除；聊天文件右键菜单仅向管理员显示“保存到群文件”，打开目录选择后调用 `shareReference`。

- [ ] **Step 4: 验证 UI**

运行：`cd qim-client && npx vitest run tests/unit/group-files-panel.test.ts && npm run build`

预期：PASS，构建完成。

- [ ] **Step 5: 提交**

```bash
git add qim-client/src/api/groupFiles.ts qim-client/src/components/groups/GroupFilesPanel.vue qim-client/src/components/shared/GroupDetail.vue qim-client/src/components/chat/ChatWindow.vue qim-client/src/components/chat/MessageContextMenu.vue qim-client/tests/unit/group-files-panel.test.ts
git commit -m "feat(groups): add shared file center"
```

## Final Verification

- [ ] 运行 `cd qim-server && go test ./app ./service ./handler`，并记录任何既有数据库 schema 测试问题。
- [ ] 运行 `cd qim-client && npx vitest run tests/unit/group-files-panel.test.ts && npm run build`。
- [ ] 手动验证：成员上传/下载、管理员目录管理、聊天附件手动转存、删除群引用不影响原聊天附件、AI 知识库列表不含普通群文件。

### Task 5: 补齐群成员上传归属与受控下载

**Files:**
- Modify: `qim-server/handler/group_file_handler.go`
- Modify: `qim-server/handler/group_file_handler_test.go`
- Modify: `qim-server/app/routes.go`
- Modify: `qim-server/service/file_space_service.go`
- Modify: `qim-client/src/api/groupFiles.ts`

**Interfaces:**
- `POST /api/v1/groups/:id/files` 接收 `{ "file_id": 12, "folder_id": 3 }`；任何群成员可将自己刚上传的个人文件归属到该群空间。
- `GET /api/v1/groups/:id/files/:file_id/download` 仅允许当前群成员下载该群空间文件。

- [ ] **Step 1: 写失败的成员上传归属与下载授权测试**

```go
func TestGroupFileHandlerAllowsMemberToAttachOwnUpload(t *testing.T) {
  response := requestAsMember(t, router, http.MethodPost, "/api/v1/groups/10/files", `{"file_id":12}`)
  require.Equal(t, http.StatusOK, response.Code)
}

func TestGroupFileHandlerRejectsFormerMemberDownload(t *testing.T) {
  response := requestAsFormerMember(t, router, http.MethodGet, "/api/v1/groups/10/files/12/download", "")
  require.Equal(t, http.StatusForbidden, response.Code)
}
```

- [ ] **Step 2: 运行并确认红灯**

运行：`cd qim-server && go test ./handler -run 'TestGroupFileHandler(AllowsMemberToAttachOwnUpload|RejectsFormerMemberDownload)' -v`

预期：FAIL，因为路由尚不存在。

- [ ] **Step 3: 写最小实现**

`FileSpaceService` 新增成员级 `AttachUpload` 和 `OpenDownload`：前者验证原文件属于当前用户、目标目录属于群空间，再将同一资源改为 group scope；后者先执行 group download 授权再返回文件存储路径。handler 仅映射请求和流式响应。

- [ ] **Step 4: 验证**

运行：`cd qim-server && go test ./handler -run TestGroupFileHandler -v && go test ./service -run TestFileSpace -v`

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add qim-server/handler/group_file_handler.go qim-server/handler/group_file_handler_test.go qim-server/app/routes.go qim-server/service/file_space_service.go qim-client/src/api/groupFiles.ts docs/superpowers/plans/2026-07-17-group-file-space.md
git commit -m "feat(groups): support member uploads and downloads"
```
