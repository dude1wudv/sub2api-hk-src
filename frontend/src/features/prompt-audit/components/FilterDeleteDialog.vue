<template>
  <BaseDialog :show="show" :title="t('admin.promptAudit.events.filterDeleteDialogTitle')" width="wide" @close="$emit('close')">
    <div class="space-y-5 text-sm">
      <p class="text-[var(--atelier-ink-secondary)]">{{ t('admin.promptAudit.events.filterDeleteDialogDesc') }}</p>

      <fieldset>
        <legend class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('admin.promptAudit.events.filterTimeRange') }}</legend>
        <div class="mt-2 flex flex-wrap gap-2" role="radiogroup" :aria-label="t('admin.promptAudit.events.filterTimeRange')">
          <label
            v-for="option in DELETE_RANGE_PRESETS"
            :key="option.id"
            class="cursor-pointer"
          >
            <input v-model="preset" type="radio" name="prompt-delete-range" :value="option.id" class="peer sr-only" :data-test="`range-preset-${option.id}`" @change="criteriaChanged" />
            <span class="inline-flex items-center rounded-lg border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] px-3 py-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)] transition-colors peer-checked:border-[var(--atelier-err)] peer-checked:bg-[var(--atelier-err-bg)] peer-checked:text-[var(--atelier-err)] peer-focus-visible:ring-2 peer-focus-visible:ring-red-500/30">
              {{ t(`admin.promptAudit.events.timePresets.${option.id}`) }}
            </span>
          </label>
        </div>
        <p class="mt-2 text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.promptAudit.events.filterTimeRangeHint') }}</p>
        <div v-if="preset === 'custom'" class="mt-3 grid gap-3 sm:grid-cols-2" data-test="custom-range">
          <label class="text-xs text-[var(--atelier-ink-secondary)]">
            <span class="font-medium">{{ t('admin.promptAudit.events.startAt') }}</span>
            <input v-model="local.start_at" type="datetime-local" class="atelier-input mt-1" :aria-label="t('admin.promptAudit.events.startAt')" @change="criteriaChanged" />
          </label>
          <label class="text-xs text-[var(--atelier-ink-secondary)]">
            <span class="font-medium">{{ t('admin.promptAudit.events.endAt') }}</span>
            <input v-model="local.end_at" type="datetime-local" class="atelier-input mt-1" :aria-label="t('admin.promptAudit.events.endAt')" @change="criteriaChanged" />
          </label>
          <p v-if="!canPreview" class="text-xs text-[var(--atelier-err)] sm:col-span-2">{{ t('admin.promptAudit.events.customRangeInvalid') }}</p>
        </div>
      </fieldset>

      <div class="grid gap-3 sm:grid-cols-2">
        <div class="text-xs text-[var(--atelier-ink-secondary)]" data-test="delete-decision">
          <span class="mb-1 block font-medium">{{ t('admin.promptAudit.events.decision') }}</span>
          <UiSelect
            :model-value="local.decision"
            :options="decisionOptions"
            :aria-label="t('admin.promptAudit.events.decision')"
            @update:model-value="local.decision = String($event ?? ''); criteriaChanged()"
          />
        </div>
        <div class="text-xs text-[var(--atelier-ink-secondary)]" data-test="delete-risk">
          <span class="mb-1 block font-medium">{{ t('admin.promptAudit.events.risk') }}</span>
          <UiSelect
            :model-value="local.risk_level"
            :options="riskOptions"
            :aria-label="t('admin.promptAudit.events.risk')"
            @update:model-value="local.risk_level = String($event ?? ''); criteriaChanged()"
          />
        </div>
      </div>

      <details class="rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/40 px-4 py-3" data-test="more-conditions">
        <summary class="cursor-pointer select-none text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('admin.promptAudit.events.moreConditions') }}</summary>
        <div class="mt-3 grid gap-3 sm:grid-cols-2">
          <label class="text-xs text-[var(--atelier-ink-secondary)]">
            <span>{{ t('admin.promptAudit.events.endpoint') }}</span>
            <input v-model="local.endpoint" type="text" class="atelier-input mt-1" :aria-label="t('admin.promptAudit.events.endpoint')" @input="criteriaChanged" />
          </label>
          <label class="text-xs text-[var(--atelier-ink-secondary)]">
            <span>{{ t('admin.promptAudit.events.keyword') }}</span>
            <input v-model="local.keyword" type="text" class="atelier-input mt-1" :aria-label="t('admin.promptAudit.events.keyword')" @input="criteriaChanged" />
          </label>
          <label class="text-xs text-[var(--atelier-ink-secondary)]">
            <span>{{ t('admin.promptAudit.events.groupId') }}</span>
            <input v-model="local.group_id" type="number" class="atelier-input mt-1" :aria-label="t('admin.promptAudit.events.groupId')" @input="criteriaChanged" />
          </label>
          <label class="text-xs text-[var(--atelier-ink-secondary)]">
            <span>{{ t('admin.promptAudit.events.userId') }}</span>
            <input v-model="local.user_id" type="number" class="atelier-input mt-1" :aria-label="t('admin.promptAudit.events.userId')" @input="criteriaChanged" />
          </label>
        </div>
      </details>

      <div v-if="preview" class="rounded-xl border border-[var(--atelier-err)]/30 bg-[var(--atelier-err-bg)] px-4 py-3" data-test="delete-preview-result">
        <p class="text-sm font-semibold text-[var(--atelier-err)]">{{ t('admin.promptAudit.events.filterDeleteCount', { count: preview.matched_count }) }}</p>
        <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs text-[var(--atelier-ink-secondary)]">
          <dt>{{ t('admin.promptAudit.events.snapshotMax') }}</dt>
          <dd class="font-mono tabular-nums">{{ preview.snapshot_max_id }}</dd>
          <dt>Filter SHA-256</dt>
          <dd class="break-all font-mono">{{ preview.filter_hash }}</dd>
          <dt>{{ t('admin.promptAudit.events.expiresAt') }}</dt>
          <dd class="font-mono tabular-nums">{{ formatDate(preview.expires_at) }}</dd>
        </dl>
        <p class="mt-2 rounded-lg bg-[var(--atelier-warn-bg)] px-3 py-2 text-xs text-[var(--atelier-warn)]">{{ t('admin.promptAudit.events.filterDeleteWarning') }}</p>
      </div>
      <p v-else class="rounded-xl border border-dashed border-[var(--atelier-border-strong)] px-4 py-3 text-xs text-[var(--atelier-ink-muted)]" data-test="delete-preview-empty">
        {{ t('admin.promptAudit.events.filterDeleteNeedPreview') }}
      </p>
    </div>

    <template #footer>
      <div class="flex flex-wrap items-center justify-end gap-3">
        <p v-if="confirmDisabledReason" class="mr-auto text-xs text-[var(--atelier-ink-muted)]" data-test="confirm-disabled-reason">
          {{ t(confirmDisabledReason) }}
        </p>
        <UiButton variant="secondary" @click="$emit('close')">{{ t('common.cancel') }}</UiButton>
        <UiButton variant="secondary" :disabled="!canPreview || previewing || deleting" :loading="previewing" data-test="run-delete-preview" @click="requestPreview">
          {{ previewing ? t('admin.promptAudit.events.filterDeletePreviewing') : t('admin.promptAudit.events.filterDeletePreviewAction') }}
        </UiButton>
        <UiButton
          variant="danger"
          :disabled="confirmDisabled"
          :loading="deleting"
          :title="confirmDisabledReason ? t(confirmDisabledReason) : undefined"
          data-test="confirm-filter-delete"
          @click="requestConfirm"
        >
          {{ deleting ? t('common.submitting') : t('admin.promptAudit.events.confirmFilterDelete') }}
        </UiButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { UiButton, UiSelect, type UiSelectOption } from '@/ui'
import type { PromptDeletePreview, PromptEventFilters } from '../types'
import {
  DELETE_RANGE_PRESETS,
  cloneData,
  emptyEventFilters,
  hasExplicitDeleteRange,
  resolveDeleteRangeFilters,
  type DeleteRangePreset,
} from '../viewModel'

const props = defineProps<{
  show: boolean
  initialFilters: PromptEventFilters
  preview: PromptDeletePreview | null
  previewing: boolean
  deleting: boolean
}>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'preview', value: PromptEventFilters): void
  (event: 'confirm', value: PromptEventFilters): void
  (event: 'criteria-change'): void
}>()
const { t, locale } = useI18n()

const preset = ref<DeleteRangePreset>('7d')
const local = reactive<PromptEventFilters>(emptyEventFilters())

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

watch(
  () => props.show,
  (visible) => {
    if (!visible) return
    const initial = cloneData(props.initialFilters)
    // Only inherit an explicit list-filter range; otherwise default to the
    // seven-day preset so a careless click can never target everything.
    preset.value = hasExplicitDeleteRange(initial) ? 'custom' : '7d'
    Object.assign(local, initial)
  },
  { immediate: true },
)

const canPreview = computed(() => preset.value !== 'custom' || hasExplicitDeleteRange(local))

// One-click flow: a valid criteria selection is enough to confirm — the parent
// mints the server-side confirmation token on the fly. The button stays
// disabled only when the range is invalid, work is in flight, or a fresh
// preview already proved there is nothing to delete.
const confirmDisabled = computed(
  () => !canPreview.value || props.previewing || props.deleting || (props.preview !== null && props.preview.matched_count === 0),
)
const confirmDisabledReason = computed(() => {
  if (props.previewing || props.deleting) return ''
  if (!canPreview.value) return 'admin.promptAudit.events.filterDeleteConfirmInvalidRange'
  if (props.preview && props.preview.matched_count === 0) return 'admin.promptAudit.events.filterDeleteConfirmNoMatches'
  return ''
})

function criteriaChanged() {
  emit('criteria-change')
}
function requestPreview() {
  if (!canPreview.value) return
  emit('preview', resolveDeleteRangeFilters(local, preset.value))
}
function requestConfirm() {
  if (confirmDisabled.value) return
  emit('confirm', resolveDeleteRangeFilters(local, preset.value))
}
function formatDate(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value))
}
</script>
