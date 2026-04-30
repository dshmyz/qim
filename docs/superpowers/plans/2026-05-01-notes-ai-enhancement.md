# 笔记模块 AI 增强实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 QIM 笔记模块增加 AI 分析、标签系统、导出功能，优化界面布局

**架构：** 前端采用组件化拆分，将 NotesApp.vue 拆分为多个子组件；后端复用现有 AI 服务，新增分析和导出 API

**技术栈：** Vue 3 + TypeScript + Go + GORM

---

## 文件结构

### 后端文件

| 文件 | 职责 | 操作 |
|------|------|------|
| `qim-server/model/model.go` | Note 模型定义 | 修改：增加 Tags、Summary 字段 |
| `qim-server/handler/note_handler.go` | 笔记相关 API | 创建：新的 handler 文件 |
| `qim-server/app/routes.go` | 路由注册 | 修改：增加新 API 路由 |

### 前端文件

| 文件 | 职责 | 操作 |
|------|------|------|
| `qim-client/src/components/apps/NotesApp.vue` | 主组件 | 重构：拆分为子组件 |
| `qim-client/src/components/apps/notes/NoteEditor.vue` | 编辑器组件 | 创建 |
| `qim-client/src/components/apps/notes/NoteToolbar.vue` | 工具栏组件 | 创建 |
| `qim-client/src/components/apps/notes/NoteCard.vue` | 笔记卡片组件 | 创建 |
| `qim-client/src/components/apps/notes/NoteTagFilter.vue` | 标签筛选器 | 创建 |
| `qim-client/src/components/apps/notes/AIAnalysisModal.vue` | AI 分析弹窗 | 创建 |
| `qim-client/src/composables/useNotes.ts` | 笔记相关逻辑 | 创建 |
| `qim-client/src/types/note.ts` | 类型定义 | 创建 |

---

## 任务 1：后端 - Note 模型增加字段

**文件：**
- 修改：`qim-server/model/model.go:153-163`

- [ ] **步骤 1：修改 Note 模型**

在 `model.go` 中找到 `type Note struct`，修改为：

```go
type Note struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Title     string         `json:"title" gorm:"size:500;not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	Type      string         `json:"type" gorm:"size:20;default:'note'"`
	Style     string         `json:"style" gorm:"type:text;default:'{}'"`
	Tags      string         `json:"tags" gorm:"type:text"`           // JSON 数组字符串
	Summary   string         `json:"summary" gorm:"type:text"`        // AI 生成的摘要
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
```

- [ ] **步骤 2：运行数据库迁移**

```bash
cd qim-server && go run . --migrate
```

预期：数据库 notes 表增加 tags 和 summary 列

- [ ] **步骤 3：Commit**

```bash
git add qim-server/model/model.go
git commit -m "feat(note): add tags and summary fields to Note model"
```

---

## 任务 2：后端 - 创建笔记 Handler 文件

**文件：**
- 创建：`qim-server/handler/note_handler.go`

- [ ] **步骤 1：创建 note_handler.go**

```go
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

type NoteTagsRequest struct {
	Tags []string `json:"tags"`
}

type NoteSummaryRequest struct {
	Summary string `json:"summary"`
}

type AIAnalyzeRequest struct {
	Content string `json:"content"`
}

type AIAnalyzeResponse struct {
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	ActionItems []string `json:"action_items"`
}

func AnalyzeNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "笔记不存在"})
		return
	}

	aiService := c.MustGet("ai_service").(*ai.AIService)
	if aiService == nil || !aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "AI 服务未配置"})
		return
	}

	systemPrompt := `你是一个笔记分析助手。分析以下笔记内容，返回 JSON 格式结果：
1. summary: 笔记摘要（100字以内）
2. tags: 推荐标签（最多5个，简洁明了）
3. action_items: 提取的行动项（如果有，最多5个）

只返回 JSON，格式：{"summary": "...", "tags": ["标签1", "标签2"], "action_items": ["行动项1"]}`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: note.Content},
	}

	result, err := aiService.GetCompletion(messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI 分析失败"})
		return
	}

	var analyzeResult AIAnalyzeResponse
	jsonStr := result
	if idx := findJSONStart(result); idx >= 0 {
		jsonStr = result[idx:]
		if endIdx := findJSONEnd(jsonStr); endIdx >= 0 {
			jsonStr = jsonStr[:endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &analyzeResult); err != nil {
		analyzeResult = AIAnalyzeResponse{
			Summary:     result[:min(100, len(result))],
			Tags:        []string{},
			ActionItems: []string{},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": analyzeResult,
	})
}

func ExportNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "笔记不存在"})
		return
	}

	filename := fmt.Sprintf("%s.md", note.Title)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.String(http.StatusOK, note.Content)
}

func UpdateNoteTags(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	var req NoteTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	tagsJSON, _ := json.Marshal(req.Tags)

	db := database.GetDB()
	if err := db.Model(&model.Note{}).Where("id = ? AND user_id = ?", uint(noteID), userID).Update("tags", string(tagsJSON)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func UpdateNoteSummary(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	var req NoteSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	if err := db.Model(&model.Note{}).Where("id = ? AND user_id = ?", uint(noteID), userID).Update("summary", req.Summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' || c == '[' {
			return i
		}
	}
	return -1
}

func findJSONEnd(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '}' || s[i] == ']' {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/handler/note_handler.go
git commit -m "feat(note): add note handler with AI analyze and export APIs"
```

---

## 任务 3：后端 - 注册路由

**文件：**
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：添加新路由**

在 `routes.go` 中找到笔记相关路由（约 196-200 行），添加新路由：

```go
// 笔记管理
authed.GET("/notes", handler.GetNotes)
authed.GET("/notes/:id", handler.GetNote)
authed.POST("/notes", handler.CreateNote)
authed.PUT("/notes/:id", handler.UpdateNote)
authed.DELETE("/notes/:id", handler.DeleteNote)
authed.POST("/notes/:id/analyze", handler.AnalyzeNote)
authed.GET("/notes/:id/export", handler.ExportNote)
authed.PATCH("/notes/:id/tags", handler.UpdateNoteTags)
authed.PATCH("/notes/:id/summary", handler.UpdateNoteSummary)
```

- [ ] **步骤 2：验证编译**

```bash
cd qim-server && go build .
```

预期：编译成功

- [ ] **步骤 3：Commit**

```bash
git add qim-server/app/routes.go
git commit -m "feat(note): add routes for AI analyze, export, tags and summary"
```

---

## 任务 4：前端 - 创建类型定义

**文件：**
- 创建：`qim-client/src/types/note.ts`

- [ ] **步骤 1：创建类型文件**

```typescript
export interface Note {
  id: number
  user_id: number
  title: string
  content: string
  type: 'note' | 'sticky'
  style: string
  tags: string[]
  summary: string
  created_at: string
  updated_at: string
}

export interface AIAnalyzeResult {
  summary: string
  tags: string[]
  action_items: string[]
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/types/note.ts
git commit -m "feat(note): add Note and AIAnalyzeResult types"
```

---

## 任务 5：前端 - 创建 useNotes composable

**文件：**
- 创建：`qim-client/src/composables/useNotes.ts`

- [ ] **步骤 1：创建 composable**

```typescript
import { ref } from 'vue'
import { useRequest } from './useRequest'
import type { Note, AIAnalyzeResult } from '../types/note'

export function useNotes() {
  const { get, post, patch, serverUrl } = useRequest()
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchNotes = async (): Promise<Note[]> => {
    loading.value = true
    error.value = null
    try {
      const response = await get<any>('/api/v1/notes')
      const notes = response.data || []
      return notes.map((n: any) => ({
        ...n,
        tags: parseTags(n.tags)
      }))
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const createNote = async (data: Partial<Note>): Promise<Note | null> => {
    loading.value = true
    error.value = null
    try {
      const response = await post<any>('/api/v1/notes', {
        title: data.title || '新笔记',
        content: data.content || '',
        type: data.type || 'note',
        tags: JSON.stringify(data.tags || [])
      })
      return { ...response.data, tags: parseTags(response.data.tags) }
    } catch (e: any) {
      error.value = e.message
      return null
    } finally {
      loading.value = false
    }
  }

  const updateNote = async (id: number, data: Partial<Note>): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      await post<any>(`/api/v1/notes/${id}`, {
        title: data.title,
        content: data.content,
        tags: JSON.stringify(data.tags || [])
      })
      return true
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const deleteNote = async (id: number): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      await get<any>(`/api/v1/notes/${id}`, { method: 'DELETE' })
      return true
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const analyzeNote = async (id: number): Promise<AIAnalyzeResult | null> => {
    loading.value = true
    error.value = null
    try {
      const response = await post<any>(`/api/v1/notes/${id}/analyze`, {})
      return response.data
    } catch (e: any) {
      error.value = e.message
      return null
    } finally {
      loading.value = false
    }
  }

  const updateNoteTags = async (id: number, tags: string[]): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      await patch<any>(`/api/v1/notes/${id}/tags`, { tags })
      return true
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const updateNoteSummary = async (id: number, summary: string): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      await patch<any>(`/api/v1/notes/${id}/summary`, { summary })
      return true
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const exportNote = (id: number, title: string) => {
    const token = localStorage.getItem('token')
    const url = `${serverUrl.value}/api/v1/notes/${id}/export`
    
    const link = document.createElement('a')
    link.href = url
    link.download = `${title}.md`
    link.click()
  }

  return {
    loading,
    error,
    fetchNotes,
    createNote,
    updateNote,
    deleteNote,
    analyzeNote,
    updateNoteTags,
    updateNoteSummary,
    exportNote
  }
}

function parseTags(tags: any): string[] {
  if (!tags) return []
  if (Array.isArray(tags)) return tags
  if (typeof tags === 'string') {
    try {
      const parsed = JSON.parse(tags)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/composables/useNotes.ts
git commit -m "feat(note): add useNotes composable for note operations"
```

---

## 任务 6：前端 - 创建 NoteCard 组件

**文件：**
- 创建：`qim-client/src/components/apps/notes/NoteCard.vue`

- [ ] **步骤 1：创建组件**

```vue
<template>
  <div
    class="note-card"
    :class="{ active: isActive }"
    @click="$emit('select')"
  >
    <div class="note-card-header">
      <h3 class="note-title">{{ note.title }}</h3>
      <div class="note-actions" v-if="showActions">
        <button class="note-action-btn" @click.stop="$emit('edit')" title="编辑">
          <i class="fas fa-edit"></i>
        </button>
        <button class="note-action-btn delete" @click.stop="$emit('delete')" title="删除">
          <i class="fas fa-trash"></i>
        </button>
      </div>
    </div>
    <p class="note-summary">{{ displaySummary }}</p>
    <div class="note-tags" v-if="note.tags && note.tags.length > 0">
      <span
        v-for="tag in note.tags"
        :key="tag"
        class="note-tag"
        @click.stop="$emit('filter-tag', tag)"
      >
        {{ tag }}
      </span>
    </div>
    <div class="note-footer">
      <span class="note-date">{{ formatDate(note.updated_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Note } from '../../../types/note'

const props = defineProps<{
  note: Note
  isActive?: boolean
}>()

defineEmits<{
  select: []
  edit: []
  delete: []
  'filter-tag': [tag: string]
}>()

const showActions = ref(false)

const displaySummary = computed(() => {
  if (props.note.summary) {
    return props.note.summary
  }
  const content = props.note.content || ''
  return content.length > 50 ? content.substring(0, 50) + '...' : content
})

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped>
.note-card {
  padding: 16px;
  margin-bottom: 8px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.note-card:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
}

.note-card:hover .note-actions {
  opacity: 1;
}

.note-card.active {
  background: var(--hover-color);
  border-color: var(--primary-color);
}

.note-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
}

.note-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.note-actions {
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.note-action-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.note-action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.note-action-btn.delete:hover {
  color: var(--danger-color);
}

.note-summary {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 8px 0;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.note-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.note-tag {
  font-size: 12px;
  padding: 2px 8px;
  background: var(--primary-light);
  color: var(--primary-color);
  border-radius: 10px;
  cursor: pointer;
}

.note-tag:hover {
  background: var(--primary-color);
  color: white;
}

.note-footer {
  display: flex;
  justify-content: flex-end;
}

.note-date {
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/apps/notes/NoteCard.vue
git commit -m "feat(note): add NoteCard component"
```

---

## 任务 7：前端 - 创建 NoteToolbar 组件

**文件：**
- 创建：`qim-client/src/components/apps/notes/NoteToolbar.vue`

- [ ] **步骤 1：创建组件**

```vue
<template>
  <div class="note-toolbar">
    <div class="toolbar-left">
      <button
        :class="['mode-btn', { active: mode === 'edit' }]"
        @click="$emit('update:mode', 'edit')"
      >
        <i class="fas fa-edit"></i>
        编辑
      </button>
      <button
        :class="['mode-btn', { active: mode === 'preview' }]"
        @click="$emit('update:mode', 'preview')"
      >
        <i class="fas fa-eye"></i>
        预览
      </button>
    </div>
    <div class="toolbar-right">
      <button class="toolbar-btn save" @click="$emit('save')" :disabled="saving">
        <i class="fas fa-save"></i>
        {{ saving ? '保存中...' : '保存' }}
      </button>
      <button class="toolbar-btn ai" @click="$emit('analyze')" :disabled="analyzing">
        <i class="fas fa-magic"></i>
        {{ analyzing ? '分析中...' : 'AI 分析' }}
      </button>
      <button class="toolbar-btn export" @click="$emit('export')">
        <i class="fas fa-download"></i>
        导出
      </button>
      <button class="toolbar-btn share" @click="$emit('share')">
        <i class="fas fa-share-alt"></i>
        分享
      </button>
      <button class="toolbar-btn delete" @click="$emit('delete')">
        <i class="fas fa-trash"></i>
        删除
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  mode: 'edit' | 'preview'
  saving?: boolean
  analyzing?: boolean
}>()

defineEmits<{
  'update:mode': [mode: 'edit' | 'preview']
  save: []
  analyze: []
  export: []
  share: []
  delete: []
}>()
</script>

<style scoped>
.note-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  margin-bottom: 16px;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  gap: 8px;
}

.mode-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  background: var(--bg-color);
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.mode-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.mode-btn.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.toolbar-btn {
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  background: var(--bg-color);
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.toolbar-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.toolbar-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toolbar-btn.save:hover:not(:disabled) {
  background: var(--success-color);
  border-color: var(--success-color);
  color: white;
}

.toolbar-btn.ai:hover:not(:disabled) {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.toolbar-btn.delete:hover:not(:disabled) {
  background: var(--danger-color);
  border-color: var(--danger-color);
  color: white;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/apps/notes/NoteToolbar.vue
git commit -m "feat(note): add NoteToolbar component"
```

---

## 任务 8：前端 - 创建 NoteEditor 组件

**文件：**
- 创建：`qim-client/src/components/apps/notes/NoteEditor.vue`

- [ ] **步骤 1：创建组件**

```vue
<template>
  <div class="note-editor">
    <input
      v-model="localTitle"
      class="note-title-input"
      placeholder="笔记标题"
      @input="$emit('update:title', localTitle)"
    />
    <div v-if="mode === 'edit'" class="editor-area">
      <div class="editor-toolbar">
        <button class="format-btn" @click="insertFormat('**', '**')" title="粗体">
          <strong>B</strong>
        </button>
        <button class="format-btn" @click="insertFormat('*', '*')" title="斜体">
          <em>I</em>
        </button>
        <button class="format-btn" @click="insertFormat('# ', '')" title="标题">
          H
        </button>
        <button class="format-btn" @click="insertFormat('- ', '')" title="列表">
          <i class="fas fa-list"></i>
        </button>
        <button class="format-btn" @click="insertFormat('`', '`')" title="代码">
          <i class="fas fa-code"></i>
        </button>
        <button class="format-btn" @click="insertFormat('```\\n', '\\n```')" title="代码块">
          <i class="fas fa-file-code"></i>
        </button>
      </div>
      <textarea
        ref="textareaRef"
        v-model="localContent"
        class="note-content-input"
        placeholder="使用 Markdown 编写笔记..."
        @input="$emit('update:content', localContent)"
      ></textarea>
    </div>
    <div v-else class="preview-area">
      <div class="preview-content" v-html="renderedContent"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { sanitizeMarkdown } from '../../../utils/sanitize'

const props = defineProps<{
  title: string
  content: string
  mode: 'edit' | 'preview'
}>()

const emit = defineEmits<{
  'update:title': [title: string]
  'update:content': [content: string]
}>()

const localTitle = ref(props.title)
const localContent = ref(props.content)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

watch(() => props.title, (val) => { localTitle.value = val })
watch(() => props.content, (val) => { localContent.value = val })

const renderedContent = computed(() => {
  return sanitizeMarkdown(renderMarkdown(localContent.value))
})

function renderMarkdown(content: string): string {
  let html = content
  
  html = html.replace(/^# (.*$)/gm, '<h1>$1</h1>')
  html = html.replace(/^## (.*$)/gm, '<h2>$1</h2>')
  html = html.replace(/^### (.*$)/gm, '<h3>$1</h3>')
  html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')
  html = html.replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
  html = html.replace(/`(.*?)`/g, '<code>$1</code>')
  html = html.replace(/^- (.*$)/gm, '<li>$1</li>')
  html = html.replace(/\[(.*?)\]\((.*?)\)/g, '<a href="$2" target="_blank">$1</a>')
  html = html.replace(/\n/g, '<br>')
  
  return html
}

function insertFormat(prefix: string, suffix: string) {
  if (!textareaRef.value) return
  
  const textarea = textareaRef.value
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const selectedText = localContent.value.substring(start, end)
  
  const newContent = 
    localContent.value.substring(0, start) +
    prefix + selectedText + suffix +
    localContent.value.substring(end)
  
  localContent.value = newContent
  emit('update:content', newContent)
  
  setTimeout(() => {
    textarea.focus()
    textarea.setSelectionRange(start + prefix.length, end + prefix.length)
  }, 0)
}
</script>

<style scoped>
.note-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.note-title-input {
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  background: var(--bg-color);
  outline: none;
  transition: all 0.2s;
}

.note-title-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.editor-area,
.preview-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.editor-toolbar {
  display: flex;
  gap: 4px;
  padding: 8px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px 8px 0 0;
  flex-wrap: wrap;
}

.format-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  color: var(--text-primary);
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.format-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.note-content-input {
  flex: 1;
  padding: 16px;
  border: 1px solid var(--border-color);
  border-top: none;
  border-radius: 0 0 8px 8px;
  font-size: 14px;
  font-family: 'Monaco', 'Menlo', monospace;
  line-height: 1.6;
  color: var(--text-primary);
  background: var(--bg-color);
  resize: none;
  outline: none;
}

.note-content-input:focus {
  border-color: var(--primary-color);
}

.preview-area {
  padding: 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-color);
  overflow-y: auto;
}

.preview-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-primary);
}

.preview-content :deep(h1) {
  font-size: 24px;
  margin: 16px 0 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}

.preview-content :deep(h2) {
  font-size: 20px;
  margin: 14px 0 6px;
}

.preview-content :deep(h3) {
  font-size: 18px;
  margin: 12px 0 4px;
}

.preview-content :deep(code) {
  background: var(--code-bg);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', monospace;
}

.preview-content :deep(pre) {
  background: var(--code-bg);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
}

.preview-content :deep(a) {
  color: var(--primary-color);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/apps/notes/NoteEditor.vue
git commit -m "feat(note): add NoteEditor component with edit/preview modes"
```

---

## 任务 9：前端 - 创建 NoteTagFilter 组件

**文件：**
- 创建：`qim-client/src/components/apps/notes/NoteTagFilter.vue`

- [ ] **步骤 1：创建组件**

```vue
<template>
  <div class="tag-filter" v-if="allTags.length > 0">
    <div class="tag-filter-header">
      <span class="filter-label">标签筛选</span>
      <button v-if="selectedTag" class="clear-btn" @click="$emit('clear')">
        清除
      </button>
    </div>
    <div class="tag-list">
      <span
        v-for="tag in allTags"
        :key="tag"
        :class="['tag-item', { active: selectedTag === tag }]"
        @click="$emit('select', tag)"
      >
        {{ tag }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  allTags: string[]
  selectedTag: string | null
}>()

defineEmits<{
  select: [tag: string]
  clear: []
}>()
</script>

<style scoped>
.tag-filter {
  padding: 12px;
  border-bottom: 1px solid var(--border-color);
}

.tag-filter-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.filter-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.clear-btn {
  font-size: 12px;
  color: var(--primary-color);
  background: none;
  border: none;
  cursor: pointer;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tag-item {
  font-size: 12px;
  padding: 4px 10px;
  background: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.tag-item:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.tag-item.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/apps/notes/NoteTagFilter.vue
git commit -m "feat(note): add NoteTagFilter component"
```

---

## 任务 10：前端 - 创建 AIAnalysisModal 组件

**文件：**
- 创建：`qim-client/src/components/apps/notes/AIAnalysisModal.vue`

- [ ] **步骤 1：创建组件**

```vue
<template>
  <ModalContainer
    :visible="visible"
    title="AI 分析结果"
    @close="$emit('close')"
  >
    <div class="analysis-result">
      <div class="result-section">
        <h4>摘要</h4>
        <p class="summary-text">{{ result?.summary || '暂无摘要' }}</p>
      </div>
      
      <div class="result-section">
        <h4>推荐标签</h4>
        <div class="tags-container">
          <span
            v-for="tag in result?.tags || []"
            :key="tag"
            :class="['tag-item', { selected: selectedTags.includes(tag) }]"
            @click="toggleTag(tag)"
          >
            {{ tag }}
          </span>
          <span v-if="!result?.tags?.length" class="no-tags">暂无推荐标签</span>
        </div>
      </div>
      
      <div class="result-section" v-if="result?.action_items?.length">
        <h4>提取的行动项</h4>
        <ul class="action-list">
          <li v-for="(item, index) in result.action_items" :key="index">
            {{ item }}
          </li>
        </ul>
      </div>
    </div>
    
    <template #footer>
      <button class="modal-btn cancel" @click="$emit('close')">取消</button>
      <button class="modal-btn confirm" @click="handleConfirm">
        保存摘要和标签
      </button>
    </template>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ModalContainer from '../../shared/ModalContainer.vue'
import type { AIAnalyzeResult } from '../../../types/note'

const props = defineProps<{
  visible: boolean
  result: AIAnalyzeResult | null
}>()

const emit = defineEmits<{
  close: []
  confirm: [summary: string, tags: string[]]
}>()

const selectedTags = ref<string[]>([])

watch(() => props.result, (newResult) => {
  if (newResult?.tags) {
    selectedTags.value = [...newResult.tags]
  }
})

function toggleTag(tag: string) {
  const index = selectedTags.value.indexOf(tag)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tag)
  }
}

function handleConfirm() {
  emit('confirm', props.result?.summary || '', selectedTags.value)
}
</script>

<style scoped>
.analysis-result {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.result-section h4 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.summary-text {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0;
  padding: 12px;
  background: var(--bg-color);
  border-radius: 6px;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  font-size: 13px;
  padding: 6px 12px;
  background: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.tag-item:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.tag-item.selected {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.no-tags {
  font-size: 13px;
  color: var(--text-tertiary);
}

.action-list {
  margin: 0;
  padding-left: 20px;
}

.action-list li {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.8;
}

.modal-btn {
  padding: 8px 20px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.modal-btn.cancel {
  background: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-btn.cancel:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.modal-btn.confirm {
  background: var(--primary-color);
  color: white;
  border: none;
}

.modal-btn.confirm:hover {
  background: var(--active-color);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/apps/notes/AIAnalysisModal.vue
git commit -m "feat(note): add AIAnalysisModal component"
```

---

## 任务 11：前端 - 重构 NotesApp.vue

**文件：**
- 修改：`qim-client/src/components/apps/NotesApp.vue`

- [ ] **步骤 1：重构主组件**

将现有的 NotesApp.vue 替换为新版本，整合所有子组件。由于文件较大，这里只展示关键结构：

```vue
<template>
  <div class="notes-app">
    <AppHeader title="笔记" @back="$emit('back')">
      <template #extra-buttons>
        <button class="icon-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
      </template>
      <template #actions>
        <button class="action-btn primary" @click="handleCreate">+ 新建笔记</button>
      </template>
    </AppHeader>
    
    <div class="notes-content">
      <div class="notes-sidebar">
        <div class="notes-search-box">
          <input v-model="searchQuery" type="text" placeholder="搜索笔记..." />
        </div>
        <NoteTagFilter
          :all-tags="allTags"
          :selected-tag="selectedTag"
          @select="selectedTag = $event"
          @clear="selectedTag = null"
        />
        <div class="notes-list">
          <NoteCard
            v-for="note in filteredNotes"
            :key="note.id"
            :note="note"
            :is-active="selectedNoteId === note.id"
            @select="selectNote(note.id)"
            @edit="editNote(note)"
            @delete="handleDelete(note.id)"
            @filter-tag="selectedTag = $event"
          />
        </div>
      </div>
      
      <div class="note-main">
        <template v-if="selectedNote">
          <NoteToolbar
            v-model:mode="editorMode"
            :saving="saving"
            :analyzing="analyzing"
            @save="handleSave"
            @analyze="handleAnalyze"
            @export="handleExport"
            @share="handleShare"
            @delete="handleDelete(selectedNote.id)"
          />
          <NoteEditor
            v-model:title="selectedNote.title"
            v-model:content="selectedNote.content"
            :mode="editorMode"
          />
        </template>
        <div v-else class="empty-state">
          <i class="fas fa-book"></i>
          <p>选择一个笔记或创建新笔记</p>
        </div>
      </div>
    </div>
    
    <AIAnalysisModal
      :visible="showAnalysisModal"
      :result="analysisResult"
      @close="showAnalysisModal = false"
      @confirm="handleAnalysisConfirm"
    />
  </div>
</template>
```

- [ ] **步骤 2：添加脚本逻辑**

```typescript
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import AppHeader from '../AppHeader.vue'
import NoteCard from './notes/NoteCard.vue'
import NoteToolbar from './notes/NoteToolbar.vue'
import NoteEditor from './notes/NoteEditor.vue'
import NoteTagFilter from './notes/NoteTagFilter.vue'
import AIAnalysisModal from './notes/AIAnalysisModal.vue'
import { useNotes } from '../../../composables/useNotes'
import QMessage from '../../../utils/qmessage'
import type { Note, AIAnalyzeResult } from '../../../types/note'

const emit = defineEmits(['back', 'toggleSidebar'])

const { 
  fetchNotes, 
  createNote, 
  updateNote, 
  deleteNote, 
  analyzeNote,
  updateNoteTags,
  updateNoteSummary,
  exportNote 
} = useNotes()

const notes = ref<Note[]>([])
const selectedNoteId = ref<number | null>(null)
const selectedNote = ref<Note | null>(null)
const searchQuery = ref('')
const selectedTag = ref<string | null>(null)
const editorMode = ref<'edit' | 'preview'>('edit')
const saving = ref(false)
const analyzing = ref(false)
const showAnalysisModal = ref(false)
const analysisResult = ref<AIAnalyzeResult | null>(null)

const allTags = computed(() => {
  const tags = new Set<string>()
  notes.value.forEach(n => n.tags?.forEach(t => tags.add(t)))
  return Array.from(tags)
})

const filteredNotes = computed(() => {
  let result = notes.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(n => 
      n.title.toLowerCase().includes(q) || 
      n.content.toLowerCase().includes(q)
    )
  }
  if (selectedTag.value) {
    result = result.filter(n => n.tags?.includes(selectedTag.value!))
  }
  return result
})

function selectNote(id: number) {
  selectedNoteId.value = id
  selectedNote.value = notes.value.find(n => n.id === id) || null
}

async function handleCreate() {
  const note = await createNote({ title: '新笔记', content: '' })
  if (note) {
    notes.value.unshift(note)
    selectNote(note.id)
  }
}

async function handleSave() {
  if (!selectedNote.value) return
  saving.value = true
  const ok = await updateNote(selectedNote.value.id, {
    title: selectedNote.value.title,
    content: selectedNote.value.content,
    tags: selectedNote.value.tags
  })
  saving.value = false
  if (ok) {
    QMessage.success('保存成功')
  }
}

async function handleAnalyze() {
  if (!selectedNote.value) return
  analyzing.value = true
  const result = await analyzeNote(selectedNote.value.id)
  analyzing.value = false
  if (result) {
    analysisResult.value = result
    showAnalysisModal.value = true
  }
}

async function handleAnalysisConfirm(summary: string, tags: string[]) {
  if (!selectedNote.value) return
  await updateNoteSummary(selectedNote.value.id, summary)
  await updateNoteTags(selectedNote.value.id, tags)
  selectedNote.value.summary = summary
  selectedNote.value.tags = tags
  showAnalysisModal.value = false
  QMessage.success('已保存摘要和标签')
}

function handleExport() {
  if (!selectedNote.value) return
  exportNote(selectedNote.value.id, selectedNote.value.title)
}

function handleShare() {
  if (!selectedNote.value) return
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type: 'note', data: selectedNote.value }
  }))
}

async function handleDelete(id: number) {
  if (!confirm('确定要删除这个笔记吗？')) return
  const ok = await deleteNote(id)
  if (ok) {
    notes.value = notes.value.filter(n => n.id !== id)
    if (selectedNoteId.value === id) {
      selectedNoteId.value = null
      selectedNote.value = null
    }
    QMessage.success('删除成功')
  }
}

onMounted(async () => {
  notes.value = await fetchNotes()
})
</script>
```

- [ ] **步骤 3：验证编译**

```bash
cd qim-client && npm run typecheck
```

预期：无类型错误

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/components/apps/NotesApp.vue
git commit -m "refactor(note): restructure NotesApp with sub-components"
```

---

## 任务 12：集成测试

- [ ] **步骤 1：启动后端服务**

```bash
cd qim-server && go run .
```

- [ ] **步骤 2：启动前端服务**

```bash
cd qim-client && npm run dev
```

- [ ] **步骤 3：手动测试功能**

测试清单：
- [ ] 创建新笔记
- [ ] 编辑笔记内容
- [ ] 切换编辑/预览模式
- [ ] 保存笔记
- [ ] AI 分析笔记
- [ ] 确认 AI 分析结果（摘要、标签）
- [ ] 标签筛选
- [ ] 导出 Markdown
- [ ] 删除笔记

- [ ] **步骤 4：最终 Commit**

```bash
git add -A
git commit -m "feat(note): complete AI enhancement for notes module"
```
