<template>
  <el-dialog
    v-model="visible"
    :title="`会话成员 - ${conversation?.name || ''}`"
    width="600px"
  >
    <el-table :data="members" v-loading="loading" size="small" max-height="400">
      <el-table-column prop="userId" label="用户ID" width="100" />
      <el-table-column label="成员" min-width="150">
        <template #default="{ row }">
          <div class="member-cell">
            <el-avatar :size="28" :src="row.avatar">{{ (row.nickname || row.username).charAt(0) }}</el-avatar>
            <span>{{ row.nickname || row.username }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="角色" width="100">
        <template #default="{ row }">
          <el-tag size="small" v-if="row.role">{{ roleLabel(row.role) }}</el-tag>
          <span v-else class="text-muted">普通成员</span>
        </template>
      </el-table-column>
      <el-table-column prop="joinedAt" label="加入时间" width="180" />
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Conversation, ConversationMember } from '@/types'
import { getGroupMembers } from '@/api/groups'

const props = defineProps<{
  modelValue: boolean
  conversation?: Conversation | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const members = ref<ConversationMember[]>([])
const loading = ref(false)

const roleLabel = (role: string): string => {
  const map: Record<string, string> = { owner: '群主', admin: '管理员', member: '成员' }
  return map[role] || role
}

const fetchMembers = async () => {
  if (!props.conversation) return
  loading.value = true
  try {
    const { data } = await getGroupMembers(props.conversation.id, { page: 1, pageSize: 100 })
    members.value = data.data.list
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

defineExpose({ fetchMembers })
</script>

<style scoped>
.member-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
