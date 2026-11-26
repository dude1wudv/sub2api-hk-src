<template>
  <section class="portal-panel cost-calc-card" :aria-label="zh ? '模型费用估算器' : 'Model Cost Estimator'">
    <header class="calc-header">
      <div>
        <h2>{{ zh ? '模型费用估算器' : 'Model Cost Estimator' }}</h2>
        <p>{{ zh ? '结合分组倍率、缓存命中率与预估吞吐，实时计算单次、日均与月度开销。' : 'Estimate per-request, daily, and monthly spend with group multipliers and prompt cache hit ratio.' }}</p>
      </div>
      <MdButton v-if="selectedModelId" variant="outlined" size="sm" @click="tryInPlayground">
        <template #icon><Icon name="play" size="sm" aria-hidden="true" /></template>
        {{ zh ? '去调试台实测' : 'Try in Playground' }}
      </MdButton>
    </header>

    <div v-if="pricedItems.length === 0" class="calc-empty">
      <h3>{{ zh ? '当前暂无可用 Token 单价数据' : 'No token pricing available' }}</h3>
      <p>{{ zh ? '请先在「模型与定价」标签页确认当前分组已公开 Token 计费价格。' : 'Check the catalog tab once token pricing is published for your groups.' }}</p>
    </div>

    <div v-else class="calc-layout">
      <div class="calc-controls">
        <div class="portal-field">
          <span class="calc-field-label">{{ zh ? '选择分组与模型' : 'Select group & model' }}</span>
          <Select
            v-model="selectedKey"
            :options="modelOptions"
            searchable
            :placeholder="zh ? '搜索或选择分组与模型' : 'Search or select group & model'"
            :aria-label="zh ? '选择分组与模型' : 'Select group & model'"
          >
            <template #selected="{ option }">
              <span v-if="option" class="calc-select-item">
                <ModelIcon :model="String(option.modelName || '')" size="18px" />
                <span class="calc-select-model">{{ option.modelName }}</span>
                <span class="calc-select-group">{{ option.groupName }} · ×{{ option.groupMultiplier }}</span>
              </span>
              <span v-else>{{ zh ? '搜索或选择分组与模型' : 'Search or select group & model' }}</span>
            </template>
            <template #option="{ option, selected }">
              <span class="calc-select-item">
                <ModelIcon :model="String(option.modelName || '')" size="18px" />
                <span class="calc-select-model">{{ option.modelName }}</span>
                <span class="calc-select-group">{{ option.groupName }} · ×{{ option.groupMultiplier }}</span>
              </span>
              <Icon v-if="selected" name="check" size="sm" aria-hidden="true" />
            </template>
          </Select>
        </div>

        <div class="calc-slider-group">
          <div class="calc-slider-item">
            <div class="calc-slider-head">
              <label for="calc-in">{{ zh ? '平均输入 Tokens / 次' : 'Avg input tokens / req' }}</label>
              <output for="calc-in">{{ inputTokens.toLocaleString() }}</output>
            </div>
            <MdSlider
              id="calc-in"
              v-model="inputTokens"
              :min="200"
              :max="128000"
              :step="200"
              :show-value="false"
              :labeled="false"
              :aria-label="zh ? '平均输入 Tokens / 次' : 'Avg input tokens / req'"
            />
          </div>

          <div class="calc-slider-item">
            <div class="calc-slider-head">
              <label for="calc-out">{{ zh ? '平均输出 Tokens / 次' : 'Avg output tokens / req' }}</label>
              <output for="calc-out">{{ outputTokens.toLocaleString() }}</output>
            </div>
            <MdSlider
              id="calc-out"
              v-model="outputTokens"
              :min="100"
              :max="16000"
              :step="100"
              :show-value="false"
              :labeled="false"
              :aria-label="zh ? '平均输出 Tokens / 次' : 'Avg output tokens / req'"
            />
          </div>

          <div class="calc-slider-item">
            <div class="calc-slider-head">
              <label for="calc-cache">{{ zh ? 'Prompt 缓存命中率' : 'Prompt cache hit rate' }}</label>
              <output for="calc-cache">{{ cacheHitPct }}%</output>
            </div>
            <MdSlider
              id="calc-cache"
              v-model="cacheHitPct"
              :min="0"
              :max="95"
              :step="5"
              :show-value="false"
              :labeled="false"
              :aria-label="zh ? 'Prompt 缓存命中率' : 'Prompt cache hit rate'"
            />
          </div>

          <div class="calc-slider-item">
            <div class="calc-slider-head">
              <label for="calc-daily">{{ zh ? '预计日均请求数' : 'Estimated requests / day' }}</label>
              <output for="calc-daily">{{ dailyRequests.toLocaleString() }}</output>
            </div>
            <MdSlider
              id="calc-daily"
              v-model="dailyRequests"
              :min="10"
              :max="5000"
              :step="10"
              :show-value="false"
              :labeled="false"
              :aria-label="zh ? '预计日均请求数' : 'Estimated requests / day'"
            />
          </div>
        </div>

        <div v-if="selectedEntry && selectedEntry.peakRateEnabled" class="calc-switch-card">
          <div class="calc-switch-copy">
            <strong>{{ zh ? `按高峰期倍率 (${selectedEntry.peakMultiplier}×) 估算` : `Simulate peak multiplier (${selectedEntry.peakMultiplier}×)` }}</strong>
            <small v-if="selectedEntry.peakWindow">{{ zh ? '高峰时段' : 'Peak window' }}: {{ selectedEntry.peakWindow }}</small>
          </div>
          <MdSwitch v-model="applyPeak" :aria-label="zh ? '按高峰期倍率估算' : 'Simulate peak multiplier'" />
        </div>
      </div>

      <div v-if="selectedEntry" class="calc-ledger">
        <div class="calc-ledger-head">
          <span class="calc-ledger-icon" aria-hidden="true">
            <ModelIcon :model="selectedEntry.modelName" size="24px" />
          </span>
          <div class="calc-ledger-identity">
            <strong>{{ selectedEntry.modelName }}</strong>
            <span>{{ selectedEntry.groupName }}</span>
          </div>
          <span class="calc-ledger-rate">{{ zh ? '生效倍率' : 'Rate' }} ×{{ effectiveMultiplier }}</span>
        </div>

        <div class="calc-ledger-hero">
          <div class="calc-ledger-primary">
            <span class="calc-ledger-label">{{ zh ? '预计月费用（30天）' : 'Estimated monthly (30d)' }}</span>
            <strong class="calc-ledger-amount">${{ monthlyCost.toFixed(2) }}</strong>
            <small>{{ zh ? `日均 $${dailyCost.toFixed(3)} · ${dailyRequests.toLocaleString()} 次/日` : `$${dailyCost.toFixed(3)}/day · ${dailyRequests.toLocaleString()} req/day` }}</small>
          </div>

          <div class="calc-ledger-subgrid">
            <div class="calc-ledger-stat">
              <span class="calc-ledger-label">{{ zh ? '单次请求费用' : 'Per request' }}</span>
              <strong>${{ perRequestCost.toFixed(5) }}</strong>
              <small>{{ zh ? `${inputTokens.toLocaleString()} 入 / ${outputTokens.toLocaleString()} 出` : `${inputTokens.toLocaleString()} in / ${outputTokens.toLocaleString()} out` }}</small>
            </div>
            <div class="calc-ledger-stat">
              <span class="calc-ledger-label">{{ zh ? '缓存预计月省' : 'Monthly cache savings' }}</span>
              <strong class="calc-ledger-savings">${{ monthlyCacheSavings.toFixed(2) }}</strong>
              <small>{{ cacheHitPct }}% {{ zh ? '输入命中缓存' : 'cached input' }}</small>
            </div>
          </div>
        </div>

        <div class="calc-breakdown">
          <div class="calc-breakdown-head">
            <h3>{{ zh ? '单价基线' : 'Unit pricing baseline' }}</h3>
            <span>USD / 1M tokens</span>
          </div>
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
              <dt>{{ zh ? '缓存读取' : 'Cache read' }}</dt>
              <dd>${{ selectedEntry.cacheReadPerM.toFixed(3) }}</dd>
            </div>
            <div>
              <dt>{{ zh ? '分组基础倍率' : 'Group rate' }}</dt>
              <dd>×{{ selectedEntry.groupMultiplier }}</dd>
            </div>
          </dl>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { MdButton, MdSlider, MdSwitch } from '@/components/md'
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

const modelOptions = computed<SelectOption[]>(() =>
  pricedItems.value.map((item) => ({
    value: item.key,
    label: `${item.modelName} · ${item.groupName} (×${item.groupMultiplier})`,
    modelName: item.modelName,
    groupName: item.groupName,
    groupMultiplier: item.groupMultiplier
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
.cost-calc-card {
  background: rgb(var(--surface));
}

.calc-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding-bottom: 18px;
  margin-bottom: 22px;
  border-bottom: 1px solid rgb(var(--line));
}

.calc-header h2 {
  margin: 0;
  font-size: 16.5px;
  font-weight: 650;
  letter-spacing: -0.01em;
  color: rgb(var(--ink));
}

.calc-header p {
  margin: 4px 0 0;
  font-size: 13px;
  color: rgb(var(--ink-3));
}

.calc-empty {
  padding: 44px 20px;
  text-align: center;
  border: 1px dashed rgb(var(--line));
  border-radius: 12px;
  background: rgb(var(--surface-2) / 0.45);
}

.calc-empty h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: rgb(var(--ink));
}

.calc-empty p {
  margin: 6px 0 0;
  font-size: 13px;
  color: rgb(var(--ink-3));
}

.calc-layout {
  display: grid;
  grid-template-columns: minmax(300px, 430px) minmax(0, 1fr);
  gap: 24px;
  align-items: start;
}

.calc-controls {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.calc-field-label {
  font-size: 13px;
  font-weight: 550;
  color: rgb(var(--ink-2));
}

.calc-select-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.calc-select-model {
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 600;
  color: rgb(var(--ink));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calc-select-group {
  margin-left: auto;
  flex-shrink: 0;
  font-size: 11.5px;
  color: rgb(var(--ink-3));
  font-variant-numeric: tabular-nums;
}

.calc-slider-group {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border: 1px solid rgb(var(--line));
  border-radius: 12px;
  background: rgb(var(--surface));
}

.calc-slider-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.calc-slider-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.calc-slider-head label {
  font-size: 12.5px;
  font-weight: 550;
  color: rgb(var(--ink-2));
}

.calc-slider-head output {
  padding: 2px 8px;
  border-radius: 6px;
  background: rgb(var(--surface-2));
  border: 1px solid rgb(var(--line));
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 600;
  color: rgb(var(--ink));
  font-variant-numeric: tabular-nums;
}

.calc-switch-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid rgb(var(--line));
  border-radius: 12px;
  background: rgb(var(--surface));
}

.calc-switch-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.calc-switch-copy strong {
  font-size: 13px;
  font-weight: 600;
  color: rgb(var(--ink));
}

.calc-switch-copy small {
  font-size: 12px;
  color: rgb(var(--ink-3));
}

/* OpenAI-white editorial Cost Ledger (zero colored AI-slop KPI fills) */
.calc-ledger {
  display: flex;
  flex-direction: column;
  border: 1px solid rgb(var(--line));
  border-radius: 14px;
  background: rgb(var(--surface));
  overflow: hidden;
  min-width: 0;
}

.calc-ledger-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2) / 0.4);
}

.calc-ledger-icon {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 10px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface));
}

.calc-ledger-identity {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.calc-ledger-identity strong {
  font-family: var(--font-mono);
  font-size: 14px;
  font-weight: 650;
  color: rgb(var(--ink));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calc-ledger-identity span {
  font-size: 12.5px;
  color: rgb(var(--ink-3));
}

.calc-ledger-rate {
  flex-shrink: 0;
  padding: 3px 10px;
  border-radius: 999px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface));
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 600;
  color: rgb(var(--ink));
  font-variant-numeric: tabular-nums;
}

.calc-ledger-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr);
  border-bottom: 1px solid rgb(var(--line));
}

.calc-ledger-primary {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
  padding: 24px 22px;
  border-right: 1px solid rgb(var(--line));
}

.calc-ledger-label {
  font-size: 12px;
  font-weight: 550;
  color: rgb(var(--ink-3));
}

.calc-ledger-amount {
  font-family: var(--font-mono);
  font-size: 34px;
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: rgb(var(--ink));
  font-variant-numeric: tabular-nums;
}

.calc-ledger-primary small,
.calc-ledger-stat small {
  font-size: 12px;
  color: rgb(var(--ink-3));
  font-variant-numeric: tabular-nums;
}

.calc-ledger-subgrid {
  display: grid;
  grid-template-rows: 1fr 1fr;
}

.calc-ledger-stat {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 3px;
  padding: 16px 20px;
}

.calc-ledger-stat + .calc-ledger-stat {
  border-top: 1px solid rgb(var(--line));
}

.calc-ledger-stat strong {
  font-family: var(--font-mono);
  font-size: 20px;
  font-weight: 650;
  line-height: 1.25;
  color: rgb(var(--ink));
  font-variant-numeric: tabular-nums;
}

.calc-ledger-stat strong.calc-ledger-savings {
  color: rgb(var(--text-ok));
}

.calc-breakdown {
  padding: 16px 20px 18px;
}

.calc-breakdown-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.calc-breakdown-head h3 {
  margin: 0;
  font-size: 12.5px;
  font-weight: 600;
  color: rgb(var(--ink-2));
}

.calc-breakdown-head span {
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: rgb(var(--ink-3));
}

.calc-breakdown dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.calc-breakdown dl > div {
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2) / 0.45);
}

.calc-breakdown dt {
  font-size: 11.5px;
  color: rgb(var(--ink-3));
}

.calc-breakdown dd {
  margin: 4px 0 0;
  font-family: var(--font-mono);
  font-size: 14px;
  font-weight: 650;
  color: rgb(var(--ink));
  font-variant-numeric: tabular-nums;
}

@media (max-width: 960px) {
  .calc-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 640px) {
  .calc-ledger-hero {
    grid-template-columns: minmax(0, 1fr);
  }

  .calc-ledger-primary {
    border-right: 0;
    border-bottom: 1px solid rgb(var(--line));
  }

  .calc-breakdown dl {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
