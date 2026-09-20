import { describe, expect, it } from 'vitest'
import { readFileSync, existsSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(__dirname, '../..')

describe('win7 electron main entry config', () => {
  it('uses the unified CommonJS bundle entry for Electron 22', () => {
    const config = JSON.parse(readFileSync(resolve(root, 'electron-builder.win7.json'), 'utf8'))

    expect(config.electronVersion).toBe('22.3.27')
    expect(config.extraMetadata?.main).toBe('electron/main.cjs')
  })

  it('no longer ships the obsolete win7-specific bootstrap', () => {
    const config = JSON.parse(readFileSync(resolve(root, 'electron-builder.win7.json'), 'utf8'))

    expect(config.files).not.toContain('electron/main.win7.cjs')
    expect(existsSync(resolve(root, 'electron/main.win7.cjs'))).toBe(false)
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

  it('builds the CJS bundle before packing for every platform', () => {
    const pkg = JSON.parse(readFileSync(resolve(root, 'package.json'), 'utf8'))

    expect(pkg.scripts['electron:build:win7']).toBe('node scripts/build-main.cjs && electron-builder --win --x64 -c electron-builder.win7.json --publish never')
    expect(pkg.scripts['electron:build']).toContain('node scripts/build-main.cjs &&')
  })

  it('does not require a top-level esbuild on the Win7 build machine', () => {
    const pkg = JSON.parse(readFileSync(resolve(root, 'package.json'), 'utf8'))

    expect(pkg.scripts['build:electron-main:win7']).toBeUndefined()
    expect(pkg.devDependencies.esbuild).toBeUndefined()
  })

  it('unified entry requires the generated bundle and logs startup failure', () => {
    const bootstrap = readFileSync(resolve(root, 'electron/main.cjs'), 'utf8')

    expect(bootstrap).toContain("require('./main.bundle.cjs')")
    expect(bootstrap).toContain('qim-main-error.log')
  })
})
