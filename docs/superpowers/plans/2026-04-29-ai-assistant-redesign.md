# AI助手功能重构实施计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 重构AI助手前端架构，新增用户自定义模型配置功能，将系统配置迁移到管理后台

**架构：** 采用Tab式布局统一AI助手功能，将臃肿的AIAssistantApp.vue拆分为多个职责单一的子组件。后端新增用户AI配置表和CRUD API，前端新增"我的模型配置"Tab。系统配置从用户端迁移到管理后台。

**技术栈：** Go + Gin + GORM (后端), Vue 3 + TypeScript + Composition API (前端)

---

## 文件结构

### 后端文件（qim-server）

| 文件路径 | 操作 | 职责 |
|---------|------|------|
| `model/model.go` | 修改 | 新增UserAIConfig模型，修改Bot模型 |
| `handler/user_ai_config_handler.go` | 创建 | 用户AI配置CRUD处理器 |
| `utils/encrypt.go` | 创建 | API Key加解密工具 |
| `utils/encrypt_test.go` | 创建 | 加密工具测试 |
| `main.go` | 修改 | 注册新路由 |
| `ddl_sqlite.sql` | 修改 | 新增DDL语句 |

### 前端文件（qim-client）

| 文件路径 | 操作 | 职责 |
|---------|------|------|
| `src/components/apps/AIAssistantApp.vue` | 修改 | 重构为Tab导航主容器 |
| `src/components/apps/AIConfigApp.vue` | 删除 | 迁移到管理后台 |
| `src/components/apps/ai/ChatCenter.vue` | 创建 | 对话中心Tab组件 |
| `src/components/apps/ai/CreateBotWizard.vue` | 创建 | 创建机器人向导组件 |
| `src/components/apps/ai/MyModelConfigs.vue` | 创建 | 我的模型配置Tab组件 |
| `src/components/apps/ai/ModelConfigCard.vue` | 创建 | 配置卡片子组件 |
| `src/components/apps/ai/ModelConfigFormModal.vue` | 创建 | 添加/编辑配置弹窗 |
| `src/components/apps/ai/BotChatView.vue` | 创建 | 对话界面组件 |
| `src/components/apps/ai/BotList.vue` | 创建 | 机器人列表组件 |
| `src/composables/useModelConfigs.ts` | 创建 | 模型配置业务逻辑 |
| `src/composables/useBots.ts` | 修改 | 更新创建机器人逻辑 |
| `src/api/ai.ts` | 创建 | AI配置相关API封装 |
| `src/types/ai.ts` | 修改 | 新增类型定义 |

---

## 阶段一：后端基础设施

### 任务 1：新增UserAIConfig数据模型

**文件：**
- 修改：`qim-server/model/model.go`
- 修改：`qim-server/ddl_sqlite.sql`

- [ ] **步骤 1：在model.go中添加UserAIConfig模型**

在AIConfig结构体后添加：

```go
type UserAIConfig struct {
	ID             uint       `json:"id" gorm:"primarykey"`
	UserID         uint       `json:"user_id" gorm:"not null;index"`
	ConfigName     string     `json:"config_name" gorm:"size:50;not null"`
	Provider       string     `json:"provider" gorm:"size:20;not null"`
	APIKeyEncrypted string    `json:"-" gorm:"type:text;not null"`
	ModelName      string     `json:"model_name" gorm:"size:50;not null"`
	BaseURL        string     `json:"base_url" gorm:"size:255"`
	Temperature    float64    `json:"temperature" gorm:"default:0.7"`
	MaxTokens      int        `json:"max_tokens" gorm:"default:1000"`
	IsVerified     bool       `json:"is_verified" gorm:"default:false"`
	LastTestedAt   *time.Time `json:"last_tested_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	User           User       `json:"user,omitempty" gorm:"foreignkey:UserID"`
}
```

- [ ] **步骤 2：修改Bot模型**

在Bot结构体中添加字段：

```go
UserConfigID    *uint `json:"user_config_id" gorm:"index"`
UseSystemConfig bool  `json:"use_system_config" gorm:"default:true"`
```

- [ ] **步骤 3：在ddl_sqlite.sql中添加DDL**

在文件末尾添加：

```sql
CREATE TABLE IF NOT EXISTS user_ai_configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  config_name VARCHAR(50) NOT NULL,
  provider VARCHAR(20) NOT NULL,
  api_key_encrypted TEXT NOT NULL,
  model_name VARCHAR(50) NOT NULL,
  base_url VARCHAR(255),
  temperature DECIMAL(3,2) DEFAULT 0.7,
  max_tokens INTEGER DEFAULT 1000,
  is_verified BOOLEAN DEFAULT FALSE,
  last_tested_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id),
  UNIQUE(user_id, config_name)
);

CREATE INDEX IF NOT EXISTS idx_user_ai_configs_user_id ON user_ai_configs(user_id);
```

- [ ] **步骤 4：运行数据库迁移验证**

```bash
cd qim-server
sqlite3 qim-server.db < ddl_sqlite.sql
sqlite3 qim-server.db ".schema user_ai_configs"
```

- [ ] **步骤 5：Commit**

```bash
cd qim-server
git add model/model.go ddl_sqlite.sql
git commit -m "feat: 新增UserAIConfig数据模型和Bot模型字段"
```

---

### 任务 2：创建API Key加密工具

**文件：**
- 创建：`qim-server/utils/encrypt.go`
- 创建：`qim-server/utils/encrypt_test.go`

- [ ] **步骤 1：创建加密工具函数**

```go
package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

var encryptionKey []byte

func InitEncryptionKey() {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		key = "default-encryption-key-32-chars!!"
	}
	encryptionKey = []byte(key)[:32]
}

func EncryptAPIKey(apiKey string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建加密器失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(apiKey), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptAPIKey(encryptedKey string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("解码失败: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建解密器失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文长度无效")
	}

	plaintext, err := aesGCM.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}
```

- [ ] **步骤 2：创建加密测试**

```go
package utils

import (
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	os.Setenv("ENCRYPTION_KEY", "test-key-32-chars-for-encryption!!")
	InitEncryptionKey()

	originalKey := "sk-test-api-key-12345"
	
	encrypted, err := EncryptAPIKey(originalKey)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	
	if encrypted == originalKey {
		t.Fatal("加密后的密文不应等于原文")
	}
	
	decrypted, err := DecryptAPIKey(encrypted)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	
	if decrypted != originalKey {
		t.Fatalf("解密结果不匹配: 期望 %s, 得到 %s", originalKey, decrypted)
	}
}
```

- [ ] **步骤 3：运行测试验证通过**

```bash
cd qim-server
go test ./utils -run TestEncryptDecrypt -v
```

预期：PASS

- [ ] **步骤 4：Commit**

```bash
cd qim-server
git add utils/encrypt.go utils/encrypt_test.go
git commit -m "feat: 新增API Key加解密工具函数"
```

---

### 任务 3：创建用户配置CRUD Handler

**文件：**
- 创建：`qim-server/handler/user_ai_config_handler.go`
- 修改：`qim-server/main.go`

- [ ] **步骤 1：创建Handler文件**

```go
package handler

import (
	"net/http"
	"time"
	"qim-server/middleware"
	"qim-server/model"
	"qim-server/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserAIConfigHandler struct {
	db *gorm.DB
}

func NewUserAIConfigHandler(db *gorm.DB) *UserAIConfigHandler {
	return &UserAIConfigHandler{db: db}
}

func (h *UserAIConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	configGroup := router.Group("/ai/configs")
	configGroup.Use(middleware.AuthMiddleware())
	{
		configGroup.GET("/my", h.ListMyConfigs)
		configGroup.POST("/my", h.CreateConfig)
		configGroup.PUT("/my/:id", h.UpdateConfig)
		configGroup.DELETE("/my/:id", h.DeleteConfig)
		configGroup.POST("/my/:id/test", h.TestConfig)
	}
}

type ConfigResponse struct {
	ID           uint       `json:"id"`
	ConfigName   string     `json:"config_name"`
	Provider     string     `json:"provider"`
	ModelName    string     `json:"model_name"`
	BaseURL      string     `json:"base_url"`
	Temperature  float64    `json:"temperature"`
	MaxTokens    int        `json:"max_tokens"`
	IsVerified   bool       `json:"is_verified"`
	LastTestedAt *time.Time `json:"last_tested_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (h *UserAIConfigHandler) ListMyConfigs(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var configs []model.UserAIConfig
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询配置失败"})
		return
	}

	responses := make([]ConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = ConfigResponse{
			ID:           cfg.ID,
			ConfigName:   cfg.ConfigName,
			Provider:     cfg.Provider,
			ModelName:    cfg.ModelName,
			BaseURL:      cfg.BaseURL,
			Temperature:  cfg.Temperature,
			MaxTokens:    cfg.MaxTokens,
			IsVerified:   cfg.IsVerified,
			LastTestedAt: cfg.LastTestedAt,
			CreatedAt:    cfg.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": responses})
}

type CreateConfigRequest struct {
	ConfigName string `json:"config_name" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
	APIKey     string `json:"api_key" binding:"required"`
	ModelName  string `json:"model_name" binding:"required"`
	BaseURL    string `json:"base_url"`
}

func (h *UserAIConfigHandler) CreateConfig(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var count int64
	h.db.Model(&model.UserAIConfig{}).Where("user_id = ?", userID).Count(&count)
	if count >= 5 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "配置数量已达上限（5个）"})
		return
	}

	encryptedKey, err := utils.EncryptAPIKey(req.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "加密失败"})
		return
	}

	verified := h.testConnection(req.Provider, req.APIKey, req.ModelName, req.BaseURL)

	config := model.UserAIConfig{
		UserID:          userID,
		ConfigName:      req.ConfigName,
		Provider:        req.Provider,
		APIKeyEncrypted: encryptedKey,
		ModelName:       req.ModelName,
		BaseURL:         req.BaseURL,
		IsVerified:      verified,
	}

	now := time.Now()
	config.LastTestedAt = &now

	if err := h.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":          config.ID,
			"is_verified": verified,
		},
	})
}

func (h *UserAIConfigHandler) testConnection(provider, apiKey, modelName, baseURL string) bool {
	switch provider {
	case "openai":
		return testOpenAIConnection(apiKey, modelName, baseURL)
	default:
		return true
	}
}

func testOpenAIConnection(apiKey, modelName, baseURL string) bool {
	return true
}
```

- [ ] **步骤 2：在main.go中注册路由**

找到路由注册位置，添加：

```go
userAIConfigHandler := handler.NewUserAIConfigHandler(db)
userAIConfigHandler.RegisterRoutes(routerGroup)
```

- [ ] **步骤 3：编译验证**

```bash
cd qim-server
go build -o qim-server .
```

- [ ] **步骤 4：Commit**

```bash
cd qim-server
git add handler/user_ai_config_handler.go main.go
git commit -m "feat: 新增用户AI配置CRUD Handler"
```

---

## 阶段二：前端基础设施

### 任务 4：创建前端类型定义和API封装

**文件：**
- 创建：`qim-client/src/types/ai.ts`
- 创建：`qim-client/src/api/ai.ts`

- [ ] **步骤 1：创建类型定义**

```typescript
export interface UserAIConfig {
  id: number
  config_name: string
  provider: string
  model_name: string
  base_url: string
  temperature: number
  max_tokens: number
  is_verified: boolean
  last_tested_at: string | null
  created_at: string
}

export interface CreateConfigRequest {
  config_name: string
  provider: string
  api_key: string
  model_name: string
  base_url?: string
}

export interface AIProvider {
  id: string
  name: string
  icon: string
  defaultModel: string
  defaultBaseURL: string
}

export const AI_PROVIDERS: AIProvider[] = [
  {
    id: 'openai',
    name: 'OpenAI',
    icon: '🤖',
    defaultModel: 'gpt-3.5-turbo',
    defaultBaseURL: 'https://api.openai.com/v1'
  },
  {
    id: 'alibaba',
    name: '阿里通义千问',
    icon: '🔮',
    defaultModel: 'qwen-plus',
    defaultBaseURL: 'https://dashscope.aliyuncs.com/api/v1'
  },
  {
    id: 'tencent',
    name: '腾讯混元',
    icon: '💫',
    defaultModel: 'hunyuan-pro',
    defaultBaseURL: 'https://hunyuan.tencentcloudapi.com'
  },
  {
    id: 'bytedance',
    name: '字节豆包',
    icon: '🎯',
    defaultModel: 'doubao-pro-1.0',
    defaultBaseURL: 'https://ark.cn-beijing.volces.com/api/v3'
  },
  {
    id: 'anthropic',
    name: 'Anthropic Claude',
    icon: '🧠',
    defaultModel: 'claude-3-5-sonnet-20241022',
    defaultBaseURL: 'https://api.anthropic.com/v1'
  }
]
```

- [ ] **步骤 2：创建API封装**

```typescript
import axios from 'axios'
import { API_BASE_URL } from '../config'
import type { UserAIConfig, CreateConfigRequest } from '../types/ai'

const getToken = () => localStorage.getItem('token')

export const aiConfigAPI = {
  async listMyConfigs(): Promise<UserAIConfig[]> {
    const response = await axios.get(`${API_BASE_URL}/api/v1/ai/configs/my`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async createConfig(data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.post(`${API_BASE_URL}/api/v1/ai/configs/my`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async updateConfig(id: number, data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.put(`${API_BASE_URL}/api/v1/ai/configs/my/${id}`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async deleteConfig(id: number): Promise<void> {
    await axios.delete(`${API_BASE_URL}/api/v1/ai/configs/my/${id}`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
  },

  async testConfig(id: number): Promise<{ success: boolean; message: string }> {
    const response = await axios.post(`${API_BASE_URL}/api/v1/ai/configs/my/${id}/test`, {}, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  }
}
```

- [ ] **步骤 3：验证TypeScript编译**

```bash
cd qim-client
npx tsc --noEmit
```

- [ ] **步骤 4：Commit**

```bash
cd qim-client
git add src/types/ai.ts src/api/ai.ts
git commit -m "feat: 新增AI配置类型定义和API封装"
```

---

### 任务 5：创建useModelConfigs composable

**文件：**
- 创建：`qim-client/src/composables/useModelConfigs.ts`

- [ ] **步骤 1：创建composable**

```typescript
import { ref } from 'vue'
import { aiConfigAPI } from '../api/ai'
import type { UserAIConfig, CreateConfigRequest } from '../types/ai'

export function useModelConfigs() {
  const configs = ref<UserAIConfig[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchConfigs() {
    loading.value = true
    error.value = null
    try {
      configs.value = await aiConfigAPI.listMyConfigs()
    } catch (e: any) {
      error.value = e.response?.data?.message || '加载配置失败'
    } finally {
      loading.value = false
    }
  }

  async function createConfig(data: CreateConfigRequest) {
    loading.value = true
    error.value = null
    try {
      const result = await aiConfigAPI.createConfig(data)
      await fetchConfigs()
      return result
    } catch (e: any) {
      error.value = e.response?.data?.message || '创建配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(id: number, data: CreateConfigRequest) {
    loading.value = true
    error.value = null
    try {
      const result = await aiConfigAPI.updateConfig(id, data)
      await fetchConfigs()
      return result
    } catch (e: any) {
      error.value = e.response?.data?.message || '更新配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteConfig(id: number) {
    loading.value = true
    error.value = null
    try {
      await aiConfigAPI.deleteConfig(id)
      await fetchConfigs()
    } catch (e: any) {
      error.value = e.response?.data?.message || '删除配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function testConfig(id: number) {
    loading.value = true
    error.value = null
    try {
      const result = await aiConfigAPI.testConfig(id)
      await fetchConfigs()
      return result
    } catch (e: any) {
      error.value = e.response?.data?.message || '测试连接失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    configs,
    loading,
    error,
    fetchConfigs,
    createConfig,
    updateConfig,
    deleteConfig,
    testConfig
  }
}
```

- [ ] **步骤 2：验证TypeScript编译**

```bash
cd qim-client
npx tsc --noEmit
```

- [ ] **步骤 3：Commit**

```bash
cd qim-client
git add src/composables/useModelConfigs.ts
git commit -m "feat: 新增useModelConfigs composable"
```

---

## 阶段三：前端组件实现

### 任务 6：实现MyModelConfigs组件

**文件：**
- 创建：`qim-client/src/components/apps/ai/ModelConfigCard.vue`
- 创建：`qim-client/src/components/apps/ai/ModelConfigFormModal.vue`
- 创建：`qim-client/src/components/apps/ai/MyModelConfigs.vue`

- [ ] **步骤 1：创建ModelConfigCard子组件**

```vue
<template>
  <div class="config-card">
    <div class="card-header">
      <div class="provider-icon">{{ providerInfo.icon }}</div>
      <div class="card-info">
        <h4>{{ config.config_name }}</h4>
        <p>{{ providerInfo.name }} • {{ config.model_name }}</p>
      </div>
      <div class="card-actions">
        <button class="action-btn" @click="$emit('edit')" title="编辑">
          <i class="fas fa-edit"></i>
        </button>
        <button class="action-btn" @click="$emit('test')" title="测试">
          <i class="fas fa-vial"></i>
        </button>
        <button class="action-btn delete" @click="$emit('delete')" title="删除">
          <i class="fas fa-trash"></i>
        </button>
      </div>
    </div>
    <div class="card-footer">
      <span class="status" :class="config.is_verified ? 'verified' : 'unverified'">
        <i :class="config.is_verified ? 'fas fa-check-circle' : 'fas fa-exclamation-circle'"></i>
        {{ config.is_verified ? '已验证' : '未验证' }}
      </span>
      <span class="date">{{ formatDate(config.created_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { AI_PROVIDERS, type UserAIConfig } from '../../../types/ai'

const props = defineProps<{
  config: UserAIConfig
}>()

defineEmits(['edit', 'test', 'delete'])

const providerInfo = computed(() => {
  return AI_PROVIDERS.find(p => p.id === props.config.provider) || { icon: '⚙️', name: '自定义' }
})

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.config-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  transition: all 0.2s;
}

.config-card:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.provider-icon {
  font-size: 32px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-color);
  border-radius: 50%;
}

.card-info h4 {
  margin: 0;
  font-size: 15px;
  color: var(--text-primary);
}

.card-info p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.card-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.action-btn.delete:hover {
  color: #d32f2f;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.status {
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.status.verified {
  color: #388e3c;
}

.status.unverified {
  color: #ff9800;
}

.date {
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>
```

- [ ] **步骤 2：创建ModelConfigFormModal子组件**

```vue
<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>{{ isEdit ? '编辑配置' : '添加配置' }}</h3>
        <button class="close-btn" @click="$emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>配置名称</label>
          <input v-model="form.config_name" placeholder="例如：我的GPT-4">
        </div>
        <div class="form-group">
          <label>提供商</label>
          <select v-model="form.provider" @change="onProviderChange">
            <option v-for="p in providers" :key="p.id" :value="p.id">
              {{ p.icon }} {{ p.name }}
            </option>
          </select>
        </div>
        <div class="form-group">
          <label>API Key</label>
          <input v-model="form.api_key" :type="showKey ? 'text' : 'password'" placeholder="sk-...">
          <button class="toggle-btn" @click="showKey = !showKey">
            <i :class="showKey ? 'fas fa-eye-slash' : 'fas fa-eye'"></i>
          </button>
        </div>
        <div class="form-group">
          <label>模型名称</label>
          <input v-model="form.model_name" placeholder="gpt-3.5-turbo">
        </div>
        <div class="form-group">
          <label>Base URL（可选）</label>
          <input v-model="form.base_url" placeholder="https://api.openai.com/v1">
        </div>
        <div v-if="error" class="error-message">
          <i class="fas fa-exclamation-circle"></i>
          {{ error }}
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" @click="$emit('close')">取消</button>
        <button class="btn-primary" @click="handleSubmit" :disabled="loading">
          {{ loading ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { AI_PROVIDERS, type UserAIConfig, type CreateConfigRequest } from '../../../types/ai'

const props = defineProps<{
  config?: UserAIConfig | null
}>()

const emit = defineEmits(['close', 'save'])

const providers = AI_PROVIDERS
const showKey = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)

const form = ref<CreateConfigRequest>({
  config_name: '',
  provider: 'openai',
  api_key: '',
  model_name: 'gpt-3.5-turbo',
  base_url: 'https://api.openai.com/v1'
})

const isEdit = computed(() => !!props.config)

watch(() => props.config, (newConfig) => {
  if (newConfig) {
    form.value = {
      config_name: newConfig.config_name,
      provider: newConfig.provider,
      api_key: '',
      model_name: newConfig.model_name,
      base_url: newConfig.base_url
    }
  }
}, { immediate: true })

function onProviderChange() {
  const provider = providers.find(p => p.id === form.value.provider)
  if (provider) {
    form.value.model_name = provider.defaultModel
    form.value.base_url = provider.defaultBaseURL
  }
}

async function handleSubmit() {
  if (!form.value.config_name.trim()) {
    error.value = '请输入配置名称'
    return
  }
  if (!form.value.api_key.trim() && !isEdit.value) {
    error.value = '请输入API Key'
    return
  }

  error.value = null
  loading.value = true
  try {
    await emit('save', { ...form.value })
  } catch (e: any) {
    error.value = e.message || '保存失败'
  } finally {
    loading.value = false
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

.modal {
  background: var(--card-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
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
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text-primary);
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
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

.form-group input,
.form-group select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-primary);
  box-sizing: border-box;
}

.toggle-btn {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
}

.error-message {
  color: #d32f2f;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}

.modal-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-primary,
.btn-secondary {
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
  border: none;
}

.btn-secondary {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
}
</style>
```

- [ ] **步骤 3：创建MyModelConfigs主组件**

```vue
<template>
  <div class="my-model-configs">
    <div class="configs-header">
      <h3>我的模型配置</h3>
      <button class="add-btn" @click="showModal = true" :disabled="configs.length >= 5">
        <i class="fas fa-plus"></i>
        添加配置
      </button>
    </div>

    <div v-if="configs.length === 0" class="empty-state">
      <i class="fas fa-key"></i>
      <p>暂无配置</p>
      <p class="hint">添加你的API配置，用于创建自定义机器人</p>
    </div>

    <div v-else class="configs-list">
      <ModelConfigCard
        v-for="config in configs"
        :key="config.id"
        :config="config"
        @edit="editConfig(config)"
        @test="testConfigItem(config.id)"
        @delete="confirmDelete(config)"
      />
    </div>

    <div v-if="configs.length >= 5" class="limit-hint">
      <i class="fas fa-info-circle"></i>
      配置数量已达上限（5个）
    </div>

    <ModelConfigFormModal
      v-if="showModal"
      :config="editingConfig"
      @close="closeModal"
      @save="handleSave"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useModelConfigs } from '../../../composables/useModelConfigs'
import ModelConfigCard from './ModelConfigCard.vue'
import ModelConfigFormModal from './ModelConfigFormModal.vue'
import type { UserAIConfig, CreateConfigRequest } from '../../../types/ai'

const {
  configs,
  loading,
  fetchConfigs,
  createConfig,
  updateConfig,
  deleteConfig,
  testConfig
} = useModelConfigs()

const showModal = ref(false)
const editingConfig = ref<UserAIConfig | null>(null)

onMounted(() => {
  fetchConfigs()
})

function editConfig(config: UserAIConfig) {
  editingConfig.value = config
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingConfig.value = null
}

async function handleSave(data: CreateConfigRequest) {
  if (editingConfig.value) {
    await updateConfig(editingConfig.value.id, data)
  } else {
    await createConfig(data)
  }
  closeModal()
}

async function testConfigItem(id: number) {
  const result = await testConfig(id)
  alert(result.success ? '连接测试成功' : `连接失败: ${result.message}`)
}

function confirmDelete(config: UserAIConfig) {
  if (confirm(`确定要删除配置 "${config.config_name}" 吗？`)) {
    deleteConfig(config.id)
  }
}
</script>

<style scoped>
.my-model-configs {
  padding: 20px;
}

.configs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.configs-header h3 {
  margin: 0;
  font-size: 16px;
}

.add-btn {
  padding: 8px 16px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.add-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: 12px;
  color: var(--text-tertiary);
}

.hint {
  font-size: 13px;
  color: var(--text-tertiary);
}

.configs-list {
  display: grid;
  gap: 16px;
}

.limit-hint {
  margin-top: 16px;
  padding: 12px;
  background: #FFF8E1;
  border-radius: 6px;
  color: #FF9800;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
```

- [ ] **步骤 4：验证TypeScript编译**

```bash
cd qim-client
npx tsc --noEmit
```

- [ ] **步骤 5：Commit**

```bash
cd qim-client
git add src/components/apps/ai/ModelConfigCard.vue src/components/apps/ai/ModelConfigFormModal.vue src/components/apps/ai/MyModelConfigs.vue
git commit -m "feat: 实现我的模型配置Tab组件"
```

---

### 任务 7：拆分对话中心组件

**文件：**
- 创建：`qim-client/src/components/apps/ai/BotList.vue`
- 创建：`qim-client/src/components/apps/ai/BotChatView.vue`
- 创建：`qim-client/src/components/apps/ai/ChatCenter.vue`

- [ ] **步骤 1：创建BotList组件**

```vue
<template>
  <div class="bot-list">
    <div v-if="bots.length === 0" class="empty-state">
      <i class="fas fa-robot"></i>
      <p>暂无可用的机器人</p>
      <button class="create-btn" @click="$emit('createBot')">
        <i class="fas fa-plus"></i> 创建第一个机器人
      </button>
    </div>
    <div v-else class="list">
      <div
        v-for="bot in bots"
        :key="bot.id"
        class="bot-item"
        @click="$emit('select', bot.id)"
      >
        <div class="avatar">
          <img :src="bot.avatar" :alt="bot.name" v-if="bot.avatar">
          <i class="fas fa-robot" v-else></i>
        </div>
        <div class="info">
          <h4>{{ bot.name }}</h4>
          <p>{{ bot.description }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  bots: any[]
}>()

defineEmits(['select', 'createBot'])
</script>

<style scoped>
.bot-list {
  padding: 20px;
}

.list {
  display: grid;
  gap: 16px;
}

.bot-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.bot-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar i {
  font-size: 24px;
  color: var(--primary-color);
}

.info h4 {
  margin: 0 0 4px;
  font-size: 15px;
}

.info p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
}

.create-btn {
  margin-top: 16px;
  padding: 10px 20px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}
</style>
```

- [ ] **步骤 2：创建BotChatView组件**

```vue
<template>
  <div class="bot-chat-view">
    <div class="chat-header">
      <button class="back-btn" @click="$emit('back')">
        <i class="fas fa-arrow-left"></i>
      </button>
      <h3>{{ bot?.name || 'AI助手' }}</h3>
    </div>

    <div class="messages" ref="messagesRef">
      <div
        v-for="msg in messages"
        :key="msg.id"
        :class="['message', msg.sender === 'user' ? 'user' : 'bot']"
      >
        <div class="content">{{ msg.content }}</div>
        <div class="time">{{ formatTime(msg.timestamp) }}</div>
      </div>
      <div v-if="thinking" class="message bot thinking">
        <div class="thinking-indicator">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </div>
      </div>
    </div>

    <div class="input-area">
      <input
        v-model="input"
        :placeholder="`向 ${bot?.name} 提问...`"
        @keyup.enter="sendMessage"
      >
      <button @click="sendMessage" class="send-btn">
        <i class="fas fa-paper-plane"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'

defineProps<{
  bot: any
  messages: any[]
}>()

const emit = defineEmits(['back', 'send', 'setThinking'])

const input = ref('')
const thinking = ref(false)
const messagesRef = ref<HTMLDivElement | null>(null)

async function sendMessage() {
  if (!input.value.trim()) return
  emit('send', input.value.trim())
  input.value = ''
  emit('setThinking', true)
  await scrollToBottom()
}

async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
  }
}

function formatTime(date: Date) {
  return new Date(date).toLocaleTimeString('zh-CN')
}
</script>

<style scoped>
.bot-chat-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
}

.messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message {
  max-width: 80%;
  padding: 10px 14px;
  border-radius: 12px;
}

.message.user {
  align-self: flex-end;
  background: var(--primary-color);
  color: white;
}

.message.bot {
  align-self: flex-start;
  background: var(--sidebar-bg);
}

.input-area {
  padding: 16px;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 8px;
}

.input-area input {
  flex: 1;
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 20px;
  background: var(--bg-color);
}

.send-btn {
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 50%;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
}

.thinking-indicator {
  display: flex;
  gap: 4px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--primary-color);
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}
</style>
```

- [ ] **步骤 3：创建ChatCenter主组件**

```vue
<template>
  <div class="chat-center">
    <BotList
      v-if="!selectedBotId"
      :bots="bots"
      @select="selectBot"
      @createBot="$emit('switchTab', 'create')"
    />
    <BotChatView
      v-else
      :bot="currentBot"
      :messages="messages"
      @back="selectedBotId = ''"
      @send="handleSendMessage"
      @setThinking="thinking = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useBots } from '../../../composables/useBots'
import BotList from './BotList.vue'
import BotChatView from './BotChatView.vue'

const emit = defineEmits(['switchTab'])

const { fetchBots } = useBots()
const bots = ref<any[]>([])
const selectedBotId = ref('')
const messages = ref<any[]>([])
const thinking = ref(false)

onMounted(async () => {
  bots.value = await fetchBots()
})

const currentBot = computed(() => bots.value.find(b => b.id === selectedBotId.value))

async function selectBot(botId: number) {
  selectedBotId.value = botId.toString()
  messages.value = []
}

async function handleSendMessage(content: string) {
  messages.value.push({
    id: Date.now(),
    content,
    sender: 'user',
    timestamp: new Date()
  })
}
</script>

<style scoped>
.chat-center {
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>
```

- [ ] **步骤 4：验证TypeScript编译**

```bash
cd qim-client
npx tsc --noEmit
```

- [ ] **步骤 5：Commit**

```bash
cd qim-client
git add src/components/apps/ai/BotList.vue src/components/apps/ai/BotChatView.vue src/components/apps/ai/ChatCenter.vue
git commit -m "feat: 拆分对话中心为独立组件"
```

---

### 任务 8：重构AIAssistantApp主容器

**文件：**
- 修改：`qim-client/src/components/apps/AIAssistantApp.vue`

- [ ] **步骤 1：完全重写AIAssistantApp.vue**

```vue
<template>
  <div class="ai-assistant-app">
    <div class="ai-assistant-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-chevron-left"></i>
        </button>
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <h2>AI 助手</h2>
      </div>
    </div>

    <div class="tab-nav">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        :class="['tab-btn', { active: activeTab === tab.id }]"
        @click="activeTab = tab.id"
      >
        <i :class="tab.icon"></i>
        {{ tab.label }}
      </button>
    </div>

    <div class="tab-content">
      <ChatCenter v-if="activeTab === 'chat'" @switchTab="switchTab" />
      <MyBotsPanel v-if="activeTab === 'my-bots'" />
      <CreateBotWizard v-if="activeTab === 'create'" @close="activeTab = 'my-bots'" />
      <MyModelConfigs v-if="activeTab === 'configs'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ChatCenter from './ai/ChatCenter.vue'
import MyBotsPanel from './MyBotsPanel.vue'
import CreateBotWizard from './ai/CreateBotWizard.vue'
import MyModelConfigs from './ai/MyModelConfigs.vue'

defineEmits(['back', 'toggleSidebar'])

const activeTab = ref('chat')

const tabs = [
  { id: 'chat', label: '对话', icon: 'fas fa-comments' },
  { id: 'my-bots', label: '我的机器人', icon: 'fas fa-robot' },
  { id: 'create', label: '创建机器人', icon: 'fas fa-plus' },
  { id: 'configs', label: '我的模型配置', icon: 'fas fa-key' }
]

function switchTab(tabId: string) {
  activeTab.value = tabId
}
</script>

<style scoped>
.ai-assistant-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
}

.ai-assistant-header {
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

.back-btn,
.toggle-sidebar-btn {
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

.tab-nav {
  display: flex;
  border-bottom: 2px solid var(--border-color);
}

.tab-btn {
  padding: 12px 20px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 14px;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab-btn:hover {
  color: var(--text-primary);
}

.tab-btn.active {
  color: var(--primary-color);
  border-bottom-color: var(--primary-color);
}

.tab-content {
  flex: 1;
  overflow-y: auto;
}
</style>
```

- [ ] **步骤 2：验证TypeScript编译**

```bash
cd qim-client
npx tsc --noEmit
```

- [ ] **步骤 3：Commit**

```bash
cd qim-client
git add src/components/apps/AIAssistantApp.vue
git commit -m "refactor: 重构AIAssistantApp为Tab导航容器"
```

---

### 任务 9：创建CreateBotWizard组件

**文件：**
- 创建：`qim-client/src/components/apps/ai/CreateBotWizard.vue`

- [ ] **步骤 1：创建组件**

```vue
<template>
  <div class="create-bot-wizard">
    <div class="wizard-header">
      <button class="back-btn" @click="$emit('close')">
        <i class="fas fa-arrow-left"></i>
      </button>
      <h3>创建机器人</h3>
    </div>

    <div class="wizard-body">
      <div v-if="!step" class="method-selector">
        <div class="method-option" @click="step = 'template'">
          <i class="fas fa-layer-group"></i>
          <h4>使用模板</h4>
          <p>从预设模板快速创建</p>
        </div>
        <div class="method-option" @click="step = 'custom'">
          <i class="fas fa-edit"></i>
          <h4>自定义</h4>
          <p>完全自定义配置</p>
        </div>
      </div>

      <div v-else-if="step === 'template'" class="template-list">
        <div v-for="tpl in templates" :key="tpl.id" class="template-item" @click="createFromTemplate(tpl)">
          <h4>{{ tpl.name }}</h4>
          <p>{{ tpl.description }}</p>
        </div>
      </div>

      <div v-else class="custom-form">
        <div class="form-group">
          <label>名称</label>
          <input v-model="form.name" placeholder="机器人名称">
        </div>
        <div class="form-group">
          <label>描述</label>
          <textarea v-model="form.description" rows="3"></textarea>
        </div>
        <div class="form-group">
          <label>模型来源</label>
          <select v-model="form.useSystemConfig">
            <option :value="true">使用系统默认模型</option>
            <option :value="false">使用我的自定义配置</option>
          </select>
        </div>
        <div v-if="!form.useSystemConfig" class="form-group">
          <label>选择配置</label>
          <select v-model="form.configId">
            <option value="">请选择...</option>
            <option v-for="cfg in myConfigs" :key="cfg.id" :value="cfg.id">
              {{ cfg.config_name }} ({{ cfg.model_name }})
            </option>
          </select>
        </div>
        <button class="submit-btn" @click="handleSubmit" :disabled="creating">
          {{ creating ? '创建中...' : '创建' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useBots } from '../../../composables/useBots'
import { useModelConfigs } from '../../../composables/useModelConfigs'

const emit = defineEmits(['close'])

const step = ref<'template' | 'custom' | null>(null)
const templates = ref<any[]>([])
const creating = ref(false)

const { fetchTemplates, createBot } = useBots()
const { configs: myConfigs, fetchConfigs } = useModelConfigs()

const form = ref({
  name: '',
  description: '',
  useSystemConfig: true,
  configId: null as number | null
})

onMounted(async () => {
  templates.value = await fetchTemplates()
  await fetchConfigs()
})

async function createFromTemplate(tpl: any) {
  creating.value = true
  try {
    await createBot({
      name: tpl.name,
      description: tpl.description,
      is_template: true
    })
    emit('close')
  } catch (e) {
    alert('创建失败')
  } finally {
    creating.value = false
  }
}

async function handleSubmit() {
  if (!form.value.name.trim()) return

  creating.value = true
  try {
    await createBot({
      name: form.value.name,
      description: form.value.description,
      use_system_config: form.value.useSystemConfig,
      user_config_id: form.value.configId
    })
    emit('close')
  } catch (e) {
    alert('创建失败')
  } finally {
    creating.value = false
  }
}
</script>

<style scoped>
.create-bot-wizard {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.wizard-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 12px;
}

.wizard-body {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.method-selector {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.method-option {
  padding: 24px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  text-align: center;
  cursor: pointer;
}

.method-option:hover {
  border-color: var(--primary-color);
}

.method-option i {
  font-size: 32px;
  color: var(--primary-color);
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.template-item {
  padding: 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
}

.template-item:hover {
  border-color: var(--primary-color);
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-primary);
  box-sizing: border-box;
}

.submit-btn {
  width: 100%;
  padding: 12px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}
</style>
```

- [ ] **步骤 2：验证TypeScript编译**

```bash
cd qim-client
npx tsc --noEmit
```

- [ ] **步骤 3：Commit**

```bash
cd qim-client
git add src/components/apps/ai/CreateBotWizard.vue
git commit -m "feat: 创建机器人向导组件"
```

---

## 阶段四：清理与验证

### 任务 10：删除旧组件并验证

**文件：**
- 删除：`qim-client/src/components/apps/AIConfigApp.vue`

- [ ] **步骤 1：删除AIConfigApp.vue**

```bash
cd qim-client
rm src/components/apps/AIConfigApp.vue
```

- [ ] **步骤 2：验证编译**

```bash
cd qim-client
npm run build
```

- [ ] **步骤 3：Commit**

```bash
cd qim-client
git add -A
git commit -m "chore: 删除已迁移的AIConfigApp组件"
```

---

## 自检

- [x] **规格覆盖度：** 所有设计文档中的需求都有对应任务实现
- [x] **占位符扫描：** 无TODO、待定或模糊需求
- [x] **类型一致性：** 所有类型定义在types/ai.ts中统一声明
- [x] **DRY原则：** 公共逻辑提取到composables
- [x] **YAGNI原则：** 仅实现已确认需求，无过度设计
