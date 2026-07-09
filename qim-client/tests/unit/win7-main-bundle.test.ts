import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(__dirname, '../..')

describe('win7 electron main entry config', () => {
  it('uses a CommonJS bootstrap entry for Electron 22', () => {
    const config = JSON.parse(readFileSync(resolve(root, 'electron-builder.win7.json'), 'utf8'))

    expect(config.electronVersion).toBe('22.3.27')
    expect(config.extraMetadata?.main).toBe('electron/main.win7.cjs')
    expect(config.files).toContain('electron/main.win7.cjs')
  })

  it('builds the Win7 package as an NSIS installer with the Win7-specific config', () => {
    const config = JSON.parse(readFileSync(resolve(root, 'electron-builder.win7.json'), 'utf8'))

    expect(config.win?.target).toEqual([
      {
        target: 'nsis',
        arch: ['x64'],
      },
    ])
    expect(config.nsis).toMatchObject({
      oneClick: false,
      perMachine: false,
      allowElevation: true,
      allowToChangeInstallationDirectory: true,
      createDesktopShortcut: true,
      createStartMenuShortcut: true,
    })
  })

  it('does not require esbuild on the Win7 build machine', () => {
    const pkg = JSON.parse(readFileSync(resolve(root, 'package.json'), 'utf8'))

    expect(pkg.scripts['build:electron-main:win7']).toBeUndefined()
    expect(pkg.scripts['electron:build:win7']).toBe('electron-builder --win --x64 -c electron-builder.win7.json --publish never')
    expect(pkg.devDependencies.esbuild).toBeUndefined()
  })

  it('bootstraps the existing ESM main entry with dynamic import', () => {
    const bootstrap = readFileSync(resolve(root, 'electron/main.win7.cjs'), 'utf8')

    expect(bootstrap).toContain("path.join(__dirname, 'main.js')")
    expect(bootstrap).toContain('import(mainUrl)')
  })
})
