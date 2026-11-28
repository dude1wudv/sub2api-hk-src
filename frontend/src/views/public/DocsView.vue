<template>
  <div class="docs-page min-h-screen bg-[var(--atelier-canvas)] text-[var(--atelier-ink)]">
    <header class="docs-header border-b border-[var(--atelier-border)] bg-[var(--atelier-surface)]">
      <div class="docs-header-inner">
        <router-link to="/home" aria-label="Home"><PatrickBrand /></router-link>
        <nav class="docs-header-nav" :aria-label="text('主导航', 'Main navigation')">
          <router-link v-if="showModels" to="/model-plaza">{{ text('模型与定价', 'Models & pricing') }}</router-link>
          <router-link :to="auth.isAuthenticated ? '/keys' : '/login'">{{ text('API 密钥', 'API keys') }}</router-link>
          <ColorModeToggle />
        </nav>
      </div>
    </header>

    <div class="docs-layout">
      <aside class="docs-sidebar" :aria-label="text('文档导航', 'Documentation navigation')">
        <p class="docs-sidebar-title">{{ text('使用文档', 'Documentation') }}</p>
        <nav class="docs-nav">
          <a v-for="item in sections" :key="item.id" :href="`#${item.id}`">{{ item.label }}</a>
        </nav>
      </aside>

      <main id="docs-content" class="docs-content">
        <p class="docs-eyebrow">{{ app.siteName || 'patrickapi' }} / API</p>
        <h1>{{ text('开发者文档', 'Developer documentation') }}</h1>
        <p class="docs-lead">{{ text('从创建密钥到发送第一条请求，使用已有的 SDK 和客户端接入本站模型。', 'Create a key and make your first request using your existing SDK or client.') }}</p>
        <nav class="docs-mobile-nav" :aria-label="text('文档导航', 'Documentation navigation')">
          <a v-for="item in sections" :key="item.id" :href="`#${item.id}`">{{ item.label }}</a>
        </nav>

        <section id="quickstart">
          <h2>{{ text('快速开始', 'Quick start') }}</h2>
          <ol>
            <li>{{ text('登录控制台，在 API 密钥页面创建密钥并选择可用分组。', 'Sign in, create an API key, and select an available group.') }} <router-link to="/keys">{{ text('管理密钥', 'Manage keys') }} →</router-link></li>
            <li>{{ text('确认账户余额或订阅可用，并从模型列表选择该密钥可用的完整模型 ID。', 'Make sure your balance or subscription is available, then select a complete model ID available to the key.') }}</li>
            <li>{{ text('将密钥写入环境变量，先查询模型，再发送请求。请把示例中的模型名替换为实际可用的 ID。', 'Set the key in an environment variable, list models, and send a request. Replace the example model with an available ID.') }}</li>
          </ol>
          <CodeBlock label="cURL" :value="quickstart" :copied="copied === quickstart" @copy="copy" />
          <p>{{ text('响应文本通常在 choices[0].message.content。请求和最终费用可以在用量页面查看。', 'Response text is usually in choices[0].message.content. View requests and finalized charges in Usage.') }}</p>
        </section>

        <section id="connection">
          <h2>{{ text('连接与鉴权', 'Connection and authentication') }}</h2>
          <div class="docs-endpoints">
            <div><span>OpenAI Base URL</span><code>{{ openAIBase }}</code></div>
            <div><span>Anthropic Base URL</span><code>{{ apiRoot }}</code></div>
          </div>
          <p>{{ text('OpenAI 兼容接口使用 Authorization: Bearer 密钥。Anthropic Messages 可使用 x-api-key 和 anthropic-version: 2023-06-01。SDK 会自行补齐接口路径，请避免重复添加 /v1。', 'OpenAI-compatible endpoints use Authorization: Bearer. Anthropic Messages accepts x-api-key with anthropic-version: 2023-06-01. SDKs append endpoint paths, so avoid adding /v1 twice.') }}</p>
          <p>{{ text('密钥对应账户和分组的权限、额度及模型范围。请将私有密钥保存在环境变量或安全配置中。', 'Your key inherits account and group permissions, quotas, and model access. Store private keys in environment variables or secure configuration.') }}</p>
        </section>

        <section id="models">
          <h2>{{ text('模型与费用', 'Models and pricing') }}</h2>
          <p>{{ text('GET /v1/models 返回当前密钥可见的模型。可用模型、价格、缓存价格和分组倍率以本站当前配置为准，实际扣费以用量记录为准。', 'GET /v1/models lists models visible to your key. Availability, prices, cache rates and group multipliers follow the current site configuration. Final charges appear in usage records.') }}</p>
          <router-link v-if="showModels" to="/model-plaza">{{ text('查看本站模型与定价', 'View models and pricing') }} →</router-link>
          <router-link v-else to="/keys">{{ text('登录后查看密钥和可用分组', 'Sign in to view keys and available groups') }} →</router-link>
        </section>

        <section id="integrations">
          <h2>{{ text('接入客户端', 'Connect a client') }}</h2>
          <p>{{ text('根据密钥绑定分组支持的协议，填写上方地址和模型 ID。以下环境变量示例适用于终端，密钥请替换为自己的值。', 'Choose a protocol supported by the key’s group and provide its URL and model ID. Replace the key in these terminal examples.') }}</p>
          <h3>Python · OpenAI SDK</h3>
          <CodeBlock label="Python" :value="pythonExample" :copied="copied === pythonExample" @copy="copy" />
          <h3>Claude Code</h3>
          <CodeBlock label="Shell" :value="claudeExample" :copied="copied === claudeExample" @copy="copy" />
          <h3>Codex</h3>
          <p>{{ text('在 Codex 自定义 provider 中使用以下配置，并设置环境变量。模型须支持 Responses 协议。', 'Configure a custom Codex provider and set the environment variable. The model must support Responses.') }}</p>
          <CodeBlock label="config.toml" :value="codexExample" :copied="copied === codexExample" @copy="copy" />
          <div class="docs-code"><pre><code>export SITE_API_KEY="YOUR_API_KEY"
codex</code></pre></div>
        </section>

        <section id="troubleshooting">
          <h2>{{ text('常见问题', 'Troubleshooting') }}</h2>
          <h3>401 · {{ text('鉴权失败', 'Authentication failed') }}</h3>
          <p>{{ text('检查密钥是否复制完整、已启用且请求头格式正确。', 'Check that the key is complete, enabled, and sent in the correct header.') }}</p>
          <h3>403 · {{ text('无法访问', 'Access denied') }}</h3>
          <p>{{ text('结合错误信息检查分组、模型权限、账户余额或订阅状态。', 'Inspect the error message and check group, model permissions, balance, and subscription status.') }}</p>
          <h3>429 · {{ text('请求受限', 'Rate limited') }}</h3>
          <p>{{ text('降低请求频率或并发，稍后再试。上游负载也可能影响可用性。', 'Reduce request rate or concurrency and retry later. Upstream load can also affect availability.') }}</p>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore } from '@/stores'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { useClipboard } from '@/composables/useClipboard'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import ColorModeToggle from '@/components/common/ColorModeToggle.vue'

const CodeBlock = defineComponent({
  props: { label: { type: String, required: true }, value: { type: String, required: true }, copied: Boolean },
  emits: ['copy'],
  setup(props, { emit }) {
    return () => h('div', { class: 'docs-code' }, [
      h('div', { class: 'docs-code-head' }, [h('span', props.label), h('button', { type: 'button', onClick: () => emit('copy', props.value) }, props.copied ? 'Copied' : 'Copy')]),
      h('pre', [h('code', props.value)])
    ])
  }
})
const app = useAppStore()
const auth = useAuthStore()
const { locale } = useI18n()
const { apiRoot, openAIBase } = usePatrickApiBase()
const { copyToClipboard } = useClipboard()
const copied = ref('')
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const showModels = computed(() => app.cachedPublicSettings?.model_plaza_enabled === true && (auth.isAuthenticated || app.cachedPublicSettings?.model_plaza_require_auth !== true))
const sections = computed(() => [
  { id: 'quickstart', label: text('快速开始', 'Quick start') },
  { id: 'connection', label: text('连接与鉴权', 'Connection') },
  { id: 'models', label: text('模型与费用', 'Models & pricing') },
  { id: 'integrations', label: text('接入客户端', 'Clients') },
  { id: 'troubleshooting', label: text('常见问题', 'Troubleshooting') }
])
const quickstart = computed(() => `export API_KEY="YOUR_API_KEY"\ncurl ${openAIBase.value}/models -H "Authorization: Bearer $API_KEY"\n\ncurl ${openAIBase.value}/chat/completions \\
  -H "Authorization: Bearer $API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"YOUR_MODEL_ID","messages":[{"role":"user","content":"你好"}]}'`)
const pythonExample = computed(() => `import os\nfrom openai import OpenAI\n\nclient = OpenAI(base_url="${openAIBase.value}", api_key=os.environ["API_KEY"])\nresponse = client.chat.completions.create(\n    model="YOUR_MODEL_ID",\n    messages=[{"role": "user", "content": "你好"}],\n)\nprint(response.choices[0].message.content)`)
const claudeExample = computed(() => `export ANTHROPIC_BASE_URL="${apiRoot.value}"\nexport ANTHROPIC_AUTH_TOKEN="YOUR_API_KEY"\nexport ANTHROPIC_MODEL="YOUR_MODEL_ID"\nclaude`)
const codexExample = computed(() => `model = "YOUR_MODEL_ID"\nmodel_provider = "site"\n\n[model_providers.site]\nname = "API gateway"\nbase_url = "${openAIBase.value}"\nwire_api = "responses"\nenv_key = "SITE_API_KEY"`)

async function copy(value: string) {
  if (await copyToClipboard(value)) copied.value = value
}
</script>

<style scoped>
.docs-header { position: sticky; top: 0; z-index: 10; }
.docs-header-inner { max-width: 1280px; height: 60px; margin: auto; padding: 0 24px; display: flex; align-items: center; justify-content: space-between; gap: 20px; }
.docs-header-nav { display: flex; align-items: center; gap: 20px; font-size: 14px; color: var(--atelier-ink-secondary); }
.docs-header-nav a:hover, .docs-nav a:hover, .docs-mobile-nav a:hover { color: var(--atelier-ink); }
.docs-layout { max-width: 1280px; margin: auto; display: grid; grid-template-columns: 230px minmax(0, 1fr); }
.docs-sidebar { padding: 42px 24px; border-right: 1px solid var(--atelier-border); }
.docs-sidebar-title { font-size: 13px; font-weight: 700; margin-bottom: 16px; }
.docs-nav { display: grid; gap: 5px; position: sticky; top: 85px; }
.docs-nav a, .docs-mobile-nav a { padding: 8px 10px; border-radius: 8px; color: var(--atelier-ink-secondary); font-size: 14px; }
.docs-nav a:hover, .docs-mobile-nav a:hover { background: var(--atelier-hover); }
.docs-content { max-width: 850px; padding: 56px 48px 100px; min-width: 0; line-height: 1.75; }
.docs-eyebrow { color: var(--atelier-ink-muted); font-size: 13px; font-weight: 600; }
.docs-content h1 { margin-top: 10px; font-size: clamp(32px, 4vw, 42px); line-height: 1.2; letter-spacing: -.035em; font-weight: 700; }
.docs-lead { margin-top: 16px; font-size: 16px; color: var(--atelier-ink-secondary); }
.docs-content section { padding-top: 50px; scroll-margin-top: 70px; }
.docs-content h2 { margin-bottom: 16px; font-size: 24px; font-weight: 650; letter-spacing: -.02em; }
.docs-content h3 { margin: 24px 0 8px; font-size: 16px; font-weight: 650; }
.docs-content p, .docs-content li { color: var(--atelier-ink-secondary); font-size: 14px; }
.docs-content p + p { margin-top: 12px; }
.docs-content ol { list-style: decimal; padding-left: 22px; display: grid; gap: 10px; }
.docs-content section a { color: var(--atelier-focus); }
.docs-code { margin: 18px 0; border-radius: 12px; background: #14171f; color: #f1f5f9; overflow: hidden; }
.docs-code-head { display: flex; justify-content: space-between; align-items: center; padding: 10px 16px; border-bottom: 1px solid rgba(255,255,255,.12); font-size: 12px; color: #cbd5e1; }
.docs-code-head button { padding: 3px 8px; border-radius: 5px; border: 1px solid rgba(255,255,255,.25); }
.docs-code pre { overflow-x: auto; padding: 16px; font-size: 12px; line-height: 1.65; }
.docs-endpoints { display: grid; gap: 10px; margin-bottom: 16px; }
.docs-endpoints div { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px; padding: 12px 14px; border: 1px solid var(--atelier-border); border-radius: 10px; background: var(--atelier-surface); font-size: 13px; }
.docs-endpoints code { overflow-wrap: anywhere; }
.docs-mobile-nav { display: none; }
@media (max-width: 760px) {
  .docs-header-inner { padding: 0 16px; }
  .docs-header-nav { gap: 12px; font-size: 12px; }
  .docs-layout { display: block; }
  .docs-sidebar { display: none; }
  .docs-content { padding: 36px 20px 70px; }
  .docs-mobile-nav { display: flex; overflow-x: auto; gap: 4px; margin-top: 20px; padding-bottom: 4px; white-space: nowrap; }
}
</style>
