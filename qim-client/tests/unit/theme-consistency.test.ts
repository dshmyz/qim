import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

/**
 * 主题一致性守卫：防止主题命名双轨再次脱节。
 *
 * 背景：项目曾出现两套主题命名并存——themes.css 用 chinesered/grassgreen/
 * sacredyellow 命名，而主题切换入口（SettingsPanel）用 crimson-red/
 * emerald-green——改 themes.css 死段用户端零变化。
 *
 * 守卫三条规则：
 * 1. SettingsPanel 注册的每个主题 id，必须在 themes.css 或 design-tokens.css
 *    至少一个文件中有 [data-theme="<id>"] 段（不允许无样式的主题）
 * 2. themes.css / design-tokens.css 中不允许出现不在注册列表内的死段
 *    （防"改了半天不生效"）
 * 3. design-tokens.css 的主题段内不允许同一变量重复定义
 *    （段内靠后的同名变量会静默覆盖靠前的，曾导致图标栏改了没反应）
 */
const ROOT = resolve(__dirname, '../../src/assets/styles')

function readThemeIdsFromSettingsPanel(): string[] {
  const src = readFileSync(resolve(ROOT, '../../components/settings/SettingsPanel.vue'), 'utf-8')
  const block = src.match(/const themes: Theme\[\] = \[([\s\S]*?)\n\]/)
  if (!block) throw new Error('SettingsPanel.vue 中未找到 themes 数组')
  return [...block[1].matchAll(/id: '([^']+)'/g)].map(m => m[1])
}

function readThemeSectionNames(file: string): string[] {
  const src = readFileSync(resolve(ROOT, file), 'utf-8')
  return [...src.matchAll(/\[data-theme="([^"]+)"\] \{/g)].map(m => m[1])
}

function findDuplicateVarsInSection(file: string, sectionName: string): string[] {
  const src = readFileSync(resolve(ROOT, file), 'utf-8')
  const srcs = src.split(`[data-theme="${sectionName}"] {`)[1] || ''
  const body = (srcs.split('\n}')[0] || '')
  const names = [...body.matchAll(/^\s*--([\w-]+):/gm)].map(m => m[1])
  const dupes = names.filter((n, i) => names.indexOf(n) !== i)
  return [...new Set(dupes)]
}

describe('主题一致性守卫', () => {
  const registered = readThemeIdsFromSettingsPanel()
  const themesNames = readThemeSectionNames('themes.css')
  const designNames = readThemeSectionNames('design-tokens.css')

  it('快照：注册的主题清单（防意外丢失）', () => {
    expect(registered).toEqual([
      'modern-light', 'elegant-dark', 'monochrome-elegance', 'crimson-red',
      'emerald-green', 'elegant-purple', 'warm-amber', 'ocean-blue',
      'mediterranean-dream', 'spring-blossom',
    ])
    expect(registered.length).toBe(10)
  })

  it('规则1：注册的每个主题在 CSS 中有样式段（不允许无样式主题）', () => {
    const known = new Set([...themesNames, ...designNames])
    const missing = registered.filter(id => !known.has(id))
    expect(missing).toEqual([])
  })

  it('规则2：CSS 中不允许存在未注册的死段', () => {
    const registeredSet = new Set(registered)
    const deadInThemes = themesNames.filter(n => !registeredSet.has(n))
    const deadInDesign = designNames.filter(n => !registeredSet.has(n))
    expect(deadInThemes).toEqual([])
    expect(deadInDesign).toEqual([])
  })

  it('规则3：design-tokens 主题段内无重复变量定义', () => {
    for (const name of designNames) {
      const dupes = findDuplicateVarsInSection('design-tokens.css', name)
      expect(dupes).toEqual([])
    }
  })
})
