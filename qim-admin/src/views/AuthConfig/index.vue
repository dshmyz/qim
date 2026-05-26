<template>
  <div class="auth-config">
    <div class="header">
      <h2>认证配置</h2>
      <el-button type="primary" @click="showCreateDialog">新建认证提供者</el-button>
    </div>

    <el-alert
      title="认证配置说明"
      type="info"
      :closable="false"
      style="margin-bottom: 20px"
    >
      <template #default>
        <p>配置外部认证方式，支持LDAP、OAuth2.0、CAS等协议。用户在桌面应用登录时可选择不同的认证方式。</p>
        <p style="margin-top: 8px">
          <strong>认证类型：</strong>
          <el-tag size="small" type="success">直接认证</el-tag> 用户名密码方式（如LDAP）
          <el-tag size="small" type="warning" style="margin-left: 8px">重定向认证</el-tag> 跳转认证（如OAuth、CAS）
        </p>
      </template>
    </el-alert>

    <el-table :data="providers" v-loading="loading" style="width: 100%">
      <el-table-column prop="name" label="名称" width="120" />
      <el-table-column prop="display_name" label="显示名称" width="150" />
      <el-table-column prop="type" label="类型" width="120">
        <template #default="{ row }">
          <el-tag :type="row.type === 'direct' ? 'success' : 'warning'">
            {{ row.type === 'direct' ? '直接认证' : '重定向认证' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="状态" width="80">
        <template #default="{ row }">
          <el-switch v-model="row.enabled" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column label="操作">
        <template #default="{ row }">
          <el-button size="small" @click="editProvider(row)">编辑</el-button>
          <el-button size="small" type="primary" @click="testProvider(row)">测试</el-button>
          <el-button size="small" type="danger" @click="deleteProvider(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑认证提供者' : '新建认证提供者'" width="800px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" :disabled="isEdit" placeholder="唯一标识，如 ldap、oauth、cas" />
          <div class="form-tip">认证提供者的唯一标识，用于系统内部识别</div>
        </el-form-item>

        <el-form-item label="显示名称" required>
          <el-input v-model="form.display_name" placeholder="用户看到的名称，如 '企业LDAP登录'" />
          <div class="form-tip">在登录页面显示的友好名称</div>
        </el-form-item>

        <el-form-item label="认证类型" required>
          <el-select v-model="form.type" :disabled="isEdit" @change="onTypeChange">
            <el-option label="直接认证（用户名密码）" value="direct" />
            <el-option label="重定向认证（OAuth/CAS）" value="redirect" />
          </el-select>
          <div class="form-tip">
            直接认证：用户输入用户名密码后直接验证（如LDAP）<br/>
            重定向认证：跳转到第三方页面认证（如OAuth、CAS）
          </div>
        </el-form-item>

        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="1" :max="1000" />
          <div class="form-tip">数字越小优先级越高，系统会按优先级依次尝试认证</div>
        </el-form-item>

        <el-form-item label="图标">
          <el-input v-model="form.icon" placeholder="FontAwesome图标类名，如 fas fa-users" />
          <div class="form-tip">可选，登录页面显示的图标</div>
        </el-form-item>

        <el-form-item label="配置模板">
          <el-select v-model="configTemplate" @change="applyTemplate" style="width: 100%">
            <el-option label="选择配置模板..." value="" />
            <el-option label="LDAP 配置" value="ldap" />
            <el-option label="OAuth2.0 配置" value="oauth" />
            <el-option label="CAS 配置" value="cas" />
          </el-select>
        </el-form-item>

        <el-form-item label="配置JSON" required>
          <el-input v-model="form.config" type="textarea" :rows="12" />
          <div class="form-tip">根据认证类型填写相应的配置信息，必须是有效的JSON格式</div>
          <div class="form-tip" style="color: #e6a23c; margin-top: 4px;">
            ⚠️ redirect_url（OAuth）/ service_url（CAS）必须填写 Electron 本地回调地址，与主进程 AUTH_CALLBACK_BASE 一致，默认为 <code>http://localhost:23578/{oauth,cas}/callback</code>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="testDialogVisible" title="测试认证" width="400px">
      <el-form :model="testForm" label-width="100px">
        <el-form-item label="用户名">
          <el-input v-model="testForm.test_username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="testForm.test_password" type="password" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="testDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="runTest">测试</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAuthProviders, createAuthProvider, updateAuthProvider, testAuthProvider, deleteAuthProvider } from '@/api/authProvider'
import type { AuthProvider } from '@/types/auth'

const providers = ref<AuthProvider[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const testDialogVisible = ref(false)
const isEdit = ref(false)
const currentProvider = ref<AuthProvider | null>(null)
const configTemplate = ref('')

interface AuthProviderForm {
  name: string
  display_name: string
  type: 'direct' | 'redirect'
  priority: number
  icon: string
  config: string
  enabled: boolean
}

const form = ref<AuthProviderForm>({
  name: '',
  display_name: '',
  type: 'direct',
  priority: 100,
  icon: '',
  config: '{}',
  enabled: true
})

const testForm = ref({
  test_username: '',
  test_password: ''
})

const configTemplates: Record<string, Omit<AuthProviderForm, 'priority' | 'enabled'>> = {
  ldap: {
    name: 'ldap',
    display_name: '企业LDAP登录',
    type: 'direct',
    icon: 'fas fa-users',
    config: JSON.stringify({
      host: 'ldap.example.com',
      port: 389,
      use_ssl: false,
      bind_dn: 'cn=admin,dc=example,dc=com',
      bind_password: 'admin_password',
      user_search_base: 'ou=users,dc=example,dc=com',
      user_search_filter: '(uid={username})',
      attributes: {
        username: 'uid',
        email: 'mail',
        name: 'cn'
      }
    }, null, 2)
  },
  oauth: {
    name: 'oauth',
    display_name: 'OAuth2.0登录',
    type: 'redirect',
    icon: 'fab fa-google',
    config: JSON.stringify({
      client_id: 'your_client_id',
      client_secret: 'your_client_secret',
      auth_url: 'https://accounts.google.com/o/oauth2/v2/auth',
      token_url: 'https://oauth2.googleapis.com/token',
      user_info_url: 'https://www.googleapis.com/oauth2/v2/userinfo',
      redirect_url: 'http://localhost:23578/oauth/callback',
      scope: 'openid email profile'
    }, null, 2)
  },
  cas: {
    name: 'cas',
    display_name: 'CAS单点登录',
    type: 'redirect',
    icon: 'fas fa-university',
    config: JSON.stringify({
      cas_url: 'https://cas.example.com',
      service_url: 'http://localhost:23578/cas/callback',
      validate_url: 'https://cas.example.com/serviceValidate'
    }, null, 2)
  }
}

const onTypeChange = () => {
  configTemplate.value = ''
}

const applyTemplate = () => {
  if (!configTemplate.value) return
  
  const template = configTemplates[configTemplate.value as keyof typeof configTemplates]
  if (template) {
    form.value.name = template.name
    form.value.display_name = template.display_name
    form.value.type = template.type
    form.value.icon = template.icon
    form.value.config = template.config
    ElMessage.success('已应用配置模板，请根据实际情况修改配置信息')
  }
}

const loadProviders = async () => {
  loading.value = true
  try {
    const res = await getAuthProviders()
    providers.value = res.data.data || []
  } catch (error) {
    ElMessage.error('加载认证提供者失败')
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isEdit.value = false
  configTemplate.value = ''
  form.value = {
    name: '',
    display_name: '',
    type: 'direct',
    priority: 100,
    icon: '',
    config: '{}',
    enabled: true
  }
  dialogVisible.value = true
}

const editProvider = (provider: AuthProvider) => {
  isEdit.value = true
  currentProvider.value = provider
  configTemplate.value = ''
  form.value = {
    name: provider.name,
    display_name: provider.display_name,
    type: provider.type,
    priority: provider.priority,
    icon: provider.icon,
    config: provider.config,
    enabled: provider.enabled
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
    if (isEdit.value && currentProvider.value) {
      await updateAuthProvider(currentProvider.value.id, form.value)
      ElMessage.success('更新成功')
    } else {
      await createAuthProvider(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadProviders()
  } catch (error) {
    ElMessage.error('操作失败')
  }
}

const toggleEnabled = async (provider: AuthProvider) => {
  try {
    await updateAuthProvider(provider.id, { enabled: provider.enabled })
    ElMessage.success('状态更新成功')
  } catch (error) {
    ElMessage.error('状态更新失败')
    provider.enabled = !provider.enabled
  }
}

const testProvider = (provider: AuthProvider) => {
  currentProvider.value = provider
  testForm.value = {
    test_username: '',
    test_password: ''
  }
  testDialogVisible.value = true
}

const runTest = async () => {
  if (!currentProvider.value) return
  
  try {
    await testAuthProvider(currentProvider.value.id, testForm.value)
    ElMessage.success('测试成功')
    testDialogVisible.value = false
  } catch (error) {
    ElMessage.error('测试失败')
  }
}

const deleteProvider = async (provider: AuthProvider) => {
  try {
    await ElMessageBox.confirm('确定要删除该认证提供者吗？', '提示', {
      type: 'warning'
    })
    await deleteAuthProvider(provider.id)
    ElMessage.success('删除成功')
    loadProviders()
  } catch (error) {
    // 用户取消
  }
}

onMounted(() => {
  loadProviders()
})
</script>

<style scoped>
.auth-config {
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
