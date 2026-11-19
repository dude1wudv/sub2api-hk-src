<template>
  <MdCard variant="outlined" padding="md" class="cost-calc-card">
    <template #header>
      <div>
        <h2 class="md-card-title">{{ zh ? '模型费用估算器' : 'Model Cost Estimator' }}</h2>
        <p class="md-card-subtitle">
          {{ zh ? '结合分组倍率、缓存命中率与预估吞吐，实时计算单次/日/月开销。' : 'Estimate per-request, daily, and monthly spend with group multipliers and prompt cache hit ratio.' }}
        </p>
      </div>
    </template>
    <template #actions>
      <MdButton v-if="selectedModelId" variant="tonal" size="sm" @click="tryInPlayground">
        <template #icon><Icon name="play" size="xs" /></template>
        {{ zh ? '去调试台实测' : 'Try in Playground' }}
      </MdButton>
    </template>

    <MdEmptyState
      v-if="pricedItems.length === 0"
      compact
      :title="zh ? '当前暂无可用 Token 单价数据' : 'No token pricing available'"
      :description="zh ? '请先在「模型与定价」标签页确认当前分组已公开 Token 计费价格。' : 'Check the catalog tab once token pricing is published for your groups.'"
    />

    <div v-else class="calc-layout">
      <div class="calc-controls">
        <MdSelect
          v-model="selectedKey"
          :label="zh ? '选择分组与模型' : 'Select group & model'"
          :options="modelOptions"
        />

        <MdSlider
          v-model="inputTokens"
          :min="200"
          :max="128000"
          :step="200"
          :label="zh ? '平均输入 Tokens / 次' : 'Avg input tokens / req'"
        />

        <MdSlider
          v-model="outputTokens"
          :min="100"
          :max="16000"
          :step="100"
          :label="zh ? '平均输出 Tokens / 次' : 'Avg output tokens / req'"
        />

        <MdSlider
          v-model="cacheHitPct"
          :min="0"
          :max="95"
          :step="5"
          unit="%"
          :label="zh ? 'Prompt 缓存命中率' : 'Prompt cache hit rate'"
        />

        <MdSlider
          v-model="dailyRequests"
          :min="10"
          :max="5000"
          :step="10"
          :label="zh ? '预计日均请求数' : 'Estimated requests / day'"
        />

        <MdSwitch
          v-if="selectedEntry && selectedEntry.peakRateEnabled"
          v-model="applyPeak"
          :label="zh ? `按高峰期倍率 (${selectedEntry.peakMultiplier}x) 估算` : `Simulate peak multiplier (${selectedEntry.peakMultiplier}x)`"
          :description="selectedEntry.peakWindow ? `${zh ? '高峰时段' : 'Peak window'}: ${selectedEntry.peakWindow}` : undefined"
        />
      </div>

      <div v-if="selectedEntry" class="calc-results">
        <div class="calc-kpi-grid">
          <div class="calc-kpi calc-kpi--primary">
            <span>{{ zh ? '预计月费用 (30天)' : 'Est. monthly (30d)' }}</span>
            <strong>${{ monthlyCost.toFixed(2) }}</strong>
            <small>{{ zh ? `日均 $${dailyCost.toFixed(3)} · ${dailyRequests} 次/日` : `$${dailyCost.toFixed(3)}/day · ${dailyRequests} req/day` }}</small>
          </div>
          <div class="calc-kpi">
            <span>{{ zh ? '单次请求费用' : 'Per request' }}</span>
            <strong>${{ perRequestCost.toFixed(5) }}</strong>
            <small>{{ zh ? `生效倍率 ${effectiveMultiplier}x` : `Effective ${effectiveMultiplier}x` }}</small>
          </div>
          <div class="calc-kpi">
            <span>{{ zh ? '缓存预计月省' : 'Monthly cache savings' }}</span>
            <strong>${{ monthlyCacheSavings.toFixed(2) }}</strong>
            <small>{{ cacheHitPct }}% {{ zh ? '输入命中缓存' : 'cached input' }}</small>
          </div>
        </div>

        <div class="calc-breakdown">
          <h3>{{ zh ? '当前模型费率基线（每 1M Tokens）' : 'Unit pricing baseline (per 1M tokens)' }}</h3>
          <dl>
            <div>
              <dt>{{ zh ? '输入单价' : 'Input' }}</dt>
              <dd>${{ selectedEntry.inputPerM.toFixed(3) }}</dd>
            </div>
            <div>
              <dt>{{ zh ? '输出单价' : 'Output' }}</dt>
              <dd>${{ selectedEntry.outputPerM.toFixed(3) }}</dd>
            </div>
            <div>
              <dt>{{ zh ? '缓存读取单价' : 'Cache read' }}</dt>
              <dd>${{ selectedEntry.cacheReadPerM.toFixed(3) }}</dd>
            </div>
            <div>
              <dt>{{ zh ? '分组基础倍率' : 'Group multiplier' }}</dt>
              <dd>{{ selectedEntry.groupMultiplier }}x</dd>
            </div>
          </dl>
        </div>
      </div>
    </div>
  </MdCard>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MdButton, MdCard, MdEmptyState, MdSelect, MdSlider, MdSwitch, type MdSelectOption } from '@/components/md'
import Icon from '@/components/icons/Icon.vue'
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
  peakWindow: string
}

const props = defineProps<{
  groups: ModelPlazaGroup[]
}>()

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const router = useRouter()

const inputTokens = ref(4000)
const outputTokens = ref(1000)
const cacheHitPct = ref(40)
const dailyRequests = ref(200)
const applyPeak = ref(false)
const selectedKey = ref('')

const pricedItems = computed<PricedEntry[]>(() => {
  const entries: PricedEntry[] = []
  for (const group of props.groups || []) {
    const groupMult = Number(group.rate_multiplier || 1)
    const peakMult = Number(group.peak_rate_multiplier || 1)
    const peakWindow = group.peak_start && group.peak_end ? `${group.peak_start}–${group.peak_end}` : ''
    for (const model of group.models || []) {
      const p: any = model.pricing || model.official_pricing
      if (!p) continue
      const inputRaw = Number(p.input_price ?? p.input_cost_per_token ?? 0)
      const outputRaw = Number(p.output_price ?? p.output_cost_per_token ?? 0)
      if (inputRaw <= 0 && outputRaw <= 0) continue
      // Normalize per-token (< 0.01) to per-1M tokens
      const toPerM = (v: number) => (v > 0 && v < 0.01 ? v * 1_000_000 : v)
      const inputPerM = toPerM(inputRaw)
      const outputPerM = toPerM(outputRaw)
      const cacheReadRaw = Number(p.cache_read_price ?? p.cache_read_input_token_cost ?? 0)
      const cacheReadPerM = cacheReadRaw > 0 ? toPerM(cacheReadRaw) : inputPerM * 0.25

      entries.push({
        key: `${group.id}::${model.name}`,
        groupName: group.name,
        modelName: model.name,
        inputPerM,
        outputPerM,
        cacheReadPerM,
        groupMultiplier: groupMult,
        peakRateEnabled: Boolean(group.peak_rate_enabled && peakMult > 1),
        peakMultiplier: peakMult,
        peakWindow
      })
    }
  }
  return entries.slice(0, 120)
})

watch(
  pricedItems,
  (list) => {
    if (list.length && !list.some((item) => item.key === selectedKey.value)) {
      selectedKey.value = list[0].key
    }
  },
  { immediate: true }
)

const modelOptions = computed<MdSelectOption[]>(() =>
  pricedItems.value.map((item) => ({
    value: item.key,
    label: `${item.modelName} · ${item.groupName}`,
    supportingText: `$${item.inputPerM.toFixed(2)} / $${item.outputPerM.toFixed(2)} per 1M (${item.groupMultiplier}x)`
  }))
)

const selectedEntry = computed(() => pricedItems.value.find((item) => item.key === selectedKey.value) || null)
const selectedModelId = computed(() => selectedEntry.value?.modelName || '')

const effectiveMultiplier = computed(() => {
  if (!selectedEntry.value) return 1
  const peak = applyPeak.value && selectedEntry.value.peakRateEnabled ? selectedEntry.value.peakMultiplier : 1
  return Number((selectedEntry.value.groupMultiplier * peak).toFixed(4))
})

const perRequestCost = computed(() => {
  const e = selectedEntry.value
  if (!e) return 0
  const hit = cacheHitPct.value / 100
  const uncachedInput = inputTokens.value * (1 - hit)
  const cachedInput = inputTokens.value * hit
  const baseUsd =
    (uncachedInput * e.inputPerM + cachedInput * e.cacheReadPerM + outputTokens.value * e.outputPerM) / 1_000_000
  return baseUsd * effectiveMultiplier.value
})

const perRequestNoCache = computed(() => {
  const e = selectedEntry.value
  if (!e) return 0
  const baseUsd = (inputTokens.value * e.inputPerM + outputTokens.value * e.outputPerM) / 1_000_000
  return baseUsd * effectiveMultiplier.value
})

const dailyCost = computed(() => perRequestCost.value * dailyRequests.value)
const monthlyCost = computed(() => dailyCost.value * 30)
const monthlyCacheSavings = computed(() => Math.max(0, (perRequestNoCache.value - perRequestCost.value) * dailyRequests.value * 30))

function tryInPlayground() {
  if (!selectedModelId.value) return
  void router.push({ path: '/playground', query: { model: selectedModelId.value } })
}
</script>

<style scoped>
.calc-layout {
  display: grid;
  grid-template-columns: minmax(280px, 420px) minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.calc-controls {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.calc-results {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.calc-kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}

.calc-kpi {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 16px;
  border-radius: 14px;
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
}

.calc-kpi--primary {
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
  border-color: color-mix(in srgb, var(--md-sys-color-primary) 25%, transparent);
}

.calc-kpi span {
  font-size: 0.78rem;
  opacity: 0.85;
}

.calc-kpi strong {
  font-size: 1.5rem;
  font-weight: 700;
  font-family: var(--font-mono);
}

.calc-kpi small {
  font-size: 0.75rem;
  opacity: 0.8;
}

.calc-breakdown {
  padding: 16px;
  border-radius: 14px;
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
}

.calc-breakdown h3 {
  margin: 0 0 12px;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--md-sys-color-on-surface-variant);
}

.calc-breakdown dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 10px;
}

.calc-breakdown dt {
  font-size: 0.75rem;
  color: var(--md-sys-color-on-surface-variant);
}

.calc-breakdown dd {
  margin: 2px 0 0;
  font-family: var(--font-mono);
  font-weight: 600;
  color: var(--md-sys-color-on-surface);
}

@media (max-width: 960px) {
  .calc-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
