<template>
  <div class="org-picker">
    <div v-if="visibleNodes.length === 0" class="empty">
      {{ searchQuery ? '没有找到匹配的成员' : '暂无组织架构' }}
    </div>
    <template v-for="node in visibleNodes" :key="node.key">
      <!-- 部门节点 -->
      <div v-if="node.type === 'dept'" class="node dept-node" :style="{ paddingLeft: node.level * 18 + 8 + 'px' }">
        <span class="toggle" @click="toggleExpand(node.id)">{{ expandedIds.has(node.id) ? '▼' : '▶' }}</span>
        <input type="checkbox" :checked="isDeptAllSelected(node.dept!)" :indeterminate.prop="isDeptIndeterminate(node.dept!)" @change="toggleDept(node.dept!)" />
        <span class="dept-name" @click="toggleExpand(node.id)">{{ node.name }}</span>
        <span class="dept-count">{{ deptSelectedCount(node.dept!) }}/{{ deptEmployeeCount(node.dept!) }}</span>
      </div>
      <!-- 员工节点 -->
      <div v-else class="node emp-node" :style="{ paddingLeft: node.level * 18 + 8 + 'px' }" @click="toggleEmployee(node.employee!)">
        <span class="toggle-placeholder"></span>
        <input type="checkbox" :checked="isSelected(node.id)" @click.stop @change="toggleEmployee(node.employee!)" />
        <Avatar :src="node.avatar" :name="node.name" :alt="node.name" size="sm" />
        <span class="emp-name">{{ node.name }}</span>
        <span class="emp-position">{{ node.position || '' }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Avatar from './Avatar.vue'
import type { Department } from '../../composables/useOrganizationLogic'

// 宽松员工类型：兼容 useOrganizationLogic.Employee(id: string) 与 GroupModals.Employee(id: string|number)
interface PickableEmployee {
  id: string | number
  name: string
  avatar: string
  position?: string
}

interface FlatNode {
  type: 'dept' | 'employee'
  key: string
  id: string
  name: string
  level: number
  dept?: Department
  avatar?: string
  position?: string
  employee?: PickableEmployee
}

const props = defineProps<{
  orgStructure: Department[]
  selectedMembers: PickableEmployee[]
  searchQuery?: string
}>()

const emit = defineEmits<{
  'update:selectedMembers': [members: PickableEmployee[]]
}>()

// 默认展开顶层部门，减少点击
const expandedIds = ref<Set<string>>(new Set(props.orgStructure.map(d => d.id)))
// orgStructure 异步加载时，第一次有值默认展开顶层
watch(() => props.orgStructure, (depts) => {
  if (depts.length > 0 && expandedIds.value.size === 0) {
    expandedIds.value = new Set(depts.map(d => d.id))
  }
}, { immediate: true })

// ===== 搜索过滤（本地，保留组织上下文）=====
function employeeMatches(emp: any, query: string): boolean {
  const fields = [
    emp.name || '',
    emp.nickname || '',
    emp.username || '',
    emp.position || '',
    emp.department || ''
  ]
  return fields.some(f => f.toLowerCase().includes(query))
}

function filterDepartment(dept: Department, query: string): Department | null {
  const deptNameMatch = dept.name.toLowerCase().includes(query)
  const filteredEmployees = dept.employees ? dept.employees.filter(e => employeeMatches(e, query)) : []
  const filteredChildren = dept.subDepartments
    ? dept.subDepartments.map(c => filterDepartment(c, query)).filter((d): d is Department => d !== null)
    : []
  const hasMatch = deptNameMatch || filteredEmployees.length > 0 || filteredChildren.length > 0
  if (!hasMatch) return null
  // 部门名命中时保留其全部员工（整部门匹配），否则只保留命中员工
  return { ...dept, subDepartments: filteredChildren, employees: deptNameMatch ? dept.employees : filteredEmployees }
}

const filteredOrgStructure = computed<Department[]>(() => {
  const q = props.searchQuery?.trim().toLowerCase()
  if (!q) return props.orgStructure
  return props.orgStructure.map(d => filterDepartment(d, q)).filter((d): d is Department => d !== null)
})

function collectAllDeptIds(depts: Department[]): string[] {
  const ids: string[] = []
  const walk = (d: Department) => {
    ids.push(d.id)
    d.subDepartments?.forEach(walk)
  }
  depts.forEach(walk)
  return ids
}

// 搜索时自动展开所有匹配部门，让命中员工立即可见；清空搜索时回到用户手动展开状态
watch(() => props.searchQuery, (q) => {
  if (q && q.trim()) {
    expandedIds.value = new Set(collectAllDeptIds(filteredOrgStructure.value))
  }
}, { immediate: true })

function flatten(depts: Department[], level = 0): FlatNode[] {
  const nodes: FlatNode[] = []
  for (const dept of depts) {
    nodes.push({ type: 'dept', key: 'dept-' + dept.id, id: dept.id, name: dept.name, level, dept })
    if (dept.employees?.length) {
      for (const emp of dept.employees) {
        nodes.push({ type: 'employee', key: 'emp-' + emp.id, id: String(emp.id), name: emp.name, level: level + 1, avatar: emp.avatar, position: emp.position, employee: emp })
      }
    }
    if (dept.subDepartments?.length) {
      nodes.push(...flatten(dept.subDepartments, level + 1))
    }
  }
  return nodes
}

const flatNodes = computed(() => flatten(filteredOrgStructure.value))

// 可见节点：折叠的部门节点其后代（level 更深）隐藏，直到回到同级或更上层
const visibleNodes = computed(() => {
  const result: FlatNode[] = []
  let hideBelowLevel = -1
  for (const node of flatNodes.value) {
    if (hideBelowLevel >= 0 && node.level > hideBelowLevel) continue
    hideBelowLevel = -1
    result.push(node)
    if (node.type === 'dept' && !expandedIds.value.has(node.id)) {
      hideBelowLevel = node.level
    }
  }
  return result
})

function toggleExpand(id: string) {
  const next = new Set(expandedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedIds.value = next
}

const selectedIds = computed(() => new Set(props.selectedMembers.map(m => String(m.id))))

function isSelected(id: string) {
  return selectedIds.value.has(id)
}

// 递归收集部门子树所有员工
function collectDeptEmployees(dept: Department): PickableEmployee[] {
  const list: PickableEmployee[] = []
  const collect = (d: Department) => {
    if (d.employees) list.push(...d.employees)
    if (d.subDepartments) d.subDepartments.forEach(collect)
  }
  collect(dept)
  return list
}

function deptEmployeeCount(dept: Department): number {
  return collectDeptEmployees(dept).length
}
function deptSelectedCount(dept: Department): number {
  return collectDeptEmployees(dept).filter(e => selectedIds.value.has(String(e.id))).length
}
function isDeptAllSelected(dept: Department): boolean {
  const emps = collectDeptEmployees(dept)
  return emps.length > 0 && emps.every(e => selectedIds.value.has(String(e.id)))
}
function isDeptIndeterminate(dept: Department): boolean {
  const emps = collectDeptEmployees(dept)
  const cnt = emps.filter(e => selectedIds.value.has(String(e.id))).length
  return cnt > 0 && cnt < emps.length
}

function emitList(list: PickableEmployee[]) {
  const seen = new Set<string>()
  const deduped: PickableEmployee[] = []
  for (const e of list) {
    const id = String(e.id)
    if (!seen.has(id)) {
      seen.add(id)
      deduped.push(e)
    }
  }
  emit('update:selectedMembers', deduped)
}

function toggleEmployee(emp: PickableEmployee) {
  const list = [...props.selectedMembers]
  const idx = list.findIndex(e => String(e.id) === String(emp.id))
  if (idx >= 0) list.splice(idx, 1)
  else list.push(emp)
  emitList(list)
}

function toggleDept(dept: Department) {
  const deptEmps = collectDeptEmployees(dept)
  const deptEmpIds = new Set(deptEmps.map(e => String(e.id)))
  const list = [...props.selectedMembers]
  if (isDeptAllSelected(dept)) {
    emitList(list.filter(e => !deptEmpIds.has(String(e.id))))
  } else {
    emitList([...list, ...deptEmps])
  }
}
</script>

<style scoped>
.org-picker {
  max-height: 360px;
  overflow-y: auto;
}
.node {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  cursor: pointer;
  transition: background 0.15s;
}
.node:hover {
  background: var(--hover-color, rgba(0, 0, 0, 0.04));
}
.dept-node {
  font-weight: 500;
}
.toggle {
  width: 12px;
  font-size: 11px;
  color: var(--text-secondary, #999);
  cursor: pointer;
  flex-shrink: 0;
}
.toggle-placeholder {
  width: 12px;
  flex-shrink: 0;
}
.dept-name {
  flex: 1;
  font-size: 13px;
  color: var(--text-color, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dept-count {
  font-size: 11px;
  color: var(--text-secondary, #999);
  flex-shrink: 0;
}
.emp-name {
  font-size: 13px;
  color: var(--text-color, #333);
  min-width: 50px;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.emp-position {
  font-size: 11px;
  color: var(--text-secondary, #999);
  flex-shrink: 0;
}
.empty {
  text-align: center;
  color: var(--text-secondary, #999);
  padding: 32px 0;
  font-size: 13px;
}
</style>
