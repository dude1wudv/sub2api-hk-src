import { apiClient } from './client'

export interface ExternalQualityEvent {
  id: string | number
  model_id: string
  status: 'success' | 'degraded'
  score: number | null
  created_at: string
  response_text?: string
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
