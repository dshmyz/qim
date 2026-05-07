# AI分身与AI助手融合实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现 AI分身与AI助手的渐进融合，将分身定位为"个人 AI 能力层"

**架构：** 在 AI助手中心增加"我的分身"Tab，复用 AI能力层，保持身份继承和向后兼容

**技术栈：** Vue 3 + TypeScript + useRequest composable + 项目共享组件

---

## 文件结构

### 前端文件

| 文件 | 职责 | 操作 |
|------|------|------|
| `src/components/apps/ai/MyAvatar.vue` | 我的分身面板组件 | 新建 |
| `src/components/apps/ai/ChatCenter.vue` | 修改 Tab 配置，增加分身入口 | 修改 |
| `src/api/avatar.ts` | 新增工具绑定相关 API | 修改 |
| `src/types/avatar.ts` | 新增 AvatarWithTools 类型 | 修改 |
| `src/composables/useAvatar.ts` | 新增工具能力相关方法 | 修改 |

### 后端文件

| 文件 | 职责 | 操作 |
|------|------|------|
| `handler/avatar_handler.go` | 新增工具绑定接口 | 修改 |
| `model/avatar.go` | 新增 AvatarToolBinding 模型 | 修改 |
| `app/routes.go` | 注册工具绑定路由 | 修改 |

---

## Phase 1：UI 统一入口（第1周）

### 任务 1：修改 Tab 配置，增加分身入口

**文件：**
- 修改：`qim-client/src/components/apps/ai/ChatCenter.vue`

- [ ] **步骤 1：查看当前 Tab 配置**

运行：`cat qim-client/src/components/apps/ai/ChatCenter.vue | head -100`

- [ ] **步骤 2：修改 tabs 配置**

```typescript
const tabs = [
  { id: 'chat', label: '对话中心', icon: 'message-circle' },
  { id: 'bots', label: '我的机器人', icon: 'bot' },
  { id: 'create', label: '创建机器人', icon: 'plus-circle' },
  { id: 'avatar', label: '我的分身', icon: 'user-circle' }
]
```

- [ ] **步骤 3：添加 Tab 切换逻辑**

```typescript
const activeTab = ref('chat')

function handleTabChange(tabId: string) {
  activeTab.value = tabId
}
```

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/components/apps/ai/ChatCenter.vue
git commit -m "feat: add avatar tab to AI assistant center"
```

---

### 任务 2：创建我的分身面板组件

**文件：**
- 创建：`qim-client/src/components/apps/ai/MyAvatar.vue`

- [ ] **步骤 1：创建基础组件结构**

```vue
<template>
  <div class="my-avatar-panel">
    <div class="avatar-header">
      <div class="avatar-avatar">
        <img :src="currentUser.avatar" alt="avatar" />
        <span class="learning-badge" v-if="learningProgress < 100">学习中</span>
      </div>
      <div class="avatar-info">
        <h3>{{ currentUser.name }}的分身</h3>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: learningProgress + '%' }"></div>
        </div>
        <span class="progress-text">学习进度: {{ learningProgress }}%</span>
      </div>
    </div>
    
    <div class="persona-preview" v-if="persona">
      <h4>人设预览:</h4>
      <p>{{ persona.description }}</p>
    </div>
    
    <div class="tools-section">
      <h4>可用能力:</h4>
      <div class="tools-grid">
        <label v-for="tool in availableTools" :key="tool.id" class="tool-checkbox">
          <input type="checkbox" :checked="tool.enabled" @change="toggleTool(tool.id)" />
          <span>{{ tool.name }}</span>
        </label>
      </div>
    </div>
    
    <div class="actions">
      <button class="btn-primary" @click="toggleAvatar">{{ avatarEnabled ? '关闭分身' : '开启分身' }}</button>
      <button class="btn-secondary" @click="goToSettings">详细设置</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useCurrentUser } from '@/composables/useCurrentUser'
import { useAvatar } from '@/composables/useAvatar'

const currentUser = useCurrentUser()
const { avatarConfig, learningProgress, persona, availableTools, enabled: avatarEnabled } = useAvatar()

function toggleTool(toolId: string) {
  // 后续实现工具开关
}

function toggleAvatar() {
  // 后续实现分身开关
}

function goToSettings() {
  // 跳转分身设置
}

onMounted(() => {
  // 初始化数据
})
</script>

<style scoped>
.my-avatar-panel {
  padding: 20px;
  max-width: 600px;
  margin: 0 auto;
}

.avatar-header {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
}

.avatar-avatar {
  position: relative;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
}

.learning-badge {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--primary-color);
  color: white;
  font-size: 12px;
  text-align: center;
  padding: 2px;
}

.progress-bar {
  height: 8px;
  background: #eee;
  border-radius: 4px;
  overflow: hidden;
  margin: 8px 0;
}

.progress-fill {
  height: 100%;
  background: var(--primary-color);
  transition: width 0.3s;
}

.tools-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin-top: 12px;
}

.tool-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.actions {
  display: flex;
  gap: 12px;
  margin-top: 24px;
}
</style>
```

- [ ] **步骤 2：创建组件文件**

运行：`cat > qim-client/src/components/apps/ai/MyAvatar.vue << 'EOF'`（粘贴上述内容）

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/apps/ai/MyAvatar.vue
git commit -m "feat: create MyAvatar component"
```

---

### 任务 3：在 ChatCenter 中引入 MyAvatar 组件

**文件：**
- 修改：`qim-client/src/components/apps/ai/ChatCenter.vue`

- [ ] **步骤 1：导入 MyAvatar 组件**

```typescript
import MyAvatar from './MyAvatar.vue'
```

- [ ] **步骤 2：添加 Tab 内容渲染**

```typescript
<template>
  <div class="chat-center">
    <div class="tabs">
      <button 
        v-for="tab in tabs" 
        :key="tab.id"
        :class="['tab-btn', { active: activeTab === tab.id }]"
        @click="handleTabChange(tab.id)"
      >
        <component :is="getIcon(tab.icon)" />
        {{ tab.label }}
      </button>
    </div>
    
    <div class="tab-content">
      <ChatPanel v-if="activeTab === 'chat'" />
      <MyBotsPanel v-if="activeTab === 'bots'" />
      <CreateBotWizard v-if="activeTab === 'create'" />
      <MyAvatar v-if="activeTab === 'avatar'" />
    </div>
  </div>
</template>
```

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/apps/ai/ChatCenter.vue
git commit -m "feat: integrate MyAvatar component into ChatCenter"
```

---

## Phase 2：能力复用（第2-3周）

### 任务 4：新增 AvatarWithTools 类型

**文件：**
- 修改：`qim-client/src/types/avatar.ts`

- [ ] **步骤 1：查看现有类型定义**

运行：`cat qim-client/src/types/avatar.ts`

- [ ] **步骤 2：添加新类型**

```typescript
export interface AvatarToolBinding {
  avatarId: string
  toolId: string
  enabled: boolean
  priority: number
}

export interface AvatarWithTools {
  id: string
  enabled: boolean
  persona: AvatarPersona
  availableTools: AITool[]
  lastActiveAt: Date
}
```

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/types/avatar.ts
git commit -m "feat: add AvatarWithTools type"
```

---

### 任务 5：新增工具绑定 API

**文件：**
- 修改：`qim-client/src/api/avatar.ts`

- [ ] **步骤 1：查看现有 API**

运行：`cat qim-client/src/api/avatar.ts`

- [ ] **步骤 2：添加工具相关 API**

```typescript
export async function getAvatarWithTools(avatarId: string): Promise<AvatarWithTools> {
  const [avatar, tools] = await Promise.all([
    fetchAvatar(avatarId),
    fetchAITools()
  ])
  return { ...avatar, availableTools: tools }
}

export async function bindToolToAvatar(avatarId: string, toolId: string): Promise<void> {
  await request({
    url: `/api/avatar/${avatarId}/tools/${toolId}`,
    method: 'POST'
  })
}

export async function unbindToolFromAvatar(avatarId: string, toolId: string): Promise<void> {
  await request({
    url: `/api/avatar/${avatarId}/tools/${toolId}`,
    method: 'DELETE'
  })
}
```

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/api/avatar.ts
git commit -m "feat: add tool binding APIs"
```

---

### 任务 6：后端 - 新增工具绑定模型

**文件：**
- 修改：`qim-server/model/avatar.go`

- [ ] **步骤 1：添加 AvatarToolBinding 模型**

```go
type AvatarToolBinding struct {
  ID        uint   `gorm:"primaryKey"`
  AvatarID  uint   `gorm:"index"`
  ToolID    string `gorm:"size:64"`
  Enabled   bool   `gorm:"default:true"`
  Priority  int    `gorm:"default:1"`
  CreatedAt time.Time
  UpdatedAt time.Time
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/model/avatar.go
git commit -m "feat: add AvatarToolBinding model"
```

---

### 任务 7：后端 - 新增工具绑定 Handler

**文件：**
- 修改：`qim-server/handler/avatar_handler.go`

- [ ] **步骤 1：添加工具绑定接口**

```go
func GetAvatarTools(c *gin.Context) {
  avatarID := c.Param("id")
  var bindings []AvatarToolBinding
  db := database.GetDB()
  db.Where("avatar_id = ?", avatarID).Find(&bindings)
  
  tools := getAvailableTools()
  result := make([]map[string]interface{}, 0)
  for _, tool := range tools {
    bound := false
    priority := 1
    for _, b := range bindings {
      if b.ToolID == tool.ID {
        bound = b.Enabled
        priority = b.Priority
        break
      }
    }
    result = append(result, map[string]interface{}{
      "id":          tool.ID,
      "name":        tool.Name,
      "description": tool.Description,
      "enabled":     bound,
      "priority":    priority,
    })
  }
  
  c.JSON(http.StatusOK, gin.H{"data": result})
}

func BindTool(c *gin.Context) {
  avatarID := c.Param("id")
  toolID := c.Param("toolId")
  
  db := database.GetDB()
  var binding AvatarToolBinding
  if err := db.Where("avatar_id = ? AND tool_id = ?", avatarID, toolID).First(&binding).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
      binding = AvatarToolBinding{
        AvatarID: mustParseUint(avatarID),
        ToolID:   toolID,
        Enabled:  true,
        Priority: 1,
      }
      db.Create(&binding)
    } else {
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      return
    }
  } else {
    binding.Enabled = true
    db.Save(&binding)
  }
  
  c.JSON(http.StatusOK, gin.H{"success": true})
}

func UnbindTool(c *gin.Context) {
  avatarID := c.Param("id")
  toolID := c.Param("toolId")
  
  db := database.GetDB()
  if err := db.Where("avatar_id = ? AND tool_id = ?", avatarID, toolID).
    Update("enabled", false).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  
  c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/handler/avatar_handler.go
git commit -m "feat: add tool binding handlers"
```

---

### 任务 8：注册工具绑定路由

**文件：**
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：添加路由**

```go
avatarGroup := r.Group("/api/avatar/:id")
{
  avatarGroup.GET("/tools", handler.GetAvatarTools)
  avatarGroup.POST("/tools/:toolId", handler.BindTool)
  avatarGroup.DELETE("/tools/:toolId", handler.UnbindTool)
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/app/routes.go
git commit -m "feat: register tool binding routes"
```

---

## Phase 3：风格学习深化（第4-5周）

### 任务 9：扩展学习来源

**文件：**
- 修改：`qim-server/service/avatar_service.go`

- [ ] **步骤 1：添加多来源学习逻辑**

```go
func (s *AvatarService) LearnFromMultipleSources(userID uint) error {
  db := s.db
  
  messages := make([]model.Message, 0)
  db.Where("sender_id = ?", userID).Order("created_at DESC").Limit(500).Find(&messages)
  
  var botConfigs []model.Bot
  db.Where("owner_id = ?", userID).Find(&botConfigs)
  
  var aiActions []model.AIActionLog
  db.Where("user_id = ?", userID).Order("created_at DESC").Limit(100).Find(&aiActions)
  
  learningData := LearningData{
    Messages:     processMessages(messages),
    BotConfigs:   processBotConfigs(botConfigs),
    AIActions:    processAIActions(aiActions),
    MessageWeight: 0.6,
    BotWeight:     0.2,
    ActionWeight:  0.2,
  }
  
  return s.UpdatePersona(userID, learningData)
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/service/avatar_service.go
git commit -m "feat: extend persona learning sources"
```

---

### 任务 10：前端 - 实现工具开关功能

**文件：**
- 修改：`qim-client/src/composables/useAvatar.ts`

- [ ] **步骤 1：添加工具操作方法**

```typescript
export function useAvatar() {
  const { data: avatarWithTools } = useRequest(() => getAvatarWithTools(currentUser.value.id))
  
  async function toggleTool(toolId: string) {
    const tool = avatarWithTools.value?.availableTools.find(t => t.id === toolId)
    if (!tool) return
    
    if (tool.enabled) {
      await unbindToolFromAvatar(currentUser.value.id, toolId)
    } else {
      await bindToolToAvatar(currentUser.value.id, toolId)
    }
    
    refresh()
  }
  
  return {
    avatarWithTools,
    toggleTool,
    // ... 其他方法
  }
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/composables/useAvatar.ts
git commit -m "feat: implement tool toggle functionality"
```

---

## 测试计划

### 前端测试

| 测试项 | 文件 | 命令 |
|--------|------|------|
| Tab 切换 | `tests/unit/ai-components.test.ts` | `vitest run tests/unit/ai-components.test.ts` |
| 工具绑定 | `tests/unit/ai-actions.test.ts` | `vitest run tests/unit/ai-actions.test.ts` |

### 后端测试

| 测试项 | 文件 | 命令 |
|--------|------|------|
| 工具绑定 API | `tests/unit/avatar_handler_test.go` | `go test ./tests/unit/... -run Avatar` |

---

## 注意事项

1. **向后兼容**：所有变更均为增量增强，不影响现有功能
2. **权限控制**：工具绑定需要验证用户身份
3. **限流保护**：后端需添加 API 限流
4. **错误处理**：前端需处理网络错误和权限错误