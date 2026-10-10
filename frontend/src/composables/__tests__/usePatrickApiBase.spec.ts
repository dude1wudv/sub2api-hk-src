import { beforeEach, describe, expect, it, vi } from 'vitest'
const { appStore } = vi.hoisted(() => ({
  appStore: { cachedPublicSettings: {} as Record<string, unknown>, apiBaseUrl: '' },
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

import { usePatrickApiBase } from '../usePatrickApiBase'

describe('usePatrickApiBase', () => {
  beforeEach(() => {
    appStore.cachedPublicSettings = {}
    appStore.apiBaseUrl = ''
  })

  it('uses the fixed Patrick API domain by default', () => {
    const { apiRoot, openAIBase } = usePatrickApiBase()
    expect(apiRoot.value).toBe('https://patrickapi.microedulab.com')
    expect(openAIBase.value).toBe('https://patrickapi.microedulab.com/v1')
  })

  it('prefers the configured public API URL, sanitizes it, and removes trailing /v1', () => {
    appStore.cachedPublicSettings.api_base_url = 'https://api.example.test/v1///'
    appStore.apiBaseUrl = 'https://fallback.example.test'
    const { apiRoot, openAIBase } = usePatrickApiBase()
    expect(apiRoot.value).toBe('https://api.example.test')
    expect(openAIBase.value).toBe('https://api.example.test/v1')
  })

  it('uses the app API URL when no public override exists', () => {
    appStore.apiBaseUrl = 'https://api.example.test/base/'
    const { apiRoot } = usePatrickApiBase()
    expect(apiRoot.value).toBe('https://api.example.test/base')
  })

  it('falls back to the fixed domain when configured URLs are unsafe', () => {
    appStore.cachedPublicSettings.api_base_url = 'javascript:alert(1)'
    appStore.apiBaseUrl = '//untrusted.example.test'
    const { apiRoot } = usePatrickApiBase()
    expect(apiRoot.value).toBe('https://patrickapi.microedulab.com')
  })
})
