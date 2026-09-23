export interface TurnstileRenderOptions {
  sitekey: string
  callback: (token: string) => void
  'expired-callback'?: () => void
  'error-callback'?: (code: string) => boolean | void
  'timeout-callback'?: () => void
  theme?: 'light' | 'dark' | 'auto'
  size?: 'normal' | 'compact' | 'flexible'
}

declare global {
  interface Window {
    turnstile?: {
      render: (container: HTMLElement, options: TurnstileRenderOptions) => string
      reset: (widgetId?: string) => void
      remove: (widgetId?: string) => void
    }
    onTurnstileLoad?: () => void
  }
}

const scriptSelector = 'script[src^="https://challenges.cloudflare.com/turnstile/v0/api.js"]'
let pending: Promise<void> | null = null

// All mounted widgets share one SDK request and callback. A failed request can
// be retried; it must not leave a script element that waits forever on remount.
export function loadTurnstile(): Promise<void> {
  if (window.turnstile) return Promise.resolve()
  if (pending) return pending

  pending = new Promise<void>((resolve, reject) => {
    let script = document.querySelector<HTMLScriptElement>(scriptSelector)
    const created = !script
    if (!script) script = document.createElement('script')
    const element = script
    const previousCallback = window.onTurnstileLoad
    let settled = false
    const cleanup = () => {
      clearTimeout(timer)
      element.removeEventListener('load', loaded)
      element.removeEventListener('error', failed)
      if (window.onTurnstileLoad === loaded) window.onTurnstileLoad = previousCallback
    }
    const loaded = () => {
      if (settled || !window.turnstile) return
      settled = true
      cleanup()
      resolve()
    }
    const failed = () => {
      if (settled) return
      settled = true
      cleanup()
      element.remove()
      reject(new Error('Unable to load Cloudflare Turnstile'))
    }
    const timer = setTimeout(failed, 15000)
    window.onTurnstileLoad = loaded
    element.addEventListener('load', loaded)
    element.addEventListener('error', failed)
    if (created) {
      element.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit&onload=onTurnstileLoad'
      element.async = true
      element.defer = true
      document.head.appendChild(element)
    }
  }).finally(() => { pending = null })
  return pending
}
