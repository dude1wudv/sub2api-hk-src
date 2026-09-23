import { computed } from 'vue'
import { useAppearance } from './useAppearance'

/** Read the active theme after appearance changes; Chart.js cannot resolve CSS variables. */
export function useGlacierChartPalette(fallback: string[]) {
  const { style, isDark } = useAppearance()
  return computed(() => {
    if (!style.value || typeof document === 'undefined') return fallback
    const tokens = getComputedStyle(document.documentElement)
    const shade = isDark.value ? '300' : '500'
    const colors = [`--color-primary-${shade}`, `--color-accent-${shade}`]
      .map(name => tokens.getPropertyValue(name).trim())
    if (colors.some(value => !value)) return fallback
    const palette = [
      // Consumers append an alpha byte, so keep all entries in six-digit hex.
      ...colors.map(value => '#' + value.split(/\s+/).map(channel => Number(channel).toString(16).padStart(2, '0')).join('')),
      ...(isDark.value
        ? ['#b7a8f2', '#e9bb7c', '#7bceac', '#e59fae', '#8faecc', '#a2b8f5', '#b0d394', '#d1b2c9']
        : ['#8c79bc', '#b68b4c', '#479c7b', '#bd7487', '#6c8ba7', '#688ac7', '#7d9857', '#997d9b'])
    ]
    // Material themes supply a full categorical palette; preserve glacier's
    // existing colors and the six-digit format used by alpha-byte consumers.
    return palette.map((fallbackColor, index) => {
      const channels = tokens.getPropertyValue(`--chart-${index + 1}`).trim().split(/\s+/).map(Number)
      return channels.length === 3 && channels.every(value => Number.isInteger(value) && value >= 0 && value <= 255)
        ? '#' + channels.map(value => value.toString(16).padStart(2, '0')).join('')
        : fallbackColor
    })
  })
}
