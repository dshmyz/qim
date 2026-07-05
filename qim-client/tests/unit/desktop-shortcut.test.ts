import { describe, test, expect } from 'vitest'
import { execSync } from 'node:child_process'
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs'
import { resolve } from 'node:path'
import { tmpdir } from 'node:os'

// after-install-linux.sh 里定义了 resolve_desktop_dir，是桌面快捷方式逻辑里
// 最容易出错的环节（XDG_DESKTOP_DIR 解析、中文目录回退）。这里用 source 加载
// 函数后直接调用，验证真实 shell 行为，而非字符串匹配。
const SCRIPT = resolve(__dirname, '../../electron/after-install-linux.sh')

function resolveDesktopDir(home: string): string {
  // QIM_MAIN=0 跳过主流程，仅加载函数；再调用 resolve_desktop_dir
  const cmd = `QIM_MAIN=0; . '${SCRIPT}'; resolve_desktop_dir '${home}'`
  return execSync(cmd, { shell: '/bin/sh' }).toString().trim()
}

function makeHome(): string {
  return mkdtempSync(resolve(tmpdir(), 'qim-desktop-test-'))
}

describe('resolve_desktop_dir', () => {
  test('优先 XDG_DESKTOP_DIR（中文桌面目录）', () => {
    const home = makeHome()
    try {
      mkdirSync(resolve(home, '.config'), { recursive: true })
      mkdirSync(resolve(home, '桌面'), { recursive: true })
      writeFileSync(resolve(home, '.config/user-dirs.dirs'), 'XDG_DESKTOP_DIR="$HOME/桌面"\n')
      expect(resolveDesktopDir(home)).toBe(`${home}/桌面`)
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  test('XDG_DESKTOP_DIR 指向不存在目录时回退到 Desktop', () => {
    const home = makeHome()
    try {
      mkdirSync(resolve(home, '.config'), { recursive: true })
      mkdirSync(resolve(home, 'Desktop'), { recursive: true })
      writeFileSync(resolve(home, '.config/user-dirs.dirs'), 'XDG_DESKTOP_DIR="$HOME/NonExistent"\n')
      expect(resolveDesktopDir(home)).toBe(`${home}/Desktop`)
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  test('无 user-dirs.dirs 时回退到 Desktop', () => {
    const home = makeHome()
    try {
      mkdirSync(resolve(home, 'Desktop'), { recursive: true })
      expect(resolveDesktopDir(home)).toBe(`${home}/Desktop`)
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  test('仅有「桌面」目录时也能识别', () => {
    const home = makeHome()
    try {
      mkdirSync(resolve(home, '桌面'), { recursive: true })
      expect(resolveDesktopDir(home)).toBe(`${home}/桌面`)
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  test('无任何桌面目录时返回空', () => {
    const home = makeHome()
    try {
      expect(resolveDesktopDir(home)).toBe('')
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })
})
