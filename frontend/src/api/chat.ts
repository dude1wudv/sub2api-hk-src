import { apiClient } from './client'
import type { UserSupportedModelPricing } from './channels'

export interface ChatCapabilities {
  transport: 'chat_completions' | 'messages' | 'responses' | 'gemini'
  image_input: boolean
  document_input: boolean
  native_deepseek_extensions: boolean
  native_files: boolean
  native_search: boolean
  native_read_url: boolean
  reasoning: boolean
  reasoning_efforts?: ChatReasoningEffort[]
  tool_call: boolean
  search: boolean
  read_url: boolean
  voice: false
  reasons: Record<string, string>
}
export interface ChatCatalogGroup {
  id: number
  name: string
  platform: string
  subscription_type: string
  rate_multiplier: number
  user_rate_multiplier: number
}
export interface ChatCatalogModel {
  id: string
  pricing: UserSupportedModelPricing | null
  capabilities: ChatCapabilities
  available: boolean
  unavailable_reason: string
}
export interface ChatCatalog { group: ChatCatalogGroup; models: ChatCatalogModel[] }
export type ChatReasoningEffort = 'low' | 'medium' | 'high'
export interface ChatPreset {
  id: string
  group_id: number
  model: string
  display_name: string
  sort_order: number
  is_default: boolean
  normal_reasoning_effort: ChatReasoningEffort
  thinking_reasoning_effort: ChatReasoningEffort
}
export interface ChatPresetView extends ChatPreset {
  group: ChatCatalogGroup
  capabilities: ChatCapabilities
  available: boolean
  unavailable_reason: string
}
export interface ChatPresetBinding { group_id: number; model: string; display_name?: string }
export type ChatPresetPatch = Partial<ChatPresetBinding> & { is_default?: boolean; normal_reasoning_effort?: ChatReasoningEffort; thinking_reasoning_effort?: ChatReasoningEffort }

export const chatAPI = {
  async catalog(groupId: number, search = '', signal?: AbortSignal): Promise<ChatCatalog> {
    return (await apiClient.get<ChatCatalog>('/chat/catalog', { params: { group_id: groupId, search }, signal })).data
  },
  async presets(signal?: AbortSignal): Promise<ChatPresetView[]> {
    return (await apiClient.get<ChatPresetView[]>('/chat/presets', { signal })).data
  },
  async createPreset(binding: ChatPresetBinding): Promise<ChatPreset> {
    return (await apiClient.post<ChatPreset>('/chat/presets', binding)).data
  },
  async updatePreset(id: string, patch: ChatPresetPatch): Promise<ChatPreset> {
    return (await apiClient.patch<ChatPreset>(`/chat/presets/${encodeURIComponent(id)}`, patch)).data
  },
  async orderPresets(ids: string[]): Promise<void> {
    await apiClient.put('/chat/presets/order', { ids })
  },
  async deletePreset(id: string): Promise<void> {
    await apiClient.delete(`/chat/presets/${encodeURIComponent(id)}`)
  },
  async reconnectPreset(id: string): Promise<void> {
    await apiClient.post(`/chat/presets/${encodeURIComponent(id)}/reconnect`)
  },
}
