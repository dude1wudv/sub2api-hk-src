<template>
  <div v-if="activeTotal > 0" class="live-activity-wrap">
    <RouterLink
      to="/usage"
      class="live-activity-strip"
      :aria-label="zh ? `${activeTotal} 个进行中请求` : `${activeTotal} active requests`"
    >
      <span class="live-activity-pulse" aria-hidden="true" />
      <span class="live-activity-text">
        <strong>{{ activeTotal }}</strong>
        {{ zh ? '个进行中请求' : activeTotal === 1 ? 'active request' : 'active requests' }}
        <template v-if="topModel">
          <span class="live-activity-sep" aria-hidden="true">·</span>
          <code>{{ topModel }}</code>
        </template>
      </span>
      <span class="live-activity-link">
        {{ zh ? '查看实时详情' : 'Inspect' }}
        <span aria-hidden="true">→</span>
      </span>
    </RouterLink>
  </div>
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
.live-activity-wrap {
  margin-bottom: 20px;
  display: flex;
  align-items: center;
}

.live-activity-strip {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;
  max-width: 100%;
  padding: 7px 14px;
  border-radius: 999px;
  background: rgb(var(--surface));
  color: rgb(var(--ink));
  font-size: 13px;
  line-height: 1.4;
  text-decoration: none;
  border: 1px solid rgb(var(--line));
  box-shadow: var(--shadow-panel);
  transition: background-color var(--motion-fast, 150ms) ease, border-color var(--motion-fast, 150ms) ease;
}

.live-activity-strip:hover {
  background: rgb(var(--surface-2));
  border-color: rgb(var(--line-strong));
}

.live-activity-strip:focus-visible {
  outline: 2px solid rgb(var(--focus-ring));
  outline-offset: 2px;
}

.live-activity-pulse {
  width: 8px;
  height: 8px;
  border-radius: 9999px;
  background: #10b981;
  box-shadow: 0 0 0 3px rgb(16 185 129 / 0.22);
  flex-shrink: 0;
  animation: live-pulse 2s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% {
    box-shadow: 0 0 0 2px rgb(16 185 129 / 0.26);
  }
  50% {
    box-shadow: 0 0 0 5px rgb(16 185 129 / 0.08);
  }
}

.live-activity-text {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 5px;
  min-width: 0;
  color: rgb(var(--ink-2));
}

.live-activity-text strong {
  color: rgb(var(--ink));
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.live-activity-sep {
  color: rgb(var(--ink-3));
}

.live-activity-text code {
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 500;
  padding: 1px 7px;
  border-radius: 6px;
  color: rgb(var(--ink));
  background: rgb(var(--surface-2));
  border: 1px solid rgb(var(--line));
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.live-activity-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding-left: 8px;
  border-left: 1px solid rgb(var(--line));
  color: rgb(var(--ink));
  font-weight: 600;
  white-space: nowrap;
}

@media (max-width: 767px) {
  .live-activity-strip {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    width: 100%;
    border-radius: 12px;
  }

  .live-activity-text code {
    max-width: 100%;
  }

  .live-activity-link {
    border-left: 0;
    padding-left: 0;
  }
}
</style>
