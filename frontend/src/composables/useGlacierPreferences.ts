import { computed, ref, type ComputedRef, type InjectionKey } from 'vue'
import { useAppearance } from './useAppearance'

export type TableDensity = 'comfortable' | 'compact'
export const tableDensityKey: InjectionKey<ComputedRef<TableDensity | undefined>> = Symbol('table-density')
const density = ref<TableDensity>('comfortable')
let initialized = false

export function useGlacierPreferences() {
  if (!initialized) {
    try {
      density.value = localStorage.getItem('glacier-table-density') === 'compact' ? 'compact' : 'comfortable'
    } catch { /* Storage may be unavailable. */ }
    initialized = true
  }
  const { style } = useAppearance()
  const isGlassLayout = computed(() => ['aurora', 'lagoon', 'graphite', 'glacier'].includes(style.value))
  const effectiveDensity = computed(() => isGlassLayout.value ? density.value : undefined)
  function setDensity(value: TableDensity) {
    density.value = value
    try { localStorage.setItem('glacier-table-density', value) } catch { /* Keep session preference. */ }
  }
  return { density, isGlassLayout, effectiveDensity, setDensity }
}
