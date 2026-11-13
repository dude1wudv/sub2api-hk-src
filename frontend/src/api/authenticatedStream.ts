import axios, { type AxiosRequestTransformer } from 'axios'
import { buildApiUrl } from './client'
import { refreshAuthTokens } from './tokenRefresh'
import { useAuthStore } from '@/stores/auth'
import { getLocale } from '@/i18n'

export class ChatTransportError extends Error {
  constructor(message: string, readonly code = 'CHAT_REQUEST_FAILED', readonly status?: number) {
    super(message)
    this.name = 'ChatTransportError'
  }
}

export const chatStoredUserID = (storage: Storage = localStorage): number | null => {
  try {
    const user: unknown = JSON.parse(storage.getItem('auth_user') || 'null')
    return user && typeof user === 'object' && 'id' in user && typeof user.id === 'number' && Number.isSafeInteger(user.id) && user.id > 0 ? user.id : null
  }
  catch { return null }
}

/** A client-side consistency check, not JWT verification. The API still verifies
 * the signature and permissions; this prevents cross-tab token/user write races. */
export const chatTokenUserID = (token: string | null): number | null => {
  try {
    const parts = token?.split('.')
    if (parts?.length !== 3 || !parts[1] || parts[1].length > 16384) return null
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const bytes = Uint8Array.from(atob(base64 + '='.repeat((4 - base64.length % 4) % 4)), char => char.charCodeAt(0))
    const payload: unknown = JSON.parse(new TextDecoder().decode(bytes))
    if (!payload || typeof payload !== 'object' || !('user_id' in payload)) return null
    const id = payload.user_id
    return typeof id === 'number' && Number.isSafeInteger(id) && id > 0 ? id : null
  } catch { return null }
}

/** Runs after Axios auth interceptors, including a refresh retry, immediately
 * before dispatch. A pre-request bridge check alone has an async TOCTOU gap. */
export const chatRequestTransforms = (userID: number): AxiosRequestTransformer[] => {
  const defaults = axios.defaults.transformRequest
  return [...(Array.isArray(defaults) ? defaults : defaults ? [defaults] : []), (data, headers) => {
    const authorization = headers.get('Authorization')
    const token = typeof authorization === 'string' && authorization.startsWith('Bearer ') ? authorization.slice(7) : null
    if (chatStoredUserID() !== userID || chatTokenUserID(token) !== userID || chatTokenUserID(localStorage.getItem('auth_token')) !== userID) {
      throw new ChatTransportError('登录状态已改变，请重新进入聊天', 'AUTH_SESSION_CHANGED', 401)
    }
    return data
  }]
}

/** Only an HTTP 401 received before a response body starts can be retried.
 * Network errors, SSE errors and reader failures never replay a paid request. */
export async function authenticatedChatFetch(
  path: string,
  init: RequestInit & { signal: AbortSignal },
): Promise<Response> {
  const method = init.method ?? 'GET'
  const allowed = method === 'GET' ? /^\/chat\/attachments\/[A-Za-z0-9_-]+\/content$/.test(path)
    : method === 'POST' && /^\/chat\/(attachments|imports\/[A-Za-z0-9_-]+\/attachments|sessions\/[A-Za-z0-9_-]+\/generate)$/.test(path)
  if (!allowed) throw new ChatTransportError('不允许的聊天路径', 'CHAT_BRIDGE_FORBIDDEN')
  const auth = useAuthStore()
  const userID = auth.user?.id
  const assertSession = () => {
    init.signal.throwIfAborted()
    if (!userID || !auth.token || (auth.user && auth.user.id !== userID) || chatStoredUserID() !== userID || chatTokenUserID(localStorage.getItem('auth_token')) !== userID) {
      throw new ChatTransportError('登录状态已改变，请重新进入聊天', 'AUTH_SESSION_CHANGED', 401)
    }
  }
  for (let attempt = 0; attempt < 2; attempt++) {
    assertSession()
    const token = localStorage.getItem('auth_token')
    if (!token || chatTokenUserID(token) !== userID) throw new ChatTransportError('请重新登录', 'AUTH_SESSION_CHANGED', 401)
    const headers = new Headers(init.headers)
    headers.set('Authorization', `Bearer ${token}`)
    headers.set('Accept-Language', getLocale())
    const response = await fetch(buildApiUrl(path), { ...init, headers, credentials: 'same-origin', redirect: 'error' })
    try { assertSession() } catch (error) {
      await response.body?.cancel().catch(() => {})
      throw error
    }
    if (response.status !== 401 || response.headers.get('content-type')?.includes('text/event-stream')) return response
    await response.body?.cancel()
    if (attempt === 1 || !localStorage.getItem('refresh_token')) {
      await auth.logout()
      throw new ChatTransportError('登录已过期，请重新登录', 'AUTH_EXPIRED', 401)
    }
    try {
      const tokens = await refreshAuthTokens({ failedAccessToken: token })
      assertSession()
      await auth.setToken(tokens.access_token)
      assertSession()
    } catch (error) {
      init.signal.throwIfAborted()
      // A replaced session must not be cleared by an old request.
      assertSession()
      let status: number | undefined
      if (error && typeof error === 'object') {
        if ('status' in error && typeof error.status === 'number') status = error.status
        if ('response' in error && error.response && typeof error.response === 'object'
          && 'status' in error.response && typeof error.response.status === 'number') status = error.response.status
      }
      if (status === 400 || status === 401 || status === 403) await auth.logout()
      throw new ChatTransportError('无法刷新登录状态，请稍后重试', 'TOKEN_REFRESH_FAILED', status)
    }
  }
  throw new ChatTransportError('登录已过期', 'AUTH_EXPIRED', 401)
}

export async function requireChatResponse(response: Response): Promise<Response> {
  if (response.ok) return response
  let data: unknown
  if (response.headers.get('content-type')?.includes('application/json')) {
    try { data = await response.json() } catch { /* An invalid error body is not a reason to retry. */ }
  } else {
    await response.body?.cancel().catch(() => {})
  }
  if (response.status === 423 && data && typeof data === 'object' && 'code' in data && data.code === 'ADMIN_COMPLIANCE_ACK_REQUIRED') {
    window.dispatchEvent(new CustomEvent('admin-compliance-required', { detail: 'metadata' in data ? data.metadata : {} }))
  }
  throw new ChatTransportError(
    data && typeof data === 'object' && 'message' in data && typeof data.message === 'string' ? data.message : `聊天请求失败 (${response.status})`,
    data && typeof data === 'object' && 'code' in data && typeof data.code === 'string' ? data.code : 'CHAT_REQUEST_FAILED',
    response.status,
  )
}

export async function streamAuthenticatedChat(
  path: string,
  body: unknown,
  signal: AbortSignal,
  onChunk: (chunk: Uint8Array) => void,
): Promise<void> {
  const response = await requireChatResponse(await authenticatedChatFetch(path, {
    method: 'POST', headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
    body: JSON.stringify(body), signal,
  }))
  if (!response.headers.get('content-type')?.includes('text/event-stream') || !response.body) {
    await response.body?.cancel().catch(() => {})
    throw new ChatTransportError('聊天流响应无效', 'CHAT_STREAM_INVALID')
  }
  const reader = response.body.getReader()
  const cancel = () => { void reader.cancel().catch(() => {}) }
  signal.addEventListener('abort', cancel, { once: true })
  try {
    signal.throwIfAborted()
    for (;;) {
      const { value, done } = await reader.read()
      signal.throwIfAborted()
      if (done) return
      onChunk(value)
    }
  } finally {
    signal.removeEventListener('abort', cancel)
    await reader.cancel().catch(() => {})
    reader.releaseLock()
  }
}
