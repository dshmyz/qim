import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const component = readFileSync(
  resolve(__dirname, '../src/views/components/MessageRemindWebhookConfig.vue'),
  'utf8'
)

describe('message reminder webhook configuration', () => {
  it('lets administrators save a target system name', () => {
    expect(component).toContain('label="系统名称"')
    expect(component).toContain('v-model="config.system_name"')
    expect(component).toContain('placeholder="例如：企业微信、飞书、Slack"')
    expect(component).toContain('system_name: string')
    expect(component).toContain("system_name: ''")
  })
})
