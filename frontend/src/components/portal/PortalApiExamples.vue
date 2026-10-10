<template>
  <section id="portal-api-examples" class="portal-section portal-api-examples" aria-labelledby="portal-api-examples-title">
    <div class="portal-section-heading"><h2 id="portal-api-examples-title">{{ text('调用示例', 'Request examples') }}</h2><div class="portal-api-protocols" role="group" :aria-label="text('API 协议', 'API protocol')"><button type="button" :aria-pressed="protocol === 'openai'" @click="protocol = 'openai'">OpenAI</button><button type="button" :aria-pressed="protocol === 'anthropic'" @click="protocol = 'anthropic'">Anthropic</button></div></div>
    <div class="portal-api-endpoint">
      <span>{{ protocol === 'openai' ? 'OpenAI Base URL' : 'Anthropic Base URL' }}</span><code>{{ baseURL }}</code><button type="button" class="portal-inline-button" :aria-label="text('复制接口地址', 'Copy base URL')" @click="copyToClipboard(baseURL)"><Icon name="copy" size="sm" aria-hidden="true" /></button>
    </div>
    <label class="portal-api-example-model"><span>{{ text('模型', 'Model') }}</span><input :value="model" :placeholder="text('填写当前分组支持的模型', 'A model supported by this group')" @input="$emit('update:model', ($event.target as HTMLInputElement).value)" /></label>
    <p class="portal-api-example-note">{{ text('以下为文本消息示例。请使用所选分组支持的协议与模型；Python 和 Node.js 从 API_KEY 环境变量读取密钥。', 'Text-message examples. Use a protocol and model supported by your group. Python and Node.js read the API_KEY environment variable.') }}</p>
    <div class="portal-api-code">
      <div class="portal-api-code-toolbar"><div role="group" :aria-label="text('示例语言', 'Example language')"><button v-for="item in languages" :key="item.value" type="button" :aria-pressed="language === item.value" @click="language = item.value">{{ item.label }}</button></div><button type="button" class="portal-inline-button" @click="copyToClipboard(code)"><Icon name="copy" size="sm" aria-hidden="true" />{{ text('复制', 'Copy') }}</button></div>
      <pre><code>{{ code }}</code></pre>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { useClipboard } from '@/composables/useClipboard'
const props = defineProps<{ model: string }>()
defineEmits<{ 'update:model': [value: string] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const { apiRoot, openAIBase } = usePatrickApiBase()
const { copyToClipboard } = useClipboard()
const protocol = ref<'openai' | 'anthropic'>('openai')
const language = ref<'curl' | 'python' | 'nodejs'>('curl')
const languages = [{ value: 'curl', label: 'curl' }, { value: 'python', label: 'Python' }, { value: 'nodejs', label: 'Node.js' }] as const
const baseURL = computed(() => protocol.value === 'openai' ? openAIBase.value : apiRoot.value)
const endpoint = computed(() => protocol.value === 'openai' ? openAIBase.value + '/chat/completions' : apiRoot.value + '/v1/messages')
const body = computed(() => ({
  model: props.model.trim() || 'YOUR_MODEL',
  ...(protocol.value === 'anthropic' ? { max_tokens: 512 } : {}),
  messages: [{ role: 'user', content: 'Hello!' }],
}))
function shellQuote(value: string) { return "'" + value.replace(/'/g, "'\\''") + "'" }
const code = computed(() => {
  const json = JSON.stringify(body.value, null, 2)
  if (language.value === 'curl') {
    return [
      'API_KEY="YOUR_API_KEY"',
      'curl --request POST ' + shellQuote(endpoint.value) + ' \\',
      protocol.value === 'openai' ? '  --header "Authorization: Bearer $API_KEY" \\' : '  --header "x-api-key: $API_KEY" \\',
      ...(protocol.value === 'anthropic' ? ['  --header "anthropic-version: 2023-06-01" \\'] : []),
      '  --header "Content-Type: application/json" \\',
      '  --data ' + shellQuote(json),
    ].join('\n')
  }
  if (language.value === 'python') {
    return [
      'import json',
      'import os',
      'import urllib.request',
      '',
      'payload = ' + json,
      'headers = {',
      '    "Content-Type": "application/json",',
      protocol.value === 'openai' ? '    "Authorization": "Bearer " + os.environ["API_KEY"],' : '    "x-api-key": os.environ["API_KEY"],',
      ...(protocol.value === 'anthropic' ? ['    "anthropic-version": "2023-06-01",'] : []),
      '}',
      'request = urllib.request.Request(',
      '    ' + JSON.stringify(endpoint.value) + ',',
      '    data=json.dumps(payload).encode("utf-8"),',
      '    headers=headers,',
      '    method="POST",',
      ')',
      'with urllib.request.urlopen(request) as response:',
      '    print(json.loads(response.read()))',
    ].join('\n')
  }
  return [
    'const apiKey = process.env.API_KEY;',
    'if (!apiKey) throw new Error("Set API_KEY first");',
    '',
    'const response = await fetch(' + JSON.stringify(endpoint.value) + ', {',
    '  method: "POST",',
    '  headers: {',
    '    "Content-Type": "application/json",',
    protocol.value === 'openai' ? '    "Authorization": "Bearer " + apiKey,' : '    "x-api-key": apiKey,',
    ...(protocol.value === 'anthropic' ? ['    "anthropic-version": "2023-06-01",'] : []),
    '  },',
    '  body: JSON.stringify(' + json + '),',
    '});',
    'if (!response.ok) throw new Error(await response.text());',
    'console.log(await response.json());',
  ].join('\n')
})
</script>

<style scoped>
.portal-api-examples { scroll-margin-top:80px; }
.portal-api-protocols { display:flex; gap:16px; font-size:12px; }
.portal-api-protocols button { padding:5px 0; border-bottom:1px solid transparent; color:#929088; background:none; }
.portal-api-protocols button[aria-pressed='true'] { border-bottom-color:#000; color:#000; }
.portal-api-endpoint { display:flex; flex-wrap:wrap; align-items:center; gap:10px 16px; padding:12px 0; font-size:12px; }
.portal-api-endpoint > span { color:#76736b; }
.portal-api-endpoint code { min-width:0; overflow-wrap:anywhere; }
.portal-api-example-model { display:flex; align-items:center; gap:18px; font-size:12px; }
.portal-api-example-model > span { flex-shrink:0; color:#76736b; }
.portal-api-example-model input { width:min(420px,100%); min-width:0; padding:7px 10px; border:1px solid #efede3; border-radius:6px; color:#000; background:#fffdf7; font:inherit; }
.portal-api-example-note { margin:10px 0 16px; font-size:12px; line-height:1.7; color:#76736b; }
.portal-api-code { overflow:hidden; border:1px solid #efede3; border-radius:10px; background:#fffdf7; }
.portal-api-code-toolbar { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:10px 16px; border-bottom:1px solid #efede3; background:#f7f5ee; font-size:12px; }
.portal-api-code-toolbar > div { display:flex; gap:18px; }
.portal-api-code-toolbar > div button { padding:4px 0; color:#929088; background:none; }
.portal-api-code-toolbar > div button[aria-pressed='true'] { color:#000; }
.portal-api-code pre { max-height:440px; overflow:auto; margin:0; padding:20px; color:#000; font-family:ui-monospace,SFMono-Regular,Consolas,monospace; font-size:12px; line-height:1.75; tab-size:2; }
@media(max-width:520px) { .portal-api-code-toolbar { padding-inline:12px; } .portal-api-code-toolbar > div { gap:12px; } .portal-api-code pre { padding:16px; } }
</style>
