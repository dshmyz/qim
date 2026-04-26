<template>
  <div class="sensitive-words-page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :model="searchForm" inline>
          <el-form-item label="关键词">
            <el-input
              v-model="searchForm.keyword"
              placeholder="请输入敏感词"
              clearable
              @keyup.enter="handleSearch"
            />
          </el-form-item>
          <el-form-item label="分类">
            <el-select v-model="searchForm.category" placeholder="请选择分类" clearable>
              <el-option
                v-for="cat in categories"
                :key="cat"
                :label="cat"
                :value="cat"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
        <div class="action-buttons">
          <el-button type="primary" @click="handleCreate">添加敏感词</el-button>
          <el-button type="success" @click="batchDialogVisible = true">批量导入</el-button>
        </div>
      </div>

      <!-- 敏感词列表 -->
      <el-table :data="sensitiveWords" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="word" label="敏感词" min-width="160">
          <template #default="{ row }">
            <el-tag :type="levelType(row.level)">{{ row.word }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="category" label="分类" width="120" />
        <el-table-column label="等级" width="100">
          <template #default="{ row }">
            <el-tag :type="levelType(row.level)" size="small">
              {{ levelLabel(row.level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              active-value="active"
              inactive-value="inactive"
              @change="handleToggleStatus(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除该敏感词吗？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm>
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
          @size-change="fetchSensitiveWords"
          @current-change="fetchSensitiveWords"
        />
      </div>
    </el-card>

    <!-- 添加/编辑敏感词对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑敏感词' : '添加敏感词'"
      width="500px"
    >
      <el-form
        ref="wordFormRef"
        :model="wordForm"
        :rules="wordRules"
        label-width="80px"
      >
        <el-form-item label="敏感词" prop="word">
          <el-input v-model="wordForm.word" :disabled="isEdit" placeholder="请输入敏感词" />
        </el-form-item>
        <el-form-item label="分类" prop="category">
          <el-select v-model="wordForm.category" placeholder="请选择分类" style="width: 100%">
            <el-option
              v-for="cat in categories"
              :key="cat"
              :label="cat"
              :value="cat"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="等级" prop="level">
          <el-radio-group v-model="wordForm.level">
            <el-radio label="low">低</el-radio>
            <el-radio label="medium">中</el-radio>
            <el-radio label="high">高</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 批量导入对话框 -->
    <el-dialog
      v-model="batchDialogVisible"
      title="批量导入敏感词"
      width="500px"
    >
      <el-form
        ref="batchFormRef"
        :model="batchForm"
        :rules="batchRules"
        label-width="80px"
      >
        <el-form-item label="敏感词" prop="words">
          <el-input
            v-model="batchForm.words"
            type="textarea"
            :rows="6"
            placeholder="每行一个敏感词"
          />
        </el-form-item>
        <el-form-item label="分类" prop="category">
          <el-select v-model="batchForm.category" placeholder="请选择分类" style="width: 100%">
            <el-option
              v-for="cat in categories"
              :key="cat"
              :label="cat"
              :value="cat"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="等级" prop="level">
          <el-radio-group v-model="batchForm.level">
            <el-radio label="low">低</el-radio>
            <el-radio label="medium">中</el-radio>
            <el-radio label="high">高</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchSubmitting" @click="handleBatchSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { SensitiveWord } from '@/types'
import {
  getSensitiveWords,
  createSensitiveWord,
  updateSensitiveWord,
  deleteSensitiveWord,
  toggleSensitiveWordStatus,
  batchCreateSensitiveWords,
} from '@/api/sensitiveWords'

// 分类选项
const categories = ['政治', '色情', '暴力', '辱骂', '广告', '其他']

// 搜索和分页
const searchForm = reactive({ keyword: '', category: '' })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const sensitiveWords = ref<SensitiveWord[]>([])
const loading = ref(false)

// 表单
const dialogVisible = ref(false)
const isEdit = ref(false)
const wordFormRef = ref<FormInstance>()
const submitting = ref(false)
const wordForm = reactive({
  id: 0,
  word: '',
  category: '',
  level: 'medium' as 'low' | 'medium' | 'high',
})

const wordRules: FormRules = {
  word: [{ required: true, message: '请输入敏感词', trigger: 'blur' }],
  category: [{ required: true, message: '请选择分类', trigger: 'change' }],
}

// 批量导入表单
const batchDialogVisible = ref(false)
const batchFormRef = ref<FormInstance>()
const batchSubmitting = ref(false)
const batchForm = reactive({
  words: '',
  category: '',
  level: 'medium' as 'low' | 'medium' | 'high',
})

const batchRules: FormRules = {
  words: [{ required: true, message: '请输入敏感词列表', trigger: 'blur' }],
  category: [{ required: true, message: '请选择分类', trigger: 'change' }],
}

// 工具函数
const levelLabel = (level: string): string => {
  const map: Record<string, string> = { low: '低', medium: '中', high: '高' }
  return map[level] || level
}

const levelType = (level: string): 'success' | 'warning' | 'danger' => {
  const map: Record<string, 'success' | 'warning' | 'danger'> = { low: 'success', medium: 'warning', high: 'danger' }
  return map[level] || 'info'
}

// 获取列表
const fetchSensitiveWords = async () => {
  loading.value = true
  try {
    const { data } = await getSensitiveWords({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword || undefined,
      category: searchForm.category || undefined,
    })
    sensitiveWords.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchSensitiveWords()
}

const handleReset = () => {
  searchForm.keyword = ''
  searchForm.category = ''
  handleSearch()
}

// 添加
const handleCreate = () => {
  isEdit.value = false
  resetWordForm()
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: SensitiveWord) => {
  isEdit.value = true
  wordForm.id = row.id
  wordForm.word = row.word
  wordForm.category = row.category
  wordForm.level = row.level
  dialogVisible.value = true
}

const resetWordForm = () => {
  wordForm.id = 0
  wordForm.word = ''
  wordForm.category = ''
  wordForm.level = 'medium'
}

// 提交
const handleSubmit = async () => {
  if (!wordFormRef.value) return
  await wordFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) {
        await updateSensitiveWord(wordForm.id, {
          category: wordForm.category,
          level: wordForm.level,
        })
        ElMessage.success('更新成功')
      } else {
        await createSensitiveWord({
          word: wordForm.word,
          category: wordForm.category,
          level: wordForm.level,
        })
        ElMessage.success('添加成功')
      }
      dialogVisible.value = false
      fetchSensitiveWords()
    } catch {
      // 错误已在请求拦截器中处理
    } finally {
      submitting.value = false
    }
  })
}

// 删除
const handleDelete = async (id: number) => {
  try {
    await deleteSensitiveWord(id)
    ElMessage.success('删除成功')
    fetchSensitiveWords()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 切换状态
const handleToggleStatus = async (row: SensitiveWord) => {
  try {
    await toggleSensitiveWordStatus(row.id, row.status)
    ElMessage.success('状态更新成功')
  } catch {
    // 错误已在请求拦截器中处理
    row.status = row.status === 'active' ? 'inactive' : 'active'
  }
}

// 批量提交
const handleBatchSubmit = async () => {
  if (!batchFormRef.value) return
  await batchFormRef.value.validate(async (valid) => {
    if (!valid) return
    batchSubmitting.value = true
    try {
      const words = batchForm.words.split('\n').map(w => w.trim()).filter(Boolean)
      await batchCreateSensitiveWords({
        words,
        category: batchForm.category,
        level: batchForm.level,
      })
      ElMessage.success(`成功导入 ${words.length} 个敏感词`)
      batchDialogVisible.value = false
      fetchSensitiveWords()
    } catch {
      // 错误已在请求拦截器中处理
    } finally {
      batchSubmitting.value = false
    }
  })
}

onMounted(fetchSensitiveWords)
</script>

<style scoped>
.sensitive-words-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.action-buttons {
  display: flex;
  gap: var(--space-2);
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
