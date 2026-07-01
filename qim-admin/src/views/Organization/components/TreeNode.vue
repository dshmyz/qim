<template>
  <div class="tree-node">
    <div
      class="node-content"
      :class="{ 'is-selected': selectedId === node.id }"
      :style="{ paddingLeft: level * 24 + 'px' }"
      @click="emit('select', node)"
    >
      <div class="node-connector">
        <div v-if="level > 0" class="connector-line" :class="{ 'is-last': isLast }"></div>
        <div v-if="level > 0" class="connector-branch"></div>
      </div>

      <div v-if="hasChildren" class="expand-btn" @click.stop="toggleExpand">
        <el-icon :size="14" class="expand-icon" :class="{ 'is-expanded': isExpanded }">
          <ArrowRight />
        </el-icon>
      </div>
      <div v-else class="expand-placeholder"></div>

      <div class="node-icon">
        <el-icon><OfficeBuilding /></el-icon>
      </div>

      <div class="node-info">
        <span class="node-name">{{ node.name }}</span>
        <span class="node-code">{{ node.code }}</span>
      </div>

      <div class="node-actions">
        <el-button
          size="small"
          :icon="Top"
          circle
          :disabled="isFirst"
          @click.stop="emit('move', { node, direction: 'up' })"
        />
        <el-button
          size="small"
          :icon="Bottom"
          circle
          :disabled="isLast"
          @click.stop="emit('move', { node, direction: 'down' })"
        />
        <el-button
          size="small"
          :icon="Plus"
          circle
          @click.stop="emit('add-child', node)"
        />
        <el-button
          size="small"
          :icon="Delete"
          circle
          type="danger"
          plain
          @click.stop="emit('delete', node)"
        />
      </div>
    </div>

    <div v-if="hasChildren && isExpanded" class="node-children">
      <template v-for="(child, index) in children" :key="child.id">
        <TreeNode
          :node="child"
          :level="level + 1"
          :is-first="index === 0"
          :is-last="index === children.length - 1"
          :selected-id="selectedId"
          @select="emit('select', $event)"
          @add-child="emit('add-child', $event)"
          @delete="emit('delete', $event)"
          @move="emit('move', $event)"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ArrowRight, Plus, Delete, OfficeBuilding, Top, Bottom } from '@element-plus/icons-vue'
import type { Organization } from '@/types'

const props = defineProps<{
  node: Organization
  level: number
  isFirst: boolean
  isLast: boolean
  selectedId?: number
}>()

const emit = defineEmits<{
  select: [node: Organization]
  'add-child': [node: Organization]
  delete: [node: Organization]
  move: [payload: { node: Organization; direction: 'up' | 'down' }]
}>()

const isExpanded = ref(true)

const children = computed(() => {
  return (props.node as any).children || []
})

const hasChildren = computed(() => {
  return children.value.length > 0
})

const toggleExpand = () => {
  isExpanded.value = !isExpanded.value
}
</script>

<style scoped>
.tree-node {
  user-select: none;
}

.node-content {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
  position: relative;
}

.node-content:hover {
  background: var(--color-surface-hover);
}

.node-content.is-selected {
  background: var(--color-primary-lighter);
}

.node-connector {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 24px;
  pointer-events: none;
}

.connector-line {
  position: absolute;
  left: 11px;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--color-border);
}

.connector-line.is-last {
  bottom: 50%;
}

.connector-branch {
  position: absolute;
  left: 11px;
  top: 50%;
  width: 12px;
  height: 1px;
  background: var(--color-border);
}

.expand-btn {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  background: var(--color-surface-hover);
  flex-shrink: 0;
  transition: all var(--duration-fast);
}

.expand-btn:hover {
  background: var(--color-surface-active);
}

.expand-icon {
  transition: transform var(--duration-fast);
  color: var(--color-text-secondary);
}

.expand-icon.is-expanded {
  transform: rotate(90deg);
}

.expand-placeholder {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.node-icon {
  width: 28px;
  height: 28px;
  background: var(--gradient-primary);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.node-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.node-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.node-code {
  font-size: 11px;
  color: var(--color-text-muted);
}

.node-actions {
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity var(--duration-fast);
}

.node-content:hover .node-actions {
  opacity: 1;
}

.node-children {
  position: relative;
}

.node-children::before {
  content: '';
  position: absolute;
  left: 11px;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--color-border);
}
</style>
