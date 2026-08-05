<template>
  <div class="versions-page">
    <el-tabs v-model="activeTab" type="border-card">
      <!-- 客户端版本 Tab -->
      <el-tab-pane label="客户端版本" name="client">
        <el-row :gutter="20">
          <el-col :span="16">
            <el-card shadow="never">
              <template #header>
                <div class="card-header">
                  <span>客户端版本管理</span>
                  <el-button type="primary" @click="handleCreateClient">发布新版本</el-button>
                </div>
              </template>
              <VersionTable
                :versions="clientStore.versions"
                :loading="clientStore.loading"
                @edit="handleEditClient"
                @delete="handleDeleteClient"
              />
            </el-card>
          </el-col>
          <el-col :span="8">
            <VersionDistributionChart
              :distribution="clientStore.distribution"
              :loading="clientStore.loading"
              @refresh="handleLoadDistribution"
            />
          </el-col>
        </el-row>
      </el-tab-pane>

      <!-- CLI 版本 Tab -->
      <el-tab-pane label="CLI 版本" name="cli">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>CLI 版本管理</span>
              <el-button type="primary" @click="handleCreateCLI">发布新版本</el-button>
            </div>
          </template>

          <el-table :data="cliVersions" v-loading="cliLoading" stripe border style="width: 100%">
            <el-table-column prop="version" label="版本" width="120" />
            <el-table-column label="产物" width="120">
              <template #default="{ row }">
                <el-tag :type="row.product === 'mcp' ? 'warning' : 'primary'" size="small" effect="plain">
                  {{ row.product === 'mcp' ? 'mcp' : 'cli' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="平台" width="160">
              <template #default="{ row }">
                <el-tag size="small">{{ row.platform || `${row.os}/${row.arch}` }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="downloadUrl" label="下载地址" min-width="200" show-overflow-tooltip />
            <el-table-column label="SHA256" width="130">
              <template #default="{ row }">
                <el-text v-if="row.sha256" type="info" size="small" truncated>{{ row.sha256.slice(0, 16) }}…</el-text>
                <span v-else style="color: #909399">-</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                  {{ row.status === 'active' ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="强制更新" width="90">
              <template #default="{ row }">
                <el-tag :type="row.forceUpdate ? 'danger' : 'info'" size="small">
                  {{ row.forceUpdate ? '是' : '否' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="releaseDate" label="发布日期" width="110" />
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="handleEditCLI(row)">编辑</el-button>
                <el-button
                  link
                  :type="row.status === 'active' ? 'warning' : 'success'"
                  size="small"
                  @click="handleToggleCLI(row)"
                >
                  {{ row.status === 'active' ? '禁用' : '启用' }}
                </el-button>
                <el-button link type="danger" size="small" @click="handleDeleteCLI(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 客户端版本对话框 -->
    <VersionFormDialog
      v-model:visible="clientDialogVisible"
      :is-edit="isClientEdit"
      :version-data="currentClientVersion"
      :submit-loading="clientSubmitLoading"
      @confirm="handleSubmitClient"
    />

    <!-- CLI 版本对话框 -->
    <el-dialog
      v-model="cliDialogVisible"
      :title="isCLIEdit ? '编辑 CLI 版本' : '发布 CLI 版本'"
      width="520px"
      destroy-on-close
    >
      <el-form :model="cliForm" label-width="90px" label-position="right">
        <el-form-item label="版本号" required>
          <el-input v-model="cliForm.version" placeholder="如 1.0.0" :disabled="isCLIEdit" />
        </el-form-item>
        <el-form-item label="产物" required>
          <el-radio-group v-model="cliForm.product" :disabled="isCLIEdit">
            <el-radio value="cli">cli（命令行工具）</el-radio>
            <el-radio value="mcp">mcp（MCP Server）</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="平台" required>
          <el-select v-model="cliForm.os" style="width: 140px" placeholder="操作系统" :disabled="isCLIEdit">
            <el-option label="macOS" value="darwin" />
            <el-option label="Linux" value="linux" />
            <el-option label="Windows" value="windows" />
          </el-select>
          <el-select v-model="cliForm.arch" style="width: 140px; margin-left: 10px" placeholder="架构" :disabled="isCLIEdit">
            <el-option label="arm64" value="arm64" />
            <el-option label="amd64" value="amd64" />
          </el-select>
        </el-form-item>
        <el-form-item label="下载地址" required>
          <div class="download-url-input">
            <el-input
              v-model="cliForm.downloadUrl"
              :disabled="true"
              placeholder="上传二进制文件后自动生成下载链接"
            />
            <el-upload
              :show-file-list="false"
              :before-upload="beforeCLIUpload"
              :http-request="handleCLIUpload"
              accept="*"
            >
              <el-button type="primary" :loading="cliUploading">
                {{ cliUploading ? '上传中' : '上传' }}
              </el-button>
            </el-upload>
          </div>
          <div v-if="cliUploadProgress > 0 && cliUploadProgress < 100" class="upload-progress">
            <el-progress :percentage="cliUploadProgress" :stroke-width="6" />
          </div>
          <div class="form-item-tip">上传 CLI 二进制文件，系统会自动生成公开下载链接。</div>
        </el-form-item>
        <el-form-item label="SHA256">
          <el-input v-model="cliForm.sha256" placeholder="上传后自动计算（可手动覆盖）" />
        </el-form-item>
        <el-form-item label="更新说明">
          <el-input v-model="cliForm.updateNotes" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
        <el-form-item label="强制更新">
          <el-switch v-model="cliForm.forceUpdate" />
        </el-form-item>
        <el-form-item label="灰度比例" v-if="!cliForm.forceUpdate">
          <el-slider v-model="cliForm.rolloutPercentage" :min="0" :max="100" :step="5" show-input />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cliDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="cliSubmitLoading" @click="handleSubmitCLI">
          {{ isCLIEdit ? '更新' : '发布' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useClientStore } from '@/stores/client'
import type { ClientVersion, CreateVersionParams, UpdateVersionParams } from '@/types/client'
import VersionTable from './components/VersionTable.vue'
import VersionDistributionChart from './components/VersionDistributionChart.vue'
import VersionFormDialog from './components/VersionFormDialog.vue'
import { request } from '@/utils/request'
import {
  getCLIVersions,
  createCLIVersion,
  updateCLIVersion,
  deleteCLIVersion,
  toggleCLIVersionStatus,
} from '@/api/versions'

const clientStore = useClientStore()
const activeTab = ref('client')

// ==================== 客户端版本 ====================
const clientDialogVisible = ref(false)
const isClientEdit = ref(false)
const clientSubmitLoading = ref(false)
const currentClientVersion = reactive<Partial<ClientVersion>>({})

onMounted(async () => {
  await Promise.all([
    clientStore.loadVersions(),
    clientStore.loadDistribution(),
  ])
})

function handleCreateClient() {
  isClientEdit.value = false
  Object.assign(currentClientVersion, {
    releaseDate: new Date().toISOString().split('T')[0],
    platform: 'windows',
    forceUpdate: false,
    rolloutPercentage: 100,
  })
  clientDialogVisible.value = true
}

function handleEditClient(row: ClientVersion) {
  isClientEdit.value = true
  Object.assign(currentClientVersion, { ...row })
  clientDialogVisible.value = true
}

async function handleSubmitClient(data: Record<string, unknown>) {
  clientSubmitLoading.value = true
  try {
    if (isClientEdit.value) {
      const id = currentClientVersion.id
      if (!id) { ElMessage.error('缺少版本 ID'); return }
      await clientStore.editVersion(id, data as unknown as UpdateVersionParams)
      ElMessage.success('更新成功')
    } else {
      await clientStore.addVersion(data as unknown as CreateVersionParams)
      ElMessage.success('发布成功')
    }
    clientDialogVisible.value = false
    Object.assign(currentClientVersion, {})
    await clientStore.loadVersions()
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '操作失败')
  } finally {
    clientSubmitLoading.value = false
  }
}

async function handleDeleteClient(id: number) {
  try {
    await ElMessageBox.confirm('确定删除该版本吗？此操作不可恢复。', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await clientStore.removeVersion(id)
    ElMessage.success('删除成功')
    await clientStore.loadVersions()
  } catch (error: unknown) {
    if (error !== 'cancel') {
      ElMessage.error(error instanceof Error ? error.message : '删除失败')
    }
  }
}

async function handleLoadDistribution() {
  try {
    await clientStore.loadDistribution()
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '加载失败')
  }
}

// ==================== CLI 版本 ====================
interface CLIVersionRow {
  id: number
  version: string
  product?: string // "cli" | "mcp"
  platform: string
  os: string
  arch: string
  downloadUrl: string
  sha256: string
  fileSize: number
  updateNotes: string
  forceUpdate: boolean
  rolloutPercentage: number
  status: 'active' | 'inactive'
  releaseDate: string
  createdAt: string
}

const cliVersions = ref<CLIVersionRow[]>([])
const cliLoading = ref(false)
const cliDialogVisible = ref(false)
const isCLIEdit = ref(false)
const cliSubmitLoading = ref(false)
const cliEditingId = ref<number | null>(null)
const cliUploading = ref(false)
const cliUploadProgress = ref(0)

const cliForm = reactive({
  version: '',
  product: 'cli' as 'cli' | 'mcp',
  os: 'darwin',
  arch: 'arm64',
  downloadUrl: '',
  sha256: '',
  fileSize: 0,
  updateNotes: '',
  forceUpdate: false,
  rolloutPercentage: 100,
})

// 切换到 CLI tab 时加载数据
watch(activeTab, (tab) => {
  if (tab === 'cli' && cliVersions.value.length === 0) {
    loadCLIVersions()
  }
})

async function loadCLIVersions() {
  cliLoading.value = true
  try {
    const res = await getCLIVersions()
    cliVersions.value = (res.data as any)?.data?.list || (Array.isArray(res.data?.data) ? res.data.data : [])
  } catch (e: any) {
    ElMessage.error(e.message || '加载失败')
  } finally {
    cliLoading.value = false
  }
}

function resetCLIForm() {
  cliForm.version = ''
  cliForm.product = 'cli'
  cliForm.os = 'darwin'
  cliForm.arch = 'arm64'
  cliForm.downloadUrl = ''
  cliForm.sha256 = ''
  cliForm.fileSize = 0
  cliForm.updateNotes = ''
  cliForm.forceUpdate = false
  cliForm.rolloutPercentage = 100
  cliEditingId.value = null
  cliUploading.value = false
  cliUploadProgress.value = 0
}

function beforeCLIUpload(file: File) {
  const maxSize = 500 * 1024 * 1024
  if (file.size > maxSize) {
    ElMessage.error('文件大小不能超过 500MB')
    return false
  }
  return true
}

async function handleCLIUpload(options: { file: File }) {
  cliUploading.value = true
  cliUploadProgress.value = 0
  try {
    const sha256 = await calculateSHA256(options.file)
    const formData = new FormData()
    formData.append('file', options.file)
    formData.append('source', 'version')

    const response = await request({
      url: '/v1/upload',
      method: 'post',
      data: formData,
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total) {
          cliUploadProgress.value = Math.round((progressEvent.loaded / progressEvent.total) * 100)
        }
      },
    })

    const res = response.data as { code: number; data: { id: number; url: string; name: string } }
    if (res.code === 0 && res.data.url) {
      // 存储相对路径，后端根据请求 Host 动态拼接完整 URL
      cliForm.downloadUrl = `/api/v1/public/files/${res.data.id}/download`
      cliForm.sha256 = sha256
      cliForm.fileSize = options.file.size
      ElMessage.success('上传成功，下载链接已自动填入')
    } else {
      ElMessage.error('上传失败')
    }
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '上传失败')
  } finally {
    cliUploading.value = false
    cliUploadProgress.value = 0
  }
}

// SHA-256 纯 JS 实现，仅在 Web Crypto 不可用时（非 HTTPS / 非 localhost 的
// 非安全上下文）作为兜底，保证管理员在内网 HTTP 访问时仍可发布 CLI / MCP 版本。
// 已用标准测试向量与随机长度二进制校验（与 Node crypto 结果一致）。
function sha256HexFallback(bytes: Uint8Array): string {
  const rightRotate = (value: number, amount: number) => (value >>> amount) | (value << (32 - amount))
  const words = new Uint32Array(((bytes.length + 9 + 63) >> 6) << 4)
  const bitLen = bytes.length * 8
  for (let i = 0; i < bytes.length; i++) {
    words[i >> 2] |= bytes[i] << ((3 - (i % 4)) * 8)
  }
  const wlen = ((bytes.length + 8) >> 6) + 1
  words[bytes.length >> 2] |= 0x80 << ((((3 - (bytes.length % 4)) * 8) % 32) + 0)
  words[wlen * 16 - 2] = Math.floor(bitLen / 0x100000000)
  words[wlen * 16 - 1] = bitLen

  const k = [
    0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,
    0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,
    0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
    0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,
    0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,
    0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
    0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,
    0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2,
  ]
  let h = [0x6a09e667,0xbb67ae85,0x3c6ef372,0xa54ff53a,0x510e527f,0x9b05688c,0x1f83d9ab,0x5be0cd19]
  const w = new Int32Array(64)
  for (let off = 0; off < wlen * 16; off += 16) {
    const oldHash = h.slice(0)
    for (let t = 0; t < 16; t++) w[t] = words[off + t] | 0
    for (let t = 16; t < 64; t++) {
      const w15 = w[t - 15], w2 = w[t - 2]
      w[t] = (w[t - 16]
        + (rightRotate(w15, 7) ^ rightRotate(w15, 18) ^ (w15 >>> 3))
        + w[t - 7]
        + (rightRotate(w2, 17) ^ rightRotate(w2, 19) ^ (w2 >>> 10))) | 0
    }
    let h0 = h[0], h1 = h[1], h2 = h[2], h3 = h[3], h4 = h[4], h5 = h[5], h6 = h[6], h7 = h[7]
    for (let t = 0; t < 64; t++) {
      const temp1 = (h7
        + (rightRotate(h4, 6) ^ rightRotate(h4, 11) ^ rightRotate(h4, 25))
        + ((h4 & h5) ^ (~h4 & h6)) + k[t] + w[t]) | 0
      const temp2 = ((rightRotate(h0, 2) ^ rightRotate(h0, 13) ^ rightRotate(h0, 22))
        + ((h0 & h1) ^ (h0 & h2) ^ (h1 & h2))) | 0
      h7 = h6; h6 = h5; h5 = h4; h4 = (h3 + temp1) | 0; h3 = h2; h2 = h1; h1 = h0; h0 = (temp1 + temp2) | 0
    }
    h[0] = (h0 + oldHash[0]) | 0; h[1] = (h1 + oldHash[1]) | 0; h[2] = (h2 + oldHash[2]) | 0; h[3] = (h3 + oldHash[3]) | 0
    h[4] = (h4 + oldHash[4]) | 0; h[5] = (h5 + oldHash[5]) | 0; h[6] = (h6 + oldHash[6]) | 0; h[7] = (h7 + oldHash[7]) | 0
  }
  const HEX = '0123456789abcdef'
  let out = ''
  for (let i = 0; i < 8; i++) {
    for (let j = 3; j >= 0; j--) {
      const b = (h[i] >>> (j * 8)) & 255
      out += HEX[b >> 4] + HEX[b & 0xf]
    }
  }
  return out
}

async function calculateSHA256(file: File): Promise<string> {
  const buffer = await file.arrayBuffer()
  // 非安全上下文（HTTP 内网访问）下 crypto.subtle 不可用，退回纯 JS 计算，
  // 避免在发布 CLI / MCP 版本上传二进制时抛 "Cannot read properties of undefined (reading 'digest')"。
  if (crypto?.subtle) {
    const digest = await crypto.subtle.digest('SHA-256', buffer)
    return Array.from(new Uint8Array(digest))
      .map(byte => byte.toString(16).padStart(2, '0'))
      .join('')
  }
  return sha256HexFallback(new Uint8Array(buffer))
}

function handleCreateCLI() {
  isCLIEdit.value = false
  resetCLIForm()
  cliDialogVisible.value = true
}

function handleEditCLI(row: CLIVersionRow) {
  isCLIEdit.value = true
  cliEditingId.value = row.id
  cliForm.version = row.version
  cliForm.product = (row.product === 'mcp' ? 'mcp' : 'cli')
  cliForm.os = row.os || 'darwin'
  cliForm.arch = row.arch || 'arm64'
  cliForm.downloadUrl = row.downloadUrl
  cliForm.sha256 = row.sha256 || ''
  cliForm.fileSize = row.fileSize || 0
  cliForm.updateNotes = row.updateNotes || ''
  cliForm.forceUpdate = row.forceUpdate || false
  cliForm.rolloutPercentage = row.rolloutPercentage ?? 100
  cliDialogVisible.value = true
}

async function handleSubmitCLI() {
  if (!cliForm.version || !cliForm.os || !cliForm.arch || !cliForm.downloadUrl || !cliForm.sha256) {
    ElMessage.warning('请填写版本号、平台、下载地址和 SHA256')
    return
  }
  cliSubmitLoading.value = true
  try {
    if (isCLIEdit.value && cliEditingId.value) {
      await updateCLIVersion(cliEditingId.value, {
        downloadUrl: cliForm.downloadUrl,
        sha256: cliForm.sha256,
        fileSize: cliForm.fileSize,
        updateNotes: cliForm.updateNotes,
        forceUpdate: cliForm.forceUpdate,
      })
      ElMessage.success('更新成功')
    } else {
      await createCLIVersion({
        version: cliForm.version,
        product: cliForm.product,
        os: cliForm.os,
        arch: cliForm.arch,
        downloadUrl: cliForm.downloadUrl,
        sha256: cliForm.sha256,
        fileSize: cliForm.fileSize,
        updateNotes: cliForm.updateNotes,
        forceUpdate: cliForm.forceUpdate,
        rolloutPercentage: cliForm.rolloutPercentage,
      })
      ElMessage.success('发布成功')
    }
    cliDialogVisible.value = false
    await loadCLIVersions()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    cliSubmitLoading.value = false
  }
}

async function handleToggleCLI(row: CLIVersionRow) {
  const newStatus = row.status === 'active' ? 'inactive' : 'active'
  try {
    await toggleCLIVersionStatus(row.id, newStatus as any)
    ElMessage.success(newStatus === 'active' ? '已启用' : '已禁用')
    await loadCLIVersions()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleDeleteCLI(id: number) {
  try {
    await ElMessageBox.confirm('确定删除该版本吗？此操作不可恢复。', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteCLIVersion(id)
    ElMessage.success('删除成功')
    await loadCLIVersions()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '删除失败')
    }
  }
}
</script>

<style scoped>
.versions-page {
  display: flex;
  flex-direction: column;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-item-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
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
