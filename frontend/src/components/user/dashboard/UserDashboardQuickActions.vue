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
    <header><p class="portal-eyebrow">MAKE SOMETHING</p><h2>{{ tx('下一步，开始创造。', 'Make your next move.') }}</h2><p>{{ tx('把想法接入应用，其余交给模型。', 'Connect your idea. Let the models do the rest.') }}</p></header>
    <InkSurface as="button" interactive type="button" class="workspace-action-card workspace-action-primary" @click="router.push('/keys?create=1')"><span class="workspace-action-icon"><Icon name="plus" /></span><span class="workspace-action-copy"><strong>{{ tx('创建 API 密钥', 'Create an API key') }}</strong><small>{{ tx('为你的应用建立连接', 'Connect your application') }}</small></span><Icon name="arrowRight" size="sm" /></InkSurface>
    <InkSurface as="button" interactive type="button" class="workspace-action-card" @click="copyToClipboard(openAIBase)"><span class="workspace-action-icon"><Icon :name="copied ? 'check' : 'copy'" /></span><span><strong aria-live="polite">{{ copied ? tx('接口地址已复制', 'Endpoint copied') : tx('复制接口地址', 'Copy API endpoint') }}</strong><small class="workspace-endpoint" :title="openAIBase">{{ openAIBase }}</small></span><Icon name="arrowRight" size="sm" /></InkSurface>
    <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="workspace-action-card"><span class="workspace-action-icon"><Icon name="book" /></span><span class="workspace-action-copy"><strong>{{ tx('接入文档', 'Documentation') }}</strong><small>{{ tx('从第一条请求开始', 'Start with your first request') }}</small></span><Icon name="externalLink" size="sm" /></a>
    <button v-else type="button" class="workspace-action-card" @click="router.push('/keys?guide=1')"><span class="workspace-action-icon"><Icon name="book" /></span><span class="workspace-action-copy"><strong>{{ tx('接入指南', 'Quick start') }}</strong><small>{{ tx('查看接口地址和调用示例', 'Base URLs and request examples') }}</small></span><Icon name="arrowRight" size="sm" /></button>
  </aside>
</template>
