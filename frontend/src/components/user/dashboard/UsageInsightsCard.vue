<template>
  <MdCard variant="outlined" padding="md" class="insights-card">
    <template #header>
      <div>
        <h2 class="md-card-title">{{ zh ? '用量画像与续航预测' : 'Usage Insights & Runway' }}</h2>
        <p class="md-card-subtitle">
          {{ zh ? '基于近期调用自动估算余额续航、主力模型分布与缓存收益。' : 'Estimated balance runway, top models, and prompt cache savings from recent activity.' }}
        </p>
      </div>
    </template>
    <template #actions>
      <MdButton variant="tonal" size="sm" @click="copySummary">
        <template #icon><Icon name="copy" size="xs" /></template>
        {{ zh ? '复制摘要' : 'Copy brief' }}
      </MdButton>
    </template>

    <div class="insights-grid">
      <!-- Balance Runway -->
      <div v-if="!isSimple" class="insight-box insight-box--tonal">
        <div class="insight-label-row">
          <span>{{ zh ? '预计余额续航' : 'Estimated runway' }}</span>
          <MdBadge :tone="runwayTone" dot>{{ runwayBadgeText }}</MdBadge>
        </div>
        <strong class="insight-big">{{ runwayHeadline }}</strong>
        <p class="insight-hint">
          {{ zh ? `近期日均消费 $${dailyBurn.toFixed(3)} · 当前余额 $${Number(balance || 0).toFixed(2)}` : `Avg $${dailyBurn.toFixed(3)}/day · Balance $${Number(balance || 0).toFixed(2)}` }}
        </p>
        <MdProgress
          variant="linear"
          :value="runwayProgress"
          :tone="runwayProgressTone"
          :aria-label="zh ? '余额续航进度' : 'Balance runway'"
        />
        <div v-if="estimatedDays !== null && estimatedDays < 7" class="insight-cta">
          <MdButton variant="filled" size="sm" @click="router.push('/purchase')">
            {{ zh ? '立即补充余额' : 'Top up balance' }}
          </MdButton>
        </div>
      </div>

      <!-- Top Models Distribution -->
      <div class="insight-box">
        <div class="insight-label-row">
          <span>{{ zh ? '主力模型 Top 3' : 'Top 3 models' }}</span>
          <span class="insight-muted">{{ zh ? '按消耗占比' : 'By spend share' }}</span>
        </div>
        <div v-if="topModels.length" class="insight-models">
          <div v-for="item in topModels" :key="item.model" class="insight-model-row">
            <div class="insight-model-meta">
              <code :title="item.model">{{ item.model }}</code>
              <span>{{ item.sharePct }}% · ${{ item.cost.toFixed(3) }}</span>
            </div>
            <MdProgress variant="linear" :value="item.shareRatio" />
          </div>
        </div>
        <p v-else class="insight-hint">{{ zh ? '暂无近期模型分布数据。' : 'No recent model activity yet.' }}</p>
      </div>

      <!-- Cache Efficiency & Average Request Size -->
      <div class="insight-box">
        <div class="insight-label-row">
          <span>{{ zh ? '效率与缓存画像' : 'Efficiency & cache' }}</span>
          <MdBadge v-if="cacheHitPct > 0" tone="success">{{ cacheHitPct }}% {{ zh ? '命中' : 'cached' }}</MdBadge>
        </div>
        <dl class="insight-dl">
          <div>
            <dt>{{ zh ? '今日请求 / 费用' : 'Today req / spend' }}</dt>
            <dd>{{ todayRequests }} · ${{ todayCost.toFixed(4) }}</dd>
          </div>
          <div>
            <dt>{{ zh ? '单次平均 Token' : 'Avg tokens / req' }}</dt>
            <dd>{{ avgTokensPerReq.toLocaleString() }}</dd>
          </div>
          <div>
            <dt>{{ zh ? '缓存读取占比' : 'Cache read share' }}</dt>
            <dd>{{ cacheHitPct }}%</dd>
          </div>
        </dl>
      </div>
    </div>
  </MdCard>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MdBadge, MdButton, MdCard, MdProgress } from '@/components/md'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import type { ModelStat, TrendDataPoint } from '@/types'
import type { UserDashboardStats } from '@/api/usage'

const props = defineProps<{
  balance: number
  stats: UserDashboardStats
  trend: TrendDataPoint[]
  models: ModelStat[]
  isSimple?: boolean
}>()

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const router = useRouter()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const dailyBurn = computed(() => {
  if (props.trend.length > 0) {
    const sum = props.trend.reduce((acc, point) => acc + Number(point.actual_cost ?? point.cost ?? 0), 0)
    const activeDays = Math.max(1, props.trend.length)
    return sum / activeDays
  }
  return Number(props.stats.today_actual_cost ?? props.stats.today_cost ?? 0)
})

const estimatedDays = computed<number | null>(() => {
  if (dailyBurn.value <= 0.0001) return null
  const bal = Number(props.balance || 0)
  if (bal <= 0) return 0
  return Math.floor(bal / dailyBurn.value)
})

const runwayHeadline = computed(() => {
  if (estimatedDays.value === null) return zh.value ? '消耗极低 · 充裕' : 'Low burn · Plenty'
  if (estimatedDays.value >= 365) return zh.value ? '365+ 天' : '365+ days'
  return zh.value ? `约 ${estimatedDays.value} 天` : `~${estimatedDays.value} days`
})

const runwayTone = computed<'success' | 'warn' | 'error' | 'primary'>(() => {
  if (estimatedDays.value === null || estimatedDays.value >= 14) return 'success'
  if (estimatedDays.value >= 5) return 'warn'
  return 'error'
})

const runwayProgress = computed(() => {
  if (estimatedDays.value === null) return 1
  return Math.max(0.05, Math.min(1, estimatedDays.value / 30))
})

const runwayProgressTone = computed<'primary' | 'warn' | 'error'>(() => {
  if (estimatedDays.value === null || estimatedDays.value >= 14) return 'primary'
  if (estimatedDays.value >= 5) return 'warn'
  return 'error'
})

const runwayBadgeText = computed(() => {
  if (estimatedDays.value === null || estimatedDays.value >= 14) return zh.value ? '健康' : 'Healthy'
  if (estimatedDays.value >= 5) return zh.value ? '留意余额' : 'Watch'
  return zh.value ? '建议充值' : 'Low'
})

const topModels = computed(() => {
  const totalCost = props.models.reduce((acc, m) => acc + Number(m.actual_cost ?? m.cost ?? 0), 0)
  const sorted = [...props.models]
    .sort((a, b) => Number(b.actual_cost ?? b.cost ?? 0) - Number(a.actual_cost ?? a.cost ?? 0))
    .slice(0, 3)

  return sorted.map((m) => {
    const cost = Number(m.actual_cost ?? m.cost ?? 0)
    const ratio = totalCost > 0 ? cost / totalCost : 0
    return {
      model: m.model,
      cost,
      shareRatio: ratio,
      sharePct: Math.round(ratio * 100)
    }
  })
})

const todayRequests = computed(() => Number(props.stats.today_requests ?? 0))
const todayCost = computed(() => Number(props.stats.today_actual_cost ?? props.stats.today_cost ?? 0))
const avgTokensPerReq = computed(() => {
  const reqs = Number(props.stats.total_requests ?? 0)
  const tokens = Number(props.stats.total_tokens ?? 0)
  return reqs > 0 ? Math.round(tokens / reqs) : 0
})

const cacheHitPct = computed(() => {
  const statsAny = props.stats as unknown as Record<string, unknown>
  const cacheRead = Number(statsAny.total_cache_read_tokens ?? statsAny.cache_read_tokens ?? 0)
  const totalInput = Number(statsAny.total_input_tokens ?? statsAny.total_tokens ?? 0)
  if (cacheRead <= 0 || totalInput <= 0) return 0
  return Math.min(100, Math.round((cacheRead / totalInput) * 100))
})

async function copySummary() {
  const lines = [
    zh.value ? 'Sub2API 用量画像摘要' : 'Sub2API Usage Insights',
    `${zh.value ? '当前余额' : 'Balance'}: $${Number(props.balance || 0).toFixed(2)} (${runwayHeadline.value})`,
    `${zh.value ? '今日调用' : 'Today'}: ${todayRequests.value} req · $${todayCost.value.toFixed(4)}`,
    `${zh.value ? '主力模型' : 'Top models'}: ${topModels.value.map((m) => `${m.model} (${m.sharePct}%)`).join(', ') || 'N/A'}`
  ]
  if (await copyToClipboard(lines.join('\n'))) {
    appStore.showSuccess(zh.value ? '已复制用量摘要' : 'Usage brief copied')
  }
}
</script>

<style scoped>
.insights-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 14px;
}

.insight-box {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  border-radius: 14px;
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
}

.insight-box--tonal {
  background: color-mix(in srgb, var(--md-sys-color-primary-container) 35%, var(--md-sys-color-surface-container-low));
}

.insight-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--md-sys-color-on-surface-variant);
}

.insight-muted {
  font-weight: 400;
  font-size: 0.75rem;
}

.insight-big {
  font-size: 1.5rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--md-sys-color-on-surface);
}

.insight-hint {
  margin: 0;
  font-size: 0.78rem;
  color: var(--md-sys-color-on-surface-variant);
}

.insight-cta {
  margin-top: 4px;
}

.insight-models {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.insight-model-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.insight-model-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 0.78rem;
}

.insight-model-meta code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-mono);
  color: var(--md-sys-color-on-surface);
}

.insight-model-meta span {
  flex-shrink: 0;
  color: var(--md-sys-color-on-surface-variant);
  font-variant-numeric: tabular-nums;
}

.insight-dl {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.insight-dl > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 0.8125rem;
}

.insight-dl dt {
  color: var(--md-sys-color-on-surface-variant);
}

.insight-dl dd {
  margin: 0;
  font-weight: 600;
  font-family: var(--font-mono);
  color: var(--md-sys-color-on-surface);
}
</style>
