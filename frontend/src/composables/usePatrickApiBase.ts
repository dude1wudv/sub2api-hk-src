import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import { sanitizeUrl } from '@/utils/url'

export function usePatrickApiBase() {
  const appStore = useAppStore()
  const apiRoot = computed(() => {
    const configured = appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || ''
    const sanitized = sanitizeUrl(configured)
    if (sanitized) {
      const url = new URL(sanitized)
      if (!url.username && !url.password && !url.search && !url.hash) return sanitized.replace(/\/+$/, '').replace(/(?:\/v1)+$/, '')
    }
    return 'https://patrickapi.microedulab.com'
  })
  const openAIBase = computed(() => `${apiRoot.value}/v1`)
  return { apiRoot, openAIBase }
}
