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
