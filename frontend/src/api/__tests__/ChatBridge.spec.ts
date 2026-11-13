import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { ChatHostBridge, isChatBridgeRequest } from '../chatBridge'
import type { BridgeResponse } from '../../../../chat-demo/src/services/sub2apiBridgeContract'
import type * as AuthenticatedStream from '../authenticatedStream'

const { get, request, transport, stream, requireResponse, deleteDatabase } = vi.hoisted(() => ({
  get: vi.fn(), request: vi.fn(), transport: vi.fn(), stream: vi.fn(), requireResponse: vi.fn(), deleteDatabase: vi.fn(),
}))
vi.mock('../client', () => ({ apiClient: { get, request }, buildApiUrl: (path: string) => `/api/v1${path}` }))
vi.mock('@/stores/auth', () => ({ useAuthStore: vi.fn() }))
vi.mock('@/i18n', () => ({ getLocale: () => 'zh-CN' }))
vi.mock('../authenticatedStream', async (importOriginal) => ({
  ...await importOriginal<typeof AuthenticatedStream>(),
  authenticatedChatFetch: transport, streamAuthenticatedChat: stream, requireChatResponse: requireResponse,
}))

const nonce = '1d7dd751-4ad7-4b14-8640-caf6d8478ce0'
const nextNonce = '1d7dd751-4ad7-4b14-8640-caf6d8478ce1'
const envelope = (payload: Record<string, unknown>, id = 'request-1') => ({ type: 'request', request_id: id, nonce, ...payload })
const preset = {
  id: 'preset-1', group_id: 11, model: 'public-model', display_name: 'My model', sort_order: 0, is_default: true, available: true, unavailable_reason: '',
  group: { id: 11, name: 'Group X', platform: 'openai', subscription_type: 'standard', rate_multiplier: 1, user_rate_multiplier: 1, upstream_key: 'must-not-cross' },
  capabilities: { transport: 'chat_completions', image_input: false, voice: true, reasons: {}, upstream_key: 'must-not-cross' },
  api_key: 'must-not-cross',
}

class TestPort extends EventTarget {
  onmessage: ((event: MessageEvent<unknown>) => void) | null = null
  onmessageerror: (() => void) | null = null
  sent: BridgeResponse[] = []
  start = vi.fn()
  close = vi.fn()
  postMessage = vi.fn((message: BridgeResponse) => { this.sent.push(message) })
  receive(data: unknown) { this.onmessage?.(new MessageEvent('message', { data })) }
}

const hosts: ChatHostBridge[] = []
function createHost() {
  const frame = document.createElement('iframe')
  frame.src = '/chat-app/'
  document.body.appendChild(frame)
  const port = new TestPort()
  const child = new TestPort()
  // DOM test seams implement only the MessageChannel operations exercised here.
  const channel = vi.fn(() => ({ port1: port, port2: child }) as unknown as MessageChannel)
  const navigate = vi.fn()
  const logout = vi.fn(async () => {})
  const invalid = vi.fn()
  const post = vi.spyOn(frame.contentWindow!, 'postMessage').mockImplementation(() => {})
  const bridge = new ChatHostBridge({ frame: () => frame, userID: () => 7, locale: () => 'zh-CN', navigate, logout, onSessionInvalid: invalid, channel })
  hosts.push(bridge)
  bridge.mount()
  const ready = (overrides: MessageEventInit = {}) => window.dispatchEvent(new MessageEvent('message', {
    source: frame.contentWindow, origin: window.location.origin, data: { type: 'sub2api:chat:ready', version: 1, nonce }, ...overrides,
  }))
  return { bridge, frame, port, channel, post, ready, navigate, logout, invalid }
}

describe('Chat bridge finite API validation', () => {
  it.each([
    '//evil.test/sessions', 'https://evil.test/sessions', '/sessions/../presets', '/sessions/%2e%2e/presets',
    '/sessions/a%2fb', '/sessions/a%5cb', '/sessions/a#token', '/sessions/a\\b', '/sessions/a%252fb',
    '/sessions?token=secret', '/sessions?limit=1&limit=2', '/sessions?limit=101', '/sessions?limit=0',
    '/sessions?cursor=x&api_key=secret', '/api/v1/keys', '/catalog', '/presets/order', '/sessions?limit=1?token=x',
  ])('rejects path/credential attacks: %s', (path) => {
    expect(isChatBridgeRequest(envelope({ command: 'json', method: 'GET', path }), nonce)).toBe(false)
  })

  it('rejects forged nonces, unknown commands/methods, headers and arbitrary navigation', () => {
    expect(isChatBridgeRequest(envelope({ command: 'json', method: 'GET', path: '/sessions', nonce: nextNonce }), nonce)).toBe(false)
    for (const method of ['TRACE', 'toString', '__proto__', 'get']) expect(isChatBridgeRequest(envelope({ command: 'json', method, path: '/sessions' }), nonce)).toBe(false)
    expect(isChatBridgeRequest(envelope({ command: 'json', method: 'GET', path: '/sessions', headers: { Authorization: 'injected' } }), nonce)).toBe(false)
    expect(isChatBridgeRequest(envelope({ command: 'navigate', path: 'https://evil.test/' }), nonce)).toBe(false)
    expect(isChatBridgeRequest(envelope({ command: 'navigate', path: '/dashboard?redirect=evil' }), nonce)).toBe(false)
    expect(isChatBridgeRequest(envelope({ command: 'bootstrap', api_key: 'injected' }), nonce)).toBe(false)
  })

  it('allows only documented paths, methods, pagination and fixed navigation', () => {
    for (const [method, path] of [['GET', '/sessions?limit=100&cursor=abc%3D'], ['PUT', '/sessions/a'], ['PATCH', '/sessions/a'], ['DELETE', '/sessions'], ['POST', '/imports/a/commit'], ['POST', '/sessions/a/stop']]) {
      expect(isChatBridgeRequest(envelope({ command: 'json', method, path }), nonce)).toBe(true)
    }
    for (const path of ['/dashboard', '/chat-models']) expect(isChatBridgeRequest(envelope({ command: 'navigate', path }), nonce)).toBe(true)
    expect(isChatBridgeRequest(envelope({ command: 'stream', path: '/sessions/a/generate', body: {} }), nonce)).toBe(true)
    expect(isChatBridgeRequest(envelope({ command: 'download', path: '/attachments/a/content' }), nonce)).toBe(true)
  })
})

describe('Chat host MessageChannel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    get.mockReset(); request.mockReset(); transport.mockReset(); stream.mockReset(); requireResponse.mockReset()
    localStorage.clear()
    localStorage.setItem('auth_token', `header.${btoa(JSON.stringify({ user_id: 7 }))}.host-only-token`)
    localStorage.setItem('auth_user', JSON.stringify({ id: 7 }))
    vi.stubGlobal('indexedDB', { deleteDatabase })
    get.mockImplementation(async (path: string) => ({ data: path === '/chat/presets' ? [preset] : { preferences: { themePreference: 'dark', thinking: true, apiKey: 'must-not-cross', location: { region: 'CN', apiKey: 'must-not-cross' } }, history_revision: 4 } }))
    request.mockResolvedValue({ data: { items: [], next_cursor: '' } })
    requireResponse.mockImplementation(async (response: Response) => response)
  })
  afterEach(() => {
    hosts.splice(0).forEach((host) => host.destroy())
    document.body.replaceChildren()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('ignores forged origin, source, version, nonce format and extra ports before an exact handshake', () => {
    const host = createHost()
    host.ready({ origin: 'https://evil.test' })
    host.ready({ source: window })
    host.ready({ data: { type: 'sub2api:chat:ready', version: 2, nonce } })
    host.ready({ data: { type: 'sub2api:chat:ready', version: 1, nonce: 'attacker' } })
    host.ready({ ports: [{} as MessagePort] })
    expect(host.channel).not.toHaveBeenCalled()
    host.ready()
    expect(host.channel).toHaveBeenCalledTimes(1)
    expect(host.post).toHaveBeenCalledWith({ type: 'sub2api:chat:connect', version: 1, nonce }, window.location.origin, expect.any(Array))
    host.bridge.frameLoaded() // ready can fire before the iframe load event
    expect(host.port.close).not.toHaveBeenCalled()
    host.ready()
    expect(host.channel).toHaveBeenCalledTimes(1)
  })

  it('does not respond to wrong-nonce port messages or dispatch forbidden paths', async () => {
    const host = createHost(); host.ready()
    host.port.receive(envelope({ command: 'json', method: 'GET', path: '/sessions', nonce: nextNonce }))
    host.port.receive(envelope({ command: 'json', method: 'POST', path: '/keys' }))
    await flushPromises()
    expect(request).not.toHaveBeenCalled()
    expect(host.port.sent).toEqual([])
  })

  it('bootstraps the exact fork shape and never includes credentials or the full user', async () => {
    const host = createHost(); host.ready()
    host.port.receive(envelope({ command: 'bootstrap' }))
    await flushPromises()
    expect(host.port.sent[0]).toMatchObject({ type: 'result', request_id: 'request-1', nonce, data: {
      user: { id: 7 }, presets: [{ id: 'preset-1', group: { id: 11 }, capabilities: { voice: false } }],
      preferences: { themePreference: 'dark', thinking: true, location: { region: 'CN' } }, history_revision: 4, locale: 'zh-CN',
    } })
    expect(JSON.stringify(host.port.sent)).not.toMatch(/must-not-cross|host-only-token|apiKey|upstream_key|api_key/)
    expect(get.mock.calls.every(([, config]) => config.signal instanceof AbortSignal)).toBe(true)
  })

  it('deduplicates active and completed request IDs and uses apiClient for JSON', async () => {
    const host = createHost(); host.ready()
    const message = envelope({ command: 'json', method: 'GET', path: '/sessions?limit=50' })
    host.port.receive(message); host.port.receive(message)
    await flushPromises()
    host.port.receive(message)
    await flushPromises()
    expect(request).toHaveBeenCalledTimes(1)
    expect(request).toHaveBeenCalledWith(expect.objectContaining({ method: 'GET', url: '/chat/sessions?limit=50', signal: expect.any(AbortSignal) }))
    expect(host.port.sent).toHaveLength(1)
  })

  it('uploads FormData with file/filename/purpose, and downloads a Blob', async () => {
    const host = createHost(); host.ready()
    transport.mockResolvedValueOnce(new Response(JSON.stringify({ code: 0, data: { id: 'local-attachment' } }), { headers: { 'Content-Type': 'application/json' } }))
    const file = new Blob(['hello'], { type: 'text/plain' })
    host.port.receive(envelope({ command: 'upload', path: '/attachments', body: file, filename: 'notes.txt', purpose: 'user_data' }))
    await flushPromises()
    const [path, options] = transport.mock.calls[0]
    expect(path).toBe('/chat/attachments')
    expect(options.body).toBeInstanceOf(FormData)
    expect(options.body.get('file').name).toBe('notes.txt')
    expect(options.body.get('filename')).toBe('notes.txt')
    expect(options.body.get('purpose')).toBe('user_data')
    expect(host.port.sent[0]).toMatchObject({ type: 'result', data: { id: 'local-attachment' } })
    const blob = new Blob(['private bytes'])
    transport.mockResolvedValueOnce({ blob: async () => blob })
    host.port.receive(envelope({ command: 'download', path: '/attachments/local-attachment/content' }, 'download'))
    await flushPromises()
    expect(host.port.sent[1].data).toBe(blob)
  })

  it('forwards raw SSE Uint8Arrays and an end only after the stream resolves', async () => {
    const host = createHost(); host.ready()
    const bytes = new Uint8Array([228, 184, 173, 13, 10])
    const gate = Promise.withResolvers<void>()
    stream.mockImplementationOnce(async (_path, _body, _signal, chunk) => { chunk(bytes); await gate.promise })
    host.port.receive(envelope({ command: 'stream', path: '/sessions/a/generate', body: { request_id: 'generation-id' } }))
    await flushPromises()
    expect(host.port.sent).toEqual([{ type: 'chunk', request_id: 'request-1', nonce, data: bytes }])
    gate.resolve()
    await flushPromises()
    expect(host.port.sent[1]).toEqual({ type: 'end', request_id: 'request-1', nonce })
  })

  it.each(['reload', 'unmount', 'logout', 'cancel', 'cross-tab-logout'])('aborts active work on %s and clears only this user temporary cache', async (reason) => {
    const host = createHost(); host.ready()
    const gate = Promise.withResolvers<void>()
    let signal!: AbortSignal
    stream.mockImplementationOnce(async (_path, _body, incomingSignal) => { signal = incomingSignal; await gate.promise })
    host.port.receive(envelope({ command: 'stream', path: '/sessions/a/generate', body: {} }))
    await flushPromises()
    if (reason === 'reload') host.frame.contentWindow!.dispatchEvent(new Event('pagehide'))
    if (reason === 'unmount') host.bridge.destroy()
    if (reason === 'logout') host.port.receive(envelope({ command: 'logout' }, 'logout'))
    if (reason === 'cancel') host.port.receive(envelope({ command: 'cancel', target_request_id: 'request-1' }, 'cancel'))
    if (reason === 'cross-tab-logout') { localStorage.removeItem('auth_token'); window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token', newValue: null })) }
    expect(signal.aborted).toBe(true)
    if (reason !== 'cancel') {
      expect(host.port.close).toHaveBeenCalled()
      expect(deleteDatabase).toHaveBeenCalledWith('sub2api-chat-import:7')
    }
    if (reason === 'logout') expect(host.logout).toHaveBeenCalledTimes(1)
    gate.resolve()
    await flushPromises()
    expect(host.port.sent.some((response) => response.type === 'end')).toBe(false)
  })

  it('rejects retired handshake nonces and preserves in-flight work on same-user storage refresh', () => {
    const host = createHost(); host.ready()
    localStorage.setItem('auth_token', `header.${btoa(JSON.stringify({ user_id: 7 }))}.peer-refresh`)
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token' }))
    localStorage.setItem('auth_user', JSON.stringify({ id: 7, balance: 3 }))
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_user' }))
    expect(host.port.close).not.toHaveBeenCalled()
    host.bridge.disconnect(); host.ready()
    expect(host.channel).toHaveBeenCalledTimes(1)
    host.ready({ data: { type: 'sub2api:chat:ready', version: 1, nonce: nextNonce } })
    expect(host.channel).toHaveBeenCalledTimes(2)
  })

  it('rejects a stale host after a cross-tab account switch even when Pinia still holds the old user', async () => {
    const host = createHost(); host.ready()
    localStorage.setItem('auth_user', JSON.stringify({ id: 8 }))
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_user' }))
    host.ready({ data: { type: 'sub2api:chat:ready', version: 1, nonce: nextNonce } })
    await flushPromises()
    expect(host.channel).toHaveBeenCalledTimes(1)
    expect(host.invalid).toHaveBeenCalledTimes(1)
    expect(get).not.toHaveBeenCalled()
  })

  it('disconnects when a new account token precedes the auth_user write', async () => {
    const host = createHost(); host.ready()
    localStorage.setItem('auth_token', `header.${btoa(JSON.stringify({ user_id: 8 }))}.test`)
    host.port.receive(envelope({ command: 'json', method: 'DELETE', path: '/sessions' }))
    await flushPromises()
    expect(request).not.toHaveBeenCalled()
    expect(host.port.close).toHaveBeenCalled()
  })

  it('enforces ordinary/upload deadlines without imposing a 30s stream timeout', async () => {
    vi.useFakeTimers()
    const host = createHost(); host.ready()
    const pending = Promise.withResolvers<never>()
    request.mockReturnValue(pending.promise)
    transport.mockReturnValue(pending.promise)
    stream.mockReturnValue(pending.promise)
    host.port.receive(envelope({ command: 'json', method: 'GET', path: '/sessions' }, 'json'))
    host.port.receive(envelope({ command: 'upload', path: '/attachments', body: new Blob(['x']), filename: 'a.txt' }, 'upload'))
    host.port.receive(envelope({ command: 'stream', path: '/sessions/a/generate', body: {} }, 'stream'))
    const jsonSignal = request.mock.calls[0][0].signal
    const uploadSignal = transport.mock.calls[0][1].signal
    const streamSignal = stream.mock.calls[0][2]
    vi.advanceTimersByTime(30000)
    expect(jsonSignal.aborted).toBe(true)
    expect(uploadSignal.aborted).toBe(false)
    expect(streamSignal.aborted).toBe(false)
    vi.advanceTimersByTime(90000)
    expect(uploadSignal.aborted).toBe(true)
    expect(streamSignal.aborted).toBe(false)
    host.bridge.destroy()
    pending.reject(new Error('closed'))
    vi.useRealTimers()
  })

  it('only navigates to fixed targets and returns revision conflicts without error internals', async () => {
    const host = createHost(); host.ready()
    host.port.receive(envelope({ command: 'navigate', path: '/chat-models' }, 'navigate'))
    request.mockRejectedValueOnce({ status: 409, code: 'CHAT_REVISION_CONFLICT', message: 'conflict', config: { headers: { Authorization: 'secret' } } })
    host.port.receive(envelope({ command: 'json', method: 'PUT', path: '/sessions/a', body: {} }))
    await flushPromises()
    expect(host.navigate).toHaveBeenCalledWith('/chat-models')
    expect(host.port.sent).toContainEqual(expect.objectContaining({ type: 'error', error: { status: 409, code: 'CHAT_REVISION_CONFLICT', message: '此会话已在其他设备更新，请重新载入' } }))
    expect(JSON.stringify(host.port.sent)).not.toContain('Authorization')
  })
})
