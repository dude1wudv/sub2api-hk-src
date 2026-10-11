<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import InkSurface from '@/components/common/InkSurface.vue'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { sanitizeUrl } from '@/utils/url'
const router = useRouter()
const { locale } = useI18n()
const tx = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const app = useAppStore()
const { copied, copyToClipboard } = useClipboard()
const { openAIBase } = usePatrickApiBase()
const docUrl = computed(() => sanitizeUrl(app.docUrl))
</script>

<template>
  <aside class="workspace-launchpad">
    <header><h2>{{ tx('快捷操作', 'Quick actions') }}</h2></header>
    <div class="workspace-action-grid">
      <InkSurface as="button" interactive type="button" class="workspace-action-card workspace-action-primary" @click="router.push('/keys?create=1')"><span class="workspace-action-icon"><Icon name="plus" /></span><span class="workspace-action-copy"><strong>{{ tx('创建 API 密钥', 'Create an API key') }}</strong></span><Icon name="arrowRight" size="sm" /></InkSurface>
      <InkSurface as="button" interactive type="button" class="workspace-action-card" @click="copyToClipboard(openAIBase)"><span class="workspace-action-icon"><Icon :name="copied ? 'check' : 'copy'" /></span><span class="workspace-action-copy"><strong aria-live="polite">{{ copied ? tx('接口地址已复制', 'Endpoint copied') : tx('复制接口地址', 'Copy API endpoint') }}</strong><small class="workspace-endpoint" :title="openAIBase">{{ openAIBase }}</small></span><Icon name="arrowRight" size="sm" /></InkSurface>
      <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="workspace-action-card"><span class="workspace-action-icon"><Icon name="book" /></span><span class="workspace-action-copy"><strong>{{ tx('接入文档', 'Documentation') }}</strong></span><Icon name="externalLink" size="sm" /></a>
      <button v-else type="button" class="workspace-action-card" @click="router.push('/keys?guide=1')"><span class="workspace-action-icon"><Icon name="book" /></span><span class="workspace-action-copy"><strong>{{ tx('接入指南', 'Quick start') }}</strong></span><Icon name="arrowRight" size="sm" /></button>
    </div>
  </aside>
</template>
