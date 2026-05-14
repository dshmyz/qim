<template>
  <div class="message-bubble mini-app-message" :class="{ self: isSelf }">
    <div class="mini-app-info" @click="openMiniApp">
      <div class="mini-app-icon-container">
        <img :src="miniAppData?.icon" class="mini-app-icon" :alt="miniAppData?.name" />
        <div class="mini-app-type-label">小程序</div>
      </div>
      <div class="mini-app-details">
        <div class="mini-app-name">{{ miniAppData?.name }}</div>
        <div class="mini-app-description">{{ miniAppData?.description }}</div>
      </div>
      <div class="mini-app-arrow">
        <i class="fas fa-chevron-right"></i>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  miniAppData?: {
    icon: string
    name: string
    description: string
  }
  isSelf?: boolean
}>()

const emit = defineEmits<{
  open: [data: any]
}>()

const openMiniApp = () => {
  emit('open', props.miniAppData)
}
</script>

<style scoped>
.mini-app-message {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.95) 0%, rgba(255, 255, 255, 0.85) 100%);
  border-radius: 14px;
  padding: 14px;
  width: fit-content;
  min-width: 260px;
  max-width: 100%;
  box-shadow: 
    0 2px 8px rgba(0, 0, 0, 0.04),
    0 8px 24px rgba(0, 0, 0, 0.06),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-sizing: border-box;
  border: 1px solid rgba(0, 0, 0, 0.04);
  backdrop-filter: blur(10px);
  position: relative;
  overflow: hidden;
  cursor: pointer;
}

.mini-app-message::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, #4facfe, #00f2fe, #43e97b);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.mini-app-message:hover {
  box-shadow: 
    0 4px 12px rgba(0, 0, 0, 0.06),
    0 12px 32px rgba(0, 0, 0, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.9);
  transform: translateY(-2px);
}

.mini-app-message:hover::before {
  opacity: 1;
}

.mini-app-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.mini-app-icon-container {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.mini-app-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  object-fit: cover;
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  border: none;
  box-shadow: 
    0 4px 12px rgba(79, 172, 254, 0.25),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  display: block;
}

.mini-app-icon::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.2) 0%, transparent 100%);
  border-radius: 12px;
  z-index: 1;
}

.mini-app-icon-container:hover .mini-app-icon {
  transform: scale(1.08) rotate(-2deg);
  box-shadow: 
    0 6px 20px rgba(79, 172, 254, 0.35),
    inset 0 1px 0 rgba(255, 255, 255, 0.3);
}

.mini-app-type-label {
  font-size: 9px;
  font-weight: 500;
  color: #6b7280;
  background: rgba(107, 114, 128, 0.08);
  padding: 2px 6px;
  border-radius: 6px;
  display: block;
  text-align: center;
  white-space: nowrap;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.mini-app-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mini-app-name {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a2e;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: left;
  letter-spacing: -0.01em;
  line-height: 1.4;
}

.mini-app-description {
  font-size: 12px;
  color: #6b7280;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: left;
  font-weight: 500;
}

.mini-app-arrow {
  color: #9ca3af;
  font-size: 12px;
  flex-shrink: 0;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  align-self: center;
  margin-left: 4px;
}

.mini-app-message:hover .mini-app-arrow {
  color: #4facfe;
  transform: translateX(4px);
}

/* 自己的小程序消息样式 */
.mini-app-message.self {
  background: linear-gradient(135deg, #3182ce 0%, #4299e1 50%, #63b3ed 100%);
  border: none;
  box-shadow: 
    0 4px 12px rgba(49, 130, 206, 0.25),
    0 12px 32px rgba(66, 153, 225, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.15);
}

.mini-app-message.self::before {
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.3), rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.3));
}

.mini-app-message.self .mini-app-name {
  color: #ffffff;
  font-weight: 600;
}

.mini-app-message.self .mini-app-description {
  color: rgba(255, 255, 255, 0.85);
}

.mini-app-message.self .mini-app-icon {
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 
    0 4px 12px rgba(0, 0, 0, 0.15),
    inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.mini-app-message.self .mini-app-type-label {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.85);
}

.mini-app-message.self .mini-app-arrow {
  color: rgba(255, 255, 255, 0.8);
}

.mini-app-message.self:hover .mini-app-arrow {
  color: #ffffff;
}
</style>