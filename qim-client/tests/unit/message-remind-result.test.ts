import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const mainView = readFileSync(resolve(__dirname, '../../src/views/Main.vue'), 'utf8')

describe('message reminder result', () => {
  it('shows the configured target system name and falls back for old receipts', () => {
    expect(mainView).toContain("const systemName = data.system_name?.trim() || '外部系统'")
    expect(mainView).toContain("showMessage({ message: `提醒已送达${systemName}`, type: 'success' })")
  })
})
