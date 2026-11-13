import { apiClient } from './client'
import { authenticatedChatFetch, requireChatResponse, streamAuthenticatedChat, ChatTransportError, chatStoredUserID, chatTokenUserID, chatRequestTransforms } from './authenticatedStream'
// The fork is the protocol authority; type-only imports do not bundle React into Vue.
import type { BRIDGE_VERSION, BridgePayloads, BridgeRequest, BridgeResponse, ChatBootstrap, ChatPresetView } from '../../../chat-demo/src/services/sub2apiBridgeContract'

export const CHAT_BRIDGE_VERSION: typeof BRIDGE_VERSION = 1
const ID = '[A-Za-z0-9_-]+'
const NONCE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
const record = (value: unknown): value is Record<string, unknown> => !!value && typeof value === 'object' && !Array.isArray(value)
const keysOnly = (value: Record<string, unknown>, allowed: string[]) => Object.keys(value).every((key) => allowed.includes(key))
const routes: Record<string, RegExp[]> = {
  GET: [/^\/presets$/, /^\/preferences$/, /^\/sessions$/, new RegExp(`^/sessions/${ID}$`)],
  POST: [/^\/imports$/, new RegExp(`^/imports/${ID}/commit$`), new RegExp(`^/sessions/${ID}/stop$`)],
  PUT: [/^\/preferences$/, new RegExp(`^/sessions/${ID}$`), new RegExp(`^/imports/${ID}/sessions/${ID}$`)],
  PATCH: [new RegExp(`^/sessions/${ID}$`)],
  DELETE: [/^\/sessions$/, new RegExp(`^/sessions/${ID}$`), new RegExp(`^/imports/${ID}$`)],
}

function safePath(value: unknown): string | null {
  if (typeof value !== 'string' || value.length > 4096 || !value.startsWith('/') || value.startsWith('//')
    || /[\\#\s\u0000-\u001f\u007f]/.test(value)) return null
  const [path, query, extra] = value.split('?')
  // No URL normalization, percent-decoding, fragments or encoded path separators.
  if (!path || path.includes('%') || extra !== undefined) return null
  if (query !== undefined) {
    if (path !== '/sessions' || !query) return null
    const params = new URLSearchParams(query)
    const seen = new Set<string>()
    for (const [key, item] of params) {
      if (seen.has(key) || (key !== 'limit' && key !== 'cursor')) return null
      if (key === 'limit' && (!/^\d{1,3}$/.test(item) || Number(item) < 1 || Number(item) > 100)) return null
      if (key === 'cursor' && (!item || item.length > 2048)) return null
      seen.add(key)
    }
  }
  return path
}

/** Finite, method-specific fork API. Never forward URLs, headers or credentials. */
export function isChatBridgeRequest(value: unknown, nonce: string): value is BridgeRequest {
  if (!record(value) || value.type !== 'request' || value.nonce !== nonce || typeof value.request_id !== 'string'
    || !/^[A-Za-z0-9_-]{1,128}$/.test(value.request_id) || typeof value.command !== 'string') return false
  const base = ['type', 'command', 'request_id', 'nonce']
  const only = (...keys: string[]) => keysOnly(value, [...base, ...keys])
  if (value.command === 'bootstrap' || value.command === 'logout') return only()
  if (value.command === 'cancel') return only('target_request_id') && typeof value.target_request_id === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(value.target_request_id)
  if (value.command === 'navigate') return only('path') && (value.path === '/dashboard' || value.path === '/chat-models')
  const path = safePath(value.path)
  if (!path) return false
  if (value.command !== 'json' && typeof value.path === 'string' && value.path.includes('?')) return false
  if (value.command === 'stream') return only('path', 'body') && record(value.body) && new RegExp(`^/sessions/${ID}/generate$`).test(path)
  if (value.command === 'download') return only('path') && new RegExp(`^/attachments/${ID}/content$`).test(path)
  if (value.command === 'upload') return only('path', 'body', 'filename', 'purpose')
    && (path === '/attachments' || new RegExp(`^/imports/${ID}/attachments$`).test(path))
    && value.body instanceof Blob && value.body.size <= 64 * 1024 * 1024
    && typeof value.filename === 'string' && value.filename.length > 0 && value.filename.length <= 1024
    && !/[\u0000-\u001f\u007f]/.test(value.filename)
    && (value.purpose === undefined || value.purpose === 'user_data' || value.purpose === 'rich_layout_recognition')
    && (value.purpose !== 'user_data' || value.body.size <= 32 * 1024 * 1024)
  return value.command === 'json' && only('method', 'path', 'body') && typeof value.method === 'string'
    && ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].includes(value.method)
    && (value.method !== 'GET' || value.body === undefined) && (value.method === 'GET' || !String(value.path).includes('?'))
    && (routes[value.method]?.some((route) => route.test(path)) ?? false)
}

/** Return only documented, noncredential preferences, even if storage contains old fields. */
export function chatPreferences(value: unknown): Record<string, unknown> {
  if (!record(value)) return {}
  const output: Record<string, unknown> = {}
  if (value.locale === 'zh-CN' || value.locale === 'en-US') output.locale = value.locale
  if (value.themePreference === 'light' || value.themePreference === 'dark' || value.themePreference === 'system') output.themePreference = value.themePreference
  if (value.ttsVoice === 'stella' || value.ttsVoice === 'tide') output.ttsVoice = value.ttsVoice
  for (const key of ['thinking', 'search', 'siderCollapsed']) if (typeof value[key] === 'boolean') output[key] = value[key]
  if (record(value.location) && typeof value.location.region === 'string' && value.location.region.length <= 64) {
    const location: Record<string, string> = { region: value.location.region }
    for (const key of ['province', 'city']) {
      const part = value.location[key]
      if (typeof part === 'string' && part.length <= 128) location[key] = part
    }
    output.location = location
  }
  return output
}

function presetView(value: unknown): ChatPresetView {
  if (!record(value) || typeof value.id !== 'string' || typeof value.group_id !== 'number' || typeof value.model !== 'string'
    || typeof value.display_name !== 'string' || typeof value.sort_order !== 'number' || typeof value.is_default !== 'boolean'
    || typeof value.available !== 'boolean' || typeof value.unavailable_reason !== 'string' || !record(value.group) || !record(value.capabilities)) {
    throw new ChatTransportError('聊天模型响应无效', 'CHAT_BRIDGE_PROTOCOL')
  }
  const group = value.group
  const cap = value.capabilities
  if (typeof group.id !== 'number' || typeof group.name !== 'string' || typeof group.platform !== 'string'
    || typeof group.subscription_type !== 'string' || typeof group.rate_multiplier !== 'number' || typeof group.user_rate_multiplier !== 'number'
    || !['chat_completions', 'messages', 'responses'].includes(String(cap.transport))) throw new ChatTransportError('聊天模型能力无效', 'CHAT_BRIDGE_PROTOCOL')
  const reasons: Record<string, string> = {}
  if (record(cap.reasons)) for (const [key, reason] of Object.entries(cap.reasons)) if (typeof reason === 'string') reasons[key] = reason
  // Each property is projected explicitly; account/key fields never enter the frame.
  return {
    id: value.id, group_id: value.group_id, model: value.model, display_name: value.display_name,
    sort_order: value.sort_order, is_default: value.is_default, available: value.available, unavailable_reason: value.unavailable_reason,
    group: { id: group.id, name: group.name, platform: group.platform, subscription_type: group.subscription_type, rate_multiplier: group.rate_multiplier, user_rate_multiplier: group.user_rate_multiplier },
    capabilities: {
      transport: cap.transport === 'messages' ? 'messages' : cap.transport === 'responses' ? 'responses' : 'chat_completions',
      image_input: cap.image_input === true, document_input: cap.document_input === true,
      native_deepseek_extensions: cap.native_deepseek_extensions === true, native_files: cap.native_files === true,
      native_search: cap.native_search === true, native_read_url: cap.native_read_url === true,
      reasoning: cap.reasoning === true, tool_call: cap.tool_call === true, search: cap.search === true, read_url: cap.read_url === true,
      voice: false, reasons,
    },
  }
}

interface ChatHostOptions {
  frame: () => HTMLIFrameElement | null
  userID: () => number | null
  locale: () => string
  navigate: (path: BridgePayloads['navigate']['path']) => void | Promise<unknown>
  logout: () => Promise<void>
  onSessionInvalid?: () => void
  host?: Window
  channel?: () => MessageChannel
}

export class ChatHostBridge {
  private readonly host: Window
  private port?: MessagePort
  private nonce = ''
  private userID: number | null = null
  private document?: Document
  private source?: Window
  private active = new Map<string, AbortController>()
  private seen = new Set<string>()
  private mounted = false
  private handshakes = new Set<string>()
  constructor(private readonly options: ChatHostOptions) { this.host = options.host ?? window }

  mount() {
    if (this.mounted) return
    this.mounted = true
    this.host.addEventListener('message', this.onReady)
    this.host.addEventListener('pagehide', this.disconnect)
    this.host.addEventListener('storage', this.onStorage)
  }

  private sessionMatches(userID: number | null): boolean {
    return !!userID && this.options.userID() === userID && chatStoredUserID(this.host.localStorage) === userID && chatTokenUserID(this.host.localStorage.getItem('auth_token')) === userID
  }

  private onStorage = (event: StorageEvent) => {
    if ((event.key === null || event.key === 'auth_user' || event.key === 'auth_token') && !this.sessionMatches(this.userID)) {
      this.disconnect()
      this.options.onSessionInvalid?.()
    }
  }

  /** A ready message can precede load. Only tear down a *different* document. */
  frameLoaded = () => {
    const frame = this.options.frame()
    try { if (this.document && frame?.contentDocument === this.document) return } catch { /* Navigated away. */ }
    this.disconnect()
  }

  private onReady = (event: MessageEvent<unknown>) => {
    const frame = this.options.frame()
    const data = event.data
    const userID = this.options.userID()
    if (!this.mounted || !frame || !this.sessionMatches(userID) || event.origin !== this.host.location.origin || event.origin === 'null'
      || event.source !== frame.contentWindow || event.ports.length !== 0 || !record(data)
      || !keysOnly(data, ['type', 'version', 'nonce']) || data.type !== 'sub2api:chat:ready'
      || data.version !== CHAT_BRIDGE_VERSION || typeof data.nonce !== 'string' || !NONCE.test(data.nonce)) return
    const src = new URL(frame.src, this.host.location.href)
    if (src.origin !== this.host.location.origin || src.pathname !== '/chat-app/' || src.search || src.hash) return
    if (this.handshakes.has(data.nonce) || this.handshakes.size >= 10000) return
    this.handshakes.add(data.nonce)
    this.disconnect()
    const source = frame.contentWindow
    if (!source) return
    const channel = this.options.channel?.() ?? new MessageChannel()
    this.port = channel.port1
    this.nonce = data.nonce
    this.userID = userID
    this.source = source
    try {
      this.document = frame.contentDocument ?? undefined
      source.addEventListener('pagehide', this.disconnect, { once: true })
      channel.port1.onmessage = (message: MessageEvent<unknown>) => { void this.receive(message.data) }
      channel.port1.onmessageerror = this.disconnect
      channel.port1.addEventListener('close', this.disconnect)
      channel.port1.start()
      source.postMessage({ type: 'sub2api:chat:connect', version: CHAT_BRIDGE_VERSION, nonce: this.nonce }, event.origin, [channel.port2])
    } catch {
      channel.port2.close()
      this.disconnect()
    }
  }

  disconnect = () => {
    this.source?.removeEventListener('pagehide', this.disconnect)
    this.source = undefined
    this.document = undefined
    for (const controller of this.active.values()) controller.abort()
    this.active.clear()
    if (this.port) {
      this.port.onmessage = null
      this.port.onmessageerror = null
      this.port.removeEventListener('close', this.disconnect)
      this.port.close()
    }
    // Only the fork's disposable, per-user import cache; never cloud history or auth storage.
    if (this.userID) {
      try { this.host.indexedDB?.deleteDatabase(`sub2api-chat-import:${this.userID}`) } catch { /* Private browsing can disable IndexedDB. */ }
    }
    this.port = undefined
    this.nonce = ''
    this.userID = null
  }

  destroy() {
    this.mounted = false
    this.host.removeEventListener('message', this.onReady)
    this.host.removeEventListener('pagehide', this.disconnect)
    this.host.removeEventListener('storage', this.onStorage)
    this.disconnect()
    this.seen.clear()
    this.handshakes.clear()
  }

  private async bootstrap(signal: AbortSignal): Promise<ChatBootstrap> {
    if (!this.userID) throw new ChatTransportError('登录状态已改变', 'AUTH_SESSION_CHANGED', 401)
    const transformRequest = chatRequestTransforms(this.userID)
    const [presets, preferences] = await Promise.all([
      apiClient.get<unknown>('/chat/presets', { signal, transformRequest }),
      apiClient.get<unknown>('/chat/preferences', { signal, transformRequest }),
    ])
    if (!this.userID || !Array.isArray(presets.data) || !record(preferences.data)
      || typeof preferences.data.history_revision !== 'number' || !Number.isSafeInteger(preferences.data.history_revision)
      || preferences.data.history_revision < 0) throw new ChatTransportError('聊天初始化响应无效', 'CHAT_BRIDGE_PROTOCOL')
    return {
      user: { id: this.userID }, presets: presets.data.map(presetView), preferences: chatPreferences(preferences.data.preferences),
      history_revision: preferences.data.history_revision, locale: this.options.locale().startsWith('zh') ? 'zh-CN' : 'en-US',
    }
  }

  private async receive(value: unknown) {
    if (!isChatBridgeRequest(value, this.nonce) || !this.port || !this.userID) return
    const request = value
    const port = this.port
    const nonce = this.nonce
    if (!this.sessionMatches(this.userID)) { this.disconnect(); this.options.onSessionInvalid?.(); return }
    // Retain completed IDs too: duplicates must never replay a generation or mutation.
    if (this.seen.has(request.request_id)) return
    if (this.seen.size >= 10000) { this.disconnect(); return }
    this.seen.add(request.request_id)
    const controller = new AbortController()
    this.active.set(request.request_id, controller)
    const timeout = request.command === 'stream' ? undefined : setTimeout(() => controller.abort(),
      request.command === 'upload' || request.command === 'download' ? 120000 : 30000)
    const reply = (message: Omit<BridgeResponse, 'nonce' | 'request_id'>) => {
      if (this.port !== port || this.nonce !== nonce || !this.sessionMatches(this.userID)) return
      try { port.postMessage({ ...message, request_id: request.request_id, nonce } satisfies BridgeResponse) }
      catch { this.disconnect() }
    }
    try {
      const data = await this.execute(request, controller.signal, (chunk) => reply({ type: 'chunk', data: chunk }))
      controller.signal.throwIfAborted()
      reply({ type: request.command === 'stream' ? 'end' : 'result', ...(request.command === 'stream' ? {} : { data }) })
    } catch (error) {
      // Never serialize Axios config, request headers, response objects or raw errors.
      let code = controller.signal.aborted ? 'CHAT_REQUEST_ABORTED' : 'CHAT_REQUEST_FAILED'
      let message = controller.signal.aborted ? '请求已取消或超时' : '聊天请求失败，请重试'
      let status: number | undefined
      if (record(error) || error instanceof Error) {
        if ('code' in error && typeof error.code === 'string') code = error.code
        if ('message' in error && typeof error.message === 'string') message = error.message
        if ('status' in error && typeof error.status === 'number') status = error.status
      }
      reply({ type: 'error', error: { code, message: code === 'CHAT_REVISION_CONFLICT' ? '此会话已在其他设备更新，请重新载入' : message, ...(status === undefined ? {} : { status }) } })
    } finally {
      clearTimeout(timeout)
      if (this.active.get(request.request_id) === controller) this.active.delete(request.request_id)
    }
  }

  private async execute(request: BridgeRequest, signal: AbortSignal, chunk: (value: Uint8Array) => void): Promise<unknown> {
    switch (request.command) {
      case 'bootstrap': return this.bootstrap(signal)
      case 'cancel': this.active.get(request.target_request_id)?.abort(); return
      case 'navigate': await this.options.navigate(request.path); return
      case 'logout':
        // Close immediately, before a potentially slow server-side logout completes.
        this.disconnect()
        await this.options.logout()
        return
      case 'json': {
        const body = request.method === 'PUT' && request.path === '/preferences' ? chatPreferences(request.body) : request.body
        const response = await apiClient.request<unknown>({ method: request.method, url: `/chat${request.path}`, data: body, signal, transformRequest: chatRequestTransforms(this.userID!) })
        if (request.method === 'GET' && request.path === '/presets') {
          if (!Array.isArray(response.data)) throw new ChatTransportError('聊天模型响应无效', 'CHAT_BRIDGE_PROTOCOL')
          return response.data.map(presetView)
        }
        if (request.path === '/preferences' && record(response.data)) return {
          preferences: chatPreferences(response.data.preferences), history_revision: response.data.history_revision,
        }
        return response.data
      }
      case 'upload': {
        const body = new FormData()
        body.append('file', request.body, request.filename)
        body.append('filename', request.filename)
        if (request.purpose) body.append('purpose', request.purpose)
        const response = await requireChatResponse(await authenticatedChatFetch(`/chat${request.path}`, { method: 'POST', body, signal }))
        const result: unknown = await response.json()
        if (record(result) && 'code' in result) {
          if (result.code !== 0) throw new ChatTransportError(typeof result.message === 'string' ? result.message : '上传失败', typeof result.code === 'string' ? result.code : 'CHAT_UPLOAD_FAILED')
          return result.data
        }
        return result
      }
      case 'download': {
        const response = await requireChatResponse(await authenticatedChatFetch(`/chat${request.path}`, { method: 'GET', signal }))
        return response.blob()
      }
      case 'stream': await streamAuthenticatedChat(`/chat${request.path}`, request.body, signal, chunk); return
    }
  }
}
