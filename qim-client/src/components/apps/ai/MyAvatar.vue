<template>
  <div class="my-avatar-panel">
    <div class="avatar-view-toggle">
      <button
        :class="['toggle-btn', { active: viewMode === 'overview' }]"
        @click="viewMode = 'overview'"
      >
        <i class="fas fa-chart-pie"></i> 概览
      </button>
      <button
        :class="['toggle-btn', { active: viewMode === 'settings' }]"
        @click="viewMode = 'settings'"
      >
        <i class="fas fa-cog"></i> 详细设置
      </button>
    </div>

    <div v-if="viewMode === 'overview'" class="overview-section">
      <!-- 未创建分身 -->
      <template v-if="!avatarConfig">
        <div class="empty-state">
          <i class="fas fa-user-astronaut empty-icon"></i>
          <h3>还没有分身</h3>
          <p>创建你的 AI 分身，在你不在时代替回复消息</p>
          <button class="btn-primary" @click="viewMode = 'settings'">
            <i class="fas fa-plus"></i> 创建分身
          </button>
        </div>
      </template>

      <!-- 已创建分身 -->
      <template v-else>
        <div class="avatar-header">
          <div class="avatar-avatar">
            <Avatar v-if="currentUser" :src="currentUser.avatar" :name="currentUser.nickname || currentUser.username" :alt="avatar" size="xl" />
            <span class="learning-badge" v-if="learningStatus === 'learning'">学习中</span>
          </div>
          <div class="avatar-info">
            <h3>{{ currentUser ? (currentUser.nickname || currentUser.username) : '' }}的分身</h3>
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

        <div class="actions">
          <!-- 已启用：可关闭 -->
          <button v-if="avatarEnabled && approvalStatus === 'approved'" class="btn-primary" @click="toggleAvatar(false)">
            关闭分身
          </button>
          <!-- 已停用（审批通过但被关闭）：可重新启用 -->
          <button v-else-if="!avatarEnabled && approvalStatus === 'approved'" class="btn-primary" @click="toggleAvatar(true)">
            重新启用
          </button>
          <!-- 待审批 -->
          <button v-else-if="approvalStatus === 'pending'" class="btn-primary btn-disabled" disabled>
            审批中...
          </button>
          <!-- 被拒绝：可重新申请 -->
          <button v-else-if="approvalStatus === 'rejected'" class="btn-primary" @click="toggleAvatar(true)">
            重新申请
          </button>
        </div>
      </template>
    </div>

    <AvatarSettingsPanel v-else-if="viewMode === 'settings'" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useCurrentUser } from '@/composables/useCurrentUser'
import { useAvatar } from '@/composables/useAvatar'
import { useAvatarPersona } from '@/composables/useAvatarPersona'
import { generateAvatar } from '@/utils/avatar'
import AvatarSettingsPanel from '../../avatar/AvatarSettingsPanel.vue'

const viewMode = ref<'overview' | 'settings'>('overview')

const { currentUser } = useCurrentUser()
const avatar = useAvatar()
const personaState = useAvatarPersona()

// 从 useAvatar 获取配置
const avatarConfig = computed(() => avatar.config.value)
const avatarEnabled = computed(() => avatarConfig.value?.enabled ?? false)
const approvalStatus = computed(() => avatar.avatarApprovalStatus.value)

// 从 useAvatarPersona 获取学习状态
const learningStatus = computed(() => personaState.learnStatus.value.status)
const learningProgress = computed(() => personaState.learnStatus.value.progress)
const persona = computed(() => personaState.learnedPersona.value)

async function toggleAvatar(enabled: boolean) {
  if (!avatarConfig.value) return
  try {
    await avatar.toggleEnabled(enabled)
  } catch (e) {
    console.error('切换分身失败:', e)
  }
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
  padding: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.avatar-view-toggle {
  display: flex;
  gap: 8px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--border-color);
}

.toggle-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.toggle-btn:hover {
  background: var(--hover-color);
}

.toggle-btn.active {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.overview-section {
  padding: 20px;
  max-width: 600px;
  margin: 0 auto;
  flex: 1;
  overflow-y: auto;
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

.btn-primary:disabled,
.btn-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
}

.empty-icon {
  font-size: 48px;
  color: var(--text-secondary, #999);
  margin-bottom: 16px;
}

.empty-state h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
}

.empty-state p {
  margin: 0 0 24px 0;
  color: var(--text-secondary, #666);
  font-size: 14px;
}
</style>
