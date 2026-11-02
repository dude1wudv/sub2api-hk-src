<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { useClipboard } from '@/composables/useClipboard'

defineProps<{ keyCount: number }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const { openAIBase } = usePatrickApiBase()
const { copied, copyToClipboard } = useClipboard()
</script>

<template>
  <section class="workspace-onboarding" aria-labelledby="onboarding-title">
    <h2 id="onboarding-title">{{ copy('跑通第一条请求', 'Make your first request') }}</h2>
    <p class="portal-muted">{{ copy('完成下面三步，即可把模型接入你的应用。', 'Follow these three steps to connect a model to your application.') }}</p>
    <ol>
      <li class="portal-panel"><h3>01 · {{ copy('创建 API 密钥', 'Create an API key') }}</h3><p>{{ keyCount > 0 ? copy('你已创建密钥，可在密钥页复制或管理。', 'Your key is ready. Copy or manage it on the keys page.') : copy('为应用创建一个密钥，并选择可用分组。', 'Create a key for your application and choose an available group.') }}</p><RouterLink class="portal-button" :to="keyCount > 0 ? '/keys' : '/keys?create=1'">{{ keyCount > 0 ? copy('管理密钥', 'Manage keys') : copy('创建首个密钥', 'Create your first key') }}</RouterLink></li>
      <li class="portal-panel"><h3>02 · {{ copy('复制 Base URL', 'Copy the Base URL') }}</h3><p>{{ copy('这是 OpenAI 兼容客户端使用的接入地址。', 'Use this address with an OpenAI-compatible client.') }}</p><code>{{ openAIBase }}</code><button type="button" class="portal-button" @click="copyToClipboard(openAIBase)">{{ copied ? copy('已复制', 'Copied') : copy('复制 Base URL', 'Copy Base URL') }}</button></li>
      <li class="portal-panel"><h3>03 · {{ copy('发送第一条请求', 'Send your first request') }}</h3><p>{{ copy('参考调用示例，使用分组支持的模型；成功请求会出现在用量页。', 'Follow an example with a model supported by your group. Successful requests appear in Usage.') }}</p><RouterLink class="portal-button" to="/home#home-examples">{{ copy('查看调用示例', 'View request examples') }}</RouterLink></li>
    </ol>
  </section>
</template>

<style scoped>
.workspace-onboarding { margin:24px 0; }
h2 { margin:0 0 8px; font-size:22px; font-weight:600; }
ol { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:20px; margin:20px 0 0; padding:0; list-style:none; }
li { display:flex; flex-direction:column; align-items:flex-start; gap:16px; min-width:0; }
h3 { font-size:18px; font-weight:600; margin:0; }
li p { margin:0; color:rgb(var(--ink-2)); flex:1; }
code { overflow-wrap:anywhere; font-size:13px; }
@media(max-width:899px) { ol { grid-template-columns:1fr; } }
</style>
