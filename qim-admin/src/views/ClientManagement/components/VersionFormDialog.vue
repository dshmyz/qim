<template>
  <el-dialog
    :model-value="visible"
    :title="isEdit ? '编辑版本' : '发布新版本'"
    width="600px"
    @update:model-value="$emit('update:visible', $event)"
    @closed="handleReset"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
    >
      <el-form-item label="版本号" prop="version">
        <el-input v-model="form.version" :disabled="isEdit" placeholder="例如：2.1.0" />
      </el-form-item>
      <el-form-item label="平台" prop="platform">
        <el-radio-group v-model="form.platform" :disabled="isEdit">
          <el-radio label="windows">Windows</el-radio>
          <el-radio label="macos">macOS</el-radio>
          <el-radio label="linux">Linux</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="发布日期" prop="releaseDate">
        <el-date-picker
          v-model="form.releaseDate"
          type="date"
          placeholder="选择日期"
          value-format="YYYY-MM-DD"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="更新说明" prop="updateNotes">
        <el-input
          v-model="form.updateNotes"
          type="textarea"
          :rows="4"
          placeholder="请输入版本更新说明"
        />
      </el-form-item>
      <el-form-item label="下载链接" prop="downloadUrl">
        <div class="download-url-input">
          <el-input v-model="form.downloadUrl" placeholder="请输入安装包下载链接或上传文件" />
          <el-upload
            ref="uploadRef"
            :show-file-list="false"
            :before-upload="beforeUpload"
            :http-request="handleUpload"
            accept=".exe,.dmg,.AppImage,.zip,.tar.gz"
          >
            <el-button type="primary" :loading="uploading">
              {{ uploading ? '上传中' : '上传' }}
            </el-button>
          </el-upload>
        </div>
        <div v-if="uploadProgress > 0 && uploadProgress < 100" class="upload-progress">
          <el-progress :percentage="uploadProgress" :stroke-width="6" />
        </div>
      </el-form-item>
      <el-form-item label="灰度发布" prop="rolloutPercentage">
        <el-input-number
          v-model="form.rolloutPercentage"
          :min="0"
          :max="100"
          :step="10"
          controls-position="right"
          style="width: 100%"
        />
        <div class="form-item-tip">灰度百分比，0 表示关闭灰度，100 表示全量发布</div>
      </el-form-item>
      <el-form-item label="强制更新">
        <el-switch v-model="form.forceUpdate" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:visible', false)">取消</el-button>
      <el-button type="primary" :loading="submitLoading" @click="handleConfirm">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import type { ClientVersion } from '@/types/client'
import { request } from '@/utils/request'

interface Props {
  visible: boolean
  isEdit: boolean
  versionData?: Partial<ClientVersion>
  submitLoading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  isEdit: false,
  versionData: () => ({}),
  submitLoading: false,
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [data: Record<string, unknown>]
}>()

const formRef = ref<FormInstance>()
const uploadRef = ref()
const uploading = ref(false)
const uploadProgress = ref(0)

const form = reactive({
  id: 0,
  version: '',
  platform: 'windows' as 'windows' | 'macos' | 'linux',
  releaseDate: '',
  updateNotes: '',
  forceUpdate: false,
  rolloutPercentage: 100,
  downloadUrl: '',
})

const rules: FormRules = {
  version: [
    { required: true, message: '请输入版本号', trigger: 'blur' },
    { pattern: /^\d+\.\d+\.\d+$/, message: '版本号格式不正确（如：2.1.0）', trigger: 'blur' },
  ],
  platform: [{ required: true, message: '请选择平台', trigger: 'change' }],
  releaseDate: [{ required: true, message: '请选择发布日期', trigger: 'change' }],
  updateNotes: [{ required: true, message: '请输入更新说明', trigger: 'blur' }],
  downloadUrl: [
    { required: true, message: '请输入下载链接', trigger: 'blur' },
    { type: 'url', message: '请输入有效的 URL', trigger: 'blur' },
  ],
  rolloutPercentage: [{ required: true, message: '请设置灰度百分比', trigger: 'change' }],
}

watch(() => props.versionData, (newData) => {
  if (newData) {
    form.id = newData.id || 0
    form.version = newData.version || ''
    form.platform = newData.platform || 'windows'
    form.releaseDate = newData.releaseDate || ''
    form.updateNotes = newData.updateNotes || ''
    form.forceUpdate = newData.forceUpdate ?? false
    form.rolloutPercentage = newData.rolloutPercentage ?? 100
    form.downloadUrl = newData.downloadUrl || ''
  }
}, { deep: true, immediate: true })

function handleReset() {
  formRef.value?.resetFields()
  form.id = 0
  form.version = ''
  form.platform = 'windows'
  form.releaseDate = ''
  form.updateNotes = ''
  form.forceUpdate = false
  form.rolloutPercentage = 100
  form.downloadUrl = ''
  uploading.value = false
  uploadProgress.value = 0
}

function beforeUpload(file: File) {
  const maxSize = 500 * 1024 * 1024 // 500MB
  if (file.size > maxSize) {
    ElMessage.error('文件大小不能超过 500MB')
    return false
  }
  return true
}

async function handleUpload(options: { file: File }) {
  uploading.value = true
  uploadProgress.value = 0

  try {
    const formData = new FormData()
    formData.append('file', options.file)
    formData.append('source', 'version')

    const response = await request({
      url: '/v1/upload',
      method: 'post',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total) {
          uploadProgress.value = Math.round((progressEvent.loaded / progressEvent.total) * 100)
        }
      },
    })

    const res = response.data as { code: number; data: { id: number; url: string; name: string } }
    if (res.code === 0 && res.data.url) {
      // 使用公开下载接口（无需认证）
      const downloadUrl = `${window.location.origin}/api/v1/public/files/${res.data.id}/download`
      form.downloadUrl = downloadUrl
      ElMessage.success('上传成功，下载链接已自动填入')
    } else {
      ElMessage.error('上传失败')
    }
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '上传失败'
    ElMessage.error(message)
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

async function handleConfirm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    emit('confirm', { ...form })
  })
}
</script>

<style scoped>
.form-item-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.download-url-input {
  display: flex;
  gap: 8px;
  width: 100%;
}

.download-url-input .el-input {
  flex: 1;
}

.upload-progress {
  margin-top: 8px;
}
</style>
