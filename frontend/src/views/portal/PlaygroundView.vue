<template>
  <div class="portal-page playground-page">
    <PortalPageHeader
      :title="zh ? 'API 调试台' : 'API Playground'"
      :description="zh ? '直接选用你的 API 密钥向网关发送测试请求，实时观测首字延迟、吞吐速率、原始 JSON 与当前生效倍率。' : 'Send live gateway requests with your API key to inspect TTFT, throughput, raw JSON payloads, and effective billing multipliers.'"
    >
      <span v-if="billingInfo" class="pg-rate-pill" :class="{ 'is-peak': peakActive }">
        <span class="pg-rate-dot" aria-hidden="true" />
        {{ zh ? '生效倍率' : 'Effective' }}
        <strong>{{ formatMultiplier(billingInfo.effective_rate_multiplier) }}</strong>
        <template v-if="peakActive"> · {{ zh ? '高峰期' : 'Peak' }}</template>
      </span>
      <button type="button" class="portal-button secondary" @click="router.push('/keys')">
        <Icon name="key" size="sm" aria-hidden="true" />
        {{ zh ? '管理密钥' : 'Manage keys' }}
      </button>
      <button type="button" class="portal-button" @click="copyActiveSnippet">
        <Icon name="copy" size="sm" aria-hidden="true" />
        {{ zh ? '复制代码' : 'Copy snippet' }}
      </button>
    </PortalPageHeader>

    <div class="playground-grid">
      <!-- Left: Request configuration -->
      <section class="portal-panel pg-config-panel" :aria-label="zh ? '请求配置' : 'Request parameters'">
        <header class="pg-panel-header">
          <div>
            <h2>{{ zh ? '请求配置' : 'Request Parameters' }}</h2>
            <p>{{ zh ? '密钥仅保存在当前内存会话中，不会写入本地存储。' : 'API keys stay in memory for this session only and are never saved to localStorage.' }}</p>
          </div>
        </header>

        <div class="pg-stack">
          <div v-if="keyOptions.length > 0" class="portal-field">
            <span class="pg-field-label">{{ zh ? 'API 密钥' : 'API Key' }}</span>
            <Select
              v-model="selectedKeyId"
              :options="keyOptions"
              :searchable="keyOptions.length > 5"
              :placeholder="zh ? '选择 API 密钥' : 'Select API key'"
              :aria-label="zh ? '选择 API 密钥' : 'Select API key'"
            />
          </div>

          <label v-if="selectedKeyId === '__custom__' || keyOptions.length === 0" class="portal-field">
            <span>{{ zh ? '临时测试 API Key' : 'Custom API Key' }}</span>
            <input
              v-model="customKey"
              type="password"
              class="pg-mono-input"
              autocomplete="off"
              :placeholder="zh ? '粘贴 sk-... 密钥（仅本次调试有效）' : 'Paste sk-... key (memory only)'"
            />
          </label>

          <div class="portal-field">
            <span class="pg-field-label">{{ zh ? '网关协议' : 'Protocol' }}</span>
            <div class="pg-segmented" role="radiogroup" :aria-label="zh ? '网关协议' : 'Protocol'">
              <button
                type="button"
                role="radio"
                :aria-checked="protocol === 'openai'"
                :class="{ active: protocol === 'openai' }"
                @click="protocol = 'openai'"
              >
                <strong>OpenAI</strong>
                <code>/v1/chat/completions</code>
              </button>
              <button
                type="button"
                role="radio"
                :aria-checked="protocol === 'anthropic'"
                :class="{ active: protocol === 'anthropic' }"
                @click="protocol = 'anthropic'"
              >
                <strong>Anthropic</strong>
                <code>/v1/messages</code>
              </button>
            </div>
          </div>

          <label class="portal-field">
            <span>{{ zh ? '模型名称' : 'Model ID' }}</span>
            <input
              v-model="model"
              type="text"
              class="pg-mono-input"
              placeholder="gpt-4o / claude-sonnet-4-5"
            />
          </label>

          <div v-if="quickModels.length" class="pg-quick-section">
            <span class="pg-micro-label">{{ zh ? '常用 / 推荐模型' : 'Quick models' }}</span>
            <div class="pg-chip-group" role="group" :aria-label="zh ? '常用模型' : 'Quick models'">
              <button
                v-for="item in quickModels"
                :key="item"
                type="button"
                class="pg-chip"
                :class="{ active: model === item }"
                :aria-pressed="model === item"
                @click="model = item"
              >
                <ModelIcon :model="item" size="14px" />
                <span>{{ item }}</span>
              </button>
            </div>
          </div>

          <label class="portal-field">
            <span>{{ zh ? 'System Prompt（可选）' : 'System prompt (optional)' }}</span>
            <textarea
              v-model="systemPrompt"
              rows="2"
              :placeholder="zh ? '设定角色、语气或输出格式要求…' : 'Define role, style, or output format…'"
            />
          </label>

          <div class="pg-quick-section">
            <span class="pg-micro-label">{{ zh ? '快捷测试场景' : 'Prompt presets' }}</span>
            <div class="pg-chip-group" role="group" :aria-label="zh ? '快捷测试场景' : 'Prompt presets'">
              <button
                v-for="preset in promptPresets"
                :key="preset.id"
                type="button"
                class="pg-chip"
                :class="{ active: activePresetId === preset.id }"
                :aria-pressed="activePresetId === preset.id"
                @click="applyPreset(preset)"
              >
                <Icon v-if="activePresetId === preset.id" name="check" size="xs" aria-hidden="true" />
                <span>{{ preset.label }}</span>
              </button>
            </div>
          </div>

          <label class="portal-field">
            <span class="pg-label-row">
              <span>User Prompt</span>
              <small>{{ zh ? 'Ctrl / ⌘ + Enter 发送' : 'Ctrl / ⌘ + Enter to send' }}</small>
            </span>
            <textarea
              v-model="userPrompt"
              rows="4"
              :placeholder="zh ? '输入你想测试的问题或指令…' : 'Enter a prompt to test…'"
              @keydown.ctrl.enter.prevent="runRequest"
              @keydown.meta.enter.prevent="runRequest"
            />
          </label>

          <div class="pg-params-box">
            <div class="pg-slider-field">
              <div class="pg-slider-head">
                <label for="pg-temp">{{ zh ? '随机度 (Temperature)' : 'Temperature' }}</label>
                <output for="pg-temp">{{ temperature.toFixed(1) }}</output>
              </div>
              <MdSlider
                id="pg-temp"
                v-model="temperature"
                :min="0"
                :max="2"
                :step="0.1"
                :show-value="false"
                :labeled="false"
                :aria-label="zh ? '随机度 (Temperature)' : 'Temperature'"
              />
            </div>

            <div class="pg-slider-field">
              <div class="pg-slider-head">
                <label for="pg-max-tokens">{{ zh ? '最大输出 (Max Tokens)' : 'Max tokens' }}</label>
                <output for="pg-max-tokens">{{ maxTokens }}</output>
              </div>
              <MdSlider
                id="pg-max-tokens"
                v-model="maxTokens"
                :min="64"
                :max="4096"
                :step="64"
                :show-value="false"
                :labeled="false"
                :aria-label="zh ? '最大输出 (Max Tokens)' : 'Max tokens'"
              />
            </div>
          </div>

          <div class="pg-switch-card">
            <div class="pg-switch-copy">
              <strong>{{ zh ? '启用 SSE 流式输出' : 'Stream SSE response' }}</strong>
              <small>{{ zh ? '开启后可精准测量首字延迟 (TTFT)' : 'Enables accurate Time-To-First-Token measurement' }}</small>
            </div>
            <MdSwitch v-model="stream" :aria-label="zh ? '启用 SSE 流式输出' : 'Stream SSE response'" />
          </div>

          <div class="pg-actions">
            <button
              v-if="!running"
              type="button"
              class="portal-button pg-run-btn"
              :disabled="!canRun"
              @click="runRequest"
            >
              <Icon name="play" size="sm" aria-hidden="true" />
              {{ zh ? '发送测试请求' : 'Send request' }}
            </button>
            <button
              v-else
              type="button"
              class="portal-button danger pg-run-btn"
              @click="stopRequest"
            >
              <Icon name="x" size="sm" aria-hidden="true" />
              {{ zh ? '中止生成' : 'Stop' }}
            </button>
            <button
              type="button"
              class="portal-button secondary"
              :disabled="running"
              @click="clearOutput"
            >
              {{ zh ? '清空输出' : 'Clear' }}
            </button>
          </div>
        </div>
      </section>

      <!-- Right: Live telemetry & output inspector -->
      <section class="portal-panel pg-output-panel" :aria-label="zh ? '响应与观测面板' : 'Response and telemetry'">
        <header class="pg-panel-header pg-output-head">
          <div class="pg-output-title-group">
            <h2>{{ zh ? '响应与观测面板' : 'Response & Telemetry' }}</h2>
            <span class="pg-endpoint-badge">
              <span class="pg-method">POST</span>
              <code>{{ protocol === 'anthropic' ? '/v1/messages' : '/v1/chat/completions' }}</code>
            </span>
          </div>
          <span v-if="running" class="pg-live-status" role="status">
            <span class="pg-live-dot" aria-hidden="true" />
            {{ zh ? '流式接收中…' : 'Streaming…' }}
          </span>
        </header>

        <div v-if="running" class="pg-progress-track" aria-hidden="true">
          <div class="pg-progress-bar" />
        </div>

        <div class="pg-kpi-grid">
          <div class="pg-kpi">
            <span>{{ zh ? '首字延迟 (TTFT)' : 'TTFT' }}</span>
            <strong>{{ ttftMs !== null ? `${ttftMs} ms` : '—' }}</strong>
          </div>
          <div class="pg-kpi">
            <span>{{ zh ? '总耗时' : 'Total time' }}</span>
            <strong>{{ totalMs !== null ? `${totalMs} ms` : '—' }}</strong>
          </div>
          <div class="pg-kpi">
            <span>{{ zh ? 'Token 消耗 (入/出)' : 'Tokens (in/out)' }}</span>
            <strong>{{ usage.totalTokens > 0 ? `${usage.inputTokens} / ${usage.outputTokens}` : '—' }}</strong>
          </div>
          <div class="pg-kpi">
            <span>{{ zh ? '生成速率' : 'Throughput' }}</span>
            <strong>{{ tokensPerSecond !== null ? `${tokensPerSecond} tok/s` : '—' }}</strong>
          </div>
        </div>

        <div v-if="billingInfo" class="pg-billing-banner">
          <div class="pg-billing-details">
            <span>{{ zh ? '分组倍率' : 'Group' }} <strong>{{ formatMultiplier(billingInfo.group_rate_multiplier) }}</strong></span>
            <span v-if="billingInfo.user_rate_multiplier != null" class="pg-billing-sep">·</span>
            <span v-if="billingInfo.user_rate_multiplier != null">{{ zh ? '个人覆盖' : 'Personal' }} <strong>{{ formatMultiplier(billingInfo.user_rate_multiplier) }}</strong></span>
            <template v-if="billingInfo.peak_rate_enabled && billingInfo.peak_start && billingInfo.peak_end">
              <span class="pg-billing-sep">·</span>
              <span>{{ zh ? '高峰时段' : 'Peak window' }} <strong>{{ billingInfo.peak_start }}–{{ billingInfo.peak_end }}</strong></span>
            </template>
          </div>
          <span class="pg-billing-tag" :class="{ 'is-peak': peakActive }">
            {{ zh ? '当前生效' : 'Active' }} {{ formatMultiplier(billingInfo.effective_rate_multiplier) }}
          </span>
        </div>

        <div class="portal-tabs pg-tabs" role="tablist" :aria-label="zh ? '响应视图' : 'Output views'">
          <button
            v-for="tab in outputTabs"
            :key="tab.id"
            type="button"
            role="tab"
            :aria-selected="outputTab === tab.id"
            :class="{ active: outputTab === tab.id }"
            @click="outputTab = tab.id"
          >
            {{ tab.label }}
          </button>
        </div>

        <div v-if="errorMessage" class="pg-error-box" role="alert">
          <Icon name="exclamationCircle" size="sm" aria-hidden="true" />
          <div>
            <strong>{{ zh ? '请求失败' : 'Request failed' }}</strong>
            <p>{{ errorMessage }}</p>
          </div>
        </div>

        <!-- Tab 1: Rendered text output -->
        <div v-if="outputTab === 'response'" class="pg-tab-panel">
          <div v-if="outputText" class="pg-response-body">
            <pre class="pg-response-pre">{{ outputText }}</pre>
            <div class="pg-panel-foot">
              <button type="button" class="portal-button secondary pg-sm-btn" @click="copyText(outputText)">
                <Icon name="copy" size="xs" aria-hidden="true" />
                {{ zh ? '复制回复文本' : 'Copy output' }}
              </button>
            </div>
          </div>
          <div v-else class="pg-empty-state">
            <div class="pg-empty-icon" aria-hidden="true">
              <Icon name="terminal" size="lg" />
            </div>
            <h3>{{ running ? (zh ? '正在等待首字返回…' : 'Waiting for first token…') : (zh ? '尚未发起请求' : 'No response yet') }}</h3>
            <p>{{ zh ? '在左侧选择 API Key 与模型，点击「发送测试请求」即可实时查看流式回复与延迟指标。' : 'Select an API key and model on the left, then click Send request to inspect live output and latency.' }}</p>
          </div>
        </div>

        <!-- Tab 2: Raw JSON -->
        <div v-else-if="outputTab === 'json'" class="pg-tab-panel pg-json-grid">
          <div class="pg-code-block">
            <div class="pg-code-head">
              <span>Request Body</span>
              <button type="button" class="portal-inline-button" @click="copyText(formattedRequestJson)">
                {{ zh ? '复制' : 'Copy' }}
              </button>
            </div>
            <pre>{{ formattedRequestJson }}</pre>
          </div>
          <div class="pg-code-block">
            <div class="pg-code-head">
              <span>{{ zh ? '响应原始负载' : 'Raw Response Payload' }}</span>
              <button v-if="formattedResponseJson" type="button" class="portal-inline-button" @click="copyText(formattedResponseJson)">
                {{ zh ? '复制' : 'Copy' }}
              </button>
            </div>
            <pre>{{ formattedResponseJson || (zh ? '// 发送请求后在此展示原始响应 JSON' : '// Send a request to inspect raw JSON') }}</pre>
          </div>
        </div>

        <!-- Tab 3: Code snippets -->
        <div v-else class="pg-tab-panel">
          <div class="pg-snippet-bar">
            <div class="pg-chip-group" role="group" :aria-label="zh ? '代码语言' : 'Snippet language'">
              <button
                type="button"
                class="pg-chip"
                :class="{ active: snippetLang === 'curl' }"
                :aria-pressed="snippetLang === 'curl'"
                @click="snippetLang = 'curl'"
              >cURL</button>
              <button
                type="button"
                class="pg-chip"
                :class="{ active: snippetLang === 'python' }"
                :aria-pressed="snippetLang === 'python'"
                @click="snippetLang = 'python'"
              >Python</button>
              <button
                type="button"
                class="pg-chip"
                :class="{ active: snippetLang === 'node' }"
                :aria-pressed="snippetLang === 'node'"
                @click="snippetLang = 'node'"
              >Node.js</button>
            </div>
            <button type="button" class="portal-button secondary pg-sm-btn" @click="copyActiveSnippet">
              <Icon name="copy" size="xs" aria-hidden="true" />
              {{ zh ? '复制代码' : 'Copy code' }}
            </button>
          </div>
          <div class="pg-code-block">
            <pre>{{ activeSnippet }}</pre>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { MdSlider, MdSwitch } from '@/components/md'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { keysAPI } from '@/api/keys'
import { getModelPlaza } from '@/api/modelPlaza'
import { formatMultiplier, getKeyBillingInfo, isPeakRateActive, type KeyBillingInfo } from '@/api/keyBilling'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import type { ApiKey } from '@/types'
import {
  buildPlaygroundBody,
  buildPlaygroundEndpoint,
  buildPlaygroundHeaders,
  buildSnippet,
  extractNonStreamResult,
  parseSseDataLine,
  type PlaygroundProtocol,
  type PlaygroundUsage
} from '@/utils/playgroundStream'

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const { openAIBase } = usePatrickApiBase()

const apiKeys = ref<ApiKey[]>([])
const selectedKeyId = ref<string>('')
const customKey = ref('')
const protocol = ref<PlaygroundProtocol>('openai')
const model = ref('gpt-4o-mini')
const quickModels = ref<string[]>(['gpt-4o-mini', 'gpt-4o', 'claude-sonnet-4-5', 'gemini-2.5-pro'])
const systemPrompt = ref('')
const userPrompt = ref('')
const activePresetId = ref('')
const temperature = ref(0.7)
const maxTokens = ref(1024)
const stream = ref(true)

const running = ref(false)
const outputText = ref('')
const errorMessage = ref('')
const rawResponsePayload = ref<unknown>(null)
const ttftMs = ref<number | null>(null)
const totalMs = ref<number | null>(null)
const usage = ref<PlaygroundUsage>({ inputTokens: 0, outputTokens: 0, totalTokens: 0 })
const outputTab = ref('response')
const snippetLang = ref<'curl' | 'python' | 'node'>('curl')
const billingInfo = ref<KeyBillingInfo | null>(null)
let abortController: AbortController | null = null
let billingController: AbortController | null = null

const keyOptions = computed<SelectOption[]>(() => {
  const list: SelectOption[] = apiKeys.value
    .filter((k) => k.status === 'active' && Boolean(k.key))
    .map((k) => ({
      value: String(k.id),
      label: `${k.name} · ${k.group?.name || 'Default'}`
    }))
  list.push({
    value: '__custom__',
    label: zh.value ? '手动粘贴临时 API Key…' : 'Paste temporary API Key…'
  })
  return list
})

const activeApiKey = computed(() => {
  if (selectedKeyId.value === '__custom__' || !selectedKeyId.value) {
    return customKey.value.trim()
  }
  const found = apiKeys.value.find((k) => String(k.id) === selectedKeyId.value)
  return found?.key?.trim() || ''
})

const peakActive = computed(() => isPeakRateActive(billingInfo.value))

const promptPresets = computed(() => [
  {
    id: 'ping',
    label: zh.value ? '连通性测速' : 'Latency ping',
    system: '',
    user: zh.value ? '请只回复一行：OK，并说明你当前的模型能力特点（30字以内）。' : 'Reply with OK and a one-sentence capability summary.'
  },
  {
    id: 'json',
    label: zh.value ? 'JSON 结构化' : 'JSON output',
    system: 'Always respond with valid compact JSON without markdown fences.',
    user: zh.value ? '返回包含 status, model_tier, strengths(数组) 的示例 JSON。' : 'Return a sample JSON with status, model_tier, and strengths array.'
  },
  {
    id: 'code',
    label: zh.value ? '代码生成' : 'Code snippet',
    system: 'Be concise and write idiomatic TypeScript.',
    user: zh.value ? '写一个带超时的 TypeScript fetch 封装函数。' : 'Write a concise TypeScript fetch wrapper with AbortController timeout.'
  }
])

const outputTabs = computed(() => [
  { id: 'response', label: zh.value ? '实时输出' : 'Response' },
  { id: 'json', label: 'Request / JSON' },
  { id: 'snippet', label: zh.value ? '导出代码' : 'Code Snippets' }
])

const canRun = computed(() => Boolean(activeApiKey.value && model.value.trim() && userPrompt.value.trim()))

const tokensPerSecond = computed(() => {
  if (!usage.value.outputTokens || !totalMs.value || totalMs.value <= 0) return null
  const secs = Math.max(0.05, (totalMs.value - (ttftMs.value ?? 0)) / 1000)
  return (usage.value.outputTokens / secs).toFixed(1)
})

const requestOptions = computed(() => ({
  protocol: protocol.value,
  model: model.value,
  systemPrompt: systemPrompt.value,
  userPrompt: userPrompt.value,
  temperature: temperature.value,
  maxTokens: maxTokens.value,
  stream: stream.value
}))

const formattedRequestJson = computed(() => JSON.stringify(buildPlaygroundBody(requestOptions.value), null, 2))
const formattedResponseJson = computed(() =>
  rawResponsePayload.value ? JSON.stringify(rawResponsePayload.value, null, 2) : ''
)

const activeSnippet = computed(() =>
  buildSnippet(snippetLang.value, requestOptions.value, openAIBase.value || window.location.origin)
)

function applyPreset(preset: { id: string; system: string; user: string }) {
  activePresetId.value = preset.id
  systemPrompt.value = preset.system
  userPrompt.value = preset.user
}

async function copyText(text: string) {
  if (await copyToClipboard(text)) {
    appStore.showSuccess(zh.value ? '已复制到剪贴板' : 'Copied to clipboard')
  }
}

async function copyActiveSnippet() {
  await copyText(activeSnippet.value)
}

function clearOutput() {
  outputText.value = ''
  errorMessage.value = ''
  rawResponsePayload.value = null
  ttftMs.value = null
  totalMs.value = null
  usage.value = { inputTokens: 0, outputTokens: 0, totalTokens: 0 }
}

function stopRequest() {
  abortController?.abort()
  abortController = null
  running.value = false
}

async function refreshBilling(key: string) {
  billingController?.abort()
  billingInfo.value = null
  if (!key) return
  billingController = new AbortController()
  const res = await getKeyBillingInfo(key, billingController.signal)
  if (res.data) {
    billingInfo.value = res.data
  }
}

watch(activeApiKey, (nextKey) => {
  void refreshBilling(nextKey)
})

async function runRequest() {
  if (!canRun.value || running.value) return
  clearOutput()
  running.value = true
  outputTab.value = 'response'
  abortController = new AbortController()

  const startedAt = performance.now()
  const endpoint = buildPlaygroundEndpoint(protocol.value)
  const headers = buildPlaygroundHeaders(protocol.value, activeApiKey.value)
  const body = buildPlaygroundBody(requestOptions.value)

  try {
    const response = await fetch(endpoint, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
      signal: abortController.signal
    })

    if (!response.ok) {
      let msg = `HTTP ${response.status}`
      try {
        const errJson = await response.json()
        rawResponsePayload.value = errJson
        msg = errJson?.error?.message || errJson?.message || msg
      } catch {
        const rawText = await response.text().catch(() => '')
        if (rawText) msg = `${msg}: ${rawText.slice(0, 240)}`
      }
      errorMessage.value = msg
      totalMs.value = Math.round(performance.now() - startedAt)
      return
    }

    if (!stream.value || !response.body) {
      const payload = await response.json()
      const elapsed = Math.round(performance.now() - startedAt)
      ttftMs.value = elapsed
      totalMs.value = elapsed
      rawResponsePayload.value = payload
      const parsed = extractNonStreamResult(protocol.value, payload)
      outputText.value = parsed.text
      usage.value = parsed.usage
      return
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    const events: unknown[] = []

    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split(/\r?\n/)
      buffer = lines.pop() ?? ''

      for (const line of lines) {
        if (!line.startsWith('data:')) continue
        const chunk = parseSseDataLine(protocol.value, line.slice(5))
        if (chunk.rawEvent && events.length < 40) {
          events.push(chunk.rawEvent)
        }
        if (chunk.textDelta) {
          if (ttftMs.value === null) {
            ttftMs.value = Math.round(performance.now() - startedAt)
          }
          outputText.value += chunk.textDelta
        }
        if (chunk.usage) {
          const inputTokens = chunk.usage.inputTokens ?? usage.value.inputTokens
          const outputTokens = chunk.usage.outputTokens ?? usage.value.outputTokens
          const totalTokens = chunk.usage.totalTokens ?? inputTokens + outputTokens
          usage.value = { inputTokens, outputTokens, totalTokens }
        }
      }
    }

    totalMs.value = Math.round(performance.now() - startedAt)
    if (ttftMs.value === null) ttftMs.value = totalMs.value
    rawResponsePayload.value = { streamed_events_sample: events, usage: usage.value }
  } catch (err: unknown) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      totalMs.value = Math.round(performance.now() - startedAt)
      return
    }
    errorMessage.value = err instanceof Error ? err.message : String(err)
    totalMs.value = Math.round(performance.now() - startedAt)
  } finally {
    running.value = false
    abortController = null
  }
}

onMounted(async () => {
  if (typeof route.query.model === 'string' && route.query.model.trim()) {
    model.value = route.query.model.trim()
    if (model.value.toLowerCase().includes('claude')) {
      protocol.value = 'anthropic'
    }
  }
  if (!userPrompt.value) {
    applyPreset(promptPresets.value[0])
  }

  try {
    const res = await keysAPI.list(1, 50)
    apiKeys.value = res.items || []
    const requestedKeyId = typeof route.query.keyId === 'string' ? route.query.keyId : ''
    const matched = apiKeys.value.find((k) => String(k.id) === requestedKeyId && k.status === 'active')
    const firstActive = matched || apiKeys.value.find((k) => k.status === 'active' && Boolean(k.key))
    selectedKeyId.value = firstActive ? String(firstActive.id) : '__custom__'
  } catch {
    selectedKeyId.value = '__custom__'
  }

  try {
    const plaza = await getModelPlaza()
    const names = (plaza.groups || [])
      .flatMap((g) => g.models || [])
      .map((m) => m.name)
      .filter((n): n is string => typeof n === 'string' && n.length > 0)
      .slice(0, 8)
    if (names.length > 0) {
      quickModels.value = Array.from(new Set(names))
      if (!route.query.model) {
        model.value = quickModels.value[0]
      }
    }
  } catch {
    // Keep fallback quick model suggestions
  }
})

onBeforeUnmount(() => {
  abortController?.abort()
  billingController?.abort()
})
</script>

<style scoped>
.playground-page {
  min-width: 0;
}

.pg-rate-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1px solid rgb(var(--brand-moss) / 0.28);
  background: rgb(var(--brand-moss) / 0.1);
  color: rgb(var(--brand-moss));
  font-size: 12.5px;
  font-weight: 550;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.pg-rate-pill.is-peak {
  border-color: rgb(var(--text-warn) / 0.35);
  background: rgb(var(--status-warn-bg) / 0.7);
  color: rgb(var(--text-warn));
}

.pg-rate-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
}

.playground-grid {
  display: grid;
  grid-template-columns: minmax(360px, 440px) minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.pg-config-panel,
.pg-output-panel {
  min-width: 0;
  padding: 22px 24px;
}

.pg-panel-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 16px;
  margin-bottom: 18px;
  border-bottom: 1px solid rgb(var(--line));
}

.pg-panel-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 650;
  letter-spacing: -0.01em;
  color: rgb(var(--ink));
}

.pg-panel-header p {
  margin: 4px 0 0;
  font-size: 12.5px;
  color: rgb(var(--ink-3));
  line-height: 1.5;
}

.pg-stack {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.pg-field-label {
  font-size: 13px;
  font-weight: 550;
  color: rgb(var(--ink-2));
}

.pg-label-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.pg-label-row small {
  font-size: 11.5px;
  font-weight: 400;
  color: rgb(var(--ink-3));
}

.pg-mono-input {
  font-family: var(--font-mono);
  font-size: 13px;
}

.pg-segmented {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  padding: 4px;
  border: 1px solid rgb(var(--line));
  border-radius: 10px;
  background: rgb(var(--surface-2));
}

.pg-segmented button {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-width: 0;
  padding: 8px 11px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: rgb(var(--ink-2));
  text-align: left;
  cursor: pointer;
  transition: background-color var(--motion-fast) ease, border-color var(--motion-fast) ease, color var(--motion-fast) ease, box-shadow var(--motion-fast) ease;
}

.pg-segmented button strong {
  font-size: 13px;
  font-weight: 600;
}

.pg-segmented button code {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-mono);
  font-size: 11px;
  color: rgb(var(--ink-3));
}

.pg-segmented button:hover:not(.active) {
  color: rgb(var(--ink));
  background: rgb(var(--surface) / 0.55);
}

.pg-segmented button.active {
  background: rgb(var(--surface));
  border-color: rgb(var(--brand-moss) / 0.35);
  color: rgb(var(--ink));
  box-shadow: 0 1px 3px rgb(15 23 42 / 0.08);
}

.pg-segmented button.active code {
  color: rgb(var(--brand-moss));
}

.pg-quick-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.pg-micro-label {
  font-size: 12px;
  font-weight: 600;
  color: rgb(var(--ink-3));
}

.pg-chip-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.pg-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 5px 10px;
  border: 1px solid rgb(var(--line));
  border-radius: 8px;
  background: rgb(var(--surface));
  color: rgb(var(--ink-2));
  font: inherit;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.4;
  cursor: pointer;
  transition: border-color var(--motion-fast) ease, background-color var(--motion-fast) ease, color var(--motion-fast) ease;
}

.pg-chip span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pg-chip:hover {
  border-color: rgb(var(--line-strong));
  color: rgb(var(--ink));
  background: rgb(var(--surface-2) / 0.6);
}

.pg-chip.active {
  border-color: rgb(var(--brand-moss));
  background: rgb(var(--brand-moss) / 0.1);
  color: rgb(var(--brand-moss));
  font-weight: 600;
}

.pg-params-box {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding: 14px;
  border: 1px solid rgb(var(--line));
  border-radius: 10px;
  background: rgb(var(--surface-2) / 0.45);
}

.pg-slider-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.pg-slider-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.pg-slider-head label {
  font-size: 12px;
  font-weight: 550;
  color: rgb(var(--ink-2));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pg-slider-head output {
  padding: 1px 7px;
  border-radius: 6px;
  background: rgb(var(--surface));
  border: 1px solid rgb(var(--line));
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 600;
  color: rgb(var(--ink));
  font-variant-numeric: tabular-nums;
}

.pg-range {
  width: 100%;
  accent-color: rgb(var(--brand-moss));
  cursor: pointer;
}

.pg-switch-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--line));
  border-radius: 10px;
  background: rgb(var(--surface-2) / 0.35);
}

.pg-switch-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.pg-switch-copy strong {
  font-size: 13px;
  font-weight: 600;
  color: rgb(var(--ink));
}

.pg-switch-copy small {
  font-size: 12px;
  color: rgb(var(--ink-3));
}

.pg-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding-top: 4px;
}

.pg-run-btn {
  flex: 1 1 160px;
}

.pg-output-head {
  align-items: center;
}

.pg-output-title-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.pg-endpoint-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 9px 3px 4px;
  border: 1px solid rgb(var(--line));
  border-radius: 7px;
  background: rgb(var(--surface-2) / 0.7);
  font-family: var(--font-mono);
  font-size: 12px;
  color: rgb(var(--ink-2));
}

.pg-method {
  padding: 1px 6px;
  border-radius: 4px;
  background: rgb(var(--brand-moss) / 0.14);
  color: rgb(var(--brand-moss));
  font-weight: 700;
  font-size: 10.5px;
}

.pg-live-status {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgb(var(--brand-moss) / 0.12);
  color: rgb(var(--brand-moss));
  font-size: 12px;
  font-weight: 600;
}

.pg-live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
  animation: pg-blink 1.2s ease-in-out infinite;
}

@keyframes pg-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

.pg-progress-track {
  position: relative;
  height: 3px;
  margin: -10px 0 16px;
  border-radius: 999px;
  background: rgb(var(--brand-moss) / 0.16);
  overflow: hidden;
}

.pg-progress-bar {
  position: absolute;
  inset: 0 auto 0 0;
  width: 38%;
  border-radius: 999px;
  background: rgb(var(--brand-moss));
  animation: pg-slide 1.3s ease-in-out infinite;
}

@keyframes pg-slide {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(320%); }
}

.pg-kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 16px;
}

.pg-kpi {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--line));
  border-radius: 10px;
  background: rgb(var(--surface-2) / 0.45);
  min-width: 0;
}

.pg-kpi span {
  font-size: 11.5px;
  color: rgb(var(--ink-3));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pg-kpi strong {
  font-family: var(--font-mono);
  font-size: 15px;
  font-weight: 650;
  color: rgb(var(--ink));
  font-variant-numeric: tabular-nums;
}

.pg-billing-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
  margin-bottom: 16px;
  padding: 9px 14px;
  border: 1px solid rgb(var(--line));
  border-radius: 10px;
  background: rgb(var(--surface-2) / 0.4);
  font-size: 12.5px;
  color: rgb(var(--ink-2));
}

.pg-billing-details {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.pg-billing-details strong {
  color: rgb(var(--ink));
  font-family: var(--font-mono);
}

.pg-billing-sep {
  color: rgb(var(--ink-3));
}

.pg-billing-tag {
  padding: 2px 9px;
  border-radius: 999px;
  background: rgb(var(--brand-moss) / 0.12);
  color: rgb(var(--brand-moss));
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 650;
}

.pg-billing-tag.is-peak {
  background: rgb(var(--status-warn-bg) / 0.8);
  color: rgb(var(--text-warn));
}

.pg-tabs {
  margin-bottom: 16px;
}

.pg-error-box {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 14px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--text-err) / 0.3);
  border-radius: 10px;
  background: rgb(var(--status-err-bg) / 0.6);
  color: rgb(var(--text-err));
  font-size: 13px;
}

.pg-error-box p {
  margin: 3px 0 0;
  word-break: break-word;
  color: rgb(var(--ink));
}

.pg-tab-panel {
  min-width: 0;
}

.pg-response-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pg-response-pre {
  margin: 0;
  padding: 18px;
  min-height: 260px;
  max-height: 540px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  border-radius: 12px;
  background: rgb(var(--surface-2) / 0.45);
  border: 1px solid rgb(var(--line));
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.65;
  color: rgb(var(--ink));
}

.pg-panel-foot {
  display: flex;
  justify-content: flex-end;
}

.pg-sm-btn {
  min-height: 34px;
  padding: 5px 12px;
  font-size: 12.5px;
}

.pg-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  min-height: 260px;
  padding: 36px 24px;
  border: 1px dashed rgb(var(--line-strong) / 0.75);
  border-radius: 12px;
  background: rgb(var(--surface-2) / 0.3);
}

.pg-empty-icon {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  margin-bottom: 12px;
  border-radius: 12px;
  background: rgb(var(--surface));
  border: 1px solid rgb(var(--line));
  color: rgb(var(--ink-3));
}

.pg-empty-state h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: rgb(var(--ink));
}

.pg-empty-state p {
  margin: 6px 0 0;
  max-width: 420px;
  font-size: 13px;
  line-height: 1.6;
  color: rgb(var(--ink-3));
}

.pg-json-grid {
  display: grid;
  gap: 14px;
}

.pg-snippet-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.pg-code-block {
  border-radius: 12px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2) / 0.5);
  overflow: hidden;
}

.pg-code-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  border-bottom: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2) / 0.75);
  font-size: 12px;
  font-weight: 600;
  color: rgb(var(--ink-2));
}

.pg-code-block pre {
  margin: 0;
  padding: 14px 16px;
  max-height: 380px;
  overflow: auto;
  font-family: var(--font-mono);
  font-size: 12.5px;
  line-height: 1.6;
  color: rgb(var(--ink));
}

.playground-page button:focus-visible,
.playground-page input:focus-visible,
.playground-page textarea:focus-visible {
  outline: 2px solid rgb(var(--focus-ring));
  outline-offset: 2px;
}

@media (max-width: 1080px) {
  .playground-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 680px) {
  .pg-kpi-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .pg-params-box,
  .pg-segmented {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
