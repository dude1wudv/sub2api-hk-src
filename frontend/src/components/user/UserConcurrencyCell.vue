<template>
  <div class="flex flex-col items-start">
    <span
      :class="[
        'inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium tabular-nums',
        statusClass
      ]"
    >
      <!-- Four-square grid icon -->
      <svg class="h-3 w-3 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
      </svg>
      <span class="font-mono">{{ current }}</span>
      <span class="text-gray-400 dark:text-dark-500">/</span>
      <span class="font-mono">{{ max }}</span>
    </span>
    <details v-if="userId" class="text-xs" @toggle="toggle">
      <summary class="cursor-pointer text-gray-500">{{ locale.startsWith('zh') ? '按组查看' : 'By group' }}</summary>
      <div v-if="loading" role="status">{{ locale.startsWith('zh') ? '加载中…' : 'Loading…' }}</div>
      <div v-else-if="error" role="alert">{{ locale.startsWith('zh') ? '暂不可用' : 'Unavailable' }} <button type="button" @click="load">↻</button></div>
      <div v-else-if="!known">{{ locale.startsWith('zh') ? '状态未知' : 'Status unknown' }}</div>
      <div v-else-if="!items.length">{{ locale.startsWith('zh') ? '无在途请求' : 'No requests in progress' }}</div>
      <ul v-else><li v-for="item in items" :key="item.group_id">{{ item.group_id ? (item.group_name || `#${item.group_id}`) : (locale.startsWith('zh') ? '尚未选组' : 'Group pending') }} · {{ item.inflight }}<small v-if="item.models?.length"> ({{ item.models.join(', ') }})</small></li></ul>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'

const props = defineProps<{
  current: number
  max: number
  userId?: number
}>()
const { locale } = useI18n()
const items = ref<Array<{group_id:number; group_name?:string; inflight:number; models:string[]}>>([])
const known = ref(false), loading = ref(false), error = ref(false)
let controller: AbortController | null = null
let seq = 0
async function load() {
  if (!props.userId) return
  const current = ++seq; controller?.abort(); controller = new AbortController(); const signal = controller.signal
  loading.value = true; error.value = false
  try {
    const { data } = await apiClient.get<{items: typeof items.value; runtime_known:boolean}>(`/admin/users/${props.userId}/inflight`, { signal })
    if (current === seq && !signal.aborted) { items.value = data.items; known.value = data.runtime_known }
  } catch { if (current === seq && !signal.aborted) error.value = true }
  finally { if (current === seq) loading.value = false }
}
function toggle(event: Event) { if ((event.target as HTMLDetailsElement).open) void load(); else { ++seq; controller?.abort(); loading.value = false } }
onBeforeUnmount(() => { ++seq; controller?.abort() })

// Status color based on usage
const statusClass = computed(() => {
  const { current, max } = props

  // Full: red
  if (current >= max && max > 0) {
    return 'bg-err-50 text-err-700 dark:bg-err-950/40 dark:text-err-300'
  }
  // In use: yellow/amber
  if (current > 0) {
    return 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
  }
  // Idle: gray
  return 'bg-gray-100/80 text-gray-600 dark:bg-dark-700/60 dark:text-dark-300'
})
</script>
