import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'

interface PaginationState {
  page: number
  pageSize: number
  total: number
}

export interface FormField {
  name: string
  label: string
  type: 'input' | 'textarea' | 'password' | 'select' | 'switch' | 'number'
  props?: Record<string, unknown>
  options?: Array<{ label: string; value: unknown }>
  required?: boolean
}

interface EntityAPI<T> {
  getList: (params: Record<string, unknown>) => Promise<{ data: { data: { list: T[]; total: number } } }>
  create: (data: Record<string, unknown>) => Promise<void>
  update: (id: number, data: Record<string, unknown>) => Promise<void>
  delete: (id: number) => Promise<void>
}

interface UseEntityOptions<T> {
  api: EntityAPI<T>
  searchFields: string[]
  formFields: FormField[]
  successMessages?: {
    create?: string
    update?: string
    delete?: string
  }
}

export function useEntity<T extends { id: number }>(options: UseEntityOptions<T>) {
  const { api, searchFields, formFields, successMessages = {} } = options

  // List state
  const list = ref<T[]>([])
  const loading = ref(false)
  const pagination = reactive<PaginationState>({ page: 1, pageSize: 10, total: 0 })

  // Search
  const searchForm = reactive<Record<string, unknown>>({})
  for (const field of searchFields) {
    searchForm[field] = ''
  }

  // Dialog
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const currentRow = ref<T | null>(null)
  const formData = ref<Record<string, unknown>>({})
  const submitting = ref(false)

  // Fetch data
  async function fetchData() {
    loading.value = true
    try {
      const params: Record<string, unknown> = {
        page: pagination.page,
        pageSize: pagination.pageSize,
      }
      for (const field of searchFields) {
        if (searchForm[field]) {
          params[field] = searchForm[field]
        }
      }
      const { data } = await api.getList(params)
      list.value = data.data.list
      pagination.total = data.data.total
    } catch (error) {
      console.error('[useEntity] fetch failed:', error)
    } finally {
      loading.value = false
    }
  }

  // Search
  function handleSearch() {
    pagination.page = 1
    fetchData()
  }

  function handleReset() {
    for (const field of searchFields) {
      searchForm[field] = ''
    }
    handleSearch()
  }

  // Create
  function handleCreate() {
    dialogMode.value = 'create'
    currentRow.value = null
    formData.value = {}
    dialogVisible.value = true
  }

  // Edit
  function handleEdit(row: T) {
    dialogMode.value = 'edit'
    currentRow.value = row
    formData.value = { ...row }
    dialogVisible.value = true
  }

  // Delete
  async function handleDelete(id: number) {
    try {
      await api.delete(id)
      ElMessage.success(successMessages.delete || '删除成功')
      fetchData()
    } catch (error) {
      if (error !== 'cancel') {
        console.error('[useEntity] delete failed:', error)
      }
    }
  }

  // Save
  async function handleSave(data: Record<string, unknown>) {
    submitting.value = true
    try {
      if (dialogMode.value === 'create') {
        await api.create(data)
        ElMessage.success(successMessages.create || '创建成功')
      } else if (currentRow.value) {
        await api.update(currentRow.value.id, data)
        ElMessage.success(successMessages.update || '更新成功')
      }
      dialogVisible.value = false
      fetchData()
    } catch (error) {
      console.error('[useEntity] save failed:', error)
    } finally {
      submitting.value = false
    }
  }

  return {
    list,
    loading,
    pagination,
    searchForm,
    dialogVisible,
    dialogMode,
    currentRow,
    formData,
    submitting,
    formFields,
    handleSearch,
    handleReset,
    handleCreate,
    handleEdit,
    handleDelete,
    handleSave,
    fetchData,
  }
}
