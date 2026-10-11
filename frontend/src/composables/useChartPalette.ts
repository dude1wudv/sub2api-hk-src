import { computed } from 'vue'
import { useColorMode } from './useColorMode'

const lightPalette = ['#3876b5', '#337a63', '#967122', '#8062a8', '#b75363', '#2f7d87', '#866746', '#626d8c', '#6b7540', '#aa6238']
const darkPalette = ['#85b5e5', '#8dc9ad', '#d6ba76', '#b7a0dc', '#e397a7', '#80c5ce', '#c7ab88', '#a2aed6', '#bcc88c', '#dda778']
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
    const text = readColor(tokens, '--chart-text', dark ? '#b3b4a9' : '#6f7068')
    const tooltipText = readColor(tokens, '--chart-tooltip-text', dark ? '#22231f' : '#fffdf7')
    return {
      text,
      grid: readColor(tokens, '--chart-grid', dark ? '#393b34' : '#e8e4da'),
      tooltip: {
        backgroundColor: readColor(tokens, '--chart-tooltip-bg', dark ? '#f4f1e8' : '#22231f'),
        titleColor: tooltipText,
        bodyColor: tooltipText,
        footerColor: tooltipText,
        borderColor: readColor(tokens, '--chart-tooltip-border', dark ? '#cdcdbf' : '#393b33'),
        borderWidth: 1,
      },
    }
  })
}
