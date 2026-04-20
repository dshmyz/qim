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
  width: 60px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 10px;
}

.option-item {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-radius: 8px;
  margin-bottom: 4px;
  transition: all 0.2s ease;
  position: relative;
}

.option-item:hover {
  background: var(--hover-bg);
}

.option-item.active {
  background: var(--primary-color);
}

.option-item.active .option-icon {
  color: white;
}

.option-icon {
  font-size: 18px;
  color: var(--text-secondary);
}

.option-spacer {
  flex: 1;
}

.settings-option {
  margin-top: auto;
  margin-bottom: 10px;
}
</style>
