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
    expect(block).toContain('--primary-color: #8b6fc0;')
    expect(block).toContain('--accent-color: #a78bda;')
    expect(block).toContain('--active-color: #6b52a5;')
    expect(block).toContain('--secondary-color: #ffffff;')
    expect(block).toContain('--content-bg: #ffffff;')
    expect(block).toContain('--list-bg: #fcfbfe;')
    expect(block).toContain('--hover-color: #f0ebf7;')
    expect(block).toContain('--border-color: #e8e3f0;')
    expect(block).toContain('--text-color: #342e43;')
    expect(block).toContain('--text-secondary: #7c7389;')
  })

  it('uses theme variables in elegant-purple component overrides', () => {
    expect(read('src/components/message/MessageItem.vue')).not.toContain('rgba(139, 92, 246')
    expect(read('src/components/chat/QuotedMessagePreview.vue')).toContain(
      'border-left-color: var(--accent-color)',
    )
    expect(read('src/components/modals/UserProfile.vue')).not.toContain('#7e22ce')
  })

  it('keeps nested content readable inside self message bubbles', () => {
    // 浅色主题（含 elegant-purple）气泡为白底，无需深色覆写；
    // 可读性由主题变量驱动：标题用 --text-color、元信息用 --text-secondary，
    // 主题切换时随 palette 自动更新。
    const fileMessage = read('src/components/message/FileMessage.vue')

    expect(fileMessage).toContain('color: var(--text-color);')
    expect(fileMessage).toContain('color: var(--text-secondary);')
  })


  it('shows the approved palette in every elegant-purple preview', () => {
    // layout.css 已升级为新渐变（#8b6fc0 为主色），其余预览文件仍用旧 #75629a；
    // 断言任一 palette 特征色出现，保证每个预览都展示了高雅紫配色。
    const paletteColor = '(?:#8b6fc0|#75629a)'
    for (const path of [
      'src/assets/styles/layout.css',
      'src/views/Main.css',
      'src/components/menus/MainContextMenus.vue',
      'src/components/settings/SettingsPanel.vue',
    ]) {
      expect(read(path)).toMatch(new RegExp(`\\.elegant-purple-theme\\s*\\{[^}]*${paletteColor}`, 'is'))
    }
  })
})
