# 群组 AI 助手个性化增强实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将群组 AI 助手设置扩展为支持人设风格、触发规则细化、知识库绑定的多 Tab 配置面板

**架构：** 后端新增 Group 模型字段和文档绑定 API，前端重构 GroupAIPanel 为 Tab 结构（拆分为4个子组件），智能回复引擎将配置注入 prompt

**技术栈：** Go/Gin/GORM (后端), Vue 3/TypeScript (前端)

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-server/model/model.go` | 修改 | Group 结构体新增8个 AI 个性化字段 |
| `qim-server/model/model.go` | 修改 | 新增 GroupDocument 结构体 |
| `qim-server/handler/group_handler.go` | 修改 | UpdateGroupAISettings 支持新字段 |
| `qim-server/handler/group_document_handler.go` | 创建 | 群文档绑定 API（增/删/查） |
| `qim-server/app/routes.go` | 修改 | 注册群文档绑定路由 |
| `qim-server/handler/smart_reply_handler.go` | 修改 | 注入人设、语言、长度到 prompt，支持防刷屏和关键词过滤 |
| `qim-server/handler/prompt_builder.go` | 修改 | buildRules 动态生成规则，注入自定义 prompt |
| `qim-client/src/components/ai/GroupAIPanel.vue` | 修改 | 重构为 Tab 结构 |
| `qim-client/src/components/ai/ai-settings/AIBaseSettings.vue` | 创建 | 基础设置 Tab 组件 |
| `qim-client/src/components/ai/ai-settings/AIPersonaSettings.vue` | 创建 | 人设风格 Tab 组件 |
| `qim-client/src/components/ai/ai-settings/AITriggerSettings.vue` | 创建 | 触发规则 Tab 组件 |
| `qim-client/src/components/ai/ai-settings/AIKnowledgeSettings.vue` | 创建 | 知识库 Tab 组件 |
| `qim-client/src/types/ai.ts` | 修改 | 新增 GroupAISettings 类型 |
| `qim-client/src/types/index.ts` | 修改 | Conversation 新增 AI 字段 |
| `qim-client/src/components/chat/ChatHeader.vue` | 修改 | 传递新 AI props |
| `qim-client/src/components/chat/ChatWindow.vue` | 修改 | 处理新 AI 设置更新 |

---

### 任务 1：后端数据模型

**文件：**
- 修改：`qim-server/model/model.go`

- [ ] **步骤 1：Group 结构体新增字段**

```go
type Group struct {
	ID                uint         `json:"id" gorm:"primarykey"`
	ConversationID    uint         `json:"conversation_id" gorm:"uniqueIndex;not null"`
	GroupType         string       `json:"group_type" gorm:"size:20;not null"`
	Name              string       `json:"name" gorm:"size:200;not null"`
	Avatar            string       `json:"avatar" gorm:"size:500"`
	CreatorID         uint         `json:"creator_id" gorm:"not null"`
	Announcement      string       `json:"announcement" gorm:"type:text"`
	InvitePermission  string       `json:"invite_permission" gorm:"size:20;default:'owner_admin'"`
	AIEnabled         bool         `json:"ai_enabled" gorm:"default:false"`
	AIReplyMode       string       `json:"ai_reply_mode" gorm:"size:20;default:'mention_only'"`
	AIAssistantName   string       `json:"ai_assistant_name" gorm:"size:100;default:'AI助手'"`
	AIPersonality     string       `json:"ai_personality" gorm:"size:20;default:'professional'"`
	AICustomPrompt    string       `json:"ai_custom_prompt" gorm:"type:text"`
	AILanguage        string       `json:"ai_language" gorm:"size:10;default:'auto'"`
	AIMaxLength       string       `json:"ai_max_length" gorm:"size:10;default:'medium'"`
	AIMentionReplyMode string      `json:"ai_mention_reply_mode" gorm:"size:10;default:'mention'"`
	AIAntiSpamInterval int         `json:"ai_anti_spam_interval" gorm:"default:5"`
	AITriggerKeywords string       `json:"ai_trigger_keywords" gorm:"type:text"`
	AILearnEnabled    bool         `json:"ai_learn_enabled" gorm:"default:false"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	Conversation      Conversation `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
	Documents         []GroupDocument `json:"documents,omitempty" gorm:"foreignkey:GroupID"`
}
```

- [ ] **步骤 2：新增 GroupDocument 结构体**

```go
type GroupDocument struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	GroupID   uint      `json:"group_id" gorm:"not null;index"`
	FileID    uint      `json:"file_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	Group     Group     `json:"group,omitempty" gorm:"foreignkey:GroupID"`
	File      File      `json:"file,omitempty" gorm:"foreignkey:FileID"`
}
```

- [ ] **步骤 3：Commit**

```bash
git add qim-server/model/model.go
git commit -m "feat: Group模型新增AI个性化字段和GroupDocument关联表"
```

---

### 任务 2：后端 API 更新

**文件：**
- 修改：`qim-server/handler/group_handler.go`
- 创建：`qim-server/handler/group_document_handler.go`
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：更新 UpdateGroupAISettings handler**

在 `qim-server/handler/group_handler.go` 中，修改 `UpdateGroupAISettings` 函数的请求体和更新逻辑：

```go
var req struct {
	AIEnabled          *bool  `json:"ai_enabled"`
	AIReplyMode        string `json:"ai_reply_mode"`
	AIAssistantName    string `json:"ai_assistant_name"`
	AIPersonality      string `json:"ai_personality"`
	AICustomPrompt     string `json:"ai_custom_prompt"`
	AILanguage         string `json:"ai_language"`
	AIMaxLength        string `json:"ai_max_length"`
	AIMentionReplyMode string `json:"ai_mention_reply_mode"`
	AIAntiSpamInterval *int   `json:"ai_anti_spam_interval"`
	AITriggerKeywords  string `json:"ai_trigger_keywords"`
	AILearnEnabled     *bool  `json:"ai_learn_enabled"`
}

// ... 更新逻辑
if req.AIPersonality != "" {
	group.AIPersonality = req.AIPersonality
}
if req.AICustomPrompt != "" {
	group.AICustomPrompt = req.AICustomPrompt
}
if req.AILanguage != "" {
	group.AILanguage = req.AILanguage
}
if req.AIMaxLength != "" {
	group.AIMaxLength = req.AIMaxLength
}
if req.AIMentionReplyMode != "" {
	group.AIMentionReplyMode = req.AIMentionReplyMode
}
if req.AIAntiSpamInterval != nil {
	group.AIAntiSpamInterval = *req.AIAntiSpamInterval
}
if req.AITriggerKeywords != "" {
	group.AITriggerKeywords = req.AITriggerKeywords
}
if req.AILearnEnabled != nil {
	group.AILearnEnabled = *req.AILearnEnabled
}
db.Save(&group)
```

- [ ] **步骤 2：创建 group_document_handler.go**

```go
package handler

import (
	"net/http"
	"qim-server/database"
	"qim-server/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetGroupDocuments(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊不存在"})
		return
	}

	var documents []model.GroupDocument
	db.Preload("File").Where("group_id = ?", group.ID).Find(&documents)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": documents,
	})
}

func AddGroupDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	convID, _ := strconv.ParseUint(convIDStr, 10, 32)

	var req struct {
		FileID uint `json:"file_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊不存在"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是成员"})
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以管理知识库"})
		return
	}

	doc := model.GroupDocument{GroupID: group.ID, FileID: req.FileID}
	db.Create(&doc)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "文档绑定成功", "data": doc})
}

func RemoveGroupDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	fileIDStr := c.Param("file_id")
	convID, _ := strconv.ParseUint(convIDStr, 10, 32)
	fileID, _ := strconv.ParseUint(fileIDStr, 10, 32)

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊不存在"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是成员"})
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以管理知识库"})
		return
	}

	db.Where("group_id = ? AND file_id = ?", group.ID, uint(fileID)).Delete(&model.GroupDocument{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "文档解绑成功"})
}
```

- [ ] **步骤 3：注册路由**

在 `qim-server/app/routes.go` 的群聊管理区域新增：

```go
// 群知识库管理
authed.GET("/conversations/:id/ai-documents", handler.GetGroupDocuments)
authed.POST("/conversations/:id/ai-documents", handler.AddGroupDocument)
authed.DELETE("/conversations/:id/ai-documents/:file_id", handler.RemoveGroupDocument)
```

- [ ] **步骤 4：Commit**

```bash
git add qim-server/handler/group_handler.go qim-server/handler/group_document_handler.go qim-server/app/routes.go
git commit -m "feat: 新增群AI设置API和知识库文档绑定API"
```

---

### 任务 3：前端类型定义

**文件：**
- 修改：`qim-client/src/types/ai.ts`
- 修改：`qim-client/src/types/index.ts`

- [ ] **步骤 1：新增 GroupAISettings 类型**

在 `qim-client/src/types/ai.ts` 中新增：

```typescript
export interface GroupAISettings {
  aiEnabled: boolean
  aiAssistantName: string
  aiReplyMode: string
  aiPersonality: string
  aiCustomPrompt: string
  aiLanguage: string
  aiMaxLength: string
  aiMentionReplyMode: string
  aiAntiSpamInterval: number
  aiTriggerKeywords: string[]
  aiLearnEnabled: boolean
}

export interface GroupDocument {
  id: number
  group_id: number
  file_id: number
  created_at: string
  file?: {
    id: number
    name: string
    size: number
    type: string
  }
}
```

- [ ] **步骤 2：更新 Conversation 类型**

在 `qim-client/src/types/index.ts` 的 Conversation 接口中新增：

```typescript
ai_personality?: string
ai_custom_prompt?: string
ai_language?: string
ai_max_length?: string
ai_mention_reply_mode?: string
ai_anti_spam_interval?: number
ai_trigger_keywords?: string
ai_learn_enabled?: boolean
```

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/types/ai.ts qim-client/src/types/index.ts
git commit -m "feat: 新增GroupAISettings和GroupDocument类型"
```

---

### 任务 4：前端子组件 — 基础设置

**文件：**
- 创建：`qim-client/src/components/ai/ai-settings/AIBaseSettings.vue`

- [ ] **步骤 1：创建基础设置组件**

```vue
<template>
  <div class="ai-base-settings">
    <div class="setting-item">
      <label class="toggle-label">
        <span>启用 AI 助手</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.aiEnabled" @change="update('aiEnabled', $event)" />
          <span class="slider round"></span>
        </label>
      </label>
    </div>

    <div v-if="modelValue.aiEnabled" class="advanced-settings">
      <div class="setting-item">
        <label>AI 助手名称</label>
        <input
          type="text"
          :value="modelValue.aiAssistantName"
          @input="update('aiAssistantName', ($event.target as HTMLInputElement).value)"
          class="form-input"
          placeholder="请输入 AI 助手名称"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GroupAISettings } from '../../../types/ai'

interface Props {
  modelValue: GroupAISettings
}

interface Emits {
  (e: 'update:modelValue', value: GroupAISettings): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function update<K extends keyof GroupAISettings>(key: K, value: GroupAISettings[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}
</script>

<style scoped>
.ai-base-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-input:focus { outline: none; border-color: var(--primary-color); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
.advanced-settings { margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--border-color); }
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/ai/ai-settings/AIBaseSettings.vue
git commit -m "feat: 创建AI基础设置子组件"
```

---

### 任务 5：前端子组件 — 人设风格

**文件：**
- 创建：`qim-client/src/components/ai/ai-settings/AIPersonaSettings.vue`

- [ ] **步骤 1：创建人设风格组件**

```vue
<template>
  <div class="ai-persona-settings">
    <div class="setting-section">
      <label class="section-label">预设人设</label>
      <div class="persona-grid">
        <div
          v-for="p in personas"
          :key="p.value"
          :class="['persona-card', { active: modelValue.aiPersonality === p.value }]"
          @click="update('aiPersonality', p.value)"
        >
          <div class="persona-icon">{{ p.icon }}</div>
          <div class="persona-name">{{ p.name }}</div>
          <div class="persona-desc">{{ p.desc }}</div>
        </div>
      </div>
    </div>

    <div class="setting-section">
      <label class="section-label">自定义系统提示词（可选）</label>
      <textarea
        :value="modelValue.aiCustomPrompt"
        @input="update('aiCustomPrompt', ($event.target as HTMLTextAreaElement).value)"
        class="form-textarea"
        placeholder="输入自定义提示词，将覆盖预设人设。留空则使用预设人设。"
        rows="5"
      ></textarea>
      <span class="setting-hint">自定义提示词优先级高于预设人设</span>
    </div>

    <div class="setting-row">
      <div class="setting-item">
        <label>回复语言</label>
        <select :value="modelValue.aiLanguage" @change="update('aiLanguage', ($event.target as HTMLSelectElement).value)" class="form-select">
          <option value="auto">自动（跟随提问语言）</option>
          <option value="zh">中文</option>
          <option value="en">English</option>
          <option value="ja">日本語</option>
        </select>
      </div>

      <div class="setting-item">
        <label>回复长度</label>
        <select :value="modelValue.aiMaxLength" @change="update('aiMaxLength', ($event.target as HTMLSelectElement).value)" class="form-select">
          <option value="short">简短（1-2句）</option>
          <option value="medium">适中（3-5句）</option>
          <option value="long">详细（不限）</option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GroupAISettings } from '../../../types/ai'

interface Props { modelValue: GroupAISettings }
interface Emits { (e: 'update:modelValue', value: GroupAISettings): void }

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function update<K extends keyof GroupAISettings>(key: K, value: GroupAISettings[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

const personas = [
  { value: 'professional', icon: '🎓', name: '专业严谨', desc: '回答专业、严谨、客观' },
  { value: 'casual', icon: '😊', name: '轻松幽默', desc: '语气活泼、善用表情' },
  { value: 'concise', icon: '⚡', name: '简洁高效', desc: '直奔主题、不废话' },
  { value: 'friendly', icon: '🤗', name: '贴心助手', desc: '温暖亲切、有耐心' },
  { value: 'technical', icon: '💻', name: '技术专家', desc: '偏重技术深度和细节' }
]
</script>

<style scoped>
.ai-persona-settings { padding: 16px; }
.setting-section { margin-bottom: 20px; }
.section-label { display: block; margin-bottom: 10px; font-size: 14px; font-weight: 500; }
.persona-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 10px; }
.persona-card { padding: 14px; border: 2px solid var(--border-color); border-radius: 10px; cursor: pointer; text-align: center; transition: all 0.2s; }
.persona-card:hover { border-color: var(--primary-color); }
.persona-card.active { border-color: var(--primary-color); background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1)); }
.persona-icon { font-size: 28px; margin-bottom: 6px; }
.persona-name { font-size: 14px; font-weight: 600; margin-bottom: 4px; }
.persona-desc { font-size: 12px; color: var(--text-secondary); }
.form-textarea { width: 100%; padding: 10px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; resize: vertical; box-sizing: border-box; font-family: inherit; }
.form-textarea:focus { outline: none; border-color: var(--primary-color); }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.setting-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.setting-item label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.form-select { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus { outline: none; border-color: var(--primary-color); }
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/ai/ai-settings/AIPersonaSettings.vue
git commit -m "feat: 创建AI人设风格子组件"
```

---

### 任务 6：前端子组件 — 触发规则

**文件：**
- 创建：`qim-client/src/components/ai/ai-settings/AITriggerSettings.vue`

- [ ] **步骤 1：创建触发规则组件**

```vue
<template>
  <div class="ai-trigger-settings">
    <div class="setting-item">
      <label>回复模式</label>
      <select :value="modelValue.aiReplyMode" @change="update('aiReplyMode', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="mention_only">仅被 @ 时回复</option>
        <option value="smart">智能判断回复</option>
        <option value="always">始终回复</option>
        <option value="off">关闭 AI 回复</option>
      </select>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>@ 后回复方式</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.aiMentionReplyMode === 'mention'" @change="update('aiMentionReplyMode', $event.target.checked ? 'mention' : 'direct')" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">{{ modelValue.aiMentionReplyMode === 'mention' ? '@提问者后回复' : '直接回复' }}</span>
    </div>

    <div class="setting-item">
      <label>防刷屏间隔</label>
      <select :value="modelValue.aiAntiSpamInterval" @change="update('aiAntiSpamInterval', Number(($event.target as HTMLSelectElement).value))" class="form-select">
        <option :value="0">关闭</option>
        <option :value="3">3 分钟</option>
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="15">15 分钟</option>
      </select>
      <span class="setting-hint">同一话题在此间隔内只回复一次</span>
    </div>

    <div class="setting-item">
      <label>触发关键词（可选）</label>
      <div class="keyword-input-wrapper">
        <input
          :value="keywordInput"
          @input="keywordInput = ($event.target as HTMLInputElement).value"
          @keydown.enter.prevent="addKeyword"
          class="form-input"
          placeholder="输入关键词后按回车"
        />
        <div class="keyword-tags">
          <span v-for="(kw, i) in modelValue.aiTriggerKeywords" :key="i" class="keyword-tag">
            {{ kw }}
            <button class="remove-tag" @click="removeKeyword(i)">×</button>
          </span>
        </div>
      </div>
      <span class="setting-hint">设置后仅当消息包含关键词时 AI 才触发（留空则不限）</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { GroupAISettings } from '../../../types/ai'

interface Props { modelValue: GroupAISettings }
interface Emits { (e: 'update:modelValue', value: GroupAISettings): void }

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const keywordInput = ref('')

function update<K extends keyof GroupAISettings>(key: K, value: GroupAISettings[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function addKeyword() {
  const kw = keywordInput.value.trim()
  if (kw && !props.modelValue.aiTriggerKeywords.includes(kw)) {
    update('aiTriggerKeywords', [...props.modelValue.aiTriggerKeywords, kw])
  }
  keywordInput.value = ''
}

function removeKeyword(index: number) {
  const keywords = [...props.modelValue.aiTriggerKeywords]
  keywords.splice(index, 1)
  update('aiTriggerKeywords', keywords)
}
</script>

<style scoped>
.ai-trigger-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
.keyword-input-wrapper { display: flex; flex-direction: column; gap: 8px; }
.keyword-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.keyword-tag { display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1)); color: var(--primary-color); border-radius: 12px; font-size: 13px; }
.remove-tag { background: none; border: none; color: var(--primary-color); cursor: pointer; font-size: 14px; padding: 0; width: 16px; height: 16px; display: flex; align-items: center; justify-content: center; }
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/ai/ai-settings/AITriggerSettings.vue
git commit -m "feat: 创建AI触发规则子组件"
```

---

### 任务 7：前端子组件 — 知识库

**文件：**
- 创建：`qim-client/src/components/ai/ai-settings/AIKnowledgeSettings.vue`

- [ ] **步骤 1：创建知识库组件**

```vue
<template>
  <div class="ai-knowledge-settings">
    <div class="setting-section">
      <div class="section-header">
        <label class="section-label">绑定文档</label>
        <button class="add-btn" @click="showFilePicker = true">
          <i class="fas fa-plus"></i> 添加文档
        </button>
      </div>

      <div v-if="documents.length === 0" class="empty-state">
        <i class="fas fa-folder-open"></i>
        <p>暂未绑定任何文档</p>
      </div>

      <div class="document-list">
        <div v-for="doc in documents" :key="doc.id" class="document-item">
          <div class="doc-info">
            <i :class="getFileIcon(doc.file?.type || '')" class="doc-icon"></i>
            <div class="doc-name">{{ doc.file?.name || '未知文件' }}</div>
            <div class="doc-size">{{ formatSize(doc.file?.size || 0) }}</div>
          </div>
          <button class="remove-btn" @click="removeDocument(doc)" title="移除">
            <i class="fas fa-trash-alt"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 文件选择弹窗 -->
    <div v-if="showFilePicker" class="modal-overlay" @click="showFilePicker = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>选择文档</h3>
          <button class="close-btn" @click="showFilePicker = false">&times;</button>
        </div>
        <div class="modal-body">
          <div class="file-picker-list">
            <div v-for="file in availableFiles" :key="file.id" class="file-option" @click="toggleFile(file)">
              <input type="checkbox" :checked="isFileSelected(file.id)" />
              <span>{{ file.name }}</span>
            </div>
            <div v-if="availableFiles.length === 0" class="empty-picker">暂无可用文件</div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="showFilePicker = false">取消</button>
          <button class="btn btn-primary" @click="confirmAddDocuments">确认</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { GroupDocument } from '../../../types/ai'

interface Props {
  groupId: number
  documents: GroupDocument[]
}

interface Emits {
  (e: 'add', fileIds: number[]): void
  (e: 'remove', fileId: number): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const showFilePicker = ref(false)
const availableFiles = ref<any[]>([])
const selectedFileIds = ref<number[]>([])

async function loadAvailableFiles() {
  try {
    const response = await fetch('/api/v1/files?page_size=100', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    const data = await response.json()
    availableFiles.value = data.data?.files || []
  } catch (e) {
    console.error('加载文件列表失败', e)
  }
}

watch(showFilePicker, async (val) => {
  if (val) {
    await loadAvailableFiles()
    selectedFileIds.value = []
  }
})

function isFileSelected(fileId: number) {
  return selectedFileIds.value.includes(fileId)
}

function toggleFile(file: any) {
  const idx = selectedFileIds.value.indexOf(file.id)
  if (idx >= 0) {
    selectedFileIds.value.splice(idx, 1)
  } else {
    selectedFileIds.value.push(file.id)
  }
}

function confirmAddDocuments() {
  if (selectedFileIds.value.length > 0) {
    emit('add', [...selectedFileIds.value])
  }
  showFilePicker.value = false
}

function removeDocument(doc: GroupDocument) {
  emit('remove', doc.file_id)
}

function getFileIcon(type: string) {
  if (type.includes('pdf')) return 'fas fa-file-pdf'
  if (type.includes('word') || type.includes('document')) return 'fas fa-file-word'
  if (type.includes('excel') || type.includes('sheet')) return 'fas fa-file-excel'
  if (type.includes('text')) return 'fas fa-file-alt'
  return 'fas fa-file'
}

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped>
.ai-knowledge-settings { padding: 16px; }
.setting-section { margin-bottom: 20px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.section-label { font-size: 14px; font-weight: 500; }
.add-btn { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 13px; cursor: pointer; }
.add-btn:hover { border-color: var(--primary-color); color: var(--primary-color); }
.empty-state { text-align: center; padding: 32px; color: var(--text-secondary); }
.empty-state i { font-size: 40px; margin-bottom: 8px; display: block; }
.document-list { display: flex; flex-direction: column; gap: 8px; }
.document-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: var(--bg-color); border: 1px solid var(--border-color); border-radius: 8px; }
.doc-info { display: flex; align-items: center; gap: 10px; }
.doc-icon { font-size: 20px; color: var(--text-secondary); }
.doc-name { font-size: 14px; }
.doc-size { font-size: 12px; color: var(--text-secondary); }
.remove-btn { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 6px; font-size: 14px; border-radius: 4px; }
.remove-btn:hover { color: #ef4444; background: rgba(239, 68, 68, 0.1); }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0, 0, 0, 0.5); display: flex; align-items: center; justify-content: center; z-index: 2000; }
.modal-content { background: var(--sidebar-bg); border-radius: 12px; width: 90%; max-width: 500px; max-height: 80vh; overflow: hidden; display: flex; flex-direction: column; }
.modal-header { display: flex; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border-color); }
.modal-header h3 { margin: 0; font-size: 16px; }
.modal-header .close-btn { background: none; border: none; font-size: 20px; cursor: pointer; color: var(--text-secondary); }
.modal-body { padding: 16px 20px; overflow-y: auto; flex: 1; }
.file-picker-list { display: flex; flex-direction: column; gap: 4px; }
.file-option { display: flex; align-items: center; gap: 8px; padding: 8px; border-radius: 6px; cursor: pointer; }
.file-option:hover { background: var(--hover-color); }
.empty-picker { text-align: center; padding: 20px; color: var(--text-secondary); }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 12px 20px; border-top: 1px solid var(--border-color); }
.btn { padding: 8px 16px; border-radius: 6px; font-size: 14px; cursor: pointer; border: none; }
.btn-primary { background: var(--primary-color); color: white; }
.btn-secondary { background: var(--bg-color); color: var(--text-color); border: 1px solid var(--border-color); }
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/ai/ai-settings/AIKnowledgeSettings.vue
git commit -m "feat: 创建AI知识库子组件"
```

---

### 任务 8：重构 GroupAIPanel 为 Tab 结构

**文件：**
- 修改：`qim-client/src/components/ai/GroupAIPanel.vue`

- [ ] **步骤 1：重写 GroupAIPanel.vue**

```vue
<template>
  <div class="group-ai-panel">
    <h4>AI 助手设置</h4>

    <div class="tab-bar">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['tab-btn', { active: activeTab === tab.key }]"
        @click="activeTab = tab.key"
      >
        <i :class="tab.icon"></i>
        <span>{{ tab.label }}</span>
      </button>
    </div>

    <div class="tab-content">
      <AIBaseSettings
        v-if="activeTab === 'base'"
        v-model="settings"
      />
      <AIPersonaSettings
        v-if="activeTab === 'persona'"
        v-model="settings"
      />
      <AITriggerSettings
        v-if="activeTab === 'trigger'"
        v-model="settings"
      />
      <AIKnowledgeSettings
        v-if="activeTab === 'knowledge'"
        :group-id="groupId"
        :documents="documents"
        @add="handleAddDocuments"
        @remove="handleRemoveDocument"
      />
    </div>

    <div class="tab-footer">
      <button class="btn btn-primary" @click="saveSettings" :disabled="saving">
        {{ saving ? '保存中...' : '保存设置' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import AIBaseSettings from './ai-settings/AIBaseSettings.vue'
import AIPersonaSettings from './ai-settings/AIPersonaSettings.vue'
import AITriggerSettings from './ai-settings/AITriggerSettings.vue'
import AIKnowledgeSettings from './ai-settings/AIKnowledgeSettings.vue'
import type { GroupAISettings, GroupDocument } from '../../types/ai'

interface Props {
  groupId: number
  aiEnabled?: boolean
  aiAssistantName?: string
  aiReplyMode?: string
  aiPersonality?: string
  aiCustomPrompt?: string
  aiLanguage?: string
  aiMaxLength?: string
  aiMentionReplyMode?: string
  aiAntiSpamInterval?: number
  aiTriggerKeywords?: string[]
  aiLearnEnabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  aiEnabled: false,
  aiAssistantName: 'AI助手',
  aiReplyMode: 'mention_only',
  aiPersonality: 'professional',
  aiCustomPrompt: '',
  aiLanguage: 'auto',
  aiMaxLength: 'medium',
  aiMentionReplyMode: 'mention',
  aiAntiSpamInterval: 5,
  aiTriggerKeywords: () => [],
  aiLearnEnabled: false
})

const emit = defineEmits<{
  (e: 'update', settings: GroupAISettings): void
}>()

const activeTab = ref('base')

const tabs = [
  { key: 'base', label: '基础设置', icon: 'fas fa-cog' },
  { key: 'persona', label: '人设风格', icon: 'fas fa-palette' },
  { key: 'trigger', label: '触发规则', icon: 'fas fa-bolt' },
  { key: 'knowledge', label: '知识库', icon: 'fas fa-book' }
]

const settings = ref<GroupAISettings>({
  aiEnabled: props.aiEnabled,
  aiAssistantName: props.aiAssistantName,
  aiReplyMode: props.aiReplyMode,
  aiPersonality: props.aiPersonality,
  aiCustomPrompt: props.aiCustomPrompt,
  aiLanguage: props.aiLanguage,
  aiMaxLength: props.aiMaxLength,
  aiMentionReplyMode: props.aiMentionReplyMode,
  aiAntiSpamInterval: props.aiAntiSpamInterval,
  aiTriggerKeywords: [...props.aiTriggerKeywords],
  aiLearnEnabled: props.aiLearnEnabled
})

watch(() => [props.aiEnabled, props.aiAssistantName, props.aiReplyMode, props.aiPersonality, props.aiLanguage], () => {
  settings.value = {
    aiEnabled: props.aiEnabled,
    aiAssistantName: props.aiAssistantName,
    aiReplyMode: props.aiReplyMode,
    aiPersonality: props.aiPersonality,
    aiCustomPrompt: props.aiCustomPrompt,
    aiLanguage: props.aiLanguage,
    aiMaxLength: props.aiMaxLength,
    aiMentionReplyMode: props.aiMentionReplyMode,
    aiAntiSpamInterval: props.aiAntiSpamInterval,
    aiTriggerKeywords: [...props.aiTriggerKeywords],
    aiLearnEnabled: props.aiLearnEnabled
  }
})

const saving = ref(false)

async function saveSettings() {
  saving.value = true
  try {
    emit('update', { ...settings.value })
  } finally {
    saving.value = false
  }
}

const documents = ref<GroupDocument[]>([])

async function loadDocuments() {
  try {
    const response = await fetch(`/api/v1/conversations/${props.groupId}/ai-documents`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    const data = await response.json()
    if (data.code === 0) {
      documents.value = data.data || []
    }
  } catch (e) {
    console.error('加载知识库失败', e)
  }
}

async function handleAddDocuments(fileIds: number[]) {
  for (const fileId of fileIds) {
    try {
      await fetch(`/api/v1/conversations/${props.groupId}/ai-documents`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({ file_id: fileId })
      })
    } catch (e) {
      console.error('添加文档失败', e)
    }
  }
  await loadDocuments()
}

async function handleRemoveDocument(fileId: number) {
  try {
    await fetch(`/api/v1/conversations/${props.groupId}/ai-documents/${fileId}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    await loadDocuments()
  } catch (e) {
    console.error('移除文档失败', e)
  }
}

onMounted(() => {
  loadDocuments()
})
</script>

<style scoped>
.group-ai-panel { padding: 0; background: var(--card-bg); border-radius: 12px; overflow: hidden; }
.group-ai-panel h4 { margin: 0; padding: 16px 20px; font-size: 16px; font-weight: 600; border-bottom: 1px solid var(--border-color); }
.tab-bar { display: flex; border-bottom: 1px solid var(--border-color); }
.tab-btn { flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px; padding: 12px 8px; border: none; background: none; cursor: pointer; font-size: 13px; color: var(--text-secondary); border-bottom: 2px solid transparent; transition: all 0.2s; }
.tab-btn:hover { color: var(--text-color); background: var(--hover-color); }
.tab-btn.active { color: var(--primary-color); border-bottom-color: var(--primary-color); background: var(--primary-color-alpha, rgba(99, 102, 241, 0.05)); }
.tab-content { min-height: 200px; }
.tab-footer { padding: 12px 20px; border-top: 1px solid var(--border-color); display: flex; justify-content: flex-end; }
.btn { padding: 8px 20px; border-radius: 6px; font-size: 14px; cursor: pointer; border: none; font-weight: 500; }
.btn-primary { background: var(--primary-color); color: white; }
.btn-primary:hover { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/ai/GroupAIPanel.vue
git commit -m "feat: 重构GroupAIPanel为Tab结构"
```

---

### 任务 9：前端上层组件适配

**文件：**
- 修改：`qim-client/src/components/chat/ChatHeader.vue`
- 修改：`qim-client/src/components/chat/GroupPanel.vue`
- 修改：`qim-client/src/components/chat/ChatWindow.vue`

- [ ] **步骤 1：更新 ChatHeader props 和 emits**

在 ChatHeader.vue 中传递新增的 AI 属性（从 conversation 中读取），并在 emit 中添加新的字段传递。

- [ ] **步骤 2：更新 GroupPanel props**

GroupPanel 需要接收新的 AI 设置 props 并转发给 GroupAIPanel：

```typescript
// Props 新增
aiPersonality?: string
aiCustomPrompt?: string
aiLanguage?: string
aiMaxLength?: string
aiMentionReplyMode?: string
aiAntiSpamInterval?: number
aiTriggerKeywords?: string[]
aiLearnEnabled?: boolean

// Emits 更新
'update-ai-settings': [settings: {
  aiEnabled: boolean;
  aiAssistantName: string;
  aiReplyMode: string;
  aiPersonality: string;
  aiCustomPrompt: string;
  aiLanguage: string;
  aiMaxLength: string;
  aiMentionReplyMode: string;
  aiAntiSpamInterval: number;
  aiTriggerKeywords: string[];
  aiLearnEnabled: boolean;
}]
```

- [ ] **步骤 3：更新 ChatWindow 的 handleUpdateAISettings**

```typescript
const handleUpdateAISettings = async (settings: any) => {
  if (!props.conversation?.id) return

  try {
    const response = await request(`/api/v1/conversations/${props.conversation.id}/ai-settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ai_enabled: settings.aiEnabled,
        ai_assistant_name: settings.aiAssistantName,
        ai_reply_mode: settings.aiReplyMode,
        ai_personality: settings.aiPersonality,
        ai_custom_prompt: settings.aiCustomPrompt,
        ai_language: settings.aiLanguage,
        ai_max_length: settings.aiMaxLength,
        ai_mention_reply_mode: settings.aiMentionReplyMode,
        ai_anti_spam_interval: settings.aiAntiSpamInterval,
        ai_trigger_keywords: settings.aiTriggerKeywords.join(','),
        ai_learn_enabled: settings.aiLearnEnabled
      })
    })

    const data = await response.json()
    if (data.code === 0) {
      QMessage.success('AI 设置已更新')
    } else {
      QMessage.error(data.message || 'AI 设置更新失败')
    }
  } catch (error: any) {
    QMessage.error('AI 设置更新失败')
  }
}
```

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/components/chat/ChatHeader.vue qim-client/src/components/chat/GroupPanel.vue qim-client/src/components/chat/ChatWindow.vue
git commit -m "feat: 更新上层组件传递新AI设置"
```

---

### 任务 10：Prompt 构建器增强

**文件：**
- 修改：`qim-server/handler/prompt_builder.go`

- [ ] **步骤 1：修改 buildRules 方法**

将硬编码的规则改为根据 Group 配置动态生成：

```go
func (b *SmartPromptBuilder) buildRules(ctx *PromptContext) string {
	var rules []string

	// 语言规则
	if ctx.Group != nil {
		switch ctx.Group.AILanguage {
		case "zh":
			rules = append(rules, "使用中文回复")
		case "en":
			rules = append(rules, "Reply in English")
		case "ja":
			rules = append(rules, "日本語で回答してください")
		default:
			rules = append(rules, "使用与提问者相同的语言回复")
		}
	} else {
		rules = append(rules, "使用中文回复")
	}

	// 长度规则
	if ctx.Group != nil {
		switch ctx.Group.AIMaxLength {
		case "short":
			rules = append(rules, "回答要简短，控制在1-2句话以内")
		case "long":
			// 不限
		default:
			rules = append(rules, "回答长度控制在3-5句话")
		}
	} else {
		rules = append(rules, "回答要简洁、专业、准确")
	}

	rules = append(rules, "优先使用知识库中的内容回答")
	rules = append(rules, "如果知识库中没有相关内容，使用你的通用知识回答，但明确说明\"以下回答基于通用知识，建议核实\"")

	if ctx.Group != nil && ctx.Group.AICustomPrompt != "" {
		rules = append(rules, "额外要求: "+ctx.Group.AICustomPrompt)
	}

	return "\n\n回复规则：\n- " + strings.Join(rules, "\n- ")
}
```

- [ ] **步骤 2：修改 BuildSystemPrompt 注入人设**

```go
func (b *SmartPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	// 优先使用自定义 prompt
	if ctx.Group != nil && ctx.Group.AICustomPrompt != "" {
		return ctx.Group.AICustomPrompt + b.buildTimeInfo() + b.buildGroupInfo(ctx) + b.buildRules(ctx)
	}

	// 使用预设人设
	personalityPrompt := b.buildPersonalityPrompt(ctx)

	prompt := personalityPrompt
	prompt += b.buildTimeInfo()
	prompt += b.buildGroupInfo(ctx)
	prompt += b.buildMessageHistory(ctx)
	prompt += b.buildUserInfo(ctx)
	prompt += b.buildTaskInfo(ctx)
	prompt += b.buildKnowledgeContext(ctx)
	prompt += b.buildGroupStats(ctx)
	prompt += b.buildRules(ctx)

	return prompt
}

func (b *SmartPromptBuilder) buildPersonalityPrompt(ctx *PromptContext) string {
	if ctx.Group == nil {
		return "你是 QIM 企业即时通讯系统中的智能助手。"
	}

	switch ctx.Group.AIPersonality {
	case "casual":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，性格轻松幽默。在回答中可以适当使用表情和emoji，语气活泼。"
	case "concise":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，风格简洁高效。回答直奔主题，不废话，只说重点。"
	case "friendly":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，性格温暖亲切。回答要有耐心，语气友善，像一个贴心的伙伴。"
	case "technical":
		return "你是 QIM 企业即时通讯系统中的技术专家 AI 助手。回答要有技术深度，关注细节，必要时提供代码示例和技术方案。"
	default: // professional
		return "你是 QIM 企业即时通讯系统中的智能助手，风格专业严谨。回答要专业、客观、有条理。"
	}
}
```

- [ ] **步骤 3：Commit**

```bash
git add qim-server/handler/prompt_builder.go
git commit -m "feat: Prompt构建器支持人设、语言、长度配置"
```

---

### 任务 11：SmartReplyEngine 增强

**文件：**
- 修改：`qim-server/handler/smart_reply_handler.go`

- [ ] **步骤 1：添加防刷屏和触发关键词过滤**

在 `HandleMessage` 函数中，根据群配置决定是否触发：

```go
// 触发关键词过滤
if group.AITriggerKeywords != "" {
	keywords := strings.Split(group.AITriggerKeywords, ",")
	hasKeyword := false
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		log.Printf("[SmartReply] 消息不包含触发关键词，跳过")
		return
	}
}

// 防刷屏检查
if group.AIAntiSpamInterval > 0 {
	var lastAIMsg model.Message
	err := db.Where("conversation_id = ? AND sender_id = 0 AND created_at > ?",
		conversationID, time.Now().Add(-time.Duration(group.AIAntiSpamInterval)*time.Minute)).
		Order("created_at DESC").First(&lastAIMsg).Error
	if err == nil {
		log.Printf("[SmartReply] 防刷屏间隔内，跳过回复")
		return
	}
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/handler/smart_reply_handler.go
git commit -m "feat: SmartReplyEngine支持防刷屏间隔和触发关键词过滤"
```

---

### 任务 12：数据库迁移

**文件：**
- 修改：`qim-server/app/database.go` 或迁移脚本

- [ ] **步骤 1：确保 GORM AutoMigrate 包含新字段**

在数据库初始化代码中确认 Group 和 GroupDocument 模型被 AutoMigrate：

```go
db.AutoMigrate(&model.Group{}, &model.GroupDocument{})
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/app/database.go
git commit -m "chore: 数据库迁移新增AI个性化字段"
```

---

### 任务 13：联调验证

- [ ] **步骤 1：前端设置保存 → 后端存储验证**

1. 启动后端和前端
2. 进入群聊 → 更多菜单 → AI 助手设置
3. 在各个 Tab 中修改设置
4. 点击"保存设置"
5. 检查数据库 Group 表字段是否更新
6. 刷新页面，确认设置是否被正确读取

- [ ] **步骤 2：AI 回复使用新配置验证**

1. 设置人设为"轻松幽默"
2. 在群里 @AI 提问
3. 验证 AI 回复风格是否符合设置
4. 设置触发关键词为"项目"
5. 发送不包含关键词的消息，验证 AI 不回复
6. 发送包含关键词的消息，验证 AI 回复
