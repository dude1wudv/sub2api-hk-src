/**
 * Admin Charity Pool API endpoints
 * Manages free.xjtu.shop charity pool settings, groups, sub-quotas, concurrency and key rotation.
 */

import { apiClient } from '../client'

export interface CharityPoolSettings {
  enabled: boolean
  daily_budget_usd: number
  max_concurrency: number
  per_ip_daily_limit_usd: number
  per_ip_max_concurrency: number
  per_ip_rpm_limit: number
  auto_rotate_daily: boolean
  rotation_timezone: string
  site_title: string
  site_notice: string
  host_allowlist: string[]
  system_user_id?: number | null
  created_at?: string
  updated_at?: string
}

export interface CharityAdminGroup {
  id: number
  group_id: number
  display_name: string
  description: string
  enabled: boolean
  daily_limit_usd: number | null
  max_concurrency: number
  sort_order: number
  active_api_key_id?: number | null
  active_api_key: string
  last_rotated_date: string
  last_rotated_at?: string | null
  group_name: string
  group_platform: string
  group_status: string
  account_count: number
  active_account_count: number
  used_today_usd: number
  request_count_today: number
  total_tokens_today: number
  current_concurrency: number
}

export interface CharityDailyTrendPoint {
  usage_date: string
  total_cost_usd: number
  request_count: number
  total_tokens: number
}

export interface CharityAdminOverview {
  settings: CharityPoolSettings
  usage_date: string
  next_reset_at: string
  used_today_usd: number
  remaining_today_usd: number | null
  request_count_today: number
  total_tokens_today: number
  current_concurrency: number
  groups: CharityAdminGroup[]
  trend: CharityDailyTrendPoint[]
}

export interface UpdateCharitySettingsPayload {
  enabled: boolean
  daily_budget_usd: number
  max_concurrency: number
  per_ip_daily_limit_usd: number
  per_ip_max_concurrency: number
  per_ip_rpm_limit: number
  auto_rotate_daily: boolean
  rotation_timezone: string
  site_title: string
  site_notice: string
  host_allowlist: string[]
}

export interface CreateCharityGroupPayload {
  group_id?: number | null
  create_group_name?: string
  create_group_platform?: string
  create_group_description?: string
  display_name: string
  description: string
  enabled: boolean
  daily_limit_usd: number | null
  max_concurrency: number
  sort_order: number
}

export interface UpdateCharityGroupPayload {
  display_name: string
  description: string
  enabled: boolean
  daily_limit_usd: number | null
  max_concurrency: number
  sort_order: number
}

export async function getOverview(): Promise<CharityAdminOverview> {
  const { data } = await apiClient.get<CharityAdminOverview>('/admin/charity/overview')
  return data
}

export async function updateSettings(payload: UpdateCharitySettingsPayload): Promise<CharityPoolSettings> {
  const { data } = await apiClient.put<CharityPoolSettings>('/admin/charity/settings', payload)
  return data
}

export async function addGroup(payload: CreateCharityGroupPayload): Promise<CharityAdminGroup> {
  const { data } = await apiClient.post<CharityAdminGroup>('/admin/charity/groups', payload)
  return data
}

export async function updateGroup(id: number, payload: UpdateCharityGroupPayload): Promise<CharityAdminGroup> {
  const { data } = await apiClient.put<CharityAdminGroup>(`/admin/charity/groups/${id}`, payload)
  return data
}

export async function removeGroup(id: number): Promise<void> {
  await apiClient.delete(`/admin/charity/groups/${id}`)
}

export async function rotateGroupKey(id: number): Promise<CharityAdminGroup> {
  const { data } = await apiClient.post<CharityAdminGroup>(`/admin/charity/groups/${id}/rotate-key`)
  return data
}

export async function rotateAllKeys(): Promise<{ groups: CharityAdminGroup[] }> {
  const { data } = await apiClient.post<{ groups: CharityAdminGroup[] }>('/admin/charity/rotate-all')
  return data
}

export const adminCharityAPI = {
  getOverview,
  updateSettings,
  addGroup,
  updateGroup,
  removeGroup,
  rotateGroupKey,
  rotateAllKeys
}

export default adminCharityAPI
