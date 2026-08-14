import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')

describe('Electron permission policy', () => {
  it('allows sanitized clipboard writes while leaving other permissions denied', () => {
    expect(mainProcess).toContain("['media', 'clipboard-sanitized-write', 'clipboard-read', 'notifications', 'fullscreen'].includes(permission)")
    expect(mainProcess).toContain('callback(false)')
  })
})
