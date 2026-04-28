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
              <div class="avatar-setting">
                <div class="current-avatar">
                  <img :src="currentUserAvatar" :alt="currentUser?.username || 'avatar'" />
                  <button class="change-avatar-btn">更换</button>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>昵称</label>
              <input type="text" v-model="localProfile.nickname" class="settings-input" />
            </div>
            <div class="settings-item">
              <label>账号</label>
              <span class="settings-value">{{ currentUser?.username || '' }}</span>
            </div>
            <div class="settings-item">
              <label>签名</label>
              <textarea v-model="localProfile.signature" class="settings-textarea" placeholder="输入个人签名"></textarea>
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
              <button class="clear-cache-btn" @click="$emit('clearCache')">清除</button>
            </div>
            <div class="settings-item">
              <label>双因素认证</label>
              <label class="switch"><input type="checkbox" v-model="localAdvancedSettings.twoFactorEnabled" @change="$emit('saveTwoFactor', $event.target.checked)" /><span class="slider round"></span></label>
              <div class="settings-hint">开启后，下次登录需要输入验证码</div>
            </div>
            <div class="settings-item">
              <label>账号安全</label>
              <button class="security-btn" @click="$emit('openSecurity')">查看</button>
            </div>
            <div class="settings-item">
              <label>关于</label>
              <div class="about-info"><span>版本：1.0.0</span></div>
            </div>
          </div>
          
          <div v-if="localTab === 'file'" class="settings-section">
            <div class="settings-section-header"><h4>文件设置</h4></div>
            <div class="settings-item">
              <label>默认保存目录</label>
              <div class="file-path-setting">
                <input type="text" v-model="localFileSettings.defaultSaveDirectory" class="settings-input" placeholder="选择默认保存目录" />
                <button class="browse-btn" @click="$emit('browseDirectory')">浏览</button>
              </div>
              <div class="settings-hint">设置接收文件的默认保存位置</div>
            </div>
            <div class="settings-item">
              <label>文件自动下载</label>
              <label class="switch"><input type="checkbox" v-model="localFileSettings.autoDownload" /><span class="slider round"></span></label>
            </div>
            <div class="settings-item">
              <label>最大上传文件大小</label>
              <div class="file-size-setting">
                <input type="number" v-model.number="localFileSettings.maxFileSize" class="settings-input" placeholder="文件大小限制" />
                <span class="size-unit">MB</span>
              </div>
              <div class="settings-hint">设置单个文件的最大上传大小</div>
            </div>
            <div class="settings-item">
              <label>允许的文件类型</label>
              <input type="text" v-model="localFileSettings.allowedFileTypes" class="settings-input" placeholder="例如：jpg,png,pdf,doc" />
              <div class="settings-hint">设置允许上传的文件类型，用逗号分隔</div>
            </div>
            <div class="settings-item">
              <label>图片自动预览</label>
              <label class="switch"><input type="checkbox" v-model="localFileSettings.autoPreviewImages" /><span class="slider round"></span></label>
            </div>
            <div class="settings-item">
              <label>文件历史记录</label>
              <label class="switch"><input type="checkbox" v-model="localFileSettings.enableFileHistory" /><span class="slider round"></span></label>
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
  if (!props.currentUser?.avatar) return 'https://api.dicebear.com/7.x/avataaars/svg?seed=me'
  if (props.currentUser.avatar.startsWith('http')) return props.currentUser.avatar
  return props.serverUrl + props.currentUser.avatar
})

const save = () => {
  emit('save', {
    profile: { ...localProfile.value },
    messageSettings: { ...localMessageSettings.value },
    appearanceSettings: { ...localAppearanceSettings.value }
  })
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
  background: var(--modal-bg, #fff);
  border-radius: 12px;
  width: 800px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.settings-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color, #eee);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.settings-header h3 {
  margin: 0;
  font-size: 20px;
  color: var(--text-color, #333);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary, #999);
}

.settings-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.settings-sidebar {
  width: 200px;
  background: var(--sidebar-bg, #f5f5f5);
  padding: 16px 0;
}

.settings-sidebar-item {
  padding: 12px 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-color, #333);
}

.settings-sidebar-item:hover,
.settings-sidebar-item.active {
  background: var(--hover-color, #e8e8e8);
}

.settings-main {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.settings-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.settings-section-header h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: var(--text-color, #333);
}

.settings-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.settings-item label {
  min-width: 120px;
  color: var(--text-color, #333);
}

.settings-input,
.settings-textarea,
.settings-select {
  padding: 8px 12px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  flex: 1;
}

.settings-textarea {
  min-height: 80px;
  resize: vertical;
}

.avatar-setting .current-avatar {
  position: relative;
  display: inline-block;
}

.avatar-setting img {
  width: 80px;
  height: 80px;
  border-radius: 50%;
}

.change-avatar-btn {
  position: absolute;
  bottom: 0;
  right: 0;
  padding: 4px 8px;
  border: none;
  border-radius: 4px;
  background: var(--primary-color, #409eff);
  color: white;
  cursor: pointer;
}

.switch {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 24px;
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
  background-color: var(--primary-color, #409eff);
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
  gap: 12px;
  flex: 1;
  overflow-x: auto;
}

.theme-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  padding: 8px;
  border-radius: 8px;
}

.theme-option.active {
  background: var(--hover-color, #e8e8e8);
}

.theme-preview {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 2px solid var(--border-color, #ddd);
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
}

.font-size-value {
  min-width: 40px;
  color: var(--text-color, #333);
}

.settings-hint {
  font-size: 12px;
  color: var(--text-secondary, #999);
  width: 100%;
  margin-top: 4px;
}

.clear-cache-btn,
.security-btn {
  padding: 6px 16px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  background: var(--btn-bg, #f5f5f5);
  cursor: pointer;
  color: var(--text-color, #333);
}

.file-path-setting {
  display: flex;
  gap: 8px;
  flex: 1;
}

.browse-btn {
  padding: 6px 16px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  background: var(--btn-bg, #f5f5f5);
  cursor: pointer;
  color: var(--text-color, #333);
}

.file-size-setting {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.size-unit {
  color: var(--text-color, #333);
}

.settings-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, #eee);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-btn,
.save-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.cancel-btn {
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
}

.save-btn {
  background: var(--primary-color, #409eff);
  color: white;
}
</style>
