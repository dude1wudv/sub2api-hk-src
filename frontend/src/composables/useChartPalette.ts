import { computed } from 'vue'
import { useColorMode } from './useColorMode'

/**
 * Validated CVD-safe 8-slot categorical palette (OKLab adjacent ΔE >= 8 under deuteranopia/protanopia/tritanopia,
 * normal-vision ΔE >= 19 in both light and dark modes), plus 2 overflow slots when callers have not folded to "Other".
 */
const lightPalette = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#6250d6', '#e34948', '#0d9488', '#2563eb']
const darkPalette = ['#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#008300', '#9085e9', '#e66767', '#2dd4bf', '#60a5fa']
const hexColor = /^#[0-9a-f]{6}$/i

function readColor(tokens: CSSStyleDeclaration | undefined, name: string, fallback: string) {
  const value = tokens?.getPropertyValue(name).trim() ?? ''
  return hexColor.test(value) ? value.toLowerCase() : fallback
}

/** Chart.js requires resolved colors; six-digit hex also supports alpha suffixes. */
export function useChartPalette() {
  const { isDark } = useColorMode()
  return computed(() => {
    const fallback = isDark.value ? darkPalette : lightPalette
    const tokens = typeof document === 'undefined' ? undefined : getComputedStyle(document.documentElement)
    return fallback.map((color, index) => readColor(tokens, `--chart-${index + 1}`, color))
  })
}

export function useChartTheme() {
  const { isDark } = useColorMode()
  return computed(() => {
    const dark = isDark.value
    const tokens = typeof document === 'undefined' ? undefined : getComputedStyle(document.documentElement)
    const text = readColor(tokens, '--chart-text', dark ? '#94a3b8' : '#475569')
    const tooltipText = readColor(tokens, '--chart-tooltip-text', dark ? '#f8fafc' : '#0f172a')
    return {
      text,
      grid: readColor(tokens, '--chart-grid', dark ? '#1e293b' : '#e2e8f0'),
      pointBorder: dark ? '#11161d' : '#ffffff',
      tooltip: {
        backgroundColor: readColor(tokens, '--chart-tooltip-bg', dark ? '#1e293b' : '#ffffff'),
        titleColor: tooltipText,
        bodyColor: tooltipText,
        footerColor: tooltipText,
        borderColor: readColor(tokens, '--chart-tooltip-border', dark ? '#334155' : '#cbd5e1'),
        borderWidth: 1,
        padding: 10,
        cornerRadius: 10,
      },
    }
  })
}
