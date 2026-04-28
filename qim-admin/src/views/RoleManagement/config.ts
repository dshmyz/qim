import type { FormField } from '@/composables/useEntity'

export const permissionOptions = [
  { label: '查看用户', value: 'user:read' },
  { label: '创建用户', value: 'user:create' },
  { label: '编辑用户', value: 'user:update' },
  { label: '删除用户', value: 'user:delete' },
  { label: '查看群组', value: 'group:read' },
  { label: '创建群组', value: 'group:create' },
  { label: '编辑群组', value: 'group:update' },
  { label: '删除群组', value: 'group:delete' },
  { label: '查看角色', value: 'role:read' },
  { label: '创建角色', value: 'role:create' },
  { label: '编辑角色', value: 'role:update' },
  { label: '删除角色', value: 'role:delete' },
]

// 从 options 派生标签映射，额外保留历史权限映射
export const permissionLabelMap: Record<string, string> = {
  ...Object.fromEntries(permissionOptions.map((opt) => [opt.value, opt.label])),
  'message:read': '查看消息',
  'message:write': '发送消息',
  'message:delete': '删除消息',
  'system:config': '系统配置',
  'system:log': '查看日志',
}

export const roleFields: FormField[] = [
  { name: 'name', label: '角色名称', type: 'input', required: true },
  { name: 'code', label: '角色代码', type: 'input', required: true },
  { name: 'description', label: '描述', type: 'textarea' },
  { name: 'permissions', label: '权限', type: 'select', props: { multiple: true, filterable: true }, options: permissionOptions },
]

export const roleRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入角色代码', trigger: 'blur' }],
}
