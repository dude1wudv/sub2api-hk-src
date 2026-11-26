<template>
  <BrandModelIcon v-if="resolvedKey" :model="resolvedKey" :size="size" />
  <BrandMark v-else-if="isComposite" :size="numericSize" class="text-[var(--atelier-ink)]" />
  <BrandModelIcon v-else :model="platform" :size="size" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BrandModelIcon from './BrandModelIcon.vue'
import BrandMark from './BrandMark.vue'

const props = withDefaults(
  defineProps<{
    platform?: string | null
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | string
  }>(),
  {
    platform: '',
    size: 'sm'
  }
)

const normalized = computed(() => (props.platform || '').trim().toLowerCase())
const isComposite = computed(() => normalized.value === 'composite' || normalized.value === 'antigravity')

const platformMap: Record<string, string> = {
  anthropic: 'claude',
  claude: 'claude',
  openai: 'openai',
  gemini: 'gemini',
  google: 'google',
  grok: 'grok',
  xai: 'grok',
  kimi: 'kimi',
  moonshot: 'kimi',
  zhipu: 'zhipu',
  minimax: 'minimax',
  stepfun: 'stepfun',
  deepseek: 'deepseek',
  qwen: 'qwen',
  doubao: 'doubao',
  mistral: 'mistral',
  bedrock: 'bedrock',
  vertexai: 'vertexai',
  cline: 'cline',
  command_code: 'cohere',
  opencode_go: 'openrouter',
  mirasim: 'openai'
}

const resolvedKey = computed(() => platformMap[normalized.value] || '')
const numericSize = computed(() => {
  switch (props.size) {
    case 'xs':
      return 14
    case 'sm':
      return 16
    case 'md':
      return 20
    case 'lg':
      return 24
    case 'xl':
      return 32
    default: {
      const parsed = Number.parseInt(String(props.size), 10)
      return Number.isFinite(parsed) ? parsed : 16
    }
  }
})
</script>
