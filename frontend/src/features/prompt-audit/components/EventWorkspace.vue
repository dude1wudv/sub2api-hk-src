<template>
  <section aria-labelledby="prompt-events-title" class="py-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 id="prompt-events-title" class="text-base font-semibold text-[var(--atelier-ink)]">{{ t('admin.promptAudit.events.title') }}</h2>
        <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">{{ t('admin.promptAudit.events.description') }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <UiButton variant="secondary" size="sm" :disabled="selectedIds.length === 0" @click="$emit('batch-delete')">
          {{ t('admin.promptAudit.events.deleteSelected', { count: selectedIds.length }) }}
        </UiButton>
        <UiButton variant="danger" size="sm" data-test="filter-delete" @click="$emit('preview-delete')">
          {{ t('admin.promptAudit.events.deleteByFilter') }}
        </UiButton>
      </div>
    </div>

    <form class="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-5" @submit.prevent="applyFilters">
      <div class="text-xs text-[var(--atelier-ink-secondary)]">
        <span class="mb-1 block font-medium">{{ t('admin.promptAudit.events.decision') }}</span>
        <UiSelect
          :model-value="localFilters.decision"
          :options="decisionOptions"
          :aria-label="t('admin.promptAudit.events.decision')"
          size="sm"
          @update:model-value="localFilters.decision = String($event ?? ''); filtersChanged()"
        />
      </div>
      <div class="text-xs text-[var(--atelier-ink-secondary)]">
        <span class="mb-1 block font-medium">{{ t('admin.promptAudit.events.risk') }}</span>
        <UiSelect
          :model-value="localFilters.risk_level"
          :options="riskOptions"
          :aria-label="t('admin.promptAudit.events.risk')"
          size="sm"
          @update:model-value="localFilters.risk_level = String($event ?? ''); filtersChanged()"
        />
      </div>
      <FilterInput v-model="localFilters.endpoint" :label="t('admin.promptAudit.events.endpoint')" @change="filtersChanged" />
      <FilterInput v-model="localFilters.group_id" :label="t('admin.promptAudit.events.groupId')" type="number" @change="filtersChanged" />
      <FilterInput v-model="localFilters.user_id" :label="t('admin.promptAudit.events.userId')" type="number" @change="filtersChanged" />
      <FilterInput v-model="localFilters.api_key_id" :label="t('admin.promptAudit.events.apiKeyId')" type="number" @change="filtersChanged" />
      <FilterInput v-model="localFilters.request_id" :label="t('admin.promptAudit.events.requestId')" @change="filtersChanged" />
      <FilterInput v-model="localFilters.prompt_hash" :label="t('admin.promptAudit.events.promptHash')" @change="filtersChanged" />
      <FilterInput v-model="localFilters.keyword" :label="t('admin.promptAudit.events.keyword')" @change="filtersChanged" />
      <label class="text-xs text-[var(--atelier-ink-secondary)]">
        <span class="font-medium">{{ t('admin.promptAudit.events.startAt') }}</span>
        <input v-model="localFilters.start_at" type="datetime-local" class="atelier-input mt-1 !min-h-8 !py-1 !text-xs" :aria-label="t('admin.promptAudit.events.startAt')" @change="filtersChanged" />
      </label>
      <label class="text-xs text-[var(--atelier-ink-secondary)]">
        <span class="font-medium">{{ t('admin.promptAudit.events.endAt') }}</span>
        <input v-model="localFilters.end_at" type="datetime-local" class="atelier-input mt-1 !min-h-8 !py-1 !text-xs" :aria-label="t('admin.promptAudit.events.endAt')" @change="filtersChanged" />
      </label>
      <div class="flex items-end gap-2 sm:col-span-2">
        <UiButton type="submit" variant="primary" size="sm">{{ t('common.search') }}</UiButton>
        <UiButton variant="ghost" size="sm" @click="resetFilters">{{ t('common.reset') }}</UiButton>
      </div>
    </form>
    <div v-if="error" role="alert" class="mt-4 rounded-xl border border-[var(--atelier-err)]/30 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]">{{ error }}</div>
    <div class="mt-5 overflow-x-auto rounded-xl border border-[var(--atelier-border)]">
      <div v-if="mobile" class="mobile-records">
        <label class="flex min-h-11 items-center gap-3 px-4"><input type="checkbox" :checked="allSelected" @change="toggleAll" />{{ t('admin.promptAudit.events.selectAll') }}</label>
        <div v-if="loading" class="mobile-record-state">{{ t('common.loading') }}</div>
        <div v-else-if="events.length === 0" class="mobile-record-state">{{ t('admin.promptAudit.events.empty') }}</div>
        <dl v-for="event in events" v-else :key="event.id" :data-test="`event-${event.id}`" class="mobile-record">
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.events.selectEvent', { id: event.id }) }}</dt><dd><label class="flex min-h-11 min-w-11 items-center"><input type="checkbox" :checked="selectedIds.includes(event.id)" :aria-label="t('admin.promptAudit.events.selectEvent', { id: event.id })" @change="toggleOne(event.id)" /></label></dd></div>
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.events.time') }}</dt><dd class="font-mono tabular-nums">{{ formatDate(event.created_at) }}</dd></div>
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.events.identity') }}</dt><dd><CopyLine :label="t('admin.promptAudit.events.user')" :value="event.snapshot.username" />
            <CopyLine :label="t('admin.promptAudit.events.email')" :value="event.snapshot.user_email" />
            <CopyLine :label="t('admin.promptAudit.events.apiKey')" :value="event.snapshot.api_key_name" /></dd></div>
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.events.group') }}</dt><dd>{{ event.snapshot.group_name || '—' }}</dd></div>
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.events.route') }}</dt><dd><p class="font-medium text-[var(--atelier-ink)]">{{ event.snapshot.endpoint }}</p>
            <p class="mt-1 text-xs text-[var(--atelier-ink-muted)]">{{ event.snapshot.model }} · {{ event.snapshot.protocol }} · {{ event.snapshot.stage || 'http' }}</p></dd></div>
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.events.result') }}</dt><dd><span class="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-medium" :class="decisionClass(event.decision)"><UiStatusDot :tone="decisionTone(event.decision)" />{{ formatDecisionRisk(event.decision, event.risk_level) }}</span>
            <p class="mt-2 text-xs text-[var(--atelier-ink-muted)]">{{ formatCategories(event.categories) }}</p></dd></div>
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.events.preview') }}</dt><dd><p class="line-clamp-2 break-words text-[var(--atelier-ink-secondary)]">{{ event.snapshot.redacted_preview || '—' }}</p></dd></div>
          <div class="mobile-record-field"><dt>{{ t('admin.promptAudit.common.actions') }}</dt><dd class="flex gap-1"><UiButton variant="ghost" size="sm" @click="$emit('view', event.id)">{{ t('common.view') }}</UiButton>
            <UiButton variant="ghost" size="sm" class="!text-[var(--atelier-err)]" @click="$emit('delete', event.id)">{{ t('common.delete') }}</UiButton></dd></div>
        </dl>
      </div>
      <table v-else class="atelier-table min-w-[1120px]">
        <thead>
          <tr>
            <th class="w-10"><input type="checkbox" :checked="allSelected" :aria-label="t('admin.promptAudit.events.selectAll')" @change="toggleAll" /></th>
            <th>{{ t('admin.promptAudit.events.time') }}</th>
            <th>{{ t('admin.promptAudit.events.identity') }}</th>
            <th>{{ t('admin.promptAudit.events.group') }}</th>
            <th>{{ t('admin.promptAudit.events.route') }}</th>
            <th>{{ t('admin.promptAudit.events.result') }}</th>
            <th>{{ t('admin.promptAudit.events.preview') }}</th>
            <th class="!text-right">{{ t('admin.promptAudit.common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="8" class="px-4 py-12 text-center text-[var(--atelier-ink-muted)]" aria-busy="true">{{ t('common.loading') }}</td></tr>
          <tr v-else-if="events.length === 0"><td colspan="8" class="px-4 py-12 text-center text-[var(--atelier-ink-muted)]">{{ t('admin.promptAudit.events.empty') }}</td></tr>
          <tr v-for="event in events" v-else :key="event.id" :data-test="`event-${event.id}`" class="align-top">
            <td><input type="checkbox" :checked="selectedIds.includes(event.id)" :aria-label="t('admin.promptAudit.events.selectEvent', { id: event.id })" @change="toggleOne(event.id)" /></td>
            <td class="whitespace-nowrap font-mono text-xs tabular-nums text-[var(--atelier-ink-secondary)]">{{ formatDate(event.created_at) }}</td>
            <td>
              <CopyLine :label="t('admin.promptAudit.events.user')" :value="event.snapshot.username" />
              <CopyLine :label="t('admin.promptAudit.events.email')" :value="event.snapshot.user_email" />
              <CopyLine :label="t('admin.promptAudit.events.apiKey')" :value="event.snapshot.api_key_name" />
            </td>
            <td class="text-[var(--atelier-ink-secondary)]">{{ event.snapshot.group_name || '—' }}</td>
            <td>
              <p class="font-medium text-[var(--atelier-ink)]">{{ event.snapshot.endpoint }}</p>
              <p class="mt-1 text-xs text-[var(--atelier-ink-muted)]">{{ event.snapshot.model }} · {{ event.snapshot.protocol }} · {{ event.snapshot.stage || 'http' }}</p>
            </td>
            <td>
              <span class="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-medium" :class="decisionClass(event.decision)">
                <UiStatusDot :tone="decisionTone(event.decision)" />
                {{ formatDecisionRisk(event.decision, event.risk_level) }}
              </span>
              <p class="mt-2 max-w-48 truncate text-xs text-[var(--atelier-ink-muted)]" :title="formatCategories(event.categories)">{{ formatCategories(event.categories) }}</p>
            </td>
            <td class="max-w-xs"><p class="line-clamp-2 break-words text-[var(--atelier-ink-secondary)]">{{ event.snapshot.redacted_preview || '—' }}</p></td>
            <td class="whitespace-nowrap text-right">
              <UiButton variant="ghost" size="sm" @click="$emit('view', event.id)">{{ t('common.view') }}</UiButton>
              <UiButton variant="ghost" size="sm" class="!text-[var(--atelier-err)]" @click="$emit('delete', event.id)">{{ t('common.delete') }}</UiButton>
            </td>
          </tr>
        </tbody>
      </table>
      <Pagination :total="total" :page="page" :page-size="pageSize" @update:page="$emit('page', $event)" @update:page-size="$emit('page-size', $event)" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
import { computed, defineComponent, h, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Pagination from '@/components/common/Pagination.vue'
import { UiButton, UiSelect, UiStatusDot, type UiSelectOption } from '@/ui'
import type { PromptAuditEvent, PromptEventFilters } from '../types'
import { cloneData, emptyEventFilters, SCANNER_CATALOG } from '../viewModel'

const props = defineProps<{
  events: PromptAuditEvent[]; total: number; page: number; pageSize: number
  filters: PromptEventFilters; selectedIds: number[]; loading: boolean; error: string
}>()
const emit = defineEmits<{
  (event: 'filters-change', value: PromptEventFilters): void
  (event: 'search', value: PromptEventFilters): void
  (event: 'selection', value: number[]): void
  (event: 'page', value: number): void
  (event: 'page-size', value: number): void
  (event: 'view', id: number): void
  (event: 'delete', id: number): void
  (event: 'batch-delete'): void
  (event: 'preview-delete'): void
}>()
const { t, locale } = useI18n()
const localFilters = reactive<PromptEventFilters>(cloneData(props.filters))
watch(() => props.filters, (value) => Object.assign(localFilters, cloneData(value)), { deep: true })
const allSelected = computed(() => props.events.length > 0 && props.events.every((event) => props.selectedIds.includes(event.id)))

const decisionOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: t('common.all') },
  { value: 'pass', label: t('admin.promptAudit.decisions.pass') },
  { value: 'flag', label: t('admin.promptAudit.decisions.flag') },
  { value: 'critical', label: t('admin.promptAudit.decisions.critical') },
])

const riskOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: t('common.all') },
  { value: 'low', label: t('admin.promptAudit.riskLevels.low') },
  { value: 'medium', label: t('admin.promptAudit.riskLevels.medium') },
  { value: 'high', label: t('admin.promptAudit.riskLevels.high') },
  { value: 'critical', label: t('admin.promptAudit.riskLevels.critical') },
])

const FilterInput = defineComponent({
  props: { modelValue: { type: String, required: true }, label: { type: String, required: true }, type: { type: String, default: 'text' } },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit: componentEmit }) {
    return () => h('label', { class: 'text-xs text-gray-600 dark:text-dark-200' }, [
      h('span', componentProps.label),
      h('input', {
        value: componentProps.modelValue, type: componentProps.type, class: 'input mt-1 w-full', 'aria-label': componentProps.label,
        onInput: (event: Event) => componentEmit('update:modelValue', (event.target as HTMLInputElement).value),
        onChange: () => componentEmit('change'),
      }),
    ])
  },
})

const CopyLine = defineComponent({
  props: { label: { type: String, required: true }, value: { type: String, default: '' } },
  setup(componentProps) {
    return () => h('div', { class: ['flex items-center gap-1 text-xs', mobile.value ? 'min-w-0 flex-wrap' : 'max-w-56'] }, [
      h('span', { class: 'w-16 flex-none text-gray-500 dark:text-dark-400' }, componentProps.label),
      h('span', { class: ['min-w-0 flex-1 text-gray-800 dark:text-dark-100', mobile.value ? '[overflow-wrap:anywhere]' : 'truncate'] }, componentProps.value || '—'),
      componentProps.value ? h('button', {
        type: 'button', class: 'text-primary-600 hover:underline', 'aria-label': `${t('common.copy')} ${componentProps.label}`,
        onClick: () => navigator.clipboard?.writeText(componentProps.value),
      }, t('common.copy')) : null,
    ])
  },
})

function filtersChanged() {
  emit('filters-change', cloneData(localFilters))
}
function applyFilters() {
  const value = cloneData(localFilters)
  emit('filters-change', value)
  emit('search', value)
}
function resetFilters() {
  Object.assign(localFilters, emptyEventFilters())
  applyFilters()
}
function toggleOne(id: number) {
  const selected = new Set(props.selectedIds)
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
  emit('selection', [...selected])
}
function toggleAll() {
  emit('selection', allSelected.value ? [] : props.events.map((event) => event.id))
}
function formatDate(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'medium' }).format(new Date(value))
}
function decisionTone(decision: string): 'ok' | 'warn' | 'err' {
  if (decision === 'critical') return 'err'
  if (decision === 'flag') return 'warn'
  return 'ok'
}
function decisionClass(decision: string): string {
  if (decision === 'critical') return 'bg-[var(--atelier-err-bg)] text-[var(--atelier-err)]'
  if (decision === 'flag') return 'bg-[var(--atelier-warn-bg)] text-[var(--atelier-warn)]'
  return 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]'
}
const DECISIONS = new Set(['pass', 'flag', 'critical'])
const RISK_LEVELS = new Set(['low', 'medium', 'high', 'critical'])

function translateDecision(decision: string): string {
  return DECISIONS.has(decision) ? t(`admin.promptAudit.decisions.${decision}`) : decision
}
function translateRiskLevel(riskLevel: string): string {
  return RISK_LEVELS.has(riskLevel) ? t(`admin.promptAudit.riskLevels.${riskLevel}`) : riskLevel
}
function translateCategory(category: string): string {
  return SCANNER_CATALOG.some((scanner) => scanner.id === category)
    ? t(`admin.promptAudit.scanners.${category}`)
    : category
}
function formatDecisionRisk(decision: string, riskLevel: string): string {
  return `${translateDecision(decision)} · ${translateRiskLevel(riskLevel)}`
}
function formatCategories(categories: string[]): string {
  if (!categories.length) return '—'
  return categories.map(translateCategory).join(', ')
}
</script>
