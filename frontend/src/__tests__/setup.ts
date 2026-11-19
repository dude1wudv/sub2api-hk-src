/**
 * Vitest 测试环境设置
 * 提供全局 mock 和测试工具
 */
import { config } from '@vue/test-utils'
import { vi } from 'vitest'

function createMemoryStorage(): Storage {
  const values = new Map<string, string>()

  return {
    get length() {
      return values.size
    },
    clear() {
      values.clear()
    },
    getItem(key: string) {
      return values.has(key) ? values.get(key)! : null
    },
    key(index: number) {
      return Array.from(values.keys())[index] ?? null
    },
    removeItem(key: string) {
      values.delete(key)
    },
    setItem(key: string, value: string) {
      values.set(key, String(value))
    }
  }
}

if (typeof globalThis.localStorage === 'undefined' || typeof globalThis.localStorage.getItem !== 'function') {
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: createMemoryStorage()
  })
}

if (typeof window !== 'undefined' && typeof window.localStorage.getItem !== 'function') {
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: globalThis.localStorage
  })
}

// Mock requestIdleCallback (Safari < 15 不支持)
if (typeof globalThis.requestIdleCallback === 'undefined') {
  globalThis.requestIdleCallback = ((callback: IdleRequestCallback) => {
    return window.setTimeout(() => callback({ didTimeout: false, timeRemaining: () => 50 }), 1)
  }) as unknown as typeof requestIdleCallback
}

if (typeof globalThis.cancelIdleCallback === 'undefined') {
  globalThis.cancelIdleCallback = ((id: number) => {
    window.clearTimeout(id)
  }) as unknown as typeof cancelIdleCallback
}

// Mock matchMedia (jsdom 未实现;DataTable 等组件依赖它做桌面/移动分支)
if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  window.matchMedia = ((query: string) => ({
    matches: /^\(min-width:/.test(query), // 默认桌面：最小宽度匹配，手机与触屏查询不匹配
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia
}

// Mock IntersectionObserver
class MockIntersectionObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
}

globalThis.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver

// Mock ResizeObserver
class MockResizeObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
}

globalThis.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver

// Mock or patch ElementInternals & HTMLDialogElement for @material/web under jsdom
if (typeof HTMLElement !== 'undefined') {
  const originalAttachInternals = HTMLElement.prototype.attachInternals
  HTMLElement.prototype.attachInternals = function attachInternals() {
    const internals =
      typeof originalAttachInternals === 'function'
        ? (originalAttachInternals.call(this) as unknown as Record<string, unknown>)
        : {
            shadowRoot: this.shadowRoot,
            form: null,
            states: new Set(),
            role: null,
            labels: []
          }
    if (typeof internals.setFormValue !== 'function') {
      internals.setFormValue = vi.fn()
    }
    if (typeof internals.setValidity !== 'function') {
      internals.setValidity = vi.fn()
    }
    if (typeof internals.checkValidity !== 'function') {
      internals.checkValidity = () => true
    }
    if (typeof internals.reportValidity !== 'function') {
      internals.reportValidity = () => true
    }
    if (!internals.validity) {
      internals.validity = { valid: true }
    }
    return internals as unknown as ElementInternals
  }
}

if (typeof HTMLDialogElement !== 'undefined') {
  if (typeof HTMLDialogElement.prototype.showModal !== 'function') {
    HTMLDialogElement.prototype.showModal = function showModal() {
      this.setAttribute('open', '')
    }
  }
  if (typeof HTMLDialogElement.prototype.show !== 'function') {
    HTMLDialogElement.prototype.show = function show() {
      this.setAttribute('open', '')
    }
  }
  if (typeof HTMLDialogElement.prototype.close !== 'function') {
    HTMLDialogElement.prototype.close = function close(returnValue?: string) {
      if (returnValue !== undefined) this.returnValue = returnValue
      this.removeAttribute('open')
      this.dispatchEvent(new Event('close'))
    }
  }
}

// Vue Test Utils 全局配置
config.global.stubs = {
  // 可以在这里添加全局 stub
}

// 设置全局测试超时
vi.setConfig({ testTimeout: 10000 })
