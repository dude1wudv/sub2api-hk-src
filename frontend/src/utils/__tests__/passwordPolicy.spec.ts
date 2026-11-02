import { describe, expect, it } from 'vitest'
import { DEFAULT_PASSWORD_POLICY, newPasswordError, resolvePasswordPolicy, validateNewPassword } from '../passwordPolicy'

describe('new password policy', () => {
  it('requires twelve Unicode code points, including astral characters', () => {
    expect(validateNewPassword('abcdefghijk')).toBe('too-short')
    expect(validateNewPassword('abcdefghijkl')).toBeNull()
    expect(validateNewPassword('😀'.repeat(6))).toBe('too-short')
    expect(validateNewPassword('😀'.repeat(12))).toBeNull()
  })

  it('checks bcrypt bytes without trimming or normalizing the secret', () => {
    expect(validateNewPassword('a'.repeat(72))).toBeNull()
    expect(validateNewPassword('a'.repeat(73))).toBe('too-long')
    expect(validateNewPassword('密'.repeat(24))).toBeNull()
    expect(validateNewPassword('密'.repeat(25))).toBe('too-long')
    expect(validateNewPassword(' '.repeat(12))).toBeNull()
  })

  it('defaults safely when settings are missing and explains each failure', () => {
    expect(resolvePasswordPolicy(null)).toEqual(DEFAULT_PASSWORD_POLICY)
    expect(resolvePasswordPolicy({ password_policy: { min_length: 6, max_bytes: 100 } })).toEqual(DEFAULT_PASSWORD_POLICY)
    expect(newPasswordError('short', 'zh-CN')).toContain('12')
    expect(newPasswordError('密'.repeat(25), 'zh-CN')).toContain('72')
    expect(newPasswordError('a valid long passphrase', 'en')).toBe('')
  })
})
