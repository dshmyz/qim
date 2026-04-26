<template>
  <div class="organization-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>组织架构管理</h3>
        <el-button type="primary" @click="handleCreateDepartment">创建部门</el-button>
      </div>

      <el-row :gutter="20">
        <!-- 部门树 -->
        <el-col :span="10">
          <el-card shadow="never" class="tree-card">
            <template #header>
              <span>部门列表</span>
            </template>
            <el-tree
              v-loading="treeLoading"
              :data="departmentTree"
              :props="{ label: 'name', children: 'children' }"
              node-key="id"
              highlight-current
              default-expand-all
              @node-click="handleNodeClick"
            >
              <template #default="{ node, data }">
                <span class="tree-node-label">
                  <el-icon><OfficeBuilding /></el-icon>
                  {{ node.label }}
                </span>
                <span class="tree-node-actions">
                  <el-button size="small" text @click.stop="handleAddSubDepartment(data)">
                    <el-icon><Plus /></el-icon>
                  </el-button>
                  <el-button size="small" text type="danger" @click.stop="handleDeleteDepartment(data)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </span>
              </template>
            </el-tree>
          </el-card>
        </el-col>

        <!-- 部门详情 -->
        <el-col :span="14">
          <el-card v-if="selectedDepartment" shadow="never" class="detail-card">
            <template #header>
              <div class="detail-header">
                <span>部门详情</span>
                <el-button type="warning" size="small" @click="handleAddEmployee">添加员工</el-button>
              </div>
            </template>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="部门名称">{{ selectedDepartment.name }}</el-descriptions-item>
              <el-descriptions-item label="编码">{{ selectedDepartment.code || '-' }}</el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="selectedDepartment.status === 'active' ? 'success' : 'info'">
                  {{ selectedDepartment.status === 'active' ? '启用' : '停用' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ selectedDepartment.createdAt }}</el-descriptions-item>
              <el-descriptions-item label="描述" :span="2">{{ selectedDepartment.description || '-' }}</el-descriptions-item>
            </el-descriptions>

            <!-- 部门员工列表 -->
            <div class="employee-section">
              <h4>部门员工</h4>
              <el-table
                :data="employees"
                v-loading="employeesLoading"
                size="small"
                max-height="300"
              >
                <el-table-column prop="id" label="ID" width="80" />
                <el-table-column prop="username" label="用户名" />
                <el-table-column prop="nickname" label="昵称" />
                <el-table-column prop="email" label="邮箱" />
                <el-table-column label="操作" width="100">
                  <template #default="{ row }">
                    <el-popconfirm
                      title="确定将该员工移出部门吗？"
                      @confirm="handleRemoveEmployee(row.id)"
                    >
                      <template #reference>
                        <el-button size="small" type="danger" text>移出</el-button>
                      </template>
                    </el-popconfirm>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
          <el-empty v-else description="请选择左侧部门查看详情" :image-size="100" />
        </el-col>
      </el-row>
    </el-card>

    <!-- 创建/编辑部门对话框 -->
    <el-dialog
      v-model="departmentDialogVisible"
      :title="isEdit ? '编辑部门' : '创建部门'"
      width="450px"
    >
      <el-form
        ref="departmentFormRef"
        :model="departmentForm"
        :rules="departmentRules"
        label-width="80px"
      >
        <el-form-item label="部门名称" prop="name">
          <el-input v-model="departmentForm.name" />
        </el-form-item>
        <el-form-item label="上级部门">
          <el-select v-model="departmentForm.parentId" placeholder="无上级部门（根部门）" clearable>
            <el-option
              v-for="dept in departmentOptions"
              :key="dept.id"
              :label="dept.name"
              :value="dept.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="编码" prop="code">
          <el-input v-model="departmentForm.code" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="departmentForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="departmentDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleDepartmentSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 添加员工到部门对话框 -->
    <el-dialog
      v-model="employeeDialogVisible"
      title="添加员工到部门"
      width="450px"
    >
      <el-form label-width="80px">
        <el-form-item label="选择员工">
          <el-select
            v-model="selectedEmployeeId"
            filterable
            remote
            :remote-method="searchEmployees"
            :loading="employeeSearchLoading"
            placeholder="请输入用户名搜索"
          >
            <el-option
              v-for="emp in employeeSearchResults"
              :key="emp.id"
              :label="emp.nickname || emp.username"
              :value="emp.id"
            />
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
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { OfficeBuilding, Plus, Delete } from '@element-plus/icons-vue'
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

// 部门树
const departmentTree = ref<Organization[]>([])
const treeLoading = ref(false)
const selectedDepartment = ref<Organization | null>(null)
const employees = ref<any[]>([])
const employeesLoading = ref(false)

// 部门表单
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

// 员工选择
const employeeDialogVisible = ref(false)
const selectedEmployeeId = ref<number | null>(null)
const employeeSearchResults = ref<User[]>([])
const employeeSearchLoading = ref(false)

// 获取部门选项（扁平化树用于下拉选择）
const departmentOptions = ref<Organization[]>([])

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

// 加载组织树
const fetchTree = async () => {
  treeLoading.value = true
  try {
    const { data } = await getOrganizationTree()
    departmentTree.value = data.data
    departmentOptions.value = flattenDepartments(data.data)
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    treeLoading.value = false
  }
}

// 选中部门节点
const handleNodeClick = (data: Organization) => {
  selectedDepartment.value = data
  fetchEmployees(data.id)
}

const fetchEmployees = async (departmentId: number) => {
  employeesLoading.value = true
  try {
    const { data } = await getDepartmentEmployees(departmentId, { page: 1, pageSize: 100 })
    employees.value = data.data.list
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    employeesLoading.value = false
  }
}

// 创建部门
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
      // 错误已在请求拦截器中处理
    } finally {
      submitting.value = false
    }
  })
}

// 删除部门
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
      // 错误已在请求拦截器中处理
    }
  }
}

// 添加员工
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
    employeeSearchResults.value = data.data.list
  } catch (error) {
    // 错误已在请求拦截器中处理
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
    // 错误已在请求拦截器中处理
  } finally {
    submitting.value = false
  }
}

const handleRemoveEmployee = async (userId: number) => {
  if (!selectedDepartment.value) return
  try {
    await removeEmployeeFromDepartment(selectedDepartment.value.id, userId)
    ElMessage.success('移出成功')
    fetchEmployees(selectedDepartment.value.id)
  } catch (error) {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchTree)
</script>

<style scoped>
.organization-page {
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

.tree-card {
  min-height: 500px;
}

.tree-node-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}

.tree-node-actions {
  margin-left: auto;
}

.detail-card {
  min-height: 500px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.employee-section {
  margin-top: var(--space-6);
}

.employee-section h4 {
  margin: 0 0 var(--space-3);
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
}
</style>
