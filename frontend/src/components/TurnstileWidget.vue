<template>
  <div v-if="siteKey" class="turnstile-wrapper relative">
    <div
      v-if="loading"
      role="status"
      class="absolute inset-0 flex items-center justify-center rounded-lg border border-gray-200 bg-gray-50 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-400"
    >
      {{ t('auth.captchaLoading') }}
    </div>
    <div v-if="failed" role="alert" class="rounded-lg border border-red-200 p-3 text-sm dark:border-red-800">
      <p>{{ t('auth.captchaLoadFailed') }}</p>
      <button type="button" class="btn btn-secondary mt-2" @click="initialize">
        {{ t('auth.captchaRetry') }}
      </button>
    </div>
    <div ref="containerRef" class="turnstile-container"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { loadTurnstile } from '@/utils/turnstile'

const props = withDefaults(
  defineProps<{
    siteKey: string
    theme?: 'light' | 'dark' | 'auto'
    size?: 'normal' | 'compact' | 'flexible'
  }>(),
  {
    theme: 'auto',
    size: 'flexible'
  }
)

const emit = defineEmits<{
  (e: 'verify', token: string): void
  (e: 'expire'): void
  (e: 'error'): void
}>()

const { t } = useI18n()
const loading = ref(true)
const failed = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const widgetId = ref<string | null>(null)
let mounted = false
let generation = 0

function removeWidget() {
  if (widgetId.value) {
    try {
      window.turnstile?.remove(widgetId.value)
    } catch {
      // Ignore errors when removing
    }
    widgetId.value = null
  }

}

async function initialize() {
  const current = ++generation
  if (!mounted) return
  if (widgetId.value) emit('expire')
  removeWidget()
  failed.value = false
  loading.value = Boolean(props.siteKey)
  if (!props.siteKey) return
  try {
    await loadTurnstile()
    await nextTick()
    if (!mounted || current !== generation || !containerRef.value || !window.turnstile) return
    containerRef.value.innerHTML = ''
    const active = () => mounted && current === generation
    const onFailure = () => {
      if (!active()) return true
      failed.value = true
      emit('error')
      return true
    }
    widgetId.value = window.turnstile.render(containerRef.value, {
      sitekey: props.siteKey,
      callback: (token: string) => {
        if (!active()) return
        failed.value = false
        emit('verify', token)
      },
      'expired-callback': () => { if (active()) emit('expire') },
      'error-callback': onFailure,
      'timeout-callback': onFailure,
      theme: props.theme,
      size: props.size
    })
  } catch {
    if (mounted && current === generation) {
      failed.value = true
      emit('error')
    }
  } finally {
    if (mounted && current === generation) loading.value = false
  }
}

function reset() {
  emit('expire')
  if (window.turnstile && widgetId.value) {
    window.turnstile.reset(widgetId.value)
  } else {
    void initialize()
  }
}

// Expose reset method to parent
defineExpose({ reset })

onMounted(() => {
  mounted = true
  void initialize()
})

onUnmounted(() => {
  mounted = false
  generation++
  removeWidget()
})

// Re-render when siteKey changes
watch(
  () => [props.siteKey, props.theme],
  () => { void initialize() },
  { flush: 'post' }
)
</script>

<style scoped>
.turnstile-wrapper {
  width: 100%;
}

.turnstile-container {
  width: 100%;
  min-height: 65px;
}

/* Make the Turnstile iframe fill the container width */
.turnstile-container :deep(iframe) {
  width: 100% !important;
}
</style>
