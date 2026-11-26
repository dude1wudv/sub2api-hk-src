<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import TrafficPanel from '@/components/usage/TrafficPanel.vue'
import { UiTabs } from '@/ui'

const props = withDefaults(defineProps<{ admin?: boolean }>(), { admin: false })
const { locale } = useI18n()
const router = useRouter()
const zh = computed(() => locale.value.startsWith('zh'))

const tabs = computed(() => [
  { value: 'usage', label: zh.value ? 'Token 用量与明细' : 'Token usage & logs' },
  { value: 'traffic', label: zh.value ? '网络流量监控' : 'Network traffic' },
])

function handleTabChange(val: string) {
  if (val === 'usage') {
    void router.push('/usage')
  }
}
</script>

<template>
  <div class="mx-auto max-w-[1440px] space-y-6 pb-10">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ zh ? '流量记录' : 'Traffic records' }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ zh ? 'HTTP/SSE 应用流量实时统计与请求明细。' : 'Real-time HTTP/SSE application traffic statistics and request details.' }}
      </p>
    </header>

    <UiTabs
      v-if="!props.admin"
      model-value="traffic"
      :tabs="tabs"
      :aria-label="zh ? '用量与流量视图' : 'Usage and traffic views'"
      @update:model-value="handleTabChange"
    />

    <TrafficPanel :admin="props.admin" />
  </div>
</template>
