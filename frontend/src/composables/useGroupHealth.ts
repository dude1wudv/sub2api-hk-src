import { computed, onScopeDispose, ref, watch, type Ref } from 'vue'
import { getMatrix, type MonitorMatrixRow } from '@/api/channelMonitorV2'

export type GroupHealth = 'healthy' | 'warning' | 'critical' | 'unknown'

export function latestGroupHealth(row: MonitorMatrixRow, dataThrough: string, bucketSeconds: number): GroupHealth {
  const latest = row.buckets.reduce<(typeof row.buckets)[number] | undefined>(
    (a, b) => !a || Date.parse(b.bucket_start) > Date.parse(a.bucket_start) ? b : a, undefined,
  )
  if (!latest || !Number.isFinite(Date.parse(latest.bucket_start))) return 'unknown'
  const age = Date.now() - Date.parse(dataThrough)
  if (!Number.isFinite(age) || age < -60_000 || age > Math.max(120, bucketSeconds * 2) * 1000) return 'unknown'
  if (Date.parse(dataThrough) - Date.parse(latest.bucket_start) > bucketSeconds * 2000) return 'unknown'
  const status = latest.health?.overall
  return status === 'healthy' || status === 'warning' || status === 'critical' ? status : 'unknown'
}

// Component-local data never survives a user/session change. This reads existing
// aggregates; it does not trigger provider probes.
export function useGroupHealth(open: Ref<boolean>) {
  const values = ref<Record<number, GroupHealth>>({})
  let controller: AbortController | undefined
  let loadedAt = 0
  const load = async () => {
    if (!open.value || (loadedAt !== 0 && Date.now() - loadedAt < 60_000)) return
    controller?.abort()
    const current = new AbortController()
    controller = current
    try {
      const matrix = await getMatrix({ range: '90m', platforms: [], groupIds: [], models: [] }, 'platform_group', false, current.signal)
      if (current.signal.aborted) return
      values.value = Object.fromEntries(matrix.items.filter(row => row.group_id != null).map(row => [
        row.group_id!, latestGroupHealth(row, matrix.coverage.data_through, matrix.coverage.bucket_seconds),
      ]))
    } catch {
      if (!current.signal.aborted) values.value = {}
    } finally {
      if (!current.signal.aborted) loadedAt = Date.now()
    }
  }
  watch(open, active => {
    if (!active) controller?.abort()
    else void load()
  }, { immediate: true })
  const timer = setInterval(() => { void load() }, 60_000)
  onScopeDispose(() => {
    controller?.abort()
    clearInterval(timer)
  })
  return computed(() => values.value)
}
