import { describe, expect, it } from 'vitest'
import { adminModuleGroups, adminRouteRecords, sidebarModuleGroups } from '@/config/adminModules'

describe('admin module configuration', () => {
  it('keeps route records generated from the module registry', () => {
    const routePaths = adminRouteRecords.map(route => route.path)

    expect(routePaths).toContain('')
    expect(routePaths).toContain('users')
    expect(routePaths).toContain('ai-config')
    expect(routePaths).toContain('server-monitor')
    expect(routePaths).toHaveLength(31)
  })

  it('uses clearer top-level groups for the sidebar', () => {
    expect(adminModuleGroups.map(group => group.title)).toEqual([
      '概览',
      '身份与组织',
      '沟通治理',
      '应用与集成',
      'AI 与知识',
      '安全审计',
      '系统运维',
    ])
  })

  it('renders only modules with menu permissions in sidebar groups', () => {
    const sidebarItems = sidebarModuleGroups.flatMap(group => group.items)

    expect(sidebarItems.some(item => item.path === '/login')).toBe(false)
    expect(sidebarItems.find(item => item.path === '/ai-assistant')?.permission).toBe('ai:read')
    expect(sidebarItems.find(item => item.path === '/approvals')?.role).toBe('system_admin')
  })
})
