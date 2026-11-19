<template>
  <div class="portal-page playground-page">
    <MdPageHeader
      :eyebrow="zh ? '开发者调试' : 'Developer Sandbox'"
      :title="zh ? 'API 调试台' : 'API Playground'"
      :description="zh ? '直接选用你的 API 密钥向网关发送测试请求，实时观测首字延迟、吞吐速率、原始 JSON 与当前生效倍率。' : 'Send live gateway requests with your API key to inspect TTFT, throughput, raw JSON payloads, and effective billing multipliers.'"
    >
      <template #badge>
        <MdBadge v-if="billingInfo" :tone="peakActive ? 'warn' : 'primary'" dot>
          {{ zh ? '生效倍率' : 'Effective' }} {{ formatMultiplier(billingInfo.effective_rate_multiplier) }}
          <template v-if="peakActive"> · {{ zh ? '高峰期' : 'Peak' }}</template>
        </MdBadge>
      </template>
      <template #actions>
        <MdButton variant="outlined" size="sm" @click="router.push('/keys')">
          {{ zh ? '管理密钥' : 'Manage keys' }}
        </MdButton>
        <MdButton variant="tonal" size="sm" @click="copyActiveSnippet">
          {{ zh ? '复制代码' : 'Copy snippet' }}
        </MdButton>
      </template>
    </MdPageHeader>

    <div class="playground-grid">
      <!-- Left: Request configuration -->
      <MdCard variant="outlined" padding="md" class="playground-config">
        <template #header>
          <div>
            <h2 class="md-card-title">{{ zh ? '请求配置' : 'Request Parameters' }}</h2>
            <p class="md-card-subtitle">{{ zh ? '密钥仅保存在当前内存会话中，不会写入本地存储。' : 'API keys stay in memory for this session only and are never saved to localStorage.' }}</p>
          </div>
        </template>

        <div class="pg-stack">
          <MdSelect
            v-if="keyOptions.length > 0"
            v-model="selectedKeyId"
            :label="zh ? '选择 API 密钥' : 'Select API key'"
            :options="keyOptions"
          />

          <MdTextField
            v-if="selectedKeyId === '__custom__' || keyOptions.length === 0"
            v-model="customKey"
            type="password"
            mono
            :label="zh ? '临时测试 API Key' : 'Custom API Key'"
            :placeholder="zh ? '粘贴 sk-... 密钥（仅本次调试有效）' : 'Paste sk-... key (memory only)'"
          />

          <div class="pg-row-2">
            <MdSelect
              v-model="protocol"
              :label="zh ? '网关协议' : 'Protocol'"
              :options="protocolOptions"
            />
            <MdTextField
              v-model="model"
              mono
              :label="zh ? '模型名称' : 'Model ID'"
              placeholder="gpt-4o / claude-sonnet-4-5"
            />
          </div>

          <div v-if="quickModels.length" class="pg-quick-section">
            <span class="pg-micro-label">{{ zh ? '常用/推荐模型' : 'Quick models' }}</span>
            <MdChipSet>
              <MdFilterChip
                v-for="item in quickModels"
                :key="item"
                :label="item"
                :model-value="model === item"
                @update:model-value="model = item"
              />
            </MdChipSet>
          </div>

          <MdTextField
            v-model="systemPrompt"
            type="textarea"
            :rows="2"
            :label="zh ? 'System Prompt（可选）' : 'System prompt (optional)'"
            :placeholder="zh ? '设定角色、语气或输出格式要求…' : 'Define role, style, or output format…'"
          />

          <div class="pg-quick-section">
            <span class="pg-micro-label">{{ zh ? '快捷测试场景' : 'Prompt presets' }}</span>
            <MdChipSet>
              <MdFilterChip
                v-for="preset in promptPresets"
                :key="preset.id"
                :label="preset.label"
                :model-value="activePresetId === preset.id"
                @update:model-value="applyPreset(preset)"
              />
            </MdChipSet>
          </div>

          <MdTextField
            v-model="userPrompt"
            type="textarea"
            :rows="4"
            :label="zh ? 'User Prompt' : 'User prompt'"
            :placeholder="zh ? '输入你想测试的问题或指令…' : 'Enter a prompt to test…'"
          />

          <div class="pg-sliders">
            <MdSlider
              v-model="temperature"
              :min="0"
              :max="2"
              :step="0.1"
              :label="zh ? '随机度 (Temperature)' : 'Temperature'"
            />
            <MdSlider
              v-model="maxTokens"
              :min="64"
              :max="4096"
              :step="64"
              :label="zh ? '最大输出 (Max Tokens)' : 'Max tokens'"
            />
          </div>

          <div class="pg-switch-row">
            <MdSwitch
              v-model="stream"
              :label="zh ? '启用 SSE 流式输出' : 'Stream SSE response'"
              :description="zh ? '开启后可精准测量首字延迟 (TTFT)' : 'Enables accurate Time-To-First-Token measurement'"
            />
          </div>

          <div class="pg-actions">
            <MdButton
              v-if="!running"
              variant="filled"
              :disabled="!canRun"
              @click="runRequest"
            >
              <template #icon><Icon name="play" size="sm" /></template>
              {{ zh ? '发送测试请求' : 'Send request' }}
            </MdButton>
            <MdButton
              v-else
              variant="tonal"
              tone="danger"
              @click="stopRequest"
            >
              <template #icon><Icon name="x" size="sm" /></template>
              {{ zh ? '中止生成' : 'Stop' }}
            </MdButton>
            <MdButton variant="text" :disabled="running" @click="clearOutput">
              {{ zh ? '清空输出' : 'Clear' }}
            </MdButton>
          </div>
        </div>
      </MdCard>

      <!-- Right: Live telemetry & output inspector -->
      <MdCard variant="elevated" padding="md" class="playground-output">
        <template #header>
          <div class="pg-output-head">
            <div>
              <h2 class="md-card-title">{{ zh ? '响应与观测面板' : 'Response & Telemetry' }}</h2>
              <p class="md-card-subtitle">
                <code>{{ protocol === 'anthropic' ? 'POST /v1/messages' : 'POST /v1/chat/completions' }}</code>
              </p>
            </div>
            <div class="pg-metrics-strip">
              <div class="pg-metric">
                <span>TTFT</span>
                <strong>{{ ttftMs !== null ? `${ttftMs} ms` : '—' }}</strong>
              </div>
              <div class="pg-metric">
                <span>{{ zh ? '总耗时' : 'Total' }}</span>
                <strong>{{ totalMs !== null ? `${totalMs} ms` : '—' }}</strong>
              </div>
              <div class="pg-metric">
                <span>Tokens</span>
                <strong>{{ usage.totalTokens > 0 ? `${usage.inputTokens} / ${usage.outputTokens}` : '—' }}</strong>
              </div>
              <div class="pg-metric">
                <span>{{ zh ? '速率' : 'Speed' }}</span>
                <strong>{{ tokensPerSecond !== null ? `${tokensPerSecond} tok/s` : '—' }}</strong>
              </div>
            </div>
          </div>
        </template>

        <MdProgress v-if="running" variant="linear" indeterminate class="pg-progress" />

        <div v-if="billingInfo" class="pg-billing-banner">
          <span>
            {{ zh ? '分组倍率' : 'Group' }}: <strong>{{ formatMultiplier(billingInfo.group_rate_multiplier) }}</strong>
            <template v-if="billingInfo.user_rate_multiplier != null">
              · {{ zh ? '个人覆盖' : 'Personal' }}: <strong>{{ formatMultiplier(billingInfo.user_rate_multiplier) }}</strong>
            </template>
            <template v-if="billingInfo.peak_rate_enabled && billingInfo.peak_start && billingInfo.peak_end">
              · {{ zh ? '高峰时段' : 'Peak window' }}: <strong>{{ billingInfo.peak_start }}–{{ billingInfo.peak_end }}</strong>
            </template>
          </span>
          <MdBadge :tone="peakActive ? 'warn' : 'success'">
            {{ zh ? '当前生效' : 'Active' }} {{ formatMultiplier(billingInfo.effective_rate_multiplier) }}
          </MdBadge>
        </div>

        <MdTabs v-model="outputTab" :items="outputTabs" variant="secondary" />

        <div v-if="errorMessage" class="pg-error-box" role="alert">
          <strong>{{ zh ? '请求失败' : 'Request failed' }}</strong>
          <p>{{ errorMessage }}</p>
        </div>

        <!-- Tab 1: Rendered text output -->
        <div v-if="outputTab === 'response'" class="pg-panel">
          <div v-if="outputText" class="pg-response-body">
            <pre class="pg-response-pre">{{ outputText }}</pre>
            <div class="pg-panel-foot">
              <MdButton variant="text" size="sm" @click="copyText(outputText)">
                {{ zh ? '复制回复文本' : 'Copy output' }}
              </MdButton>
            </div>
          </div>
          <MdEmptyState
            v-else
            compact
            :title="running ? (zh ? '正在等待首字返回…' : 'Waiting for first token…') : (zh ? '尚未发起请求' : 'No response yet')"
            :description="zh ? '在左侧选择 API Key 与模型，点击「发送测试请求」即可实时查看流式回复与延迟指标。' : 'Select an API key and model on the left, then click Send request to inspect live output and latency.'"
          />
        </div>

        <!-- Tab 2: Raw JSON -->
        <div v-else-if="outputTab === 'json'" class="pg-panel pg-json-grid">
          <div class="pg-code-block">
            <div class="pg-code-head">
              <span>Request Body</span>
              <MdButton variant="text" size="sm" @click="copyText(formattedRequestJson)">
                {{ zh ? '复制' : 'Copy' }}
              </MdButton>
            </div>
            <pre>{{ formattedRequestJson }}</pre>
          </div>
          <div class="pg-code-block">
            <div class="pg-code-head">
              <span>{{ zh ? '响应原始负载' : 'Raw Response Payload' }}</span>
              <MdButton v-if="formattedResponseJson" variant="text" size="sm" @click="copyText(formattedResponseJson)">
                {{ zh ? '复制' : 'Copy' }}
              </MdButton>
            </div>
            <pre>{{ formattedResponseJson || (zh ? '// 发送请求后在此展示原始响应 JSON' : '// Send a request to inspect raw JSON') }}</pre>
          </div>
        </div>

        <!-- Tab 3: Code snippets -->
        <div v-else class="pg-panel">
          <div class="pg-snippet-bar">
            <MdChipSet>
              <MdFilterChip label="cURL" :model-value="snippetLang === 'curl'" @update:model-value="snippetLang = 'curl'" />
              <MdFilterChip label="Python" :model-value="snippetLang === 'python'" @update:model-value="snippetLang = 'python'" />
              <MdFilterChip label="Node.js" :model-value="snippetLang === 'node'" @update:model-value="snippetLang = 'node'" />
            </MdChipSet>
            <MdButton variant="tonal" size="sm" @click="copyActiveSnippet">
              {{ zh ? '复制代码' : 'Copy code' }}
            </MdButton>
          </div>
          <div class="pg-code-block">
            <pre>{{ activeSnippet }}</pre>
          </div>
        </div>
      </MdCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  MdBadge,
  MdButton,
  MdCard,
  MdChipSet,
  MdEmptyState,
  MdFilterChip,
  MdPageHeader,
  MdProgress,
  MdSelect,
  MdSlider,
  MdSwitch,
  MdTabs,
  MdTextField,
  type MdSelectOption
} from '@/components/md'
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

const protocolOptions = computed<MdSelectOption[]>(() => [
  { value: 'openai', label: 'OpenAI (/v1/chat/completions)' },
  { value: 'anthropic', label: 'Anthropic (/v1/messages)' }
])

const keyOptions = computed<MdSelectOption[]>(() => {
  const list: MdSelectOption[] = apiKeys.value
    .filter((k) => k.status === 'active' && Boolean(k.key))
    .map((k) => ({
      value: String(k.id),
      label: `${k.name} (${k.group?.name || 'Default'})`,
      supportingText: `${k.key.slice(0, 8)}...${k.key.slice(-4)}`
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
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.playground-grid {
  display: grid;
  grid-template-columns: minmax(320px, 430px) minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.pg-stack {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.pg-row-2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.pg-quick-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pg-micro-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--md-sys-color-on-surface-variant);
}

.pg-sliders {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding: 12px;
  border-radius: 12px;
  background: var(--md-sys-color-surface-container-low);
}

.pg-switch-row {
  padding: 8px 4px;
}

.pg-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.pg-output-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}

.pg-metrics-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pg-metric {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 12px;
  border-radius: 10px;
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
  min-width: 78px;
}

.pg-metric span {
  font-size: 0.7rem;
  color: var(--md-sys-color-on-surface-variant);
}

.pg-metric strong {
  font-size: 0.85rem;
  font-family: var(--font-mono);
  color: var(--md-sys-color-on-surface);
}

.pg-progress {
  margin-bottom: 12px;
}

.pg-billing-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  border-radius: 10px;
  background: var(--md-sys-color-surface-container-low);
  font-size: 0.8125rem;
  color: var(--md-sys-color-on-surface-variant);
}

.pg-error-box {
  margin-top: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  background: var(--md-sys-color-error-container);
  color: var(--md-sys-color-on-error-container);
  font-size: 0.875rem;
}

.pg-error-box p {
  margin: 4px 0 0;
  word-break: break-word;
}

.pg-panel {
  margin-top: 14px;
}

.pg-response-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pg-response-pre {
  margin: 0;
  padding: 16px;
  min-height: 220px;
  max-height: 520px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  border-radius: 12px;
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
  font-family: var(--font-sans);
  font-size: 0.9rem;
  line-height: 1.6;
  color: var(--md-sys-color-on-surface);
}

.pg-panel-foot {
  display: flex;
  justify-content: flex-end;
}

.pg-json-grid {
  display: grid;
  gap: 12px;
}

.pg-snippet-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}

.pg-code-block {
  border-radius: 12px;
  border: 1px solid var(--md-sys-color-outline-variant);
  background: var(--md-sys-color-surface-container-low);
  overflow: hidden;
}

.pg-code-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--md-sys-color-on-surface-variant);
}

.pg-code-block pre {
  margin: 0;
  padding: 12px 14px;
  max-height: 360px;
  overflow: auto;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  line-height: 1.55;
  color: var(--md-sys-color-on-surface);
}

@media (max-width: 1024px) {
  .playground-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 640px) {
  .pg-row-2,
  .pg-sliders {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
