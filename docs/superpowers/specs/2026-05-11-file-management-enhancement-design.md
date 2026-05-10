# QIM 文件管理增强功能设计文档

**日期：** 2026-05-11  
**版本：** 1.0  
**作者：** AI Assistant

---

## 概述

### 目标

完善 QIM 文件管理功能，实现：
1. 大文件分片上传（支持 < 100MB 文件）
2. 上传进度显示（全局进度条）
3. 文件预览增强（PDF + 纯文本）

### 背景

当前文件管理功能存在以下不足：
- 大文件上传容易失败，无断点续传
- 上传过程无进度反馈
- 文件预览支持格式有限（仅图片、视频、音频）

### 非目标

- 不支持超大文件（> 100MB）
- 不支持 Office 文档在线预览
- 不支持 CAD 图纸预览

---

## 整体架构

### 系统分层

```
┌─────────────────────────────────────────┐
│           用户界面层                      │
│  - FileManagementApp.vue                │
│  - UploadProgressBar.vue (新增)         │
│  - FilePreviewModal.vue (增强)          │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│           业务逻辑层                      │
│  - useFileUpload.ts (新增)              │
│  - useUploadStore.ts (新增, Pinia)      │
│  - fileApi.ts (增强)                    │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│           后端服务层                      │
│  - file_handler.go (增强)               │
│  - file_service.go (增强)               │
│  - chunk_storage.go (新增)              │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│           存储层                          │
│  - 本地文件系统 / S3                      │
│  - 数据库 (chunks 表)                    │
└─────────────────────────────────────────┘
```

### 核心特性

- **智能分片策略**：根据文件大小自动选择分片策略
- **秒传**：通过 MD5 检测避免重复上传
- **断点续传**：记录已上传分片，支持中断后继续
- **并发上传**：多个分片同时上传，提升速度
- **全局进度条**：统一显示所有上传任务进度
- **文件预览增强**：支持 PDF 和纯文本预览

---

## 数据库设计

### 新增表：file_chunks（文件分片记录）

```sql
CREATE TABLE file_chunks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  upload_id VARCHAR(64) NOT NULL,           -- 上传任务唯一标识
  file_hash VARCHAR(64) NOT NULL,           -- 文件整体 MD5
  chunk_index INTEGER NOT NULL,             -- 分片序号（从0开始）
  chunk_hash VARCHAR(64) NOT NULL,          -- 分片 MD5
  chunk_size INTEGER NOT NULL,              -- 分片大小
  storage_path VARCHAR(512),                -- 分片存储路径
  status VARCHAR(20) DEFAULT 'pending',     -- pending/uploaded/merged
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  UNIQUE(upload_id, chunk_index),
  INDEX idx_upload_id (upload_id),
  INDEX idx_file_hash (file_hash)
);
```

### 新增表：upload_tasks（上传任务记录）

```sql
CREATE TABLE upload_tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  upload_id VARCHAR(64) NOT NULL UNIQUE,    -- 上传任务唯一标识
  user_id INTEGER NOT NULL,
  filename VARCHAR(255) NOT NULL,
  file_size INTEGER NOT NULL,
  file_hash VARCHAR(64),                    -- 文件整体 MD5
  total_chunks INTEGER NOT NULL,
  uploaded_chunks INTEGER DEFAULT 0,        -- 已上传分片数
  folder_id INTEGER,
  status VARCHAR(20) DEFAULT 'pending',     -- pending/uploading/completed/failed/cancelled
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  INDEX idx_user_id (user_id),
  INDEX idx_status (status)
);
```

### 修改表：files（文件表）

```sql
ALTER TABLE files ADD COLUMN file_hash VARCHAR(64);
ALTER TABLE files ADD COLUMN upload_id VARCHAR(64);
CREATE INDEX idx_file_hash ON files(file_hash);
```

---

## 后端 API 设计

### 1. 初始化上传

**接口：** `POST /api/v1/files/upload/init`

**请求：**
```json
{
  "filename": "document.pdf",
  "file_size": 52428800,
  "file_hash": "a1b2c3d4e5f6...",
  "folder_id": null
}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "upload_id": "upload_abc123",
    "chunk_size": 5242880,
    "total_chunks": 10,
    "uploaded_chunks": [0, 1, 2],
    "file_exists": false,
    "file_id": null
  }
}
```

**逻辑：**
1. 生成唯一 `upload_id`
2. 检查 `file_hash` 是否已存在（秒传）
3. 如果存在未完成的上传任务，返回已上传分片列表（断点续传）
4. 创建上传任务记录
5. 返回分片策略和任务信息

### 2. 上传分片

**接口：** `POST /api/v1/files/upload/chunk`

**请求：**
```
Content-Type: multipart/form-data

upload_id: upload_abc123
chunk_index: 0
chunk_hash: hash123
file: <binary data>
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "chunk_index": 0,
    "uploaded": true
  }
}
```

**逻辑：**
1. 验证 `upload_id` 和 `chunk_index`
2. 校验分片 MD5
3. 存储分片到临时位置
4. 更新分片状态为 `uploaded`
5. 更新上传任务进度

### 3. 完成上传

**接口：** `POST /api/v1/files/upload/complete`

**请求：**
```json
{
  "upload_id": "upload_abc123"
}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "file_id": 123,
    "name": "document.pdf",
    "size": 52428800,
    "url": "/uploads/xxx.pdf"
  }
}
```

**逻辑：**
1. 检查所有分片是否已上传
2. 合并所有分片为完整文件
3. 计算文件 MD5，验证完整性
4. 创建文件记录
5. 清理临时分片和上传任务
6. 返回文件信息

### 4. 取消上传

**接口：** `POST /api/v1/files/upload/cancel`

**请求：**
```json
{
  "upload_id": "upload_abc123"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "上传已取消"
}
```

**逻辑：**
1. 标记上传任务为 `cancelled`
2. 删除已上传的分片文件
3. 删除分片记录
4. 删除上传任务记录

---

## 前端组件设计

### 1. UploadProgressBar.vue（全局上传进度条）

**位置：** `qim-client/src/components/common/UploadProgressBar.vue`

**功能：**
- 固定在页面顶部
- 显示当前上传任务数量和总进度
- 支持展开/收起
- 显示每个文件的上传状态和进度
- 支持取消上传

**UI 设计：**
```
┌────────────────────────────────────────────┐
│ 📤 上传中 (2/3)              [展开 ▼]      │
├────────────────────────────────────────────┤
│ document.pdf        ████████░░ 80%  40MB  │
│ image.png           ██████████ 100% 完成   │
│ video.mp4           ██░░░░░░░░ 20%  15MB  │
└────────────────────────────────────────────┘
```

**Props：**
```typescript
interface Props {
  visible: boolean
}
```

**Events：**
```typescript
emit('cancel', uploadId: string)
emit('retry', uploadId: string)
emit('clear-completed')
```

### 2. useUploadStore.ts（Pinia 状态管理）

**位置：** `qim-client/src/stores/upload.ts`

**状态：**
```typescript
interface UploadState {
  tasks: UploadTask[]
  isExpanded: boolean
}

interface UploadTask {
  uploadId: string
  file: File
  folderId: number | null
  status: 'pending' | 'uploading' | 'completed' | 'failed' | 'cancelled'
  progress: number          // 0-100
  uploadedSize: number      // 已上传字节数
  totalSize: number         // 总字节数
  uploadedChunks: number[]  // 已上传分片索引
  totalChunks: number       // 总分片数
  error?: string
  retryCount: number        // 重试次数
}
```

**方法：**
```typescript
actions: {
  addTask(file: File, folderId?: number): Promise<string>
  updateProgress(uploadId: string, progress: number)
  updateChunkProgress(uploadId: string, chunkIndex: number)
  markCompleted(uploadId: string, fileId: number)
  markFailed(uploadId: string, error: string)
  cancelTask(uploadId: string)
  removeTask(uploadId: string)
  clearCompleted()
  toggleExpanded()
}
```

### 3. useFileUpload.ts（上传逻辑 Composable）

**位置：** `qim-client/src/composables/useFileUpload.ts`

**方法：**
```typescript
// 初始化上传
async function initUpload(file: File, folderId?: number): Promise<InitResponse>

// 上传分片
async function uploadChunk(
  uploadId: string,
  chunk: Blob,
  chunkIndex: number,
  onProgress?: (progress: number) => void
): Promise<void>

// 完成上传
async function completeUpload(uploadId: string): Promise<FileItem>

// 取消上传
async function cancelUpload(uploadId: string): Promise<void>

// 计算文件 MD5
async function calculateMD5(file: File): Promise<string>

// 文件分片
function splitFile(file: File, chunkSize: number): Blob[]

// 智能分片策略
function getChunkStrategy(fileSize: number): { chunkSize: number; totalChunks: number }
```

**智能分片策略：**
```typescript
function getChunkStrategy(fileSize: number) {
  if (fileSize < 10 * 1024 * 1024) {
    // < 10MB: 不分片
    return { chunkSize: fileSize, totalChunks: 1 }
  } else if (fileSize < 50 * 1024 * 1024) {
    // 10-50MB: 5MB 每片
    const chunkSize = 5 * 1024 * 1024
    return { chunkSize, totalChunks: Math.ceil(fileSize / chunkSize) }
  } else {
    // 50-100MB: 10MB 每片
    const chunkSize = 10 * 1024 * 1024
    return { chunkSize, totalChunks: Math.ceil(fileSize / chunkSize) }
  }
}
```

### 4. FilePreviewModal.vue（文件预览增强）

**位置：** `qim-client/src/components/apps/file/FilePreviewModal.vue`

**新增功能：**

#### PDF 预览

**依赖：** `pdfjs-dist`

**实现：**
```typescript
import * as pdfjsLib from 'pdfjs-dist'

// 加载 PDF
const loadingTask = pdfjsLib.getDocument(url)
const pdf = await loadingTask.promise

// 渲染页面
const page = await pdf.getPage(pageNumber)
const canvas = canvasRef.value
const context = canvas.getContext('2d')

const viewport = page.getViewport({ scale: 1.5 })
canvas.height = viewport.height
canvas.width = viewport.width

await page.render({
  canvasContext: context,
  viewport: viewport
}).promise
```

**功能：**
- 页码导航（上一页/下一页）
- 缩放控制（放大/缩小/适应宽度）
- 全屏查看
- 文本选择和复制

#### 纯文本预览

**支持格式：** `.txt`, `.log`, `.md`, `.json`, `.xml`, `.csv`, 代码文件

**实现：**
```typescript
// 获取文本内容
const response = await fetch(fileUrl)
const text = await response.text()

// 显示优化
- 保留换行和空格
- 等宽字体显示
- 行号显示（可选）
- 语法高亮（可选，使用 highlight.js）
```

**UI 设计：**
```
┌────────────────────────────────────────────┐
│ 📄 document.pdf                    [×]     │
├────────────────────────────────────────────┤
│ [←] [→] 第 3/10 页  [−] 100% [+] [全屏]   │
├────────────────────────────────────────────┤
│                                            │
│                                            │
│           PDF 内容渲染区域                  │
│                                            │
│                                            │
├────────────────────────────────────────────┤
│ 40 MB • 2024-01-15 上传    [下载] [分享]  │
└────────────────────────────────────────────┘
```

---

## 上传流程设计

### 完整上传流程

```
用户选择文件
    ↓
判断文件大小
    ├─ < 10MB → 直接上传（不分片）
    ├─ 10-50MB → 分片上传（5MB 每片）
    └─ 50-100MB → 分片上传（10MB 每片）
    ↓
计算文件 MD5（Web Worker，不阻塞 UI）
    ↓
调用初始化接口
    ├─ 秒传成功 → 直接完成，显示成功
    └─ 需要上传 → 继续
    ↓
创建上传任务，存入 Pinia Store
    ↓
显示全局进度条
    ↓
并发上传分片（最多 3 个并发）
    ├─ 成功 → 更新进度
    └─ 失败 → 自动重试（最多 3 次）
    ↓
所有分片上传完成
    ↓
调用完成接口，后端合并分片
    ↓
更新文件列表，显示成功提示
    ↓
清理上传任务
```

### 断点续传流程

```
用户选择文件
    ↓
计算文件 MD5
    ↓
调用初始化接口
    ↓
返回已上传分片列表 [0, 1, 2]
    ↓
只上传未完成的分片 [3, 4, 5, ...]
    ↓
继续正常流程
```

### 取消上传流程

```
用户点击取消按钮
    ↓
标记任务为"已取消"
    ↓
中止正在进行的分片上传
    ↓
调用取消接口
    ↓
后端清理已上传分片
    ↓
从任务列表移除
```

---

## 错误处理设计

### 前端错误处理

**错误类型：**

1. **网络错误**
   - 分片上传失败 → 自动重试（最多3次）
   - 重试失败 → 标记任务失败，提示用户

2. **文件错误**
   - 文件大小超限 → 上传前检查，提示用户
   - 文件类型不支持 → 上传前检查，提示用户
   - 文件损坏 → 上传时检测，提示用户

3. **服务器错误**
   - 存储空间不足 → 提示管理员
   - 权限错误 → 提示用户
   - 服务器异常 → 提示用户稍后重试

4. **用户取消**
   - 清理已上传分片
   - 更新任务状态

**错误提示 UI：**
```
┌────────────────────────────────────────────┐
│ ❌ 上传失败                                 │
├────────────────────────────────────────────┤
│ document.pdf        网络错误，已重试3次     │
│                     [重试] [取消]           │
└────────────────────────────────────────────┘
```

### 后端错误处理

**错误类型：**

1. **分片校验失败**
   - MD5 不匹配 → 返回错误，要求重新上传

2. **存储错误**
   - 磁盘空间不足 → 返回错误
   - 文件写入失败 → 返回错误

3. **合并错误**
   - 分片缺失 → 返回错误，要求重新上传
   - 合并失败 → 返回错误

4. **并发冲突**
   - 同一文件同时上传 → 使用 upload_id 隔离
   - 分片重复上传 → 幂等性处理

**错误响应格式：**
```json
{
  "code": 400,
  "message": "分片校验失败，请重新上传",
  "data": {
    "chunk_index": 5,
    "expected_hash": "abc123",
    "actual_hash": "def456"
  }
}
```

---

## 性能优化

### 前端优化

1. **Web Worker 计算 MD5**
   - 在 Web Worker 中计算文件 MD5
   - 不阻塞主线程 UI 渲染

2. **并发上传**
   - 最多 3 个分片同时上传
   - 使用 Promise.all 控制并发

3. **懒加载**
   - PDF 按页加载，不一次性加载所有页
   - 图片预览使用缩略图

4. **缓存**
   - 已预览的文件缓存到内存
   - 使用 Service Worker 缓存静态资源

5. **虚拟滚动**
   - 大文本文件使用虚拟滚动
   - 只渲染可见区域

### 后端优化

1. **分片存储**
   - 分片存储在临时目录
   - 合并后删除临时文件

2. **并发控制**
   - 使用互斥锁防止并发冲突
   - 分片上传幂等性处理

3. **定期清理**
   - 定期清理未完成的上传任务
   - 清理超过 24 小时的临时分片

---

## 测试计划

### 单元测试

**前端：**
- MD5 计算准确性
- 文件分片逻辑
- 进度计算逻辑
- 错误处理逻辑

**后端：**
- 分片上传接口
- 分片合并逻辑
- 秒传检测逻辑
- 断点续传逻辑

### 集成测试

**场景：**
1. 小文件上传（< 10MB）
2. 中等文件上传（10-50MB）
3. 大文件上传（50-100MB）
4. 秒传场景
5. 断点续传场景
6. 取消上传场景
7. 并发上传多个文件
8. 网络中断恢复

### 性能测试

**指标：**
- 上传速度（MB/s）
- 内存占用（峰值）
- CPU 占用率
- 并发上传能力

---

## 实施计划

### 阶段 1：数据库和后端基础（1-2天）

1. 创建数据库表
2. 实现后端 API 接口
3. 实现分片存储逻辑
4. 实现分片合并逻辑
5. 编写单元测试

### 阶段 2：前端上传功能（2-3天）

1. 实现 useUploadStore
2. 实现 useFileUpload
3. 实现 UploadProgressBar
4. 集成到 FileManagementApp
5. 测试上传功能

### 阶段 3：文件预览增强（1天）

1. 安装 pdfjs-dist
2. 实现 PDF 预览
3. 实现纯文本预览
4. 优化预览体验
5. 测试预览功能

### 阶段 4：测试和优化（1天）

1. 编写集成测试
2. 性能测试和优化
3. Bug 修复
4. 文档完善

**总计：** 5-7 天

---

## 风险和缓解措施

### 风险 1：大文件上传内存占用高

**缓解措施：**
- 使用流式上传，不一次性加载整个文件
- 使用 Web Worker 计算 MD5
- 限制并发上传数量

### 风险 2：网络不稳定导致上传失败

**缓解措施：**
- 实现自动重试机制
- 支持断点续传
- 提供手动重试选项

### 风险 3：分片合并失败

**缓解措施：**
- 合并前验证所有分片完整性
- 保留临时分片，支持重新合并
- 记录详细日志，便于排查

### 风险 4：并发上传冲突

**缓解措施：**
- 使用 upload_id 隔离不同上传任务
- 实现分片上传幂等性
- 使用数据库事务保证一致性

---

## 未来扩展

### 短期扩展（可选）

1. **Office 文档预览**
   - 使用 docx-preview、xlsx 等库
   - 或使用后端转换为 PDF

2. **Markdown 预览增强**
   - 支持 Markdown 渲染
   - 支持代码高亮

3. **图片预览增强**
   - 支持图片缩放、旋转
   - 支持图片编辑（裁剪、标注）

### 长期扩展

1. **支持超大文件（> 100MB）**
   - 实现更复杂的分片策略
   - 支持秒传和断点续传
   - 优化内存占用

2. **文件版本管理**
   - 支持文件历史版本
   - 支持版本对比
   - 支持版本回退

3. **文件协作**
   - 支持多人同时编辑
   - 实时同步
   - 冲突解决

---

## 总结

本设计文档详细描述了 QIM 文件管理增强功能的实现方案，包括：

1. **大文件分片上传**：智能分片策略，支持秒传和断点续传
2. **上传进度显示**：全局进度条，实时反馈上传状态
3. **文件预览增强**：支持 PDF 和纯文本预览

通过合理的架构设计、完善的错误处理和性能优化，确保功能稳定可靠，用户体验良好。

---

## 附录

### A. 技术栈

**前端：**
- Vue 3
- TypeScript
- Pinia
- Axios
- pdfjs-dist
- highlight.js（可选）

**后端：**
- Go
- Gin
- GORM
- 本地文件系统 / S3

**数据库：**
- SQLite / MySQL

### B. 参考资料

- [pdf.js 官方文档](https://mozilla.github.io/pdf.js/)
- [File API - MDN](https://developer.mozilla.org/en-US/docs/Web/API/File)
- [Web Workers - MDN](https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API)
- [分片上传最佳实践](https://cloud.tencent.com/document/product/436/14106)

### C. 变更历史

| 版本 | 日期 | 作者 | 变更说明 |
|------|------|------|----------|
| 1.0 | 2026-05-11 | AI Assistant | 初始版本 |
