<template>
  <details class="inflight" @toggle="toggle">
    <summary>{{ text('正在处理的请求', 'Requests in progress') }} <span v-if="known">{{ total }}</span></summary>
    <div class="inflight-body">
      <form class="inflight-controls" @submit.prevent="refresh">
        <label>{{ text('模型', 'Model') }} <input v-model.trim="model" type="search" /></label>
        <label>{{ text('请求 ID', 'Request ID') }} <input v-model.trim="requestId" type="search" /></label>
        <button type="submit" :disabled="loading">{{ loading ? text('加载中…', 'Loading…') : text('刷新', 'Refresh') }}</button>
      </form>
      <p class="inflight-hint">{{ text('金额为预留值，并非最终扣费；Token 可能为估算。', 'Reserved amounts are not final charges; token counts may be estimates.') }}</p>
      <p v-if="error" role="alert">{{ error }}</p>
      <p v-else-if="loading && !items.length" role="status">{{ text('正在加载…', 'Loading…') }}</p>
      <p v-else-if="!known">{{ text('当前状态暂不可用。', 'Current status is unavailable.') }}</p>
      <p v-else-if="!items.length">{{ text('目前没有正在处理的请求。', 'No requests are currently in progress.') }}</p>
      <div v-else class="inflight-scroll"><table><thead><tr><th>{{ text('模型 / 分组', 'Model / group') }}</th><th>{{ text('开始 / 耗时', 'Started / elapsed') }}</th><th>Tokens</th><th>{{ text('预留', 'Reserved') }}</th><th>{{ text('请求', 'Request') }}</th><th v-if="admin">{{ text('用户 / 账号', 'User / account') }}</th></tr></thead>
        <tbody><tr v-for="item in items" :key="itemKey(item)"><td><strong>{{ item.model || '—' }}</strong><small>{{ item.group_id ? (item.group_name || `#${item.group_id}`) : text('尚未选组', 'Group pending') }}</small></td><td>{{ date(item.started_at) }}<small>{{ elapsed(item.started_at) }}<template v-if="item.first_token_ms != null"> · {{ text('首字', 'First token') }} {{ item.first_token_ms }}ms</template></small></td><td>{{ item.input_tokens ?? '—' }} / {{ item.output_tokens ?? '—' }}<small v-if="item.tokens_estimated">{{ text('估算', 'Estimated') }}</small></td><td>{{ item.reserved_amount == null ? '—' : '$' + item.reserved_amount.toFixed(4) }}</td><td><code>{{ item.request_id.slice(0, 12) }}</code><small>{{ item.api_key_name || `Key #${item.api_key_id}` }}</small></td><td v-if="admin">{{ item.email || `#${item.user_id}` }}<small>{{ item.account_name || (item.account_id ? `#${item.account_id}` : '—') }}</small></td></tr></tbody></table></div>
      <button v-if="known && items.length < total" type="button" :disabled="loading" @click="more">{{ text('加载更多', 'Load more') }}</button>
    </div>
  </details>
</template>
<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getInflight, type InflightItem } from '@/api/inflight'
const props = defineProps<{ admin?: boolean }>()
const { locale } = useI18n(); const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const items = ref<InflightItem[]>([]), total = ref(0), known = ref(false), loading = ref(false), error = ref(''), page = ref(1), model = ref(''), requestId = ref('')
let controller: AbortController | null = null; let sequence = 0; let open = false; let clock: ReturnType<typeof setInterval> | undefined
const now = ref(Date.now())
const itemKey = (item: InflightItem) => `${item.user_id}:${item.request_id}`
const date = (value: string) => { const d = new Date(value); return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString(locale.value) }
const elapsed = (value: string) => { const ms = now.value - Date.parse(value); return Number.isFinite(ms) ? `${Math.max(0, Math.floor(ms / 1000))}s` : '—' }
async function load(append = false) {
  const current = ++sequence; controller?.abort(); controller = new AbortController(); const signal = controller.signal
  loading.value = true; error.value = ''
  try {
    const result = await getInflight(!!props.admin, { page: page.value, page_size: props.admin ? 50 : 10, model: model.value || undefined, request_id: requestId.value || undefined }, signal)
    if (current !== sequence || signal.aborted || !open) return
    known.value = result.runtime_known; total.value = result.total
    items.value = append ? [...new Map([...items.value, ...result.items].map(item => [itemKey(item), item])).values()] : result.items
  } catch { if (current === sequence && !signal.aborted) { error.value = text('在途请求加载失败，请重试。', 'Unable to load requests. Please retry.'); known.value = false } }
  finally { if (current === sequence) loading.value = false }
}
function refresh() { page.value = 1; items.value = []; void load() }
function more() { page.value++; void load(true) }
function toggle(event: Event) { const next = (event.target as HTMLDetailsElement).open; if (open === next) return; open = next; clearInterval(clock); if (open) { now.value = Date.now(); clock = setInterval(() => { now.value = Date.now() }, 1000); refresh() } else { ++sequence; controller?.abort(); loading.value = false; items.value = []; known.value = false } }
onBeforeUnmount(() => { ++sequence; controller?.abort(); clearInterval(clock) })
</script>
<style scoped>
.inflight { border:1px solid rgb(var(--line, 220 220 220)); border-radius:10px; background:rgb(var(--surface, 255 255 255)); margin-bottom:18px; } summary { cursor:pointer; padding:16px; font-weight:600; } summary span { margin-left:8px; color:rgb(var(--ink-3, 100 100 100)); } .inflight-body { padding:0 16px 16px; font-size:13px; } .inflight-controls { display:flex; flex-wrap:wrap; align-items:end; gap:10px; } .inflight-controls label { display:grid; gap:4px; } input { max-width:180px; padding:6px 8px; border:1px solid rgb(var(--line, 200 200 200)); border-radius:5px; } button { padding:6px 10px; border:1px solid rgb(var(--line, 200 200 200)); border-radius:5px; } .inflight-hint { color:rgb(var(--ink-3, 100 100 100)); margin:10px 0; } .inflight-scroll { overflow:auto; } table { width:100%; border-collapse:collapse; } td,th { padding:9px; border-bottom:1px solid rgb(var(--line, 220 220 220)); text-align:left; white-space:nowrap; } small { display:block; color:rgb(var(--ink-3, 100 100 100)); margin-top:3px; }
</style>
