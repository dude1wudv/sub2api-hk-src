<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getExternalQuality, getExternalQualityArtwork, type ExternalQualityEvent, type ExternalQualityReport } from '@/api/externalQuality'
import { userGroupsAPI } from '@/api/groups'
import type { Group } from '@/types'
import { qualityArtifactDocument } from '@/utils/qualityArtifact'

const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const groups = ref<Group[]>([])
const reports = ref<Record<number, ExternalQualityReport>>({})
const loading = ref(false)
const failed = ref(false)
let controller: AbortController | undefined
let timer: ReturnType<typeof setInterval> | undefined
let alive = true
const eligible = computed(() => groups.value.filter(group => [72, 237].includes(group.id)).sort((a, b) => a.id - b.id))
const selectedModels = ref<Record<number, string>>({})
const artwork = ref<Record<string, string>>({})
const artworkLoading = ref<Record<string, boolean>>({})
const artworkFailed = ref<Record<string, boolean>>({})
const models = (groupId: number) => [...new Set((reports.value[groupId]?.events || []).map(event => event.model_id))]
const visibleEvents = (groupId: number) => (reports.value[groupId]?.events || [])
  .filter(event => !selectedModels.value[groupId] || event.model_id === selectedModels.value[groupId]).slice(0, 10)
const eventKey = (groupId: number, eventId: string | number) => `${groupId}:${eventId}`
const verdict = (event: ExternalQualityEvent) => event.status === 'completed'
  ? text('生成完成 · 待对比', 'Generated · awaiting review')
  : event.status === 'success' ? text('正常', 'Passed') : text('降智', 'Degraded')
async function loadArtwork(groupId: number, event: ExternalQualityEvent) {
  const key = eventKey(groupId, event.id)
  if (!alive || artwork.value[key] || artworkLoading.value[key]) return
  if (event.response_text) { artwork.value[key] = event.response_text; return }
  if (!event.has_artwork) return
  artworkLoading.value[key] = true
  artworkFailed.value[key] = false
  try {
    const result = await getExternalQualityArtwork(groupId, event.id, controller?.signal)
    if (alive) artwork.value[key] = result.response_text
  } catch { if (alive) artworkFailed.value[key] = true }
  finally { if (alive) artworkLoading.value[key] = false }
}
function onToggle(event: Event, groupId: number, result: ExternalQualityEvent) {
  if ((event.target as HTMLDetailsElement).open) void loadArtwork(groupId, result)
}
const date = (value?: string | null) => value ? new Date(value).toLocaleString(locale.value) : '—'
async function refresh() {
  if (document.hidden || loading.value) return
  controller?.abort()
  const current = new AbortController(); controller = current
  loading.value = true; failed.value = false
  try {
    groups.value = await userGroupsAPI.getAvailable()
    if (!alive || current.signal.aborted) return
    const results = await Promise.allSettled(eligible.value.map(group => getExternalQuality(group.id, current.signal)))
    if (!alive || current.signal.aborted) return
    const next = { ...reports.value }
    results.forEach((result, index) => {
      if (result.status === 'fulfilled') next[eligible.value[index].id] = result.value
      else failed.value = true
    })
    const allowed = new Set(eligible.value.map(group => group.id))
    reports.value = Object.fromEntries(Object.entries(next).filter(([id]) => allowed.has(Number(id))))
    const retained = new Set(Object.values(reports.value).flatMap(report => report.events.map(event => eventKey(report.group_id, event.id))))
    artwork.value = Object.fromEntries(Object.entries(artwork.value).filter(([key]) => retained.has(key)))
    artworkFailed.value = Object.fromEntries(Object.entries(artworkFailed.value).filter(([key]) => retained.has(key)))
  } catch { if (alive && !current.signal.aborted) failed.value = true }
  finally { if (alive && !current.signal.aborted) loading.value = false }
}
onMounted(() => { void refresh(); timer = setInterval(() => { void refresh() }, 60000) })
onBeforeUnmount(() => { alive = false; controller?.abort(); clearInterval(timer) })
</script>
<template>
  <section class="quality-results" :aria-busy="loading" aria-labelledby="quality-results-title">
    <header><div><h2 id="quality-results-title">{{ text('模型降智检测', 'Model quality checks') }}</h2><p>{{ text('查看模型测试产物，对比生成效果。', 'Compare recent model test outputs.') }}</p></div><button type="button" class="btn btn-secondary" :disabled="loading" @click="refresh">{{ loading ? text('刷新中…', 'Refreshing…') : text('刷新结果', 'Refresh results') }}</button></header>
    <p v-if="failed" role="status" class="quality-muted">{{ text('部分结果暂时无法更新，已保留上次记录。', 'Some results could not be refreshed. Previous records are retained.') }}</p>
    <p v-if="!eligible.length && !loading" class="quality-muted">{{ text('暂无可查看的测试分组。', 'No test groups are currently available.') }}</p>
    <div class="quality-grid">
      <article v-for="group in eligible" :key="group.id" class="quality-card">
        <div class="quality-heading"><h3>{{ group.name }}</h3><span v-if="reports[group.id]?.status === 'stale'" class="quality-stale">{{ text('待更新', 'Awaiting update') }}</span></div>
        <p class="quality-model">{{ reports[group.id]?.model || '—' }}</p>
        <p class="quality-muted">{{ text('最近检测 ', 'Last tested ') }}{{ date(reports[group.id]?.last_success_at || reports[group.id]?.updated_at) }}</p>
        <div v-if="models(group.id).length" class="quality-model-tabs" role="group" :aria-label="text('按模型查看', 'Filter by model')">
          <button type="button" :aria-pressed="!selectedModels[group.id]" @click="selectedModels[group.id] = ''">{{ text('全部', 'All') }}</button>
          <button v-for="model in models(group.id)" :key="model" type="button" :aria-pressed="selectedModels[group.id] === model" @click="selectedModels[group.id] = model">{{ model }}</button>
        </div>
        <p class="quality-muted">{{ text('产物供人工对比；生成完成不代表质量通过。', 'Outputs are for manual comparison; completion is not a quality verdict.') }}</p>
        <p v-if="!reports[group.id]?.events.length" class="quality-empty">{{ text('暂无测试结果', 'No test results yet') }}</p>
        <details v-for="event in visibleEvents(group.id)" :key="event.id" class="quality-event" @toggle="onToggle($event, group.id, event)">
          <summary><span class="quality-verdict" :data-status="event.status">{{ verdict(event) }}</span><strong v-if="(event.score ?? event.audit_summary?.score) != null">{{ event.score ?? event.audit_summary?.score }} / 100</strong><time>{{ date(event.created_at) }}</time></summary>
          <p v-if="event.model_id" class="quality-model">{{ event.model_id }}<span v-if="event.response_model && event.response_model !== event.model_id"> → {{ event.response_model }}</span><span v-if="event.duration_ms"> · {{ (event.duration_ms / 1000).toFixed(1) }} s</span></p>
          <dl v-if="event.audit_summary?.sections?.length" class="quality-sections"><div v-for="section in event.audit_summary.sections" :key="section.name"><dt>{{ section.name }}</dt><dd>{{ section.score ?? section.status ?? '—' }}</dd></div></dl>
          <p v-if="artworkLoading[eventKey(group.id, event.id)]" class="quality-muted" role="status">{{ text('正在加载测试产物…', 'Loading test output…') }}</p>
          <button v-else-if="artworkFailed[eventKey(group.id, event.id)]" type="button" class="btn btn-secondary" @click="loadArtwork(group.id, event)">{{ text('加载失败，点击重试', 'Failed to load. Retry') }}</button>
          <iframe v-else-if="qualityArtifactDocument(artwork[eventKey(group.id, event.id)])" class="quality-artifact" sandbox="" referrerpolicy="no-referrer" :title="text('测试产物', 'Test output')" :srcdoc="qualityArtifactDocument(artwork[eventKey(group.id, event.id)]) || ''" />
          <pre v-else-if="artwork[eventKey(group.id, event.id)]" class="quality-output">{{ artwork[eventKey(group.id, event.id)] }}</pre>
          <p v-else class="quality-muted">{{ text('本条记录没有测试产物。', 'No test output is available for this record.') }}</p>
        </details>
      </article>
    </div>
  </section>
</template>
<style scoped>
.quality-results { color:rgb(var(--ink)); }
.quality-results>header,.quality-heading { display:flex; align-items:flex-start; justify-content:space-between; gap:16px; }
.quality-results h2 { margin:0; font-size:18px; font-weight:600; }
.quality-results header p,.quality-muted { margin:6px 0; color:rgb(var(--ink-3)); font-size:12px; }
.quality-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(min(100%,320px),1fr)); gap:16px; margin-top:16px; }
.quality-card { min-width:0; padding:20px; border:1px solid rgb(var(--line-strong)); border-radius:14px; background:rgb(var(--surface)); }
.quality-heading h3 { margin:0; font-size:14px; font-weight:600; }
.quality-model { overflow-wrap:anywhere; color:rgb(var(--ink-2)); font-size:12px; }
.quality-empty { margin:24px 0; text-align:center; color:rgb(var(--ink-3)); font-size:13px; }
.quality-stale { flex:none; color:rgb(var(--text-warn)); font-size:12px; }
.quality-event { margin-top:12px; border-top:1px solid rgb(var(--line)); }
.quality-event summary { display:flex; flex-wrap:wrap; align-items:center; gap:10px; padding:12px 0; cursor:pointer; font-size:12px; }
.quality-event time { margin-left:auto; color:rgb(var(--ink-3)); }
.quality-verdict { padding:3px 8px; border-radius:999px; color:rgb(var(--text-ok)); background:rgb(var(--surface-2)); }
.quality-verdict[data-status=completed] { color:rgb(var(--ink-2)); }
.quality-model-tabs { display:flex; flex-wrap:wrap; gap:6px; margin:12px 0; }
.quality-model-tabs button { padding:4px 9px; border-radius:8px; border:1px solid rgb(var(--line)); font-size:12px; color:rgb(var(--ink-2)); }
.quality-model-tabs button[aria-pressed=true] { background:rgb(var(--surface-2)); border-color:rgb(var(--line-strong)); color:rgb(var(--ink)); }
.quality-verdict[data-status=degraded] { color:rgb(var(--text-warn)); }
.quality-sections { display:grid; gap:8px; font-size:12px; }
.quality-sections div { display:flex; justify-content:space-between; gap:12px; }
.quality-sections dd { margin:0; }
.quality-output { max-height:420px; overflow:auto; padding:12px; border-radius:8px; background:rgb(var(--surface-2)); font-size:12px; white-space:pre-wrap; overflow-wrap:anywhere; }
.quality-artifact { display:block; width:100%; height:420px; border:1px solid rgb(var(--line)); border-radius:8px; background:white; }
</style>
