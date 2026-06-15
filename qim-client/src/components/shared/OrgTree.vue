<template>
  <div v-if="orgStructure.length === 0" class="empty-org">
    <div class="placeholder-content">
      <i class="fas fa-building fa-4x"></i>
      <h3>暂无组织架构</h3>
      <p>暂无部门数据，请联系管理员配置组织架构</p>
    </div>
  </div>
  <div v-else-if="searchQuery && filteredOrgStructure.length === 0" class="empty-org">
    <div class="placeholder-content">
      <i class="fas fa-search fa-4x"></i>
      <h3>未找到匹配结果</h3>
      <p>没有找到与 "{{ searchQuery }}" 相关的员工或部门</p>
    </div>
  </div>
  <div v-else class="tree-container">
    <template v-for="department in filteredOrgStructure" :key="department.id">
      <div class="tree-node department-node">
        <div class="tree-node-content" @click="toggleDepartment(department.id)">
          <span class="toggle-icon">{{ expandedDepartments.includes(department.id) ? '▼' : '▶' }}</span>
          <span class="node-name department-name">{{ department.name }}</span>
        </div>
        <div v-if="expandedDepartments.includes(department.id)" class="tree-children">
          <div v-if="department.employees && department.employees.length > 0">
            <div v-for="employee in department.employees" :key="employee.id" class="tree-node employee-node">
              <div class="tree-node-content" @click="$emit('selectUser', employee)" @dblclick="$emit('startPrivateChat', employee)" @contextmenu.prevent="$emit('userContextMenu', $event, employee)">
                <span class="employee-avatar-container">
                  <Avatar :src="employee.avatar" :name="employee.name" :alt="employee.name" size="sm" class="employee-avatar" />
                </span>
                <span class="node-name employee-name">{{ employee.name }}</span>
                <span class="employee-position">{{ employee.position }}</span>
              </div>
            </div>
          </div>
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
                                <Avatar :src="employee.avatar" :name="employee.name" :alt="employee.name" size="sm" class="employee-avatar" />
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
                        <Avatar :src="employee.avatar" :name="employee.name" :alt="employee.name" size="sm" class="employee-avatar" />
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
import { ref, computed, watch } from 'vue'
import Avatar from './Avatar.vue'

interface OrgDepartment {
  id: string
  name: string
  subDepartments: OrgDepartment[]
  employees?: any[]
}

interface Props {
  orgStructure: OrgDepartment[]
  searchQuery?: string
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

function employeeMatches(employee: any, query: string): boolean {
  const fields = [
    employee.name || '',
    employee.nickname || '',
    employee.username || '',
    employee.position || '',
    employee.email || '',
    employee.mobile || '',
    employee.department || ''
  ]
  return fields.some(f => f.toLowerCase().includes(query))
}

function filterDepartment(dept: OrgDepartment, query: string): OrgDepartment | null {
  const deptNameMatch = dept.name.toLowerCase().includes(query)

  const filteredEmployees = dept.employees
    ? dept.employees.filter(emp => employeeMatches(emp, query))
    : []

  const filteredChildren = dept.subDepartments
    ? dept.subDepartments
        .map(child => filterDepartment(child, query))
        .filter((d): d is OrgDepartment => d !== null)
    : []

  const hasMatch = deptNameMatch || filteredEmployees.length > 0 || filteredChildren.length > 0
  if (!hasMatch) return null

  return {
    ...dept,
    subDepartments: filteredChildren,
    employees: deptNameMatch ? dept.employees : filteredEmployees
  }
}

const filteredOrgStructure = computed(() => {
  if (!props.searchQuery || !props.searchQuery.trim()) {
    return props.orgStructure
  }
  const query = props.searchQuery.toLowerCase().trim()
  return props.orgStructure
    .map(dept => filterDepartment(dept, query))
    .filter((d): d is OrgDepartment => d !== null)
})

function collectMatchedDepartmentIds(
  dept: OrgDepartment,
  query: string,
  results: string[]
): boolean {
  let hasMatch = dept.name.toLowerCase().includes(query)

  if (dept.employees) {
    const empMatch = dept.employees.some(emp => employeeMatches(emp, query))
    if (empMatch) hasMatch = true
  }

  const childMatches: string[] = []
  dept.subDepartments?.forEach(child => {
    const subIds: string[] = []
    if (collectMatchedDepartmentIds(child, query, subIds)) {
      hasMatch = true
      childMatches.push(child.id)
      childMatches.push(...subIds)
    }
  })

  if (hasMatch) {
    results.push(dept.id)
    results.push(...childMatches)
  }

  return hasMatch
}

watch(() => props.searchQuery, (newQuery) => {
  if (!newQuery || !newQuery.trim()) {
    return
  }
  const query = newQuery.toLowerCase().trim()
  const matchedDeptIds: string[] = []
  const matchedSubDeptIds: Record<string, string[]> = {}

  props.orgStructure.forEach(dept => {
    const subIds: string[] = []
    dept.subDepartments?.forEach(child => {
      const grandIds: string[] = []
      if (collectMatchedDepartmentIds(child, query, grandIds)) {
        if (!matchedSubDeptIds[dept.id]) {
          matchedSubDeptIds[dept.id] = []
        }
        matchedSubDeptIds[dept.id].push(child.id)
        if (grandIds.length > 0) {
          matchedSubDeptIds[child.id] = [...(matchedSubDeptIds[child.id] || []), ...grandIds]
        }
      }
    })

    const deptHasMatch = dept.name.toLowerCase().includes(query)
      || (dept.employees && dept.employees.some(emp => employeeMatches(emp, query)))
      || (matchedSubDeptIds[dept.id] && matchedSubDeptIds[dept.id].length > 0)

    if (deptHasMatch) {
      matchedDeptIds.push(dept.id)
    }
  })

  expandedDepartments.value = matchedDeptIds
  expandedSubDepartments.value = matchedSubDeptIds

})
</script>

<style scoped>
.tree-container {
  background: var(--panel-bg);
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
  color: var(--text-color);
  font-weight: 500;
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

.empty-org {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  background: var(--panel-bg);
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  margin: 8px 8px;
}

.empty-org .placeholder-content {
  text-align: center;
  color: var(--text-secondary, #666);
}

.empty-org .placeholder-content i {
  color: var(--text-tertiary, #999);
  margin-bottom: 16px;
}

.empty-org .placeholder-content h3 {
  margin: 0 0 8px 0;
  color: var(--text-primary, #333);
}

.empty-org .placeholder-content p {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary, #666);
}

</style>
