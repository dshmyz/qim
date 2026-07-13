import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const root = resolve(__dirname, '../..')
const read = (path: string) => readFileSync(resolve(root, path), 'utf8')

describe('elegant purple theme', () => {
  it('defines the approved Moonlight Lavender tokens', () => {
    const css = read('src/assets/styles/themes.css')
    const block = css.match(/\[data-theme="elegant-purple"\]\s*\{([\s\S]*?)\n\}/)?.[1]

    expect(block).toBeTruthy()
    expect(block).toContain('--primary-color: #75629a;')
    expect(block).toContain('--accent-color: #8b78b4;')
    expect(block).toContain('--active-color: #665486;')
    expect(block).toContain('--secondary-color: #ffffff;')
    expect(block).toContain('--content-bg: #ffffff;')
    expect(block).toContain('--list-bg: #fcfbfe;')
    expect(block).toContain('--hover-color: #efebf6;')
    expect(block).toContain('--border-color: #e5e0ef;')
    expect(block).toContain('--text-color: #342e43;')
    expect(block).toContain('--text-secondary: #746d81;')
  })

  it('uses theme variables in elegant-purple component overrides', () => {
    expect(read('src/components/message/MessageItem.vue')).not.toContain('rgba(139, 92, 246')
    expect(read('src/components/chat/QuotedMessagePreview.vue')).toContain(
      'border-left-color: var(--accent-color)',
    )
    expect(read('src/components/modals/UserProfile.vue')).not.toContain('#7e22ce')
  })

  it('keeps nested content readable inside self message bubbles', () => {
    const fileMessage = read('src/components/message/FileMessage.vue')

    expect(fileMessage).toContain(
      ':global([data-theme="elegant-purple"] .message-item.self .file-message.self)',
    )
    expect(fileMessage).toContain(
      ':global([data-theme="elegant-purple"] .message-item.self .attachment-card__title)',
    )
    expect(fileMessage).toContain('background: var(--hover-color);')
    expect(fileMessage).toContain('color: var(--text-secondary);')
  })


  it('shows the approved palette in every elegant-purple preview', () => {
    for (const path of [
      'src/assets/styles/layout.css',
      'src/views/Main.css',
      'src/components/menus/MainContextMenus.vue',
      'src/components/settings/SettingsPanel.vue',
    ]) {
      expect(read(path)).toMatch(/\.elegant-purple-theme\s*\{[^}]*#75629a/is)
    }
  })
})
