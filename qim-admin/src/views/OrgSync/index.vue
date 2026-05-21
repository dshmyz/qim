<template>
  <div class="org-sync">
    <div class="header">
      <h2>组织架构同步</h2>
      <el-button type="primary" @click="showCreateDialog">新建同步配置</el-button>
    </div>

    <el-table :data="configs" v-loading="loading" style="width: 100%">
      <el-table-column prop="name" label="名称" width="180" />
      <el-table-column prop="sync_type" label="同步类型" width="120" />
      <el-table-column prop="schedule" label="调度时间" width="120" />
      <el-table-column prop="enabled" label="状态" width="100">
        <template #default="{ row }">
          <el-switch v-model="row.enabled" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column prop="last_sync_status" label="最后状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.last_sync_status === 'success' ? 'success' : 'danger'">
            {{ row.last_sync_status || '未同步' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="last_sync_at" label="最后同步时间" width="180" />
      <el-table-column label="操作">
        <template #default="{ row }">
          <el-button size="small" @click="editConfig(row)">编辑</el-button>
          <el-button size="small" type="primary" @click="triggerSync(row)">触发同步</el-button>
          <el-button size="small" @click="viewLogs(row)">查看日志</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑同步配置' : '新建同步配置'" width="600px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="同步类型">
          <el-select v-model="form.sync_type">
            <el-option label="定时同步" value="schedule" />
            <el-option label="手动同步" value="manual" />
            <el-option label="实时同步" value="realtime" />
          </el-select>
        </el-form-item>
        <el-form-item label="调度时间" v-if="form.sync_type === 'schedule'">
          <el-input v-model="form.schedule" placeholder="例如: 0 2 * * *" />
        </el-form-item>
        <el-form-item label="配置">
          <el-input v-model="form.config" type="textarea" :rows="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="logsDialogVisible" title="同步日志" width="800px">
      <el-table :data="logs" v-loading="logsLoading">
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : row.status === 'running' ? 'warning' : 'danger'">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="started_at" label="开始时间" width="180" />
        <el-table-column prop="finished_at" label="结束时间" width="180" />
        <el-table-column prop="stats" label="统计信息" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getOrgSyncConfigs, createOrgSyncConfig, updateOrgSyncConfig, triggerOrgSync, getOrgSyncLogs } from '@/api/orgSync'
import type { OrgSyncConfig, OrgSyncLog } from '@/types/auth'

const configs = ref<OrgSyncConfig[]>([])
const logs = ref<OrgSyncLog[]>([])
const loading = ref(false)
const logsLoading = ref(false)
const dialogVisible = ref(false)
const logsDialogVisible = ref(false)
const isEdit = ref(false)
const currentConfig = ref<OrgSyncConfig | null>(null)

const form = ref({
  name: '',
  sync_type: 'manual',
  schedule: '',
  config: '{}',
  enabled: true
})

const loadConfigs = async () => {
  loading.value = true
  try {
    const res = await getOrgSyncConfigs()
    configs.value = res.data.data
  } catch (error) {
    ElMessage.error('加载同步配置失败')
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isEdit.value = false
  form.value = {
    name: '',
    sync_type: 'manual',
    schedule: '',
    config: '{}',
    enabled: true
  }
  dialogVisible.value = true
}

const editConfig = (config: OrgSyncConfig) => {
  isEdit.value = true
  currentConfig.value = config
  form.value = {
    name: config.name,
    sync_type: config.sync_type,
    schedule: config.schedule,
    config: config.config,
    enabled: config.enabled
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  try {
    if (isEdit.value && currentConfig.value) {
      await updateOrgSyncConfig(currentConfig.value.id, form.value)
      ElMessage.success('更新成功')
    } else {
      await createOrgSyncConfig(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadConfigs()
  } catch (error) {
    ElMessage.error('操作失败')
  }
}

const toggleEnabled = async (config: OrgSyncConfig) => {
  try {
    await updateOrgSyncConfig(config.id, { enabled: config.enabled })
    ElMessage.success('状态更新成功')
  } catch (error) {
    ElMessage.error('状态更新失败')
    config.enabled = !config.enabled
  }
}

const triggerSync = async (config: OrgSyncConfig) => {
  try {
    await triggerOrgSync(config.id)
    ElMessage.success('同步任务已触发')
    loadConfigs()
  } catch (error) {
    ElMessage.error('触发同步失败')
  }
}

const viewLogs = async (config: OrgSyncConfig) => {
  logsLoading.value = true
  logsDialogVisible.value = true
  try {
    const res = await getOrgSyncLogs(config.id)
    logs.value = res.data.data.items
  } catch (error) {
    ElMessage.error('加载日志失败')
  } finally {
    logsLoading.value = false
  }
}

onMounted(() => {
  loadConfigs()
})
</script>

<style scoped>
.org-sync {
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h2 {
  margin: 0;
}
</style>
