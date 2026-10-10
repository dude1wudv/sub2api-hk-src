import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import { sanitizeUrl } from '@/utils/url'

export function usePatrickApiBase() {
  const appStore = useAppStore()
  const apiRoot = computed(() => {
    const configured = appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || ''
    return (sanitizeUrl(configured) || 'https://patrickapi.microedulab.com').replace(/\/+$/, '').replace(/\/v1$/, '')
  })
  const openAIBase = computed(() => `${apiRoot.value}/v1`)
  return { apiRoot, openAIBase }
}
