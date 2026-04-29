<template>
  <div class="side-options">
    <div class="side-options-title">QIM</div>
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
      :class="{ active: activeOption === 'channels' }"
      @click="$emit('update:activeOption', 'channels')"
      title="频道"
    >
      <span class="option-icon"><i class="fas fa-bullhorn"></i></span>
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
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 60px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 0;
  padding-bottom: 16px;
  z-index: 200;
  flex-shrink: 0;
  gap: 20px;
}

.side-options-title {
  width: 100%;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  color: var(--side-options-title-color, var(--text-color));
  letter-spacing: 0.5px;
  user-select: none;
  -webkit-app-region: drag;
}

.option-item {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-radius: 8px;
  margin: 2px 0;
  transition: all 0.2s ease;
  position: relative;
  color: var(--text-secondary);
}

.option-item:hover {
  background: var(--hover-bg);
  color: var(--primary-color);
}

.option-item.active {
  background: var(--primary-color);
  color: white;
}

.option-item.active .option-icon {
  color: white;
}

.option-icon {
  font-size: 18px;
  color: inherit;
}

.option-spacer {
  flex: 1;
}

.settings-option {
  margin-top: auto;
  margin-bottom: 10px;
}

@media (max-width: 767px) {
  .side-options {
    position: relative;
    width: 100% !important;
    height: 60px !important;
    flex-direction: row;
    padding: 0 5px;
    gap: 16px;
    justify-content: space-around;
  }
  
  .side-options-title {
    display: none;
  }
  
  .option-spacer {
    display: none;
  }
  
  .settings-option {
    transform: none;
    margin-top: 0;
    margin-bottom: 0;

  }
  
  .settings-option:hover {
    transform: none;
  }
}
</style>
