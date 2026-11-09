<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getExternalQuality, type ExternalQualityReport } from '@/api/externalQuality'
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
const eligible = computed(() => groups.value.filter(group => [72, 237, 179, 223].includes(group.id)))
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
  } catch { if (alive && !current.signal.aborted) failed.value = true }
  finally { if (alive && !current.signal.aborted) loading.value = false }
}
onMounted(() => { void refresh(); timer = setInterval(() => { void refresh() }, 60000) })
onBeforeUnmount(() => { alive = false; controller?.abort(); clearInterval(timer) })
</script>
<template>
  <section class="quality-results" :aria-busy="loading" aria-labelledby="quality-results-title">
    <header><div><h2 id="quality-results-title">{{ text('模型降智检测', 'Model quality checks') }}</h2><p>{{ text('查看最近测试评分与测试产物。', 'Recent test scores and outputs.') }}</p></div><button type="button" class="btn btn-secondary" :disabled="loading" @click="refresh">{{ loading ? text('刷新中…', 'Refreshing…') : text('刷新结果', 'Refresh results') }}</button></header>
    <p v-if="failed" role="status" class="quality-muted">{{ text('部分结果暂时无法更新，已保留上次记录。', 'Some results could not be refreshed. Previous records are retained.') }}</p>
    <p v-if="!eligible.length && !loading" class="quality-muted">{{ text('暂无可查看的测试分组。', 'No test groups are currently available.') }}</p>
    <div class="quality-grid">
      <article v-for="group in eligible" :key="group.id" class="quality-card">
        <div class="quality-heading"><h3>{{ group.name }}</h3><span v-if="reports[group.id]?.status === 'stale'" class="quality-stale">{{ text('待更新', 'Awaiting update') }}</span></div>
        <p class="quality-model">{{ reports[group.id]?.model || ([179,223].includes(group.id) ? 'deepseek-v4.1-flash' : '—') }}</p>
        <p class="quality-muted">{{ text('更新于 ', 'Updated ') }}{{ date(reports[group.id]?.last_success_at || reports[group.id]?.updated_at) }}</p>
        <p v-if="!reports[group.id]?.events.length" class="quality-empty">{{ text('暂无测试结果', 'No test results yet') }}</p>
        <details v-for="event in reports[group.id]?.events || []" :key="event.id" class="quality-event">
          <summary><span class="quality-verdict" :data-status="event.status">{{ event.status === 'success' ? text('正常', 'Passed') : text('降智', 'Degraded') }}</span><strong v-if="(event.score ?? event.audit_summary?.score) != null">{{ event.score ?? event.audit_summary?.score }} / 100</strong><time>{{ date(event.created_at) }}</time></summary>
          <p v-if="event.model_id" class="quality-model">{{ event.model_id }}</p>
          <dl v-if="event.audit_summary?.sections?.length" class="quality-sections"><div v-for="section in event.audit_summary.sections" :key="section.name"><dt>{{ section.name }}</dt><dd>{{ section.score ?? section.status ?? '—' }}</dd></div></dl>
          <iframe v-if="qualityArtifactDocument(event.response_text)" class="quality-artifact" sandbox="" referrerpolicy="no-referrer" :title="text('测试产物', 'Test output')" :srcdoc="qualityArtifactDocument(event.response_text) || ''" />
          <pre v-else-if="event.response_text" class="quality-output">{{ event.response_text }}</pre>
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
.quality-verdict[data-status=degraded] { color:rgb(var(--text-warn)); }
.quality-sections { display:grid; gap:8px; font-size:12px; }
.quality-sections div { display:flex; justify-content:space-between; gap:12px; }
.quality-sections dd { margin:0; }
.quality-output { max-height:420px; overflow:auto; padding:12px; border-radius:8px; background:rgb(var(--surface-2)); font-size:12px; white-space:pre-wrap; overflow-wrap:anywhere; }
.quality-artifact { display:block; width:100%; height:420px; border:1px solid rgb(var(--line)); border-radius:8px; background:white; }
</style>
