import { afterEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { useColorMode } from '../useColorMode'
import { useChartPalette, useChartTheme } from '../useChartPalette'

afterEach(() => { vi.restoreAllMocks(); useColorMode().setMode('light') })

describe('chart palette', () => {
  it('reads root hex tokens again after a color-mode switch, including canvas labels and tooltips', () => {
    const { setMode } = useColorMode()
    setMode('light')
    vi.spyOn(window, 'getComputedStyle').mockImplementation(() => ({
      getPropertyValue: (name: string) => {
        const dark = document.documentElement.classList.contains('dark')
        if (name === '--chart-1') return dark ? '#AABBCC' : '#112233'
        if (name === '--chart-text') return dark ? '#eeeeee' : '#222222'
        if (name === '--chart-tooltip-bg') return dark ? '#242424' : '#ffffff'
        return ''
      },
    }) as CSSStyleDeclaration)
    const palette = useChartPalette(), theme = useChartTheme()
    expect(palette.value[0]).toBe('#112233')
    expect(theme.value.text).toBe('#222222')
    setMode('dark')
    expect(palette.value[0]).toBe('#aabbcc')
    expect(theme.value.text).toBe('#eeeeee')
    expect(theme.value.tooltip.backgroundColor).toBe('#242424')
    expect(palette.value.every(color => /^#[0-9a-f]{6}$/.test(color))).toBe(true)
  })

  it('uses valid mode defaults when a root token is absent or malformed', () => {
    useColorMode().setMode('dark')
    vi.spyOn(window, 'getComputedStyle').mockReturnValue({ getPropertyValue: () => 'rgb(1 2 3)' } as unknown as CSSStyleDeclaration)
    const palette = useChartPalette().value
    const theme = useChartTheme().value
    expect(palette).toHaveLength(10)
    expect(palette.every(color => /^#[0-9a-f]{6}$/.test(color))).toBe(true)
    expect(`${palette[0]}20`).toMatch(/^#[0-9a-f]{8}$/)
    expect(theme.tooltip.backgroundColor).toBe('#f4f1e8')
    expect(theme.tooltip.titleColor).toBe('#22231f')
    expect(theme.tooltip.borderColor).toBe('#cdcdbf')
  })

  it('uses the light tooltip surface when a root token is absent', () => {
    useColorMode().setMode('light')
    vi.spyOn(window, 'getComputedStyle').mockReturnValue({ getPropertyValue: () => '' } as unknown as CSSStyleDeclaration)
    const theme = useChartTheme().value
    expect(theme.tooltip.backgroundColor).toBe('#fffffc')
    expect(theme.tooltip.titleColor).toBe('#22231f')
    expect(theme.tooltip.bodyColor).toBe('#22231f')
    expect(theme.tooltip.borderColor).toBe('#e8e4da')
  })

  it('declares every chart root token as six-digit hex in both modes', () => {
    const css = readFileSync(resolve(process.cwd(), 'src/styles/design-system.css'), 'utf8')
    const tokens = [...css.matchAll(/(--chart-[\w-]+)\s*:\s*([^;]+);/g)]
    const required = [...Array.from({ length: 10 }, (_, i) => `--chart-${i + 1}`), '--chart-text', '--chart-grid', '--chart-tooltip-bg', '--chart-tooltip-text', '--chart-tooltip-border']
    for (const name of required) {
      const values = tokens.filter(token => token[1] === name)
      expect(values.length, name).toBe(2)
      for (const token of values) expect(token[2].trim(), name).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })
})
