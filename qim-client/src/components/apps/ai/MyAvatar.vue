<template>
  <div class="my-avatar-panel">
    <div class="avatar-header">
      <div class="avatar-avatar">
        <img :src="currentUser.avatar || generateAvatar(currentUser.username)" alt="avatar" />
        <span class="learning-badge" v-if="learningStatus === 'learning'">学习中</span>
      </div>
      <div class="avatar-info">
        <h3>{{ currentUser.nickname || currentUser.username }}的分身</h3>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: learningProgress + '%' }"></div>
        </div>
        <span class="progress-text">学习进度: {{ learningProgress }}%</span>
      </div>
    </div>

    <div class="persona-preview" v-if="persona">
      <h4>人设预览:</h4>
      <p>{{ persona }}</p>
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
      <button class="btn-primary" @click="toggleAvatar">
        {{ avatarEnabled ? '关闭分身' : '开启分身' }}
      </button>
      <button class="btn-secondary" @click="goToSettings">详细设置</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useCurrentUser } from '@/composables/useCurrentUser'
import { useAvatar } from '@/composables/useAvatar'
import { useAvatarPersona } from '@/composables/useAvatarPersona'
import { generateAvatar } from '@/utils/avatar'

const currentUser = useCurrentUser()
const avatar = useAvatar()
const personaState = useAvatarPersona()

// 从 useAvatar 获取配置
const avatarConfig = computed(() => avatar.config.value)
const avatarEnabled = computed(() => avatarConfig.value?.enabled ?? false)

// 从 useAvatarPersona 获取学习状态
const learningStatus = computed(() => personaState.learnStatus.value.status)
const learningProgress = computed(() => personaState.learnStatus.value.progress)
const persona = computed(() => personaState.learnedPersona.value)

// 模拟可用工具（后续需要从配置或API获取）
const availableTools = ref([
  { id: 'chat', name: '智能对话', enabled: true },
  { id: 'reply', name: '自动回复', enabled: false },
  { id: 'summary', name: '摘要总结', enabled: false },
  { id: 'search', name: '知识检索', enabled: false }
])

function toggleTool(toolId: string) {
  const tool = availableTools.value.find(t => t.id === toolId)
  if (tool) {
    tool.enabled = !tool.enabled
  }
}

async function toggleAvatar() {
  if (!avatarConfig.value) return
  try {
    await avatar.toggleEnabled(!avatarEnabled.value)
  } catch (e) {
    console.error('切换分身失败:', e)
  }
}

function goToSettings() {
  // 后续实现跳转分身设置
  console.log('跳转到分身设置页面')
}

onMounted(async () => {
  // 初始化数据
  await avatar.fetchConfig()
  if (avatarConfig.value) {
    await personaState.fetchLearnedPersona()
    await personaState.fetchLearnStatus()
  }
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

.avatar-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
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

.avatar-info {
  flex: 1;
}

.avatar-info h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
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

.progress-text {
  font-size: 12px;
  color: #666;
}

.persona-preview {
  margin-bottom: 24px;
  padding: 12px;
  background: #f5f5f5;
  border-radius: 8px;
}

.persona-preview h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #333;
}

.persona-preview p {
  margin: 0;
  font-size: 14px;
  color: #666;
  line-height: 1.5;
}

.tools-section h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #333;
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
  padding: 8px;
  background: #fafafa;
  border-radius: 4px;
}

.tool-checkbox:hover {
  background: #f0f0f0;
}

.actions {
  display: flex;
  gap: 12px;
  margin-top: 24px;
}

.btn-primary {
  flex: 1;
  padding: 10px 20px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  flex: 1;
  padding: 10px 20px;
  background: white;
  color: var(--primary-color);
  border: 1px solid var(--primary-color);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-secondary:hover {
  background: #f5f5f5;
}
</style>
