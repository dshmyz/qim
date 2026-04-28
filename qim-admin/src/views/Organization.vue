<template>
  <div class="organization-page">
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">组织架构</h1>
        <p class="page-subtitle">管理公司部门结构和员工归属</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="handleCreateDepartment">
        创建部门
      </el-button>
    </div>

    <div class="content-grid">
      <div class="tree-panel">
        <div class="panel-header">
          <h3 class="panel-title">部门列表</h3>
          <span class="panel-count">{{ departmentCount }} 个部门</span>
        </div>
        <div v-loading="treeLoading" class="tree-content">
          <div
            v-for="dept in flatDepartments"
            :key="dept.id"
            class="dept-card"
            :class="{ 'is-active': selectedDepartment?.id === dept.id }"
            :style="{ paddingLeft: dept.level * 20 + 'px' }"
            @click="handleNodeClick(dept)"
          >
            <div class="dept-icon">
              <el-icon><OfficeBuilding /></el-icon>
            </div>
            <div class="dept-info">
              <span class="dept-name">{{ dept.name }}</span>
              <span class="dept-code">{{ dept.code }}</span>
            </div>
            <div class="dept-actions">
              <el-button
                size="small"
                :icon="Plus"
                circle
                @click.stop="handleAddSubDepartment(dept)"
              />
              <el-button
                size="small"
                :icon="Delete"
                circle
                type="danger"
                plain
                @click.stop="handleDeleteDepartment(dept)"
              />
            </div>
          </div>
          <el-empty v-if="!treeLoading && flatDepartments.length === 0" description="暂无部门数据" :image-size="64" />
        </div>
      </div>

      <div class="detail-panel">
        <template v-if="selectedDepartment">
          <div class="detail-card">
            <div class="detail-header">
              <div class="dept-title">
                <h2>{{ selectedDepartment.name }}</h2>
                <el-tag :type="selectedDepartment.status === 'active' ? 'success' : 'info'" size="small">
                  {{ selectedDepartment.status === 'active' ? '启用' : '停用' }}
                </el-tag>
              </div>
              <el-button type="primary" :icon="UserFilled" @click="handleAddEmployee">
                添加员工
              </el-button>
            </div>

            <div class="detail-info">
              <div class="info-item">
                <span class="info-label">部门编码</span>
                <span class="info-value">{{ selectedDepartment.code || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">创建时间</span>
                <span class="info-value">{{ selectedDepartment.createdAt }}</span>
              </div>
              <div class="info-item full-width">
                <span class="info-label">部门描述</span>
                <span class="info-value">{{ selectedDepartment.description || '暂无描述' }}</span>
              </div>
            </div>
          </div>

          <div class="employees-card">
            <div class="card-header">
              <h3>部门员工</h3>
              <span class="employee-count">{{ employees.length }} 人</span>
            </div>
            <div v-loading="employeesLoading" class="employees-grid">
              <div
                v-for="emp in employees"
                :key="emp.id"
                class="employee-item"
              >
                <el-avatar :size="48" :src="emp.avatar" class="employee-avatar">
                  {{ (emp.nickname || emp.username)?.charAt(0)?.toUpperCase() || '?' }}
                </el-avatar>
                <div class="employee-info">
                  <span class="employee-name">{{ emp.nickname || emp.username }}</span>
                  <span class="employee-email">{{ emp.email }}</span>
                </div>
                <el-button
                  size="small"
                  type="danger"
                  text
                  @click="handleRemoveEmployee(emp.id)"
                >
                  移出
                </el-button>
              </div>
              <el-empty v-if="!employeesLoading && employees.length === 0" description="暂无员工" :image-size="48" />
            </div>
          </div>
        </template>
        <el-empty v-else description="请选择左侧部门查看详情" :image-size="120">
          <template #image>
            <el-icon :size="80" color="var(--color-text-muted)"><OfficeBuilding /></el-icon>
          </template>
        </el-empty>
      </div>
    </div>

    <el-dialog
      v-model="departmentDialogVisible"
      :title="isEdit ? '编辑部门' : '创建部门'"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="departmentFormRef"
        :model="departmentForm"
        :rules="departmentRules"
        label-width="80px"
        label-position="top"
      >
        <el-form-item label="部门名称" prop="name">
          <el-input v-model="departmentForm.name" placeholder="请输入部门名称" />
        </el-form-item>
        <el-form-item label="上级部门">
          <el-select v-model="departmentForm.parentId" placeholder="无上级部门（根部门）" clearable style="width: 100%">
            <el-option
              v-for="dept in departmentOptions"
              :key="dept.id"
              :label="dept.name"
              :value="dept.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="部门编码" prop="code">
          <el-input v-model="departmentForm.code" placeholder="请输入部门编码" />
        </el-form-item>
        <el-form-item label="部门描述">
          <el-input v-model="departmentForm.description" type="textarea" :rows="3" placeholder="请输入部门描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="departmentDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleDepartmentSubmit">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="employeeDialogVisible"
      title="添加员工到部门"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form label-position="top">
        <el-form-item label="选择员工">
          <el-select
            v-model="selectedEmployeeId"
            filterable
            remote
            :remote-method="searchEmployees"
            :loading="employeeSearchLoading"
            placeholder="请输入用户名搜索"
            style="width: 100%"
          >
            <el-option
              v-for="emp in employeeSearchResults"
              :key="emp.id"
              :label="emp.nickname || emp.username"
              :value="emp.id"
            >
              <div style="display: flex; align-items: center; gap: 8px;">
                <el-avatar :size="24" :src="emp.avatar">
                  {{ (emp.nickname || emp.username)?.charAt(0)?.toUpperCase() }}
                </el-avatar>
                <span>{{ emp.nickname || emp.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="employeeDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleAddEmployeeSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { OfficeBuilding, Plus, Delete, UserFilled } from '@element-plus/icons-vue'
import type { Organization, User } from '@/types'
import {
  getOrganizationTree,
  createDepartment,
  updateDepartment,
  deleteDepartment,
  addEmployeeToDepartment,
  removeEmployeeFromDepartment,
  getDepartmentEmployees,
} from '@/api/organization'
import { getUsers } from '@/api/users'

const departmentTree = ref<Organization[]>([])
const treeLoading = ref(false)
const selectedDepartment = ref<Organization | null>(null)
const employees = ref<any[]>([])
const employeesLoading = ref(false)

const departmentDialogVisible = ref(false)
const isEdit = ref(false)
const departmentFormRef = ref<FormInstance>()
const submitting = ref(false)
const departmentForm = reactive({
  id: 0,
  name: '',
  parentId: null as number | null,
  code: '',
  description: '',
})

const departmentRules: FormRules = {
  name: [{ required: true, message: '请输入部门名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入部门编码', trigger: 'blur' }],
}

const employeeDialogVisible = ref(false)
const selectedEmployeeId = ref<number | null>(null)
const employeeSearchResults = ref<User[]>([])
const employeeSearchLoading = ref(false)

const departmentOptions = ref<Organization[]>([])

interface FlatDepartment extends Organization {
  level: number
}

const flatDepartments = computed<FlatDepartment[]>(() => {
  const result: FlatDepartment[] = []
  const traverse = (items: Organization[], level: number = 0) => {
    items.forEach((item) => {
      result.push({ ...item, level })
      if ((item as any).children && (item as any).children.length > 0) {
        traverse((item as any).children, level + 1)
      }
    })
  }
  traverse(departmentTree.value)
  return result
})

const departmentCount = computed(() => flatDepartments.value.length)

const flattenDepartments = (depts: Organization[]): Organization[] => {
  const result: Organization[] = []
  const traverse = (items: Organization[]) => {
    items.forEach((item) => {
      result.push(item)
      if ((item as any).children) {
        traverse((item as any).children)
      }
    })
  }
  traverse(depts)
  return result
}

const fetchTree = async () => {
  treeLoading.value = true
  try {
    const { data } = await getOrganizationTree()
    departmentTree.value = data.data
    departmentOptions.value = flattenDepartments(data.data)
  } catch (error) {
  } finally {
    treeLoading.value = false
  }
}

const handleNodeClick = (data: Organization) => {
  selectedDepartment.value = data
  fetchEmployees(data.id)
}

const fetchEmployees = async (departmentId: number) => {
  employeesLoading.value = true
  try {
    const { data } = await getDepartmentEmployees(departmentId, { page: 1, pageSize: 100 })
    employees.value = data.data.list ?? []
  } catch (error) {
  } finally {
    employeesLoading.value = false
  }
}

const handleCreateDepartment = () => {
  isEdit.value = false
  resetDepartmentForm()
  departmentDialogVisible.value = true
}

const handleAddSubDepartment = (data: Organization) => {
  isEdit.value = false
  resetDepartmentForm()
  departmentForm.parentId = data.id
  departmentDialogVisible.value = true
}

const resetDepartmentForm = () => {
  departmentForm.id = 0
  departmentForm.name = ''
  departmentForm.parentId = null
  departmentForm.code = ''
  departmentForm.description = ''
}

const handleDepartmentSubmit = async () => {
  if (!departmentFormRef.value) return
  await departmentFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await createDepartment({
        name: departmentForm.name,
        parentId: departmentForm.parentId,
        code: departmentForm.code,
        description: departmentForm.description,
      })
      ElMessage.success('创建成功')
      departmentDialogVisible.value = false
      fetchTree()
    } catch (error) {
    } finally {
      submitting.value = false
    }
  })
}

const handleDeleteDepartment = async (data: Organization) => {
  try {
    await ElMessageBox.confirm(`确定删除部门「${data.name}」吗？`, '提示', { type: 'warning' })
    await deleteDepartment(data.id)
    ElMessage.success('删除成功')
    if (selectedDepartment.value?.id === data.id) {
      selectedDepartment.value = null
    }
    fetchTree()
  } catch (error) {
    if (error !== 'cancel') {
    }
  }
}

const handleAddEmployee = () => {
  selectedEmployeeId.value = null
  employeeDialogVisible.value = true
}

const searchEmployees = async (query: string) => {
  if (!query) {
    employeeSearchResults.value = []
    return
  }
  employeeSearchLoading.value = true
  try {
    const { data } = await getUsers({ page: 1, pageSize: 20, keyword: query })
    employeeSearchResults.value = data.data.list ?? []
  } catch (error) {
  } finally {
    employeeSearchLoading.value = false
  }
}

const handleAddEmployeeSubmit = async () => {
  if (!selectedDepartment.value || !selectedEmployeeId.value) {
    ElMessage.warning('请选择员工')
    return
  }
  submitting.value = true
  try {
    await addEmployeeToDepartment({
      departmentId: selectedDepartment.value.id,
      userId: selectedEmployeeId.value,
    })
    ElMessage.success('添加成功')
    employeeDialogVisible.value = false
    fetchEmployees(selectedDepartment.value.id)
  } catch (error) {
  } finally {
    submitting.value = false
  }
}

const handleRemoveEmployee = async (userId: number) => {
  if (!selectedDepartment.value) return
  try {
    await ElMessageBox.confirm('确定将该员工移出部门吗？', '提示', { type: 'warning' })
    await removeEmployeeFromDepartment(selectedDepartment.value.id, userId)
    ElMessage.success('移出成功')
    fetchEmployees(selectedDepartment.value.id)
  } catch (error) {
    if (error !== 'cancel') {
    }
  }
}

onMounted(fetchTree)
</script>

<style scoped>
.organization-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 800;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.page-subtitle {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.content-grid {
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: var(--space-4);
  min-height: 600px;
}

@media (max-width: 1024px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
}

.tree-panel {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-card);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border-light);
}

.panel-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.panel-count {
  font-size: 13px;
  color: var(--color-text-muted);
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-3);
}

.dept-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
  margin-bottom: var(--space-1);
}

.dept-card:hover {
  background: var(--color-surface-hover);
}

.dept-card.is-active {
  background: var(--color-primary-lighter);
}

.dept-icon {
  width: 36px;
  height: 36px;
  background: var(--gradient-primary);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.dept-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dept-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.dept-code {
  font-size: 12px;
  color: var(--color-text-muted);
}

.dept-actions {
  display: flex;
  gap: var(--space-1);
  opacity: 0;
  transition: opacity var(--duration-fast);
}

.dept-card:hover .dept-actions {
  opacity: 1;
}

.detail-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.detail-card {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-card);
  padding: var(--space-5);
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--space-4);
}

.dept-title {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.dept-title h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
  color: var(--color-text-primary);
}

.detail-info {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-4);
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.info-item.full-width {
  grid-column: span 2;
}

.info-label {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 500;
}

.info-value {
  font-size: 14px;
  color: var(--color-text-primary);
  font-weight: 600;
}

.employees-card {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-card);
  padding: var(--space-5);
  flex: 1;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.employee-count {
  font-size: 13px;
  color: var(--color-text-muted);
}

.employees-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-3);
}

.employee-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  background: var(--color-surface-hover);
  transition: all var(--duration-fast) var(--ease-out);
}

.employee-item:hover {
  background: var(--color-surface-active);
}

.employee-avatar {
  background: var(--gradient-primary);
  color: white;
  font-weight: 600;
  flex-shrink: 0;
}

.employee-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.employee-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.employee-email {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
