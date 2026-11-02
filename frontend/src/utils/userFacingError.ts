import { extractApiErrorCode, extractApiErrorMetadata } from './apiError'

export function retryAfterSeconds(error: unknown): number | null {
  const value = error as { metadata?: Record<string, unknown>; retry_after?: unknown; response?: { headers?: Record<string, unknown> } } | null
  const metadata = extractApiErrorMetadata(error)
  const raw = metadata?.retry_after_seconds ?? metadata?.retry_after ?? value?.retry_after ?? value?.response?.headers?.['retry-after']
  const seconds = Number(raw)
  if (Number.isFinite(seconds) && seconds > 0) return Math.ceil(seconds)
  if (typeof raw === 'string') {
    const delay = Math.ceil((Date.parse(raw) - Date.now()) / 1000)
    if (Number.isFinite(delay) && delay > 0) return delay
  }
  return null
}

/** User pages never present an unlocalized implementation error as the explanation. */
export function userFacingError(error: unknown, locale: string, fallback: string): string {
  const value = error as { status?: number; message?: string; response?: { status?: number } } | null
  const code = extractApiErrorCode(error)?.toUpperCase() || ''
  const status = value?.status ?? value?.response?.status
  const zh = locale.startsWith('zh')
  if (status === 429 || /RATE_LIMIT|TOO_MANY/.test(code) || /rate.limit|too many requests/i.test(value?.message || '')) {
    const seconds = retryAfterSeconds(error)
    return zh ? seconds ? `操作过于频繁，请在 ${seconds} 秒后重试。` : '操作过于频繁，请稍后重试。' : seconds ? `Too many requests. Try again in ${seconds} seconds.` : 'Too many requests. Please try again later.'
  }
  if (status === 401) return zh ? '登录状态已失效，请重新登录。' : 'Your session has expired. Please sign in again.'
  if (status === 403) return zh ? '当前账户没有执行此操作的权限。' : 'Your account does not have permission for this action.'
  if (/MCP_DISABLED/.test(code) || /MCP is disabled/i.test(value?.message || '')) return zh ? '本站暂未开放 MCP。' : 'MCP is not available on this site yet.'
  const message = value?.message?.trim()
  if (message && (!zh || /[\u3400-\u9fff]/.test(message))) return message
  return fallback
}
