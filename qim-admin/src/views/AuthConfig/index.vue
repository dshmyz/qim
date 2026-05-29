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
          <el-input v-model="form.name" :disabled="isEdit" placeholder="唯一标识，如 company_ldap、google_oauth" />
          <div class="form-tip">认证提供者的唯一标识，用于系统内部识别</div>
        </el-form-item>

        <el-form-item label="协议类型" required>
          <el-select v-model="form.protocol" :disabled="isEdit" @change="onProtocolChange">
            <el-option label="LDAP" value="ldap" />
            <el-option label="OAuth2.0" value="oauth" />
            <el-option label="CAS" value="cas" />
          </el-select>
          <div class="form-tip">选择认证协议类型，决定系统如何处理认证流程</div>
        </el-form-item>

        <el-form-item label="显示名称" required>
          <el-input v-model="form.display_name" placeholder="用户看到的名称，如 '企业LDAP登录'" />
          <div class="form-tip">在登录页面显示的友好名称</div>
        </el-form-item>

        <el-form-item label="认证类型">
          <el-input :value="form.type === 'direct' ? '直接认证（用户名密码）' : '重定向认证（OAuth/CAS）'" disabled />
          <div class="form-tip">根据协议类型自动设置，LDAP 为直接认证，OAuth/CAS 为重定向认证</div>
        </el-form-item>

        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="1" :max="1000" />
          <div class="form-tip">数字越小优先级越高，系统会按优先级依次尝试认证</div>
        </el-form-item>

        <el-form-item label="图标">
          <el-input v-model="form.icon" placeholder="FontAwesome图标类名，如 fas fa-users" />
          <div class="form-tip">可选，登录页面显示的图标</div>
        </el-form-item>

        <el-form-item label="配置JSON" required>
          <el-input v-model="form.config" type="textarea" :rows="12" />
          <div class="form-tip">根据认证类型填写相应的配置信息，必须是有效的JSON格式</div>
          <div class="form-tip" v-if="form.type === 'direct'" style="color: #409eff; margin-top: 4px;">
            <strong>LDAP 配置字段说明：</strong><br/>
            <code>server</code> — LDAP 服务器地址，仅填 IP 或域名，不要加 <code>ldap://</code> 前缀<br/>
            <code>port</code> — 端口号，普通连接默认 389，TLS/SSL 连接默认 636<br/>
            <code>use_tls</code> — 是否启用 TLS 加密连接，启用时端口应改为 636<br/>
            <code>bind_dn</code> — 管理员绑定 DN，如 <code>cn=admin,dc=example,dc=com</code><br/>
            <code>bind_password</code> — 管理员绑定密码<br/>
            <code>base_dn</code> — 用户搜索基准 DN，如 <code>ou=users,dc=example,dc=com</code><br/>
            <code>filter</code> — 用户搜索过滤器，<code>%s</code> 会被替换为用户名，默认 <code>(uid=%s)</code><br/>
            <code>attribute_mapping</code> — 属性映射，将 LDAP 属性映射为系统标准字段，key 为标准字段名，value 为 LDAP 属性名
          </div>
          <div class="form-tip" v-if="form.type === 'redirect' && form.name === 'oauth'" style="color: #409eff; margin-top: 4px;">
            <strong>OAuth2.0 配置字段说明：</strong><br/>
            <code>client_id</code> — 客户端 ID<br/>
            <code>client_secret</code> — 客户端密钥<br/>
            <code>auth_url</code> — 授权地址<br/>
            <code>token_url</code> — 令牌地址<br/>
            <code>user_info_url</code> — 用户信息接口地址<br/>
            <code>redirect_url</code> — 回调地址，必须与 Electron 主进程一致<br/>
            <code>scope</code> — 授权范围<br/>
            <code>attribute_mapping</code> — 属性映射，将 OAuth 返回的字段映射为系统标准字段，key 为标准字段名，value 为 OAuth 返回的 JSON 字段名
          </div>
          <div class="form-tip" v-if="form.type === 'redirect' && form.name === 'cas'" style="color: #409eff; margin-top: 4px;">
            <strong>CAS 配置字段说明：</strong><br/>
            <code>server_url</code> — CAS 服务器地址，如 <code>https://cas.example.com</code><br/>
            <code>service_url</code> — 回调地址，必须与 Electron 主进程一致<br/>
            <code>validate_url</code> — 票据验证地址，留空则默认为 <code>{server_url}/serviceValidate</code><br/>
            <code>use_proxy</code> — 是否启用代理模式<br/>
            <code>attribute_mapping</code> — 属性映射，将 CAS 返回的属性映射为系统标准字段，key 为标准字段名，value 为 CAS 属性名
          </div>
          <div class="form-tip" style="color: #67c23a; margin-top: 4px;">
            <strong>标准字段名说明（attribute_mapping 的 key）：</strong><br/>
            <code>username</code> — 用户名（必填）&nbsp;&nbsp;<code>nickname</code> — 昵称&nbsp;&nbsp;<code>email</code> — 邮箱&nbsp;&nbsp;<code>phone</code> — 电话&nbsp;&nbsp;<code>avatar</code> — 头像<br/>
            未配置的字段会使用默认映射，无需全部填写
          </div>
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

interface AuthProviderForm {
  name: string
  protocol: 'ldap' | 'oauth' | 'cas'
  display_name: string
  type: 'direct' | 'redirect'
  priority: number
  icon: string
  config: string
  enabled: boolean
}

const form = ref<AuthProviderForm>({
  name: '',
  protocol: 'ldap',
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
    protocol: 'ldap',
    display_name: '企业LDAP登录',
    type: 'direct',
    icon: 'fas fa-users',
    config: JSON.stringify({
      server: '192.168.1.100',
      port: 389,
      use_tls: false,
      bind_dn: 'cn=admin,dc=example,dc=com',
      bind_password: 'admin_password',
      base_dn: 'ou=users,dc=example,dc=com',
      filter: '(uid=%s)',
      attribute_mapping: {
        username: 'uid',
        nickname: 'cn',
        email: 'mail',
        phone: 'telephonenumber',
        avatar: 'jpegphoto'
      }
    }, null, 2)
  },
  oauth: {
    name: 'oauth',
    protocol: 'oauth',
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
      scope: 'openid email profile',
      attribute_mapping: {
        username: 'login',
        nickname: 'name',
        email: 'email',
        phone: 'phone',
        avatar: 'picture'
      }
    }, null, 2)
  },
  cas: {
    name: 'cas',
    protocol: 'cas',
    display_name: 'CAS单点登录',
    type: 'redirect',
    icon: 'fas fa-university',
    config: JSON.stringify({
      server_url: 'https://cas.example.com',
      service_url: 'http://localhost:23578/cas/callback',
      validate_url: 'https://cas.example.com/serviceValidate',
      use_proxy: false,
      attribute_mapping: {
        username: 'username',
        nickname: 'displayName',
        email: 'mail',
        phone: 'phone',
        avatar: 'avatar'
      }
    }, null, 2)
  }
}

const protocolTypeMap: Record<string, 'direct' | 'redirect'> = {
  ldap: 'direct',
  oauth: 'redirect',
  cas: 'redirect'
}

const onProtocolChange = (protocol: string) => {
  form.value.type = protocolTypeMap[protocol] || 'direct'

  const template = configTemplates[protocol as keyof typeof configTemplates]
  if (template) {
    form.value.name = template.name
    form.value.display_name = template.display_name
    form.value.icon = template.icon
    form.value.config = template.config
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
  form.value = {
    name: '',
    protocol: 'ldap',
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
  form.value = {
    name: provider.name,
    protocol: (provider.protocol || 'ldap') as 'ldap' | 'oauth' | 'cas',
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
