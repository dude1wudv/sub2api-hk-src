<template>
  <UiDrawer
    :open="open"
    :title="zh ? '模型费用估算器' : 'Interactive Cost Estimator'"
    :subtitle="zh ? '按真实分组倍率与缓存命中率估算开销并横向对比' : 'Estimate spend with live group multipliers & prompt cache savings'"
    width="lg"
    @close="emit('close')"
  >
    <div v-if="pricedItems.length === 0" class="py-12 text-center">
      <p class="text-sm font-medium text-[var(--atelier-ink)]">
        {{ zh ? '当前暂无可用 Token 单价数据' : 'No token pricing available' }}
      </p>
      <p class="mt-1 text-xs text-[var(--atelier-ink-muted)]">
        {{ zh ? '请确认当前分组已公开 Token 计费价格。' : 'Check back once token pricing is published for your groups.' }}
      </p>
    </div>

    <div v-else class="space-y-6">
      <!-- Add Model to Compare Selector -->
      <div class="space-y-1.5">
        <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
          {{ zh ? '添加对比模型' : 'Add model to compare' }}
        </span>
        <UiSelect
          :model-value="pickerValue"
          :options="pickerOptions"
          :placeholder="zh ? '选择分组与模型加入估算…' : 'Select a group & model to compare…'"
          :aria-label="zh ? '添加对比模型' : 'Add model to compare'"
          @update:model-value="onPickModel"
        />
      </div>

      <!-- Sliders & Switches Control Surface (NO dot-matrix texture) -->
      <section class="space-y-4 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] p-4">
        <UiSlider
          v-model="inputTokens"
          :min="200"
          :max="128000"
          :step="200"
          :label="zh ? '平均输入 Tokens / 次' : 'Avg input tokens / call'"
        >
          <template #value>{{ inputTokens.toLocaleString() }}</template>
        </UiSlider>

        <UiSlider
          v-model="outputTokens"
          :min="100"
          :max="16000"
          :step="100"
          :label="zh ? '平均输出 Tokens / 次' : 'Avg output tokens / call'"
        >
          <template #value>{{ outputTokens.toLocaleString() }}</template>
        </UiSlider>

        <UiSlider
          v-model="dailyRequests"
          :min="10"
          :max="5000"
          :step="10"
          :label="zh ? '日均调用次数' : 'Daily calls'"
        >
          <template #value>{{ dailyRequests.toLocaleString() }} {{ zh ? '次/日' : '/ day' }}</template>
        </UiSlider>

        <div class="space-y-3 border-t border-[var(--atelier-border)] pt-3">
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <p class="text-xs font-medium text-[var(--atelier-ink)]">
                {{ zh ? '启用 Prompt 缓存命中估算' : 'Prompt cache hit rate' }}
              </p>
              <p class="text-[11px] text-[var(--atelier-ink-muted)]">
                {{ cacheEnabled
                  ? (zh ? `当前按 ${cacheHitPct}% 输入命中缓存计算` : `Simulating ${cacheHitPct}% cached input tokens`)
                  : (zh ? '按 0% 缓存命中（全量输入计费）' : '0% cache hits (full input billing)') }}
              </p>
            </div>
            <UiSwitch
              v-model="cacheEnabled"
              :aria-label="zh ? '启用 Prompt 缓存命中估算' : 'Enable prompt cache hit rate'"
            />
          </div>

          <UiSlider
            v-if="cacheEnabled"
            v-model="cacheHitPct"
            :min="10"
            :max="95"
            :step="5"
            :label="zh ? '缓存命中率' : 'Cache hit ratio'"
          >
            <template #value>{{ cacheHitPct }}%</template>
          </UiSlider>

          <div
            v-if="hasAnyPeak"
            class="flex items-center justify-between gap-3 border-t border-[var(--atelier-border-subtle)] pt-3"
          >
            <div class="min-w-0">
              <p class="text-xs font-medium text-[var(--atelier-ink)]">
                {{ zh ? '按高峰期倍率估算' : 'Simulate peak-hour multipliers' }}
              </p>
              <p class="text-[11px] text-[var(--atelier-ink-muted)]">
                {{ zh ? '对配置了高峰倍率的分组应用高峰费率' : 'Apply peak-window multipliers where configured' }}
              </p>
            </div>
            <UiSwitch
              v-model="applyPeak"
              :aria-label="zh ? '按高峰期倍率估算' : 'Simulate peak-hour multipliers'"
            />
          </div>
        </div>
      </section>

      <!-- Compared Models Results -->
      <section class="space-y-3">
        <div class="flex items-center justify-between gap-2">
          <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--atelier-ink-muted)]">
            {{ zh ? `估算对比 (${computedRows.length})` : `Comparison (${computedRows.length})` }}
          </h3>
          <button
            v-if="comparedKeys.length > 1"
            type="button"
            class="text-xs font-medium text-[var(--atelier-ink-muted)] transition-colors hover:text-[var(--atelier-ink)]"
            @click="emit('clear-compare')"
          >
            {{ zh ? '清空自选' : 'Reset selection' }}
          </button>
        </div>

        <ul class="divide-y divide-[var(--atelier-border-subtle)] rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)]">
          <li
            v-for="(row, idx) in computedRows"
            :key="row.entry.key"
            class="group relative p-4 transition-colors hover:bg-[var(--atelier-hover)]"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="flex min-w-0 items-center gap-2.5">
                <BrandModelIcon :model="row.entry.modelName" size="sm" />
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-1.5">
                    <strong class="truncate text-sm font-semibold text-[var(--atelier-ink)]">
                      {{ row.entry.modelName }}
                    </strong>
                    <span
                      v-if="computedRows.length > 1 && idx === 0"
                      class="rounded-full bg-[var(--atelier-ok-bg)] px-2 py-0.5 text-[10.5px] font-medium text-[var(--atelier-ok)]"
                    >
                      {{ zh ? '预估最省' : 'Lowest' }}
                    </span>
                  </div>
                  <p class="mt-0.5 truncate text-xs text-[var(--atelier-ink-muted)]">
                    {{ row.entry.groupName }} · <span class="font-mono tabular-nums">×{{ row.effectiveMultiplier }}</span>
                  </p>
                </div>
              </div>

              <div class="flex shrink-0 items-center gap-2">
                <!-- Hover-revealed micro "去实测 →" / "Try →" link -->
                <button
                  type="button"
                  class="inline-flex items-center gap-1 rounded-md border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] px-2 py-1 text-[11.5px] font-medium text-[var(--atelier-ink)] opacity-0 transition-opacity hover:bg-[var(--atelier-subtle)] focus-visible:opacity-100 group-hover:opacity-100"
                  @click="tryInPlayground(row.entry.modelName)"
                >
                  {{ zh ? '去实测 →' : 'Try →' }}
                </button>
                <button
                  v-if="comparedKeys.includes(row.entry.key)"
                  type="button"
                  class="inline-flex h-6 w-6 items-center justify-center rounded text-[var(--atelier-ink-muted)] hover:bg-[var(--atelier-subtle)] hover:text-[var(--atelier-ink)]"
                  :aria-label="zh ? '移除对比' : 'Remove from comparison'"
                  @click="emit('remove-compare', row.entry.key)"
                >
                  ×
                </button>
              </div>
            </div>

            <div class="mt-3 grid grid-cols-3 gap-3 border-t border-[var(--atelier-border-subtle)] pt-3 font-mono tabular-nums">
              <div>
                <span class="block font-sans text-[11px] text-[var(--atelier-ink-muted)]">
                  {{ zh ? '预计月费 (30天)' : 'Monthly (30d)' }}
                </span>
                <strong class="mt-0.5 block text-base font-semibold text-[var(--atelier-ink)]">
                  ${{ row.monthlyCost.toFixed(2) }}
                </strong>
              </div>
              <div>
                <span class="block font-sans text-[11px] text-[var(--atelier-ink-muted)]">
                  {{ zh ? '日均 / 单次' : 'Daily / Call' }}
                </span>
                <span class="mt-0.5 block text-xs font-medium text-[var(--atelier-ink)]">
                  ${{ row.dailyCost.toFixed(3) }}
                </span>
                <span class="block text-[11px] text-[var(--atelier-ink-muted)]">
                  ${{ row.perRequestCost.toFixed(5) }}
                </span>
              </div>
              <div>
                <span class="block font-sans text-[11px] text-[var(--atelier-ink-muted)]">
                  {{ zh ? '缓存月省' : 'Cache saved' }}
                </span>
                <span
                  class="mt-0.5 block text-xs font-medium"
                  :class="row.monthlyCacheSavings > 0 ? 'text-[var(--atelier-ok)]' : 'text-[var(--atelier-ink-muted)]'"
                >
                  {{ row.monthlyCacheSavings > 0 ? `-$${row.monthlyCacheSavings.toFixed(2)}` : '$0.00' }}
                </span>
                <span class="block text-[11px] text-[var(--atelier-ink-muted)]">
                  ${{ row.entry.inputPerM.toFixed(2) }} / ${{ row.entry.outputPerM.toFixed(2) }}
                </span>
              </div>
            </div>
          </li>
        </ul>
      </section>
    </div>

    <template #footer>
      <UiButton variant="secondary" size="md" @click="emit('close')">
        {{ zh ? '关闭' : 'Close' }}
      </UiButton>
    </template>
  </UiDrawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { UiDrawer, UiButton, UiSelect, UiSlider, UiSwitch, BrandModelIcon } from '@/ui'
import type { ModelPlazaGroup } from '@/api/modelPlaza'

interface PricedEntry {
  key: string
  groupName: string
  modelName: string
  inputPerM: number
  outputPerM: number
  cacheReadPerM: number
  groupMultiplier: number
  peakRateEnabled: boolean
  peakMultiplier: number
}

const props = withDefaults(
  defineProps<{
    open: boolean
    groups: ModelPlazaGroup[]
    comparedKeys?: string[]
  }>(),
  {
    comparedKeys: () => []
  }
)

const emit = defineEmits<{
  close: []
  'add-compare': [key: string]
  'remove-compare': [key: string]
  'clear-compare': []
}>()

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const router = useRouter()

const inputTokens = ref(4000)
const outputTokens = ref(1000)
const dailyRequests = ref(200)
const cacheEnabled = ref(true)
const cacheHitPct = ref(50)
const applyPeak = ref(false)
const pickerValue = ref('')

const pricedItems = computed<PricedEntry[]>(() => {
  const entries: PricedEntry[] = []
  for (const group of props.groups || []) {
    const groupMult = Number(group.user_rate_multiplier ?? group.rate_multiplier ?? 1)
    const peakMult = Number(group.peak_rate_multiplier || 1)
    for (const model of group.models || []) {
      const p: any = model.pricing || model.official_pricing
      if (!p) continue
      const inputRaw = Number(p.input_price ?? p.input_cost_per_token ?? 0)
      const outputRaw = Number(p.output_price ?? p.output_cost_per_token ?? 0)
      if (inputRaw <= 0 && outputRaw <= 0) continue
      const toPerM = (v: number) => (v > 0 && v < 0.01 ? v * 1_000_000 : v)
      const inputPerM = toPerM(inputRaw)
      const outputPerM = toPerM(outputRaw)
      const cacheReadRaw = Number(p.cache_read_price ?? p.cache_read_input_token_cost ?? 0)
      const cacheReadPerM = cacheReadRaw > 0 ? toPerM(cacheReadRaw) : inputPerM * 0.25

      entries.push({
        key: `${group.id}::${model.platform}::${model.name}`,
        groupName: group.name,
        modelName: model.name,
        inputPerM,
        outputPerM,
        cacheReadPerM,
        groupMultiplier: Number.isFinite(groupMult) ? groupMult : 1,
        peakRateEnabled: Boolean(group.peak_rate_enabled),
        peakMultiplier: Number.isFinite(peakMult) && peakMult > 0 ? peakMult : 1
      })
    }
  }
  return entries
})

const pickerOptions = computed(() =>
  pricedItems.value.map((entry) => ({
    value: entry.key,
    label: `${entry.modelName} (${entry.groupName} · ×${entry.groupMultiplier})`,
    disabled: props.comparedKeys.includes(entry.key)
  }))
)

function onPickModel(val: string | number | boolean | null) {
  const key = String(val || '')
  if (!key) return
  emit('add-compare', key)
  pickerValue.value = ''
}

const activeEntries = computed<PricedEntry[]>(() => {
  if (props.comparedKeys.length > 0) {
    const byKey = new Map(pricedItems.value.map((item) => [item.key, item]))
    const selected = props.comparedKeys
      .map((key) => byKey.get(key))
      .filter((item): item is PricedEntry => Boolean(item))
    if (selected.length > 0) return selected
  }
  return pricedItems.value.slice(0, 3)
})

const hasAnyPeak = computed(() => activeEntries.value.some((e) => e.peakRateEnabled))

const computedRows = computed(() => {
  const effectiveCacheRatio = cacheEnabled.value ? cacheHitPct.value / 100 : 0
  const inM = inputTokens.value / 1_000_000
  const outM = outputTokens.value / 1_000_000

  return activeEntries.value
    .map((entry) => {
      const mult = applyPeak.value && entry.peakRateEnabled
        ? Number((entry.groupMultiplier * entry.peakMultiplier).toFixed(3))
        : entry.groupMultiplier
      const uncachedIn = inM * (1 - effectiveCacheRatio) * entry.inputPerM
      const cachedIn = inM * effectiveCacheRatio * entry.cacheReadPerM
      const outCost = outM * entry.outputPerM
      const perRequestCost = (uncachedIn + cachedIn + outCost) * mult
      const dailyCost = perRequestCost * dailyRequests.value
      const monthlyCost = dailyCost * 30
      const noCachePerReq = (inM * entry.inputPerM + outCost) * mult
      const monthlyCacheSavings = Math.max(0, (noCachePerReq - perRequestCost) * dailyRequests.value * 30)
      return {
        entry,
        effectiveMultiplier: mult,
        perRequestCost,
        dailyCost,
        monthlyCost,
        monthlyCacheSavings
      }
    })
    .sort((a, b) => a.monthlyCost - b.monthlyCost)
})

function tryInPlayground(modelName: string) {
  emit('close')
  void router.push({ path: '/playground', query: { model: modelName } })
}
</script>
