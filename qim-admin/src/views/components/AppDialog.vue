<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑应用' : '创建应用'"
    width="500px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="80px"
    >
      <el-form-item label="应用名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入应用名称" />
      </el-form-item>
      <el-form-item label="图标URL" prop="icon">
        <el-input v-model="form.icon" placeholder="请输入图标URL" />
      </el-form-item>
      <el-form-item label="分类" prop="category">
        <el-input v-model="form.category" placeholder="请输入分类" />
      </el-form-item>
      <el-form-item label="链接地址" prop="url">
        <el-input v-model="form.url" placeholder="请输入应用链接" />
      </el-form-item>
      <el-form-item label="打开方式" prop="openType">
        <el-select v-model="form.openType" placeholder="请选择打开方式" style="width: 100%">
          <el-option label="应用内打开" value="in-app" />
          <el-option label="外部打开" value="external" />
        </el-select>
      </el-form-item>
      <el-form-item label="全局应用">
        <el-switch v-model="form.isGlobal" />
        <span style="margin-left: 8px; color: var(--color-text-secondary); font-size: 12px;">
          全局应用对所有用户可见，仅管理员可创建
        </span>
      </el-form-item>
      <el-form-item v-if="form.isGlobal" label="可见范围" prop="scopeType">
        <el-select v-model="form.scopeType" placeholder="请选择可见范围" style="width: 100%">
          <el-option label="所有人可见" value="all" />
          <el-option label="指定用户" value="users" />
          <el-option label="指定组织" value="organizations" />
          <el-option label="指定角色" value="roles" />
        </el-select>
        <div v-if="form.scopeType === 'users'" style="margin-top: 8px;">
          <el-input v-model="form.scopeValue" placeholder="请输入用户ID，多个用逗号分隔" />
          <div style="color: var(--color-text-secondary); font-size: 12px; margin-top: 4px;">
            仅这些用户可以看到并使用此内置应用
          </div>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { App } from '@/types'
import { createApp, updateApp } from '@/api/apps'

const props = defineProps<{
  modelValue: boolean
  app?: App | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const isEdit = computed(() => !!props.app)

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  id: 0,
  name: '',
  icon: '',
  category: '',
  url: '',
  openType: 'in-app' as 'in-app' | 'external',
  isGlobal: false,
  scopeType: 'all',
  scopeValue: '',
  availableOrgIDs: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入应用名称', trigger: 'blur' }],
  category: [{ required: true, message: '请输入分类', trigger: 'blur' }],
  url: [{ required: true, message: '请输入链接地址', trigger: 'blur' }],
  openType: [{ required: true, message: '请选择打开方式', trigger: 'change' }],
}

watch(
  () => props.app,
  (app) => {
    if (app) {
      form.id = app.id
      form.name = app.name
      form.icon = app.icon || ''
      form.category = app.category
      form.url = app.url
      form.openType = app.openType
      form.isGlobal = app.isGlobal || false
      form.scopeType = app.scopeType || 'all'
      form.scopeValue = app.scopeValue || ''
      form.availableOrgIDs = app.availableOrgIDs || ''
    }
  },
  { immediate: true }
)

const resetForm = () => {
  form.id = 0
  form.name = ''
  form.icon = ''
  form.category = ''
  form.url = ''
  form.openType = 'in-app'
  form.isGlobal = false
  form.scopeType = 'all'
  form.scopeValue = ''
  form.availableOrgIDs = ''
  formRef.value?.resetFields()
}

const handleClose = () => {
  visible.value = false
  resetForm()
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      if (isEdit.value) {
        await updateApp(form.id, {
          name: form.name,
          icon: form.icon || undefined,
          category: form.category,
          url: form.url,
          openType: form.openType,
          scopeType: form.scopeType,
          scopeValue: form.scopeValue || undefined,
          availableOrgIDs: form.availableOrgIDs || undefined,
        })
      } else {
        await createApp({
          name: form.name,
          icon: form.icon || undefined,
          category: form.category,
          url: form.url,
          openType: form.openType,
        })
      }
      emit('saved')
      handleClose()
    } catch {
      // 错误已在请求拦截器中处理
    } finally {
      loading.value = false
    }
  })
}
</script>
