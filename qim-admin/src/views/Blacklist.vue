<template>
  <div class="blacklist-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>黑名单管理</h3>
      </div>

      <!-- 黑名单列表 -->
      <el-table :data="blacklist" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="userId" label="用户ID" width="100" />
        <el-table-column prop="username" label="用户名" min-width="150" />
        <el-table-column prop="reason" label="封禁原因" min-width="200" show-overflow-tooltip />
        <el-table-column prop="operatorId" label="操作人ID" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'danger' : 'info'">
              {{ row.status === 'active' ? '封禁中' : '已移除' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="封禁时间" width="180" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'active'"
              size="small"
              type="success"
              @click="handleRemove(row.id)"
            >
              移出黑名单
            </el-button>
            <span v-else class="text-muted">已移出</span>
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
          @size-change="fetchBlacklist"
          @current-change="fetchBlacklist"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { BlacklistEntry } from '@/types'
import { getBlacklist, removeBlacklistEntry } from '@/api/blacklist'

// 分页
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const blacklist = ref<BlacklistEntry[]>([])
const loading = ref(false)

// 获取黑名单列表
const fetchBlacklist = async () => {
  loading.value = true
  try {
    const { data } = await getBlacklist({
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    blacklist.value = data.data.list
    pagination.total = data.data.total
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 移出黑名单
const handleRemove = async (id: number) => {
  try {
    await ElMessageBox.confirm('确定将该用户移出黑名单吗？', '提示', { type: 'warning' })
    await removeBlacklistEntry(id)
    ElMessage.success('移出成功')
    fetchBlacklist()
  } catch (error) {
    if (error !== 'cancel') {
      // 错误已在请求拦截器中处理
    }
  }
}

onMounted(fetchBlacklist)
</script>

<style scoped>
.blacklist-page {
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

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
