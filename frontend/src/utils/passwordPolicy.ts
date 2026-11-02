export interface PasswordPolicy {
  min_length: number
  max_bytes: number
}

export const DEFAULT_PASSWORD_POLICY: Readonly<PasswordPolicy> = Object.freeze({ min_length: 12, max_bytes: 72 })

export function resolvePasswordPolicy(settings?: { password_policy?: Partial<PasswordPolicy> | null } | null): PasswordPolicy {
  const policy = settings?.password_policy
  const minLength = policy?.min_length
  const maxBytes = policy?.max_bytes
  return {
    min_length: typeof minLength === 'number' && Number.isInteger(minLength) && minLength >= 12 ? minLength : DEFAULT_PASSWORD_POLICY.min_length,
    max_bytes: typeof maxBytes === 'number' && Number.isInteger(maxBytes) && maxBytes > 0 && maxBytes <= 72 ? maxBytes : DEFAULT_PASSWORD_POLICY.max_bytes,
  }
}

/** Count Unicode code points, while bcrypt's limit applies to the original UTF-8 bytes. */
export function validateNewPassword(password: string, policy: PasswordPolicy = DEFAULT_PASSWORD_POLICY): 'too-short' | 'too-long' | null {
  if (Array.from(password).length < policy.min_length) return 'too-short'
  if (new TextEncoder().encode(password).length > policy.max_bytes) return 'too-long'
  return null
}

export function passwordPolicyHint(locale: string, policy: PasswordPolicy = DEFAULT_PASSWORD_POLICY): string {
  return locale.startsWith('zh')
    ? `至少 ${policy.min_length} 个字符，最多 ${policy.max_bytes} 个 UTF-8 字节；可使用长口令。`
    : `At least ${policy.min_length} characters, up to ${policy.max_bytes} UTF-8 bytes. Passphrases are welcome.`
}

export function newPasswordError(password: string, locale: string, policy: PasswordPolicy = DEFAULT_PASSWORD_POLICY): string {
  const error = validateNewPassword(password, policy)
  if (!error) return ''
  if (error === 'too-short') return locale.startsWith('zh') ? `密码至少需要 ${policy.min_length} 个字符。` : `Use at least ${policy.min_length} characters.`
  return locale.startsWith('zh') ? `密码不能超过 ${policy.max_bytes} 个 UTF-8 字节，请缩短后重试。` : `The password must not exceed ${policy.max_bytes} UTF-8 bytes.`
}
