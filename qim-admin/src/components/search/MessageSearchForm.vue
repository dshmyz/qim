<template>
  <el-form :model="form" class="search-form" @submit.prevent>
    <div class="form-row">
      <el-form-item label="关键词">
        <el-input
          v-model="form.keyword"
          placeholder="搜索消息内容"
          clearable
          style="width: 220px"
          @keyup.enter="handleSearch"
        />
      </el-form-item>

      <el-form-item label="发送者">
        <UserRemoteSelect v-model="form.senderId" />
      </el-form-item>

      <el-form-item label="接收者">
        <UserRemoteSelect v-model="form.receiverId" placeholder="仅单聊（对端）" />
      </el-form-item>

      <el-form-item label="消息类型">
        <el-select v-model="form.messageType" placeholder="选择类型" clearable style="width: 150px">
          <el-option
            v-for="t in messageTypeOptions"
            :key="t.value"
            :label="t.label"
            :value="t.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="会话类型">
        <el-select v-model="form.conversationType" placeholder="选择类型" clearable style="width: 130px">
          <el-option label="单聊" value="single" />
          <el-option label="群聊" value="group" />
          <el-option label="讨论组" value="discussion" />
          <el-option label="机器人" value="bot" />
        </el-select>
      </el-form-item>
    </div>

    <div class="form-row">
      <el-form-item label="时间范围">
        <el-date-picker
          v-model="form.timeRange"
          type="datetimerange"
          range-separator="至"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          style="width: 380px"
        />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </el-form-item>
    </div>
  </el-form>
</template>

<script lang="ts">
// 消息类型选项：单一来源，结果页标签也引用这里（避免两处漂移）。
// 注意：<script setup> 不允许 ESM export，故放普通 <script> 块。
export const messageTypeOptions: Array<{ value: string; label: string }> = [
  { value: 'text', label: '文本' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'image', label: '图片' },
  { value: 'file', label: '文件' },
  { value: 'audio', label: '音频' },
  { value: 'video', label: '视频' },
  { value: 'card', label: '卡片' },
  { value: 'share', label: '分享' },
  { value: 'news', label: '资讯' },
  { value: 'miniApp', label: '小程序' },
  { value: 'system', label: '系统' },
]
</script>

<script setup lang="ts">
import { reactive, onMounted } from 'vue'
import type { MessageSearchParams } from '@/types/message'
import UserRemoteSelect from './UserRemoteSelect.vue'

// 默认时间范围：最近 7 天（初始与重置都回落到该范围，避免每次手选）
const DEFAULT_DAYS = 7

interface Emits {
  (e: 'search', params: Partial<MessageSearchParams>): void
}

const emit = defineEmits<Emits>()

const form = reactive({
  keyword: '',
  senderId: undefined as number | undefined,
  receiverId: undefined as number | undefined,
  messageType: '',
  conversationType: '',
  timeRange: [] as Date[],
})

function defaultTimeRange(): Date[] {
  const end = new Date()
  const start = new Date(end.getTime() - DEFAULT_DAYS * 24 * 60 * 60 * 1000)
  return [start, end]
}

function handleSearch() {
  const params: Partial<MessageSearchParams> = {
    keyword: form.keyword || undefined,
    senderId: form.senderId,
    receiverId: form.receiverId,
    messageType: form.messageType || undefined,
    conversationType: (form.conversationType as MessageSearchParams['conversationType']) || undefined,
    startTime: form.timeRange[0]?.toISOString(),
    endTime: form.timeRange[1]?.toISOString(),
  }
  emit('search', params)
}

function handleReset() {
  form.keyword = ''
  form.senderId = undefined
  form.receiverId = undefined
  form.messageType = ''
  form.conversationType = ''
  form.timeRange = defaultTimeRange()
  emit('search', {})
}

onMounted(() => {
  form.timeRange = defaultTimeRange()
})
</script>

<style scoped>
.search-form {
  margin-bottom: 8px;
}

.form-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0 8px;
  align-items: flex-start;
}

.form-row + .form-row {
  margin-top: 4px;
}
</style>
