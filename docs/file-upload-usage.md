# QIM 文件上传功能使用指南

## 功能概述

QIM 文件上传功能提供了企业级的文件管理解决方案，支持大文件上传、断点续传、秒传等高级特性，确保文件传输的高效性和可靠性。

### 核心特性

- **智能分片上传**：根据文件大小自动选择最优分片策略
- **秒传功能**：相同文件瞬间上传完成
- **断点续传**：网络中断后可从断点继续上传
- **并发上传**：基于 CPU 核心数动态调整并发数，最大化上传效率
- **进度追踪**：实时显示上传进度和状态
- **文件预览**：支持图片、视频、音频、PDF、文本等多种格式在线预览
- **错误重试**：自动重试失败的分片上传

## 使用方法

### 基本使用

#### 1. 在组件中使用 useFileUpload

```typescript
import { useFileUpload } from '@/composables/useFileUpload'

export default {
  setup() {
    const { uploadFile, tasks, cancelUpload } = useFileUpload()

    // 上传文件
    const handleUpload = async (file: File, folderId?: number) => {
      try {
        const result = await uploadFile(file, folderId)
        console.log('上传成功:', result)
      } catch (error) {
        console.error('上传失败:', error)
      }
    }

    // 取消上传
    const handleCancel = (uploadId: string) => {
      cancelUpload(uploadId)
    }

    return {
      handleUpload,
      handleCancel,
      tasks
    }
  }
}
```

#### 2. 上传任务状态

上传任务包含以下状态：

- `pending`：等待上传
- `uploading`：上传中
- `completed`：上传完成
- `failed`：上传失败
- `cancelled`：已取消

#### 3. 监控上传进度

```typescript
const { activeTasks, totalProgress } = useFileUpload()

// activeTasks: 当前正在上传的任务列表
// totalProgress: 总体上传进度（0-100）
```

### 高级用法

#### 1. 手动控制上传流程

```typescript
import { 
  calculateMD5, 
  initUpload, 
  uploadFile, 
  completeUpload 
} from '@/composables/useFileUpload'

// 1. 计算文件 MD5
const fileHash = await calculateMD5(file)

// 2. 初始化上传
const initResponse = await initUpload(file, folderId)

// 3. 检查是否秒传
if (initResponse.is_quick_upload) {
  console.log('秒传成功！')
  return
}

// 4. 上传文件
const result = await uploadFile(file, folderId)

// 5. 完成上传
const fileInfo = await completeUpload(
  initResponse.upload_id,
  fileHash,
  initResponse.total_chunks
)
```

#### 2. 自定义分片策略

```typescript
import { getChunkStrategy, splitFile } from '@/composables/useFileUpload'

// 获取推荐的分片策略
const strategy = getChunkStrategy(file.size)
console.log('分片大小:', strategy.chunkSize)
console.log('策略描述:', strategy.description)

// 手动分片
const chunks = splitFile(file, strategy.chunkSize)
```

## 分片上传策略

### 智能分片规则

系统根据文件大小自动选择最优的分片策略：

| 文件大小 | 分片大小 | 说明 |
|---------|---------|------|
| < 10MB | 不分片 | 小文件直接上传 |
| 10MB - 100MB | 2MB | 中等文件 |
| 100MB - 500MB | 5MB | 较大文件 |
| 500MB - 1GB | 10MB | 大文件 |
| > 1GB | 20MB | 超大文件 |

### 分片上传流程

1. **文件分片**：将文件按照策略切分为多个分片
2. **计算 MD5**：为每个分片计算 MD5 哈希值
3. **并发上传**：使用队列管理器并发上传分片
4. **进度更新**：实时更新上传进度
5. **服务器合并**：所有分片上传完成后，服务器合并文件

### 并发控制

- **动态并发数**：基于 CPU 核心数自动调整（`navigator.hardwareConcurrency`）
- **最大并发**：最多 5 个分片同时上传
- **最小并发**：至少 2 个分片同时上传
- **队列管理**：使用队列管理器控制并发上传

### 内存优化

- 上传完成的分片立即释放内存引用
- 避免同时加载所有分片到内存
- 使用 Blob.slice() 实现零拷贝分片

## 秒传和断点续传

### 秒传（快速上传）

#### 工作原理

1. **计算文件 MD5**：上传前计算整个文件的 MD5 哈希值
2. **服务器检查**：将 MD5 发送到服务器检查文件是否已存在
3. **秒传判定**：如果服务器存在相同文件，直接返回文件信息
4. **瞬间完成**：无需上传文件内容，立即完成上传

#### 适用场景

- 用户重复上传相同文件
- 不同用户上传相同文件
- 文件移动或复制操作

#### 性能优势

- **节省带宽**：无需传输文件内容
- **节省时间**：大文件瞬间完成上传
- **节省存储**：服务器只存储一份文件

### 断点续传

#### 工作原理

1. **初始化上传**：服务器返回已上传的分片列表
2. **跳过已上传**：只上传未完成的分片
3. **续传记录**：服务器记录每个分片的上传状态
4. **网络恢复**：网络中断后可从断点继续上传

#### 使用场景

- 网络不稳定环境
- 大文件上传
- 浏览器意外关闭后恢复

#### 实现细节

```typescript
// 初始化时获取已上传分片
const initResponse = await initUpload(file, folderId)

// initResponse.uploaded_chunks 包含已上传的分片索引
// 系统会自动跳过这些分片，只上传剩余部分
```

## 文件预览支持格式

### 支持预览的文件类型

#### 图片类型
- **格式**：JPEG, PNG, GIF, WebP, SVG, BMP
- **MIME 类型**：`image/*`
- **预览方式**：直接在浏览器中显示

#### 视频类型
- **格式**：MP4, WebM, OGG
- **MIME 类型**：`video/*`
- **预览方式**：HTML5 视频播放器

#### 音频类型
- **格式**：MP3, WAV, OGG, AAC
- **MIME 类型**：`audio/*`
- **预览方式**：HTML5 音频播放器

#### PDF 文档
- **格式**：PDF
- **MIME 类型**：`application/pdf`
- **预览方式**：PDF.js 渲染

#### 文本文件
- **格式**：TXT, JSON, XML, CSV, LOG
- **MIME 类型**：`text/*`
- **预览方式**：语法高亮显示

### 不支持预览的文件类型

以下文件类型需要下载后查看：

- Office 文档（Word, Excel, PowerPoint）
- 压缩文件（ZIP, RAR, 7Z）
- 可执行文件（EXE, DMG, APK）
- 其他二进制文件

### 预览示例

```vue
<template>
  <FilePreviewModal
    :visible="showPreview"
    :file="selectedFile"
    @close="showPreview = false"
  />
</template>

<script setup lang="ts">
import FilePreviewModal from '@/components/apps/file/FilePreviewModal.vue'
import { ref } from 'vue'

const showPreview = ref(false)
const selectedFile = ref(null)
</script>
```

## 最佳实践

### 1. 文件大小限制

```typescript
// 建议在前端进行文件大小检查
const MAX_FILE_SIZE = 2 * 1024 * 1024 * 1024 // 2GB

const handleFileSelect = (file: File) => {
  if (file.size > MAX_FILE_SIZE) {
    alert('文件大小不能超过 2GB')
    return
  }
  // 继续上传
}
```

### 2. 文件类型验证

```typescript
// 验证文件类型
const ALLOWED_TYPES = [
  'image/jpeg',
  'image/png',
  'application/pdf',
  'video/mp4'
]

const validateFileType = (file: File): boolean => {
  return ALLOWED_TYPES.includes(file.type)
}
```

### 3. 批量上传

```typescript
const handleBatchUpload = async (files: FileList, folderId?: number) => {
  const uploadPromises = Array.from(files).map(file => 
    uploadFile(file, folderId)
  )
  
  try {
    const results = await Promise.allSettled(uploadPromises)
    const succeeded = results.filter(r => r.status === 'fulfilled')
    const failed = results.filter(r => r.status === 'rejected')
    
    console.log(`成功上传 ${succeeded.length} 个文件`)
    console.log(`失败 ${failed.length} 个文件`)
  } catch (error) {
    console.error('批量上传出错:', error)
  }
}
```

### 4. 错误处理

```typescript
const handleUpload = async (file: File) => {
  try {
    const result = await uploadFile(file)
    // 上传成功
    console.log('上传成功:', result)
  } catch (error) {
    if (error instanceof Error) {
      // 显示用户友好的错误信息
      if (error.message.includes('网络')) {
        alert('网络连接失败，请检查网络后重试')
      } else if (error.message.includes('大小')) {
        alert('文件大小超出限制')
      } else {
        alert('上传失败：' + error.message)
      }
    }
  }
}
```

### 5. 上传进度显示

```vue
<template>
  <div v-for="task in activeTasks" :key="task.id" class="upload-task">
    <div class="task-info">
      <span class="filename">{{ task.filename }}</span>
      <span class="progress">{{ task.progress }}%</span>
    </div>
    <div class="progress-bar">
      <div class="progress-fill" :style="{ width: task.progress + '%' }"></div>
    </div>
    <button @click="cancelUpload(task.id)">取消</button>
  </div>
</template>
```

### 6. 网络状态监听

```typescript
// 监听网络状态变化
window.addEventListener('online', () => {
  console.log('网络已恢复')
  // 可以提示用户继续上传
})

window.addEventListener('offline', () => {
  console.log('网络已断开')
  // 可以暂停上传或提示用户
})
```

## 故障排查

### 常见问题

#### 1. 上传速度慢

**可能原因**：
- 网络带宽限制
- 服务器负载高
- 分片大小不合理

**解决方案**：
- 检查网络连接速度
- 尝试在网络低峰期上传
- 系统会自动选择最优分片策略

#### 2. 上传失败

**可能原因**：
- 网络连接中断
- 文件大小超出限制
- 服务器存储空间不足
- 文件类型不支持

**解决方案**：
- 检查网络连接
- 确认文件大小在限制范围内
- 联系管理员检查服务器存储
- 查看错误日志获取详细信息

#### 3. 秒传不生效

**可能原因**：
- 文件被修改过
- MD5 计算错误
- 服务器未找到相同文件

**解决方案**：
- 确认文件未被修改
- 检查 MD5 计算是否正确
- 联系管理员检查服务器文件库

#### 4. 断点续传失败

**可能原因**：
- 上传记录已过期
- 服务器清理了临时文件
- 分片信息不匹配

**解决方案**：
- 重新开始上传
- 联系管理员检查服务器配置
- 清除浏览器缓存后重试

#### 5. 文件预览失败

**可能原因**：
- 文件格式不支持
- 文件损坏
- 浏览器不支持该格式

**解决方案**：
- 下载文件后使用本地应用打开
- 检查文件是否完整
- 尝试使用其他浏览器

### 调试技巧

#### 1. 查看上传任务状态

```typescript
const { tasks } = useFileUpload()

// 在控制台查看所有任务
console.log('上传任务:', tasks.value)
```

#### 2. 监控上传进度

```typescript
watch(() => tasks.value, (newTasks) => {
  console.log('任务状态变化:', newTasks)
}, { deep: true })
```

#### 3. 查看网络请求

- 打开浏览器开发者工具
- 切换到 Network 标签
- 筛选 XHR 请求
- 查看上传请求的详细信息

#### 4. 检查文件 MD5

```typescript
import { calculateMD5 } from '@/composables/useFileUpload'

const fileHash = await calculateMD5(file)
console.log('文件 MD5:', fileHash)
```

## 技术细节

### 架构设计

#### 1. 分层架构

```
┌─────────────────────────────────┐
│         UI Components           │  用户界面层
├─────────────────────────────────┤
│       useFileUpload Hook        │  业务逻辑层
├─────────────────────────────────┤
│   Upload Queue Manager          │  队列管理层
├─────────────────────────────────┤
│   MD5 Worker / Chunk Utils      │  工具层
├─────────────────────────────────┤
│         File API                │  API 层
└─────────────────────────────────┘
```

#### 2. 核心模块

- **MD5 Worker**：Web Worker 后台计算 MD5，避免阻塞主线程
- **Upload Queue Manager**：队列管理器，控制并发上传
- **Upload Store**：Pinia 状态管理，管理上传任务状态
- **File API**：封装文件上传相关的 API 请求

### 关键技术点

#### 1. Web Worker 计算 MD5

```typescript
// 使用 Web Worker 在后台线程计算 MD5
const worker = new Worker(new URL('../workers/md5.worker.ts', import.meta.url))
worker.postMessage({ file })
worker.onmessage = (e) => {
  const hash = e.data.hash
  // 使用计算结果
}
```

#### 2. 文件分片

```typescript
// 使用 Blob.slice 实现零拷贝分片
const chunk = file.slice(start, end)
```

#### 3. 并发控制

```typescript
// 队列管理器控制并发
class UploadQueueManager {
  private activeCount = 0
  private maxConcurrent = 5
  
  private processQueue() {
    while (queue.length > 0 && activeCount < maxConcurrent) {
      // 启动新的上传任务
      activeCount++
      uploadChunk().then(() => {
        activeCount--
        this.processQueue() // 继续处理队列
      })
    }
  }
}
```

#### 4. 内存优化

```typescript
// 上传完成后释放分片引用
this.manager.chunks[chunkIndex] = new Blob([])
```

### 性能优化

#### 1. 动态并发数

- 根据 CPU 核心数自动调整并发数
- 最小 2 个，最大 5 个并发上传
- 公式：`Math.min(Math.max(Math.floor(cpuCores / 2), 2), 5)`

#### 2. 智能分片

- 根据文件大小选择最优分片大小
- 小文件不分片，减少开销
- 大文件使用较大分片，减少请求数

#### 3. 内存管理

- 及时释放已上传分片的内存引用
- 避免同时加载所有分片到内存
- 使用 Blob.slice 实现零拷贝

#### 4. 错误重试

- 失败的分片自动重试
- 最多重试 3 次
- 指数退避策略（1s, 2s, 3s）

### 安全性

#### 1. 文件验证

- 服务器端验证文件类型
- 验证文件大小限制
- 验证用户权限

#### 2. MD5 校验

- 上传前计算文件 MD5
- 服务器端验证 MD5 一致性
- 确保文件完整性

#### 3. 分片验证

- 每个分片计算 MD5
- 服务器验证分片完整性
- 防止数据损坏

### 兼容性

#### 浏览器支持

- Chrome 60+
- Firefox 55+
- Safari 11+
- Edge 79+

#### API 依赖

- Web Workers
- Blob API
- FileReader API
- Fetch API
- navigator.hardwareConcurrency

### 扩展性

#### 1. 自定义分片策略

```typescript
// 可以根据需求自定义分片策略
function customChunkStrategy(fileSize: number): number {
  // 自定义逻辑
  return chunkSize
}
```

#### 2. 自定义上传处理

```typescript
// 可以在上传前后添加自定义处理
const customUpload = async (file: File) => {
  // 上传前处理
  await beforeUpload(file)
  
  // 执行上传
  const result = await uploadFile(file)
  
  // 上传后处理
  await afterUpload(result)
  
  return result
}
```

#### 3. 集成第三方存储

- 可以扩展 API 层支持其他存储服务
- 如 OSS、S3、Azure Blob 等
- 只需实现统一的文件上传接口

## 总结

QIM 文件上传功能提供了完整的文件管理解决方案，通过智能分片、秒传、断点续传等特性，确保文件传输的高效性和可靠性。系统自动优化性能，开发者只需关注业务逻辑即可。

如有问题或建议，请联系开发团队。
