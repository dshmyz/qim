<template>
  <div class="render-rules-page">
    <el-card shadow="never">
      <!-- 顶部操作栏 -->
      <div class="action-bar">
        <div class="left">
          <h3>消息渲染增强规则</h3>
          <span class="version-tag" v-if="version">v{{ version }}</span>
        </div>
        <div class="right">
          <el-button type="primary" @click="handleCreate">新建规则</el-button>
          <el-button @click="fetchRules" :loading="loading">刷新</el-button>
        </div>
      </div>

      <!-- 规则列表 -->
      <el-table :data="rules" v-loading="loading" empty-text="暂无规则，点击「新建规则」添加">
        <el-table-column prop="name" label="名称" min-width="160">
          <template #default="{ row }">
            <span>{{ row.name }}</span>
            <el-tag size="small" type="info" style="margin-left: 8px">{{ row.id }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleToggleEnabled(row)" />
          </template>
        </el-table-column>
        <el-table-column prop="priority" label="优先级" width="90" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.render.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="正则" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="regex-code">{{ row.match.pattern }}</code>
          </template>
        </el-table-column>
        <el-table-column label="URL 模板" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="regex-code">{{ row.render.url_template }}</code>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row, $index }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" @click="handleTest(row)">测试</el-button>
            <el-popconfirm title="确定删除该规则吗？" @confirm="handleDelete($index)">
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 编辑对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      :title="editingRule.id && rules.find(r => r.id === editingRule.id) ? '编辑规则' : '新建规则'"
      width="720px"
      :close-on-click-modal="false"
    >
      <el-form :model="editingRule" label-width="120px" label-position="right">
        <el-divider content-position="left">基本信息</el-divider>
        <el-form-item label="ID" required>
          <el-input v-model="editingRule.id" placeholder="唯一标识，如 jira_ticket" :disabled="!!editingRule._isEdit" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="editingRule.name" placeholder="显示名称，如 Jira 工单卡片化" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="editingRule.enabled" />
        </el-form-item>
        <el-form-item label="优先级" required>
          <el-input-number v-model="editingRule.priority" :min="1" :max="100" />
          <span class="hint">数字越小越先执行（1-100）</span>
        </el-form-item>

        <el-divider content-position="left">作用域</el-divider>
        <el-form-item label="适用群组">
          <el-input v-model="groupInput" placeholder="群组 ID，逗号分隔，* 表示全部" />
        </el-form-item>
        <el-form-item label="排除群组">
          <el-input v-model="excludeGroupInput" placeholder="群组 ID，逗号分隔，支持通配 external_*" />
        </el-form-item>
        <el-form-item label="会话类型">
          <el-checkbox-group v-model="editingRule.scope.conversation_types">
            <el-checkbox label="single">单聊</el-checkbox>
            <el-checkbox label="group">群聊</el-checkbox>
            <el-checkbox label="discussion">讨论组</el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <el-divider content-position="left">匹配规则</el-divider>
        <el-form-item label="正则" required>
          <el-input v-model="editingRule.match.pattern" placeholder="如 \b([A-Z]{2,6})-(\d{1,6})\b" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="Flags">
          <el-input v-model="editingRule.match.flags" placeholder="如 g" style="width: 100px" />
        </el-form-item>
        <el-form-item label="捕获组映射">
          <div class="capture-groups">
            <div v-for="(item, idx) in captureGroupList" :key="idx" class="capture-group-item">
              <el-input v-model="item.name" placeholder="名称如 project" style="width: 160px" />
              <span class="arrow">→</span>
              <el-input-number v-model="item.index" :min="1" :max="9" style="width: 120px" />
              <el-button type="danger" size="small" circle @click="captureGroupList.splice(idx, 1)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <el-button size="small" @click="captureGroupList.push({ name: '', index: 1 })">+ 添加映射</el-button>
          </div>
        </el-form-item>

        <el-divider content-position="left">渲染配置</el-divider>
        <el-form-item label="类型" required>
          <el-select v-model="editingRule.render.type" style="width: 180px">
            <el-option label="链接卡片 (link_card)" value="link_card" />
            <el-option label="普通链接 (link)" value="link" />
            <el-option label="高亮标识 (text_chip)" value="text_chip" />
          </el-select>
        </el-form-item>
        <el-form-item label="URL 模板" required>
          <el-input v-model="editingRule.render.url_template" placeholder="如 http://jira.xxx.com/{{project}}/{{project}}-{{number}}" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="标签模板">
          <el-input v-model="editingRule.render.label_template" placeholder="如 {{project}}-{{number}}" />
        </el-form-item>
        <el-form-item label="标题模板">
          <el-input v-model="editingRule.render.title_template" placeholder="hover 提示，如 查看 Jira 工单 {{project}}-{{number}}" />
        </el-form-item>
        <el-form-item label="图标">
          <el-input v-model="editingRule.render.icon" placeholder="FontAwesome 类名，如 fab fa-jira" />
        </el-form-item>
        <el-form-item label="打开方式">
          <el-radio-group v-model="editingRule.render.target">
            <el-radio label="_blank">新窗口</el-radio>
            <el-radio label="_self">当前窗口</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="CSS 类">
          <el-input v-model="editingRule.render.class" placeholder="如 jira-ticket-card，仅字母数字连字符" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- 测试对话框 -->
    <el-dialog v-model="testDialogVisible" title="测试规则匹配" width="640px">
      <el-form label-width="100px">
        <el-form-item label="规则">
          <el-tag>{{ testingRule?.name }}</el-tag>
          <code class="regex-code" style="margin-left: 8px">{{ testingRule?.match.pattern }}</code>
        </el-form-item>
        <el-form-item label="样例文本">
          <el-input v-model="testSampleText" type="textarea" :rows="3" placeholder="输入样例文本，如 看一下 NI-30000 这个工单" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="runTest" :loading="testing">运行测试</el-button>
        </el-form-item>
        <el-form-item label="匹配结果" v-if="testResults.length">
          <div class="test-results">
            <div v-for="(r, i) in testResults" :key="i" class="test-result-item">
              <el-tag size="small">{{ r.matched }}</el-tag>
              <span class="arrow">→</span>
              <code class="regex-code">{{ r.url }}</code>
              <span class="arrow">|</span>
              <span>{{ r.label }}</span>
            </div>
          </div>
        </el-form-item>
        <el-form-item v-if="testResults.length === 0 && testRan">
          <el-alert title="未匹配到任何内容" type="info" :closable="false" />
        </el-form-item>
      </el-form>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import { getRenderRules, saveRenderRules, testRenderRule, type RenderRule, type TestRuleResult } from '@/api/renderRules'

// 编辑态临时标记字段
interface EditableRule extends RenderRule {
  _isEdit?: boolean
}

const rules = ref<RenderRule[]>([])
const loading = ref(false)
const saving = ref(false)
const version = ref(0)

// 编辑对话框状态
const editDialogVisible = ref(false)
const editingRule = reactive<EditableRule>(createEmptyRule())
const groupInput = ref('')
const excludeGroupInput = ref('')
const captureGroupList = ref<{ name: string; index: number }[]>([])

// 测试对话框状态
const testDialogVisible = ref(false)
const testingRule = ref<RenderRule | null>(null)
const testSampleText = ref('')
const testResults = ref<TestRuleResult[]>([])
const testing = ref(false)
const testRan = ref(false)

function createEmptyRule(): EditableRule {
  return {
    id: '',
    name: '',
    enabled: false,
    priority: 10,
    scope: { groups: ['*'], exclude_groups: [], conversation_types: ['single', 'group', 'discussion'] },
    match: { pattern: '', flags: 'g', capture_groups: {} },
    render: {
      type: 'link_card',
      url_template: '',
      label_template: '',
      title_template: '',
      icon: '',
      target: '_blank',
      class: '',
    },
    _isEdit: false,
  }
}

// 拉取规则
async function fetchRules() {
  loading.value = true
  try {
    const res = await getRenderRules()
    if (res.data?.code === 0 && res.data?.data) {
      rules.value = res.data.data.rules || []
      version.value = res.data.data.version || 0
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '拉取规则失败')
  } finally {
    loading.value = false
  }
}

// 新建规则
function handleCreate() {
  Object.assign(editingRule, createEmptyRule())
  groupInput.value = '*'
  excludeGroupInput.value = ''
  captureGroupList.value = []
  editDialogVisible.value = true
}

// 编辑规则
function handleEdit(row: RenderRule) {
  Object.assign(editingRule, JSON.parse(JSON.stringify(row)), { _isEdit: true })
  groupInput.value = row.scope.groups.join(',')
  excludeGroupInput.value = row.scope.exclude_groups.join(',')
  captureGroupList.value = Object.entries(row.match.capture_groups).map(([name, index]) => ({ name, index }))
  editDialogVisible.value = true
}

// 保存规则（单条保存到本地，统一提交全部）
async function handleSave() {
  if (!editingRule.id || !editingRule.name || !editingRule.match.pattern || !editingRule.render.url_template) {
    ElMessage.warning('请填写必填项（ID、名称、正则、URL 模板）')
    return
  }

  // 同步 capture_groups
  const cg: Record<string, number> = {}
  for (const item of captureGroupList.value) {
    if (item.name) cg[item.name] = item.index
  }
  editingRule.match.capture_groups = cg
  // 同步 groups
  editingRule.scope.groups = groupInput.value ? groupInput.value.split(',').map(s => s.trim()).filter(Boolean) : ['*']
  editingRule.scope.exclude_groups = excludeGroupInput.value
    ? excludeGroupInput.value.split(',').map(s => s.trim()).filter(Boolean)
    : []

  saving.value = true
  try {
    // 替换或追加
    const idx = rules.value.findIndex(r => r.id === editingRule.id)
    const { _isEdit, ...pureRule } = editingRule
    if (idx >= 0) {
      rules.value[idx] = pureRule
    } else {
      rules.value.push(pureRule)
    }
    // 统一保存全部
    const res = await saveRenderRules(rules.value)
    if (res.data?.code === 0) {
      ElMessage.success('保存成功')
      editDialogVisible.value = false
      await fetchRules()
    } else {
      ElMessage.error(res.data?.message || '保存失败')
    }
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

// 切换启用状态（统一保存全部）
async function handleToggleEnabled(_row: RenderRule) {
  try {
    await saveRenderRules(rules.value)
    ElMessage.success('状态已更新')
    await fetchRules()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '更新失败')
    await fetchRules()
  }
}

// 删除规则
async function handleDelete(index: number) {
  rules.value.splice(index, 1)
  try {
    await saveRenderRules(rules.value)
    ElMessage.success('删除成功')
    await fetchRules()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
    await fetchRules()
  }
}

// 打开测试对话框
function handleTest(row: RenderRule) {
  testingRule.value = row
  testSampleText.value = ''
  testResults.value = []
  testRan.value = false
  testDialogVisible.value = true
}

// 运行测试
async function runTest() {
  if (!testingRule.value || !testSampleText.value) {
    ElMessage.warning('请输入样例文本')
    return
  }
  testing.value = true
  testRan.value = false
  try {
    const res = await testRenderRule(testingRule.value, testSampleText.value)
    if (res.data?.code === 0 && res.data?.data) {
      testResults.value = res.data.data.results || []
    } else {
      ElMessage.error(res.data?.message || '测试失败')
    }
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '测试失败')
  } finally {
    testing.value = false
    testRan.value = true
  }
}

onMounted(() => {
  fetchRules()
})
</script>

<style scoped>
.render-rules-page {
  padding: 16px;
}
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.action-bar .left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.action-bar h3 {
  margin: 0;
  font-size: 16px;
}
.version-tag {
  color: #999;
  font-size: 12px;
}
.regex-code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #e6a23c;
}
.hint {
  margin-left: 8px;
  color: #999;
  font-size: 12px;
}
.capture-groups {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.capture-group-item {
  display: flex;
  align-items: center;
  gap: 8px;
}
.arrow {
  color: #999;
  margin: 0 4px;
}
.test-results {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.test-result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: #f5f7fa;
  border-radius: 4px;
}
</style>
