import { readdirSync, readFileSync, statSync } from 'node:fs'
import { relative, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const stylesheet = readFileSync(resolve(process.cwd(), 'src/styles/atelier/tokens.css'), 'utf8')
function tokens(selector: string): Record<string, string> {
  const block = stylesheet.slice(stylesheet.indexOf(`${selector} {`)).split('}')[0]
  return Object.fromEntries([...block.matchAll(/(--[\w-]+):\s*([^;]+);/g)].map(([, key, value]) => [key, value.trim()]))
}
const light = tokens(':root')
const dark = { ...light, ...tokens(':root.dark') }
function luminance(name: string, palette: Record<string, string>): number {
  const value = palette[name].replace(/var\((--[\w-]+)\)/g, (_, key: string) => palette[key])
  const channels = value.split(/\s+/).map(Number).map(channel => {
    const v = channel / 255
    return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4
  })
  return channels.reduce((sum, channel, index) => sum + channel * [0.2126, 0.7152, 0.0722][index], 0)
}
function contrast(foreground: string, background: string, palette: Record<string, string>): number {
  const a = luminance(foreground, palette)
  const b = luminance(background, palette)
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
}

describe.each([['light', light], ['dark', dark]] as const)('%s semantic text contrast', (_, palette) => {
  it.each(['--canvas', '--surface', '--surface-2', '--surface-elevated'])('keeps normal and supporting text readable on %s', background => {
    for (const foreground of ['--ink', '--ink-2', '--ink-3']) {
      expect(contrast(foreground, background, palette)).toBeGreaterThanOrEqual(4.5)
    }
  })
  it('keeps primary actions and status labels readable', () => {
    expect(contrast('--action-text', '--action-bg', palette)).toBeGreaterThanOrEqual(4.5)
    for (const status of ['ok', 'warn', 'err', 'link']) {
      expect(contrast(`--text-${status}`, `--status-${status}-bg`, palette)).toBeGreaterThanOrEqual(4.5)
    }
  })
})

describe('theme surfaces', () => {
  function hexLuminance(hex: string): number {
    const channels = [0, 2, 4].map(offset => Number.parseInt(hex.slice(offset + 1, offset + 3), 16)).map(channel => {
      const value = channel / 255
      return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
    })
    return channels.reduce((sum, channel, index) => sum + channel * [0.2126, 0.7152, 0.0722][index], 0)
  }
  function hexContrast(foreground: string, background: string): number {
    const a = hexLuminance(foreground)
    const b = hexLuminance(background)
    return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
  }

  it('uses crisp chart tooltip surfaces and keeps both modes readable', () => {
    expect(light['--chart-tooltip-bg']).toBe('#ffffff')
    expect(light['--chart-tooltip-text']).toBe('#14171f')
    expect(hexContrast(light['--chart-tooltip-text'], light['--chart-tooltip-bg'])).toBeGreaterThanOrEqual(4.5)
    expect(hexContrast(dark['--chart-tooltip-text'], dark['--chart-tooltip-bg'])).toBeGreaterThanOrEqual(4.5)
  })
})

describe('light mode surfaces', () => {
  const solidDarkBackground = /(?:[\w-]+:)*!?(?:bg-(?:gray|dark)-(?:700|800|900|950)|bg-black)(?![\w/-])/g
  const allowedVideoSurface = 'components/admin/account/AccountTestModal.vue'

  function vueFiles(dir: string): string[] {
    return readdirSync(dir).flatMap(name => {
      const full = resolve(dir, name)
      return statSync(full).isDirectory() ? vueFiles(full) : name.endsWith('.vue') ? [full] : []
    })
  }

  it('does not use a solid dark background without a dark-mode prefix', () => {
    const root = resolve(process.cwd(), 'src')
    const violations: string[] = []
    for (const file of vueFiles(root)) {
      const rel = relative(root, file).replace(/\\/g, '/')
      readFileSync(file, 'utf8').split('\n').forEach((line, index) => {
        for (const match of line.matchAll(solidDarkBackground)) {
          if (match[0].includes('dark:')) continue
          if (rel === allowedVideoSurface && match[0] === 'bg-black' && line.includes('border-gray-200 bg-black')) continue
          violations.push(`${rel}:${index + 1} ${match[0]}`)
        }
      })
    }
    expect(violations).toEqual([])
  })
})
