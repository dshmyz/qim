<template>
  <el-dialog
    :model-value="visible"
    title="用户分身配置"
    width="640px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <p class="avatar-config-hint">为用户 <strong>{{ username }}</strong> 管理数字分身配置</p>

    <div v-loading="loading">
      <el-empty v-if="!loading && !config" description="该用户尚未创建分身配置">
        <el-button type="primary" :loading="saving" @click="handleInitAndEnable">初始化并开启分身</el-button>
      </el-empty>

      <el-form v-else-if="config" label-width="120px" class="avatar-form">
        <el-divider content-position="left">基础信息</el-divider>
        <el-form-item label="分身名称">
          <el-input v-model="form.name" placeholder="分身名称" maxlength="100" />
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" active-text="已开启" inactive-text="已关闭" />
          <span class="field-tip">开启将写入已通过审批记录并通知用户</span>
        </el-form-item>

        <el-divider content-position="left">人设</el-divider>
        <el-form-item label="自动学习人设">
          <el-input
            :model-value="config.auto_learned_persona || '（暂无，用户使用后自动学习生成）'"
            type="textarea"
            :rows="3"
            readonly
          />
        </el-form-item>
        <el-form-item label="自定义人设补充">
          <el-input v-model="form.custom_persona_addon" type="textarea" :rows="3" placeholder="管理员可补充的人设描述" />
        </el-form-item>

        <el-divider content-position="left">触发策略</el-divider>
        <el-form-item label="触发模式">
          <el-select v-model="form.trigger_rules.mode" placeholder="选择触发模式">
            <el-option label="被@时" value="mention" />
            <el-option label="离线时" value="offline" />
            <el-option label="关键词" value="keyword" />
            <el-option label="全部消息" value="all" />
            <el-option label="智能判断" value="smart" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.trigger_rules.mode === 'keyword'" label="关键词">
          <el-input
            v-model="keywordsInput"
            placeholder="多个关键词用英文逗号分隔"
          />
        </el-form-item>
        <el-form-item label="接管冷却(分钟)">
          <el-input-number v-model="form.takeover_cooldown" :min="0" :max="1440" />
        </el-form-item>

        <el-divider content-position="left">回复策略</el-divider>
        <el-form-item label="回复长度">
          <el-select v-model="form.reply_strategy.maxReplyLength">
            <el-option label="简短" value="short" />
            <el-option label="中等" value="medium" />
            <el-option label="详细" value="long" />
          </el-select>
        </el-form-item>
        <el-form-item label="回复延迟(秒)">
          <el-input-number v-model="form.reply_strategy.replyDelay" :min="0" :max="60" />
        </el-form-item>
        <el-form-item label="置信度阈值">
          <el-input-number v-model="form.reply_strategy.confidenceThreshold" :min="0" :max="1" :precision="1" :step="0.1" />
        </el-form-item>
        <el-form-item label="免责声明样式">
          <el-select v-model="form.reply_strategy.disclaimerStyle">
            <el-option label="角标" value="badge" />
            <el-option label="脚注" value="footer" />
            <el-option label="两者" value="both" />
          </el-select>
        </el-form-item>
        <el-form-item label="回复范围外消息">
          <el-switch v-model="form.reply_strategy.replyOutOfScope" />
        </el-form-item>

        <el-divider content-position="left">知识范围</el-divider>
        <el-form-item label="可访问数据">
          <el-checkbox v-model="form.knowledge_scope.conversationHistory">会话历史</el-checkbox>
          <el-checkbox v-model="form.knowledge_scope.knowledgeDocs">知识文档</el-checkbox>
          <el-checkbox v-model="form.knowledge_scope.notes">笔记</el-checkbox>
          <el-checkbox v-model="form.knowledge_scope.tasks">任务</el-checkbox>
        </el-form-item>

        <el-divider content-position="left">模型绑定</el-divider>
        <el-form-item label="使用系统模型">
          <el-switch v-model="form.use_system_config" />
          <span class="field-tip">关闭后可指定用户自己的 AI 模型配置</span>
        </el-form-item>
        <el-form-item v-if="!form.use_system_config" label="模型配置ID">
          <el-input-number v-model="form.model_config_id" :min="1" placeholder="AI 配置 ID" />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="$emit('update:visible', false)">关闭</el-button>
      <el-button v-if="config" type="primary" :loading="saving" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { getUserAvatarConfig, updateUserAvatarConfig, type UpdateUserAvatarConfigParams } from '@/api/users'
import type { AdminAvatarConfig } from '@/types'

interface Props {
  visible: boolean
  userId: number
  username: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const loading = ref(false)
const saving = ref(false)
const config = ref<AdminAvatarConfig | null>(null)

const form = reactive({
  name: '',
  enabled: false,
  use_system_config: true,
  model_config_id: undefined as number | undefined,
  takeover_cooldown: 10,
  custom_persona_addon: '',
  trigger_rules: {
    mode: 'mention' as AdminAvatarConfig['trigger_rules']['mode'],
    keywords: [] as string[],
    timeRanges: [] as AdminAvatarConfig['trigger_rules']['timeRanges'],
    excludedConversations: [] as number[],
  },
  knowledge_scope: {
    conversationHistory: true,
    knowledgeDocs: false,
    notes: false,
    tasks: false,
  },
  reply_strategy: {
    maxReplyLength: 'medium' as AdminAvatarConfig['reply_strategy']['maxReplyLength'],
    replyDelay: 3,
    confidenceThreshold: 0.6,
    disclaimerStyle: 'badge' as AdminAvatarConfig['reply_strategy']['disclaimerStyle'],
    replyOutOfScope: false,
  },
})

const keywordsInput = computed({
  get: () => form.trigger_rules.keywords.join(', '),
  set: (val: string) => {
    form.trigger_rules.keywords = val.split(/[,，]/).map(s => s.trim()).filter(Boolean)
  },
})

const applyConfig = (cfg: AdminAvatarConfig) => {
  config.value = cfg
  form.name = cfg.name
  form.enabled = cfg.enabled
  form.use_system_config = cfg.use_system_config
  form.model_config_id = cfg.model_config_id ?? undefined
  form.takeover_cooldown = cfg.takeover_cooldown
  form.custom_persona_addon = cfg.custom_persona_addon || ''
  if (cfg.trigger_rules) {
    form.trigger_rules.mode = cfg.trigger_rules.mode || 'mention'
    form.trigger_rules.keywords = cfg.trigger_rules.keywords || []
    form.trigger_rules.timeRanges = cfg.trigger_rules.timeRanges || []
    form.trigger_rules.excludedConversations = cfg.trigger_rules.excludedConversations || []
  }
  if (cfg.knowledge_scope) {
    form.knowledge_scope = { ...form.knowledge_scope, ...cfg.knowledge_scope }
  }
  if (cfg.reply_strategy) {
    form.reply_strategy = { ...form.reply_strategy, ...cfg.reply_strategy }
  }
}

const fetchConfig = async () => {
  if (!props.userId) return
  loading.value = true
  config.value = null
  try {
    const { data } = await getUserAvatarConfig(props.userId)
    if (data.data) {
      applyConfig(data.data)
    }
  } catch (error) {
    console.error('[AvatarConfigDialog] fetch config failed:', error)
    ElMessage.error('获取分身配置失败')
  } finally {
    loading.value = false
  }
}

const buildPayload = (): UpdateUserAvatarConfigParams => ({
  name: form.name,
  enabled: form.enabled,
  use_system_config: form.use_system_config,
  model_config_id: form.use_system_config ? undefined : form.model_config_id,
  trigger_rules: form.trigger_rules,
  knowledge_scope: form.knowledge_scope,
  reply_strategy: form.reply_strategy,
  takeover_cooldown: form.takeover_cooldown,
  custom_persona_addon: form.custom_persona_addon,
})

const handleSave = async () => {
  if (!props.userId) return
  saving.value = true
  try {
    const { data } = await updateUserAvatarConfig(props.userId, buildPayload())
    if (data.data) applyConfig(data.data)
    ElMessage.success('分身配置已保存')
  } catch (error) {
    console.error('[AvatarConfigDialog] save config failed:', error)
    ElMessage.error('分身配置保存失败')
  } finally {
    saving.value = false
  }
}

const handleInitAndEnable = async () => {
  if (!props.userId) return
  saving.value = true
  try {
    const { data } = await updateUserAvatarConfig(props.userId, { enabled: true })
    if (data.data) applyConfig(data.data)
    ElMessage.success('已初始化并开启分身')
  } catch (error) {
    console.error('[AvatarConfigDialog] init failed:', error)
    ElMessage.error('初始化分身失败')
  } finally {
    saving.value = false
  }
}

watch(() => props.visible, (val) => {
  if (val && props.userId) {
    fetchConfig()
  }
})
</script>

<style scoped>
.avatar-config-hint {
  margin-bottom: 16px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.avatar-form {
  max-height: 60vh;
  overflow-y: auto;
  padding-right: 8px;
}

.field-tip {
  margin-left: 12px;
  font-size: 12px;
  color: var(--color-text-muted);
}
</style>
