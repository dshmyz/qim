<template>
  <div class="tree-container">
    <template v-for="department in orgStructure" :key="department.id">
      <div class="tree-node department-node">
        <div class="tree-node-content" @click="toggleDepartment(department.id)">
          <span class="toggle-icon">{{ expandedDepartments.includes(department.id) ? '▼' : '▶' }}</span>
          <span class="node-name department-name">{{ department.name }}</span>
        </div>
        <div v-if="expandedDepartments.includes(department.id)" class="tree-children">
          <div v-for="child in department.subDepartments" :key="child.id">
            <div class="tree-node sub-department-node">
              <div class="tree-node-content" @click="toggleSubDepartment(department.id, child.id)">
                <span class="toggle-icon">{{ expandedSubDepartments[department.id]?.includes(child.id) ? '▼' : '▶' }}</span>
                <span class="node-name sub-department-name">{{ child.name }}</span>
              </div>
              <div v-if="expandedSubDepartments[department.id]?.includes(child.id)" class="tree-children">
                <div v-if="child.subDepartments && child.subDepartments.length > 0">
                  <div v-for="grandChild in child.subDepartments" :key="grandChild.id">
                    <div class="tree-node sub-department-node">
                      <div class="tree-node-content" @click="toggleSubDepartment(child.id, grandChild.id)">
                        <span class="toggle-icon">{{ expandedSubDepartments[child.id]?.includes(grandChild.id) ? '▼' : '▶' }}</span>
                        <span class="node-name sub-department-name">{{ grandChild.name }}</span>
                      </div>
                      <div v-if="expandedSubDepartments[child.id]?.includes(grandChild.id)" class="tree-children">
                        <div v-if="grandChild.employees && grandChild.employees.length > 0">
                          <div v-for="employee in grandChild.employees" :key="employee.id" class="tree-node employee-node">
                            <div class="tree-node-content" @click="$emit('selectUser', employee)" @dblclick="$emit('startPrivateChat', employee)" @contextmenu.prevent="$emit('userContextMenu', $event, employee)">
                              <span class="employee-avatar-container">
                                <img :src="employee.avatar" :alt="employee.name" class="employee-avatar" />
                              </span>
                              <span class="node-name employee-name">{{ employee.name }}</span>
                              <span class="employee-position">{{ employee.position }}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <div v-else-if="child.employees && child.employees.length > 0">
                  <div v-for="employee in child.employees" :key="employee.id" class="tree-node employee-node">
                    <div class="tree-node-content" @click="$emit('selectUser', employee)" @dblclick="$emit('startPrivateChat', employee)" @contextmenu.prevent="$emit('userContextMenu', $event, employee)">
                      <span class="employee-avatar-container">
                        <img :src="employee.avatar" :alt="employee.name" class="employee-avatar" />
                      </span>
                      <span class="node-name employee-name">{{ employee.name }}</span>
                      <span class="employee-position">{{ employee.position }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface OrgDepartment {
  id: string
  name: string
  subDepartments: OrgDepartment[]
  employees?: any[]
}

interface Props {
  orgStructure: OrgDepartment[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'selectUser', user: any): void
  (e: 'startPrivateChat', user: any): void
  (e: 'userContextMenu', event: MouseEvent, user: any): void
}>()

const expandedDepartments = ref<string[]>([])
const expandedSubDepartments = ref<Record<string, string[]>>({})

const toggleDepartment = (id: string) => {
  const index = expandedDepartments.value.indexOf(id)
  if (index > -1) {
    expandedDepartments.value.splice(index, 1)
  } else {
    expandedDepartments.value.push(id)
  }
}

const toggleSubDepartment = (parentId: string, subId: string) => {
  if (!expandedSubDepartments.value[parentId]) {
    expandedSubDepartments.value[parentId] = []
  }
  const index = expandedSubDepartments.value[parentId].indexOf(subId)
  if (index > -1) {
    expandedSubDepartments.value[parentId].splice(index, 1)
  } else {
    expandedSubDepartments.value[parentId].push(subId)
  }
}
</script>

<style scoped>
.tree-container {
  background: #fafafa;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow: hidden;
  margin: 8px 8px;
  flex: 1;
  overflow-y: auto;
}

.tree-container::-webkit-scrollbar {
  width: 4px;
}

.tree-container::-webkit-scrollbar-track {
  background: transparent;
}

.tree-container::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
  transition: background 0.2s ease;
  height: 40px;
}

.tree-container::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

.tree-node {
  position: relative;
}

.tree-node-content {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  gap: 10px;
}

.tree-node-content:hover {
  background: var(--hover-color);
}

.toggle-icon {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.7;
  width: 12px;
  text-align: left;
  transition: transform 0.2s ease;
  flex-shrink: 0;
  margin-right: 4px;
}

.node-name {
  font-weight: 500;
  color: var(--text-color);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.department-name {
  font-size: 14px;
  color: var(--primary-color);
  font-weight: 600;
}

.sub-department-name {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
}

.employee-name {
  font-size: 13px;
  color: var(--text-color);
  font-weight: 500;
  min-width: 60px;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.employee-avatar-container {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
}

.employee-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
}

.employee-position {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.7;
  flex-shrink: 0;
  margin-left: 8px;
  white-space: nowrap;
}

.department-stats {
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
  margin-left: 6px;
  font-weight: 400;
}

.tree-children {
  margin-left: 12px;
  padding-left: 8px;
}

.department-node .tree-children {
  margin-left: 16px;
}

.sub-department-node .tree-children {
  margin-left: 16px;
}

.department-node:last-child .tree-node-content {
  border-bottom: none;
}

.sub-department-node .tree-node-content {
  padding-left: 16px;
}

.employee-node .tree-node-content {
  padding-left: 16px;
  background: transparent;
  opacity: 1;
}

.employee-node .tree-node-content:hover {
  background: var(--hover-color);
  opacity: 1;
}

.employee-node:last-child .tree-node-content {
  border-bottom: none;
}
</style>
