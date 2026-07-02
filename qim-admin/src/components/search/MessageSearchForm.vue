<template>
  <el-form :model="form" inline class="search-form">
    <el-form-item label="关键词">
      <el-input
        v-model="form.keyword"
        placeholder="搜索消息内容"
        clearable
        @keyup.enter="handleSearch"
      />
    </el-form-item>

    <el-form-item label="发送者">
      <el-select
        v-model="form.senderId"
        placeholder="选择发送者"
        clearable
        filterable
      >
        <el-option
          v-for="user in users"
          :key="user.id"
          :label="user.name"
          :value="user.id"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="消息类型">
      <el-select v-model="form.messageType" placeholder="选择类型" clearable>
        <el-option label="文本" value="text" />
        <el-option label="图片" value="image" />
        <el-option label="文件" value="file" />
        <el-option label="音频" value="audio" />
        <el-option label="视频" value="video" />
      </el-select>
    </el-form-item>

    <el-form-item label="会话类型">
      <el-select v-model="form.conversationType" placeholder="选择类型" clearable>
        <el-option label="单聊" value="single" />
        <el-option label="群聊" value="group" />
        <el-option label="讨论组" value="discussion" />
        <el-option label="机器人" value="bot" />
      </el-select>
    </el-form-item>

    <el-form-item label="时间范围">
      <el-date-picker
        v-model="form.timeRange"
        type="datetimerange"
        range-separator="至"
        start-placeholder="开始时间"
        end-placeholder="结束时间"
      />
    </el-form-item>

    <el-form-item>
      <el-button type="primary" @click="handleSearch">搜索</el-button>
      <el-button @click="handleReset">重置</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import type { MessageSearchParams } from '@/types/message'

interface Props {
  users: Array<{ id: number; name: string }>
}

interface Emits {
  (e: 'search', params: Partial<MessageSearchParams>): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const form = reactive({
  keyword: '',
  senderId: undefined as number | undefined,
  messageType: '',
  conversationType: '',
  timeRange: [] as Date[],
})

function handleSearch() {
  const params: Partial<MessageSearchParams> = {
    keyword: form.keyword || undefined,
    senderId: form.senderId,
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
  form.messageType = ''
  form.conversationType = ''
  form.timeRange = []
  emit('search', {})
}
</script>

<style scoped>
.search-form {
  margin-bottom: 20px;
}
</style>
