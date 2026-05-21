<template>
  <div class="auth-config">
    <div class="header">
      <h2>认证配置</h2>
      <el-button type="primary" @click="showCreateDialog">新建认证提供者</el-button>
    </div>

    <el-table :data="providers" v-loading="loading" style="width: 100%">
      <el-table-column prop="name" label="名称" width="180" />
      <el-table-column prop="display_name" label="显示名称" width="180" />
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="row.type === 'direct' ? 'success' : 'warning'">
            {{ row.type === 'direct' ? '直接认证' : '重定向认证' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="状态" width="100">
        <template #default="{ row }">
          <el-switch v-model="row.enabled" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="100" />
      <el-table-column label="操作">
        <template #default="{ row }">
          <el-button size="small" @click="editProvider(row)">编辑</el-button>
          <el-button size="small" type="primary" @click="testProvider(row)">测试</el-button>
          <el-button size="small" type="danger" @click="deleteProvider(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑认证提供者' : '新建认证提供者'" width="600px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="名称">
          <el-input v-model="form.name" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="form.display_name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" :disabled="isEdit">
            <el-option label="直接认证" value="direct" />
            <el-option label="重定向认证" value="redirect" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="1" :max="1000" />
        </el-form-item>
        <el-form-item label="图标">
          <el-input v-model="form.icon" />
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
import { getAuthProviders, createAuthProvider, updateAuthProvider, testAuthProvider } from '@/api/authProvider'
import type { AuthProvider } from '@/types/auth'

const providers = ref<AuthProvider[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const testDialogVisible = ref(false)
const isEdit = ref(false)
const currentProvider = ref<AuthProvider | null>(null)

const form = ref({
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

const loadProviders = async () => {
  loading.value = true
  try {
    const res = await getAuthProviders()
    providers.value = res.data.data
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
</style>
