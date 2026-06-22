<template>
  <div class="message-search">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>消息搜索</span>
        </div>
      </template>

      <MessageSearchForm :users="users" @search="handleSearch" />

      <el-table
        v-loading="messageStore.loading"
        :data="messageStore.messages"
        border
        stripe
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="senderName" label="发送者" width="120" />
        <el-table-column prop="receiverName" label="接收者" width="120" />
        <el-table-column prop="groupName" label="群组" width="120" />
        <el-table-column prop="channelName" label="频道" width="120" />
        <el-table-column prop="messageType" label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="getMessageTypeTag(row.messageType)">
              {{ getMessageTypeLabel(row.messageType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="内容" min-width="200" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleViewDetail(row)">
              详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="messageStore.total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handlePageChange"
        @current-change="handlePageChange"
      />
    </el-card>

    <el-dialog v-model="detailVisible" title="消息详情" width="600px">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="消息ID">
          {{ currentMessage?.id }}
        </el-descriptions-item>
        <el-descriptions-item label="消息类型">
          {{ getMessageTypeLabel(currentMessage?.messageType) }}
        </el-descriptions-item>
        <el-descriptions-item label="发送者">
          {{ currentMessage?.senderName }}
        </el-descriptions-item>
        <el-descriptions-item label="接收者">
          {{ currentMessage?.receiverName || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="群组">
          {{ currentMessage?.groupName || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="频道">
          {{ currentMessage?.channelName || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="发送时间" :span="2">
          {{ formatTime(currentMessage?.createdAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="消息内容" :span="2">
          {{ currentMessage?.content }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useMessageStore } from '@/stores/message'
import MessageSearchForm from '@/components/search/MessageSearchForm.vue'
import { getUsers } from '@/api/users'
import type { Message, MessageSearchParams } from '@/types/message'

const messageStore = useMessageStore()
const users = ref<Array<{ id: number; name: string }>>([])
const detailVisible = ref(false)
const currentMessage = ref<Message | null>(null)

const pagination = reactive({
  page: 1,
  pageSize: 20,
})

const searchParams = ref<Partial<MessageSearchParams>>({})

onMounted(() => {
  loadUsers()
  handleSearch({})
})

async function loadUsers() {
  try {
    const { data } = await getUsers({ page: 1, pageSize: 100 })
    users.value = (data.data.list ?? []).map((u) => ({
      id: u.id,
      name: u.nickname || u.username,
    }))
  } catch {
    users.value = []
  }
}

async function handleSearch(params: Partial<MessageSearchParams>) {
  searchParams.value = params
  pagination.page = 1
  await loadData()
}

async function handlePageChange() {
  await loadData()
}

async function loadData() {
  const params: MessageSearchParams = {
    ...searchParams.value,
    page: pagination.page,
    pageSize: pagination.pageSize,
  }
  await messageStore.search(params)
}

function handleViewDetail(message: Message) {
  currentMessage.value = message
  detailVisible.value = true
}

function getMessageTypeTag(type: string) {
  const map: Record<string, string> = {
    text: '',
    image: 'success',
    file: 'warning',
    audio: 'info',
    video: 'danger',
  }
  return map[type] || ''
}

function getMessageTypeLabel(type?: string) {
  const map: Record<string, string> = {
    text: '文本',
    image: '图片',
    file: '文件',
    audio: '音频',
    video: '视频',
  }
  return map[type || ''] || type
}

function formatTime(time?: string) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.message-search {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.el-pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
</style>
