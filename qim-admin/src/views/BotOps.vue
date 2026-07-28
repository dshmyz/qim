<template>
  <div class="bot-ops-page">
    <el-card shadow="never">
      <el-tabs v-model="activeTab">
        <!-- 投递监控 -->
        <el-tab-pane label="投递监控" name="deliveries">
          <div class="action-bar">
            <div class="left-actions">
              <el-form :model="filterForm" inline>
                <el-form-item label="状态">
                  <el-select v-model="filterForm.status" placeholder="全部状态" clearable style="width: 120px" @change="handleSearch">
                    <el-option label="待投递" value="pending" />
                    <el-option label="已投递" value="done" />
                    <el-option label="死信" value="dead" />
                  </el-select>
                </el-form-item>
                <el-form-item label="事件">
                  <el-select v-model="filterForm.event" placeholder="全部事件" clearable style="width: 160px" @change="handleSearch">
                    <el-option label="用户回复" value="bot.message" />
                    <el-option label="卡片动作" value="bot.card_action" />
                  </el-select>
                </el-form-item>
                <el-form-item label="Bot">
                  <el-select v-model="filterForm.bot_id" placeholder="全部 Bot" clearable filterable style="width: 180px" @change="handleSearch">
                    <el-option v-for="b in externalBots" :key="b.id" :label="b.name" :value="b.id" />
                  </el-select>
                </el-form-item>
              </el-form>
            </div>
            <el-button :icon="Refresh" @click="fetchDeliveries">刷新</el-button>
          </div>

          <el-table :data="deliveries" v-loading="loading" :row-class-name="deliveryRowClass">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="bot_name" label="Bot" min-width="120" show-overflow-tooltip />
            <el-table-column label="事件" width="120">
              <template #default="{ row }">
                <el-tag size="small" :type="row.event === 'bot.card_action' ? 'warning' : 'info'">
                  {{ eventLabel(row.event) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="attempts" label="尝试" width="70" align="center" />
            <el-table-column prop="created_at" label="创建时间" width="170" />
            <el-table-column label="下次重试" width="170">
              <template #default="{ row }">
                <span v-if="row.next_retry_at">{{ row.next_retry_at }}</span>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="最近错误" min-width="200" show-overflow-tooltip>
              <template #default="{ row }">
                <span v-if="row.last_error" class="error-text">{{ row.last_error }}</span>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openPayload(row)">查看 Payload</el-button>
                <el-popconfirm
                  title="确认重投该记录？"
                  @confirm="handleRedeliver(row)"
                >
                  <template #reference>
                    <el-button size="small" type="primary" :disabled="row.status === 'done'">重投</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>

          <div class="pagination-container">
            <el-pagination
              v-model:current-page="pagination.page"
              v-model:page-size="pagination.pageSize"
              :total="pagination.total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="fetchDeliveries"
              @current-change="fetchDeliveries"
            />
          </div>
        </el-tab-pane>

        <!-- 外部 Bot 列表 -->
        <el-tab-pane :label="`外部 Bot (${externalBotsTotal})`" name="bots">
          <div class="action-bar">
            <div class="left-actions">
              <el-form :model="botFilterForm" inline>
                <el-form-item label="名称">
                  <el-input v-model="botFilterForm.keyword" placeholder="搜索 Bot 名称" clearable style="width: 200px" @keyup.enter="handleBotSearch" />
                </el-form-item>
              </el-form>
            </div>
            <el-button :icon="Refresh" @click="fetchExternalBots">刷新</el-button>
          </div>

          <el-table :data="externalBots" v-loading="botsLoading">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="creator_name" label="创建者" width="140" />
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                  {{ row.is_active ? '启用' : '停用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="webhook_url" label="Webhook URL" min-width="240" show-overflow-tooltip />
            <el-table-column label="待投递" width="90" align="center">
              <template #default="{ row }">
                <el-badge v-if="row.pending_count > 0" :value="row.pending_count" type="warning" />
                <span v-else class="muted">0</span>
              </template>
            </el-table-column>
            <el-table-column label="死信" width="90" align="center">
              <template #default="{ row }">
                <el-badge v-if="row.dead_count > 0" :value="row.dead_count" type="danger" />
                <span v-else class="muted">0</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="170" />
          </el-table>

          <div class="pagination-container">
            <el-pagination
              v-model:current-page="botPagination.page"
              v-model:page-size="botPagination.pageSize"
              :total="botPagination.total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="fetchExternalBots"
              @current-change="fetchExternalBots"
            />
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- Payload 详情 -->
      <el-dialog v-model="payloadDialog" title="投递 Payload" width="700px">
        <div v-loading="payloadLoading">
          <el-descriptions :column="2" border size="small" class="payload-meta">
            <el-descriptions-item label="ID">{{ payloadDetail?.delivery.id }}</el-descriptions-item>
            <el-descriptions-item label="Bot">{{ payloadDetail?.bot_name }}</el-descriptions-item>
            <el-descriptions-item label="事件">{{ payloadDetail?.delivery.event }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ payloadDetail?.delivery.status }}</el-descriptions-item>
            <el-descriptions-item label="Webhook URL" :span="2">{{ payloadDetail?.delivery.webhook_url }}</el-descriptions-item>
          </el-descriptions>
          <pre class="payload-json">{{ prettyPayload }}</pre>
        </div>
      </el-dialog>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import {
  getWebhookDeliveries,
  getExternalBots,
  getWebhookDelivery,
  redeliverWebhook,
  type WebhookDelivery,
  type ExternalBot,
} from '@/api/botOps'

const activeTab = ref<'deliveries' | 'bots'>('deliveries')

// ===== 投递监控 =====
const filterForm = reactive({ status: '', event: '', bot_id: null as number | null })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const deliveries = ref<WebhookDelivery[]>([])
const loading = ref(false)

const fetchDeliveries = async () => {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: pagination.page, pageSize: pagination.pageSize }
    if (filterForm.status) params.status = filterForm.status
    if (filterForm.event) params.event = filterForm.event
    if (filterForm.bot_id) params.bot_id = filterForm.bot_id
    const { data } = await getWebhookDeliveries(params as any)
    deliveries.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchDeliveries()
}

const handleRedeliver = async (row: WebhookDelivery) => {
  try {
    const { data } = await redeliverWebhook(row.id)
    const result = data.data
    if (result.status === 'done') {
      ElMessage.success('重投成功，已投递')
    } else {
      ElMessage.warning(`已触发重投：当前状态 ${result.status}${result.last_error ? '，' + result.last_error : ''}`)
    }
    fetchDeliveries()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

const statusLabel = (s: string): string => ({ pending: '待投递', done: '已投递', dead: '死信' } as Record<string, string>)[s] || s
const statusType = (s: string): 'success' | 'warning' | 'danger' | 'info' => ({ pending: 'warning', done: 'success', dead: 'danger' } as const)[s as 'pending' | 'done' | 'dead'] || 'info'
const eventLabel = (e: string): string => ({ 'bot.message': '用户回复', 'bot.card_action': '卡片动作' } as Record<string, string>)[e] || e
const deliveryRowClass = ({ row }: { row: WebhookDelivery }): string => (row.status === 'dead' ? 'row-dead' : '')

// ===== Payload 详情 =====
const payloadDialog = ref(false)
const payloadLoading = ref(false)
const payloadDetail = ref<{ delivery: WebhookDelivery; bot_name: string } | null>(null)
const prettyPayload = computed(() => {
  if (!payloadDetail.value) return ''
  try {
    return JSON.stringify(JSON.parse(payloadDetail.value.delivery.payload), null, 2)
  } catch {
    return payloadDetail.value.delivery.payload
  }
})

const openPayload = async (row: WebhookDelivery) => {
  payloadDialog.value = true
  payloadLoading.value = true
  try {
    const { data } = await getWebhookDelivery(row.id)
    payloadDetail.value = data.data
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    payloadLoading.value = false
  }
}

// ===== 外部 Bot 列表 =====
const botFilterForm = reactive({ keyword: '' })
const botPagination = reactive({ page: 1, pageSize: 10, total: 0 })
const externalBots = ref<ExternalBot[]>([])
const externalBotsTotal = ref(0)
const botsLoading = ref(false)

const fetchExternalBots = async () => {
  botsLoading.value = true
  try {
    const params: Record<string, unknown> = { page: botPagination.page, pageSize: botPagination.pageSize }
    if (botFilterForm.keyword) params.keyword = botFilterForm.keyword
    const { data } = await getExternalBots(params as any)
    externalBots.value = data.data.list
    botPagination.total = data.data.total
    externalBotsTotal.value = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    botsLoading.value = false
  }
}

const handleBotSearch = () => {
  botPagination.page = 1
  fetchExternalBots()
}

onMounted(() => {
  fetchDeliveries()
  fetchExternalBots() // 同时拉一份给投递监控的 Bot 下拉
})
</script>

<style scoped>
.bot-ops-page {
  padding: 0;
}
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.left-actions :deep(.el-form-item) {
  margin-bottom: 0;
}
.pagination-container {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.muted {
  color: var(--el-text-color-placeholder);
}
.error-text {
  color: var(--el-color-danger);
}
.payload-json {
  margin-top: 12px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  font-size: 12px;
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
:deep(.row-dead) {
  background-color: var(--el-color-danger-light-9);
}
</style>
