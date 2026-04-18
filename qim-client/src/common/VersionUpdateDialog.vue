<template>
  <el-dialog
    v-model="dialogVisible"
    title="版本更新提示"
    width="500px"
    :close-on-click-modal="false"
    :close-on-press-escape="!forceUpdate"
  >
    <div class="version-update-dialog">
      <div class="version-info">
        <p><strong>当前版本：</strong>{{ currentVersion }}</p>
        <p><strong>最新版本：</strong>{{ latestVersion }}</p>
      </div>
      
      <div class="release-notes">
        <h4>更新内容：</h4>
        <pre>{{ releaseNotes }}</pre>
      </div>
    </div>
    
    <template #footer>
      <div class="dialog-footer">
        <el-button v-if="!forceUpdate" @click="dialogVisible = false">
          稍后再说
        </el-button>
        <el-button type="primary" @click="handleUpdate">
          {{ forceUpdate ? '立即更新' : '前往更新' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, defineProps, defineEmits } from 'vue'
import { openUpdateLink } from '../utils/version'

const props = defineProps<{
  visible: boolean
  currentVersion: string
  latestVersion: string
  releaseNotes: string
  updateUrl: string
  forceUpdate: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'update'): void
}>()

const dialogVisible = ref(props.visible)

// 监听 visible 变化
const updateDialogVisible = (newVal: boolean) => {
  dialogVisible.value = newVal
}

// 处理更新
const handleUpdate = () => {
  openUpdateLink(props.updateUrl)
  emit('update')
  if (!props.forceUpdate) {
    dialogVisible.value = false
  }
}

// 监听 dialogVisible 变化
const watchDialogVisible = (newVal: boolean) => {
  if (!newVal) {
    emit('close')
  }
}
</script>

<style scoped>
.version-update-dialog {
  padding: 20px 0;
}

.version-info {
  margin-bottom: 20px;
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.version-info p {
  margin: 5px 0;
  font-size: 14px;
}

.release-notes {
  margin-top: 20px;
}

.release-notes h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  font-weight: bold;
}

.release-notes pre {
  margin: 0;
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  max-height: 200px;
  overflow-y: auto;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
