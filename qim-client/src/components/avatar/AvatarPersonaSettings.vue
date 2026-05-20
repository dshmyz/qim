<template>
  <div class="avatar-persona-settings">
    <div class="learn-section">
      <div class="learn-header">
        <h4>风格学习</h4>
        <button
          v-if="learnStatus.status !== 'learning'"
          class="learn-btn"
          @click="handleLearn"
          :disabled="learnLoading"
        >
          {{ hasLearned ? '重新学习' : '开始学习' }}
        </button>
      </div>

      <div v-if="learnStatus.status === 'learning'" class="learn-progress">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${learnStatus.progress}%` }"></div>
        </div>
        <span class="progress-text">正在分析 {{ learnStatus.messageCount }} 条历史消息... {{ learnStatus.progress }}%</span>
      </div>

      <div v-else-if="learnStatus.status === 'completed'" class="learn-result">
        <div class="result-header">
          <div class="result-label">学习到的人设风格：</div>
          <button class="clear-btn" @click="handleClear" :disabled="learnLoading">
            清除
          </button>
        </div>
        <div class="result-content">{{ learnedPersona || '暂无' }}</div>
      </div>

      <div v-else-if="learnStatus.status === 'failed'" class="learn-error">
        <span>学习失败：{{ learnStatus.error }}</span>
        <button class="retry-btn" @click="handleLearn">重试</button>
      </div>

      <div v-else class="learn-idle">
        <span class="setting-hint">分身将从你的历史消息中学习说话风格和表达习惯</span>
      </div>
    </div>

    <div class="setting-item">
      <label>补充提示词</label>
      <textarea
        :value="modelValue.customPersonaAddon"
        @input="update('customPersonaAddon', ($event.target as HTMLTextAreaElement).value)"
        class="form-textarea"
        rows="4"
        placeholder="补充描述你的说话习惯、常用表达、专业领域等..."
      ></textarea>
      <span class="setting-hint">在自动学习的基础上，手动补充分身应该知道的关于你的信息</span>
    </div>

    <div class="setting-item">
      <label>风格预览</label>
      <div class="preview-area">
        <input
          v-model="previewInput"
          class="form-input"
          placeholder="输入一段话，预览分身会怎么回复..."
          @keyup.enter="handlePreview"
        />
        <button class="preview-btn" @click="handlePreview" :disabled="previewLoading">
          {{ previewLoading ? '生成中...' : '预览' }}
        </button>
      </div>
      <div v-if="previewResult" class="preview-result">
        <div class="result-label">分身回复：</div>
        <div class="result-content">{{ previewResult }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAvatarPersona } from '../../composables/useAvatarPersona'
import type { AvatarConfig } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

const {
  learnStatus,
  learnedPersona,
  loading: learnLoading,
  triggerLearn,
  fetchLearnStatus,
  fetchLearnedPersona,
  previewReply,
  clearLearnedPersona,
  stopPolling
} = useAvatarPersona()

const previewInput = ref('')
const previewResult = ref('')
const previewLoading = ref(false)

const hasLearned = computed(() => learnStatus.value.status === 'completed')

onMounted(async () => {
  await fetchLearnStatus()
  if (learnStatus.value.status === 'completed' || learnStatus.value.lastLearnedAt) {
    await fetchLearnedPersona()
  }
})

onUnmounted(() => {
  stopPolling()
})

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

async function handleLearn() {
  try {
    await triggerLearn()
    window.$QMessage.success('风格学习已开始')
  } catch {
    window.$QMessage.error('触发学习失败')
  }
}

async function handleClear() {
  try {
    await window.$QMessageBox.confirm('确定要清除学习到的风格吗？', '清除风格')
    await clearLearnedPersona()
    window.$QMessage.success('已清除学习结果')
  } catch {
  }
}

async function handlePreview() {
  if (!previewInput.value.trim() || previewLoading.value) return
  previewLoading.value = true
  try {
    previewResult.value = await previewReply(previewInput.value.trim())
  } catch {
    window.$QMessage.error('预览失败')
  } finally {
    previewLoading.value = false
  }
}
</script>

<style scoped>
.avatar-persona-settings { padding: 16px; }
.learn-section { margin-bottom: 20px; padding: 16px; background: var(--bg-color); border-radius: 8px; border: 1px solid var(--border-color); }
.learn-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.learn-header h4 { margin: 0; font-size: 14px; }
.learn-btn { padding: 6px 14px; background: var(--primary-color); color: white; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; }
.learn-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.progress-bar { height: 6px; background: var(--border-color); border-radius: 3px; overflow: hidden; margin-bottom: 8px; }
.progress-fill { height: 100%; background: var(--primary-color); border-radius: 3px; transition: width 0.3s; }
.progress-text { font-size: 12px; color: var(--text-secondary); }
.result-label { font-size: 12px; color: var(--text-secondary); margin-bottom: 6px; }
.result-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.clear-btn { padding: 4px 10px; background: transparent; border: 1px solid var(--border-color); color: var(--text-secondary); border-radius: 4px; cursor: pointer; font-size: 12px; }
.clear-btn:hover { border-color: #F44336; color: #F44336; }
.clear-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.result-content { font-size: 13px; color: var(--text-primary); line-height: 1.6; padding: 10px; background: var(--card-bg); border-radius: 6px; border: 1px solid var(--border-color); white-space: pre-wrap; }
.learn-error { color: #F44336; font-size: 13px; display: flex; align-items: center; gap: 8px; }
.retry-btn { padding: 4px 10px; background: transparent; border: 1px solid #F44336; color: #F44336; border-radius: 4px; cursor: pointer; font-size: 12px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-textarea, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; font-family: inherit; resize: vertical; }
.form-textarea:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.preview-area { display: flex; gap: 8px; }
.preview-area .form-input { flex: 1; }
.preview-btn { padding: 8px 16px; background: var(--primary-color); color: white; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; white-space: nowrap; }
.preview-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.preview-result { margin-top: 12px; }
</style>
