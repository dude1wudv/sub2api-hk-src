/**
 * Centralized API error message extraction
 *
 * The API client interceptor rejects with a plain object: { status, code, message, error }
 * This utility extracts the user-facing message from any error shape.
 */

interface ApiErrorLike {
  status?: number
  code?: number | string
  message?: string
  error?: string
  reason?: string
  metadata?: Record<string, unknown>
  response?: {
    data?: {
      detail?: string
      message?: string
      code?: number | string
    }
  }
}

/**
 * Extract the error code from an API error object.
 *
 * Prefers the string `reason` (e.g. "PAYMENT_PROVIDER_MISCONFIGURED") over the
 * numeric HTTP `code`, because reason is granular enough to drive i18n lookup
 * while HTTP code is not.
 */
export function extractApiErrorCode(err: unknown): string | undefined {
  if (!err || typeof err !== 'object') return undefined
  const e = err as ApiErrorLike
  const code = e.reason ?? e.code ?? e.response?.data?.code
  return code != null ? String(code) : undefined
}

/**
 * Extract metadata (interpolation params) from an API error object.
 * Backend errors carry `metadata` with template variables that fill i18n placeholders.
 */
export function extractApiErrorMetadata(err: unknown): Record<string, unknown> | undefined {
  if (!err || typeof err !== 'object') return undefined
  const e = err as ApiErrorLike
  return e.metadata
}

type TranslateFn = (key: string, params?: Record<string, unknown>) => string
type TranslateWithExistsFn = TranslateFn & { te?: (key: string) => boolean }

/**
 * Translate a value via i18n if a matching key exists, otherwise return the original.
 * Example: "certSerial" → t('admin.settings.payment.field_certSerial') → "证书序列号".
 */
function tryTranslate(t: TranslateFn, key: string, fallback: string): string {
  const translated = t(key)
  if (translated === key) return fallback
  const te = (t as TranslateWithExistsFn).te
  if (te && !te(key)) return fallback
  return translated
}

/**
 * Replace raw config field names in metadata (e.g. "certSerial") with their
 * localized UI labels (e.g. "证书序列号"), using the provider-config field i18n namespace.
 * Handles both single `key` and `/`-joined `keys` patterns used by wxpay errors.
 */
function localizeMetadata(metadata: Record<string, unknown>, t: TranslateFn): Record<string, unknown> {
  const out: Record<string, unknown> = { ...metadata }
  if (typeof out.key === 'string') {
    out.key = tryTranslate(t, `admin.settings.payment.field_${out.key}`, out.key)
  }
  if (typeof out.keys === 'string') {
    out.keys = out.keys
      .split('/')
      .map(k => tryTranslate(t, `admin.settings.payment.field_${k}`, k))
      .join(' / ')
  }
  return out
}

/**
 * Extract a localized error message from an API error by looking up
 * `<namespace>.<REASON>` in i18n and substituting metadata as placeholders.
 *
 * Config-field names in metadata (`key` / `keys`) are automatically translated
 * to their UI labels before substitution, so error messages read like
 * "缺少必填项：证书序列号" instead of "缺少必填项：certSerial".
 *
 * @param err      - The caught error
 * @param t        - Vue i18n translate function
 * @param namespace- i18n key prefix, e.g. "payment.errors"
 * @param fallback - Fallback key or plain string if no localized mapping exists
 */
export function extractI18nErrorMessage(
  err: unknown,
  t: TranslateFn,
  namespace: string,
  fallback: string,
): string {
  const code = extractApiErrorCode(err)
  if (code) {
    const key = `${namespace}.${code}`
    const rawMetadata = extractApiErrorMetadata(err) ?? {}
    const metadata = localizeMetadata(rawMetadata, t)
    const translated = t(key, metadata)
    // Vue i18n returns the key itself when missing; detect that and fall back.
    if (translated !== key) return translated
    // If the framework exposes `te`, use it to double-check.
    const te = (t as TranslateWithExistsFn).te
    if (te && te(key)) return translated
  }
  return extractApiErrorMessage(err, fallback)
}

/**
 * Extract a displayable error message from an API error.
 *
 * @param err - The caught error (unknown type)
 * @param fallback - Fallback message if none can be extracted (use t('common.error') or similar)
 * @param i18nMap - Optional map of error codes to i18n translated strings
 */
export function extractApiErrorMessage(
  err: unknown,
  fallback = 'Unknown error',
  i18nMap?: Record<string, string>,
): string {
  if (!err) return fallback

  // Try i18n mapping by error code first
  if (i18nMap) {
    const code = extractApiErrorCode(err)
    if (code && i18nMap[code]) return i18nMap[code]
  }

  // Plain object from API client interceptor (most common case)
  if (typeof err === 'object' && err !== null) {
    const e = err as ApiErrorLike
    // Interceptor shape: { message, error }
    if (e.message) return e.message
    if (e.error) return e.error
    // Legacy axios shape: { response.data.detail }
    if (e.response?.data?.detail) return e.response.data.detail
    if (e.response?.data?.message) return e.response.data.message
  }

  // Standard Error
  if (err instanceof Error) return err.message

  // Last resort
  const str = String(err)
  return str === '[object Object]' ? fallback : str
}

export interface HumanApiError {
  title: string
  description: string
  action: string
  severity: 'info' | 'warning' | 'error'
  code?: string
}

const normalizeErrorText = (value: unknown): string => String(value ?? '').toLowerCase()

/**
 * Convert common API/upstream/redeem errors into actionable user-facing copy.
 * Keeps raw backend text as fallback context, but avoids exposing internal terms
 * as the only explanation.
 */
export function explainHumanApiError(err: unknown, fallback = '操作失败'): HumanApiError {
  const code = extractApiErrorCode(err)
  const message = extractApiErrorMessage(err, fallback)
  const haystack = `${code ?? ''} ${message}`.toLowerCase()

  if (code === 'REDEEM_CODE_NOT_FOUND' || haystack.includes('redeem code not found')) {
    return {
      title: '兑换码不存在',
      description: '系统没有找到这个兑换码。常见原因是输错、复制时多了空格，或购买链接还没有发放成功。',
      action: '重新复制完整兑换码后再试；仍失败就联系购买渠道。',
      severity: 'warning',
      code,
    }
  }

  if (code === 'REDEEM_CODE_USED' || haystack.includes('already used')) {
    return {
      title: '兑换码已使用',
      description: '这个兑换码已经被兑换过，不能重复使用。',
      action: '在最近活动里确认是否已经到账；如果不是本人兑换，联系管理员核查。',
      severity: 'warning',
      code,
    }
  }

  if (code === 'REDEEM_CODE_EXPIRED' || haystack.includes('expired')) {
    return {
      title: '兑换码已过期',
      description: '兑换码超过了可使用时间，系统不会再发放权益。',
      action: '联系购买渠道换新码，或购买新的兑换码。',
      severity: 'warning',
      code,
    }
  }

  if (code === 'REDEEM_RATE_LIMITED' || haystack.includes('too many failed attempts')) {
    return {
      title: '尝试次数过多',
      description: '短时间内失败次数太多，系统临时限制了兑换请求。',
      action: '等一小时后再试，避免继续连续提交错误兑换码。',
      severity: 'warning',
      code,
    }
  }

  if (code === 'REDEEM_CODE_LOCKED' || haystack.includes('being processed')) {
    return {
      title: '兑换码正在处理中',
      description: '同一个兑换码有并发兑换请求，系统已锁定避免重复发放。',
      action: '稍等几秒刷新余额或订阅状态，再决定是否重试。',
      severity: 'info',
      code,
    }
  }

  if (code === 'INSUFFICIENT_BALANCE' || haystack.includes('insufficient balance')) {
    return {
      title: '余额不足',
      description: '当前余额不够完成本次请求或扣减。',
      action: '前往兑换码页面充值，或降低请求消耗后重试。',
      severity: 'warning',
      code,
    }
  }

  if (haystack.includes('group_deleted') || haystack.includes('group deleted')) {
    return {
      title: '服务分组不可用',
      description: '当前 API Key 绑定的模型分组已经被删除或上游账号状态失效。',
      action: '换一个可用 API Key，或联系管理员重新分配分组。',
      severity: 'error',
      code,
    }
  }

  if (haystack.includes('quota') && (haystack.includes('exhaust') || haystack.includes('limit'))) {
    return {
      title: '额度已用完',
      description: '当前 Key、账号或订阅分组的额度限制已触发。',
      action: '查看用量页确认消耗来源，充值或等待额度窗口重置。',
      severity: 'warning',
      code,
    }
  }

  if (haystack.includes('rate limit') || haystack.includes('429')) {
    return {
      title: '请求太频繁',
      description: '短时间内请求过多，上游或平台限流。',
      action: '降低并发或等待几分钟后重试；持续出现则联系管理员扩容。',
      severity: 'warning',
      code,
    }
  }

  if (haystack.includes('403') || haystack.includes('forbidden')) {
    return {
      title: '上游拒绝访问',
      description: '上游账号、出口代理或模型权限暂时不可用。',
      action: '稍后重试；如果一直失败，请把错误码和请求时间发给管理员。',
      severity: 'error',
      code,
    }
  }

  if (haystack.includes('timeout') || haystack.includes('network') || haystack.includes('eof')) {
    return {
      title: '网络或上游超时',
      description: '请求没有稳定到达上游，或上游返回过程中断。',
      action: '重试一次；若多次出现，请到渠道状态页查看服务状态。',
      severity: 'warning',
      code,
    }
  }

  if (normalizeErrorText(message) !== normalizeErrorText(fallback)) {
    return {
      title: fallback,
      description: message,
      action: '如果看不懂这条错误，把完整提示发给管理员。',
      severity: 'error',
      code,
    }
  }

  return {
    title: fallback,
    description: '请求没有成功，系统没有返回更具体的原因。',
    action: '刷新后重试；如果仍失败，联系管理员。',
    severity: 'error',
    code,
  }
}
