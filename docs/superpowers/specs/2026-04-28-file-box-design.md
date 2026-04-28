# 文件箱功能设计文档

> 创建时间: 2026-04-28
> 状态: 已批准

## 1. 需求概述

### 目标
实现个人文件箱功能，作为用户上传附件的统一管理和整理中心。

### 核心场景
- 用户上传文件后自动出现在文件箱
- 聊天中的附件可保存到文件箱
- 用户创建文件夹整理文件
- 按类型快速筛选查看

## 2. 数据库设计

### 2.1 Files 表（改造）

```sql
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
  `source` VARCHAR(20) DEFAULT 'upload',          -- 来源: upload/chat/share
  `source_id` VARCHAR(100),                       -- 来源ID（如消息ID）
  `is_starred` INTEGER DEFAULT 0,                 -- 是否星标
  `starred_at` DATETIME,                          -- 星标时间
  `tags` VARCHAR(500),                            -- 标签（逗号分隔）
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

### 2.2 Folders 表（改造）

```sql
CREATE TABLE IF NOT EXISTS `folders` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `parent_id` INTEGER,
  `sort_order` INTEGER DEFAULT 0,              -- 排序
  `icon` VARCHAR(50),                          -- 自定义图标
  `color` VARCHAR(20),                         -- 文件夹颜色
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_folders_user_id` ON `folders`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_parent_id` ON `folders`(`parent_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_deleted_at` ON `folders`(`deleted_at`);
```

## 3. 后端接口设计

### 3.1 现有接口（保持）

| 接口 | 方法 | 说明 |
|------|------|------|
| `POST /upload` | POST | 上传文件 |
| `GET /files` | GET | 获取文件列表 |
| `GET /files/:id/download` | GET | 下载文件 |
| `DELETE /files/:id` | DELETE | 删除文件 |
| `POST /folders` | POST | 创建文件夹 |

### 3.2 需要改造的接口

#### GET /files - 获取文件列表（扩展参数）

**Query Parameters:**
```
folder_id: string    // 文件夹ID
source: string       // 来源过滤: upload/chat/share
starred: boolean     // 是否只返回星标文件
type: string         // 类型过滤: image/document/video
search: string       // 文件名搜索
page: number         // 页码，默认1
page_size: number    // 每页数量，默认50
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "files": [...],
    "total": 1234,
    "page": 1,
    "page_size": 50
  }
}
```

#### GET /folders/tree - 获取文件夹树（完整实现）

**Query Parameters:**
```
lazy: boolean        // 是否懒加载模式
parent_id: number    // 懒加载时指定父文件夹ID
```

**Response (懒加载模式):**
```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "工作",
      "icon": "folder-briefcase",
      "color": "#4CAF50",
      "has_children": true,
      "file_count": 12
    }
  ]
}
```

**Response (完整树模式):**
```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "工作",
      "children": [
        {"id": 2, "name": "项目A", "children": []}
      ]
    }
  ]
}
```

### 3.3 新增接口

#### PUT /files/:id - 更新文件信息

**Request:**
```json
{
  "name": "新文件名",
  "folder_id": 123,
  "tags": ["work", "important"]
}
```

#### PUT /files/:id/star - 星标/取消星标

**Request:**
```json
{
  "starred": true
}
```

#### PUT /files/batch - 批量操作

**Request:**
```json
{
  "file_ids": [1, 2, 3],
  "action": "move",           // move/delete/star/add_tags
  "folder_id": 123,           // move时需要
  "tags": ["tag1", "tag2"]    // add_tags时需要
}
```

#### GET /files/starred - 获取星标文件

同 GET /files 参数，自动添加 starred=true

#### GET /files/stats - 文件统计

**Response:**
```json
{
  "code": 0,
  "data": {
    "total_count": 1234,
    "total_size": 1073741824,
    "by_type": {
      "image": {"count": 500, "size": 524288000},
      "document": {"count": 300, "size": 209715200},
      "video": {"count": 50, "size": 314572800}
    },
    "by_source": {
      "upload": 800,
      "chat": 400,
      "share": 34
    }
  }
}
```

#### PUT /folders/:id - 更新文件夹

**Request:**
```json
{
  "name": "新名称",
  "parent_id": 5,
  "icon": "folder-briefcase",
  "color": "#4CAF50",
  "sort_order": 10
}
```

#### DELETE /folders/:id - 删除文件夹

- 如果文件夹内有文件，询问是否一并删除或移动到根目录
- 递归删除子文件夹

#### GET /folders/:id/files - 获取文件夹内文件

同 GET /files 参数，自动添加 folder_id

## 4. 前端组件架构

```
FileManagementApp.vue (主容器)
├── FolderTree.vue (左侧文件夹树 - 懒加载)
│   └── FolderTreeItem.vue (单个文件夹项 - 可展开)
├── FileList.vue (中间文件列表 - 分页/虚拟滚动)
│   ├── FileGridItem.vue (网格视图文件项)
│   └── FileListItem.vue (列表视图文件项)
├── FilePreviewModal.vue (文件预览模态框)
├── CreateFolderModal.vue (创建文件夹)
├── FileActionsModal.vue (移动/重命名/标签操作)
└── FileStatsPanel.vue (文件统计面板)
```

### 4.1 组件职责

| 组件 | 职责 | 依赖 |
|------|------|------|
| FileManagementApp | 整体布局、状态管理 | 所有子组件 |
| FolderTree | 文件夹树展示、懒加载请求 | FolderTreeItem, API |
| FolderTreeItem | 单个文件夹展示、展开/收起 | API |
| FileList | 文件列表、分页/虚拟滚动 | FileGridItem/FileListItem, API |
| FileGridItem | 网格视图文件卡片 | - |
| FileListItem | 列表视图文件行 | - |
| FilePreviewModal | 图片/视频/音频预览 | - |
| CreateFolderModal | 创建/编辑文件夹表单 | API |
| FileActionsModal | 批量操作、移动、重命名 | API |
| FileStatsPanel | 文件统计信息展示 | API |

### 4.2 性能优化

**文件夹树懒加载:**
- 首次只请求 `parent_id=null` 的根文件夹
- 点击展开时请求子文件夹
- 展开状态缓存，不重复请求

**文件列表分页:**
- 默认 page_size=50
- 滚动到底部自动加载下一页
- 支持切换网格/列表视图

**虚拟滚动:**
- 使用虚拟列表组件，只渲染可视区域
- 预估卡片高度，避免跳动

## 5. 数据流

### 5.1 浏览文件箱

```
用户点击"文件箱"
    ↓
并行请求:
  - GET /folders/tree?lazy=true  (获取根文件夹)
  - GET /files?page=1&page_size=50  (获取全部文件)
    ↓
渲染左侧文件夹树 + 中间文件列表
```

### 5.2 进入文件夹

```
用户点击文件夹
    ↓
请求 GET /folders/:id/files?page=1&page_size=50
    ↓
如需子文件夹，请求 GET /folders/tree?lazy=true&parent_id=:id
    ↓
更新文件列表显示
```

### 5.3 移动文件到文件夹

```
用户选择文件 → 点击"移动" → 选择目标文件夹
    ↓
请求 PUT /files/batch { file_ids, action: "move", folder_id }
    ↓
刷新当前列表
```

## 6. 错误处理

| 错误场景 | 处理方式 |
|----------|----------|
| 上传失败 | 显示错误提示，保留重试按钮 |
| 文件夹创建失败 | 提示原因（同名/名称过长） |
| 文件移动失败 | 回滚状态，显示具体失败原因 |
| 网络断开 | 显示离线提示，排队待同步操作 |
| 权限不足 | 提示"无权限操作此文件" |
| 文件已被删除 | 列表刷新，提示"文件已被删除" |
| 文件夹内有文件时删除 | 弹出确认框，询问是否一并删除 |

## 7. 测试策略

### 7.1 单元测试

- 文件夹树构建逻辑
- 分页计算逻辑
- 文件类型判断函数
- 文件大小格式化

### 7.2 组件测试

- 文件夹树展开/收起
- 文件列表分页加载
- 模态框交互（创建/编辑/移动）
- 文件预览组件

### 7.3 E2E 测试

- 完整上传流程
- 创建文件夹 → 移动文件 → 预览
- 批量操作
- 星标/取消星标

## 8. 功能优先级

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 文件夹树懒加载 | 🔴 P0 | 解决目录多时的性能问题 |
| 文件列表分页/虚拟滚动 | 🔴 P0 | 解决单目录文件过多问题 |
| 创建/编辑/删除文件夹 | 🔴 P0 | 基础文件夹管理 |
| 文件移动到文件夹 | 🔴 P0 | 核心整理能力 |
| 文件星标 | 🟡 P1 | 快捷收藏 |
| 文件标签 | 🟡 P1 | 灵活分类 |
| 按类型/来源筛选 | 🟡 P1 | 智能分类 |
| 批量操作 | 🟢 P2 | 效率提升 |
| 文件统计 | 🟢 P2 | 数据可视化 |

## 9. 性能指标目标

| 指标 | 目标 |
|------|------|
| 首屏文件夹树加载 | < 200ms (100个根文件夹) |
| 文件列表首屏加载 | < 500ms (50个文件) |
| 滚动加载下一页 | < 300ms |
| 单文件夹支持文件数 | > 10000 (分页+虚拟滚动) |
| 文件夹树深度 | 无限制 (懒加载) |
