<template>
  <div class="version-management-page">
    <el-card shadow="never">
      <!-- 操作栏 -->
      <div class="action-bar">
        <el-button type="primary" @click="handleCreate">发布新版本</el-button>
      </div>

      <!-- 版本列表 -->
      <el-table :data="versions" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="version" label="版本号" width="140" />
        <el-table-column label="平台" width="120">
          <template #default="{ row }">
            <el-tag :type="platformType(row.platform)" size="small">
              {{ platformLabel(row.platform) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updateNotes" label="更新说明" min-width="240" show-overflow-tooltip />
        <el-table-column label="强制更新" width="100">
          <template #default="{ row }">
            <el-tag :type="row.forceUpdate ? 'danger' : 'info'" size="small">
              {{ row.forceUpdate ? '是' : '否' }}
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
        <el-table-column prop="releaseDate" label="发布日期" width="140" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="primary" @click="handleCopyDownload(row)">复制下载链接</el-button>
            <el-popconfirm title="确定删除该版本吗？" @confirm="handleDelete(row.id)">
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
          @size-change="fetchVersions"
          @current-change="fetchVersions"
        />
      </div>
    </el-card>

    <!-- 发布/编辑版本对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑版本' : '发布新版本'"
      width="600px"
    >
      <el-form
        ref="versionFormRef"
        :model="versionForm"
        :rules="versionRules"
        label-width="100px"
      >
        <el-form-item label="版本号" prop="version">
          <el-input v-model="versionForm.version" :disabled="isEdit" placeholder="例如：2.1.0" />
        </el-form-item>
        <el-form-item label="平台" prop="platform">
          <el-radio-group v-model="versionForm.platform" :disabled="isEdit">
            <el-radio label="windows">Windows</el-radio>
            <el-radio label="macos">macOS</el-radio>
            <el-radio label="linux">Linux</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="发布日期" prop="releaseDate">
          <el-date-picker
            v-model="versionForm.releaseDate"
            type="date"
            placeholder="选择日期"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="更新说明" prop="updateNotes">
          <el-input
            v-model="versionForm.updateNotes"
            type="textarea"
            :rows="4"
            placeholder="请输入版本更新说明"
          />
        </el-form-item>
        <el-form-item label="下载链接" prop="downloadUrl">
          <div class="download-url-input">
            <el-input v-model="versionForm.downloadUrl" placeholder="请输入安装包下载链接或上传文件" />
            <el-upload
              :show-file-list="false"
              :before-upload="beforeUpload"
              :http-request="handleUpload"
              accept=".exe,.dmg,.AppImage,.zip,.tar.gz"
            >
              <el-button type="primary" :loading="uploading">
                {{ uploading ? '上传中' : '上传' }}
              </el-button>
            </el-upload>
          </div>
          <div v-if="uploadProgress > 0 && uploadProgress < 100" class="upload-progress">
            <el-progress :percentage="uploadProgress" :stroke-width="6" />
          </div>
        </el-form-item>
        <el-form-item label="强制更新">
          <el-switch v-model="versionForm.forceUpdate" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { Version } from '@/types'
import { getVersions, createVersion, updateVersion, deleteVersion, toggleVersionStatus } from '@/api/versions'
import { request } from '@/utils/request'

// 分页
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const versions = ref<Version[]>([])
const loading = ref(false)

// 表单
const dialogVisible = ref(false)
const isEdit = ref(false)
const versionFormRef = ref<FormInstance>()
const submitting = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const versionForm = reactive({
  id: 0,
  version: '',
  platform: 'windows' as 'windows' | 'macos' | 'linux',
  releaseDate: '',
  updateNotes: '',
  forceUpdate: false,
  downloadUrl: '',
})

const versionRules: FormRules = {
  version: [
    { required: true, message: '请输入版本号', trigger: 'blur' },
    { pattern: /^\d+\.\d+\.\d+$/, message: '版本号格式不正确（如：2.1.0）', trigger: 'blur' },
  ],
  platform: [{ required: true, message: '请选择平台', trigger: 'change' }],
  releaseDate: [{ required: true, message: '请选择发布日期', trigger: 'change' }],
  updateNotes: [{ required: true, message: '请输入更新说明', trigger: 'blur' }],
  downloadUrl: [
    { required: true, message: '请输入下载链接', trigger: 'blur' },
    { type: 'url', message: '请输入有效的 URL', trigger: 'blur' },
  ],
}

// 工具函数
const platformLabel = (platform: string): string => {
  const map: Record<string, string> = { windows: 'Windows', macos: 'macOS', linux: 'Linux' }
  return map[platform] || platform
}

const platformType = (platform: string): 'primary' | 'success' | 'warning' => {
  const map: Record<string, 'primary' | 'success' | 'warning'> = { windows: 'primary', macos: 'success', linux: 'warning' }
  return map[platform] || 'info'
}

// 获取列表
const fetchVersions = async () => {
  loading.value = true
  try {
    const { data } = await getVersions({
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    versions.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 创建
const handleCreate = () => {
  isEdit.value = false
  resetVersionForm()
  versionForm.releaseDate = new Date().toISOString().split('T')[0]
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: Version) => {
  isEdit.value = true
  versionForm.id = row.id
  versionForm.version = row.version
  versionForm.platform = row.platform
  versionForm.releaseDate = row.releaseDate
  versionForm.updateNotes = row.updateNotes
  versionForm.forceUpdate = row.forceUpdate
  versionForm.downloadUrl = row.downloadUrl
  dialogVisible.value = true
}

const resetVersionForm = () => {
  versionForm.id = 0
  versionForm.version = ''
  versionForm.platform = 'windows'
  versionForm.releaseDate = ''
  versionForm.updateNotes = ''
  versionForm.forceUpdate = false
  versionForm.downloadUrl = ''
  uploading.value = false
  uploadProgress.value = 0
}

function beforeUpload(file: File) {
  const maxSize = 500 * 1024 * 1024 // 500MB
  if (file.size > maxSize) {
    ElMessage.error('文件大小不能超过 500MB')
    return false
  }
  return true
}

async function handleUpload(options: { file: File }) {
  uploading.value = true
  uploadProgress.value = 0

  try {
    const formData = new FormData()
    formData.append('file', options.file)
    formData.append('source', 'version')

    const response = await request({
      url: '/v1/upload',
      method: 'post',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total) {
          uploadProgress.value = Math.round((progressEvent.loaded / progressEvent.total) * 100)
        }
      },
    })

    const res = response.data as { code: number; data: { id: number; url: string; name: string } }
    if (res.code === 0 && res.data.url) {
      const downloadUrl = `${window.location.origin}/api/v1/public/files/${res.data.id}/download`
      versionForm.downloadUrl = downloadUrl
      ElMessage.success('上传成功，下载链接已自动填入')
    } else {
      ElMessage.error('上传失败')
    }
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '上传失败'
    ElMessage.error(message)
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

// 提交
const handleSubmit = async () => {
  if (!versionFormRef.value) return
  await versionFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) {
        await updateVersion(versionForm.id, {
          updateNotes: versionForm.updateNotes,
          forceUpdate: versionForm.forceUpdate,
        })
        ElMessage.success('更新成功')
      } else {
        await createVersion({
          version: versionForm.version,
          platform: versionForm.platform,
          releaseDate: versionForm.releaseDate,
          updateNotes: versionForm.updateNotes,
          forceUpdate: versionForm.forceUpdate,
          downloadUrl: versionForm.downloadUrl,
        })
        ElMessage.success('发布成功')
      }
      dialogVisible.value = false
      fetchVersions()
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
    await deleteVersion(id)
    ElMessage.success('删除成功')
    fetchVersions()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 切换状态
const handleToggleStatus = async (row: Version) => {
  try {
    await toggleVersionStatus(row.id, row.status)
    ElMessage.success('状态更新成功')
  } catch {
    // 错误已在请求拦截器中处理
    row.status = row.status === 'active' ? 'inactive' : 'active'
  }
}

// 复制下载链接
const handleCopyDownload = async (row: Version) => {
  try {
    await navigator.clipboard.writeText(row.downloadUrl)
    ElMessage.success('下载链接已复制到剪贴板')
  } catch {
    // 降级处理：使用传统方式
    const textarea = document.createElement('textarea')
    textarea.value = row.downloadUrl
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    ElMessage.success('下载链接已复制到剪贴板')
  }
}

onMounted(fetchVersions)
</script>

<style scoped>
.version-management-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.action-bar {
  display: flex;
  justify-content: flex-end;
  padding-bottom: var(--space-4);
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

.download-url-input {
  display: flex;
  gap: 8px;
  width: 100%;
}

.download-url-input .el-input {
  flex: 1;
}

.upload-progress {
  margin-top: 8px;
}
</style>
