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
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSystemConfig, updateSystemConfig } from '@/api/systemConfig'

interface MCPConn {
  name: string
  transport: string
  url: string
  token: string
  enabled: boolean
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
    groupEnabled.value = raw['external_mcp:group_enabled'] === 'true'
    const rawConns = raw.external_mcp
    if (rawConns) {
      const parsed: MCPConn[] = typeof rawConns === 'string' ? JSON.parse(rawConns) : rawConns
      conns.value = Array.isArray(parsed) ? parsed : []
    } else {
      conns.value = []
    }
  } catch (e) {
    // 配置未初始化或其他错误，使用默认空列表
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
    enabled: true
  })
}

const removeConn = (index: number) => {
  conns.value.splice(index, 1)
}

const handleSave = async () => {
  if (conns.value.some(c => !c.name.trim() || !c.url.trim())) {
    ElMessage.warning('每条启用中的连接都需要填写名称与端点 URL')
    return
  }
  submitting.value = true
  try {
    // 名称/token 去首尾空白后持久化；关闭的连接整体剔除（对应工具将从注册表摘除并热同步）
    const cleaned = conns.value
      .map(c => ({
        name: c.name.trim(),
        transport: c.transport,
        url: c.url.trim(),
        token: c.token.trim(),
        enabled: Boolean(c.enabled)
      }))
      .filter(c => c.enabled)
    await updateSystemConfig({
      'external_mcp': JSON.stringify(cleaned),
      'external_mcp:group_enabled': groupEnabled.value ? 'true' : 'false'
    } as any)
    ElMessage.success('外部 MCP 配置已保存并热同步')
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
</style>
