export interface SlowTTFTConfig {
  enabled: boolean
  threshold_seconds: number
  consecutive_count: number
  window_seconds: number
  window_count: number
  pause_seconds: number
}
export const defaultSlowTTFT = (): SlowTTFTConfig => ({
  enabled: false, threshold_seconds: 15, consecutive_count: 2,
  window_seconds: 300, window_count: 3, pause_seconds: 1800
})
export function readSlowTTFT(extra: Record<string, unknown> | null | undefined): SlowTTFTConfig {
  const value = extra?.slow_ttft_protection
  return { ...defaultSlowTTFT(), ...(typeof value === 'object' && value !== null ? value : {}) }
}
