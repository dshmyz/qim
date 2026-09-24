<template>
  <div class="avatar-draft-card" :data-testid="`avatar-draft-${draft.id}`">
    <div class="avatar-draft-card__preview">{{ draft.content }}</div>
    <div class="avatar-draft-card__meta">
      <i class="fas fa-robot"></i>
      <span>分身拟回复 · 未通过核验<template v-if="draft.reason">：{{ draft.reason }}</template></span>
    </div>
    <div class="avatar-draft-card__actions">
      <button type="button" class="draft-action draft-action--primary" :disabled="busy" @click="onApprove">
        发送
      </button>
      <button type="button" class="draft-action" :disabled="busy" @click="onEdit">
        编辑后发送
      </button>
      <button type="button" class="draft-action" :disabled="busy" @click="onDiscard">
        丢弃
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { AvatarReplyDraft } from '../../types/avatar'
import { useAvatarDraftApproval } from '../../composables/useAvatarDraftApproval'
import QMessage from '../../utils/qmessage'

const props = defineProps<{ draft: AvatarReplyDraft }>()

const { approve, discard, requestEdit } = useAvatarDraftApproval()
const busy = ref(false)

async function run(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  try {
    await action()
  } catch (e: any) {
    busy.value = false
    QMessage.error(e?.message || '操作失败，请重试')
  }
}

function onApprove() {
  run(() => approve(props.draft.conversation_id, props.draft.id))
}

function onDiscard() {
  run(() => discard(props.draft.conversation_id, props.draft.id))
}

function onEdit() {
  requestEdit(props.draft)
}
</script>

<style scoped>
/* 左缩进对齐消息气泡列：40px 头像 + 12px 间距 */
.avatar-draft-card {
  margin: 4px 0 8px 52px;
  padding: 10px 12px;
  border: 1px dashed var(--border-color, #cbd5e1);
  border-radius: 10px;
  background: var(--bg-secondary, rgba(0, 0, 0, 0.02));
  max-width: 620px;
}

.avatar-draft-card__preview {
  font-size: var(--font-size-sm);
  line-height: 1.6;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

.avatar-draft-card__meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
}

.avatar-draft-card__actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.draft-action {
  padding: 3px 12px;
  border: 1px solid var(--border-color, #cbd5e1);
  border-radius: 6px;
  background: transparent;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  cursor: pointer;
}

.draft-action:hover:not(:disabled) {
  border-color: #3b82f6;
  color: #3b82f6;
}

.draft-action--primary {
  border-color: #3b82f6;
  color: #3b82f6;
}

.draft-action:disabled {
  opacity: 0.5;
  cursor: default;
}
</style>
