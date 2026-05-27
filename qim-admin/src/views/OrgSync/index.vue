<template>
  <div class="org-sync">
    <div class="header">
      <h2>组织架构同步</h2>
      <el-button type="primary" @click="showCreateDialog">新建同步配置</el-button>
    </div>

    <el-table :data="configs" v-loading="loading" style="width: 100%">
      <el-table-column prop="name" label="名称" width="180" />
      <el-table-column prop="sync_type" label="数据源类型" width="120">
        <template #default="{ row }">
          <el-tag :type="row.sync_type === 'ldap' ? 'success' : 'primary'">
            {{ row.sync_type === 'ldap' ? 'LDAP' : row.sync_type === 'api' ? 'API' : row.sync_type }}
          </el-tag>
        </template>
      </el-table-column>
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

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑同步配置' : '新建同步配置'" width="700px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="配置名称，如 企业LDAP同步" />
        </el-form-item>
        <el-form-item label="数据源类型" required>
          <el-select v-model="form.sync_type" @change="onSyncTypeChange">
            <el-option label="LDAP 目录服务" value="ldap" />
            <el-option label="API 接口" value="api" />
          </el-select>
          <div class="form-tip">选择组织架构数据的来源类型</div>
        </el-form-item>
        <el-form-item label="调度时间">
          <el-input v-model="form.schedule" placeholder="Cron 表达式，如 0 2 * * * 表示每天凌晨2点" />
          <div class="form-tip">留空表示仅手动触发同步</div>
        </el-form-item>
        <el-form-item label="配置模板">
          <el-select v-model="configTemplate" @change="applyTemplate" style="width: 100%">
            <el-option label="选择配置模板..." value="" />
            <el-option v-if="form.sync_type === 'ldap'" label="LDAP 配置" value="ldap" />
            <el-option v-if="form.sync_type === 'api'" label="API 配置" value="api" />
          </el-select>
        </el-form-item>
        <el-form-item label="配置JSON" required>
          <el-input v-model="form.config" type="textarea" :rows="12" />
          <div class="form-tip">根据数据源类型填写相应的配置信息，必须是有效的JSON格式</div>
          <div class="form-tip" v-if="form.sync_type === 'ldap'" style="color: #409eff; margin-top: 4px;">
            <strong>LDAP 配置字段说明：</strong><br/>
            <code>server</code> — LDAP 服务器地址，仅填 IP 或域名，不要加 <code>ldap://</code> 前缀<br/>
            <code>port</code> — 端口号，普通连接默认 389，TLS 连接默认 636<br/>
            <code>use_tls</code> — 是否启用 TLS 加密连接，启用时端口应改为 636<br/>
            <code>bind_dn</code> — 管理员绑定 DN，如 <code>cn=admin,dc=example,dc=com</code><br/>
            <code>bind_password</code> — 管理员绑定密码<br/>
            <code>base_dn</code> — 搜索基准 DN，如 <code>dc=example,dc=com</code><br/>
            <code>department_filter</code> — 部门搜索过滤器，默认 <code>(objectClass=organizationalUnit)</code><br/>
            <code>user_filter</code> — 用户搜索过滤器，默认 <code>(objectClass=inetOrgPerson)</code><br/>
            <code>attribute_mapping</code> — 属性映射，如 <code>{"username":"uid","nickname":"cn","email":"mail"}</code>
          </div>
          <div class="form-tip" v-if="form.sync_type === 'api'" style="color: #409eff; margin-top: 4px;">
            <strong>API 配置字段说明：</strong><br/>
            <code>url</code> — 数据接口地址，必填<br/>
            <code>method</code> — 请求方法，默认 GET<br/>
            <code>headers</code> — 自定义请求头，如 <code>{"Authorization":"Bearer xxx"}</code><br/>
            <code>auth_token</code> — 认证令牌，会以 Bearer 方式添加到请求头<br/>
            <code>timeout</code> — 超时时间（秒），默认 30
          </div>
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
const configTemplate = ref('')

const form = ref({
  name: '',
  sync_type: 'ldap',
  schedule: '',
  config: '{}',
  enabled: true
})

const syncConfigTemplates: Record<string, string> = {
  ldap: JSON.stringify({
    server: '192.168.1.100',
    port: 389,
    use_tls: false,
    bind_dn: 'cn=admin,dc=example,dc=com',
    bind_password: 'admin_password',
    base_dn: 'dc=example,dc=com',
    department_filter: '(objectClass=organizationalUnit)',
    user_filter: '(objectClass=inetOrgPerson)',
    attribute_mapping: {
      username: 'uid',
      nickname: 'cn',
      email: 'mail'
    }
  }, null, 2),
  api: JSON.stringify({
    url: 'https://api.example.com/org',
    method: 'GET',
    headers: {},
    auth_token: '',
    timeout: 30
  }, null, 2)
}

const onSyncTypeChange = () => {
  configTemplate.value = ''
}

const applyTemplate = () => {
  if (!configTemplate.value) return
  const template = syncConfigTemplates[configTemplate.value]
  if (template) {
    form.value.config = template
    ElMessage.success('已应用配置模板，请根据实际情况修改配置信息')
  }
}

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
  configTemplate.value = ''
  form.value = {
    name: '',
    sync_type: 'ldap',
    schedule: '',
    config: '{}',
    enabled: true
  }
  dialogVisible.value = true
}

const editConfig = (config: OrgSyncConfig) => {
  isEdit.value = true
  currentConfig.value = config
  configTemplate.value = ''
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
    JSON.parse(form.value.config)
  } catch (error) {
    ElMessage.error('配置JSON格式错误')
    return
  }

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

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}
</style>
