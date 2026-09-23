import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAppearance } from '../useAppearance'
import { useGlacierChartPalette } from '../useGlacierChartPalette'

afterEach(() => {
  useAppearance().isDark.value = false
  useAppearance().setStyle('aurora')
  vi.restoreAllMocks()
})

describe('shared chart palette', () => {
  it('reads the new theme and mode, keeping alpha-compatible hex colors', () => {
    const appearance = useAppearance()
    vi.spyOn(window, 'getComputedStyle').mockImplementation(() => ({
      getPropertyValue: (name: string) => name.startsWith('--chart-') ? '' : name.endsWith('300') ? '219 169 133'
        : document.documentElement.dataset.style === 'graphite' ? '150 84 49' : '94 96 205'
    }) as CSSStyleDeclaration)
    const palette = useGlacierChartPalette(['#000000'])
    appearance.setStyle('aurora')
    expect(palette.value[0]).toBe('#5e60cd')
    appearance.setStyle('graphite')
    expect(palette.value[0]).toBe('#965431')
    appearance.isDark.value = true
    expect(palette.value[0]).toBe('#dba985')
    for (const color of palette.value) expect(color + '20').toMatch(/^#[0-9a-f]{8}$/i)
  })
  it('uses complete material palettes and falls back for invalid or absent entries', () => {
    vi.spyOn(window, 'getComputedStyle').mockImplementation(() => ({
      getPropertyValue: (name: string) => ({
        '--color-primary-500': '39 91 157', '--color-accent-500': '49 111 123',
        '--chart-1': '107 87 149', '--chart-2': '256 0 0', '--chart-3': '129 194 159'
      }[name] || '')
    }) as CSSStyleDeclaration)
    const palette = useGlacierChartPalette(['#000000']).value
    expect(palette.slice(0, 3)).toEqual(['#6b5795', '#316f7b', '#81c29f'])
    expect(palette[3]).toBe('#b68b4c')
  })
})
