<template>
  <div class="mcp-config-section">
    <el-divider content-position="left">外部 MCP 工具</el-divider>

    <p class="section-desc">
      配置 QIM 作为 MCP Client 连接的外部 MCP Server。启用后，群 @AI 普通提问的 ReAct
      循环可调用这些外部工具（如计算器、天气查询），并以前端独立工具卡片呈现调用过程。
      保存后即时热同步，无需重启服务。
    </p>

    <el-form v-loading="loading" label-width="140px">
      <el-form-item label="群 @AI 可用">
        <el-switch
          v-model="groupEnabled"
          active-text="开启"
          inactive-text="关闭"
          active-color="#67c23a"
          inactive-color="#f56c6c"
        />
        <span class="desc" style="margin-left: 8px">
          （关闭时群 @AI 普通提问走纯流式，不调用外部工具；连接仍是配置可改状态）
        </span>
      </el-form-item>

      <el-divider content-position="left">连接列表</el-divider>

      <div v-for="(conn, index) in conns" :key="index" class="conn-card">
        <div class="conn-card-header">
          <strong>连接 #{{ index + 1 }}</strong>
          <el-button type="danger" size="small" plain @click="removeConn(index)">
            删除连接
          </el-button>
        </div>
        <div class="conn-grid">
          <el-form-item label="名称">
            <el-input v-model="conn.name" placeholder="例如：demo" />
          </el-form-item>
          <el-form-item label="传输方式">
            <el-select v-model="conn.transport" style="width: 100%">
              <el-option label="streamable-http" value="streamable-http" />
              <el-option label="http" value="http" />
            </el-select>
          </el-form-item>
          <el-form-item label="端点 URL">
            <el-input v-model="conn.url" placeholder="http://localhost:9100/mcp" />
          </el-form-item>
          <el-form-item label="Bearer Token">
            <el-input
              v-model="conn.token"
              type="password"
              placeholder="可选，Authorization: Bearer"
              show-password
            />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="conn.enabled" active-text="开启" inactive-text="关闭" />
          </el-form-item>
        </div>

        <!-- 工具发现区域 -->
        <div class="tool-discovery">
          <div class="tool-discovery-header">
            <el-button
              size="small"
              :loading="conn._loadingTools"
              :disabled="!conn.url"
              @click="discoverTools(index)"
            >
              发现工具
            </el-button>
            <span v-if="conn._tools && conn._tools.length > 0" class="tool-count">
              共 {{ conn._tools.length }} 个工具
              <span v-if="conn.allowed_tools && conn.allowed_tools.length > 0">
                ，已选 {{ conn.allowed_tools.length }} 个
              </span>
              <span class="desc" style="margin-left: 4px">
                （不勾选 = 全部开放）
              </span>
            </span>
          </div>

          <div v-if="conn._toolsError" class="tool-error">
            <el-text type="danger" size="small">{{ conn._toolsError }}</el-text>
          </div>

          <div v-if="conn._tools && conn._tools.length > 0" class="tool-list">
            <el-checkbox
              v-model="conn._allChecked"
              :indeterminate="conn._indeterminate"
              @change="(val: boolean) => toggleAllTools(index, val)"
              class="tool-checkbox-all"
            >
              全选 / 全不选
            </el-checkbox>
            <el-divider style="margin: 8px 0" />
            <el-checkbox-group
              :model-value="conn.allowed_tools || []"
              @change="(val: string[]) => updateAllowedTools(index, val)"
            >
              <div v-for="tool in conn._tools" :key="tool.name" class="tool-item">
                <el-checkbox :label="tool.name">
                  <span class="tool-name">{{ tool.name }}</span>
                  <span class="tool-desc">{{ tool.description }}</span>
                </el-checkbox>
              </div>
            </el-checkbox-group>
          </div>
        </div>
      </div>

      <el-form-item>
        <el-button size="small" @click="addConn">+ 添加连接</el-button>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="submitting" @click="handleSave">
          保存配置
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSystemConfig, updateSystemConfig } from '@/api/systemConfig'
import request from '@/utils/request'

interface MCPTool {
  name: string
  description: string
}

interface MCPConn {
  name: string
  transport: string
  url: string
  token: string
  enabled: boolean
  allowed_tools: string[]
  _loadingTools?: boolean
  _tools?: MCPTool[]
  _toolsError?: string
  _allChecked?: boolean
  _indeterminate?: boolean
}

const loading = ref(false)
const submitting = ref(false)
const groupEnabled = ref(false)
const conns = ref<MCPConn[]>([])

const loadConfig = async () => {
  loading.value = true
  try {
    const { data } = await getSystemConfig()
    const raw = data.data as any
    groupEnabled.value = String(raw['external_mcp:group_enabled']) === 'true'
    const rawConns = raw.external_mcp
    if (rawConns) {
      const parsed: MCPConn[] = typeof rawConns === 'string' ? JSON.parse(rawConns) : rawConns
      conns.value = Array.isArray(parsed) ? parsed.map(c => ({
        ...c,
        allowed_tools: c.allowed_tools || [],
        _tools: undefined,
        _toolsError: undefined,
        _allChecked: false,
        _indeterminate: false,
      })) : []
    } else {
      conns.value = []
    }
  } catch (e) {
    conns.value = []
  } finally {
    loading.value = false
  }
}

const addConn = () => {
  conns.value.push({
    name: '',
    transport: 'streamable-http',
    url: '',
    token: '',
    enabled: true,
    allowed_tools: [],
    _tools: undefined,
    _toolsError: undefined,
    _allChecked: false,
    _indeterminate: false,
  })
}

const removeConn = (index: number) => {
  conns.value.splice(index, 1)
}

const discoverTools = async (index: number) => {
  const conn = conns.value[index]
  if (!conn.url) {
    ElMessage.warning('请先填写端点 URL')
    return
  }
  conn._loadingTools = true
  conn._toolsError = undefined
  try {
    const res = await request.post('/api/v1/admin/external-mcp/tools', {
      name: conn.name,
      transport: conn.transport,
      url: conn.url,
      token: conn.token,
    })
    const tools: MCPTool[] = res.data?.data || []
    conn._tools = tools
    // 自动全选（首次发现时默认全部开放）
    if (!conn.allowed_tools || conn.allowed_tools.length === 0) {
      conn.allowed_tools = tools.map(t => t.name)
    }
    updateCheckState(index)
    if (tools.length === 0) {
      conn._toolsError = '该 MCP Server 未提供任何工具'
    }
  } catch (e: any) {
    conn._tools = []
    conn._toolsError = e?.response?.data?.message || e?.message || '连接失败'
  } finally {
    conn._loadingTools = false
  }
}

const toggleAllTools = (index: number, checked: boolean) => {
  const conn = conns.value[index]
  if (checked) {
    conn.allowed_tools = (conn._tools || []).map(t => t.name)
  } else {
    conn.allowed_tools = []
  }
  updateCheckState(index)
}

const updateAllowedTools = (index: number, val: string[]) => {
  const conn = conns.value[index]
  conn.allowed_tools = val
  updateCheckState(index)
}

const updateCheckState = (index: number) => {
  const conn = conns.value[index]
  const total = conn._tools?.length || 0
  const selected = conn.allowed_tools?.length || 0
  conn._allChecked = total > 0 && selected === total
  conn._indeterminate = selected > 0 && selected < total
}

const handleSave = async () => {
  if (conns.value.filter(c => c.enabled).some(c => !c.name.trim() || !c.url.trim())) {
    ElMessage.warning('每条启用中的连接都需要填写名称与端点 URL')
    return
  }
  submitting.value = true
  try {
    const cleaned = conns.value
      .map(c => {
        const conn: any = {
          name: c.name.trim(),
          transport: c.transport,
          url: c.url.trim(),
          token: c.token.trim(),
          enabled: Boolean(c.enabled),
        }
        // 只保存非空的 allowed_tools（空 = 全部开放）
        if (c.allowed_tools && c.allowed_tools.length > 0 && c._tools) {
          // 若已选数 = 总数，不保存 allowed_tools（等同全部开放）
          if (c.allowed_tools.length < c._tools.length) {
            conn.allowed_tools = c.allowed_tools
          }
        }
        return conn
      })
      .filter(c => c.enabled)
    await updateSystemConfig({
      'external_mcp': JSON.stringify(cleaned),
      'external_mcp:group_enabled': groupEnabled.value ? 'true' : 'false'
    } as any)
    ElMessage.success('外部 MCP 配置已保存，后台热同步中')
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    submitting.value = false
  }
}

onMounted(loadConfig)
</script>

<style scoped>
.mcp-config-section {
  margin-top: 20px;
  max-width: 720px;
}

.section-desc {
  color: var(--color-text-muted, #909399);
  font-size: 13px;
  line-height: 1.6;
  margin: 0 0 16px;
}

.conn-card {
  border: 1px solid var(--color-border-light, #ebeef5);
  border-radius: 6px;
  padding: 12px 16px 0;
  margin-bottom: 12px;
  background: var(--color-bg-light, #fafafa);
}

.conn-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.conn-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 16px;
}

.desc {
  color: var(--color-text-muted, #909399);
  font-size: 12px;
}

.tool-discovery {
  padding: 8px 0 12px;
  border-top: 1px dashed var(--color-border-light, #ebeef5);
  margin-top: 8px;
}

.tool-discovery-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.tool-count {
  font-size: 12px;
  color: var(--color-text-muted, #909399);
}

.tool-error {
  margin-top: 8px;
}

.tool-list {
  margin-top: 8px;
}

.tool-checkbox-all {
  font-size: 13px;
  font-weight: 500;
}

.tool-item {
  padding: 4px 0;
  display: flex;
  align-items: flex-start;
}

.tool-name {
  font-family: monospace;
  font-size: 13px;
  margin-right: 8px;
  color: var(--color-primary, #409eff);
}

.tool-desc {
  font-size: 12px;
  color: var(--color-text-muted, #909399);
  line-height: 1.4;
}
</style>
