interface APIErrorLike {
  message?: string
  reason?: string
  code?: string | number
  response?: {
    data?: {
      detail?: string
      message?: string
      code?: string | number
    }
  }
}

function extractErrorCode(error: unknown): string {
  const err = (error || {}) as APIErrorLike
  const code = err.reason || err.code || err.response?.data?.code
  return code != null ? String(code) : ''
}

function extractErrorMessage(error: unknown): string {
  const err = (error || {}) as APIErrorLike
  return err.response?.data?.detail || err.response?.data?.message || err.message || ''
}

export function buildAuthErrorMessage(
  error: unknown,
  options: {
    fallback: string
    t?: (key: string) => string
  }
): string {
  const { fallback, t } = options
  const code = extractErrorCode(error)
  if (code && t) {
    const key = `auth.errors.${code}`
    const translated = t(key)
    if (translated !== key) return translated
  }
  const message = extractErrorMessage(error)
  return message || fallback
}
