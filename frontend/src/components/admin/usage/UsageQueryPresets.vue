<template>
  <div class="flex flex-wrap items-center gap-2">
    <button v-for="days in [1, 7, 30, 90]" :key="days" type="button" class="btn btn-ghost btn-sm" @click="setRange(days)">{{ days === 1 ? t('dates.today') : `${days}d` }}</button>
    <span class="mx-1 h-4 border-l border-gray-200 dark:border-dark-700" aria-hidden="true" />
    <SavedFilters :items="savedFilters" @save="saveCurrent" @apply="applySaved" @remove="remove" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { usePersistedTableQuery, useSavedTableFilters } from '@/composables/useTablePreferences'
import { formatDateLocalInput } from '@/utils/format'
import SavedFilters from '@/components/common/SavedFilters.vue'
const props = defineProps<{ routeId: string; startDate: string; endDate: string; filters?: object; queryState?: object }>()
const emit = defineEmits<{ change: [range: { startDate: string; endDate: string; preset: string | null }]; filters: [value: Record<string, unknown>]; restore: [value: Record<string, unknown>] }>()
const { t } = useI18n()
const route = useRoute()
const filterKeys = ['start_date', 'end_date', 'request_type', 'billing_type', 'billing_mode', 'native_compaction_v2', 'model', 'group_id', 'account_id', 'user_id', 'api_key_id', 'stream', 'upstream_model_mismatch', 'sort_by', 'sort_order', 'page', 'page_size', 'granularity', 'error_phase', 'error_category', 'status_code', 'error_model', 'error_api_key_id', 'error_page', 'error_page_size', 'error_sort_by', 'error_sort_order']
const state = reactive<Record<string, unknown>>({})
const sync = () => {
  for (const key of Object.keys(state)) delete state[key]
  for (const key of filterKeys) state[key] = null
  Object.assign(state, props.filters, props.queryState, { start_date: props.startDate, end_date: props.endDate })
  for (const key of filterKeys) if (state[key] === undefined) state[key] = null
}
sync()
const query = usePersistedTableQuery({ routeId: props.routeId, tableId: 'analytics', filters: state, filterKeys })
const { savedFilters, save, apply, remove } = useSavedTableFilters<Record<string, unknown>>({ routeId: props.routeId, tableId: 'analytics', filterKeys })
function activate(value: Record<string, unknown>) {
  const start = value.start_date
  const end = value.end_date
  if (typeof start !== 'string' || typeof end !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(start) || !/^\d{4}-\d{2}-\d{2}$/.test(end) || start > end) return
  const { start_date: _startDate, end_date: _endDate, sort_by: _sortBy, sort_order: _sortOrder, page: _page, page_size: _pageSize, granularity: _granularity, ...filters } = value
  if (props.queryState) emit('restore', value)
  else emit('filters', filters)
  emit('change', { startDate: start, endDate: end, preset: null })
}
function setRange(days: number) {
  const end = new Date()
  const start = new Date(end)
  start.setDate(start.getDate() - days + 1)
  emit('change', { startDate: formatDateLocalInput(start), endDate: formatDateLocalInput(end), preset: null })
}
function saveCurrent(name: string) { sync(); save(name, state) }
function applySaved(id: string) { const value = apply(id); if (value) activate(value) }
onMounted(() => {
  if (query.restore()) {
    if (props.queryState) emit('restore', { ...state })
    else if (!route.query.start_date && !route.query.end_date) activate(state)
  }
})
watch(() => [props.startDate, props.endDate, props.filters, props.queryState], () => { sync(); query.persist() }, { deep: true })
</script>
