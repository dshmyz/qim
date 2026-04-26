<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑小程序' : '创建小程序'"
    width="500px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="80px"
    >
      <el-form-item label="AppID" prop="appID">
        <el-input
          v-model="form.appID"
          placeholder="请输入小程序AppID"
          :disabled="isEdit"
        />
      </el-form-item>
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入小程序名称" />
      </el-form-item>
      <el-form-item label="图标URL" prop="icon">
        <el-input v-model="form.icon" placeholder="请输入图标URL" />
      </el-form-item>
      <el-form-item label="路径" prop="path">
        <el-input v-model="form.path" placeholder="请输入小程序路径" />
      </el-form-item>
      <el-form-item label="描述" prop="description">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          placeholder="请输入描述"
        />
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
import type { MiniApp } from '@/types'
import { createMiniApp, updateMiniApp } from '@/api/miniApps'

const props = defineProps<{
  modelValue: boolean
  miniApp?: MiniApp | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const isEdit = computed(() => !!props.miniApp)

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  id: 0,
  appID: '',
  name: '',
  icon: '',
  path: '',
  description: '',
})

const rules: FormRules = {
  appID: [{ required: true, message: '请输入AppID', trigger: 'blur' }],
  name: [{ required: true, message: '请输入小程序名称', trigger: 'blur' }],
  path: [{ required: true, message: '请输入小程序路径', trigger: 'blur' }],
}

watch(
  () => props.miniApp,
  (miniApp) => {
    if (miniApp) {
      form.id = miniApp.id
      form.appID = miniApp.appID
      form.name = miniApp.name
      form.icon = miniApp.icon || ''
      form.path = miniApp.path
      form.description = miniApp.description || ''
    }
  },
  { immediate: true }
)

const resetForm = () => {
  form.id = 0
  form.appID = ''
  form.name = ''
  form.icon = ''
  form.path = ''
  form.description = ''
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
        await updateMiniApp(form.id, {
          name: form.name,
          icon: form.icon || undefined,
          path: form.path,
          description: form.description || undefined,
        })
      } else {
        await createMiniApp({
          appID: form.appID,
          name: form.name,
          icon: form.icon || undefined,
          path: form.path,
          description: form.description || undefined,
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
