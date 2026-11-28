<template>
  <div class="page">
    <header class="topbar">
      <div class="brand">
        <div class="brand-mark">AI</div>
        <div>
          <h1>{{ overview?.site_title || 'XJTU Free AI' }}</h1>
          <p class="brand-sub">
            无需登录 · 复制即用 · 全站共享每日公益额度（按 Sub2API 折前 1× 原价计费）
          </p>
        </div>
      </div>

      <div class="topbar-actions">
        <a href="/docs" class="btn btn-sm">使用文档</a>
        <button type="button" class="btn btn-sm" :disabled="loading" @click="loadOverview">
          {{ loading ? '刷新中…' : '刷新状态' }}
        </button>
        <button type="button" class="btn btn-sm" @click="toggleTheme">
          {{ theme === 'dark' ? '浅色' : '深色' }}
        </button>
      </div>
    </header>

    <div v-if="overview?.site_notice" class="notice-banner">
      {{ overview.site_notice }}
    </div>

    <div v-if="errorMessage" class="alert-banner">
      {{ errorMessage }}
    </div>

    <div v-if="overview && !overview.enabled" class="alert-banner">
      公益池当前由管理员暂时关闭，请稍后再试。
    </div>

    <!-- 全站每日共享总额度主看板 -->
    <section class="card hero">
      <div class="hero-top">
        <div>
          <div class="eyebrow">
            全站今日共享池 · {{ overview?.usage_date || '加载中' }}（{{ overview?.timezone || 'Asia/Shanghai' }}）
          </div>
          <div class="hero-amount">
            <span class="hero-big tabular">
              <template v-if="overview && overview.remaining_today_usd !== null">
                剩余 {{ formatQuota(overview.remaining_today_usd) }}
              </template>
              <template v-else>
                已用 {{ formatQuota(overview?.used_today_usd) }}
              </template>
            </span>
            <span class="hero-cap tabular">
              <template v-if="overview && overview.daily_budget_usd > 0">
                已用 {{ formatQuota(overview.used_today_usd) }} / 每日总池 {{ formatQuota(overview.daily_budget_usd) }}
              </template>
              <template v-else>
                总池不设每日上限
              </template>
            </span>
          </div>
        </div>

        <div class="countdown-box">
          <div class="countdown-label">
            {{ overview?.auto_rotate_daily ? '距离 00:00 重置与 Key 轮换' : '距离北京时间 00:00 额度重置' }}
          </div>
          <div class="countdown-val mono tabular">
            {{ formatCountdown(remainingSeconds) }}
          </div>
        </div>
      </div>

      <div v-if="overview && overview.daily_budget_usd > 0" class="progress-track">
        <div
          class="progress-fill"
          :class="{ warn: poolPercent >= 90 }"
          :style="{ width: `${poolPercent}%` }"
        />
      </div>

      <div class="hero-stats">
        <div class="stat-item">
          <span class="stat-label">当前状态</span>
          <span class="stat-value">
            <template v-if="!overview">加载中…</template>
            <template v-else-if="!overview.enabled">维护暂停中</template>
            <template v-else-if="overview.pool_exhausted">今日共享池已耗尽</template>
            <template v-else>正常运行中</template>
          </span>
        </div>
        <div class="stat-item">
          <span class="stat-label">全站实时并发</span>
          <span class="stat-value tabular">
            {{ overview?.current_concurrency ?? 0 }} /
            {{ (overview?.max_concurrency ?? 0) > 0 ? overview?.max_concurrency : '不限' }}
          </span>
        </div>
        <div class="stat-item">
          <span class="stat-label">今日全站请求</span>
          <span class="stat-value tabular">
            {{ (overview?.request_count_today ?? 0).toLocaleString() }} 次
          </span>
        </div>
        <div class="stat-item">
          <span class="stat-label">计费与轮换规则</span>
          <span class="stat-value">折前 1× 原价 · 00:00 换 Key</span>
        </div>
      </div>
    </section>

    <!-- 各公益分组卡片 -->
    <div class="section-head">
      <h2>公益通道与今日公共 API Key</h2>
      <p>部分通道设有独立每日子上限，未设子上限的通道直接共享总池剩余额度</p>
    </div>

    <div v-if="overview && overview.groups.length === 0" class="card panel">
      当前暂未开放可用的公益通道，请稍后再访问。
    </div>

    <section v-else class="groups-grid">
      <article
        v-for="group in overview?.groups ?? []"
        :key="group.id"
        class="card group-card"
        :class="{ selected: selectedGroupId === group.id }"
      >
        <div class="group-header">
          <div>
            <h3 class="group-title">{{ group.name }}</h3>
            <p v-if="group.description" class="group-desc">{{ group.description }}</p>
          </div>
          <div class="badges">
            <span class="pill">{{ group.platform }}</span>
            <span class="pill" :class="group.exhausted ? 'danger' : 'ok'">
              {{ group.exhausted ? '今日已用完' : '可用' }}
            </span>
          </div>
        </div>

        <!-- 额度进度：独立子上限 vs 共享总池 -->
        <div class="quota-box">
          <div class="quota-row">
            <span>
              <template v-if="group.daily_limit_usd !== null && group.daily_limit_usd > 0">
                本组今日配额（独立上限 {{ formatQuota(group.daily_limit_usd) }}）
              </template>
              <template v-else>
                不设单组上限 · 共享总池
              </template>
            </span>
            <strong class="tabular">
              <template v-if="group.daily_limit_usd !== null && group.daily_limit_usd > 0">
                {{ formatQuota(group.used_today_usd) }} / {{ formatQuota(group.daily_limit_usd) }}
              </template>
              <template v-else>
                已用 {{ formatQuota(group.used_today_usd) }}
              </template>
            </strong>
          </div>

          <div
            v-if="group.daily_limit_usd !== null && group.daily_limit_usd > 0"
            class="progress-track thin"
          >
            <div
              class="progress-fill"
              :class="{ warn: computePercent(group.used_today_usd, group.daily_limit_usd) >= 90 }"
              :style="{ width: `${computePercent(group.used_today_usd, group.daily_limit_usd)}%` }"
            />
          </div>

          <div class="quota-meta tabular">
            <span>
              实际可用余额：
              <strong>
                {{ group.remaining_today_usd !== null ? formatQuota(group.remaining_today_usd) : '不限' }}
              </strong>
            </span>
            <span>
              在线并发：{{ group.current_concurrency }} /
              {{ group.max_concurrency > 0 ? group.max_concurrency : '跟随总池' }}
            </span>
          </div>
        </div>

        <!-- 今日公共 API Key -->
        <div class="key-box">
          <div class="key-line">
            <code class="key-value">
              {{ revealedKeys[group.id] ? group.public_api_key || '暂未开放' : maskAPIKey(group.public_api_key) }}
            </code>
          </div>
          <div class="key-actions">
            <button
              type="button"
              class="btn btn-primary btn-sm"
              :disabled="!group.public_api_key"
              @click="copyValue(group.public_api_key, `已复制「${group.name}」今日公共 Key`)"
            >
              复制今日 Key
            </button>
            <button
              type="button"
              class="btn btn-sm"
              :disabled="!group.public_api_key"
              @click="toggleReveal(group.id)"
            >
              {{ revealedKeys[group.id] ? '隐藏' : '显示完整 Key' }}
            </button>
            <button
              type="button"
              class="btn btn-sm"
              @click="selectGroupForGuide(group.id)"
            >
              {{ selectedGroupId === group.id ? '已选为接入示例' : '填入接入指南' }}
            </button>
          </div>
        </div>

        <!-- 可用模型与折前 1x 单价 -->
        <div>
          <button type="button" class="models-toggle" @click="toggleModels(group.id)">
            {{
              expandedModels[group.id]
                ? `收起可用模型与折前单价（${group.models.length}）`
                : `查看可用模型与折前 1× 单价（${group.models.length} 个模型）`
            }}
          </button>

          <div v-if="expandedModels[group.id]" class="models-table-wrap" style="margin-top: 8px">
            <table v-if="group.models.length > 0" class="models-table">
              <thead>
                <tr>
                  <th>模型</th>
                  <th>输入 / 1M</th>
                  <th>输出 / 1M</th>
                  <th>缓存读 / 1M</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="m in group.models" :key="`${m.platform}:${m.name}`">
                  <td class="mono">{{ m.name }}</td>
                  <td class="tabular">{{ formatPricePerMillion(m.input_price_per_million, m.price_currency) }}</td>
                  <td class="tabular">{{ formatPricePerMillion(m.output_price_per_million, m.price_currency) }}</td>
                  <td class="tabular">{{ formatPricePerMillion(m.cache_read_per_million, m.price_currency) }}</td>
                </tr>
              </tbody>
            </table>
            <div v-else style="padding: 10px; font-size: 12px; color: var(--text-muted)">
              该分组当前由上游动态提供模型列表，可直接请求 /v1/models 查看。
            </div>
          </div>
        </div>
      </article>
    </section>

    <!-- 当前网络状态 + Key 状态查询 -->
    <section class="two-col">
      <div class="card panel">
        <h3>当前网络（IP）今日状态</h3>
        <p class="panel-sub">
          当前出口：<code>{{ overview?.ip_status.masked_ip || '—' }}</code>（若处于校园网 NAT 出口，以下统计反映该出口总使用情况）
        </p>
        <div class="kv-list tabular">
          <div class="kv-item">
            <div class="kv-k">今日已用（折前）</div>
            <div class="kv-v">{{ formatQuota(overview?.ip_status.used_today_usd) }}</div>
          </div>
          <div class="kv-item">
            <div class="kv-k">单 IP 日限额</div>
            <div class="kv-v">
              {{
                (overview?.ip_status.daily_limit_usd ?? 0) > 0
                  ? formatQuota(overview?.ip_status.daily_limit_usd)
                  : '不限（共享总池）'
              }}
            </div>
          </div>
          <div class="kv-item">
            <div class="kv-k">单 IP 并发 / 上限</div>
            <div class="kv-v">
              {{ overview?.ip_status.current_concurrency ?? 0 }} /
              {{ (overview?.ip_status.max_concurrency ?? 0) > 0 ? overview?.ip_status.max_concurrency : '不限' }}
            </div>
          </div>
          <div class="kv-item">
            <div class="kv-k">单 IP 速率上限（RPM）</div>
            <div class="kv-v">
              {{ (overview?.ip_status.rpm_limit ?? 0) > 0 ? `${overview?.ip_status.rpm_limit} 次/分` : '不限' }}
            </div>
          </div>
        </div>
      </div>

      <div class="card panel">
        <h3>查询 Key 实时状态</h3>
        <p class="panel-sub">粘贴正在使用的公益 Key，立即确认是否已过期轮换以及所属通道剩余额度。</p>
        <form class="lookup-form" @submit.prevent="handleLookupKey">
          <input
            v-model="lookupInput"
            type="text"
            class="input mono"
            placeholder="sk-free-..."
          />
          <button type="submit" class="btn btn-primary" :disabled="lookupLoading || !lookupInput.trim()">
            {{ lookupLoading ? '查询中…' : '查询' }}
          </button>
        </form>

        <div v-if="lookupResult" class="lookup-result">
          <template v-if="!lookupResult.found">
            未找到该公益 Key，请确认是否已复制正确，或上方直接获取今日新 Key。
          </template>
          <template v-else>
            <div>
              <strong>{{ lookupResult.group_name }}</strong>
              （状态：
              <strong v-if="lookupResult.status === 'active'" style="color: var(--accent-text)">今日有效</strong>
              <strong v-else-if="lookupResult.status === 'rotated_expired'" style="color: var(--warn)">已轮换失效，请复制上方今日新 Key</strong>
              <strong v-else style="color: var(--danger)">已停用</strong>
              ）
            </div>
            <div class="tabular" style="margin-top: 4px; color: var(--text-muted)">
              总池剩余：{{
                lookupResult.pool_remaining_usd !== null ? formatQuota(lookupResult.pool_remaining_usd) : '不限'
              }}
              · 本组今日已用：{{ formatQuota(lookupResult.group_used_today_usd) }}
              <template v-if="lookupResult.group_daily_limit_usd">
                / {{ formatQuota(lookupResult.group_daily_limit_usd) }}
              </template>
            </div>
          </template>
        </div>
      </div>
    </section>

    <!-- 快速接入指南 -->
    <div class="section-head">
      <h2>快速接入指南</h2>
      <p>
        当前示例通道：<strong>{{ activeGroup?.name || '默认公益通道' }}</strong> · Base URL：<code>{{ siteOrigin }}/v1</code>
      </p>
    </div>

    <section class="card panel">
      <div class="tabs">
        <button
          v-for="tab in clientTabs"
          :key="tab.id"
          type="button"
          class="tab"
          :class="{ active: activeClient === tab.id }"
          @click="activeClient = tab.id"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="code-block">
        <pre><code>{{ activeSnippet }}</code></pre>
      </div>

      <div style="margin-top: 10px; display: flex; justify-content: flex-end">
        <button
          type="button"
          class="btn btn-sm"
          @click="copyValue(activeSnippet, '已复制接入配置')"
        >
          复制代码片段
        </button>
      </div>
    </section>

    <footer class="footer">
      本站点仅用于公益与技术学习交流 · 请合理使用，避免恶意并发与自动化刷量 · 每日北京时间 00:00 重置共享额度
    </footer>

    <div v-if="toastMessage" class="toast">{{ toastMessage }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import {
  fetchCharityOverview,
  lookupCharityKeyStatus,
  type CharityKeyLookupResult,
  type CharityPublicOverview
} from './api'
import {
  buildIntegrationSnippet,
  computePercent,
  formatCountdown,
  formatPricePerMillion,
  formatQuota,
  maskAPIKey,
  type IntegrationClient
} from './utils'

const overview = ref<CharityPublicOverview | null>(null)
const loading = ref(false)
const errorMessage = ref('')
const remainingSeconds = ref(0)
const selectedGroupId = ref<number | null>(null)
const revealedKeys = reactive<Record<number, boolean>>({})
const expandedModels = reactive<Record<number, boolean>>({})
const activeClient = ref<IntegrationClient>('claude_code')
const lookupInput = ref('')
const lookupLoading = ref(false)
const lookupResult = ref<CharityKeyLookupResult | null>(null)
const toastMessage = ref('')
const theme = ref<'light' | 'dark'>('light')

let countdownTimer: number | null = null
let toastTimer: number | null = null

const clientTabs: Array<{ id: IntegrationClient; label: string }> = [
  { id: 'claude_code', label: 'Claude Code' },
  { id: 'codex', label: 'OpenAI Codex' },
  { id: 'python', label: 'Python OpenAI SDK' },
  { id: 'curl', label: 'cURL' },
  { id: 'cherry', label: 'Cherry Studio / 通用' }
]

const siteOrigin = computed(() => {
  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin
  }
  return 'https://free.xjtu.shop'
})

const poolPercent = computed(() => {
  if (!overview.value) return 0
  return computePercent(overview.value.used_today_usd, overview.value.daily_budget_usd)
})

const activeGroup = computed(() => {
  const list = overview.value?.groups ?? []
  if (list.length === 0) return null
  return list.find((g) => g.id === selectedGroupId.value) ?? list[0]
})

const activeSnippet = computed(() => {
  const g = activeGroup.value
  const sampleModel = g?.models?.[0]?.name || (g?.platform === 'anthropic' ? 'claude-sonnet-4-6' : 'gpt-5.4')
  return buildIntegrationSnippet(activeClient.value, siteOrigin.value, g?.public_api_key || '', sampleModel)
})

function showToast(msg: string) {
  toastMessage.value = msg
  if (toastTimer !== null) window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => {
    toastMessage.value = ''
  }, 2400)
}

function applyTheme(next: 'light' | 'dark') {
  theme.value = next
  document.documentElement.dataset.theme = next
  try {
    window.localStorage.setItem('charity_theme', next)
  } catch {
    // ignore storage errors
  }
}

function toggleTheme() {
  applyTheme(theme.value === 'dark' ? 'light' : 'dark')
}

function toggleReveal(id: number) {
  revealedKeys[id] = !revealedKeys[id]
}

function toggleModels(id: number) {
  expandedModels[id] = !expandedModels[id]
}

function selectGroupForGuide(id: number) {
  selectedGroupId.value = id
  const g = overview.value?.groups.find((item) => item.id === id)
  if (g?.platform === 'anthropic') {
    activeClient.value = 'claude_code'
  } else if (g?.platform === 'openai') {
    activeClient.value = 'codex'
  }
  showToast('已将该通道的今日 Key 填入下方接入指南')
}

async function copyValue(value: string, successText: string) {
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    showToast(successText)
  } catch {
    showToast('复制失败，请手动选中复制')
  }
}

async function loadOverview() {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await fetchCharityOverview()
    overview.value = data
    remainingSeconds.value = Math.max(0, Math.floor(data.seconds_until_reset || 0))
    if (data.site_title) {
      document.title = `${data.site_title} · 公益共享 API 池`
    }
    if (selectedGroupId.value === null && data.groups.length > 0) {
      selectedGroupId.value = data.groups[0].id
    }
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : '加载公益池状态失败，请稍后刷新'
  } finally {
    loading.value = false
  }
}

async function handleLookupKey() {
  const key = lookupInput.value.trim()
  if (!key) return
  lookupLoading.value = true
  try {
    lookupResult.value = await lookupCharityKeyStatus(key)
  } catch (err: unknown) {
    showToast(err instanceof Error ? err.message : '查询失败')
  } finally {
    lookupLoading.value = false
  }
}

onMounted(() => {
  try {
    const saved = window.localStorage.getItem('charity_theme')
    if (saved === 'dark' || saved === 'light') {
      applyTheme(saved)
    } else if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
      applyTheme('dark')
    }
  } catch {
    // ignore
  }
  void loadOverview()
  countdownTimer = window.setInterval(() => {
    if (remainingSeconds.value > 0) {
      remainingSeconds.value -= 1
    } else if (overview.value) {
      void loadOverview()
    }
  }, 1000)
})

onUnmounted(() => {
  if (countdownTimer !== null) window.clearInterval(countdownTimer)
  if (toastTimer !== null) window.clearTimeout(toastTimer)
})
</script>
