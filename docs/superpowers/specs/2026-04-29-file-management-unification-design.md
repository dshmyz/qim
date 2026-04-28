# 文件管理体验统一设计

**日期：** 2026-04-29
**状态：** 设计完成，待实现
**目标：** 统一管理用户上传文件和聊天文件，提升文件管理体验

---

## 背景

当前文件管理存在以下问题：
1. 聊天文件和用户上传文件分散，无法统一管理
2. 文件箱缺少分享功能，无法将文件分享给其他人
3. 无法快速筛选不同来源的文件
4. 操作体验不一致

## 解决方案

### 核心思路
- 所有文件统一存储在 `files` 表中
- 通过 `source` 字段区分文件来源（`upload` / `chat`）
- 在文件箱中增加来源筛选功能
- 为文件增加分享功能

### 不改动部分
- 聊天消息中的文件操作保持不变
- 文件夹功能保持不变
- 现有的文件上传、下载、预览功能保持不变

---

## 详细设计

### 1. 数据模型改动

#### 1.1 文件来源标识

**目标：** 确保所有文件都有正确的来源标识

**实现方式：**

1. **用户上传文件** (`UploadFile` API)
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

2. **聊天文件** (`SendMessage` API)
   - 当消息类型为 `file` 时，更新对应文件的 `source` 字段
   - 或者在文件上传时就设置 `source = "chat"`

3. **数据迁移**
   - 现有文件的 `source` 字段默认为 `"upload"`
   - 需要一个迁移脚本更新现有数据

**数据流：**
```
用户上传 → UploadFile API → source="upload" → files表
聊天文件 → SendMessage → source="chat" → files表
```

---

### 2. 后端API改动

#### 2.1 GetFiles API（已支持，无需改动）

**功能：** 支持按来源筛选文件

**API接口：**
```
GET /api/v1/files                    // 所有文件
GET /api/v1/files?source=upload      // 我的上传
GET /api/v1/files?source=chat        // 聊天文件
GET /api/v1/files?folder_id=1        // 指定文件夹
```

**参数说明：**
- `source`: 文件来源筛选（`upload` / `chat`）
- `folder_id`: 文件夹ID筛选
- `starred`: 星标筛选
- `type`: 文件类型筛选
- `search`: 文件名搜索
- `page`: 页码
- `page_size`: 每页数量

#### 2.2 UploadFile API（需要改动）

**改动点：** 设置 `source` 字段为 `"upload"`

**改动位置：** `qim-server/handler/file_handler.go` 的 `UploadFile` 函数

#### 2.3 SendMessage API（需要改动）

**改动点：** 当消息类型为 `file` 时，设置文件的 `source` 字段为 `"chat"`

**改动位置：** `qim-server/handler/message_handler.go` 的 `SendMessage` 函数

---

### 3. 前端UI改动

#### 3.1 文件夹树组件（FolderTree.vue）

**目标：** 增加来源筛选标签

**布局设计：**
```
┌─────────────────┐
│ 文件夹          │
├─────────────────┤
│ [全部] [上传] [聊天] │ ← 新增来源标签
├─────────────────┤
│ 📁 全部文件      │
│ ⭐ 星标文件      │
│ 📂 工作文档      │
│ 📂 个人文件      │
└─────────────────┘
```

**实现细节：**
- 在文件夹树上方增加来源筛选标签
- 标签样式：使用类似标签页的设计，选中状态高亮
- 标签选项：全部、上传、聊天
- 点击标签时，触发 `source-change` 事件

**样式规范：**
- 标签高度：32px
- 标签间距：4px
- 选中状态：背景色 `var(--primary-color)`，文字白色
- 未选中状态：背景色 `var(--hover-color)`，文字 `var(--text-color)`

#### 3.2 文件管理主组件（FileManagementApp.vue）

**目标：** 支持来源筛选状态管理

**实现细节：**
- 增加状态：`selectedSource = ref<string | null>(null)`
- 修改文件加载逻辑，传递 `source` 参数
- 当来源标签切换时，重新加载文件列表

**状态管理：**
```typescript
const selectedSource = ref<string | null>(null)

const handleSourceChange = (source: string | null) => {
  selectedSource.value = source
  reset()
  loadNextPage()
}
```

#### 3.3 文件网格项组件（FileGridItem.vue）

**目标：** 增加分享按钮

**布局设计：**
```
┌──────────────┐
│  📄 文件图标  │
│              │
│  文件名.pdf  │
│  1.2 MB      │
│  [⭐] [📤] [🗑️] │ ← 操作按钮：星标、分享、删除
└──────────────┘
```

**实现细节：**
- 在操作按钮区增加分享按钮
- 图标：`fa-share-alt`
- 点击时触发 `share` 事件

#### 3.4 文件列表项组件（FileListItem.vue）

**目标：** 增加分享按钮

**布局设计：**
```
📄 文档.pdf | PDF | 1.2 MB | 2025-04-29 | [⭐] [📥] [📤] [🗑️]
```

**实现细节：**
- 在操作列增加分享按钮
- 图标：`fa-share-alt`
- 点击时触发 `share` 事件

#### 3.5 分享功能集成

**目标：** 复用现有分享功能

**实现方式：**
- 在 `FileManagementApp.vue` 中处理 `share` 事件
- 触发 `openShareModal` 自定义事件
- 复用现有的 `SharePreviewDialog` 组件

**代码示例：**
```typescript
const handleFileShare = (file: FileItem) => {
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type: 'file', data: file }
  }))
}
```

---

## 用户体验改进

### 改进前
- 聊天文件和上传文件分散，无法统一管理
- 文件箱缺少分享功能
- 无法快速筛选不同来源的文件
- 操作体验不一致

### 改进后
- 所有文件统一在文件箱管理
- 可按来源快速筛选：全部/上传/聊天
- 文件可直接分享给其他人
- 操作体验一致

---

## 技术实现复杂度

### 后端改动
- **复杂度：** 小
- **改动文件：** 
  - `qim-server/handler/file_handler.go`（UploadFile函数）
  - `qim-server/handler/message_handler.go`（SendMessage函数）

### 前端改动
- **复杂度：** 中等
- **改动文件：**
  - `qim-client/src/components/apps/file/FolderTree.vue`
  - `qim-client/src/components/apps/FileManagementApp.vue`
  - `qim-client/src/components/apps/file/FileGridItem.vue`
  - `qim-client/src/components/apps/file/FileListItem.vue`

### 数据迁移
- **复杂度：** 小
- **需要：** 更新现有文件的 `source` 字段

---

## 测试要点

### 功能测试
1. 用户上传文件，`source` 字段正确设置为 `"upload"`
2. 聊天发送文件，`source` 字段正确设置为 `"chat"`
3. 来源筛选标签切换正常，文件列表正确更新
4. 文件分享功能正常工作
5. 现有功能不受影响（上传、下载、预览、删除、星标等）

### 兼容性测试
1. 现有文件的 `source` 字段迁移正确
2. 旧版本API兼容性
3. 不同浏览器的兼容性

### 性能测试
1. 大量文件时的筛选性能
2. 来源标签切换的响应速度

---

## 实现计划

### 阶段1：后端改动
1. 修改 `UploadFile` API，设置 `source = "upload"`
2. 修改 `SendMessage` API，设置聊天文件的 `source = "chat"`
3. 编写数据迁移脚本

### 阶段2：前端UI改动
1. 修改 `FolderTree` 组件，增加来源筛选标签
2. 修改 `FileManagementApp` 组件，支持来源筛选状态
3. 修改 `FileGridItem` 组件，增加分享按钮
4. 修改 `FileListItem` 组件，增加分享按钮
5. 集成分享功能

### 阶段3：测试和优化
1. 功能测试
2. 兼容性测试
3. 性能优化

---

## 风险和注意事项

### 风险
1. 数据迁移可能影响现有数据
2. 前端改动可能影响现有功能

### 缓解措施
1. 数据迁移前备份数据库
2. 充分测试现有功能
3. 分阶段发布，先发布后端改动，再发布前端改动

---

## 后续优化方向

1. **文件统计：** 在来源标签旁显示文件数量
2. **批量操作：** 支持批量分享、批量移动
3. **文件预览优化：** 支持更多文件类型的预览
4. **搜索增强：** 支持按来源、时间、大小等条件组合搜索
