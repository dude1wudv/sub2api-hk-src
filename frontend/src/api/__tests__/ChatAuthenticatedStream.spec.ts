import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { authenticatedChatFetch, requireChatResponse, streamAuthenticatedChat, chatRequestTransforms } from '../authenticatedStream'
import { AxiosHeaders } from 'axios'

const { auth, refresh, fetchMock } = vi.hoisted(() => ({
  auth: { user: { id: 7 } as { id: number } | null, token: null as string | null, setToken: vi.fn(), logout: vi.fn() },
  refresh: vi.fn(), fetchMock: vi.fn(),
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('../tokenRefresh', () => ({ refreshAuthTokens: refresh }))
vi.mock('../client', () => ({ buildApiUrl: (path: string) => `/api/v1${path}` }))
vi.mock('@/i18n', () => ({ getLocale: () => 'zh-CN' }))
const init = () => ({ method: 'POST', signal: new AbortController().signal, body: '{}' })

const tokenFor = (id: number, label = 'test') => `header.${btoa(JSON.stringify({ user_id: id }))}.${label}`
describe('Chat authenticated transport', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fetchMock.mockReset()
    refresh.mockReset()
    localStorage.clear()
    localStorage.setItem('auth_token', tokenFor(7, 'old-access'))
    localStorage.setItem('refresh_token', 'refresh')
    localStorage.setItem('auth_user', JSON.stringify({ id: 7 }))
    auth.user = { id: 7 }
    auth.token = tokenFor(7, 'old-access')
    auth.setToken.mockImplementation(async (token: string) => { auth.token = token; localStorage.setItem('auth_token', token) })
    auth.logout.mockImplementation(async () => { auth.token = null; auth.user = null; localStorage.clear() })
    refresh.mockResolvedValue({ access_token: tokenFor(7, 'fresh-access') })
    vi.stubGlobal('fetch', fetchMock)
  })
  afterEach(() => vi.unstubAllGlobals())

  it('refreshes an explicit pre-body 401 once using the shared refresh coordinator and store', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response('ok'))
    await expect(authenticatedChatFetch('/chat/attachments', init())).resolves.toBeInstanceOf(Response)
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(refresh).toHaveBeenCalledWith({ failedAccessToken: tokenFor(7, 'old-access') })
    expect(auth.setToken).toHaveBeenCalledTimes(1)
    expect(auth.setToken).toHaveBeenCalledWith(tokenFor(7, 'fresh-access'))
    expect(fetchMock).toHaveBeenCalledTimes(2)
    const [url, options] = fetchMock.mock.calls[1]
    expect(url).toBe('/api/v1/chat/attachments')
    expect(new Headers(options.headers).get('Authorization')).toBe(`Bearer ${tokenFor(7, 'fresh-access')}`)
    expect(options).toMatchObject({ redirect: 'error', credentials: 'same-origin' })
  })

  it('never refreshes twice and logs out after the retry is rejected', async () => {
    fetchMock.mockImplementation(async () => new Response(null, { status: 401 }))
    await expect(authenticatedChatFetch('/chat/attachments', init())).rejects.toMatchObject({ code: 'AUTH_EXPIRED' })
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(auth.logout).toHaveBeenCalledTimes(1)
  })

  it('does not attempt refresh without a refresh token', async () => {
    localStorage.removeItem('refresh_token')
    fetchMock.mockResolvedValue(new Response(null, { status: 401 }))
    await expect(authenticatedChatFetch('/chat/attachments', init())).rejects.toMatchObject({ code: 'AUTH_EXPIRED' })
    expect(refresh).not.toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it.each([new TypeError('Network outcome unknown'), new Error('connection closed')])('does not replay an uncertain network result: %s', async (error) => {
    fetchMock.mockRejectedValue(error)
    await expect(authenticatedChatFetch('/chat/sessions/a/generate', init())).rejects.toBe(error)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(refresh).not.toHaveBeenCalled()
  })

  it('does not replay a non-auth HTTP failure', async () => {
    const response = new Response('busy', { status: 503 })
    fetchMock.mockResolvedValue(response)
    await expect(authenticatedChatFetch('/chat/attachments', init())).resolves.toBe(response)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(refresh).not.toHaveBeenCalled()
  })

  it('streams the original split UTF-8/CRLF bytes, not decoded or reconstructed SSE', async () => {
    const chunks = [new Uint8Array([100, 97, 116, 97, 58, 32, 228]), new Uint8Array([184, 173, 13, 10, 13, 10])]
    fetchMock.mockResolvedValue(new Response(new ReadableStream({ start(controller) {
      chunks.forEach((chunk) => controller.enqueue(chunk)); controller.close()
    } }), { headers: { 'Content-Type': 'text/event-stream' } }))
    const output: Uint8Array[] = []
    await streamAuthenticatedChat('/chat/sessions/a/generate', {}, new AbortController().signal, (chunk) => output.push(chunk))
    expect(output).toEqual(chunks)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('never refreshes or replays after SSE has started even if a reader reports 401', async () => {
    fetchMock.mockResolvedValue(new Response(new ReadableStream({ start(controller) {
      controller.enqueue(new Uint8Array([1]));
    }, pull(controller) { controller.error(new Error('401 after first SSE chunk')) } }), { headers: { 'Content-Type': 'text/event-stream' } }))
    const received = vi.fn()
    await expect(streamAuthenticatedChat('/chat/sessions/a/generate', {}, new AbortController().signal, received)).rejects.toThrow('401')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(refresh).not.toHaveBeenCalled()
  })

  it('does not refresh a 401 already marked as an SSE response', async () => {
    fetchMock.mockResolvedValue(new Response('data: error\n\n', { status: 401, headers: { 'Content-Type': 'text/event-stream' } }))
    await expect(streamAuthenticatedChat('/chat/sessions/a/generate', {}, new AbortController().signal, vi.fn())).rejects.toMatchObject({ status: 401 })
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(refresh).not.toHaveBeenCalled()
  })

  it('cancels a blocked reader and never replays it', async () => {
    const cancel = vi.fn()
    fetchMock.mockResolvedValue(new Response(new ReadableStream({ cancel }), { headers: { 'Content-Type': 'text/event-stream' } }))
    const controller = new AbortController()
    const pending = streamAuthenticatedChat('/chat/sessions/a/generate', {}, controller.signal, vi.fn())
    const rejected = expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    await flushPromises()
    controller.abort()
    await rejected
    expect(cancel).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('does not replay or log out a replacement account when refresh completes late', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 401 }))
    refresh.mockImplementationOnce(async () => {
      auth.user = { id: 8 }; auth.token = tokenFor(8)
      localStorage.setItem('auth_token', tokenFor(8))
      localStorage.setItem('auth_user', JSON.stringify({ id: 8 }))
      return { access_token: tokenFor(7, 'stale-refresh') }
    })
    await expect(authenticatedChatFetch('/chat/attachments', init())).rejects.toMatchObject({ code: 'AUTH_SESSION_CHANGED' })
    expect(auth.setToken).not.toHaveBeenCalled()
    expect(auth.logout).not.toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('does not log out on a transient refresh outage or replay a cancelled request', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 401 }))
    refresh.mockRejectedValueOnce({ response: { status: 503 } })
    await expect(authenticatedChatFetch('/chat/attachments', init())).rejects.toMatchObject({ code: 'TOKEN_REFRESH_FAILED', status: 503 })
    expect(auth.logout).not.toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const controller = new AbortController()
    controller.abort()
    await expect(authenticatedChatFetch('/chat/attachments', { method: 'POST', signal: controller.signal })).rejects.toMatchObject({ name: 'AbortError' })
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('reads the newest access token per request and blocks transport path traversal', async () => {
    localStorage.setItem('auth_token', tokenFor(7, 'peer-refreshed-access'))
    fetchMock.mockResolvedValueOnce(new Response('ok'))
    await authenticatedChatFetch('/chat/attachments', init())
    expect(new Headers(fetchMock.mock.calls[0][1].headers).get('Authorization')).toBe(`Bearer ${tokenFor(7, 'peer-refreshed-access')}`)
    await expect(authenticatedChatFetch('/chat/../auth/me', init())).rejects.toMatchObject({ code: 'CHAT_BRIDGE_FORBIDDEN' })
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('rejects a replacement token before auth_user catches up', async () => {
    localStorage.setItem('auth_token', tokenFor(8))
    fetchMock.mockResolvedValueOnce(new Response('ok'))
    await expect(authenticatedChatFetch('/chat/attachments', init())).rejects.toMatchObject({ code: 'AUTH_SESSION_CHANGED' })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('rechecks the token after asynchronous Axios auth interceptors', () => {
    const guard = chatRequestTransforms(7).at(-1)!
    const headers = new AxiosHeaders({ Authorization: `Bearer ${tokenFor(8)}` })
    expect(() => guard.call({ headers }, '{}', headers)).toThrow('登录状态已改变')
    headers.set('Authorization', `Bearer ${tokenFor(7, 'rotated')}`)
    expect(guard.call({ headers }, '{}', headers)).toBe('{}')
  })

  it('preserves the global compliance dialog event without replaying a rejected request', async () => {
    const event = vi.fn()
    window.addEventListener('admin-compliance-required', event)
    try {
      const response = new Response(JSON.stringify({ code: 'ADMIN_COMPLIANCE_ACK_REQUIRED', message: 'ack required', metadata: { version: 2 } }), { status: 423, headers: { 'Content-Type': 'application/json' } })
      await expect(requireChatResponse(response)).rejects.toMatchObject({ status: 423, code: 'ADMIN_COMPLIANCE_ACK_REQUIRED' })
      expect(event).toHaveBeenCalledTimes(1)
      expect(event.mock.calls[0][0].detail).toEqual({ version: 2 })
      expect(fetchMock).not.toHaveBeenCalled()
    } finally {
      window.removeEventListener('admin-compliance-required', event)
    }
  })
})
