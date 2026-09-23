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
      getPropertyValue: (name: string) => name.endsWith('300') ? '219 169 133'
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
})
