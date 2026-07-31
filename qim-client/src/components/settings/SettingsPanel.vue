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
          <div class="settings-sidebar-item" :class="{ active: localTab === 'data-storage' }" @click="localTab = 'data-storage'">
            <i class="fas fa-database"></i>
            <span>数据与存储</span>
          </div>
          <div class="settings-sidebar-item" :class="{ active: localTab === 'shortcut' }" @click="localTab = 'shortcut'">
            <i class="fas fa-keyboard"></i>
            <span>快捷键</span>
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
                    <Avatar
                      :src="currentUser?.avatar"
                      :name="currentUser?.username || '用户'"
                      :server-url="serverUrl"
                      :alt="currentUser?.username || 'avatar'"
                      size="lg"
                    />
                    <button class="change-avatar-btn" @click="triggerAvatarUpload">更换</button>
                  </div>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>姓名</label>
              <div class="settings-item-content">
                <span class="settings-value">{{ localProfile.nickname || '' }}</span>
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
                  <option value="all_day">全天免打扰</option>
                  <option value="custom">自定义时间段</option>
                </select>
              </div>
            </div>
            <div v-if="localMessageSettings.dndMode === 'custom'" class="settings-item">
              <label>免打扰时间</label>
              <div class="dnd-time-range">
                <input v-model="localMessageSettings.dndStartTime" type="time" class="settings-select" />
                <span>至</span>
                <input v-model="localMessageSettings.dndEndTime" type="time" class="settings-select" />
              </div>
            </div>
            <!-- C1: 发送方式 -->
            <div class="settings-item">
              <label>发送方式</label>
              <div class="settings-item-content">
                <select v-model="localMessageSettings.sendShortcut" class="settings-select" data-testid="send-shortcut-select">
                  <option value="enter">Enter 发送（Shift+Enter 换行）</option>
                  <option value="ctrl_enter">Ctrl+Enter 发送（Enter 换行）</option>
                </select>
              </div>
            </div>
            <!-- C2: @提及强提醒 -->
            <div class="settings-item">
              <label>@提及强提醒</label>
              <div class="settings-item-content">
                <label class="switch">
                  <input type="checkbox" v-model="localMessageSettings.mentionAlert" />
                  <span class="slider round"></span>
                </label>
                <div class="settings-hint">静音时被 @ 仍会提醒</div>
              </div>
            </div>
            <!-- C2: 通知内容预览 -->
            <div class="settings-item">
              <label>通知内容预览</label>
              <div class="settings-item-content">
                <select v-model="localMessageSettings.notificationPreview" class="settings-select">
                  <option value="content">显示消息内容</option>
                  <option value="simple">仅显示"新消息"</option>
                </select>
              </div>
            </div>
            <!-- C2: 勿扰例外名单 -->
            <div class="settings-item">
              <label>勿扰例外</label>
              <div class="settings-item-content">
                <div class="input-with-btn">
                  <input
                    v-model="dndExceptionInput"
                    type="text"
                    class="settings-input"
                    data-testid="dnd-exception-input"
                    placeholder="输入用户名后回车添加"
                    @keyup.enter="addDndException"
                  />
                  <button class="browse-btn" @click="addDndException">添加</button>
                </div>
                <div v-if="localMessageSettings.dndExceptions && localMessageSettings.dndExceptions.length > 0" class="exception-list">
                  <span v-for="(item, index) in localMessageSettings.dndExceptions" :key="index" class="exception-tag">
                    {{ item }}
                    <button class="exception-remove" @click="removeDndException(index)">×</button>
                  </span>
                </div>
              </div>
            </div>
            <!-- C2: 夜间自动免打扰 -->
            <div class="settings-item">
              <label>夜间免打扰</label>
              <div class="settings-item-content">
                <label class="switch">
                  <input type="checkbox" v-model="localMessageSettings.nightDndEnabled" />
                  <span class="slider round"></span>
                </label>
                <div class="settings-hint">到点自动开启免打扰</div>
              </div>
            </div>
            <div v-if="localMessageSettings.nightDndEnabled" class="settings-item">
              <label>夜间免打扰时间</label>
              <div class="dnd-time-range">
                <input v-model="localMessageSettings.nightDndStart" type="time" class="settings-select" />
                <span>至</span>
                <input v-model="localMessageSettings.nightDndEnd" type="time" class="settings-select" />
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
          
          <div v-if="localTab === 'appearance'" class="settings-section" style="margin-top: 20px;">
            <div class="settings-section-header"><h4>AI 助手</h4></div>
            <div class="settings-item">
              <label>显示 AI 悬浮球</label>
              <label class="switch"><input type="checkbox" v-model="showFloatingBall" /><span class="slider round"></span></label>
            </div>
            <div class="settings-item-hint" style="font-size: 12px; color: var(--text-color-secondary, #999); margin-left: 0;">关闭后可通过快捷键 Ctrl+Shift+L 打开 AI 侧边栏</div>
          </div>

          <div v-if="localTab === 'data-storage'" class="settings-section">
            <DataStorageSettings
              :defaultSaveDirectory="localMessageSettings.defaultSaveDirectory || ''"
              @update:defaultSaveDirectory="localMessageSettings.defaultSaveDirectory = $event"
              @browseDirectory="$emit('browseDirectory', $event)"
              @cacheCleared="handleCacheCleared"
            />
          </div>

          <div v-if="localTab === 'shortcut'" class="settings-section">
            <div class="settings-section-header"><h4>快捷键设置</h4></div>
            <ShortcutSettings ref="shortcutSettingsRef" v-model="localShortcuts" />
          </div>
        </div>
      </div>
      <div class="settings-footer">
        <button class="cancel-btn" @click="$emit('close')">取消</button>
        <button class="save-btn" @click="save">保存</button>
      </div>
    </div>
  </div>
  
  <AvatarCropper
    v-if="showCropper"
    :image-url="pendingImageUrl"
    @confirm="handleCropConfirm"
    @cancel="handleCropCancel"
  />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import Avatar from '../shared/Avatar.vue'
import { useAISidebar } from '../../composables/useAISidebar'
import AvatarCropper from '../modals/AvatarCropper.vue'
import ShortcutSettings from './ShortcutSettings.vue'
import DataStorageSettings from './DataStorageSettings.vue'
import type { ShortcutsConfig } from '../../composables/useShortcuts'
import type { MessageSettings, AppearanceSettings } from '../../composables/useSettings'

interface Theme {
  id: string
  name: string
  previewClass: string
}

const themes: Theme[] = [
  { id: 'modern-light', name: '清新白', previewClass: 'light-theme' },
  { id: 'elegant-dark', name: '炫酷黑', previewClass: 'dark-theme' },
  { id: 'monochrome-elegance', name: '单色雅', previewClass: 'monochrome-elegance-theme' },
  { id: 'crimson-red', name: '中国红', previewClass: 'crimson-red-theme' },
  { id: 'emerald-green', name: '翡翠绿', previewClass: 'emerald-green-theme' },
  { id: 'elegant-purple', name: '高雅紫', previewClass: 'elegant-purple-theme' },
  { id: 'warm-amber', name: '琥珀黄', previewClass: 'warm-amber-theme' },
  { id: 'ocean-blue', name: '海洋蓝', previewClass: 'netblue-theme' },
  { id: 'mediterranean-dream', name: '地中海', previewClass: 'mediterranean-dream-theme' },
  { id: 'spring-blossom', name: '春日花', previewClass: 'spring-blossom-theme' }
]

interface Props {
  visible: boolean
  currentUser?: { username?: string; avatar?: string }
  serverUrl: string
  profile: { nickname?: string; signature?: string }
  messageSettings: Partial<MessageSettings>
  appearanceSettings: Partial<AppearanceSettings>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'close': []
  'save': [data: { profile: any; messageSettings: any; appearanceSettings: any; avatarFile?: File; shortcuts?: ShortcutsConfig; showFloatingBall?: boolean }]
  'cacheCleared': []
  'browseDirectory': [callback: (path: string) => void]
}>()

const localTab = ref('basic')
const localProfile = ref({ ...props.profile })
const localMessageSettings = ref({ ...props.messageSettings })
const localAppearanceSettings = ref({ ...props.appearanceSettings })
const localShortcuts = ref<ShortcutsConfig>()
const shortcutSettingsRef = ref<InstanceType<typeof ShortcutSettings> | null>(null)

// AI 悬浮球开关
const { showFloatingAIBall } = useAISidebar()
const showFloatingBall = ref(showFloatingAIBall.value)

// 勿扰例外名单输入值
const dndExceptionInput = ref('')

const addDndException = () => {
  const value = dndExceptionInput.value.trim()
  if (value && !localMessageSettings.value.dndExceptions?.includes(value)) {
    localMessageSettings.value.dndExceptions = [...(localMessageSettings.value.dndExceptions || []), value]
    dndExceptionInput.value = ''
  }
}

const removeDndException = (index: number) => {
  const exceptions = [...(localMessageSettings.value.dndExceptions || [])]
  exceptions.splice(index, 1)
  localMessageSettings.value.dndExceptions = exceptions
}

const avatarInputRef = ref<HTMLInputElement | null>(null)
const showCropper = ref(false)
const pendingImageUrl = ref('')
const pendingAvatarFile = ref<File | null>(null)

watch(() => props.visible, (val) => {
  if (val) {
    localTab.value = 'basic'
    localProfile.value = { ...props.profile }
    localMessageSettings.value = { ...props.messageSettings }
    localAppearanceSettings.value = { ...props.appearanceSettings }
    showFloatingBall.value = showFloatingAIBall.value
  }
})

// 父组件重新加载设置后（例如缓存被清理后调用 loadSettings），props 会变化，
// 此时需要同步刷新 local 状态，避免用户点保存时把旧数据写回 localStorage
watch(() => props.messageSettings, (newVal) => {
  localMessageSettings.value = { ...localMessageSettings.value, ...newVal }
}, { deep: true })

watch(() => props.appearanceSettings, (newVal) => {
  localAppearanceSettings.value = { ...localAppearanceSettings.value, ...newVal }
}, { deep: true })

const triggerAvatarUpload = () => {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'
  input.onchange = (e: Event) => {
    const target = e.target as HTMLInputElement
    if (target.files && target.files.length > 0) {
      const file = target.files[0]
      if (!file.type.startsWith('image/')) return
      if (file.size > 5 * 1024 * 1024) return
      pendingImageUrl.value = URL.createObjectURL(file)
      showCropper.value = true
    }
  }
  input.click()
}

const handleCropConfirm = (croppedFile: File) => {
  pendingAvatarFile.value = croppedFile
  showCropper.value = false
  if (pendingImageUrl.value) {
    URL.revokeObjectURL(pendingImageUrl.value)
    pendingImageUrl.value = ''
  }
}

const handleCropCancel = () => {
  showCropper.value = false
  if (pendingImageUrl.value) {
    URL.revokeObjectURL(pendingImageUrl.value)
    pendingImageUrl.value = ''
  }
}

const save = () => {
  // 快捷键冲突检测
  if (shortcutSettingsRef.value) {
    const conflicts = shortcutSettingsRef.value.checkConflicts()
    if (conflicts.length > 0) {
      const c = conflicts[0]
      window.$QMessage?.warning(`快捷键冲突：「${c.a.label}」与「${c.b.label}」使用了相同的组合`, 5000)
      return
    }
  }
  emit('save', {
    profile: { ...localProfile.value },
    messageSettings: { ...localMessageSettings.value },
    appearanceSettings: { ...localAppearanceSettings.value },
    avatarFile: pendingAvatarFile.value || undefined,
    shortcuts: localShortcuts.value ? JSON.parse(JSON.stringify(localShortcuts.value)) : undefined,
    showFloatingBall: showFloatingBall.value
  })
  pendingAvatarFile.value = null
}

// 缓存被清理后通知父组件重新加载设置，避免保存时把旧数据写回 localStorage
const handleCacheCleared = () => {
  emit('cacheCleared')
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

.dnd-time-range {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.dnd-time-range .settings-select {
  max-width: 140px;
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
.elegant-purple-theme { background: #75629a; }
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

.about-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
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

.exception-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.exception-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--hover-color);
  border-radius: 12px;
  font-size: 12px;
  color: var(--text-color);
}

.exception-remove {
  border: none;
  background: none;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1;
  padding: 0;
}

.exception-remove:hover {
  color: var(--primary-color);
}
</style>
