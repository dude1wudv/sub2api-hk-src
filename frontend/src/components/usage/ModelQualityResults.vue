<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { getExternalQuality, getExternalQualityArtwork, type ExternalQualityEvent, type ExternalQualityReport } from '@/api/externalQuality'
import { qualityArtifactPlayableDocument } from '@/utils/qualityArtifact'
const props = defineProps<{ groupId: number; groupName: string }>()
const emit = defineEmits<{ (event: 'close'): void }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const report = ref<ExternalQualityReport | null>(null)
const loading = ref(false), failed = ref(false), artworkLoading = ref(false), artworkFailed = ref(false)
const selectedModel = ref('gpt-6-astra'), selectedId = ref<string | number | null>(null), rawArtwork = ref('')
const preview = ref(true), showSource = ref(false), fit = ref(true)
let reportController: AbortController | undefined, artworkController: AbortController | undefined
let timer: ReturnType<typeof setInterval> | undefined
let alive = true
const availableModels = computed(() => [...new Set(['gpt-6-astra', 'gpt-6.1-sol', ...(report.value?.events || []).map(event => event.model_id)])])
const events = computed(() => (report.value?.events || []).filter(event => event.model_id === selectedModel.value).slice(0, 10))
const selected = computed(() => events.value.find(event => event.id === selectedId.value) || null)
const safeDocument = computed(() => preview.value && !showSource.value ? qualityArtifactPlayableDocument(rawArtwork.value) : null)
const date = (value?: string | null) => value ? new Date(value).toLocaleString(locale.value) : '—'
const duration = (event: ExternalQualityEvent) => event.duration_ms == null ? '—' : `${(event.duration_ms / 1000).toFixed(1)}s`
const verdict = (event: ExternalQualityEvent) => event.status === 'completed' ? text('生成完成 · 待对比', 'Generated · awaiting review') : event.status === 'success' ? text('正常', 'Passed') : text('降智', 'Degraded')
function clearArtwork() { artworkController?.abort(); rawArtwork.value = ''; artworkLoading.value = false; artworkFailed.value = false }
async function loadArtwork() {
  clearArtwork()
  const event = selected.value
  if (!alive || !event || (!preview.value && !showSource.value)) return
  if (event.response_text) { rawArtwork.value = event.response_text; return }
  if (!event.has_artwork) return
  const controller = new AbortController(); artworkController = controller; artworkLoading.value = true
  try {
    const result = await getExternalQualityArtwork(props.groupId, event.id, controller.signal)
    if (alive && !controller.signal.aborted && selectedId.value === event.id) rawArtwork.value = result.response_text
  } catch { if (alive && !controller.signal.aborted) artworkFailed.value = true }
  finally { if (alive && !controller.signal.aborted) artworkLoading.value = false }
}
async function refresh() {
  if (!alive || document.hidden || loading.value || ![72, 237].includes(props.groupId)) return
  reportController?.abort()
  const controller = new AbortController(); reportController = controller; loading.value = true; failed.value = false
  try {
    const result = await getExternalQuality(props.groupId, controller.signal)
    if (!alive || controller.signal.aborted) return
    report.value = result
    if (!events.value.length && result.events.length) selectedModel.value = result.events[0].model_id
    if (!events.value.some(event => event.id === selectedId.value)) selectedId.value = events.value[0]?.id ?? null
  } catch { if (alive && !controller.signal.aborted) failed.value = true }
  finally { if (alive && !controller.signal.aborted) loading.value = false }
}
function close() { alive = false; clearInterval(timer); reportController?.abort(); clearArtwork(); emit('close') }
watch(selectedModel, () => { selectedId.value = events.value[0]?.id ?? null })
watch(selectedId, () => { void loadArtwork() })
watch([preview, showSource], () => { if (!preview.value && !showSource.value) clearArtwork(); else if (!rawArtwork.value) void loadArtwork() })
onMounted(() => { void refresh(); timer = setInterval(() => { void refresh() }, 60000) })
onBeforeUnmount(() => { alive = false; clearInterval(timer); reportController?.abort(); clearArtwork() })
</script>
<template>
  <BaseDialog :show="true" :title="`${text('降智检测', 'Quality check')} · ${groupName}`" width="full" @close="close">
    <div class="quality-workspace">
      <div class="quality-intro"><div><strong>{{ text('鹈鹕测试', 'Pelican test') }}</strong><p>{{ text('每个模型保留最近 10 次结果，仅供人工对比；生成完成不代表质量通过。', 'The latest 10 outputs per model, for manual comparison. Completion is not a quality verdict.') }}</p></div><button type="button" class="quality-control" :disabled="loading" @click="refresh">{{ loading ? text('刷新中…', 'Refreshing…') : text('刷新结果', 'Refresh results') }}</button></div>
      <p v-if="failed" class="quality-notice" role="status">{{ text('暂时无法更新，保留上次结果。', 'Refresh failed. Previous results are retained.') }}</p>
      <p v-if="report?.status === 'stale'" class="quality-notice" role="status">{{ text('最近检测已超过 2 小时，等待新结果。', 'The last test is over 2 hours old. Awaiting new results.') }}</p>
      <div class="quality-layout">
        <aside class="quality-history" :aria-label="text('检测历史', 'Test history')">
          <button v-for="event in events" :key="event.id" type="button" class="quality-history-item" :aria-pressed="selectedId === event.id" @click="selectedId = event.id"><strong>{{ date(event.created_at) }}</strong><span>{{ verdict(event) }}</span><small>{{ event.model_id }} · {{ duration(event) }}</small></button>
          <p v-if="!events.length" class="quality-empty">{{ loading ? text('正在加载…', 'Loading…') : text('暂无测试结果', 'No test results yet') }}</p>
        </aside>
        <main class="quality-stage">
          <p class="quality-dates">{{ text('开始：', 'Started: ') }}{{ date(selected?.created_at) }} · {{ text('结束：', 'Finished: ') }}{{ date(selected?.finished_at) }}</p>
          <div class="quality-model-tabs" role="group" :aria-label="text('切换模型', 'Choose model')"><button v-for="model in availableModels" :key="model" type="button" :aria-pressed="selectedModel === model" @click="selectedModel = model">{{ model }}</button></div>
          <div class="quality-toolbar"><button type="button" class="quality-control" :aria-pressed="fit" @click="fit = !fit">{{ text('适应窗口', 'Fit to window') }}{{ fit ? ' ✓' : '' }}</button><button type="button" class="quality-control" :aria-pressed="preview" @click="preview = !preview; showSource = false">{{ preview ? text('停止预览', 'Stop preview') : text('播放产物', 'Play output') }}</button><button type="button" class="quality-control" :aria-pressed="showSource" @click="showSource = !showSource">{{ showSource ? text('返回预览', 'Back to preview') : text('查看源码', 'View source') }}</button><span v-if="selected" class="quality-duration">{{ duration(selected) }}</span></div>
          <div class="quality-preview" :class="{ 'quality-preview-natural': !fit }">
            <p v-if="artworkLoading" class="quality-empty" role="status">{{ text('正在加载测试产物…', 'Loading test output…') }}</p>
            <button v-else-if="artworkFailed" type="button" class="quality-control quality-retry" @click="loadArtwork">{{ text('加载失败，点击重试', 'Failed to load. Retry') }}</button>
            <pre v-else-if="showSource && rawArtwork" class="quality-source">{{ rawArtwork }}</pre>
            <iframe v-else-if="safeDocument" :key="selectedId ?? ''" class="quality-artifact" sandbox="allow-scripts" referrerpolicy="no-referrer" :title="text('测试产物', 'Test output')" :srcdoc="safeDocument" />
            <pre v-else-if="preview && rawArtwork" class="quality-source">{{ rawArtwork }}</pre>
            <p v-else class="quality-empty">{{ !preview && !showSource ? text('预览已停止', 'Preview stopped') : text('本条记录没有测试产物。', 'No test output is available for this record.') }}</p>
          </div>
        </main>
      </div>
    </div>
  </BaseDialog>
</template>
<style scoped>
.quality-workspace{margin:-4px;color:rgb(var(--ink))}.quality-intro{display:flex;justify-content:space-between;align-items:flex-start;gap:16px;padding:0 2px 16px}.quality-intro strong{font-size:14px}.quality-intro p,.quality-dates{margin:6px 0 0;color:rgb(var(--ink-3));font-size:12px}.quality-control{flex-shrink:0;padding:8px 12px;border:1px solid rgb(var(--line));border-radius:10px;background:rgb(var(--surface));color:rgb(var(--ink-2));font-size:12px;transition:border-color .15s,background .15s}.quality-control:hover{background:rgb(var(--surface-2));border-color:rgb(var(--line-strong))}.quality-control:disabled{opacity:.5}.quality-notice{margin:0 0 12px;font-size:12px;color:rgb(var(--text-warn))}.quality-layout{display:grid;grid-template-columns:270px minmax(0,1fr);min-height:65vh;border-top:1px solid rgb(var(--line))}.quality-history{display:flex;flex-direction:column;gap:10px;padding:16px 16px 16px 0;max-height:72vh;overflow:auto;border-right:1px solid rgb(var(--line))}.quality-history-item{display:flex;flex-direction:column;gap:7px;padding:16px;text-align:left;border:1px solid rgb(var(--line));border-radius:16px;background:rgb(var(--surface));transition:background .15s,border-color .15s}.quality-history-item strong{font-size:13px}.quality-history-item span{color:#168879;font-size:12px}.quality-history-item small{color:rgb(var(--ink-3));font-size:11px}.quality-history-item[aria-pressed=true]{border-color:#33b9a6;background:rgba(20,184,166,.06)}.quality-stage{min-width:0;padding:16px 0 0 18px}.quality-dates{margin:0 0 12px}.quality-model-tabs{display:flex;flex-wrap:wrap;gap:8px;padding-bottom:14px;border-bottom:1px solid rgb(var(--line))}.quality-model-tabs button{padding:10px 14px;border:1px solid rgb(var(--line));border-radius:12px;color:rgb(var(--ink-2));font-size:13px}.quality-model-tabs button[aria-pressed=true]{color:#168879;border-color:#33b9a6;background:rgba(20,184,166,.06)}.quality-toolbar{display:flex;align-items:center;flex-wrap:wrap;gap:8px;padding:12px 0}.quality-duration{margin-left:auto;font-size:12px;color:rgb(var(--ink-3))}.quality-preview{overflow:auto;min-height:480px;height:55vh;border:1px solid rgb(var(--line));border-radius:16px;background:rgb(var(--surface-2))}.quality-artifact{display:block;width:100%;height:100%;border:0;background:white}.quality-preview-natural .quality-artifact{width:1200px;min-width:100%}.quality-source{margin:0;padding:18px;font:12px/1.6 monospace;white-space:pre-wrap;overflow-wrap:anywhere}.quality-empty{margin:48px 16px;text-align:center;font-size:13px;color:rgb(var(--ink-3))}.quality-retry{display:block;margin:48px auto}@media(max-width:760px){.quality-layout{grid-template-columns:1fr}.quality-history{flex-direction:row;overflow-x:auto;max-height:145px;border-right:0;padding:12px 0}.quality-history-item{flex:0 0 215px;padding:12px}.quality-stage{padding:16px 0 0}.quality-preview{min-height:400px;height:55vh}.quality-intro{flex-wrap:wrap}}@media(prefers-reduced-motion:reduce){.quality-control,.quality-history-item{transition:none}}
</style>
