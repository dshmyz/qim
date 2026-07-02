<template>
  <div class="conversations-page">
    <el-card shadow="never">
      <!-- 搜索和筛选栏 -->
      <div class="search-bar">
        <el-form :model="filterForm" inline>
          <el-form-item label="类型">
            <el-select v-model="filterForm.type" placeholder="全部类型" clearable style="width: 140px" @change="handleSearch">
              <el-option label="单聊" value="single" />
              <el-option label="群聊" value="group" />
              <el-option label="讨论组" value="discussion" />
              <el-option label="机器人" value="bot" />
            </el-select>
          </el-form-item>
          <el-form-item label="名称">
            <el-input
              v-model="filterForm.keyword"
              placeholder="请输入会话名称"
              clearable
              @keyup.enter="handleSearch"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 会话列表 -->
      <el-table :data="conversations" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="conversationTypeColor(row.type)">
              {{ conversationTypeLabel(row.type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column prop="creatorName" label="创建者" min-width="120" />
        <el-table-column prop="memberCount" label="成员数" width="100" />
        <el-table-column label="置顶" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isPinned" type="warning" size="small">已置顶</el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="lastMessageAt" label="最后消息时间" width="180" />
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="handleViewMembers(row)">查看成员</el-button>
            <el-popconfirm
              title="确定解散该会话吗？"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button size="small" type="danger">解散</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="fetchConversations"
          @current-change="fetchConversations"
        />
      </div>
    </el-card>

    <!-- 查看成员对话框 -->
    <MembersDialog
      ref="membersDialogRef"
      v-model="memberDialogVisible"
      :conversation="currentConversation"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { Conversation } from '@/types'
import { getConversations, deleteConversation, type GetConversationsParams } from '@/api/conversations'
import MembersDialog from './components/MembersDialog.vue'

// 筛选和分页
const filterForm = reactive({
  type: '',
  keyword: '',
})
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const conversations = ref<Conversation[]>([])
const loading = ref(false)

// 成员对话框
const memberDialogVisible = ref(false)
const currentConversation = ref<Conversation | null>(null)
const membersDialogRef = ref<InstanceType<typeof MembersDialog>>()

// 工具函数
const conversationTypeLabel = (type: string): string => {
  const map: Record<string, string> = { single: '单聊', group: '群聊', discussion: '讨论组', bot: '机器人' }
  return map[type] || type
}

const conversationTypeColor = (type: string): 'primary' | 'success' | 'info' | 'warning' => {
  const map: Record<string, 'primary' | 'success' | 'info' | 'warning'> = { single: 'primary', group: 'success', discussion: 'info', bot: 'warning' }
  return map[type] || 'info'
}

// 获取会话列表
const fetchConversations = async () => {
  loading.value = true
  try {
    const params: GetConversationsParams = {
      page: pagination.page,
      pageSize: pagination.pageSize,
    }
    if (filterForm.type) params.type = filterForm.type as 'single' | 'group' | 'discussion' | 'bot'
    if (filterForm.keyword) params.keyword = filterForm.keyword

    const { data } = await getConversations(params)
    conversations.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchConversations()
}

const handleReset = () => {
  filterForm.type = ''
  filterForm.keyword = ''
  handleSearch()
}

// 查看成员
const handleViewMembers = (row: Conversation) => {
  currentConversation.value = row
  memberDialogVisible.value = true
  // 延迟获取成员数据，等待对话框渲染完成
  setTimeout(() => {
    membersDialogRef.value?.fetchMembers()
  }, 100)
}

// 解散会话
const handleDelete = async (id: number) => {
  try {
    await deleteConversation(id)
    ElMessage.success('解散成功')
    fetchConversations()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchConversations)
</script>

<style scoped>
.conversations-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
