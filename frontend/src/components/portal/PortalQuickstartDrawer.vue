<template>
  <MdDialog
    :open="open"
    :headline="zh ? '一站式 API 接入中心' : 'API Quickstart Hub'"
    max-width="760px"
    @update:open="(val) => { if (!val) emit('close') }"
  >
    <div class="qs-body">
      <div class="qs-step-row">
        <div class="qs-key-picker">
          <MdSelect
            v-if="activeKeyOptions.length"
            v-model="selectedKeyId"
            :label="zh ? '步骤 1：选择 API 密钥' : 'Step 1: Choose API key'"
            :options="activeKeyOptions"
          />
          <p v-else class="qs-no-key">
            {{ zh ? '还没有可用密钥，可先关闭弹窗创建一把 API Key，下方将以占位符生成示例配置。' : 'No active key yet. Sample configs below use a placeholder key.' }}
          </p>
        </div>

        <div class="qs-ping-box">
          <MdButton
            variant="tonal"
            size="sm"
            :loading="testing"
            :disabled="!selectedKeySecret"
            @click="testConnectivity"
          >
            <template #icon><Icon name="bolt" size="xs" /></template>
            {{ zh ? '在线连通性自检' : 'Test key connectivity' }}
          </MdButton>
          <MdSwitch
            v-model="revealSecret"
            :label="zh ? '填入明文密钥' : 'Include real key'"
          />
        </div>
      </div>

      <div v-if="pingStatus" class="qs-ping-result" :class="`qs-ping-result--${pingStatus.ok ? 'ok' : 'err'}`" role="status">
        <div>
          <strong>{{ pingStatus.ok ? (zh ? '密钥连通正常' : 'Key verified') : (zh ? '连通性检查失败' : 'Verification failed') }}</strong>
          <span> · {{ pingStatus.latencyMs }} ms</span>
          <span v-if="pingStatus.detail"> · {{ pingStatus.detail }}</span>
        </div>
        <MdBadge v-if="pingStatus.effectiveMultiplier" :tone="pingStatus.peak ? 'warn' : 'primary'">
          {{ zh ? '生效倍率' : 'Effective' }} {{ pingStatus.effectiveMultiplier }}
        </MdBadge>
      </div>

      <MdTabs v-model="clientTab" :items="clientTabs" variant="secondary" />

      <div class="qs-code-card">
        <div class="qs-code-head">
          <span>{{ activeClientTitle }}</span>
          <div class="qs-code-actions">
            <MdButton variant="text" size="sm" @click="copyConfig">
              <template #icon><Icon name="copy" size="xs" /></template>
              {{ zh ? '复制配置' : 'Copy' }}
            </MdButton>
            <MdButton
              v-if="selectedKeyId"
              variant="text"
              size="sm"
              @click="openInPlayground"
            >
              <template #icon><Icon name="play" size="xs" /></template>
              {{ zh ? '去调试台试跑' : 'Try in Playground' }}
            </MdButton>
          </div>
        </div>
        <pre class="qs-pre">{{ activeConfigCode }}</pre>
      </div>
    </div>

    <template #actions>
      <MdButton variant="text" @click="emit('close')">
        {{ zh ? '关闭' : 'Close' }}
      </MdButton>
      <MdButton variant="filled" @click="copyConfig">
        {{ zh ? '复制当前配置' : 'Copy config' }}
      </MdButton>
    </template>
  </MdDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MdBadge, MdButton, MdDialog, MdSelect, MdSwitch, MdTabs, type MdSelectOption } from '@/components/md'
import Icon from '@/components/icons/Icon.vue'
import { formatMultiplier, getKeyBillingInfo, isPeakRateActive } from '@/api/keyBilling'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'

const props = defineProps<{
  open: boolean
  keys: ApiKey[]
  baseUrl: string
  initialKeyId?: number | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const router = useRouter()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const selectedKeyId = ref('')
const revealSecret = ref(false)
const clientTab = ref('claude-code')
const testing = ref(false)
const pingStatus = ref<{
  ok: boolean
  latencyMs: number
  detail: string
  effectiveMultiplier?: string
  peak?: boolean
} | null>(null)

const activeKeys = computed(() => props.keys.filter((k) => k.status === 'active' && Boolean(k.key)))

const activeKeyOptions = computed<MdSelectOption[]>(() =>
  activeKeys.value.map((k) => ({
    value: String(k.id),
    label: `${k.name}${k.group?.name ? ` (${k.group.name})` : ''}`,
    supportingText: `${k.key.slice(0, 8)}...${k.key.slice(-4)}`
  }))
)

watch(
  () => [props.open, props.initialKeyId, activeKeys.value.length] as const,
  ([isOpen, initId]) => {
    if (!isOpen) return
    pingStatus.value = null
    if (initId && activeKeys.value.some((k) => k.id === initId)) {
      selectedKeyId.value = String(initId)
    } else if (!selectedKeyId.value && activeKeys.value.length) {
      selectedKeyId.value = String(activeKeys.value[0].id)
    }
  },
  { immediate: true }
)

const selectedKey = computed(() => activeKeys.value.find((k) => String(k.id) === selectedKeyId.value) || null)
const selectedKeySecret = computed(() => selectedKey.value?.key || '')

const displayKey = computed(() => {
  if (!selectedKeySecret.value) return 'YOUR_API_KEY'
  if (revealSecret.value) return selectedKeySecret.value
  return `${selectedKeySecret.value.slice(0, 10)}••••••••${selectedKeySecret.value.slice(-4)}`
})

const cleanOrigin = computed(() => (props.baseUrl || window.location.origin).replace(/\/v1\/?$/, '').replace(/\/+$/, ''))
const openAiBase = computed(() => `${cleanOrigin.value}/v1`)

const clientTabs = computed(() => [
  { id: 'claude-code', label: 'Claude Code' },
  { id: 'codex', label: 'Codex CLI' },
  { id: 'cursor', label: 'Cursor / IDE' },
  { id: 'openai-py', label: 'OpenAI Python' },
  { id: 'anthropic', label: 'Anthropic SDK' },
  { id: 'curl', label: 'cURL' }
])

const activeClientTitle = computed(() => {
  const found = clientTabs.value.find((t) => t.id === clientTab.value)
  return found?.label || clientTab.value
})

const activeConfigCode = computed(() => {
  const key = displayKey.value
  const origin = cleanOrigin.value
  const v1 = openAiBase.value

  switch (clientTab.value) {
    case 'claude-code':
      return `# ~/.claude/settings.json 或 shell 环境变量\nexport ANTHROPIC_BASE_URL="${origin}"\nexport ANTHROPIC_AUTH_TOKEN="${key}"\n\n# 启动 Claude Code\nclaude`
    case 'codex':
      return `# ~/.codex/config.toml\nmodel_provider = "sub2api"\n\n[model_providers.sub2api]\nname = "Sub2API"\nbase_url = "${v1}"\nenv_key = "OPENAI_API_KEY"\nwire_api = "responses"\n\n# 终端导出密钥\nexport OPENAI_API_KEY="${key}"`
    case 'cursor':
      return `OpenAI Base URL:\n${v1}\n\nAPI Key:\n${key}\n\n# 在 Cursor / Cherry Studio / OpenWebUI 的「OpenAI 兼容接口」中填入以上两项即可使用。`
    case 'openai-py':
      return `from openai import OpenAI\n\nclient = OpenAI(\n    api_key="${key}",\n    base_url="${v1}",\n)\n\nresp = client.chat.completions.create(\n    model="gpt-4o-mini",\n    messages=[{"role": "user", "content": "Hello!"}],\n)\nprint(resp.choices[0].message.content)`
    case 'anthropic':
      return `import anthropic\n\nclient = anthropic.Anthropic(\n    api_key="${key}",\n    base_url="${origin}",\n)\n\nmsg = client.messages.create(\n    model="claude-sonnet-4-5",\n    max_tokens=512,\n    messages=[{"role": "user", "content": "Hello!"}],\n)\nprint(msg.content[0].text)`
    default:
      return `curl -X POST "${v1}/chat/completions" \\\n  -H "Content-Type: application/json" \\\n  -H "Authorization: Bearer ${key}" \\\n  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'`
  }
})

async function testConnectivity() {
  if (!selectedKeySecret.value || testing.value) return
  testing.value = true
  pingStatus.value = null
  const started = performance.now()
  try {
    const res = await getKeyBillingInfo(selectedKeySecret.value)
    const latencyMs = Math.round(performance.now() - started)
    if (res.data) {
      const peak = isPeakRateActive(res.data)
      pingStatus.value = {
        ok: true,
        latencyMs,
        effectiveMultiplier: formatMultiplier(res.data.effective_rate_multiplier),
        peak,
        detail: zh.value
          ? `分组倍率 ${formatMultiplier(res.data.group_rate_multiplier)}${peak ? '（当前为高峰时段）' : ''}`
          : `Group ${formatMultiplier(res.data.group_rate_multiplier)}${peak ? ' (peak active)' : ''}`
      }
      return
    }

    // Fallback to /v1/models check for simple mode where /v1/sub2api/billing returns 404
    const fallback = await fetch('/v1/models', {
      headers: { Authorization: `Bearer ${selectedKeySecret.value}` }
    })
    const fbLatency = Math.round(performance.now() - started)
    if (fallback.ok) {
      pingStatus.value = {
        ok: true,
        latencyMs: fbLatency,
        detail: zh.value ? '已通过 /v1/models 鉴权校验' : 'Authenticated via /v1/models'
      }
    } else {
      pingStatus.value = {
        ok: false,
        latencyMs: fbLatency,
        detail: res.error || `HTTP ${fallback.status}`
      }
    }
  } catch (err: unknown) {
    pingStatus.value = {
      ok: false,
      latencyMs: Math.round(performance.now() - started),
      detail: err instanceof Error ? err.message : 'Network error'
    }
  } finally {
    testing.value = false
  }
}

async function copyConfig() {
  const rawCode = revealSecret.value
    ? activeConfigCode.value
    : activeConfigCode.value.split(displayKey.value).join(selectedKeySecret.value || 'YOUR_API_KEY')
  if (await copyToClipboard(rawCode)) {
    appStore.showSuccess(zh.value ? '配置已复制（已自动带入完整密钥）' : 'Config copied with full key')
  }
}

function openInPlayground() {
  emit('close')
  void router.push({ path: '/playground', query: selectedKeyId.value ? { keyId: selectedKeyId.value } : undefined })
}
</script>

<style scoped>
.qs-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.qs-step-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}

.qs-key-picker {
  flex: 1 1 260px;
}

.qs-no-key {
  margin: 0;
  font-size: 0.85rem;
  color: var(--md-sys-color-on-surface-variant);
}

.qs-ping-box {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.qs-ping-result {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 0.825rem;
}

.qs-ping-result--ok {
  background: rgb(var(--status-ok-bg));
  color: rgb(var(--text-ok));
}

.qs-ping-result--err {
  background: var(--md-sys-color-error-container);
  color: var(--md-sys-color-on-error-container);
}

.qs-code-card {
  border-radius: 14px;
  border: 1px solid var(--md-sys-color-outline-variant);
  background: var(--md-sys-color-surface-container-low);
  overflow: hidden;
}

.qs-code-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--md-sys-color-on-surface-variant);
}

.qs-code-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.qs-pre {
  margin: 0;
  padding: 14px 16px;
  overflow: auto;
  max-height: 280px;
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  line-height: 1.55;
  color: var(--md-sys-color-on-surface);
}
</style>
