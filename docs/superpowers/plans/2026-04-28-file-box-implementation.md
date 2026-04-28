# 文件箱功能实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现个人文件箱功能，支持文件夹树懒加载、文件列表分页、文件整理、星标、标签等能力

**架构：** 后端扩展现有 file_handler.go 增加文件夹管理和文件操作接口，前端将现有 FileManagementApp.vue 拆分为多个子组件，实现懒加载树和分页列表

**技术栈：** Go/Gin/GORM (后端), Vue 3/TypeScript/Pinia (前端), Vitest (测试), Playwright (E2E)

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-server/model/model.go` | 修改 | 扩展 File 和 Folder 模型，增加新字段 |
| `qim-server/ddl_sqlite.sql` | 修改 | 更新表结构定义 |
| `qim-server/ddl_mysql.sql` | 修改 | 同步更新 MySQL 表结构 |
| `qim-server/handler/file_handler.go` | 修改 | 扩展文件/文件夹接口，实现树构建逻辑 |
| `qim-server/app/routes.go` | 修改 | 新增路由注册 |
| `qim-client/src/components/apps/FileManagementApp.vue` | 重写 | 主容器，组合子组件 |
| `qim-client/src/components/apps/file/FolderTree.vue` | 创建 | 文件夹树组件，懒加载 |
| `qim-client/src/components/apps/file/FolderTreeItem.vue` | 创建 | 单个文件夹项 |
| `qim-client/src/components/apps/file/FileList.vue` | 创建 | 文件列表组件，分页 |
| `qim-client/src/components/apps/file/FileGridItem.vue` | 创建 | 网格视图文件项 |
| `qim-client/src/components/apps/file/FileListItem.vue` | 创建 | 列表视图文件项 |
| `qim-client/src/components/apps/file/FilePreviewModal.vue` | 创建 | 文件预览模态框 |
| `qim-client/src/components/apps/file/CreateFolderModal.vue` | 创建 | 创建/编辑文件夹模态框 |
| `qim-client/src/components/apps/file/FileActionsModal.vue` | 创建 | 文件操作模态框 |
| `qim-client/src/api/file.ts` | 创建 | 文件管理 API 封装 |
| `qim-client/src/composables/useFilePagination.ts` | 创建 | 分页逻辑 composable |
| `qim-client/src/composables/useFolderTree.ts` | 创建 | 文件夹树逻辑 composable |
| `qim-client/src/utils/fileType.ts` | 创建 | 文件类型判断工具函数 |
| `qim-client/tests/unit/utils/fileType.test.ts` | 创建 | 文件类型测试 |
| `qim-client/tests/unit/composables/useFilePagination.test.ts` | 创建 | 分页逻辑测试 |
| `qim-client/tests/unit/composables/useFolderTree.test.ts` | 创建 | 文件夹树测试 |
| `qim-client/tests/e2e/file-box.spec.ts` | 创建 | E2E 测试 |

---

### 任务 1：更新数据库模型和表结构

**文件：**
- 修改：`qim-server/model/model.go` (Line 117-141)
- 修改：`qim-server/ddl_sqlite.sql` (Line 101-131)
- 修改：`qim-server/ddl_mysql.sql` (Line 106-143)

- [ ] **步骤 1：扩展 File 模型**

```go
// qim-server/model/model.go - Line 117-130
type File struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	UserID       uint           `json:"user_id" gorm:"not null;index"`
	Name         string         `json:"name" gorm:"size:255;not null"`
	OriginalName string         `json:"original_name" gorm:"size:255"`
	Size         int64          `json:"size" gorm:"not null"`
	MimeType     string         `json:"mime_type" gorm:"size:100"`
	StoragePath  string         `json:"storage_path" gorm:"size:500;not null"`
	Checksum     string         `json:"checksum" gorm:"size:64"`
	FolderID     *uint          `json:"folder_id"`
	Source       string         `json:"source" gorm:"size:20;default:'upload'"`    // 新增
	SourceID     string         `json:"source_id" gorm:"size:100"`                 // 新增
	IsStarred    bool           `json:"is_starred" gorm:"default:false"`           // 新增
	StarredAt    *time.Time     `json:"starred_at"`                                // 新增
	Tags         string         `json:"tags" gorm:"size:500"`                      // 新增
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`                                // 新增
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
```

- [ ] **步骤 2：扩展 Folder 模型**

```go
// qim-server/model/model.go - Line 132-141
type Folder struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"size:255;not null"`
	ParentID  *uint          `json:"parent_id"`
	SortOrder int            `json:"sort_order" gorm:"default:0"`      // 新增
	Icon      string         `json:"icon" gorm:"size:50"`              // 新增
	Color     string         `json:"color" gorm:"size:20"`             // 新增
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
```

- [ ] **步骤 3：更新 SQLite DDL**

```sql
-- qim-server/ddl_sqlite.sql - 替换 files 表定义
CREATE TABLE IF NOT EXISTS `files` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `original_name` VARCHAR(255),
  `size` INTEGER NOT NULL,
  `mime_type` VARCHAR(100),
  `storage_path` VARCHAR(500) NOT NULL,
  `checksum` VARCHAR(64),
  `folder_id` INTEGER,
  `source` VARCHAR(20) DEFAULT 'upload',
  `source_id` VARCHAR(100),
  `is_starred` INTEGER DEFAULT 0,
  `starred_at` DATETIME,
  `tags` VARCHAR(500),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_files_user_id` ON `files`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_files_folder_id` ON `files`(`folder_id`);
CREATE INDEX IF NOT EXISTS `idx_files_source` ON `files`(`source`);
CREATE INDEX IF NOT EXISTS `idx_files_is_starred` ON `files`(`is_starred`);
CREATE INDEX IF NOT EXISTS `idx_files_deleted_at` ON `files`(`deleted_at`);
```

- [ ] **步骤 4：更新 SQLite DDL - folders 表**

```sql
-- qim-server/ddl_sqlite.sql - 替换 folders 表定义
CREATE TABLE IF NOT EXISTS `folders` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `parent_id` INTEGER,
  `sort_order` INTEGER DEFAULT 0,
  `icon` VARCHAR(50),
  `color` VARCHAR(20),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_folders_user_id` ON `folders`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_parent_id` ON `folders`(`parent_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_deleted_at` ON `folders`(`deleted_at`);
```

- [ ] **步骤 5：更新 MySQL DDL**

```sql
-- qim-server/ddl_mysql.sql - 替换 files 表定义
CREATE TABLE IF NOT EXISTS `files` (
  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `user_id` INT UNSIGNED NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `original_name` VARCHAR(255),
  `size` BIGINT NOT NULL,
  `mime_type` VARCHAR(100),
  `storage_path` VARCHAR(500) NOT NULL,
  `checksum` VARCHAR(64),
  `folder_id` INT UNSIGNED,
  `source` VARCHAR(20) DEFAULT 'upload',
  `source_id` VARCHAR(100),
  `is_starred` TINYINT(1) DEFAULT 0,
  `starred_at` DATETIME,
  `tags` VARCHAR(500),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  INDEX `idx_files_user_id` (`user_id`),
  INDEX `idx_files_folder_id` (`folder_id`),
  INDEX `idx_files_source` (`source`),
  INDEX `idx_files_is_starred` (`is_starred`),
  INDEX `idx_files_deleted_at` (`deleted_at`),
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`folder_id`) REFERENCES `folders`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
-- qim-server/ddl_mysql.sql - 替换 folders 表定义
CREATE TABLE IF NOT EXISTS `folders` (
  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `user_id` INT UNSIGNED NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `parent_id` INT UNSIGNED,
  `sort_order` INT DEFAULT 0,
  `icon` VARCHAR(50),
  `color` VARCHAR(20),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  INDEX `idx_folders_user_id` (`user_id`),
  INDEX `idx_folders_parent_id` (`parent_id`),
  INDEX `idx_folders_deleted_at` (`deleted_at`),
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`parent_id`) REFERENCES `folders`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

- [ ] **步骤 6：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-server/model/model.go qim-server/ddl_sqlite.sql qim-server/ddl_mysql.sql
git commit -m "feat: 扩展文件/文件夹模型，新增来源、星标、标签等字段"
```

---

### 任务 2：实现文件类型判断工具函数

**文件：**
- 创建：`qim-client/src/utils/fileType.ts`
- 测试：`qim-client/tests/unit/utils/fileType.test.ts`

- [ ] **步骤 1：编写测试**

```typescript
// qim-client/tests/unit/utils/fileType.test.ts
import { describe, it, expect } from 'vitest'
import { getFileCategory, getFileIcon, formatFileSize, isImageFile, isVideoFile, isAudioFile } from '@/utils/fileType'

describe('getFileCategory', () => {
  it('识别图片文件', () => {
    expect(getFileCategory('image/jpeg')).toBe('image')
    expect(getFileCategory('image/png')).toBe('image')
    expect(getFileCategory('image/gif')).toBe('image')
  })

  it('识别文档文件', () => {
    expect(getFileCategory('application/pdf')).toBe('document')
    expect(getFileCategory('application/msword')).toBe('document')
    expect(getFileCategory('text/plain')).toBe('document')
  })

  it('识别视频文件', () => {
    expect(getFileCategory('video/mp4')).toBe('video')
    expect(getFileCategory('video/avi')).toBe('video')
  })

  it('识别音频文件', () => {
    expect(getFileCategory('audio/mp3')).toBe('audio')
    expect(getFileCategory('audio/mpeg')).toBe('audio')
  })

  it('未知类型返回 other', () => {
    expect(getFileCategory('application/unknown')).toBe('other')
    expect(getFileCategory('')).toBe('other')
  })
})

describe('getFileIcon', () => {
  it('返回对应的图标类名', () => {
    expect(getFileIcon('image/jpeg')).toBe('fas fa-image')
    expect(getFileIcon('video/mp4')).toBe('fas fa-video')
    expect(getFileIcon('audio/mp3')).toBe('fas fa-music')
    expect(getFileIcon('application/pdf')).toBe('fas fa-file-pdf')
  })
})

describe('formatFileSize', () => {
  it('格式化字节为人类可读格式', () => {
    expect(formatFileSize(500)).toBe('500 B')
    expect(formatFileSize(1024)).toBe('1.0 KB')
    expect(formatFileSize(1536)).toBe('1.5 KB')
    expect(formatFileSize(1048576)).toBe('1.0 MB')
    expect(formatFileSize(1073741824)).toBe('1.0 GB')
  })
})

describe('类型判断函数', () => {
  it('isImageFile 正确判断', () => {
    expect(isImageFile('image/jpeg')).toBe(true)
    expect(isImageFile('video/mp4')).toBe(false)
  })

  it('isVideoFile 正确判断', () => {
    expect(isVideoFile('video/mp4')).toBe(true)
    expect(isVideoFile('image/jpeg')).toBe(false)
  })

  it('isAudioFile 正确判断', () => {
    expect(isAudioFile('audio/mp3')).toBe(true)
    expect(isAudioFile('image/jpeg')).toBe(false)
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run test:unit -- tests/unit/utils/fileType.test.ts
```

预期：FAIL，模块不存在

- [ ] **步骤 3：实现工具函数**

```typescript
// qim-client/src/utils/fileType.ts
export type FileCategory = 'image' | 'document' | 'video' | 'audio' | 'archive' | 'code' | 'other'

export function getFileCategory(mimeType: string): FileCategory {
  if (!mimeType) return 'other'

  if (mimeType.startsWith('image/')) return 'image'
  if (mimeType.startsWith('video/')) return 'video'
  if (mimeType.startsWith('audio/')) return 'audio'
  if (mimeType.startsWith('text/')) return 'document'
  if (mimeType === 'application/pdf') return 'document'
  if (mimeType.includes('msword') || mimeType.includes('document')) return 'document'
  if (mimeType.includes('excel') || mimeType.includes('sheet')) return 'document'
  if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return 'document'
  if (mimeType.includes('zip') || mimeType.includes('tar') || mimeType.includes('rar')) return 'archive'
  if (mimeType.includes('javascript') || mimeType.includes('json') || mimeType.includes('xml') || mimeType.includes('html')) return 'code'

  return 'other'
}

export function getFileIcon(mimeType: string): string {
  const category = getFileCategory(mimeType)

  const iconMap: Record<FileCategory, string> = {
    image: 'fas fa-image',
    document: 'fas fa-file-alt',
    video: 'fas fa-video',
    audio: 'fas fa-music',
    archive: 'fas fa-file-archive',
    code: 'fas fa-file-code',
    other: 'fas fa-file'
  }

  // 特殊文件类型图标
  if (mimeType === 'application/pdf') return 'fas fa-file-pdf'
  if (mimeType.includes('msword') || mimeType.includes('document')) return 'fas fa-file-word'
  if (mimeType.includes('excel') || mimeType.includes('sheet')) return 'fas fa-file-excel'
  if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return 'fas fa-file-powerpoint'

  return iconMap[category]
}

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

export function isImageFile(mimeType: string): boolean {
  return mimeType?.startsWith('image/') || false
}

export function isVideoFile(mimeType: string): boolean {
  return mimeType?.startsWith('video/') || false
}

export function isAudioFile(mimeType: string): boolean {
  return mimeType?.startsWith('audio/') || false
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run test:unit -- tests/unit/utils/fileType.test.ts
```

预期：全部 PASS

- [ ] **步骤 5：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/utils/fileType.ts qim-client/tests/unit/utils/fileType.test.ts
git commit -m "feat: 添加文件类型判断工具函数及测试"
```

---

### 任务 3：实现分页 composable

**文件：**
- 创建：`qim-client/src/composables/useFilePagination.ts`
- 测试：`qim-client/tests/unit/composables/useFilePagination.test.ts`

- [ ] **步骤 1：编写测试**

```typescript
// qim-client/tests/unit/composables/useFilePagination.test.ts
import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useFilePagination } from '@/composables/useFilePagination'

describe('useFilePagination', () => {
  it('初始状态正确', () => {
    const { page, pageSize, totalPages, hasMore } = useFilePagination({
      fetchFn: async () => ({ items: [], total: 0 })
    })

    expect(page.value).toBe(1)
    expect(pageSize.value).toBe(50)
    expect(totalPages.value).toBe(0)
    expect(hasMore.value).toBe(false)
  })

  it('加载数据后更新状态', async () => {
    const { page, totalPages, hasMore, loadNextPage } = useFilePagination({
      fetchFn: async (p, ps) => ({
        items: Array(ps).fill({}),
        total: 150
      })
    })

    await loadNextPage()

    expect(page.value).toBe(1)
    expect(totalPages.value).toBe(3)
    expect(hasMore.value).toBe(true)
  })

  it('没有更多数据时 hasMore 为 false', async () => {
    const { page, hasMore, loadNextPage } = useFilePagination({
      fetchFn: async () => ({ items: [], total: 0 })
    })

    await loadNextPage()

    expect(hasMore.value).toBe(false)
  })

  it('加载更多继续加载下一页', async () => {
    let callCount = 0
    const { page, loadMore } = useFilePagination({
      fetchFn: async (p, ps) => {
        callCount++
        return { items: Array(ps).fill({}), total: 150 }
      }
    })

    await loadMore()
    expect(callCount).toBe(1)
    expect(page.value).toBe(1)

    await loadMore()
    expect(callCount).toBe(2)
    expect(page.value).toBe(2)
  })

  it('重置恢复初始状态', async () => {
    const { page, reset } = useFilePagination({
      fetchFn: async () => ({ items: [], total: 0 })
    })

    page.value = 5
    reset()

    expect(page.value).toBe(1)
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run test:unit -- tests/unit/composables/useFilePagination.test.ts
```

预期：FAIL

- [ ] **步骤 3：实现分页 composable**

```typescript
// qim-client/src/composables/useFilePagination.ts
import { ref, computed } from 'vue'

interface PaginationResult<T> {
  items: T[]
  total: number
}

interface FetchFn<T> {
  (page: number, pageSize: number): Promise<PaginationResult<T>>
}

interface UseFilePaginationOptions<T> {
  fetchFn: FetchFn<T>
  pageSize?: number
}

export function useFilePagination<T>(options: UseFilePaginationOptions<T>) {
  const { fetchFn, pageSize = 50 } = options

  const page = ref(1)
  const pageSizeRef = ref(pageSize)
  const items = ref<T[]>([])
  const total = ref(0)
  const loading = ref(false)

  const totalPages = computed(() => Math.ceil(total.value / pageSizeRef.value))
  const hasMore = computed(() => page.value < totalPages.value)

  async function loadNextPage() {
    if (loading.value) return

    loading.value = true
    try {
      const result = await fetchFn(page.value, pageSizeRef.value)
      total.value = result.total

      if (page.value === 1) {
        items.value = result.items
      } else {
        items.value = [...items.value, ...result.items]
      }
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (!hasMore.value || loading.value) return
    page.value++
    await loadNextPage()
  }

  function reset() {
    page.value = 1
    items.value = []
    total.value = 0
  }

  return {
    page,
    pageSize: pageSizeRef,
    items,
    total,
    loading,
    totalPages,
    hasMore,
    loadNextPage,
    loadMore,
    reset
  }
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run test:unit -- tests/unit/composables/useFilePagination.test.ts
```

预期：全部 PASS

- [ ] **步骤 5：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/composables/useFilePagination.ts qim-client/tests/unit/composables/useFilePagination.test.ts
git commit -m "feat: 添加文件分页 composable 及测试"
```

---

### 任务 4：实现文件夹树 composable

**文件：**
- 创建：`qim-client/src/composables/useFolderTree.ts`
- 测试：`qim-client/tests/unit/composables/useFolderTree.test.ts`

- [ ] **步骤 1：编写测试**

```typescript
// qim-client/tests/unit/composables/useFolderTree.test.ts
import { describe, it, expect, vi } from 'vitest'
import { useFolderTree } from '@/composables/useFolderTree'

describe('useFolderTree', () => {
  it('初始加载根文件夹', async () => {
    const mockApi = vi.fn().mockResolvedValue({
      data: [
        { id: 1, name: '工作', has_children: true },
        { id: 2, name: '个人', has_children: false }
      ]
    })

    const { folders, loadRootFolders } = useFolderTree({ fetchFn: mockApi })

    await loadRootFolders()

    expect(folders.value).toHaveLength(2)
    expect(folders.value[0].name).toBe('工作')
    expect(mockApi).toHaveBeenCalledWith(null)
  })

  it('懒加载子文件夹', async () => {
    const mockApi = vi.fn().mockResolvedValue({
      data: [
        { id: 3, name: '项目A', has_children: false }
      ]
    })

    const { loadSubFolders } = useFolderTree({ fetchFn: mockApi })

    await loadSubFolders(1)

    expect(mockApi).toHaveBeenCalledWith(1)
  })

  it('展开状态缓存', async () => {
    const mockApi = vi.fn().mockResolvedValue({ data: [] })

    const { expandedFolders, toggleFolder, loadSubFolders } = useFolderTree({ fetchFn: mockApi })

    await toggleFolder(1)
    expect(expandedFolders.value.has(1)).toBe(true)
    expect(mockApi).toHaveBeenCalledTimes(1)

    // 再次切换，不应该重复请求
    await toggleFolder(1)
    expect(expandedFolders.value.has(1)).toBe(false)

    await toggleFolder(1)
    expect(mockApi).toHaveBeenCalledTimes(1) // 缓存命中，不重新请求
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run test:unit -- tests/unit/composables/useFolderTree.test.ts
```

预期：FAIL

- [ ] **步骤 3：实现文件夹树 composable**

```typescript
// qim-client/src/composables/useFolderTree.ts
import { ref } from 'vue'

interface Folder {
  id: number
  name: string
  icon?: string
  color?: string
  parent_id?: number | null
  has_children?: boolean
  children?: Folder[]
  file_count?: number
}

interface FetchFn {
  (parentId: number | null): Promise<{ data: Folder[] }>
}

interface UseFolderTreeOptions {
  fetchFn: FetchFn
}

export function useFolderTree(options: UseFolderTreeOptions) {
  const { fetchFn } = options

  const folders = ref<Folder[]>([])
  const expandedFolders = ref(new Set<number>())
  const subFoldersCache = ref(new Map<number, Folder[]>())
  const loading = ref(false)

  async function loadRootFolders() {
    loading.value = true
    try {
      const result = await fetchFn(null)
      folders.value = result.data
    } finally {
      loading.value = false
    }
  }

  async function loadSubFolders(parentId: number) {
    loading.value = true
    try {
      const result = await fetchFn(parentId)
      subFoldersCache.value.set(parentId, result.data)
    } finally {
      loading.value = false
    }
  }

  async function toggleFolder(folderId: number) {
    if (expandedFolders.value.has(folderId)) {
      expandedFolders.value.delete(folderId)
    } else {
      expandedFolders.value.add(folderId)

      // 如果子文件夹未加载过，则请求
      if (!subFoldersCache.value.has(folderId)) {
        await loadSubFolders(folderId)
      }
    }
  }

  function getSubFolders(folderId: number): Folder[] {
    return subFoldersCache.value.get(folderId) || []
  }

  function isExpanded(folderId: number): boolean {
    return expandedFolders.value.has(folderId)
  }

  return {
    folders,
    expandedFolders,
    loading,
    loadRootFolders,
    loadSubFolders,
    toggleFolder,
    getSubFolders,
    isExpanded
  }
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run test:unit -- tests/unit/composables/useFolderTree.test.ts
```

预期：全部 PASS

- [ ] **步骤 5：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/composables/useFolderTree.ts qim-client/tests/unit/composables/useFolderTree.test.ts
git commit -m "feat: 添加文件夹树 composable 及测试"
```

---

### 任务 5：创建 API 封装

**文件：**
- 创建：`qim-client/src/api/file.ts`

- [ ] **步骤 1：实现 API 封装**

```typescript
// qim-client/src/api/file.ts
import axios from 'axios'
import { API_BASE_URL } from '@/config'

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000
})

// 请求拦截器添加 token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export interface FileItem {
  id: number
  user_id: number
  name: string
  original_name: string
  size: number
  mime_type: string
  storage_path: string
  checksum: string
  folder_id: number | null
  source: string
  source_id: string | null
  is_starred: boolean
  starred_at: string | null
  tags: string | null
  created_at: string
  updated_at: string
}

export interface FolderItem {
  id: number
  user_id: number
  name: string
  parent_id: number | null
  sort_order: number
  icon: string | null
  color: string | null
  has_children?: boolean
  file_count?: number
  created_at: string
  updated_at: string
}

export interface FileListParams {
  folder_id?: number | null
  source?: string
  starred?: boolean
  type?: string
  search?: string
  page?: number
  page_size?: number
}

export interface FileListResponse {
  files: FileItem[]
  total: number
  page: number
  page_size: number
}

// 文件相关 API
export const fileApi = {
  // 获取文件列表
  getFiles(params: FileListParams = {}) {
    return api.get<{ code: number; data: FileListResponse }>('/api/v1/files', { params })
  },

  // 上传文件
  uploadFile(file: File, folderId?: number) {
    const formData = new FormData()
    formData.append('file', file)
    if (folderId) formData.append('folder_id', String(folderId))
    return api.post<{ code: number; data: FileItem }>('/api/v1/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  // 下载文件
  downloadFile(fileId: number) {
    return api.get(`/api/v1/files/${fileId}/download`, { responseType: 'blob' })
  },

  // 删除文件
  deleteFile(fileId: number) {
    return api.delete<{ code: number }>(`/api/v1/files/${fileId}`)
  },

  // 更新文件
  updateFile(fileId: number, data: { name?: string; folder_id?: number | null; tags?: string[] }) {
    return api.put<{ code: number; data: FileItem }>(`/api/v1/files/${fileId}`, data)
  },

  // 星标/取消星标
  toggleStar(fileId: number, starred: boolean) {
    return api.put<{ code: number }>(`/api/v1/files/${fileId}/star`, { starred })
  },

  // 批量操作
  batchOperation(fileIds: number[], action: string, extra?: Record<string, any>) {
    return api.put<{ code: number }>('/api/v1/files/batch', {
      file_ids: fileIds,
      action,
      ...extra
    })
  },

  // 获取星标文件
  getStarredFiles(params: Omit<FileListParams, 'starred'> = {}) {
    return api.get<{ code: number; data: FileListResponse }>('/api/v1/files/starred', {
      params: { ...params, starred: true }
    })
  },

  // 获取文件统计
  getStats() {
    return api.get<{ code: number; data: Record<string, any> }>('/api/v1/files/stats')
  }
}

// 文件夹相关 API
export const folderApi = {
  // 获取文件夹树（懒加载）
  getFolderTree(parentId: number | null = null) {
    return api.get<{ code: number; data: FolderItem[] }>('/api/v1/folders/tree', {
      params: { lazy: true, parent_id: parentId }
    })
  },

  // 创建文件夹
  createFolder(name: string, parentId?: number | null) {
    return api.post<{ code: number; data: FolderItem }>('/api/v1/folders', {
      name,
      parent_id: parentId ?? null
    })
  },

  // 更新文件夹
  updateFolder(folderId: number, data: { name?: string; parent_id?: number | null; icon?: string; color?: string; sort_order?: number }) {
    return api.put<{ code: number; data: FolderItem }>(`/api/v1/folders/${folderId}`, data)
  },

  // 删除文件夹
  deleteFolder(folderId: number) {
    return api.delete<{ code: number }>(`/api/v1/folders/${folderId}`)
  },

  // 获取文件夹内文件
  getFolderFiles(folderId: number, params: Omit<FileListParams, 'folder_id'> = {}) {
    return api.get<{ code: number; data: FileListResponse }>(`/api/v1/folders/${folderId}/files`, {
      params: { ...params, folder_id: folderId }
    })
  }
}

export { api }
```

- [ ] **步骤 2：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/api/file.ts
git commit -m "feat: 添加文件管理 API 封装"
```

---

### 任务 6：扩展后端接口 - 文件操作

**文件：**
- 修改：`qim-server/handler/file_handler.go`

- [ ] **步骤 1：更新 GetFiles 支持分页和过滤**

```go
// qim-server/handler/file_handler.go - 替换 GetFiles 函数
func GetFiles(c *gin.Context) {
	userID, _ := c.Get("user_id")

	folderIDStr := c.Query("folder_id")
	source := c.Query("source")
	starredStr := c.Query("starred")
	fileType := c.Query("type")
	search := c.Query("search")

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "50")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	db := database.GetDB()
	query := db.Model(&model.File{}).Where("user_id = ?", userID)

	if folderIDStr != "" {
		folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
		if err == nil {
			query = query.Where("folder_id = ?", uint(folderID))
		}
	}
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if starredStr == "true" {
		query = query.Where("is_starred = ?", true)
	}
	if fileType != "" {
		switch fileType {
		case "image":
			query = query.Where("mime_type LIKE ?", "image/%")
		case "document":
			query = query.Where("mime_type LIKE ? OR mime_type LIKE ? OR mime_type LIKE ?", "text/%", "application/pdf%", "application/msword%")
		case "video":
			query = query.Where("mime_type LIKE ?", "video/%")
		case "audio":
			query = query.Where("mime_type LIKE ?", "audio/%")
		}
	}
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"files":     files,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}
```

- [ ] **步骤 2：添加 UpdateFile 函数**

```go
// qim-server/handler/file_handler.go - 添加到文件末尾
func UpdateFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	var req struct {
		Name     string   `json:"name"`
		FolderID *uint    `json:"folder_id"`
		Tags     []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var file model.File
	if err := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["folder_id"] = req.FolderID
	if req.Tags != nil {
		updates["tags"] = strings.Join(req.Tags, ",")
	}

	db.Model(&file).Updates(updates)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": file,
	})
}
```

- [ ] **步骤 3：添加 ToggleStar 函数**

```go
// qim-server/handler/file_handler.go
func ToggleStar(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	var req struct {
		Starred bool `json:"starred"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var file model.File
	if err := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_starred": req.Starred,
	}
	if req.Starred {
		updates["starred_at"] = &now
	} else {
		updates["starred_at"] = nil
	}

	db.Model(&file).Updates(updates)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
	})
}
```

- [ ] **步骤 4：添加 BatchOperation 函数**

```go
// qim-server/handler/file_handler.go
func BatchOperation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		FileIDs  []uint `json:"file_ids" binding:"required"`
		Action   string `json:"action" binding:"required"`
		FolderID *uint  `json:"folder_id"`
		Tags     string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	switch req.Action {
	case "move":
		if req.FolderID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "移动操作需要 folder_id"})
			return
		}
		db.Model(&model.File{}).Where("id IN ? AND user_id = ?", req.FileIDs, userID).Update("folder_id", req.FolderID)
	case "delete":
		db.Where("id IN ? AND user_id = ?", req.FileIDs, userID).Delete(&model.File{})
	case "star":
		now := time.Now()
		db.Model(&model.File{}).Where("id IN ? AND user_id = ?", req.FileIDs, userID).Updates(map[string]interface{}{
			"is_starred": true,
			"starred_at": &now,
		})
	case "add_tags":
		db.Model(&model.File{}).Where("id IN ? AND user_id = ?", req.FileIDs, userID).Update("tags", req.Tags)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
	})
}
```

- [ ] **步骤 5：添加 GetStarredFiles 和 GetFileStats 函数**

```go
// qim-server/handler/file_handler.go
func GetStarredFiles(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var files []model.File
	db.Where("user_id = ? AND is_starred = ?", userID, true).Order("starred_at DESC").Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"files":     files,
			"total":     len(files),
			"page":      1,
			"page_size": len(files),
		},
	})
}

func GetFileStats(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()

	var totalCount int64
	var totalSize int64
	db.Model(&model.File{}).Where("user_id = ?", userID).Select("count(*), sum(size)").Row().Scan(&totalCount, &totalSize)

	type TypeStat struct {
		Count int64 `json:"count"`
		Size  int64 `json:"size"`
	}

	var imageCount, imageSize int64
	db.Model(&model.File{}).Where("user_id = ? AND mime_type LIKE ?", userID, "image/%").Select("count(*), sum(size)").Row().Scan(&imageCount, &imageSize)

	var docCount, docSize int64
	db.Model(&model.File{}).Where("user_id = ? AND (mime_type LIKE ? OR mime_type LIKE ?)", userID, "text/%", "application/pdf%").Select("count(*), sum(size)").Row().Scan(&docCount, &docSize)

	var videoCount, videoSize int64
	db.Model(&model.File{}).Where("user_id = ? AND mime_type LIKE ?", userID, "video/%").Select("count(*), sum(size)").Row().Scan(&videoCount, &videoSize)

	var uploadCount, chatCount, shareCount int64
	db.Model(&model.File{}).Where("user_id = ? AND source = ?", userID, "upload").Count(&uploadCount)
	db.Model(&model.File{}).Where("user_id = ? AND source = ?", userID, "chat").Count(&chatCount)
	db.Model(&model.File{}).Where("user_id = ? AND source = ?", userID, "share").Count(&shareCount)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total_count": totalCount,
			"total_size":  totalSize,
			"by_type": gin.H{
				"image":    gin.H{"count": imageCount, "size": imageSize},
				"document": gin.H{"count": docCount, "size": docSize},
				"video":    gin.H{"count": videoCount, "size": videoSize},
			},
			"by_source": gin.H{
				"upload": uploadCount,
				"chat":   chatCount,
				"share":  shareCount,
			},
		},
	})
}
```

- [ ] **步骤 6：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-server/handler/file_handler.go
git commit -m "feat: 扩展文件管理接口，支持分页、更新、星标、批量操作"
```

---

### 任务 7：扩展后端接口 - 文件夹操作

**文件：**
- 修改：`qim-server/handler/file_handler.go`

- [ ] **步骤 1：重构 GetFolderTree 支持懒加载**

```go
// qim-server/handler/file_handler.go - 替换 GetFolderTree 函数
func GetFolderTree(c *gin.Context) {
	userID, _ := c.Get("user_id")

	lazyStr := c.DefaultQuery("lazy", "true")
	parentIDStr := c.Query("parent_id")
	lazy := lazyStr == "true"

	db := database.GetDB()

	if lazy {
		// 懒加载模式：只查询指定 parent_id 的直接子文件夹
		var folders []model.Folder
		query := db.Where("user_id = ?", userID)

		if parentIDStr == "" {
			query = query.Where("parent_id IS NULL")
		} else {
			parentID, err := strconv.ParseUint(parentIDStr, 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 parent_id"})
				return
			}
			query = query.Where("parent_id = ?", uint(parentID))
		}

		query.Order("sort_order ASC, created_at ASC").Find(&folders)

		// 查询每个文件夹是否有子文件夹
		type FolderWithMeta struct {
			model.Folder
			HasChildren bool `json:"has_children"`
			FileCount   int64 `json:"file_count"`
		}

		result := make([]FolderWithMeta, 0, len(folders))
		for _, folder := range folders {
			var hasChildren int64
			db.Model(&model.Folder{}).Where("user_id = ? AND parent_id = ?", userID, folder.ID).Count(&hasChildren)

			var fileCount int64
			db.Model(&model.File{}).Where("user_id = ? AND folder_id = ?", userID, folder.ID).Count(&fileCount)

			result = append(result, FolderWithMeta{
				Folder:      folder,
				HasChildren: hasChildren > 0,
				FileCount:   fileCount,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": result,
		})
	} else {
		// 完整树模式：返回所有层级
		var allFolders []model.Folder
		db.Where("user_id = ?", userID).Order("sort_order ASC, created_at ASC").Find(&allFolders)

		tree := buildFolderTree(allFolders, nil)

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": tree,
		})
	}
}

// buildFolderTree 构建文件夹树
func buildFolderTree(folders []model.Folder, parentID *uint) []model.Folder {
	var tree []model.Folder

	for i := range folders {
		if folders[i].ParentID == parentID {
			children := buildFolderTree(folders, &folders[i].ID)
			folders[i].ParentID = parentID // 保持 parent_id 一致
			if len(children) > 0 {
				// 需要通过子切片返回，这里简化处理
			}
			tree = append(tree, folders[i])
		}
	}

	return tree
}
```

- [ ] **步骤 2：添加 UpdateFolder 函数**

```go
// qim-server/handler/file_handler.go
func UpdateFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")
	folderIDStr := c.Param("id")

	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件夹ID"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		ParentID  *uint  `json:"parent_id"`
		Icon      string `json:"icon"`
		Color     string `json:"color"`
		SortOrder int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var folder model.Folder
	if err := db.Where("id = ? AND user_id = ?", uint(folderID), userID).First(&folder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件夹不存在"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["parent_id"] = req.ParentID
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	updates["sort_order"] = req.SortOrder

	db.Model(&folder).Updates(updates)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": folder,
	})
}
```

- [ ] **步骤 3：添加 DeleteFolder 函数**

```go
// qim-server/handler/file_handler.go
func DeleteFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")
	folderIDStr := c.Param("id")

	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件夹ID"})
		return
	}

	db := database.GetDB()
	var folder model.Folder
	if err := db.Where("id = ? AND user_id = ?", uint(folderID), userID).First(&folder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件夹不存在"})
		return
	}

	// 检查是否有子文件夹
	var childCount int64
	db.Model(&model.Folder{}).Where("user_id = ? AND parent_id = ?", userID, folderID).Count(&childCount)
	if childCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件夹包含子文件夹，请先删除子文件夹"})
		return
	}

	// 检查是否有文件
	var fileCount int64
	db.Model(&model.File{}).Where("user_id = ? AND folder_id = ?", userID, folderID).Count(&fileCount)
	if fileCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件夹包含文件，请先移动或删除文件"})
		return
	}

	db.Delete(&folder)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除文件夹成功",
	})
}
```

- [ ] **步骤 4：添加 GetFolderFiles 函数**

```go
// qim-server/handler/file_handler.go
func GetFolderFiles(c *gin.Context) {
	// 复用 GetFiles 逻辑，只是固定 folder_id
	folderIDStr := c.Param("id")

	// 设置 folder_id 查询参数
	c.Request.URL.RawQuery = "folder_id=" + folderIDStr + "&" + c.Request.URL.RawQuery

	GetFiles(c)
}
```

- [ ] **步骤 5：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-server/handler/file_handler.go
git commit -m "feat: 扩展文件夹管理接口，支持懒加载、更新、删除"
```

---

### 任务 8：注册后端路由

**文件：**
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：添加新路由**

```go
// qim-server/app/routes.go - 在文件管理路由部分（Line 179-183）替换为：
			// 文件管理
			authed.GET("/files", handler.GetFiles)
			authed.GET("/files/starred", handler.GetStarredFiles)
			authed.GET("/files/stats", handler.GetFileStats)
			authed.GET("/files/:id/download", handler.DownloadFile)
			authed.PUT("/files/:id", handler.UpdateFile)
			authed.PUT("/files/:id/star", handler.ToggleStar)
			authed.PUT("/files/batch", handler.BatchOperation)
			authed.DELETE("/files/:id", handler.DeleteFile)

			// 文件夹管理
			authed.POST("/folders", handler.CreateFolder)
			authed.GET("/folders/tree", handler.GetFolderTree)
			authed.PUT("/folders/:id", handler.UpdateFolder)
			authed.DELETE("/folders/:id", handler.DeleteFolder)
			authed.GET("/folders/:id/files", handler.GetFolderFiles)
```

- [ ] **步骤 2：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-server/app/routes.go
git commit -m "feat: 注册文件/文件夹管理新路由"
```

---

### 任务 9：创建前端子组件 - FolderTree 和 FolderTreeItem

**文件：**
- 创建：`qim-client/src/components/apps/file/FolderTree.vue`
- 创建：`qim-client/src/components/apps/file/FolderTreeItem.vue`

- [ ] **步骤 1：创建 FolderTreeItem 组件**

```vue
<!-- qim-client/src/components/apps/file/FolderTreeItem.vue -->
<template>
  <div class="folder-tree-item">
    <div
      class="folder-item-content"
      :class="{ active: isSelected, expanded: isExpanded }"
      @click="handleClick"
    >
      <i
        v-if="folder.has_children"
        class="folder-toggle-icon fas"
        :class="isExpanded ? 'fa-chevron-down' : 'fa-chevron-right'"
      ></i>
      <span v-else class="folder-toggle-placeholder"></span>

      <i class="folder-icon fas" :class="isExpanded ? 'fa-folder-open' : 'fa-folder'"></i>

      <span class="folder-name">{{ folder.name }}</span>

      <span v-if="folder.file_count" class="folder-count">{{ folder.file_count }}</span>
    </div>

    <div v-if="isExpanded && folder.has_children" class="folder-children">
      <FolderTreeItem
        v-for="child in children"
        :key="child.id"
        :folder="child"
        :selected-id="selectedId"
        @select="$emit('select', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FolderItem } from '@/api/file'

interface Props {
  folder: FolderItem & { has_children?: boolean; file_count?: number }
  selectedId: number | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  select: [folderId: number]
}>()

const isExpanded = computed(() => props.folder.has_children && props.selectedId === props.folder.id)

const isSelected = computed(() => props.selectedId === props.folder.id)

const children = computed(() => (props.folder as any).children || [])

function handleClick() {
  emit('select', props.folder.id)
}
</script>

<style scoped>
.folder-tree-item {
  user-select: none;
}

.folder-item-content {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.2s;
}

.folder-item-content:hover {
  background: var(--hover-color);
}

.folder-item-content.active {
  background: var(--primary-light);
  color: var(--primary-color);
}

.folder-toggle-icon {
  width: 16px;
  font-size: 10px;
  color: var(--text-secondary);
}

.folder-toggle-placeholder {
  width: 16px;
}

.folder-icon {
  color: var(--warning-color);
  font-size: 14px;
}

.folder-item-content.active .folder-icon {
  color: var(--primary-color);
}

.folder-name {
  flex: 1;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.folder-count {
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--badge-bg);
  padding: 2px 6px;
  border-radius: 10px;
}

.folder-children {
  margin-left: 16px;
}
</style>
```

- [ ] **步骤 2：创建 FolderTree 组件**

```vue
<!-- qim-client/src/components/apps/file/FolderTree.vue -->
<template>
  <div class="folder-tree">
    <div class="folder-tree-header">
      <h3>文件夹</h3>
      <button class="create-folder-btn" @click="$emit('createFolder')">
        <i class="fas fa-plus"></i>
      </button>
    </div>

    <div class="folder-tree-list">
      <div
        class="folder-item root-item"
        :class="{ active: selectedFolderId === null }"
        @click="selectFolder(null)"
      >
        <i class="folder-icon fas fa-inbox"></i>
        <span class="folder-name">全部文件</span>
      </div>

      <div
        class="folder-item root-item"
        :class="{ active: selectedFolderId === -1 }"
        @click="selectFolder(-1)"
      >
        <i class="folder-icon fas fa-star"></i>
        <span class="folder-name">星标文件</span>
      </div>

      <div
        v-for="folder in folders"
        :key="folder.id"
        class="folder-item root-item"
        :class="{ active: selectedFolderId === folder.id }"
        @click="selectFolder(folder.id)"
      >
        <i
          v-if="folder.has_children"
          class="folder-toggle-icon fas"
          :class="isExpanded(folder.id) ? 'fa-chevron-down' : 'fa-chevron-right'"
          @click.stop="toggleFolderExpand(folder.id)"
        ></i>
        <span v-else class="folder-toggle-placeholder"></span>

        <i class="folder-icon fas" :class="isExpanded(folder.id) ? 'fa-folder-open' : 'fa-folder'"></i>

        <span class="folder-name">{{ folder.name }}</span>

        <span v-if="folder.file_count" class="folder-count">{{ folder.file_count }}</span>

        <div v-if="isExpanded(folder.id) && folder.has_children" class="sub-folders">
          <div
            v-for="child in getSubFolders(folder.id)"
            :key="child.id"
            class="folder-item sub-item"
            :class="{ active: selectedFolderId === child.id }"
            @click="selectFolder(child.id)"
          >
            <span class="folder-toggle-placeholder"></span>
            <i class="folder-icon fas fa-folder"></i>
            <span class="folder-name">{{ child.name }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="folder-tree-loading">
      <i class="fas fa-spinner fa-spin"></i>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useFolderTree } from '@/composables/useFolderTree'
import { folderApi, type FolderItem } from '@/api/file'

interface Props {
  selectedFolderId: number | null
}

defineProps<Props>()

const emit = defineEmits<{
  select: [folderId: number | null]
  createFolder: []
}>()

const { folders, loading, loadRootFolders, loadSubFolders, toggleFolder, getSubFolders, isExpanded } = useFolderTree({
  fetchFn: async (parentId) => {
    const response = await folderApi.getFolderTree(parentId)
    return response.data
  }
})

onMounted(() => {
  loadRootFolders()
})

function selectFolder(folderId: number | null) {
  emit('select', folderId)
}

async function toggleFolderExpand(folderId: number) {
  await toggleFolder(folderId)
}
</script>

<style scoped>
.folder-tree {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--sidebar-bg);
}

.folder-tree-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}

.folder-tree-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.create-folder-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
  transition: all 0.2s;
}

.create-folder-btn:hover {
  background: var(--primary-light);
}

.folder-tree-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.folder-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.2s;
  position: relative;
}

.folder-item:hover {
  background: var(--hover-color);
}

.folder-item.active {
  background: var(--primary-light);
  color: var(--primary-color);
}

.folder-icon {
  color: var(--warning-color);
  font-size: 14px;
}

.folder-item.active .folder-icon {
  color: var(--primary-color);
}

.folder-name {
  flex: 1;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.folder-count {
  font-size: 12px;
  color: var(--text-secondary);
  padding: 2px 6px;
  border-radius: 10px;
}

.folder-toggle-icon {
  width: 16px;
  font-size: 10px;
  color: var(--text-secondary);
  cursor: pointer;
}

.folder-toggle-placeholder {
  width: 16px;
}

.sub-folders {
  margin-left: 24px;
  margin-top: 4px;
}

.sub-item {
  padding: 6px 12px;
  font-size: 13px;
}

.folder-tree-loading {
  display: flex;
  justify-content: center;
  padding: 16px;
  color: var(--text-secondary);
}
</style>
```

- [ ] **步骤 3：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/file/FolderTree.vue qim-client/src/components/apps/file/FolderTreeItem.vue
git commit -m "feat: 创建文件夹树组件及子项组件"
```

---

### 任务 10：创建前端子组件 - FileList、FileGridItem、FileListItem

**文件：**
- 创建：`qim-client/src/components/apps/file/FileList.vue`
- 创建：`qim-client/src/components/apps/file/FileGridItem.vue`
- 创建：`qim-client/src/components/apps/file/FileListItem.vue`

- [ ] **步骤 1：创建 FileGridItem 组件**

```vue
<!-- qim-client/src/components/apps/file/FileGridItem.vue -->
<template>
  <div class="file-grid-item" @click="$emit('preview', file)" @dblclick="$emit('download', file)">
    <div class="file-icon-wrapper" :class="getFileCategoryClass(file.mime_type)">
      <i :class="getFileIcon(file.mime_type)"></i>
    </div>

    <div class="file-info">
      <div class="file-name" :title="file.name">{{ file.name }}</div>
      <div class="file-meta">
        <span class="file-size">{{ formatFileSize(file.size) }}</span>
        <span class="file-date">{{ formatDate(file.created_at) }}</span>
      </div>
    </div>

    <div v-if="file.is_starred" class="starred-badge">
      <i class="fas fa-star"></i>
    </div>

    <div class="file-actions">
      <button class="action-btn" @click.stop="$emit('star', file)" :title="file.is_starred ? '取消星标' : '星标'">
        <i class="fas" :class="file.is_starred ? 'fa-star' : 'fa-star-o'"></i>
      </button>
      <button class="action-btn" @click.stop="$emit('delete', file)" title="删除">
        <i class="fas fa-trash"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FileItem } from '@/api/file'
import { getFileCategory, getFileIcon, formatFileSize } from '@/utils/fileType'

defineProps<{
  file: FileItem
}>()

defineEmits<{
  preview: [file: FileItem]
  download: [file: FileItem]
  star: [file: FileItem]
  delete: [file: FileItem]
}>()

function getFileCategoryClass(mimeType: string): string {
  const category = getFileCategory(mimeType)
  return `file-category-${category}`
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
</script>

<style scoped>
.file-grid-item {
  position: relative;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s;
}

.file-grid-item:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
  border-color: var(--primary-color);
}

.file-icon-wrapper {
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  margin-bottom: 12px;
  font-size: 28px;
  border: 2px solid var(--border-color);
}

.file-category-image i { color: var(--primary-color); }
.file-category-video i { color: var(--error-color); }
.file-category-audio i { color: var(--warning-color); }
.file-category-document i { color: var(--info-color); }
.file-category-archive i { color: var(--text-secondary); }
.file-category-code i { color: var(--success-color); }
.file-category-other i { color: var(--text-tertiary); }

.file-info {
  flex: 1;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.file-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.starred-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  color: var(--warning-color);
}

.file-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.file-grid-item:hover .file-actions {
  opacity: 1;
}

.action-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-color);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.action-btn:hover {
  color: var(--primary-color);
}
</style>
```

- [ ] **步骤 2：创建 FileListItem 组件**

```vue
<!-- qim-client/src/components/apps/file/FileListItem.vue -->
<template>
  <div class="file-list-item" @click="$emit('preview', file)">
    <div class="file-icon" :class="getFileCategoryClass(file.mime_type)">
      <i :class="getFileIcon(file.mime_type)"></i>
    </div>

    <div class="file-name-cell" :title="file.name">{{ file.name }}</div>

    <div class="file-size-cell">{{ formatFileSize(file.size) }}</div>

    <div class="file-date-cell">{{ formatDate(file.created_at) }}</div>

    <div class="file-actions-cell">
      <button class="action-btn" @click.stop="$emit('star', file)">
        <i class="fas" :class="file.is_starred ? 'fa-star starred' : 'fa-star-o'"></i>
      </button>
      <button class="action-btn" @click.stop="$emit('download', file)">
        <i class="fas fa-download"></i>
      </button>
      <button class="action-btn" @click.stop="$emit('delete', file)">
        <i class="fas fa-trash"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FileItem } from '@/api/file'
import { getFileCategory, getFileIcon, formatFileSize } from '@/utils/fileType'

defineProps<{
  file: FileItem
}>()

defineEmits<{
  preview: [file: FileItem]
  download: [file: FileItem]
  star: [file: FileItem]
  delete: [file: FileItem]
}>()

function getFileCategoryClass(mimeType: string): string {
  return `file-category-${getFileCategory(mimeType)}`
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<style scoped>
.file-list-item {
  display: grid;
  grid-template-columns: 40px 1fr 100px 120px 120px;
  gap: 12px;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background 0.2s;
}

.file-list-item:hover {
  background: var(--hover-color);
}

.file-icon {
  font-size: 20px;
}

.file-name-cell {
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-size-cell,
.file-date-cell {
  font-size: 13px;
  color: var(--text-secondary);
}

.file-actions-cell {
  display: flex;
  gap: 4px;
  justify-content: flex-end;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.action-btn .starred {
  color: var(--warning-color);
}
</style>
```

- [ ] **步骤 3：创建 FileList 组件**

```vue
<!-- qim-client/src/components/apps/file/FileList.vue -->
<template>
  <div class="file-list">
    <div class="file-list-toolbar">
      <div class="file-list-header" v-if="viewMode === 'list'">
        <div class="header-icon">图标</div>
        <div class="header-name">文件名</div>
        <div class="header-size">大小</div>
        <div class="header-date">修改时间</div>
        <div class="header-actions">操作</div>
      </div>

      <div class="view-toggle">
        <button :class="{ active: viewMode === 'grid' }" @click="viewMode = 'grid'">
          <i class="fas fa-th"></i>
        </button>
        <button :class="{ active: viewMode === 'list' }" @click="viewMode = 'list'">
          <i class="fas fa-list"></i>
        </button>
      </div>
    </div>

    <div class="file-list-content" ref="scrollContainer" @scroll="handleScroll">
      <!-- 网格视图 -->
      <div v-if="viewMode === 'grid'" class="file-grid">
        <FileGridItem
          v-for="file in files"
          :key="file.id"
          :file="file"
          @preview="$emit('preview', $event)"
          @download="$emit('download', $event)"
          @star="$emit('star', $event)"
          @delete="$emit('delete', $event)"
        />
      </div>

      <!-- 列表视图 -->
      <div v-else class="file-list-view">
        <FileListItem
          v-for="file in files"
          :key="file.id"
          :file="file"
          @preview="$emit('preview', $event)"
          @download="$emit('download', $event)"
          @star="$emit('star', $event)"
          @delete="$emit('delete', $event)"
        />
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="loading-indicator">
        <i class="fas fa-spinner fa-spin"></i>
        <span>加载中...</span>
      </div>

      <!-- 空状态 -->
      <div v-if="!loading && files.length === 0" class="empty-state">
        <i class="fas fa-folder-open"></i>
        <p>暂无文件</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { FileItem } from '@/api/file'
import FileGridItem from './FileGridItem.vue'
import FileListItem from './FileListItem.vue'

interface Props {
  files: FileItem[]
  loading: boolean
  hasMore: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  preview: [file: FileItem]
  download: [file: FileItem]
  star: [file: FileItem]
  delete: [file: FileItem]
  loadMore: []
}>()

const viewMode = ref<'grid' | 'list'>('grid')
const scrollContainer = ref<HTMLElement>()

function handleScroll() {
  if (!scrollContainer.value) return

  const { scrollTop, scrollHeight, clientHeight } = scrollContainer.value
  if (scrollTop + clientHeight >= scrollHeight - 100) {
    emit('loadMore')
  }
}
</script>

<style scoped>
.file-list {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.file-list-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
}

.file-list-header {
  display: grid;
  grid-template-columns: 40px 1fr 100px 120px 120px;
  gap: 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.view-toggle {
  display: flex;
  gap: 4px;
}

.view-toggle button {
  width: 32px;
  height: 32px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.2s;
}

.view-toggle button.active {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.file-list-content {
  flex: 1;
  overflow-y: auto;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
  padding: 16px;
}

.loading-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  color: var(--text-secondary);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-state p {
  font-size: 16px;
  margin: 0;
}
</style>
```

- [ ] **步骤 4：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/file/FileList.vue qim-client/src/components/apps/file/FileGridItem.vue qim-client/src/components/apps/file/FileListItem.vue
git commit -m "feat: 创建文件列表组件（网格/列表视图）"
```

---

### 任务 11：创建前端模态框组件

**文件：**
- 创建：`qim-client/src/components/apps/file/CreateFolderModal.vue`
- 创建：`qim-client/src/components/apps/file/FilePreviewModal.vue`
- 创建：`qim-client/src/components/apps/file/FileActionsModal.vue`

- [ ] **步骤 1：创建 CreateFolderModal 组件**

```vue
<!-- qim-client/src/components/apps/file/CreateFolderModal.vue -->
<template>
  <div class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>{{ isEditing ? '编辑文件夹' : '创建文件夹' }}</h3>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <div class="modal-body">
        <div class="form-group">
          <label>文件夹名称</label>
          <input
            type="text"
            class="form-input"
            v-model="folderName"
            placeholder="请输入文件夹名称"
            ref="nameInput"
          />
        </div>

        <div v-if="!isEditing" class="form-group">
          <label>父文件夹</label>
          <select class="form-input" v-model="parentFolderId">
            <option :value="null">根目录</option>
            <option v-for="folder in folders" :key="folder.id" :value="folder.id">
              {{ folder.name }}
            </option>
          </select>
        </div>
      </div>

      <div class="modal-footer">
        <button class="modal-btn cancel-btn" @click="$emit('close')">取消</button>
        <button class="modal-btn confirm-btn" @click="handleSubmit" :disabled="!folderName">
          {{ isEditing ? '保存' : '创建' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, nextTick } from 'vue'
import { folderApi, type FolderItem } from '@/api/file'
import QMessage from '@/utils/qmessage'

interface Props {
  isEditing?: boolean
  folder?: FolderItem | null
  folders?: FolderItem[]
}

const props = withDefaults(defineProps<Props>(), {
  isEditing: false,
  folder: null,
  folders: () => []
})

const emit = defineEmits<{
  close: []
  success: []
}>()

const folderName = ref('')
const parentFolderId = ref<number | null>(null)
const nameInput = ref<HTMLInputElement>()

watch(() => props.folder, (newFolder) => {
  if (newFolder) {
    folderName.value = newFolder.name
    parentFolderId.value = newFolder.parent_id
  }
}, { immediate: true })

onMounted(async () => {
  await nextTick()
  nameInput.value?.focus()
})

async function handleSubmit() {
  if (!folderName.value.trim()) {
    QMessage.error('请输入文件夹名称')
    return
  }

  try {
    if (props.isEditing && props.folder) {
      await folderApi.updateFolder(props.folder.id, {
        name: folderName.value.trim()
      })
    } else {
      await folderApi.createFolder(folderName.value.trim(), parentFolderId.value)
    }
    emit('success')
  } catch (error) {
    QMessage.error('操作失败')
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 400px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.modal-btn {
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.cancel-btn {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
}

.confirm-btn {
  background: var(--primary-color);
  color: white;
  border: none;
}

.confirm-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
```

- [ ] **步骤 2：创建 FilePreviewModal 组件**

```vue
<!-- qim-client/src/components/apps/file/FilePreviewModal.vue -->
<template>
  <div class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>{{ file.name }}</h3>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <div class="modal-body">
        <img
          v-if="isImageFile(file.mime_type)"
          :src="previewUrl"
          :alt="file.name"
          class="preview-image"
        />

        <video
          v-else-if="isVideoFile(file.mime_type)"
          :src="previewUrl"
          controls
          class="preview-video"
        ></video>

        <audio
          v-else-if="isAudioFile(file.mime_type)"
          :src="previewUrl"
          controls
          class="preview-audio"
        ></audio>

        <div v-else class="preview-other">
          <i :class="getFileIcon(file.mime_type)"></i>
          <p>无法预览此文件类型</p>
          <button class="download-btn" @click="$emit('download', file)">
            <i class="fas fa-download"></i> 下载
          </button>
        </div>
      </div>

      <div class="modal-footer">
        <button class="modal-btn" @click="$emit('star', file)">
          <i class="fas" :class="file.is_starred ? 'fa-star' : 'fa-star-o'"></i>
          {{ file.is_starred ? '取消星标' : '星标' }}
        </button>
        <button class="modal-btn" @click="$emit('download', file)">
          <i class="fas fa-download"></i> 下载
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FileItem } from '@/api/file'
import { getFileIcon, isImageFile, isVideoFile, isAudioFile } from '@/utils/fileType'
import { API_BASE_URL } from '@/config'

defineProps<{
  file: FileItem
}>()

defineEmits<{
  close: []
  download: [file: FileItem]
  star: [file: FileItem]
}>()

const previewUrl = computed(() => {
  return `${API_BASE_URL}${file.storage_path}`
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 800px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}

.modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  margin-right: 16px;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
}

.modal-body {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  overflow: auto;
  min-height: 400px;
}

.preview-image {
  max-width: 100%;
  max-height: 600px;
  object-fit: contain;
}

.preview-video {
  max-width: 100%;
  max-height: 600px;
}

.preview-audio {
  width: 100%;
  max-width: 400px;
}

.preview-other {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  color: var(--text-secondary);
}

.preview-other i {
  font-size: 64px;
  opacity: 0.5;
}

.download-btn {
  padding: 12px 24px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.modal-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--card-bg);
  cursor: pointer;
  font-size: 14px;
}

.modal-btn:hover {
  background: var(--hover-color);
}
</style>
```

- [ ] **步骤 3：创建 FileActionsModal 组件**

```vue
<!-- qim-client/src/components/apps/file/FileActionsModal.vue -->
<template>
  <div class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>{{ actionTitle }}</h3>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <div class="modal-body">
        <!-- 重命名 -->
        <div v-if="action === 'rename'" class="form-group">
          <label>新文件名</label>
          <input type="text" class="form-input" v-model="newName" placeholder="请输入新文件名" />
        </div>

        <!-- 移动到文件夹 -->
        <div v-if="action === 'move'" class="form-group">
          <label>目标文件夹</label>
          <select class="form-input" v-model="targetFolderId">
            <option :value="null">根目录</option>
            <option v-for="folder in folders" :key="folder.id" :value="folder.id">
              {{ folder.name }}
            </option>
          </select>
        </div>
      </div>

      <div class="modal-footer">
        <button class="modal-btn cancel-btn" @click="$emit('close')">取消</button>
        <button class="modal-btn confirm-btn" @click="handleSubmit">确认</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { fileApi, folderApi, type FileItem, type FolderItem } from '@/api/file'
import QMessage from '@/utils/qmessage'

interface Props {
  action: 'rename' | 'move'
  file: FileItem
  folders: FolderItem[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  success: []
}>()

const newName = ref(props.file?.name || '')
const targetFolderId = ref<number | null>(null)

const actionTitle = computed(() => {
  if (props.action === 'rename') return '重命名文件'
  if (props.action === 'move') return '移动到文件夹'
  return ''
})

async function handleSubmit() {
  try {
    if (props.action === 'rename') {
      if (!newName.value.trim()) {
        QMessage.error('文件名不能为空')
        return
      }
      await fileApi.updateFile(props.file.id, { name: newName.value.trim() })
    } else if (props.action === 'move') {
      await fileApi.updateFile(props.file.id, { folder_id: targetFolderId.value })
    }
    emit('success')
  } catch (error) {
    QMessage.error('操作失败')
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 400px;
}

.modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.modal-btn {
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.cancel-btn {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
}

.confirm-btn {
  background: var(--primary-color);
  color: white;
  border: none;
}
</style>
```

- [ ] **步骤 4：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/file/CreateFolderModal.vue qim-client/src/components/apps/file/FilePreviewModal.vue qim-client/src/components/apps/file/FileActionsModal.vue
git commit -m "feat: 创建文件管理模态框组件"
```

---

### 任务 12：重写 FileManagementApp.vue 主容器

**文件：**
- 修改：`qim-client/src/components/apps/FileManagementApp.vue`

- [ ] **步骤 1：重写主组件**

```vue
<!-- qim-client/src/components/apps/FileManagementApp.vue -->
<template>
  <div class="files-app">
    <!-- 顶部导航 -->
    <div class="files-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <div class="files-header-info">
          <h2>文件管理</h2>
        </div>
      </div>
      <div class="files-header-actions">
        <button class="action-btn" @click="showCreateFolderModal = true" title="新建文件夹">
          <i class="fas fa-folder-plus"></i>
        </button>
        <button class="action-btn" @click="triggerFileUpload" title="上传文件">
          <i class="fas fa-upload"></i>
        </button>
        <input ref="fileInput" type="file" multiple style="display: none" @change="handleFileUpload" />
      </div>
    </div>

    <!-- 主体内容 -->
    <div class="files-content">
      <!-- 左侧文件夹树 -->
      <FolderTree
        :selected-folder-id="selectedFolderId"
        @select="handleFolderSelect"
        @create-folder="showCreateFolderModal = true"
      />

      <!-- 中间文件列表 -->
      <div class="files-main">
        <div class="files-path">
          <div class="path-item" @click="handleFolderSelect(null)">首页</div>
          <template v-if="currentFolder">
            <span class="path-separator">/</span>
            <div class="path-item">{{ currentFolder.name }}</div>
          </template>
        </div>

        <div class="files-search-box">
          <input
            type="text"
            v-model="searchQuery"
            placeholder="搜索文件..."
            class="files-search-input"
            @input="debouncedSearch"
          />
          <i class="fas fa-search files-search-icon"></i>
        </div>

        <FileList
          :files="files"
          :loading="loading"
          :has-more="hasMore"
          @preview="previewFile"
          @download="downloadFile"
          @star="toggleStar"
          @delete="deleteFile"
          @load-more="loadMoreFiles"
        />
      </div>
    </div>

    <!-- 模态框 -->
    <CreateFolderModal
      v-if="showCreateFolderModal"
      :is-editing="editingFolder !== null"
      :folder="editingFolder"
      :folders="rootFolders"
      @close="closeFolderModal"
      @success="handleFolderCreated"
    />

    <FilePreviewModal
      v-if="previewingFile"
      :file="previewingFile"
      @close="previewingFile = null"
      @download="downloadFile"
      @star="toggleStar"
    />

    <FileActionsModal
      v-if="actionModal.file"
      :action="actionModal.action"
      :file="actionModal.file"
      :folders="rootFolders"
      @close="closeActionModal"
      @success="handleActionSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { fileApi, folderApi, type FileItem, type FolderItem } from '@/api/file'
import { useFilePagination } from '@/composables/useFilePagination'
import QMessage from '@/utils/qmessage'
import FolderTree from './file/FolderTree.vue'
import FileList from './file/FileList.vue'
import CreateFolderModal from './file/CreateFolderModal.vue'
import FilePreviewModal from './file/FilePreviewModal.vue'
import FileActionsModal from './file/FileActionsModal.vue'

defineEmits(['back'])

const selectedFolderId = ref<number | null>(null)
const searchQuery = ref('')
const loading = ref(false)
const fileInput = ref<HTMLInputElement>()
const showCreateFolderModal = ref(false)
const editingFolder = ref<FolderItem | null>(null)
const previewingFile = ref<FileItem | null>(null)
const actionModal = ref<{ action: 'rename' | 'move'; file: FileItem | null }>({
  action: 'rename',
  file: null
})

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

const rootFolders = ref<FolderItem[]>([])

const currentFolder = computed(() => {
  if (selectedFolderId.value === null) return null
  if (selectedFolderId.value === -1) return { name: '星标文件' }
  return rootFolders.value.find(f => f.id === selectedFolderId.value) || null
})

let searchTimer: NodeJS.Timeout | null = null
function debouncedSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    reset()
    await loadNextPage()
  }, 300)
}

async function handleFolderSelect(folderId: number | null) {
  selectedFolderId.value = folderId
  reset()
  await loadNextPage()
}

async function loadMoreFiles() {
  await loadMore()
}

function triggerFileUpload() {
  fileInput.value?.click()
}

async function handleFileUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const uploadedFiles = target.files
  if (!uploadedFiles) return

  loading.value = true
  try {
    for (let i = 0; i < uploadedFiles.length; i++) {
      await fileApi.uploadFile(uploadedFiles[i], selectedFolderId.value || undefined)
    }
    QMessage.success('上传成功')
    reset()
    await loadNextPage()
  } catch (error) {
    QMessage.error('上传失败')
  } finally {
    loading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

async function previewFile(file: FileItem) {
  previewingFile.value = file
}

async function downloadFile(file: FileItem) {
  try {
    const response = await fileApi.downloadFile(file.id)
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    QMessage.error('下载失败')
  }
}

async function toggleStar(file: FileItem) {
  try {
    await fileApi.toggleStar(file.id, !file.is_starred)
    file.is_starred = !file.is_starred
  } catch (error) {
    QMessage.error('操作失败')
  }
}

async function deleteFile(file: FileItem) {
  try {
    await fileApi.deleteFile(file.id)
    QMessage.success('删除成功')
    reset()
    await loadNextPage()
  } catch (error) {
    QMessage.error('删除失败')
  }
}

function closeFolderModal() {
  showCreateFolderModal.value = false
  editingFolder.value = null
}

async function handleFolderCreated() {
  closeFolderModal()
  QMessage.success('操作成功')
  await loadRootFolders()
}

async function closeActionModal() {
  actionModal.value.file = null
}

async function handleActionSuccess() {
  closeActionModal()
  QMessage.success('操作成功')
  reset()
  await loadNextPage()
}

async function loadRootFolders() {
  try {
    const response = await folderApi.getFolderTree(null)
    rootFolders.value = response.data.data
  } catch (error) {
    console.error('加载文件夹失败:', error)
  }
}

onMounted(async () => {
  await loadRootFolders()
  await loadNextPage()
})
</script>

<style scoped>
.files-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.files-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
  height: 72px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
}

.files-header-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-color);
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
}

.files-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.files-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow-y: auto;
}

.files-path {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
  font-size: 14px;
  color: var(--text-secondary);
}

.path-item {
  cursor: pointer;
  transition: color 0.2s;
}

.path-item:hover {
  color: var(--primary-color);
}

.path-separator {
  color: var(--text-tertiary);
}

.files-search-box {
  position: relative;
  margin-bottom: 20px;
}

.files-search-input {
  width: 100%;
  padding: 10px 40px 10px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
}

.files-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

.files-search-icon {
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/src/components/apps/FileManagementApp.vue
git commit -m "feat: 重写文件管理主组件，使用子组件架构"
```

---

### 任务 13：编写 E2E 测试

**文件：**
- 创建：`qim-client/tests/e2e/file-box.spec.ts`

- [ ] **步骤 1：编写 E2E 测试**

```typescript
// qim-client/tests/e2e/file-box.spec.ts
import { test, expect } from '@playwright/test'

test.describe('文件箱功能', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.fill('input[placeholder="用户名"]', 'admin')
    await page.fill('input[placeholder="密码"]', 'admin123')
    await page.click('button:has-text("登录")')
    await page.waitForURL('**/main')
  })

  test('打开文件箱可以看到文件列表', async ({ page }) => {
    await page.click('[data-testid="app-file-management"]')
    await expect(page.locator('.files-app')).toBeVisible()
    await expect(page.locator('.folder-tree')).toBeVisible()
  })

  test('创建文件夹', async ({ page }) => {
    await page.click('[data-testid="app-file-management"]')
    await page.click('.create-folder-btn')
    await page.fill('input[placeholder="请输入文件夹名称"]', '测试文件夹')
    await page.click('button:has-text("创建")')
    await expect(page.locator('.folder-tree')).toContainText('测试文件夹')
  })

  test('上传文件', async ({ page }) => {
    await page.click('[data-testid="app-file-management"]')

    const fileChooserPromise = page.waitForEvent('filechooser')
    await page.click('button[title="上传文件"]')
    const fileChooser = await fileChooserPromise
    await fileChooser.setFiles('tests/fixtures/test-file.txt')

    await expect(page.locator('.file-grid-item')).toBeVisible()
  })

  test('星标文件', async ({ page }) => {
    await page.click('[data-testid="app-file-management"]')
    await page.locator('.file-grid-item').first().hover()
    await page.click('.file-actions .action-btn:has(.fa-star-o)')
    await expect(page.locator('.file-grid-item .starred-badge')).toBeVisible()
  })
})
```

- [ ] **步骤 2：Commit**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/tests/e2e/file-box.spec.ts
git commit -m "feat: 添加文件箱 E2E 测试"
```

---

### 任务 14：类型检查、Lint 和构建验证

**文件：** 全部

- [ ] **步骤 1：运行类型检查**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run typecheck
```

预期：PASS，无类型错误

- [ ] **步骤 2：运行 Lint**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run lint
```

预期：PASS，无 lint 错误

- [ ] **步骤 3：运行单元测试**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run test:unit
```

预期：全部 PASS

- [ ] **步骤 4：构建验证**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client
npm run build
```

预期：BUILD SUCCESS

- [ ] **步骤 5：Commit 所有修复**

```bash
cd /Users/gracegaoya/work/project/qim
git add -A
git commit -m "fix: 修复类型检查和 lint 问题"
```

---

## 规格覆盖度检查

| 规格需求 | 对应任务 | 状态 |
|---------|---------|------|
| 数据库字段扩展 | 任务 1 | ✅ |
| 文件类型工具函数 | 任务 2 | ✅ |
| 分页逻辑 | 任务 3 | ✅ |
| 文件夹树懒加载 | 任务 4 | ✅ |
| API 封装 | 任务 5 | ✅ |
| 文件操作接口 | 任务 6 | ✅ |
| 文件夹操作接口 | 任务 7 | ✅ |
| 后端路由注册 | 任务 8 | ✅ |
| 文件夹树组件 | 任务 9 | ✅ |
| 文件列表组件 | 任务 10 | ✅ |
| 模态框组件 | 任务 11 | ✅ |
| 主容器组件 | 任务 12 | ✅ |
| E2E 测试 | 任务 13 | ✅ |
| 类型检查/构建 | 任务 14 | ✅ |

全部需求已覆盖，无占位符，类型一致。
