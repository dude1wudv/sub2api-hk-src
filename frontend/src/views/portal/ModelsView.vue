<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getModelPlaza, type ModelPlazaResponse } from '@/api/modelPlaza'
import { useAuthStore, useAppStore } from '@/stores'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import PortalApiModels from '@/components/portal/PortalApiModels.vue'

const route = useRoute()
const auth = useAuthStore()
const app = useAppStore()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const embedded = computed(() => route.query.embedded === '1' && auth.isAuthenticated)
const enabled = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.modelPlaza))
const data = ref<ModelPlazaResponse | null>(null)
const loading = ref(false)
const error = ref('')
const groupId = ref<number | null>(null)
const search = ref(typeof route.query.model === 'string' ? route.query.model : '')
let request: AbortController | undefined
async function load() {
  request?.abort()
  const current = new AbortController()
  request = current
  loading.value = true
  error.value = ''
  try {
    if (!app.publicSettingsLoaded) await app.fetchPublicSettings()
    if (current.signal.aborted || !enabled.value) return
    const result = await getModelPlaza({ signal: current.signal })
    if (!current.signal.aborted) data.value = result
  } catch {
    if (!current.signal.aborted) error.value = text('模型价格加载失败，请重试。', 'Could not load model pricing. Please retry.')
  } finally { if (!current.signal.aborted) loading.value = false }
}
onMounted(load)
onBeforeUnmount(() => request?.abort())
</script>

<template>
  <div class="patrick-surface portal-models" :class="{ 'portal-models-public': !embedded }">
    <header v-if="!embedded" class="portal-models-header"><RouterLink to="/home" aria-label="patrickapi"><PatrickBrand /></RouterLink><RouterLink to="/keys" class="portal-button">{{ text('开始接入', 'Get started') }} →</RouterLink></header>
    <div class="portal-page">
      <h1 class="portal-heading">{{ text('模型与价格', 'Models & pricing') }}</h1>
      <PortalApiModels v-model:selected-group-id="groupId" v-model:search="search" :groups="data?.groups || []" :description="data?.description || ''" :pricing-enabled="enabled" :loading="loading" :error="error" :server-utc-offset="app.cachedPublicSettings?.server_utc_offset" @retry="load" />
    </div>
  </div>
</template>

<style scoped>
.portal-models-public { min-height: 100dvh; background: #fffdf7; }
.portal-models-header { width: 86%; margin: 0 auto; display: flex; justify-content: space-between; align-items: center; gap: 20px; padding: 28px 0 48px; }
.portal-models-public .portal-page { max-width: 1160px; }
@media (max-width: 739px) { .portal-models-header { width: calc(100% - 32px); padding-block: 22px 30px; } }
</style>
