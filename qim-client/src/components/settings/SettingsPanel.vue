<template>
  <div v-if="visible" class="settings-modal" @click="$emit('close')">
    <div class="settings-content" @click.stop>
      <div class="settings-header">
        <h3>系统设置</h3>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>
      <div class="settings-body">
        <div class="settings-sidebar">
          <div class="settings-sidebar-item" :class="{ active: localTab === 'basic' }" @click="localTab = 'basic'">
            <i class="fas fa-user"></i>
            <span>基本设置</span>
          </div>
          <div class="settings-sidebar-item" :class="{ active: localTab === 'message' }" @click="localTab = 'message'">
            <i class="fas fa-comment"></i>
            <span>消息设置</span>
          </div>
          <div class="settings-sidebar-item" :class="{ active: localTab === 'appearance' }" @click="localTab = 'appearance'">
            <i class="fas fa-paint-brush"></i>
            <span>外观设置</span>
          </div>
          <div class="settings-sidebar-item" :class="{ active: localTab === 'advanced' }" @click="localTab = 'advanced'">
            <i class="fas fa-cog"></i>
            <span>高级设置</span>
          </div>
          <div class="settings-sidebar-item" :class="{ active: localTab === 'file' }" @click="localTab = 'file'">
            <i class="fas fa-file"></i>
            <span>文件设置</span>
          </div>
        </div>
        <div class="settings-main">
          <div v-if="localTab === 'basic'" class="settings-section">
            <div class="settings-section-header"><h4>个人信息</h4></div>
            <div class="settings-item">
              <label>头像</label>
              <div class="settings-item-content">
                <div class="avatar-setting">
                  <div class="current-avatar">
                    <img :src="currentUserAvatar" :alt="currentUser?.username || 'avatar'" />
                    <button class="change-avatar-btn">更换</button>
                  </div>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>昵称</label>
              <div class="settings-item-content">
                <input type="text" v-model="localProfile.nickname" class="settings-input" />
              </div>
            </div>
            <div class="settings-item">
              <label>账号</label>
              <div class="settings-item-content">
                <span class="settings-value">{{ currentUser?.username || '' }}</span>
              </div>
            </div>
            <div class="settings-item">
              <label>签名</label>
              <div class="settings-item-content">
                <textarea v-model="localProfile.signature" class="settings-textarea" placeholder="输入个人签名"></textarea>
              </div>
            </div>
          </div>
          
          <div v-if="localTab === 'message'" class="settings-section">
            <div class="settings-section-header"><h4>消息通知</h4></div>
            <div class="settings-item">
              <label>开启消息通知</label>
              <label class="switch"><input type="checkbox" v-model="localMessageSettings.notificationsEnabled" /><span class="slider round"></span></label>
            </div>
            <div class="settings-item">
              <label>声音提醒</label>
              <label class="switch"><input type="checkbox" v-model="localMessageSettings.soundEnabled" /><span class="slider round"></span></label>
            </div>
            <div class="settings-item">
              <label>桌面通知</label>
              <label class="switch"><input type="checkbox" v-model="localMessageSettings.desktopNotificationsEnabled" /><span class="slider round"></span></label>
            </div>
            <div class="settings-item">
              <label>消息免打扰</label>
              <div class="dnd-setting">
                <select v-model="localMessageSettings.dndMode" class="settings-select">
                  <option value="none">关闭</option>
                  <option value="work">工作时间</option>
                  <option value="custom">自定义</option>
                </select>
              </div>
            </div>
          </div>
          
          <div v-if="localTab === 'appearance'" class="settings-section">
            <div class="settings-section-header"><h4>主题设置</h4></div>
            <div class="settings-item">
              <label>主题</label>
              <div class="theme-selector">
                <div v-for="theme in themes" :key="theme.id" class="theme-option" :class="{ active: localAppearanceSettings.theme === theme.id }" @click="localAppearanceSettings.theme = theme.id">
                  <div :class="['theme-preview', theme.previewClass]"></div>
                  <span>{{ theme.name }}</span>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>字体大小</label>
              <div class="font-size-slider">
                <input type="range" v-model.number="localAppearanceSettings.fontSize" min="12" max="18" step="1" />
                <span class="font-size-value">{{ localAppearanceSettings.fontSize }}px</span>
              </div>
            </div>
          </div>
          
          <div v-if="localTab === 'advanced'" class="settings-section">
            <div class="settings-section-header"><h4>高级设置</h4></div>
            <div class="settings-item">
              <label>清除缓存</label>
              <div class="settings-item-content">
                <button class="action-btn" @click="$emit('clearCache')">清除缓存</button>
              </div>
            </div>
            <div class="settings-item">
              <label>双因素认证</label>
              <div class="settings-item-content">
                <div class="setting-row">
                  <label class="switch">
                    <input type="checkbox" v-model="localAdvancedSettings.twoFactorEnabled" @change="handleTwoFactorChange" />
                    <span class="slider round"></span>
                  </label>
                </div>
                <div class="settings-hint">开启后，下次登录需要输入验证码</div>
              </div>
            </div>
            <div class="settings-item">
              <label>账号安全</label>
              <div class="settings-item-content">
                <button class="action-btn" @click="$emit('openSecurity')">查看安全设置</button>
              </div>
            </div>
            <div class="settings-item">
              <label>关于</label>
              <div class="settings-item-content">
                <div class="about-info">
                  <span class="version-badge">v1.0.0</span>
                  <span class="about-text">当前为最新版本</span>
                </div>
              </div>
            </div>
          </div>
          
          <div v-if="localTab === 'file'" class="settings-section">
            <div class="settings-section-header"><h4>文件设置</h4></div>
            <div class="settings-item">
              <label>默认保存目录</label>
              <div class="settings-item-content">
                <div class="input-with-btn">
                  <input type="text" v-model="localFileSettings.defaultSaveDirectory" class="settings-input" placeholder="选择默认保存目录" />
                  <button class="browse-btn" @click="$emit('browseDirectory')">
                    <i class="fas fa-folder-open"></i>
                    <span>浏览</span>
                  </button>
                </div>
                <div class="settings-hint">设置接收文件的默认保存位置</div>
              </div>
            </div>
            <div class="settings-item">
              <label>文件自动下载</label>
              <div class="settings-item-content">
                <div class="setting-row">
                  <label class="switch">
                    <input type="checkbox" v-model="localFileSettings.autoDownload" />
                    <span class="slider round"></span>
                  </label>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>最大上传文件大小</label>
              <div class="settings-item-content">
                <div class="input-with-unit">
                  <input type="number" v-model.number="localFileSettings.maxFileSize" class="settings-input" placeholder="文件大小限制" />
                  <span class="size-unit">MB</span>
                </div>
                <div class="settings-hint">设置单个文件的最大上传大小</div>
              </div>
            </div>
            <div class="settings-item">
              <label>允许的文件类型</label>
              <div class="settings-item-content">
                <input type="text" v-model="localFileSettings.allowedFileTypes" class="settings-input" placeholder="例如：jpg,png,pdf,doc" />
                <div class="settings-hint">设置允许上传的文件类型，用逗号分隔</div>
              </div>
            </div>
            <div class="settings-item">
              <label>图片自动预览</label>
              <div class="settings-item-content">
                <div class="setting-row">
                  <label class="switch">
                    <input type="checkbox" v-model="localFileSettings.autoPreviewImages" />
                    <span class="slider round"></span>
                  </label>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>文件历史记录</label>
              <div class="settings-item-content">
                <div class="setting-row">
                  <label class="switch">
                    <input type="checkbox" v-model="localFileSettings.enableFileHistory" />
                    <span class="slider round"></span>
                  </label>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="settings-footer">
        <button class="cancel-btn" @click="$emit('close')">取消</button>
        <button class="save-btn" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'

interface Theme {
  id: string
  name: string
  previewClass: string
}

const themes: Theme[] = [
  { id: 'modern-light', name: '清新白', previewClass: 'light-theme' },
  { id: 'elegant-dark', name: '炫酷黑', previewClass: 'dark-theme' },
  { id: 'ocean-blue', name: '海洋蓝', previewClass: 'netblue-theme' },
  { id: 'elegant-purple', name: '高雅紫', previewClass: 'elegant-purple-theme' },
  { id: 'warm-amber', name: '琥珀黄', previewClass: 'warm-amber-theme' },
  { id: 'crimson-red', name: '中国红', previewClass: 'crimson-red-theme' },
  { id: 'emerald-green', name: '翡翠绿', previewClass: 'emerald-green-theme' },
  { id: 'mediterranean-dream', name: '地中海', previewClass: 'mediterranean-dream-theme' },
  { id: 'monochrome-elegance', name: '单色雅', previewClass: 'monochrome-elegance-theme' },
  { id: 'spring-blossom', name: '春日花', previewClass: 'spring-blossom-theme' }
]

interface Props {
  visible: boolean
  currentUser?: { username?: string; avatar?: string }
  serverUrl: string
  profile: { nickname?: string; signature?: string }
  messageSettings: { notificationsEnabled?: boolean; soundEnabled?: boolean; desktopNotificationsEnabled?: boolean; dndMode?: string }
  appearanceSettings: { theme?: string; fontSize?: number }
  advancedSettings: { twoFactorEnabled?: boolean }
  fileSettings: { defaultSaveDirectory?: string; autoDownload?: boolean; maxFileSize?: number; allowedFileTypes?: string; autoPreviewImages?: boolean; enableFileHistory?: boolean }
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'close': []
  'save': [data: { profile: any; messageSettings: any; appearanceSettings: any }]
  'clearCache': []
  'saveTwoFactor': [enabled: boolean]
  'openSecurity': []
  'browseDirectory': []
}>()

const localTab = ref('basic')
const localProfile = ref({ ...props.profile })
const localMessageSettings = ref({ ...props.messageSettings })
const localAppearanceSettings = ref({ ...props.appearanceSettings })
const localAdvancedSettings = ref({ ...props.advancedSettings })
const localFileSettings = ref({ ...props.fileSettings })

watch(() => props.visible, (val) => {
  if (val) {
    localTab.value = 'basic'
    localProfile.value = { ...props.profile }
    localMessageSettings.value = { ...props.messageSettings }
    localAppearanceSettings.value = { ...props.appearanceSettings }
    localAdvancedSettings.value = { ...props.advancedSettings }
    localFileSettings.value = { ...props.fileSettings }
  }
})

const currentUserAvatar = computed(() => {
  if (!props.currentUser?.avatar) return generateAvatar('me')
  if (isAbsoluteUrl(props.currentUser.avatar)) return props.currentUser.avatar
  return props.serverUrl + props.currentUser.avatar
})

const save = () => {
  emit('save', {
    profile: { ...localProfile.value },
    messageSettings: { ...localMessageSettings.value },
    appearanceSettings: { ...localAppearanceSettings.value }
  })
}

const handleTwoFactorChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  emit('saveTwoFactor', target.checked)
}
</script>

<style scoped>
.settings-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.settings-content {
  background: var(--modal-bg);
  border-radius: 16px;
  width: 800px;
  max-width: calc(100vw - 40px);
  height: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

.settings-header {
  padding: 24px 28px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.settings-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.close-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.settings-body {
  display: flex;
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.settings-sidebar {
  width: 200px;
  flex-shrink: 0;
  background: var(--sidebar-bg);
  padding: 12px 0;
  border-right: 1px solid var(--border-color);
}

.settings-sidebar-item {
  padding: 12px 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-color);
  font-size: 14px;
  transition: all 0.2s;
  border-left: 3px solid transparent;
}

.settings-sidebar-item:hover {
  background: var(--hover-color);
}

.settings-sidebar-item.active {
  background: var(--hover-color);
  border-left-color: var(--primary-color);
  color: var(--primary-color);
  font-weight: 500;
}

.settings-sidebar-item i {
  width: 20px;
  text-align: center;
}

.settings-main {
  flex: 1;
  overflow-y: auto;
  padding: 28px;
}

.settings-section {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.settings-section-header {
  margin-bottom: 8px;
}

.settings-section-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.settings-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  min-height: 40px;
}

.settings-item label {
  min-width: 100px;
  flex-shrink: 0;
  color: var(--text-color);
  font-size: 14px;
  padding-top: 10px;
  font-weight: 500;
}

.settings-item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 100%;
}

.settings-input,
.settings-textarea,
.settings-select {
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  flex: 1;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: border-color 0.2s;
}

.settings-input:focus,
.settings-textarea:focus,
.settings-select:focus {
  outline: none;
  border-color: var(--primary-color);
}

.settings-textarea {
  min-height: 80px;
  resize: vertical;
  font-family: inherit;
}

.settings-value {
  padding: 8px 0;
  color: var(--text-color);
  font-size: 14px;
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar-setting .current-avatar {
  position: relative;
  display: inline-block;
}

.avatar-setting img {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
}

.change-avatar-btn {
  position: absolute;
  bottom: 0;
  right: 0;
  padding: 4px 12px;
  border: none;
  border-radius: 12px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: 12px;
  transition: transform 0.2s;
}

.change-avatar-btn:hover {
  transform: scale(1.05);
}

.switch {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 24px;
  flex-shrink: 0;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
  position: absolute;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  transition: 0.4s;
  border-radius: 24px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.4s;
  border-radius: 50%;
}

input:checked + .slider {
  background-color: var(--primary-color);
}

input:checked + .slider:before {
  transform: translateX(26px);
}

.slider.round {
  border-radius: 24px;
}

.theme-selector {
  display: flex;
  flex-wrap: nowrap;
  gap: 16px;
  flex: 1;
  overflow-x: auto;
  padding: 8px 4px;
}

.theme-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 12px;
  border-radius: 12px;
  transition: all 0.2s;
}

.theme-option:hover {
  background: var(--hover-color);
}

.theme-option.active {
  background: var(--hover-color);
  box-shadow: 0 0 0 2px var(--primary-color);
}

.theme-preview {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 2px solid var(--border-color);
}

.theme-option span {
  font-size: 12px;
  color: var(--text-color);
}

.light-theme { background: #fff; }
.dark-theme { background: #333; }
.netblue-theme { background: #0078d4; }
.elegant-purple-theme { background: #6b4c9a; }
.warm-amber-theme { background: #d4893a; }
.crimson-red-theme { background: #c4352e; }
.emerald-green-theme { background: #2d8b4e; }
.mediterranean-dream-theme { background: #4a8aad; }
.monochrome-elegance-theme { background: #777; }
.spring-blossom-theme { background: #f0a1b9; }

.font-size-slider {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.font-size-slider input[type="range"] {
  flex: 1;
  cursor: pointer;
}

.font-size-value {
  min-width: 50px;
  color: var(--text-color);
  font-size: 14px;
}

.settings-hint {
  font-size: 12px;
  color: var(--text-secondary);
  width: 100%;
  margin-top: 6px;
  line-height: 1.4;
}

.action-btn {
  padding: 10px 20px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--btn-bg);
  cursor: pointer;
  color: var(--text-color);
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.action-btn:hover {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.clear-cache-btn,
.security-btn {
  padding: 10px 20px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--btn-bg);
  cursor: pointer;
  color: var(--text-color);
  font-size: 14px;
  transition: all 0.2s;
}

.clear-cache-btn:hover,
.security-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.browse-btn {
  padding: 10px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--btn-bg);
  cursor: pointer;
  color: var(--text-color);
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.browse-btn:hover {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.input-with-btn {
  display: flex;
  gap: 12px;
}

.input-with-btn .settings-input {
  flex: 1;
}

.input-with-unit {
  display: flex;
  align-items: center;
  gap: 12px;
}

.input-with-unit .settings-input {
  flex: 1;
}

.size-unit {
  color: var(--text-color);
  font-size: 14px;
  font-weight: 500;
}

.about-info {
  padding: 12px 0;
}

.version-badge {
  display: inline-block;
  padding: 4px 12px;
  background: var(--primary-color);
  color: white;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 500;
  margin-right: 12px;
}

.about-text {
  color: var(--text-secondary);
  font-size: 14px;
}

.settings-footer {
  padding: 16px 28px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
}

.cancel-btn,
.save-btn {
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
}

.cancel-btn {
  background: var(--btn-bg);
  color: var(--text-color);
}

.cancel-btn:hover {
  background: var(--hover-color);
}

.save-btn {
  background: var(--primary-color);
  color: white;
}

.save-btn:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}
</style>
