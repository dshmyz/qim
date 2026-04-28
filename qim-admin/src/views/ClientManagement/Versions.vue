<template>
  <div class="client-versions-page">
    <el-row :gutter="20">
      <!-- 左侧：版本列表 -->
      <el-col :span="16">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>客户端版本管理</span>
              <el-button type="primary" @click="handleCreate">发布新版本</el-button>
            </div>
          </template>
          <VersionTable
            :versions="clientStore.versions"
            :loading="clientStore.loading"
            @edit="handleEdit"
            @delete="handleDelete"
          />
        </el-card>
      </el-col>

      <!-- 右侧：版本分布饼图 -->
      <el-col :span="8">
        <VersionDistributionChart
          :distribution="clientStore.distribution"
          :loading="clientStore.loading"
          @refresh="handleLoadDistribution"
        />
      </el-col>
    </el-row>

    <!-- 创建/编辑版本对话框 -->
    <VersionFormDialog
      v-model:visible="dialogVisible"
      :is-edit="isEdit"
      :version-data="currentVersion"
      :submit-loading="submitLoading"
      @confirm="handleSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useClientStore } from '@/stores/client'
import type { ClientVersion, CreateVersionParams, UpdateVersionParams } from '@/types/client'
import VersionTable from './components/VersionTable.vue'
import VersionDistributionChart from './components/VersionDistributionChart.vue'
import VersionFormDialog from './components/VersionFormDialog.vue'

const clientStore = useClientStore()

// Dialog state
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const currentVersion = reactive<Partial<ClientVersion>>({})

// Load data
onMounted(async () => {
  await Promise.all([
    clientStore.loadVersions(),
    clientStore.loadDistribution(),
  ])
})

function handleCreate() {
  isEdit.value = false
  Object.assign(currentVersion, {
    releaseDate: new Date().toISOString().split('T')[0],
    platform: 'windows',
    forceUpdate: false,
    rolloutPercentage: 100,
  })
  dialogVisible.value = true
}

function handleEdit(row: ClientVersion) {
  isEdit.value = true
  Object.assign(currentVersion, { ...row })
  dialogVisible.value = true
}

async function handleSubmit(data: Record<string, unknown>) {
  submitLoading.value = true
  try {
    if (isEdit.value) {
      const id = currentVersion.id
      if (!id) {
        ElMessage.error('缺少版本 ID')
        return
      }
      await clientStore.editVersion(id, data as unknown as UpdateVersionParams)
      ElMessage.success('更新成功')
    } else {
      await clientStore.addVersion(data as unknown as CreateVersionParams)
      ElMessage.success('发布成功')
    }
    dialogVisible.value = false
    Object.assign(currentVersion, {})
    await clientStore.loadVersions()
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '操作失败'
    ElMessage.error(message)
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await ElMessageBox.confirm('确定删除该版本吗？此操作不可恢复。', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await clientStore.removeVersion(id)
    ElMessage.success('删除成功')
    await clientStore.loadVersions()
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const message = error instanceof Error ? error.message : '删除失败'
      ElMessage.error(message)
    }
  }
}

async function handleLoadDistribution() {
  try {
    await clientStore.loadDistribution()
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载失败'
    ElMessage.error(message)
  }
}
</script>

<style scoped>
.client-versions-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6, 20px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
