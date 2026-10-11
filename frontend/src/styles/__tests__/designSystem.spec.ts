import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const stylesheet = readFileSync(resolve(process.cwd(), 'src/styles/design-system.css'), 'utf8')
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
    return v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4
  })
  return channels.reduce((sum, channel, index) => sum + channel * [.2126, .7152, .0722][index], 0)
}
function contrast(foreground: string, background: string, palette: Record<string, string>): number {
  const a = luminance(foreground, palette)
  const b = luminance(background, palette)
  return (Math.max(a, b) + .05) / (Math.min(a, b) + .05)
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
