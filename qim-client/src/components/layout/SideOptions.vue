<template>
  <div class="side-options">
    <div
      class="option-item"
      :class="{ active: activeOption === 'recent' }"
      @click="$emit('update:activeOption', 'recent')"
      title="最近联系人"
    >
      <span class="option-icon"><i class="fas fa-comment-dots"></i></span>
    </div>
    <div
      class="option-item"
      :class="{ active: activeOption === 'org' }"
      @click="$emit('update:activeOption', 'org')"
      title="组织架构"
    >
      <span class="option-icon"><i class="fas fa-sitemap"></i></span>
    </div>
    <div
      class="option-item"
      :class="{ active: activeOption === 'groups' }"
      @click="$emit('update:activeOption', 'groups')"
      title="群聊"
    >
      <span class="option-icon"><i class="fas fa-user-friends"></i></span>
    </div>
    <div
      class="option-item"
      :class="{ active: activeOption === 'apps' }"
      @click="$emit('update:activeOption', 'apps')"
      title="应用"
    >
      <span class="option-icon"><i class="fas fa-cube"></i></span>
    </div>
    <div
      class="option-item"
      @click="$emit('showMoreMenu', $event)"
      title="更多"
    >
      <span class="option-icon"><i class="fas fa-ellipsis-v"></i></span>
    </div>
    <div class="option-spacer"></div>
    <div class="option-item" @click="$emit('showThemeMenu', $event)" title="皮肤">
      <span class="option-icon"><i class="fas fa-palette"></i></span>
    </div>
    <div class="option-item settings-option" @click="$emit('showSettingsMenu', $event)" title="设置">
      <span class="option-icon"><i class="fas fa-cog"></i></span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  activeOption: string
}>()

defineEmits<{
  (e: 'update:activeOption', value: string): void
  (e: 'showMoreMenu', event: MouseEvent): void
  (e: 'showThemeMenu', event: MouseEvent): void
  (e: 'showSettingsMenu', event: MouseEvent): void
}>()
</script>

<style scoped>
.side-options {
  width: 72px;
  background: var(--list-bg);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
  gap: 20px;
  box-shadow: 1px 0 3px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
}

.side-options:hover {
  box-shadow: var(--shadow-md);
}

.option-spacer {
  flex: 0.9;
}

.settings-option {
  transition: none;
  position: relative;
}

.settings-option:hover {
  transform: translateY(-24px);
  transition: none;
  box-shadow: none;
  background: var(--hover-color);
  color: var(--primary-color);
  position: relative;
}

/* 选项项样式 */
.option-item {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.3s ease;
}

.option-item:hover {
  background: var(--hover-color);
  color: var(--primary-color);
  box-shadow: var(--shadow-sm);
  transform: translateY(-2px);
}

.option-item.active {
  background: var(--primary-color);
  color: white;
  box-shadow: var(--shadow-md);
}

.option-icon {
  font-size: 20px;
  position: relative;
}

.option-item:hover .option-icon,
.option-item.active .option-icon {
  opacity: 1;
  transform: scale(1.1);
}
</style>
