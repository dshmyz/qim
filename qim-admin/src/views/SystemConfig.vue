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
})

// 获取配置
const fetchConfig = async () => {
  loading.value = true
  try {
    const { data } = await getSystemConfig()
    Object.assign(configForm, data.data)
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 保存配置
const handleSubmit = async () => {
  submitting.value = true
  try {
    await updateSystemConfig({ ...configForm })
    ElMessage.success('配置保存成功')
  } catch {
    // 错误已在请求拦截器中处理
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
