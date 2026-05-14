<template>
  <div class="avatar-skills-panel">
    <div class="panel-header">
      <h4>分身技能</h4>
      <span class="skill-count">{{ enabledCount }}/{{ tools.length }} 已启用</span>
    </div>

    <div class="skill-categories">
      <div class="category">
        <h5><i class="fas fa-brain"></i> AI 能力</h5>
        <div class="skill-grid">
          <div 
            v-for="tool in aiTools" 
            :key="tool.tool_id" 
            :class="['skill-card', { enabled: tool.enabled }]"
            @click="toggleTool(tool)"
          >
            <div class="skill-icon">
              <i :class="tool.icon"></i>
            </div>
            <div class="skill-info">
              <span class="skill-name">{{ tool.name }}</span>
              <span class="skill-desc">{{ tool.description }}</span>
            </div>
            <div :class="['skill-toggle', { active: tool.enabled }]">
              <span class="toggle-dot"></span>
            </div>
          </div>
        </div>
      </div>

      <div class="category">
        <h5><i class="fas fa-wrench"></i> 运维工具</h5>
        <div class="skill-grid">
          <div 
            v-for="tool in opsTools" 
            :key="tool.tool_id" 
            :class="['skill-card', { enabled: tool.enabled }]"
            @click="toggleTool(tool)"
          >
            <div class="skill-icon">
              <i :class="tool.icon"></i>
            </div>
            <div class="skill-info">
              <span class="skill-name">{{ tool.name }}</span>
              <span class="skill-desc">{{ tool.description }}</span>
            </div>
            <div :class="['skill-toggle', { active: tool.enabled }]">
              <span class="toggle-dot"></span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="empty-state" v-if="tools.length === 0">
      <i class="fas fa-puzzle-piece"></i>
      <p>暂无可用技能</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { avatarAPI } from '../../api/avatar'
import { request } from '../../composables/useRequest'

interface ToolBinding {
  id: number
  avatar_id: number
  tool_id: string
  enabled: boolean
  priority: number
  name: string
  description: string
  icon: string
}

const tools = ref<ToolBinding[]>([])
const loading = ref(false)

const enabledCount = computed(() => tools.value.filter(t => t.enabled).length)
const aiTools = computed(() => tools.value.filter(t => 
  ['chat', 'knowledge_search', 'knowledge_save', 'memory_search', 'summary', 'translate'].includes(t.tool_id)
))
const opsTools = computed(() => tools.value.filter(t => 
  ['server_monitor', 'log_analyzer'].includes(t.tool_id)
))

async function loadTools() {
  loading.value = true
  try {
    const data = await avatarAPI.getAvailableTools()
    tools.value = data || []
  } catch (e) {
    console.error('加载技能列表失败', e)
  } finally {
    loading.value = false
  }
}

async function toggleTool(tool: ToolBinding) {
  tool.enabled = !tool.enabled
  try {
    if (tool.enabled) {
      await request(`/api/v1/avatar/${tool.avatar_id}/tools/${tool.tool_id}`, { method: 'POST' })
    } else {
      await request(`/api/v1/avatar/${tool.avatar_id}/tools/${tool.tool_id}`, { method: 'DELETE' })
    }
  } catch (e) {
    tool.enabled = !tool.enabled
    console.error('更新技能状态失败', e)
  }
}

onMounted(() => {
  loadTools()
})
</script>

<style scoped>
.avatar-skills-panel {
  padding: 16px;
  background: var(--card-bg);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.panel-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.skill-count {
  font-size: 13px;
  color: var(--text-secondary);
}

.skill-categories {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.category h5 {
  margin: 0 0 12px;
  font-size: 14px;
  color: var(--text-secondary);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}

.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.skill-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.skill-card:hover {
  border-color: var(--primary-color);
  transform: translateY(-1px);
}

.skill-card.enabled {
  border-color: var(--primary-color);
  background: rgba(99, 102, 241, 0.05);
}

.skill-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--card-bg);
  border-radius: 10px;
  color: var(--text-secondary);
  font-size: 18px;
  flex-shrink: 0;
}

.skill-card.enabled .skill-icon {
  color: var(--primary-color);
  background: rgba(99, 102, 241, 0.1);
}

.skill-info {
  flex: 1;
  min-width: 0;
}

.skill-name {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.skill-desc {
  display: block;
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.skill-toggle {
  width: 40px;
  height: 24px;
  background: var(--border-color);
  border-radius: 12px;
  position: relative;
  transition: all 0.2s;
  flex-shrink: 0;
}

.skill-toggle.active {
  background: var(--primary-color);
}

.toggle-dot {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  background: white;
  border-radius: 50%;
  transition: left 0.2s;
}

.skill-toggle.active .toggle-dot {
  left: 18px;
}

.empty-state {
  text-align: center;
  padding: 32px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 40px;
  margin-bottom: 8px;
  display: block;
}
</style>