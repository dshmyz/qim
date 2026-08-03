<template>
  <!-- 命令列表候选项：图标 + trigger + 描述 -->
  <span class="cmd-list-item">
    <span v-if="entry.command.icon" class="cmd-list-item__icon">
      <i :class="entry.command.icon"></i>
    </span>
    <span class="cmd-list-item__trigger">{{ entry.command.trigger }}</span>
    <span class="cmd-list-item__title">{{ entry.command.title }}</span>
    <span v-if="entry.command.description" class="cmd-list-item__desc">
      {{ entry.command.description }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SlashCommand, SlashCommandItem } from '../../utils/slashCommand'

/** 命令列表条目：把 SlashCommand 包装成 SlashCommandItem（id=trigger）。 */
export interface CommandListEntry extends SlashCommandItem {
  id: string
  command: SlashCommand
}

const props = defineProps<{
  item: CommandListEntry
  active: boolean
}>()

const entry = computed(() => props.item)
</script>

<style scoped>
.cmd-list-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.cmd-list-item__icon {
  font-size: 13px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  width: 16px;
  color: var(--primary-color, #3b82f6);
}
.cmd-list-item__trigger {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color, #303133);
  flex-shrink: 0;
  font-family: monospace;
}
.cmd-list-item__title {
  font-size: 12px;
  color: var(--text-secondary, #909399);
  flex-shrink: 0;
}
.cmd-list-item__desc {
  font-size: 12px;
  color: var(--text-secondary, #909399);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
