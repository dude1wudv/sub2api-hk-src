<template>
  <div class="mx-auto max-w-[1440px] space-y-6 pb-10">
    <!-- Editorial Page Header -->
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
          {{ zh ? 'API 调试台' : 'API Playground' }}
        </h1>
        <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
          {{ zh ? '直接选用你的 API 密钥向网关发送测试请求，实时观测首字延迟、吞吐速率、原始 JSON 与当前生效倍率。' : 'Send live gateway requests with your API key to inspect TTFT, throughput, raw JSON payloads, and effective billing multipliers.' }}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <span
          v-if="billingInfo"
          class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium"
          :class="peakActive
            ? 'border-[var(--atelier-warn)]/30 bg-[var(--atelier-warn-bg)] text-[var(--atelier-warn)]'
            : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)]'"
        >
          <UiStatusDot :tone="peakActive ? 'warn' : 'ok'" />
          <span>{{ zh ? '生效倍率' : 'Effective' }}</span>
          <strong class="font-mono tabular-nums text-[var(--atelier-ink)]">
            {{ formatMultiplier(billingInfo.effective_rate_multiplier) }}
          </strong>
          <template v-if="peakActive"> · {{ zh ? '高峰期' : 'Peak' }}</template>
        </span>

        <UiButton variant="secondary" size="md" type="button" @click="router.push('/keys')">
          <Icon name="key" size="sm" aria-hidden="true" />
          {{ zh ? '管理密钥' : 'Manage keys' }}
        </UiButton>
        <UiButton variant="primary" size="md" type="button" @click="copyActiveSnippet">
          <Icon name="copy" size="sm" aria-hidden="true" />
          {{ zh ? '复制代码' : 'Copy snippet' }}
        </UiButton>
      </div>
    </header>

    <!-- Asymmetric Split Workspace -->
    <div class="grid grid-cols-1 items-start gap-5 xl:grid-cols-[440px_minmax(0,1fr)]">
      <!-- Left: Request Configuration -->
      <section class="atelier-surface overflow-hidden" :aria-label="zh ? '请求配置' : 'Request parameters'">
        <header class="border-b border-[var(--atelier-border)] px-5 py-4">
          <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ zh ? '请求配置' : 'Request Parameters' }}
          </h2>
          <p class="mt-0.5 text-xs text-[var(--atelier-ink-muted)]">
            {{ zh ? '密钥仅保存在当前内存会话中，不会写入本地存储。' : 'API keys stay in memory for this session only and are never saved to localStorage.' }}
          </p>
        </header>

        <div class="space-y-4 p-5">
          <!-- API Key Selector -->
          <div v-if="keyOptions.length > 0" class="space-y-1.5">
            <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ zh ? 'API 密钥' : 'API Key' }}
            </span>
            <Select
              v-model="selectedKeyId"
              :options="keyOptions"
              :searchable="keyOptions.length > 5"
              :placeholder="zh ? '选择 API 密钥' : 'Select API key'"
              :aria-label="zh ? '选择 API 密钥' : 'Select API key'"
            />
          </div>

          <label v-if="selectedKeyId === '__custom__' || keyOptions.length === 0" class="block space-y-1.5">
            <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ zh ? '临时测试 API Key' : 'Custom API Key' }}
            </span>
            <input
              v-model="customKey"
              type="password"
              class="atelier-input font-mono text-xs"
              autocomplete="off"
              :placeholder="zh ? '粘贴 sk-... 密钥（仅本次调试有效）' : 'Paste sk-... key (memory only)'"
            />
          </label>

          <!-- Protocol Switch -->
          <div class="space-y-1.5">
            <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ zh ? '网关协议' : 'Protocol' }}
            </span>
            <UiSegmented
              v-model="protocol"
              :options="protocolOptions"
              :aria-label="zh ? '网关协议' : 'Protocol'"
              class="w-full [&>button]:flex-1"
            >
              <template #option="{ option }">
                <span class="flex flex-col items-center leading-tight py-0.5">
                  <span class="text-xs font-semibold">{{ option.label }}</span>
                  <code class="mt-0.5 font-mono text-[10px] opacity-70">
                    {{ option.value === 'openai' ? '/v1/chat/completions' : '/v1/messages' }}
                  </code>
                </span>
              </template>
            </UiSegmented>
          </div>

          <!-- Model ID -->
          <label class="block space-y-1.5">
            <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ zh ? '模型名称' : 'Model ID' }}
            </span>
            <input
              v-model="model"
              type="text"
              class="atelier-input font-mono text-xs"
              placeholder="gpt-4o / claude-sonnet-4-5"
            />
          </label>

          <!-- Quick Model Chips -->
          <div v-if="quickModels.length" class="space-y-1.5">
            <span class="block text-[11px] font-medium text-[var(--atelier-ink-muted)]">
              {{ zh ? '常用 / 推荐模型' : 'Quick models' }}
            </span>
            <div class="flex flex-wrap gap-1.5" role="group" :aria-label="zh ? '常用模型' : 'Quick models'">
              <button
                v-for="item in quickModels"
                :key="item"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-mono text-xs transition-colors"
                :class="model === item
                  ? 'border-[var(--atelier-ink)] bg-[var(--atelier-ink)] text-[var(--atelier-ink-inverse)]'
                  : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)] hover:border-[var(--atelier-border-strong)] hover:text-[var(--atelier-ink)]'"
                :aria-pressed="model === item"
                @click="model = item"
              >
                <BrandModelIcon :model="item" size="xs" />
                <span>{{ item }}</span>
              </button>
            </div>
          </div>

          <!-- System Prompt -->
          <label class="block space-y-1.5">
            <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ zh ? 'System Prompt（可选）' : 'System prompt (optional)' }}
            </span>
            <textarea
              v-model="systemPrompt"
              rows="2"
              class="atelier-input resize-y text-xs leading-relaxed"
              :placeholder="zh ? '设定角色、语气或输出格式要求…' : 'Define role, style, or output format…'"
            />
          </label>

          <!-- Prompt Presets -->
          <div class="space-y-1.5">
            <span class="block text-[11px] font-medium text-[var(--atelier-ink-muted)]">
              {{ zh ? '快捷测试场景' : 'Prompt presets' }}
            </span>
            <div class="flex flex-wrap gap-1.5" role="group" :aria-label="zh ? '快捷测试场景' : 'Prompt presets'">
              <button
                v-for="preset in promptPresets"
                :key="preset.id"
                type="button"
                class="inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs transition-colors"
                :class="activePresetId === preset.id
                  ? 'border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] font-medium text-[var(--atelier-ink)]'
                  : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)] hover:border-[var(--atelier-border-strong)] hover:text-[var(--atelier-ink)]'"
                :aria-pressed="activePresetId === preset.id"
                @click="applyPreset(preset)"
              >
                <Icon v-if="activePresetId === preset.id" name="check" size="xs" aria-hidden="true" />
                <span>{{ preset.label }}</span>
              </button>
            </div>
          </div>

          <!-- User Prompt -->
          <label class="block space-y-1.5">
            <span class="flex items-center justify-between text-xs font-medium text-[var(--atelier-ink-secondary)]">
              <span>User Prompt</span>
              <small class="font-normal text-[11px] text-[var(--atelier-ink-muted)]">
                {{ zh ? 'Ctrl / ⌘ + Enter 发送' : 'Ctrl / ⌘ + Enter to send' }}
              </small>
            </span>
            <textarea
              v-model="userPrompt"
              rows="4"
              class="atelier-input resize-y text-sm leading-relaxed"
              :placeholder="zh ? '输入你想测试的问题或指令…' : 'Enter a prompt to test…'"
              @keydown.ctrl.enter.prevent="runRequest"
              @keydown.meta.enter.prevent="runRequest"
            />
          </label>

          <!-- Sliders -->
          <div class="atelier-surface-subtle space-y-3.5 p-4">
            <UiSlider
              v-model="temperature"
              :min="0"
              :max="2"
              :step="0.1"
              :label="zh ? '随机度 (Temperature)' : 'Temperature'"
            >
              <template #value>{{ temperature.toFixed(1) }}</template>
            </UiSlider>

            <UiSlider
              v-model="maxTokens"
              :min="64"
              :max="4096"
              :step="64"
              :label="zh ? '最大输出 (Max Tokens)' : 'Max tokens'"
            >
              <template #value>{{ maxTokens }}</template>
            </UiSlider>
          </div>

          <!-- SSE Switch -->
          <div class="flex items-center justify-between gap-3 rounded-xl border border-[var(--atelier-border)] px-4 py-3">
            <div>
              <strong class="block text-xs font-semibold text-[var(--atelier-ink)]">
                {{ zh ? '启用 SSE 流式输出' : 'Stream SSE response' }}
              </strong>
              <small class="text-[11px] text-[var(--atelier-ink-muted)]">
                {{ zh ? '开启后可精准测量首字延迟 (TTFT)' : 'Enables accurate Time-To-First-Token measurement' }}
              </small>
            </div>
            <UiSwitch v-model="stream" :aria-label="zh ? '启用 SSE 流式输出' : 'Stream SSE response'" />
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-2 pt-1">
            <UiButton
              v-if="!running"
              variant="primary"
              size="lg"
              type="button"
              class="flex-1"
              :disabled="!canRun"
              @click="runRequest"
            >
              <Icon name="play" size="sm" aria-hidden="true" />
              {{ zh ? '发送测试请求' : 'Send request' }}
            </UiButton>
            <UiButton
              v-else
              variant="danger"
              size="lg"
              type="button"
              class="flex-1"
              @click="stopRequest"
            >
              <Icon name="x" size="sm" aria-hidden="true" />
              {{ zh ? '中止生成' : 'Stop' }}
            </UiButton>
            <UiButton
              variant="secondary"
              size="lg"
              type="button"
              :disabled="running"
              @click="clearOutput"
            >
              {{ zh ? '清空输出' : 'Clear' }}
            </UiButton>
          </div>
        </div>
      </section>

      <!-- Right: Live Telemetry & Output Inspector -->
      <section class="atelier-surface flex min-h-[640px] flex-col overflow-hidden" :aria-label="zh ? '响应与观测面板' : 'Response and telemetry'">
        <header class="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--atelier-border)] px-5 py-4">
          <div class="flex flex-wrap items-center gap-2.5">
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ zh ? '响应与观测面板' : 'Response & Telemetry' }}
            </h2>
            <span class="inline-flex items-center gap-1.5 rounded-md border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono text-[11px]">
              <span class="font-semibold text-[var(--atelier-ink)]">POST</span>
              <code class="text-[var(--atelier-ink-secondary)]">{{ protocol === 'anthropic' ? '/v1/messages' : '/v1/chat/completions' }}</code>
            </span>
          </div>

          <span
            v-if="running"
            class="inline-flex items-center gap-1.5 rounded-full bg-[var(--atelier-info-bg)] px-2.5 py-1 text-xs font-medium text-[var(--atelier-info)]"
            role="status"
          >
            <UiStatusDot tone="info" class="animate-pulse" />
            {{ zh ? '流式接收中…' : 'Streaming…' }}
          </span>
        </header>

        <!-- Slim Indeterminate Bar when Running -->
        <div v-if="running" class="h-[2px] w-full overflow-hidden bg-[var(--atelier-subtle)]" aria-hidden="true">
          <div class="pg-progress-indeterminate h-full w-1/3 bg-[var(--atelier-ink)]" />
        </div>

        <!-- Horizontal Borderless Telemetry Strip -->
        <div class="grid grid-cols-2 divide-y divide-[var(--atelier-border)] border-b border-[var(--atelier-border)] sm:grid-cols-4 sm:divide-x sm:divide-y-0">
          <div class="px-5 py-3.5">
            <span class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ zh ? '首字延迟 (TTFT)' : 'TTFT' }}</span>
            <strong class="mt-0.5 block font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
              {{ ttftMs !== null ? `${ttftMs} ms` : '—' }}
            </strong>
          </div>
          <div class="px-5 py-3.5">
            <span class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ zh ? '总耗时' : 'Total time' }}</span>
            <strong class="mt-0.5 block font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
              {{ totalMs !== null ? `${totalMs} ms` : '—' }}
            </strong>
          </div>
          <div class="px-5 py-3.5">
            <span class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ zh ? 'Token 消耗 (入/出)' : 'Tokens (in/out)' }}</span>
            <strong class="mt-0.5 block font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
              {{ usage.totalTokens > 0 ? `${usage.inputTokens} / ${usage.outputTokens}` : '—' }}
            </strong>
          </div>
          <div class="px-5 py-3.5">
            <span class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ zh ? '生成速率' : 'Throughput' }}</span>
            <strong class="mt-0.5 block font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
              {{ tokensPerSecond !== null ? `${tokensPerSecond} tok/s` : '—' }}
            </strong>
          </div>
        </div>

        <!-- Billing Multiplier Info Bar -->
        <div
          v-if="billingInfo"
          class="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/55 px-5 py-2.5 text-xs"
        >
          <div class="flex flex-wrap items-center gap-2 text-[var(--atelier-ink-secondary)]">
            <span>
              {{ zh ? '分组倍率' : 'Group' }}
              <strong class="font-mono tabular-nums text-[var(--atelier-ink)]">{{ formatMultiplier(billingInfo.group_rate_multiplier) }}</strong>
            </span>
            <template v-if="billingInfo.user_rate_multiplier != null">
              <span class="text-[var(--atelier-ink-muted)]">·</span>
              <span>
                {{ zh ? '个人覆盖' : 'Personal' }}
                <strong class="font-mono tabular-nums text-[var(--atelier-ink)]">{{ formatMultiplier(billingInfo.user_rate_multiplier) }}</strong>
              </span>
            </template>
            <template v-if="billingInfo.peak_rate_enabled && billingInfo.peak_start && billingInfo.peak_end">
              <span class="text-[var(--atelier-ink-muted)]">·</span>
              <span>
                {{ zh ? '高峰时段' : 'Peak window' }}
                <strong class="font-mono tabular-nums text-[var(--atelier-ink)]">{{ billingInfo.peak_start }}–{{ billingInfo.peak_end }}</strong>
              </span>
            </template>
          </div>
          <span
            class="rounded-full px-2 py-0.5 font-mono text-[11px] font-semibold tabular-nums"
            :class="peakActive
              ? 'bg-[var(--atelier-warn-bg)] text-[var(--atelier-warn)]'
              : 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]'"
          >
            {{ zh ? '当前生效' : 'Active' }} {{ formatMultiplier(billingInfo.effective_rate_multiplier) }}
          </span>
        </div>

        <!-- Output Underline Tabs -->
        <div class="px-5 pt-2">
          <UiTabs
            v-model="outputTab"
            :tabs="outputTabs"
            :aria-label="zh ? '响应视图' : 'Output views'"
          />
        </div>

        <!-- Error Banner -->
        <div
          v-if="errorMessage"
          class="mx-5 mt-4 flex items-start gap-2.5 rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-xs text-[var(--atelier-err)]"
          role="alert"
        >
          <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" aria-hidden="true" />
          <div class="min-w-0">
            <strong class="block font-semibold">{{ zh ? '请求失败' : 'Request failed' }}</strong>
            <p class="mt-0.5 break-words">{{ errorMessage }}</p>
          </div>
        </div>

        <!-- Tab 1: Live Text Output -->
        <div v-if="outputTab === 'response'" class="flex flex-1 flex-col p-5">
          <div v-if="outputText" class="flex flex-1 flex-col gap-3">
            <pre class="min-h-[340px] flex-1 overflow-auto whitespace-pre-wrap break-words rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/50 p-4 font-sans text-sm leading-relaxed text-[var(--atelier-ink)]">{{ outputText }}</pre>
            <div class="flex justify-end">
              <UiButton variant="secondary" size="sm" type="button" @click="copyText(outputText)">
                <Icon name="copy" size="xs" aria-hidden="true" />
                {{ zh ? '复制回复文本' : 'Copy output' }}
              </UiButton>
            </div>
          </div>

          <div v-else class="relative flex flex-1 flex-col items-center justify-center overflow-hidden rounded-xl border border-dashed border-[var(--atelier-border-strong)] px-6 py-16 text-center">
            <DotMatrixBackdrop mask="center" class="opacity-60" />
            <div class="relative z-10 flex flex-col items-center">
              <div class="mb-3 flex h-11 w-11 items-center justify-center rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)]" aria-hidden="true">
                <Icon name="terminal" size="md" />
              </div>
              <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">
                {{ running ? (zh ? '正在等待首字返回…' : 'Waiting for first token…') : (zh ? '尚未发起请求' : 'No response yet') }}
              </h3>
              <p class="mt-1.5 max-w-md text-xs leading-relaxed text-[var(--atelier-ink-muted)]">
                {{ zh ? '在左侧选择 API Key 与模型，点击「发送测试请求」即可实时查看流式回复与延迟指标。' : 'Select an API key and model on the left, then click Send request to inspect live output and latency.' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Tab 2: Raw JSON -->
        <div v-else-if="outputTab === 'json'" class="grid flex-1 grid-cols-1 gap-4 p-5 lg:grid-cols-2">
          <div class="flex flex-col overflow-hidden rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/50">
            <div class="flex items-center justify-between border-b border-[var(--atelier-border)] px-3.5 py-2 text-xs font-medium text-[var(--atelier-ink-secondary)]">
              <span>Request Body</span>
              <button type="button" class="font-medium text-[var(--atelier-ink)] hover:underline" @click="copyText(formattedRequestJson)">
                {{ zh ? '复制' : 'Copy' }}
              </button>
            </div>
            <pre class="max-h-[440px] flex-1 overflow-auto p-3.5 font-mono text-xs leading-relaxed text-[var(--atelier-ink)]">{{ formattedRequestJson }}</pre>
          </div>

          <div class="flex flex-col overflow-hidden rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/50">
            <div class="flex items-center justify-between border-b border-[var(--atelier-border)] px-3.5 py-2 text-xs font-medium text-[var(--atelier-ink-secondary)]">
              <span>{{ zh ? '响应原始负载' : 'Raw Response Payload' }}</span>
              <button v-if="formattedResponseJson" type="button" class="font-medium text-[var(--atelier-ink)] hover:underline" @click="copyText(formattedResponseJson)">
                {{ zh ? '复制' : 'Copy' }}
              </button>
            </div>
            <pre class="max-h-[440px] flex-1 overflow-auto p-3.5 font-mono text-xs leading-relaxed text-[var(--atelier-ink)]">{{ formattedResponseJson || (zh ? '// 发送请求后在此展示原始响应 JSON' : '// Send a request to inspect raw JSON') }}</pre>
          </div>
        </div>

        <!-- Tab 3: Code Snippets -->
        <div v-else class="flex flex-1 flex-col gap-3 p-5">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <UiSegmented
              v-model="snippetLang"
              :options="snippetOptions"
              size="sm"
              :aria-label="zh ? '代码语言' : 'Snippet language'"
            />
            <UiButton variant="secondary" size="sm" type="button" @click="copyActiveSnippet">
              <Icon name="copy" size="xs" aria-hidden="true" />
              {{ zh ? '复制代码' : 'Copy code' }}
            </UiButton>
          </div>
          <div class="flex-1 overflow-hidden rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/50">
            <pre class="max-h-[460px] overflow-auto p-4 font-mono text-xs leading-relaxed text-[var(--atelier-ink)]">{{ activeSnippet }}</pre>
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
import Select, { type SelectOption } from '@/components/common/Select.vue'
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
import {
  BrandModelIcon,
  DotMatrixBackdrop,
  UiButton,
  UiSegmented,
  UiSlider,
  UiStatusDot,
  UiSwitch,
  UiTabs
} from '@/ui'

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

const protocolOptions = [
  { value: 'openai' as const, label: 'OpenAI' },
  { value: 'anthropic' as const, label: 'Anthropic' },
]

const snippetOptions = [
  { value: 'curl' as const, label: 'cURL' },
  { value: 'python' as const, label: 'Python' },
  { value: 'node' as const, label: 'Node.js' },
]

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
  { value: 'response', label: zh.value ? '实时输出' : 'Response' },
  { value: 'json', label: 'Request / JSON' },
  { value: 'snippet', label: zh.value ? '导出代码' : 'Code Snippets' }
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
.pg-progress-indeterminate {
  animation: pg-slide 1.4s ease-in-out infinite;
}

@keyframes pg-slide {
  0% { transform: translateX(-100%); }
  50% { transform: translateX(120%); }
  100% { transform: translateX(300%); }
}
</style>
