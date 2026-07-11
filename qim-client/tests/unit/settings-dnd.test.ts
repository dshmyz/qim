import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const settingsPanel = readFileSync(resolve(__dirname, '../../src/components/settings/SettingsPanel.vue'), 'utf8')
const useSettings = readFileSync(resolve(__dirname, '../../src/composables/useSettings.ts'), 'utf8')

describe('message do-not-disturb settings', () => {
  it('supports all-day and custom time ranges', () => {
    expect(settingsPanel).toContain('<option value="all_day">全天免打扰</option>')
    expect(settingsPanel).toContain('<option value="custom">自定义时间段</option>')
    expect(settingsPanel).toContain("localMessageSettings.dndMode === 'custom'")
    expect(settingsPanel).toContain('localMessageSettings.dndStartTime')
    expect(settingsPanel).toContain('localMessageSettings.dndEndTime')
  })

  it('persists custom do-not-disturb defaults and migrates the old work value', () => {
    expect(useSettings).toContain("dndMode: 'none'")
    expect(useSettings).toContain("dndStartTime: '22:00'")
    expect(useSettings).toContain("dndEndTime: '08:00'")
    expect(useSettings).toContain("if (parsed.dndMode === 'work') parsed.dndMode = 'all_day'")
  })
})
