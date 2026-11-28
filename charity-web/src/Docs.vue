<template>
  <div class="docs-page">
    <header class="docs-header">
      <div class="docs-header-inner">
        <a href="/" class="brand-link"><span class="brand-mark">AI</span><span><strong>{{ siteTitle }}</strong><small>公益共享 API 池</small></span></a>
        <nav><a href="/">返回公益首页</a><button type="button" class="btn btn-sm" @click="toggleTheme">{{ theme === 'dark' ? '浅色' : '深色' }}</button></nav>
      </div>
    </header>

    <div class="docs-layout">
      <aside class="docs-sidebar"><strong>使用文档</strong><nav><a v-for="section in sections" :key="section.id" :href="`#${section.id}`">{{ section.label }}</a></nav></aside>
      <main class="docs-content">
        <p class="eyebrow">free.xjtu.shop / API</p>
        <h1>公益共享 API 接入文档</h1>
        <p class="lead">无需登录，复制当天公共 Key，通过兼容协议调用已开放的无状态推理接口。</p>
        <nav class="mobile-nav"><a v-for="section in sections" :key="section.id" :href="`#${section.id}`">{{ section.label }}</a></nav>

        <section id="scope"><h2>开放范围</h2><p>公益入口只处理一次请求内完成的推理。当前文档覆盖模型发现、Chat Completions、Responses、Anthropic Messages、Gemini generateContent、embeddings，以及分组支持时的同步图片创建。</p><div class="scope-grid"><div v-for="item in scopes" :key="item.name" class="scope-card"><code>{{ item.path }}</code><strong>{{ item.name }}</strong><span>{{ item.note }}</span></div></div><p class="callout">公共 Key 每日北京时间 00:00 自动轮换。公共额度、并发、IP 频率和分组限制以首页实时状态为准。</p></section>

        <section id="quickstart"><h2>快速开始</h2><ol><li>回到<a href="/">公益首页</a>，复制当前分组的公共 Key。</li><li>使用 <code>GET /v1/models</code> 查询当天真正可用的模型 ID。</li><li>选择下方示例中的协议发送一次请求，遇到 401 时回到首页复制新 Key。</li></ol><CodeBlock label="cURL · 模型发现" :value="modelsExample" :copied="copied === modelsExample" @copy="copy" /></section>

        <section id="chat"><h2>Chat Completions</h2><p>适用于 OpenAI 兼容客户端。模型必须来自当前 Key 的 <code>/v1/models</code> 结果。</p><CodeBlock label="POST /v1/chat/completions" :value="chatExample" :copied="copied === chatExample" @copy="copy" /></section>

        <section id="responses"><h2>Responses</h2><p>需要 Responses 协议的客户端可以直接发送以下请求。是否可用取决于模型和公益分组的实时能力。</p><CodeBlock label="POST /v1/responses" :value="responsesExample" :copied="copied === responsesExample" @copy="copy" /></section>

        <section id="protocols"><h2>其他已开放协议</h2><h3>Anthropic Messages</h3><CodeBlock label="POST /v1/messages" :value="messagesExample" :copied="copied === messagesExample" @copy="copy" /><h3>Gemini generateContent</h3><CodeBlock label="POST /v1beta/models/{model}:generateContent" :value="geminiExample" :copied="copied === geminiExample" @copy="copy" /><h3>Embeddings</h3><CodeBlock label="POST /v1/embeddings" :value="embeddingsExample" :copied="copied === embeddingsExample" @copy="copy" /><h3>同步图片创建</h3><p>仅当当前分组的模型列表和能力支持图片生成时使用。接口会等待本次请求的结果，公共入口不提供异步任务查询。</p><CodeBlock label="POST /v1/images/generations" :value="imageExample" :copied="copied === imageExample" @copy="copy" /></section>

        <section id="models">
          <h2>模型与价格</h2>
          <p>下表和首页均使用各分组的 Sub2API 折前 1× 单价。国产模型标人民币，Claude 和 GPT 标美元。每 ¥1 或 $1 都扣 1 单位共享额度，直接相加，不做汇率换算。</p>
          <div v-if="groups.some((group) => group.models.length)" class="price-table-wrap">
            <table>
              <thead><tr><th>分组</th><th>模型</th><th>输入 / 1M</th><th>输出 / 1M</th><th>缓存读 / 1M</th></tr></thead>
              <tbody>
                <template v-for="group in groups" :key="group.id">
                  <tr v-for="model in group.models" :key="`${group.id}-${model.name}`">
                    <td>{{ group.name }}</td>
                    <td><code>{{ model.name }}</code></td>
                    <td>{{ formatPricePerMillion(model.input_price_per_million, model.price_currency) }}</td>
                    <td>{{ formatPricePerMillion(model.output_price_per_million, model.price_currency) }}</td>
                    <td>{{ formatPricePerMillion(model.cache_read_per_million, model.price_currency) }}</td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
          <p v-else class="callout">暂无可显示的模型，请回到首页查看实时状态。</p>
        </section>

        <section id="limits"><h2>边界与排查</h2><ul><li>公益入口不开放共享公共 Key 的创建、查询或管理 API，也不开放管理员路由。</li><li>公益入口不承诺异步任务、实时流式通道或 WebSocket。请使用本页列出的单次 HTTP 请求。</li><li>如果返回 401，请检查 Key 是否是当天首页显示的公共 Key。</li><li>如果返回 403、429 或额度错误，请查看首页的总池、分组、并发和 IP 状态。</li></ul></section>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { fetchCharityOverview, type CharityPublicGroup } from './api'
import { formatPricePerMillion } from './utils'

const theme = ref<'light' | 'dark'>('light')
const groups = ref<CharityPublicGroup[]>([])
const copied = ref('')
const origin = typeof window !== 'undefined' ? window.location.origin : 'https://free.xjtu.shop'
const siteTitle = ref('XJTU Free AI')
const model = computed(() => groups.value.flatMap((group) => group.models).find((item) => item.name)?.name || 'YOUR_MODEL_ID')
const key = 'YOUR_PUBLIC_KEY'
const sections = [
  { id: 'scope', label: '开放范围' }, { id: 'quickstart', label: '快速开始' }, { id: 'chat', label: 'Chat Completions' },
  { id: 'responses', label: 'Responses' }, { id: 'protocols', label: '其他协议' }, { id: 'models', label: '模型与价格' }, { id: 'limits', label: '边界与排查' }
]
const scopes = [
  { name: '模型发现', path: 'GET /v1/models', note: '先查询当前 Key 可用模型' },
  { name: '聊天推理', path: 'POST /v1/chat/completions', note: 'OpenAI 兼容请求' },
  { name: 'Responses 推理', path: 'POST /v1/responses', note: '按分组能力开放' },
  { name: 'Messages 推理', path: 'POST /v1/messages', note: 'Anthropic 原生入口' },
  { name: 'Gemini 推理', path: 'POST /v1beta/models/*:generateContent', note: 'Gemini 兼容入口' },
  { name: 'Embedding / 图片', path: 'POST /v1/embeddings · /v1/images/generations', note: '按模型和分组能力开放' }
]
const modelsExample = computed(() => `curl ${origin}/v1/models \\
  -H "Authorization: Bearer ${key}"`)
const chatExample = computed(() => `curl ${origin}/v1/chat/completions \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${model.value}","messages":[{"role":"user","content":"你好"}]}'`)
const responsesExample = computed(() => `curl ${origin}/v1/responses \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${model.value}","input":"用一句话介绍你自己"}'`)
const messagesExample = computed(() => `curl ${origin}/v1/messages \\
  -H "x-api-key: ${key}" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${model.value}","max_tokens":128,"messages":[{"role":"user","content":"你好"}]}'`)
const geminiExample = computed(() => `curl "${origin}/v1beta/models/${model.value}:generateContent" \\
  -H "x-goog-api-key: ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"contents":[{"parts":[{"text":"你好"}]}]}'`)
const embeddingsExample = computed(() => `curl ${origin}/v1/embeddings \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${model.value}","input":"公益 API"}'`)
const imageExample = computed(() => `curl ${origin}/v1/images/generations \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${model.value}","prompt":"一朵蓝色的云"}'`)
const CodeBlock = defineComponent({
  props: { label: { type: String, required: true }, value: { type: String, required: true }, copied: Boolean },
  emits: ['copy'],
  setup(props, { emit }) { return () => h('div', { class: 'code-block' }, [h('div', { class: 'code-head' }, [h('span', props.label), h('button', { type: 'button', onClick: () => emit('copy', props.value) }, props.copied ? '已复制' : '复制')]), h('pre', [h('code', props.value)])]) }
})
function toggleTheme() { theme.value = theme.value === 'dark' ? 'light' : 'dark'; document.documentElement.dataset.theme = theme.value }
async function copy(value: string) { try { await navigator.clipboard.writeText(value); copied.value = value } catch { copied.value = '' } }
onMounted(async () => { try { const data = await fetchCharityOverview(); groups.value = data.groups; siteTitle.value = data.site_title || siteTitle.value } catch { /* 首页提供实时状态 */ } })
</script>

<style scoped>
.docs-page { min-height: 100vh; background: var(--bg); color: var(--text); }
.docs-header { position: sticky; top: 0; z-index: 5; border-bottom: 1px solid var(--border); background: color-mix(in srgb, var(--surface) 94%, transparent); backdrop-filter: blur(12px); }
.docs-header-inner { max-width: 1120px; height: 64px; margin: auto; padding: 0 18px; display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.brand-link { display: flex; align-items: center; gap: 10px; color: var(--text); text-decoration: none; }
.brand-link small { display: block; color: var(--text-muted); font-size: 11px; }
.brand-mark { width: 34px; height: 34px; display: grid; place-items: center; border-radius: 9px; color: #fff; background: linear-gradient(135deg, var(--accent), #0284c7); font-weight: 700; font-size: 13px; }
.docs-header nav { display: flex; align-items: center; gap: 12px; font-size: 13px; }
.docs-header nav a { color: var(--text-muted); text-decoration: none; }
.btn { border: 1px solid var(--border-strong); border-radius: 8px; padding: 7px 12px; color: var(--text); background: var(--surface); cursor: pointer; }
.docs-layout { max-width: 1120px; margin: auto; display: grid; grid-template-columns: 210px minmax(0, 1fr); }
.docs-sidebar { padding: 40px 18px; border-right: 1px solid var(--border); }
.docs-sidebar nav { position: sticky; top: 90px; display: grid; gap: 4px; margin-top: 14px; }
.docs-sidebar a, .mobile-nav a { padding: 7px 9px; border-radius: 7px; color: var(--text-muted); text-decoration: none; font-size: 13px; }
.docs-sidebar a:hover, .mobile-nav a:hover { color: var(--accent-text); background: var(--accent-soft); }
.docs-content { max-width: 850px; padding: 52px 42px 90px; line-height: 1.65; }
.eyebrow { color: var(--text-muted); font-size: 12px; font-weight: 600; }
h1 { margin: 9px 0 12px; font-size: clamp(30px, 4vw, 42px); line-height: 1.15; letter-spacing: -.035em; }
.lead { color: var(--text-muted); font-size: 16px; }
section { padding-top: 46px; scroll-margin-top: 78px; }
h2 { margin: 0 0 14px; font-size: 23px; letter-spacing: -.02em; }
h3 { margin: 24px 0 8px; font-size: 16px; }
p, li { color: var(--text-muted); font-size: 14px; }
ol, ul { padding-left: 22px; display: grid; gap: 8px; }
a { color: var(--accent-text); }
code { overflow-wrap: anywhere; }
.scope-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin: 18px 0; }
.scope-card { display: grid; gap: 2px; padding: 13px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
.scope-card code { color: var(--accent-text); font-size: 11px; }
.scope-card strong { font-size: 14px; }
.scope-card span { color: var(--text-muted); font-size: 12px; }
.callout { padding: 12px 14px; border-left: 3px solid var(--accent); background: var(--accent-soft); }
.code-block { margin: 16px 0; border-radius: 10px; overflow: hidden; background: var(--code-bg); color: var(--code-text); }
.code-head { display: flex; justify-content: space-between; padding: 9px 13px; border-bottom: 1px solid rgba(255,255,255,.12); font-size: 12px; }
.code-head button { border: 1px solid rgba(255,255,255,.3); border-radius: 5px; padding: 3px 8px; color: inherit; background: transparent; cursor: pointer; }
pre { margin: 0; padding: 14px; overflow-x: auto; font-size: 12px; line-height: 1.6; }
.price-table-wrap { overflow-x: auto; border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
table { width: 100%; border-collapse: collapse; font-size: 12px; }
th, td { padding: 10px 12px; border-bottom: 1px solid var(--border); text-align: left; white-space: nowrap; }
th { color: var(--text-muted); font-weight: 600; }
tr:last-child td { border-bottom: 0; }
.mobile-nav { display: none; }
@media (max-width: 720px) { .docs-layout { display: block; } .docs-sidebar { display: none; } .docs-content { padding: 34px 18px 70px; } .mobile-nav { display: flex; gap: 3px; overflow-x: auto; margin: 18px 0 -8px; white-space: nowrap; } .scope-grid { grid-template-columns: 1fr; } }
</style>
