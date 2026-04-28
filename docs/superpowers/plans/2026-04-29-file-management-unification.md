# 文件管理体验统一实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 统一管理用户上传文件和聊天文件，增加来源筛选和分享功能

**架构：** 后端设置文件来源标识（source字段），前端在文件箱增加来源筛选标签和分享按钮，复用现有分享功能

**技术栈：** Go/Gin/GORM (后端), Vue 3/TypeScript (前端)

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-server/handler/file_handler.go` | 修改 | 设置上传文件的source字段 |
| `qim-server/handler/message_handler.go` | 修改 | 设置聊天文件的source字段 |
| `qim-client/src/components/apps/file/FolderTree.vue` | 修改 | 增加来源筛选标签 |
| `qim-client/src/components/apps/FileManagementApp.vue` | 修改 | 支持来源筛选状态和分享功能 |
| `qim-client/src/components/apps/file/FileGridItem.vue` | 修改 | 增加分享按钮 |
| `qim-client/src/components/apps/file/FileListItem.vue` | 修改 | 增加分享按钮 |

---

### 任务 1：后端 - 设置上传文件的source字段

**文件：**
- 修改：`qim-server/handler/file_handler.go`

- [ ] **步骤 1：修改UploadFile函数**

在 `qim-server/handler/file_handler.go` 的 `UploadFile` 函数中，找到创建 `fileRecord` 的代码（约第61-68行），添加 `Source` 字段：

```go
fileRecord := model.File{
    Name:         file.Filename,
    OriginalName: file.Filename,
    StoragePath:  "/uploads/" + filename,
    Size:         file.Size,
    UserID:       userID.(uint),
    Source:       "upload",  // 新增：标识为用户上传
    CreatedAt:    time.Now(),
}
```

- [ ] **步骤 2：验证修改**

检查代码是否正确编译：

```bash
cd /Users/gracegaoya/work/project/qim/qim-server
go build ./...
```

预期：编译成功，无错误

- [ ] **步骤 3：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-server/handler/file_handler.go
git commit -m "feat: 设置上传文件的source字段为upload"
```

---

### 任务 2：后端 - 设置聊天文件的source字段

**文件：**
- 修改：`qim-server/handler/message_handler.go`

- [ ] **步骤 1：修改SendMessage函数**

在 `qim-server/handler/message_handler.go` 的 `SendMessage` 函数中，找到创建消息的代码（约第259-267行），在消息创建后添加文件来源更新逻辑：

```go
// 创建消息后，如果是文件消息，更新文件的source字段
if req.Type == "file" {
    // 解析文件URL，获取文件ID
    var fileData struct {
        URL string `json:"url"`
        ID  uint   `json:"id"`
    }
    if err := json.Unmarshal([]byte(req.Content), &fileData); err == nil {
        if fileData.ID > 0 {
            db.Model(&model.File{}).Where("id = ?", fileData.ID).Update("source", "chat")
        }
    }
}
```

这段代码应该添加在 `db.Create(&msg)` 之后，`db.Preload("Sender")...` 之前。

- [ ] **步骤 2：验证修改**

检查代码是否正确编译：

```bash
cd /Users/gracegaoya/work/project/qim/qim-server
go build ./...
```

预期：编译成功，无错误

- [ ] **步骤 3：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-server/handler/message_handler.go
git commit -m "feat: 设置聊天文件的source字段为chat"
```

---

### 任务 3：前端 - FolderTree组件增加来源筛选标签

**文件：**
- 修改：`qim-client/src/components/apps/file/FolderTree.vue`

- [ ] **步骤 1：修改模板部分**

在 `<template>` 中，在文件夹树列表之前添加来源筛选标签：

```vue
<template>
  <div class="folder-tree">
    <div class="folder-tree-header">
      <h3>文件夹</h3>
      <button class="create-folder-btn" @click="$emit('createFolder')">
        <i class="fas fa-plus"></i>
      </button>
    </div>

    <!-- 新增：来源筛选标签 -->
    <div class="source-filter">
      <button
        :class="['source-tab', { active: selectedSource === null }]"
        @click="handleSourceChange(null)"
      >
        全部
      </button>
      <button
        :class="['source-tab', { active: selectedSource === 'upload' }]"
        @click="handleSourceChange('upload')"
      >
        上传
      </button>
      <button
        :class="['source-tab', { active: selectedSource === 'chat' }]"
        @click="handleSourceChange('chat')"
      >
        聊天
      </button>
    </div>

    <div class="folder-tree-list">
      <!-- 原有的文件夹列表 -->
      ...
    </div>
  </div>
</template>
```

- [ ] **步骤 2：修改脚本部分**

在 `<script setup>` 中添加来源筛选相关的逻辑：

```typescript
interface Props {
  selectedFolderId: number | null
  selectedSource?: string | null  // 新增
}

const props = withDefaults(defineProps<Props>(), {
  selectedSource: null
})

const emit = defineEmits<{
  select: [folderId: number | null]
  createFolder: []
  sourceChange: [source: string | null]  // 新增
}>()

// 新增：处理来源变化
function handleSourceChange(source: string | null) {
  emit('sourceChange', source)
}

// 原有的其他函数...
```

- [ ] **步骤 3：添加样式**

在 `<style scoped>` 中添加来源筛选标签的样式：

```css
.source-filter {
  display: flex;
  gap: 4px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-color);
}

.source-tab {
  flex: 1;
  padding: 6px 12px;
  border: none;
  background: var(--hover-color);
  color: var(--text-color);
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.source-tab:hover {
  background: var(--primary-light);
  color: var(--primary-color);
}

.source-tab.active {
  background: var(--primary-color);
  color: white;
}
```

- [ ] **步骤 4：验证修改**

运行前端类型检查：

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：无类型错误

- [ ] **步骤 5：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/file/FolderTree.vue
git commit -m "feat: FolderTree组件增加来源筛选标签"
```

---

### 任务 4：前端 - FileManagementApp组件支持来源筛选

**文件：**
- 修改：`qim-client/src/components/apps/FileManagementApp.vue`

- [ ] **步骤 1：添加来源筛选状态**

在 `<script setup>` 中添加来源筛选状态：

```typescript
const selectedSource = ref<string | null>(null)

// 修改文件加载逻辑，传递source参数
const { items: files, total, hasMore, loadNextPage, loadMore, reset } = useFilePagination<FileItem>({
  fetchFn: async (page, pageSize) => {
    const params: any = { page, page_size: pageSize }
    if (selectedFolderId.value !== null) {
      if (selectedFolderId.value === -1) {
        params.starred = true
      } else {
        params.folder_id = selectedFolderId.value
      }
    }
    // 新增：添加来源筛选
    if (selectedSource.value !== null) {
      params.source = selectedSource.value
    }
    if (searchQuery.value) {
      params.search = searchQuery.value
    }

    const response = await fileApi.getFiles(params)
    return {
      items: response.data.data.files,
      total: response.data.data.total
    }
  }
})
```

- [ ] **步骤 2：添加来源变化处理函数**

```typescript
// 处理来源变化
function handleSourceChange(source: string | null) {
  selectedSource.value = source
  reset()
  loadNextPage()
}
```

- [ ] **步骤 3：修改模板，传递来源筛选状态**

```vue
<FolderTree
  :selected-folder-id="selectedFolderId"
  :selected-source="selectedSource"
  @select="handleFolderSelect"
  @create-folder="showCreateFolderModal = true"
  @source-change="handleSourceChange"
/>
```

- [ ] **步骤 4：添加文件分享处理函数**

```typescript
// 处理文件分享
function handleFileShare(file: FileItem) {
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type: 'file', data: file }
  }))
}
```

- [ ] **步骤 5：验证修改**

运行前端类型检查：

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：无类型错误

- [ ] **步骤 6：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/FileManagementApp.vue
git commit -m "feat: FileManagementApp支持来源筛选和分享功能"
```

---

### 任务 5：前端 - FileGridItem组件增加分享按钮

**文件：**
- 修改：`qim-client/src/components/apps/file/FileGridItem.vue`

- [ ] **步骤 1：修改模板，添加分享按钮**

在文件卡片的操作按钮区添加分享按钮：

```vue
<div class="file-actions">
  <button class="action-btn" @click.stop="$emit('star', file)" :title="file.is_starred ? '取消星标' : '星标'">
    <i class="fas" :class="file.is_starred ? 'fa-star' : 'fa-star-o'"></i>
  </button>
  <!-- 新增：分享按钮 -->
  <button class="action-btn" @click.stop="$emit('share', file)" title="分享">
    <i class="fas fa-share-alt"></i>
  </button>
  <button class="action-btn" @click.stop="$emit('delete', file)" title="删除">
    <i class="fas fa-trash"></i>
  </button>
</div>
```

- [ ] **步骤 2：添加share事件定义**

```typescript
defineEmits<{
  preview: [file: FileItem]
  download: [file: FileItem]
  star: [file: FileItem]
  share: [file: FileItem]  // 新增
  delete: [file: FileItem]
}>()
```

- [ ] **步骤 3：验证修改**

运行前端类型检查：

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：无类型错误

- [ ] **步骤 4：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/file/FileGridItem.vue
git commit -m "feat: FileGridItem组件增加分享按钮"
```

---

### 任务 6：前端 - FileListItem组件增加分享按钮

**文件：**
- 修改：`qim-client/src/components/apps/file/FileListItem.vue`

- [ ] **步骤 1：修改模板，添加分享按钮**

在列表项的操作列添加分享按钮：

```vue
<div class="file-actions-cell">
  <button class="action-btn" @click.stop="$emit('star', file)">
    <i class="fas" :class="file.is_starred ? 'fa-star starred' : 'fa-star-o'"></i>
  </button>
  <button class="action-btn" @click.stop="$emit('download', file)">
    <i class="fas fa-download"></i>
  </button>
  <!-- 新增：分享按钮 -->
  <button class="action-btn" @click.stop="$emit('share', file)" title="分享">
    <i class="fas fa-share-alt"></i>
  </button>
  <button class="action-btn" @click.stop="$emit('delete', file)">
    <i class="fas fa-trash"></i>
  </button>
</div>
```

- [ ] **步骤 2：添加share事件定义**

```typescript
defineEmits<{
  preview: [file: FileItem]
  download: [file: FileItem]
  star: [file: FileItem]
  share: [file: FileItem]  // 新增
  delete: [file: FileItem]
}>()
```

- [ ] **步骤 3：验证修改**

运行前端类型检查：

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：无类型错误

- [ ] **步骤 4：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/file/FileListItem.vue
git commit -m "feat: FileListItem组件增加分享按钮"
```

---

### 任务 7：前端 - FileList组件传递分享事件

**文件：**
- 修改：`qim-client/src/components/apps/file/FileList.vue`

- [ ] **步骤 1：修改网格视图，传递分享事件**

```vue
<FileGridItem
  v-for="file in files"
  :key="file.id"
  :file="file"
  @preview="$emit('preview', $event)"
  @download="$emit('download', $event)"
  @star="$emit('star', $event)"
  @share="$emit('share', $event)"
  @delete="$emit('delete', $event)"
/>
```

- [ ] **步骤 2：修改列表视图，传递分享事件**

```vue
<FileListItem
  v-for="file in files"
  :key="file.id"
  :file="file"
  @preview="$emit('preview', $event)"
  @download="$emit('download', $event)"
  @star="$emit('star', $event)"
  @share="$emit('share', $event)"
  @delete="$emit('delete', $event)"
/>
```

- [ ] **步骤 3：添加share事件定义**

```typescript
const emit = defineEmits<{
  preview: [file: FileItem]
  download: [file: FileItem]
  star: [file: FileItem]
  share: [file: FileItem]  // 新增
  delete: [file: FileItem]
  loadMore: []
}>()
```

- [ ] **步骤 4：验证修改**

运行前端类型检查：

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：无类型错误

- [ ] **步骤 5：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/file/FileList.vue
git commit -m "feat: FileList组件传递分享事件"
```

---

### 任务 8：前端 - FileManagementApp处理分享事件

**文件：**
- 修改：`qim-client/src/components/apps/FileManagementApp.vue`

- [ ] **步骤 1：修改FileList组件调用，添加share事件处理**

```vue
<FileList
  :files="files"
  :loading="loading"
  :has-more="hasMore"
  @preview="previewFile"
  @download="downloadFile"
  @star="toggleStar"
  @share="handleFileShare"
  @delete="deleteFile"
  @load-more="loadMoreFiles"
/>
```

- [ ] **步骤 2：验证修改**

运行前端类型检查：

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：无类型错误

- [ ] **步骤 3：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/FileManagementApp.vue
git commit -m "feat: FileManagementApp处理文件分享事件"
```

---

### 任务 9：验证和测试

**文件：** 全部

- [ ] **步骤 1：运行后端编译**

```bash
cd /Users/gracegaoya/work/project/qim/qim-server
go build ./...
```

预期：编译成功，无错误

- [ ] **步骤 2：运行前端类型检查**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：无类型错误

- [ ] **步骤 3：运行前端lint**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run lint
```

预期：无lint错误

- [ ] **步骤 4：运行前端构建**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run build
```

预期：构建成功

- [ ] **步骤 5：手动测试**

启动应用，测试以下功能：
1. 用户上传文件，检查数据库中文件的 `source` 字段是否为 `"upload"`
2. 在聊天中发送文件，检查数据库中文件的 `source` 字段是否为 `"chat"`
3. 在文件箱中点击来源筛选标签，验证文件列表是否正确筛选
4. 点击文件的分享按钮，验证分享功能是否正常工作

- [ ] **步骤 6：Commit所有修复**

```bash
cd /Users/gracegaoya/work/project/qim
git add -A
git commit -m "fix: 修复类型检查和lint问题"
```

---

## 规格覆盖度检查

| 规格需求 | 对应任务 | 状态 |
|---------|---------|------|
| 设置上传文件的source字段 | 任务 1 | ✅ |
| 设置聊天文件的source字段 | 任务 2 | ✅ |
| FolderTree增加来源筛选标签 | 任务 3 | ✅ |
| FileManagementApp支持来源筛选 | 任务 4 | ✅ |
| FileGridItem增加分享按钮 | 任务 5 | ✅ |
| FileListItem增加分享按钮 | 任务 6 | ✅ |
| FileList传递分享事件 | 任务 7 | ✅ |
| FileManagementApp处理分享事件 | 任务 8 | ✅ |
| 验证和测试 | 任务 9 | ✅ |

全部需求已覆盖，无占位符，类型一致。
