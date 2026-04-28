import type { FormRules } from 'element-plus'
import type { FormField } from '@/composables/useEntity'

export const userFields: FormField[] = [
  { name: 'username', label: '用户名', type: 'input', required: true, props: { disabled: true } },
  { name: 'password', label: '密码', type: 'password', required: true, props: { showPassword: true } },
  { name: 'nickname', label: '昵称', type: 'input' },
  { name: 'email', label: '邮箱', type: 'input', required: true },
  { name: 'avatar', label: '头像', type: 'input', props: { placeholder: '请输入头像URL' } },
  { name: 'phone', label: '手机号', type: 'input' },
]

export const userRules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效邮箱', trigger: 'blur' },
  ],
}

export const roleOptions = [
  { label: '系统管理员', value: 'system_admin' },
  { label: '系统发布者', value: 'system_publisher' },
  { label: '系统审核员', value: 'system_moderator' },
  { label: '系统运营', value: 'system_operator' },
]
