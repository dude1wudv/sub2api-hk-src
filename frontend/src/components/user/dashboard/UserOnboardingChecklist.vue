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
    <div class="workspace-onboarding-head">
      <span class="onboarding-kicker">{{ copy('快速接入', 'Quick setup') }}</span>
      <h2 id="onboarding-title">{{ copy('跑通第一条请求', 'Make your first request') }}</h2>
      <p class="portal-muted">{{ copy('完成下面三步，即可把模型接入你的应用。', 'Follow these three steps to connect a model to your application.') }}</p>
    </div>
    <ol>
      <li class="portal-panel" :data-done="keyCount > 0">
        <div class="onboarding-step-top">
          <span class="onboarding-step-num">01</span>
          <span v-if="keyCount > 0" class="onboarding-done">{{ copy('已就绪', 'Ready') }}</span>
        </div>
        <h3>{{ copy('创建 API 密钥', 'Create an API key') }}</h3>
        <p>{{ keyCount > 0 ? copy('你已创建密钥，可在密钥页复制或管理。', 'Your key is ready. Copy or manage it on the keys page.') : copy('为应用创建一个密钥，并选择可用分组。', 'Create a key for your application and choose an available group.') }}</p>
        <RouterLink class="portal-button" :class="{ secondary: keyCount > 0 }" :to="keyCount > 0 ? '/keys' : '/keys?create=1'">{{ keyCount > 0 ? copy('管理密钥', 'Manage keys') : copy('创建首个密钥', 'Create your first key') }}</RouterLink>
      </li>
      <li class="portal-panel">
        <div class="onboarding-step-top"><span class="onboarding-step-num">02</span></div>
        <h3>{{ copy('复制 Base URL', 'Copy the Base URL') }}</h3>
        <p>{{ copy('这是 OpenAI 兼容客户端使用的接入地址。', 'Use this address with an OpenAI-compatible client.') }}</p>
        <code class="onboarding-endpoint">{{ openAIBase }}</code>
        <button type="button" class="portal-button secondary" @click="copyToClipboard(openAIBase)">{{ copied ? copy('已复制', 'Copied') : copy('复制 Base URL', 'Copy Base URL') }}</button>
      </li>
      <li class="portal-panel">
        <div class="onboarding-step-top"><span class="onboarding-step-num">03</span></div>
        <h3>{{ copy('发送第一条请求', 'Send your first request') }}</h3>
        <p>{{ copy('参考调用示例，使用分组支持的模型；成功请求会出现在用量页。', 'Follow an example with a model supported by your group. Successful requests appear in Usage.') }}</p>
        <RouterLink class="portal-button secondary" to="/home#home-examples">{{ copy('查看调用示例', 'View request examples') }}</RouterLink>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.workspace-onboarding { margin: 12px 0 24px; }
.workspace-onboarding-head { margin-bottom: 20px; }
.onboarding-kicker { display: inline-flex; align-items: center; padding: 3px 10px; margin-bottom: 8px; border-radius: 999px; background: rgb(var(--brand-moss) / .1); color: rgb(var(--brand-moss)); font-size: 12px; font-weight: 600; }
h2 { margin: 0 0 6px; font-size: 22px; font-weight: 650; letter-spacing: -0.02em; }
ol { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px; margin: 0; padding: 0; list-style: none; }
li { display: flex; flex-direction: column; align-items: flex-start; gap: 12px; min-width: 0; transition: border-color var(--motion-fast) ease, box-shadow var(--motion-fast) ease; }
li:hover { border-color: rgb(var(--line-strong)); box-shadow: var(--shadow-card-hover); }
li[data-done='true'] { border-color: rgb(var(--brand-moss) / .35); }
.onboarding-step-top { display: flex; width: 100%; align-items: center; justify-content: space-between; gap: 8px; }
.onboarding-step-num { display: inline-flex; align-items: center; justify-content: center; height: 26px; padding: 0 9px; border-radius: 7px; background: rgb(var(--surface-2)); color: rgb(var(--ink-2)); font-family: var(--font-mono); font-size: 12px; font-weight: 600; }
.onboarding-done { padding: 2px 8px; border-radius: 999px; background: rgb(var(--status-ok-bg)); color: rgb(var(--text-ok)); font-size: 12px; font-weight: 600; }
h3 { font-size: 16.5px; font-weight: 650; letter-spacing: -0.01em; margin: 0; }
li p { margin: 0; color: rgb(var(--ink-2)); font-size: 13.5px; line-height: 1.65; flex: 1; }
.onboarding-endpoint { width: 100%; padding: 8px 10px; border: 1px solid rgb(var(--line)); border-radius: 8px; background: rgb(var(--surface-2)); color: rgb(var(--ink)); font-family: var(--font-mono); overflow-wrap: anywhere; font-size: 12.5px; }
@media(max-width:899px) { ol { grid-template-columns: 1fr; } }
</style>
