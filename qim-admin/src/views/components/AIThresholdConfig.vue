<template>
  <div class="ai-threshold-config">
    <el-form label-width="160px" v-loading="loading" class="threshold-form">
      <el-divider content-position="left">检索与召回阈值</el-divider>
      
      <el-form-item
        v-for="item in thresholdItems"
        :key="item.key"
        :label="item.label"
      >
        <div class="form-item-with-desc">
          <!-- 布尔阈值（0/1）用 switch，避免连续输入 0.95 被 int 截断为 0 静默关闭功能 -->
          <el-switch
            v-if="item.isBoolean"
            v-model="thresholds[item.key]"
            :active-value="1"
            :inactive-value="0"
            active-text="开启"
            inactive-text="关闭"
          />
          <el-input-number
            v-else
            v-model="thresholds[item.key]"
            :min="item.min"
            :max="item.max"
            :step="item.step"
            :precision="item.precision"
          />
          <span class="desc">{{ item.description }}</span>
        </div>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="submitting" @click="handleSave">
          保存阈值
        </el-button>
        <el-button @click="loadThresholds">重置</el-button>
        <span class="save-hint">保存后立即生效，无需重启服务</span>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getAIThresholds, updateAIThresholds, getAIThresholdSchema, type AiThresholdSchema } from '@/api/aiThresholds'

const loading = ref(false)
const submitting = ref(false)
const thresholds = ref<Record<string, number>>({})
const schema = ref<AiThresholdSchema[]>([])

// 阈值项定义：根据 schema 动态生成表单
const thresholdItems = ref<Array<{
  key: string
  label: string
  description: string
  min: number
  max: number
  isBoolean: boolean
  step: number
  precision: number
}>>([])

onMounted(async () => {
  loading.value = true
  try {
    const [schemaRes, valuesRes] = await Promise.all([
      getAIThresholdSchema(),
      getAIThresholds()
    ])
    if (schemaRes.data.code === 0) {
      schema.value = schemaRes.data.data
      thresholdItems.value = schema.value.map(s => ({
        key: s.key,
        label: s.label,
        description: s.description,
        min: s.min,
        max: s.max,
        isBoolean: s.is_bool,
        step: s.max <= 1 ? 0.05 : (s.max <= 20 ? 1 : 5),
        precision: s.max <= 1 ? 2 : 0
      }))
    }
    if (valuesRes.data.code === 0) {
      thresholds.value = valuesRes.data.data
    }
  } catch (e) {
    console.error('加载阈值配置失败', e)
  } finally {
    loading.value = false
  }
})

const loadThresholds = async () => {
  loading.value = true
  try {
    const res = await getAIThresholds()
    if (res.data.code === 0) {
      thresholds.value = res.data.data
    }
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  submitting.value = true
  try {
    await updateAIThresholds(thresholds.value)
    ElMessage.success('阈值已保存，下次请求即生效')
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.ai-threshold-config { padding: 0; }
.threshold-form { max-width: 600px; }
.form-item-with-desc { display: flex; align-items: center; gap: 12px; }
.form-item-with-desc .desc { color: #999; font-size: 12px; white-space: nowrap; }
.save-hint { color: #999; font-size: 12px; margin-left: 8px; }
</style>
