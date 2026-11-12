<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getInflight, type InflightItem } from '@/api/inflight'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { durationSeverity, firstTokenSeverity, type LatencySeverity } from '@/utils/latencyHealth'

const props = defineProps<{ admin?: boolean }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const items = ref<InflightItem[]>([])
const total = ref(0)
const known = ref(false)
const loading = ref(false)
const error = ref('')
const page = ref(1)
const model = ref('')
const requestId = ref('')
const appliedFilters = ref<{ model?: string; request_id?: string }>({})
const lastUpdated = ref<number | null>(null)
const now = ref(Date.now())
let controller: AbortController | null = null
let sequence = 0
const expanded = ref(false)
let clock: ReturnType<typeof setInterval> | undefined
const itemKey = (item: InflightItem) => `${item.user_id}:${item.request_id}`
const date = (value: string) => { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? '—' : parsed.toLocaleString(locale.value, { hour12: false }) }
const number = (value: number | null | undefined) => value == null || !Number.isFinite(value) || value < 0 ? '—' : value.toLocaleString(locale.value)
const duration = (value: number | null | undefined) => value == null || !Number.isFinite(value) || value < 0 ? '—' : value < 1000 ? `${Math.round(value)}ms` : `${(value / 1000).toFixed(2)}s`
const elapsedMs = (value: string) => { const ms = now.value - Date.parse(value); return Number.isFinite(ms) ? Math.max(0, ms) : null }
const elapsed = (value: string) => { const ms = elapsedMs(value); return ms == null ? '—' : `${Math.floor(ms / 1000)}s` }
const latencyColors: Record<LatencySeverity, string> = { good: 'rgb(var(--text-ok))', warn: 'rgb(var(--text-warn))', slow: 'rgb(var(--text-warn))', critical: 'rgb(var(--text-err))' }
const latencyColor = (ms: number | null | undefined, first = false) => ms == null || !Number.isFinite(ms) || ms < 0 ? 'rgb(var(--ink-3))' : latencyColors[first ? firstTokenSeverity(ms) : durationSeverity(ms)]
const reserved = (value: number | null | undefined) => value == null || !Number.isFinite(value) || value < 0 ? '—' : value > 0 && value < 0.000001 ? '< USD 0.000001' : `USD ${value.toFixed(value > 0 && value < 0.01 ? 6 : 4)}`
const hasFirstToken = (item: InflightItem) => item.first_token_ms != null && Number.isFinite(item.first_token_ms) && item.first_token_ms >= 0

async function load(append = false) {
  const current = ++sequence
  controller?.abort()
  controller = new AbortController()
  const signal = controller.signal
  const requestedPage = append ? page.value + 1 : 1
  const filters = append ? appliedFilters.value : { model: model.value || undefined, request_id: props.admin ? requestId.value || undefined : undefined }
  loading.value = true
  error.value = ''
  try {
    const result = await getInflight(!!props.admin, {
      page: requestedPage, page_size: props.admin ? 50 : 10,
      ...filters,
    }, signal)
    if (current !== sequence || signal.aborted || !expanded.value) return
    known.value = result.runtime_known
    total.value = result.total
    items.value = !result.runtime_known ? [] : append
      ? [...new Map([...items.value, ...result.items].map(item => [itemKey(item), item])).values()]
      : result.items
    page.value = requestedPage
    appliedFilters.value = filters
    lastUpdated.value = Date.now()
  } catch {
    if (current === sequence && !signal.aborted) error.value = text('在途请求加载失败，请重试。', 'Unable to load requests. Please retry.')
  } finally { if (current === sequence) loading.value = false }
}
function refresh() { void load() }
function more() { void load(true) }
function toggle(event: Event) {
  const next = (event.target as HTMLDetailsElement).open
  if (expanded.value === next) return
  expanded.value = next
  clearInterval(clock)
  if (expanded.value) {
    now.value = Date.now()
    clock = setInterval(() => { now.value = Date.now() }, 1000)
    refresh()
  } else {
    ++sequence
    controller?.abort()
    loading.value = false
    items.value = []
    known.value = false
    error.value = ''
    lastUpdated.value = null
  }
}
onBeforeUnmount(() => { ++sequence; controller?.abort(); clearInterval(clock) })
</script>

<template>
  <details class="inflight" :class="{ 'inflight-admin': admin }" @toggle="toggle">
    <summary class="inflight-summary">
      <span class="inflight-summary-title"><span class="inflight-mark"><Icon name="clock" size="sm" /></span>{{ text('正在处理的请求', 'Requests in progress') }}<span v-if="known" class="inflight-count">{{ number(total) }}</span></span>
      <span class="inflight-summary-meta">{{ expanded ? text('收起面板 · 手动刷新', 'Collapse panel · manual refresh') : text('展开查看 · 手动刷新', 'Expand to view · manual refresh') }}<Icon name="chevronDown" size="sm" /></span>
    </summary>
    <div class="inflight-body" :aria-busy="loading">
      <form class="inflight-controls" @submit.prevent="refresh">
        <label>{{ text('模型', 'Model') }}<input v-model.trim="model" type="search" :placeholder="text('全部模型', 'All models')" /></label>
        <label v-if="admin">{{ text('请求 ID', 'Request ID') }}<input v-model.trim="requestId" type="search" :placeholder="text('搜索 ID 片段', 'Search ID fragment')" /></label>
        <button class="inflight-refresh" type="submit" :disabled="loading"><Icon name="refresh" size="sm" />{{ loading ? text('刷新中…', 'Refreshing…') : text('刷新', 'Refresh') }}</button>
        <span v-if="lastUpdated" class="inflight-updated">{{ text('上次刷新', 'Last refreshed') }} {{ new Date(lastUpdated).toLocaleTimeString(locale, { hour12: false }) }}</span>
      </form>
      <p class="inflight-hint">{{ text('请求尚未结束。金额为预留值，并非最终扣费；Token 可能为估算。刷新后更新请求状态和用量，已运行时间持续计时。', 'Requests are still running. Reserved amounts are not final charges; token counts may be estimates. Refresh to update status and usage; elapsed time keeps counting.') }}</p>
      <div v-if="error" class="inflight-error" role="alert"><span>{{ error }}<small v-if="items.length">{{ text('以下保留上次刷新的记录。', 'The previous snapshot is shown below.') }}</small></span><button type="button" @click="refresh">{{ text('重试', 'Retry') }}</button></div>
      <div v-if="loading && !items.length" class="inflight-state" role="status"><Icon name="refresh" size="md" />{{ text('正在加载在途请求…', 'Loading requests in progress…') }}</div>
      <div v-else-if="!error && !known" class="inflight-state"><Icon name="exclamationCircle" size="md" /><strong>{{ text('当前状态暂不可用', 'Current status is unavailable') }}</strong><span>{{ text('暂时无法确认在途数量，可稍后刷新。', 'The current request count is unknown. Try refreshing shortly.') }}</span></div>
      <div v-else-if="!error && !items.length" class="inflight-state"><Icon name="checkCircle" size="md" /><strong>{{ text('目前没有正在处理的请求。', 'No requests are currently in progress.') }}</strong></div>
      <div v-if="items.length" class="inflight-list">
        <article v-for="item in items" :key="itemKey(item)" class="inflight-request">
          <header class="inflight-request-heading">
            <div class="inflight-request-model"><span class="inflight-model-mark"><ModelIcon :model="item.model" size="20px" aria-hidden="true" /></span><strong>{{ item.model || '—' }}</strong><span class="inflight-type">{{ item.websocket ? 'WebSocket' : item.stream ? text('流式', 'Stream') : text('同步', 'Sync') }}</span><span class="inflight-group">{{ item.group_id ? (item.group_name || `#${item.group_id}`) : text('尚未选组', 'Group pending') }}</span></div>
            <span class="inflight-stage" :class="{ 'has-first-token': hasFirstToken(item) }"><Icon :name="hasFirstToken(item) ? 'checkCircle' : 'clock'" size="xs" aria-hidden="true" />{{ hasFirstToken(item) ? text('已收到首字', 'First token received') : text('处理中', 'Processing') }}</span>
          </header>
          <div class="inflight-request-content">
            <div class="inflight-identity">
              <p v-if="admin"><span><Icon name="user" size="xs" aria-hidden="true" />{{ text('用户', 'User') }}</span><strong>{{ item.email || `#${item.user_id}` }}</strong></p>
              <p><span><Icon name="key" size="xs" aria-hidden="true" />API Key</span><strong>{{ item.api_key_name || `Key #${item.api_key_id}` }}</strong></p>
              <p v-if="admin"><span><Icon name="server" size="xs" aria-hidden="true" />{{ text('账号', 'Account') }}</span><strong>{{ item.account_name || (item.account_id ? `#${item.account_id}` : '—') }}</strong></p>
              <p class="inflight-request-id"><span><Icon name="document" size="xs" aria-hidden="true" />{{ text('请求', 'Request') }}</span><code :title="item.request_id">{{ item.request_id }}</code></p>
            </div>
            <dl class="inflight-metrics">
              <div class="inflight-latency-metric" :style="{ '--latency-color': latencyColor(elapsedMs(item.started_at)) }"><dt><Icon name="clock" size="xs" aria-hidden="true" />{{ text('已运行', 'Elapsed') }}</dt><dd class="inflight-elapsed">{{ elapsed(item.started_at) }}</dd></div>
              <div class="inflight-latency-metric" :style="{ '--latency-color': latencyColor(item.first_token_ms, true) }"><dt><Icon name="bolt" size="xs" aria-hidden="true" />{{ text('首字', 'First token') }}</dt><dd :class="{ 'inflight-first-token': hasFirstToken(item) }">{{ hasFirstToken(item) ? duration(item.first_token_ms) : text('未记录', 'Not recorded') }}</dd></div>
              <div class="inflight-token-metric"><dt><Icon name="cube" size="xs" aria-hidden="true" />Token <span v-if="item.tokens_estimated" class="inflight-estimate">{{ text('估算', 'Estimated') }}</span></dt><dd class="inflight-token-values"><span :title="text('输入 Token', 'Input tokens')"><Icon name="arrowDown" size="sm" />{{ number(item.input_tokens) }}</span><span :title="text('输出 Token', 'Output tokens')"><Icon name="arrowUp" size="sm" />{{ number(item.output_tokens) }}</span></dd></div>
              <div class="inflight-cost-metric"><dt><Icon name="creditCard" size="xs" aria-hidden="true" />{{ text('预留金额', 'Reserved amount') }}</dt><dd class="inflight-reserved">{{ reserved(item.reserved_amount) }}</dd></div>
            </dl>
          </div>
          <footer class="inflight-request-footer"><span>{{ text('开始于', 'Started') }} <time>{{ date(item.started_at) }}</time></span><span v-if="admin && item.upstream_endpoint" class="inflight-endpoint" :title="item.upstream_endpoint">{{ text('上游', 'Upstream') }} {{ item.upstream_endpoint }}</span></footer>
        </article>
      </div>
      <footer v-if="known" class="inflight-list-footer"><span>{{ text(`已显示 ${items.length} / ${total} 个请求`, `Showing ${items.length} / ${total} requests`) }}</span><button v-if="items.length < total" type="button" :disabled="loading" @click="more">{{ text('加载更多', 'Load more') }}</button></footer>
    </div>
  </details>
</template>

<style scoped>
.inflight { margin-bottom:18px; border:1px solid rgb(var(--line)); border-radius:var(--radius-card); background:rgb(var(--surface)); color:rgb(var(--ink)); font:14px/1.6 var(--font-sans); }
.inflight-summary { display:flex; align-items:center; justify-content:space-between; gap:16px; padding:15px 18px; cursor:pointer; list-style:none; }
.inflight-summary::-webkit-details-marker { display:none; }
.inflight-summary-title, .inflight-summary-meta { display:flex; align-items:center; gap:9px; }
.inflight-summary-title { min-width:0; font-weight:500; }
.inflight-mark { display:inline-grid; place-items:center; width:30px; height:30px; flex-shrink:0; border-radius:9px; background:rgb(var(--status-warn-bg)); color:rgb(var(--text-warn)); }
.inflight-count { min-width:26px; padding:1px 8px; border:1px solid rgb(var(--line)); border-radius:999px; background:rgb(var(--surface-2)); font:500 13px/1.6 ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace; text-align:center; }
.inflight-summary-meta { color:rgb(var(--ink-3)); font-size:13px; flex-shrink:0; }
.inflight[open] .inflight-summary { border-bottom:1px solid rgb(var(--line)); }
.inflight[open] .inflight-summary-meta svg { transform:rotate(180deg); }
.inflight-body { padding:18px; }
.inflight-controls { display:flex; flex-wrap:wrap; align-items:end; gap:12px; }
.inflight-controls label { display:grid; gap:6px; min-width:0; font-size:13px; color:rgb(var(--ink-2)); }
.inflight-controls input { width:220px; max-width:100%; min-height:40px; padding:8px 11px; border:1px solid rgb(var(--line)); border-radius:var(--radius-control); background:rgb(var(--surface)); color:rgb(var(--ink)); font:inherit; }
.inflight button { min-height:40px; padding:8px 12px; border:1px solid rgb(var(--line)); border-radius:var(--radius-control); background:rgb(var(--surface)); color:rgb(var(--ink)); font:inherit; font-size:13px; }
.inflight button:hover:not(:disabled) { background:rgb(var(--surface-2)); }
.inflight button:disabled { opacity:.5; cursor:wait; }
.inflight-refresh { display:inline-flex; align-items:center; justify-content:center; gap:7px; }
.inflight-updated { margin-left:auto; padding-bottom:8px; color:rgb(var(--ink-3)); font-size:12px; }
.inflight-hint { margin:14px 0 18px; color:rgb(var(--ink-3)); font-size:13px; line-height:1.65; }
.inflight-error { display:flex; align-items:center; justify-content:space-between; gap:16px; margin-bottom:16px; padding:12px 14px; border:1px solid rgb(var(--line)); border-radius:10px; color:rgb(var(--text-err)); background:rgb(var(--status-err-bg)); }
.inflight-error small { display:block; margin-top:3px; font-size:12px; }
.inflight-state { display:flex; flex-direction:column; align-items:center; justify-content:center; gap:8px; min-height:140px; color:rgb(var(--ink-3)); text-align:center; }
.inflight-state strong { font-size:14px; font-weight:500; }
.inflight-state > span { font-size:13px; }
.inflight-list { display:grid; gap:12px; }
.inflight-request { min-width:0; overflow:hidden; border:1px solid rgb(var(--line)); border-radius:var(--radius-card); background:rgb(var(--surface)); }
.inflight-request-heading, .inflight-request-model { display:flex; align-items:center; flex-wrap:wrap; gap:8px; min-width:0; }
.inflight-request-heading { justify-content:space-between; gap:10px; padding:13px 16px; border-bottom:1px solid rgb(var(--line)); background:color-mix(in srgb,rgb(var(--surface-2)) 60%,rgb(var(--surface))); }
.inflight-model-mark { display:inline-grid; place-items:center; flex-shrink:0; width:32px; height:32px; border:1px solid rgb(var(--line)); border-radius:9px; background:rgb(var(--color-dark-100)); }
.inflight-request-model > strong { font:500 14px/1.6 ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace; overflow-wrap:anywhere; }
.inflight-type, .inflight-group { padding:1px 8px; border-radius:5px; font-size:12px; font-weight:500; color:rgb(var(--text-link)); background:rgb(var(--status-link-bg)); overflow-wrap:anywhere; }
.inflight-group { border:1px solid rgb(var(--color-link-200)); border-radius:999px; }
:global(.dark) .inflight-group { border-color:rgb(var(--color-link-900)); }
.inflight-stage { display:inline-flex; align-items:center; gap:5px; flex-shrink:0; padding:3px 9px; border-radius:999px; background:rgb(var(--status-warn-bg)); color:rgb(var(--text-warn)); font-size:12px; font-weight:500; }
.inflight-stage.has-first-token { background:rgb(var(--status-ok-bg)); color:rgb(var(--text-ok)); }
.inflight-request-content { display:grid; grid-template-columns:minmax(200px,.9fr) minmax(0,2fr); gap:20px; padding:16px; }
.inflight-identity { display:grid; gap:4px; min-width:0; }
.inflight-identity p { display:grid; grid-template-columns:74px minmax(0,1fr); gap:8px; margin:0; min-width:0; font-size:13px; }
.inflight-identity p > span { display:inline-flex; align-items:center; gap:6px; color:rgb(var(--ink-3)); }
.inflight-identity p > span svg { flex-shrink:0; color:rgb(var(--ink-3)); }
.inflight-identity strong { font-weight:500; overflow-wrap:anywhere; }
.inflight-request-id code { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; color:rgb(var(--ink-2)); font-size:12px; }
.inflight-metrics { display:grid; grid-template-columns:.75fr .9fr 1.3fr 1.15fr; gap:1px; margin:0; border-radius:10px; overflow:hidden; background:rgb(var(--line)); align-content:stretch; }
.inflight-metrics > div { min-width:0; padding:12px; background:rgb(var(--surface-2)); }
.inflight-metrics dt { display:flex; align-items:center; flex-wrap:wrap; gap:5px; color:rgb(var(--ink-3)); font-size:12px; margin-bottom:7px; }
.inflight-metrics dt svg { flex-shrink:0; }
.inflight-metrics .inflight-latency-metric { position:relative; padding-left:15px; }
.inflight-latency-metric::before { content:''; position:absolute; top:12px; bottom:12px; left:0; width:3px; border-radius:0 3px 3px 0; background:var(--latency-color); }
.inflight-metrics dd { margin:0; font:500 14px/1.6 ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace; overflow-wrap:anywhere; }
.inflight-metrics .inflight-elapsed { font-size:20px; color:var(--latency-color); }
.inflight-metrics .inflight-first-token { color:var(--latency-color); }
.inflight-metrics .inflight-cost-metric { background:color-mix(in srgb,rgb(var(--status-warn-bg)) 55%,rgb(var(--surface-2))); }
.inflight-metrics .inflight-reserved { display:inline-block; padding:3px 6px; border-radius:6px; background:rgb(var(--status-warn-bg)); font-size:13px; font-weight:600; color:rgb(var(--text-warn)); }
.inflight-token-values { display:flex; flex-wrap:wrap; gap:4px 12px; }
.inflight-token-values > span { display:inline-flex; align-items:center; gap:4px; }
.inflight-token-values > span:first-child svg { color:rgb(var(--text-ok)); }
.inflight-token-values > span:last-child svg { color:rgb(var(--text-accent)); }
.inflight-estimate { display:inline-block; padding:0 4px; border:1px solid rgb(var(--line)); border-radius:4px; color:rgb(var(--text-warn)); font:500 11px/1.6 var(--font-sans); }
.inflight-request-footer { display:flex; align-items:center; justify-content:space-between; flex-wrap:wrap; gap:8px 16px; margin-top:0; padding:10px 16px; border-top:1px solid rgb(var(--line)); color:rgb(var(--ink-3)); font-size:12px; }
.inflight-endpoint { min-width:0; max-width:100%; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.inflight-list-footer { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-top:14px; color:rgb(var(--ink-3)); font-size:13px; }
.inflight :is(summary,button,input):focus-visible { outline:2px solid rgb(var(--focus-ring)); outline-offset:3px; }
@media(max-width:1100px) { .inflight-request-content { grid-template-columns:minmax(180px,1fr) minmax(0,1.7fr); gap:18px; } .inflight-metrics { grid-template-columns:repeat(2,minmax(0,1fr)); } }
@media(max-width:767px) { .inflight-summary { padding:14px; gap:8px; } .inflight-summary-meta { font-size:0; gap:0; } .inflight-body { padding:14px; } .inflight-controls { display:grid; grid-template-columns:minmax(0,1fr); } .inflight-controls input { width:100%; min-height:44px; font-size:16px; } .inflight button { min-height:44px; } .inflight-updated { margin-left:0; padding:0; } .inflight-request-content { grid-template-columns:minmax(0,1fr); gap:18px; } .inflight-request-heading { align-items:flex-start; padding:12px; } .inflight-request-content { padding:14px 12px; } .inflight-request-footer { padding:10px 12px; } .inflight-metrics > div { padding:11px; } .inflight-metrics .inflight-latency-metric { padding-left:14px; } .inflight-request-model { flex:1; } .inflight-metrics .inflight-elapsed { font-size:18px; } }
</style>
