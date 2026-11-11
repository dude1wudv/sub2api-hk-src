import { apiClient } from './client'
export interface InflightItem {
  request_id: string; user_id: number; api_key_id: number; api_key_name?: string; group_id: number; group_name?: string; model: string
  started_at: string; updated_at?: string; first_token_ms?: number | null; input_tokens?: number | null; output_tokens?: number | null
  tokens_estimated?: boolean; reserved_amount?: number | null; stream?: boolean; websocket?: boolean
  email?: string; account_id?: number; account_name?: string; upstream_endpoint?: string
}
export interface InflightResponse { items: InflightItem[]; total: number; page: number; page_size: number; runtime_known: boolean }
export async function getInflight(admin: boolean, params: { page: number; page_size: number; model?: string; request_id?: string; group_id?: number; api_key_id?: number }, signal?: AbortSignal): Promise<InflightResponse> {
  const { data } = await apiClient.get<InflightResponse>(admin ? '/admin/usage/inflight' : '/usage/inflight', { params, signal })
  return data
}
