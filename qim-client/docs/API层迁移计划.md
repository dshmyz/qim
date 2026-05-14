# API 层迁移计划

## 📋 概述

本文档描述如何逐步将现有代码迁移到新的统一 API 层（`api/core.ts`）。

## 🎯 迁移目标

1. **统一 API 调用方式** - 所有 HTTP 请求通过 `api/core.ts` 发起
2. **统一错误处理** - 使用 `ApiError` 类处理所有错误
3. **增强类型安全** - 为所有 API 调用添加完整类型定义
4. **提升代码质量** - 减少重复代码，提高可维护性

## 📊 当前状态分析

### 现有 API 调用方式

| 方式 | 文件数 | 说明 |
|------|--------|------|
| `useRequest` | 17 | 使用自定义 composable |
| `axios` 直接调用 | 5 | 在组件/工具中直接使用 axios |
| 总计 | 22 | 需要迁移的文件 |

### 涉及的文件

#### 使用 `useRequest` 的文件（17 个）
```
src/composables/useConversation.ts ✅ 已优化
src/composables/useGroupOperations.ts
src/composables/useGroup.ts
src/composables/useMessageActions.ts
src/composables/useBotChat.ts
src/composables/useSettings.ts
src/composables/useBots.ts
src/composables/useNotes.ts
src/composables/useOrganization.ts
src/composables/useFolderTree.ts
src/composables/useAIActions.ts
src/composables/useChat.ts
src/stores/channel.ts
src/api/avatar.ts
src/api/task.ts
src/api/realtime.ts
```

#### 直接使用 `axios` 的文件（5 个）
```
src/api/file.ts ⚠️ 需要迁移
src/api/ai.ts ⚠️ 需要迁移
src/utils/axios.ts
src/utils/version.ts
```

## 🚀 分阶段迁移策略

### 第一阶段：基础模块迁移（优先级：高）

#### 1.1 创建 API 模块文件

创建以下 API 模块文件：

```
src/api/
├── modules/                    # API 模块目录
│   ├── conversation.ts         # 会话相关 API
│   ├── message.ts             # 消息相关 API ✅ 已创建
│   ├── user.ts                # 用户相关 API
│   ├── group.ts               # 群组相关 API
│   ├── organization.ts        # 组织架构 API
│   ├── notification.ts       # 通知相关 API
│   └── settings.ts            # 设置相关 API
```

#### 1.2 迁移文件 API（优先级：最高）

**目标文件**: `src/api/file.ts`

**当前代码**:
```typescript
import axios from 'axios'
import { API_BASE_URL } from '../config'

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})
```

**迁移后**:
```typescript
// src/api/modules/file.ts
import { http } from '../core'
import type { ApiResponse } from '../core'

export interface FileItem {
  id: number
  name: string
  size: number
  mime_type: string
  url: string
}

export const fileApi = {
  async getFiles(params?: FileListParams) {
    return http.get<ApiResponse<FileListResponse>>('/api/v1/files', { params })
  },

  async uploadFile(file: File, folderId?: number) {
    const formData = new FormData()
    formData.append('file', file)
    if (folderId) formData.append('folder_id', String(folderId))
    
    return http.post('/api/v1/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  async deleteFile(fileId: number) {
    return http.delete(`/api/v1/files/${fileId}`)
  }
}
```

#### 1.3 迁移 AI API（优先级：高）

**目标文件**: `src/api/ai.ts`

**迁移步骤**:

1. 创建 `src/api/modules/ai.ts`
2. 将所有 AI 相关的 API 调用迁移到新模块
3. 添加完整的类型定义
4. 替换现有引用

### 第二阶段：Composables 迁移（优先级：中）

#### 2.1 迁移策略

对于每个使用 `useRequest` 的 composable：

1. **识别 API 调用** - 找出文件中所有的 `request()` 调用
2. **创建 API 模块** - 为每个 API 调用创建对应的 API 模块
3. **更新 composable** - 使用新的 API 模块替换直接请求

#### 2.2 迁移示例：`useConversation.ts`

**当前代码**:
```typescript
// src/composables/useConversation.ts
import { request } from './useRequest'

const handlePin = async (conversation: Conversation) => {
  try {
    await request(`/api/v1/conversations/${conversation.id}/pin`, {
      method: 'PUT',
      body: JSON.stringify({ is_pinned: !conversation.is_pinned })
    })
    chatStore.pinConversation(conversation.id, !conversation.is_pinned)
  } catch (error) {
    console.error('置顶会话失败:', error)
  }
}
```

**迁移后**:
```typescript
// src/composables/useConversation.ts
import { conversationApi } from '@/api/modules/conversation'

const handlePin = async (conversation: Conversation) => {
  try {
    await conversationApi.pinConversation(conversation.id, !conversation.is_pinned)
    chatStore.pinConversation(conversation.id, !conversation.is_pinned)
  } catch (error) {
    if (error instanceof ApiError) {
      QMessage.error(error.message)
    } else {
      console.error('置顶会话失败:', error)
    }
  }
}
```

#### 2.3 具体迁移顺序

```
阶段 2.1 - 消息相关（优先级：高）
├── src/composables/useMessageActions.ts
├── src/composables/useChat.ts
└── src/composables/useBotChat.ts

阶段 2.2 - 会话相关（优先级：高）
├── src/composables/useConversation.ts ✅ 已使用 store
└── src/composables/useGroup.ts

阶段 2.3 - 用户相关（优先级：中）
├── src/composables/useSettings.ts
├── src/composables/useBots.ts
└── src/stores/channel.ts

阶段 2.4 - 其他（优先级：低）
├── src/composables/useNotes.ts
├── src/composables/useOrganization.ts
├── src/composables/useFolderTree.ts
├── src/composables/useAIActions.ts
├── src/composables/useGroupOperations.ts
└── src/api/avatar.ts
```

### 第三阶段：工具函数迁移（优先级：低）

#### 3.1 需要迁移的文件

- `src/utils/version.ts` - 版本检查 API
- `src/utils/axios.ts` - 自定义 axios 实例（考虑删除）

#### 3.2 迁移策略

如果 `utils/version.ts` 包含简单的 API 调用，可以：
1. 将其迁移到 `api/modules/system.ts`
2. 或保留为简单的 fetch 调用

## 📝 迁移步骤详解

### 步骤 1：创建 API 模块

为每个功能域创建 API 模块：

```typescript
// src/api/modules/[feature].ts
import { http } from '../core'
import type { ApiResponse } from '../core'

export interface [Feature]Item {
  id: number
  // ... 其他字段
}

class [Feature]API {
  async list(params?: any) {
    return http.get<ApiResponse<[Feature]Item[]>>('/api/v1/[features]', { params })
  }

  async get(id: number) {
    return http.get<ApiResponse<[Feature]Item>>(`/api/v1/[features]/${id}`)
  }

  async create(data: Partial<[Feature]Item>) {
    return http.post<ApiResponse<[Feature]Item>>('/api/v1/[features]', data)
  }

  async update(id: number, data: Partial<[Feature]Item>) {
    return http.put<ApiResponse<[Feature]Item>>(`/api/v1/[features]/${id}`, data)
  }

  async delete(id: number) {
    return http.delete(`/api/v1/[features]/${id}`)
  }
}

export const [feature]Api = new [Feature]API()
```

### 步骤 2：更新 Composables

```typescript
// src/composables/use[Feature].ts
import { ref, computed } from 'vue'
import { [feature]Api } from '@/api/modules/[feature]'
import type { [Feature]Item } from '@/api/modules/[feature]'
import { ApiError } from '@/api/core'

export function use[Feature]() {
  const items = ref<[Feature]Item[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchItems() {
    loading.value = true
    error.value = null
    
    try {
      const response = await [feature]Api.list()
      items.value = response.data
    } catch (err) {
      if (err instanceof ApiError) {
        error.value = err.message
        console.error('获取数据失败:', err.message)
      } else {
        error.value = '网络错误'
        console.error('获取数据失败:', err)
      }
    } finally {
      loading.value = false
    }
  }

  return {
    items,
    loading,
    error,
    fetchItems
  }
}
```

### 步骤 3：更新组件引用

在组件中更新引用：

```typescript
// 旧写法
import { request } from '@/composables/useRequest'

// 新写法
import { [feature]Api } from '@/api/modules/[feature]'
// 或
import { http } from '@/api/core'
```

### 步骤 4：删除旧代码

确认所有旧代码已迁移后，可以安全删除：

- `src/composables/useRequest.ts` - 可选择保留或删除
- 旧的自定义 axios 实例

## ⚠️ 注意事项

### 1. 向后兼容性

在迁移过程中，保持 API 接口不变：
- URL 路径保持一致
- 请求方法保持一致
- 响应格式保持一致

### 2. 错误处理

统一使用 `ApiError` 类：

```typescript
import { ApiError } from '@/api/core'

try {
  await apiCall()
} catch (error) {
  if (error instanceof ApiError) {
    // 处理业务错误
    console.error(error.code, error.message)
  } else {
    // 处理未知错误
    console.error('未知错误:', error)
  }
}
```

### 3. 类型安全

为所有 API 调用添加类型定义：

```typescript
// Good
const response = await http.get<ApiResponse<User>>('/api/v1/users/1')

// Avoid
const response = await http.get('/api/v1/users/1') // 缺少类型
```

### 4. 测试

在迁移过程中：
- 手动测试每个迁移的 API
- 添加单元测试覆盖新的 API 模块
- 确保错误处理正确工作

## 📅 实施时间表

| 阶段 | 内容 | 优先级 | 预计工作量 |
|------|------|--------|-----------|
| 第一阶段 | 文件 API、AI API 迁移 | 🔴 高 | 4-6 小时 |
| 第二阶段 | Composables 迁移 | 🟡 中 | 8-10 小时 |
| 第三阶段 | 工具函数迁移 | 🟢 低 | 2-3 小时 |
| 收尾 | 清理旧代码、文档更新 | 🟢 低 | 2-3 小时 |

**总计**: 约 16-22 小时

## 🎯 验收标准

### 完成标志

1. ✅ 所有 HTTP 请求通过 `api/core.ts` 发起
2. ✅ 所有错误通过 `ApiError` 类处理
3. ✅ 所有 API 调用具有完整的类型定义
4. ✅ 旧代码（useRequest）已移除或标记为废弃
5. ✅ 单元测试覆盖率达到 80% 以上

### 质量检查清单

- [ ] `src/api/modules/` 包含所有功能域的 API 模块
- [ ] 所有 composables 使用新的 API 模块
- [ ] 所有组件更新为使用新 API
- [ ] 错误处理统一使用 `ApiError` 类
- [ ] 文档已更新
- [ ] 单元测试已添加
- [ ] 手动测试通过

## 🔧 工具和脚本

### 迁移辅助脚本

如需要，可以创建以下脚本辅助迁移：

```bash
# 检查未迁移的 useRequest 使用
grep -r "from.*useRequest" src/**/*.ts | wc -l

# 检查未迁移的 axios 直接使用
grep -r "axios\." src/**/*.ts | grep -v node_modules | wc -l
```

### VS Code 扩展建议

- **TypeScript Vue Plugin (Volar)**
- **Vue - Official**
- **ESLint**
- **Prettier**

## 📚 相关文档

- [API 层设计文档](api-design.md)
- [组件拆分指南](../docs/组件拆分指南.md)
- [类型定义规范](../docs/类型定义规范.md)

## ❓ 常见问题

### Q1: 迁移过程中可以提交代码吗？

**A**: 可以。建议：
- 每迁移一个功能模块后提交一次
- 使用清晰的 commit message
- 包含迁移前后的对比说明

### Q2: 如何处理临时的兼容性问题？

**A**: 可以使用以下策略：
1. 创建适配层（Adapter Pattern）
2. 逐步替换，而不是一次性全部替换
3. 保持新旧代码并行，直到完全迁移

### Q3: 迁移后性能会提升吗？

**A**: 预期会有小幅提升：
- 减少重复的 axios 实例创建
- 统一的请求拦截器
- 更好的缓存策略

但主要收益是代码质量和可维护性的提升。

## 📝 更新日志

| 日期 | 版本 | 更新内容 | 作者 |
|------|------|---------|------|
| 2026-05-11 | 1.0 | 初始版本 | Claude |

---

**最后更新**: 2026-05-11
