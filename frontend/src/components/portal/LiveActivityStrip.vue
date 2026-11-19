<template>
  <RouterLink
    v-if="activeTotal > 0"
    to="/usage"
    class="live-activity-strip"
    :aria-label="zh ? `${activeTotal} 个进行中请求` : `${activeTotal} active requests`"
  >
    <span class="live-activity-pulse" aria-hidden="true" />
    <span class="live-activity-text">
      <strong>{{ activeTotal }}</strong>
      {{ zh ? '个进行中请求' : activeTotal === 1 ? 'active request' : 'active requests' }}
      <template v-if="topModel"> · <code>{{ topModel }}</code></template>
    </span>
    <span class="live-activity-link">{{ zh ? '查看实时详情 →' : 'Inspect →' }}</span>
  </RouterLink>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getInflight, type InflightItem } from '@/api/inflight'
import { useAuthStore } from '@/stores/auth'

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const authStore = useAuthStore()

const activeTotal = ref(0)
const items = ref<InflightItem[]>([])
let timer: number | undefined
let controller: AbortController | null = null

const topModel = computed(() => items.value[0]?.model || '')

async function poll() {
  if (!authStore.isAuthenticated || authStore.isSimpleMode) {
    activeTotal.value = 0
    return
  }
  if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return

  controller?.abort()
  controller = new AbortController()
  try {
    const res = await getInflight(false, { page: 1, page_size: 3 }, controller.signal)
    if (res && res.runtime_known !== false) {
      activeTotal.value = Number(res.total || res.items?.length || 0)
      items.value = Array.isArray(res.items) ? res.items : []
    } else {
      activeTotal.value = 0
      items.value = []
    }
  } catch {
    // Stay silent on rate limit or transient errors so the strip never disrupts browsing
  }
}

function handleVisibility() {
  if (document.visibilityState === 'visible') {
    void poll()
  }
}

onMounted(() => {
  void poll()
  timer = window.setInterval(() => {
    void poll()
  }, 15000)
  document.addEventListener('visibilitychange', handleVisibility)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
  controller?.abort()
  document.removeEventListener('visibilitychange', handleVisibility)
})
</script>

<style scoped>
.live-activity-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  padding: 9px 14px;
  border-radius: 12px;
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
  font-size: 0.8125rem;
  text-decoration: none;
  border: 1px solid color-mix(in srgb, var(--md-sys-color-primary) 26%, transparent);
  transition: filter var(--motion-fast, 150ms);
}

.live-activity-strip:hover {
  filter: brightness(0.98);
}

.live-activity-pulse {
  width: 8px;
  height: 8px;
  border-radius: 9999px;
  background: var(--md-sys-color-primary);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--md-sys-color-primary) 25%, transparent);
  flex-shrink: 0;
}

.live-activity-text {
  flex: 1 1 auto;
  min-width: 0;
}

.live-activity-text code {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  padding: 1px 6px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--md-sys-color-on-primary-container) 10%, transparent);
}

.live-activity-link {
  font-weight: 600;
  white-space: nowrap;
}
</style>
