import { apiClient } from './client'

export interface ExternalQualityEvent {
  id: string | number
  model_id: string
  status: 'completed' | 'success' | 'degraded'
  score: number | null
  created_at: string
  response_text?: string
  response_model?: string
  finished_at?: string
  duration_ms?: number
  has_artwork?: boolean
  audit_summary?: { score: number | null; sections?: Array<{ name: string; status?: string; score?: number | null }> }
}
export interface ExternalQualityReport {
  group_id: number
  model: string
  updated_at: string | null
  last_success_at: string | null
  status: 'ready' | 'stale' | 'unavailable'
  events: ExternalQualityEvent[]
}
export async function getExternalQuality(groupId: number, signal?: AbortSignal) {
  const { data } = await apiClient.get<ExternalQualityReport>('/channel-monitor-v2/external-quality', { params: { group_id: groupId }, signal })
  return data
}

export async function getExternalQualityArtwork(groupId: number, eventId: string | number, signal?: AbortSignal) {
  const { data } = await apiClient.get<{ id: number; group_id: number; response_text: string }>(
    `/channel-monitor-v2/external-quality/${encodeURIComponent(eventId)}/artwork`,
    { params: { group_id: groupId }, signal },
  )
  return data
}
