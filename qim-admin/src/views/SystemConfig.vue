<template>
  <div class="system-config-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>系统配置</h3>
      </div>

      <el-form
        ref="configFormRef"
        :model="configForm"
        label-width="140px"
        v-loading="loading"
        class="config-form"
      >
        <el-divider content-position="left">消息设置</el-divider>

        <el-form-item label="消息撤回时限">
          <div class="form-item-with-desc">
            <el-input-number
              v-model="configForm.messageRecallTime"
              :min="0"
              :max="7200"
              :step="30"
            />
            <span class="desc">（秒，0 表示不允许撤回）</span>
          </div>
        </el-form-item>

        <el-form-item label="已读/未读显示">
          <el-switch
            v-model="configForm.enableReadReceipt"
            active-text="开启"
            inactive-text="关闭"
          />
          <span class="desc" style="margin-left: 8px">（关闭后用户不可见已读状态，后台仍记录）</span>
        </el-form-item>

        <el-divider content-position="left">AI 设置</el-divider>

        <el-form-item label="AI 功能总开关">
          <el-switch
            v-model="configForm.enableAI"
            active-text="开启"
            inactive-text="关闭"
            active-color="#67c23a"
            inactive-color="#f56c6c"
          />
          <span class="desc" style="margin-left: 8px">（关闭后所有 AI 功能不可用：智能回复、AI助手、分身等）</span>
        </el-form-item>

        <el-divider content-position="left">文件设置</el-divider>

        <el-form-item label="最大文件大小">
          <div class="form-item-with-desc">
            <el-input-number
              v-model="configForm.maxFileSize"
              :min="1"
              :max="1024"
              :step="10"
            />
            <span class="desc">（MB）</span>
          </div>
        </el-form-item>

        <el-form-item label="图片压缩质量">
          <div class="form-item-with-desc">
            <el-slider
              v-model="configForm.imageQuality"
              :min="50"
              :max="100"
              :step="5"
              :marks="{ 50: '50%', 75: '75%', 100: '100%' }"
              show-stops
              style="width: 300px"
            />
          </div>
        </el-form-item>

        <el-divider content-position="left">安全与限流</el-divider>

        <el-form-item label="全局请求频率">
          <div class="form-item-with-desc">
            <el-input-number
              v-model="configForm.rateLimitGlobalRate"
              :min="10"
              :max="10000"
              :step="50"
            />
            <span class="desc">（次/窗口期）</span>
          </div>
        </el-form-item>

        <el-form-item label="全局窗口时长">
          <div class="form-item-with-desc">
            <el-input-number
              v-model="configForm.rateLimitGlobalWindow"
              :min="10"
              :max="3600"
              :step="10"
            />
            <span class="desc">（秒）</span>
          </div>
        </el-form-item>

        <el-form-item label="登录最大尝试次数">
          <div class="form-item-with-desc">
            <el-input-number
              v-model="configForm.rateLimitLoginMaxAttempts"
              :min="1"
              :max="100"
              :step="1"
            />
            <span class="desc">（次/窗口期，超出后封禁）</span>
          </div>
        </el-form-item>

        <el-form-item label="登录窗口时长">
          <div class="form-item-with-desc">
            <el-input-number
              v-model="configForm.rateLimitLoginWindow"
              :min="10"
              :max="3600"
              :step="10"
            />
            <span class="desc">（秒）</span>
          </div>
        </el-form-item>

        <el-form-item label="登录封禁时长">
          <div class="form-item-with-desc">
            <el-input-number
              v-model="configForm.rateLimitLoginBan"
              :min="60"
              :max="86400"
              :step="60"
            />
            <span class="desc">（秒）</span>
          </div>
        </el-form-item>

        <el-divider content-position="left">功能开关</el-divider>

        <el-form-item label="开放注册">
          <el-switch
            v-model="configForm.enableRegistration"
            active-text="开启"
            inactive-text="关闭"
          />
        </el-form-item>

        <el-form-item label="双因素认证">
          <el-switch
            v-model="configForm.enable2FA"
            active-text="开启"
            inactive-text="关闭"
          />
        </el-form-item>

        <el-form-item label="文件上传">
          <el-switch
            v-model="configForm.enableFileUpload"
            active-text="开启"
            inactive-text="关闭"
          />
        </el-form-item>

        <el-form-item label="允许的文件类型">
          <el-select
            v-model="configForm.allowedFileTypes"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="选择或输入文件扩展名"
            style="width: 100%"
          >
            <el-option label=".jpg" value=".jpg" />
            <el-option label=".jpeg" value=".jpeg" />
            <el-option label=".png" value=".png" />
            <el-option label=".gif" value=".gif" />
            <el-option label=".bmp" value=".bmp" />
            <el-option label=".webp" value=".webp" />
            <el-option label=".pdf" value=".pdf" />
            <el-option label=".doc" value=".doc" />
            <el-option label=".docx" value=".docx" />
            <el-option label=".xls" value=".xls" />
            <el-option label=".xlsx" value=".xlsx" />
            <el-option label=".ppt" value=".ppt" />
            <el-option label=".pptx" value=".pptx" />
            <el-option label=".txt" value=".txt" />
            <el-option label=".md" value=".md" />
            <el-option label=".csv" value=".csv" />
            <el-option label=".zip" value=".zip" />
            <el-option label=".rar" value=".rar" />
            <el-option label=".7z" value=".7z" />
            <el-option label=".mp3" value=".mp3" />
            <el-option label=".wav" value=".wav" />
            <el-option label=".mp4" value=".mp4" />
            <el-option label=".avi" value=".avi" />
            <el-option label=".mov" value=".mov" />
          </el-select>
          <span class="desc" style="margin-left: 8px">（可多选或输入自定义扩展名）</span>
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            :loading="submitting"
            @click="handleSubmit"
          >
            保存配置
          </el-button>
          <el-button @click="fetchConfig">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { SystemConfig } from '@/types'
import { getSystemConfig, updateSystemConfig } from '@/api/systemConfig'

const loading = ref(false)
const submitting = ref(false)
const configFormRef = ref<FormInstance>()

const configForm = reactive<SystemConfig>({
  messageRecallTime: 120,
  maxFileSize: 100,
  imageQuality: 80,
  enableRegistration: true,
  enable2FA: false,
  enableFileUpload: true,
  enableAI: true,
  enableReadReceipt: true,
  allowedFileTypes: [],
  rateLimitGlobalRate: 500,
  rateLimitGlobalWindow: 60,
  rateLimitLoginMaxAttempts: 5,
  rateLimitLoginWindow: 60,
  rateLimitLoginBan: 900,
})

const fetchConfig = async () => {
  loading.value = true
  try {
    const { data } = await getSystemConfig()
    Object.assign(configForm, data.data)
    if (typeof configForm.allowedFileTypes === 'string') {
      try {
        configForm.allowedFileTypes = JSON.parse(configForm.allowedFileTypes)
      } catch {
        configForm.allowedFileTypes = []
      }
    }
  } catch {
  } finally {
    loading.value = false
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    await updateSystemConfig({
      ...configForm,
      allowedFileTypes: JSON.stringify(configForm.allowedFileTypes),
    })
    ElMessage.success('配置保存成功，客户端将即时感知变更')
  } catch {
  } finally {
    submitting.value = false
  }
}

onMounted(fetchConfig)
</script>

<style scoped>
.system-config-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-5);
  padding-bottom: var(--space-4);
  border-bottom: 2px solid var(--color-border-light);
}

.page-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.config-form {
  max-width: 600px;
}

.form-item-with-desc {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.desc {
  color: var(--color-text-muted);
  font-size: 12px;
  white-space: nowrap;
}
</style>