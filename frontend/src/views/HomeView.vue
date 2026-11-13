<template>
  <div v-if="homeContent" class="min-h-screen" data-ink-hover="off">
    <iframe v-if="/^https?:\/\//.test(homeContent.trim())" :src="homeContent.trim()" :title="text('自定义首页','Custom homepage')" class="h-screen supports-[height:100dvh]:h-dvh w-full min-w-0 max-w-full border-0" allowfullscreen />
    <div v-else v-html="homeContent" />
  </div>
  <div v-else class="patrick-surface ink-home" :data-testid="compact ? 'compact-home' : 'patrick-home'" @keydown.esc="closeMenu(); servicesOpen = false">
    <a href="#home-main" class="home-skip">{{ text('跳到主要内容','Skip to content') }}</a>
    <div class="home-strip">
      <span class="home-strip-dot" aria-hidden="true" />
      <span>{{ text('OpenAI / Anthropic 兼容接口','OpenAI / Anthropic compatible APIs') }}</span>
      <code class="home-strip-tag">SSE · TOOL CALLS · PROMPT CACHE</code>
    </div>
    <header class="home-header">
      <div class="home-header-inner">
        <router-link to="/home" class="home-brand" aria-label="patrickapi home"><PatrickBrand /></router-link>
        <nav class="home-nav" :aria-label="text('主导航','Main navigation')">
          <div class="home-nav-group"><button type="button" :aria-expanded="servicesOpen" @click="servicesOpen = !servicesOpen">{{ text('API 服务','API services') }}<Icon name="chevronDown" size="xs" /></button>
            <div v-if="servicesOpen" class="home-popover">
              <router-link :to="consolePath">{{ text('API 接入','API access') }}<Icon name="arrowRight" size="sm" /></router-link>
              <router-link v-if="modelPlazaEnabled" to="/model-plaza">{{ text('模型与价格','Models & pricing') }}</router-link>
              <router-link to="/key-usage">{{ text('用量查询','Usage lookup') }}</router-link>
            </div>
          </div>
          <router-link v-if="modelPlazaEnabled" to="/model-plaza">{{ text('模型','Models') }}</router-link>
          <router-link v-if="!auth.isSimpleMode" to="/purchase">{{ text('充值','Top up') }}</router-link>
          <router-link v-if="subscriptionsEnabled && !auth.isSimpleMode" to="/subscriptions">{{ text('订阅','Subscriptions') }}</router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ text('开发者','Developers') }}<Icon name="chevronDown" size="xs" /></a>
        </nav>
        <div class="home-tools">
          <ColorModeToggle />
          <button type="button" class="home-language" @click="changeLanguage">{{ zh ? 'CN' : 'EN' }}<Icon name="chevronDown" size="xs" /></button>
          <router-link :to="consolePath" class="home-try" data-testid="home-console">{{ auth.isAuthenticated ? text('控制台','Console') : text('开始使用','Get started') }}</router-link>
          <button ref="menuToggle" type="button" class="home-menu-toggle" aria-controls="home-mobile-navigation" :aria-label="text('打开菜单','Open menu')" :aria-expanded="menuOpen" @click="toggleMenu"><Icon :name="menuOpen ? 'x' : 'menu'" size="md" /></button>
        </div>
        <nav v-if="menuOpen" id="home-mobile-navigation" ref="mobileNav" class="home-mobile-nav" :aria-label="text('移动导航','Mobile navigation')">
          <router-link :to="consolePath">{{ text('API 服务','API services') }}<Icon name="arrowRight" size="sm" /></router-link>
          <router-link v-if="modelPlazaEnabled" to="/model-plaza">{{ text('模型与价格','Models & pricing') }}<Icon name="arrowRight" size="sm" /></router-link>
          <router-link v-if="!auth.isSimpleMode" to="/purchase">{{ text('充值','Top up') }}<Icon name="arrowRight" size="sm" /></router-link>
          <router-link v-if="subscriptionsEnabled && !auth.isSimpleMode" to="/subscriptions">{{ text('订阅','Subscriptions') }}<Icon name="arrowRight" size="sm" /></router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ text('开发者文档','Documentation') }}<Icon name="arrowRight" size="sm" /></a>
          <router-link to="/key-usage">{{ text('用量查询','Usage lookup') }}<Icon name="arrowRight" size="sm" /></router-link>
        </nav>
      </div>
    </header>
    <main id="home-main" :inert="menuOpen || undefined">
      <section class="home-hero" :class="{ 'is-compact':compact }">
        <div class="home-hero-copy">
          <p class="home-eyebrow"><i aria-hidden="true" />{{ text('统一模型网关 · 精密计费遥测', 'UNIFIED GATEWAY · PRECISION TELEMETRY') }}</p>
          <h1>{{ text('多模型 API', 'Model APIs.') }}<br /><span>{{ text('统一接入', 'One endpoint.') }}</span></h1>
          <p class="home-hero-lead">{{ text('兼容原生 OpenAI 与 Anthropic SDK，一个密钥直连前沿推理、工具调用与缓存加速，请求耗时与 Token 账单逐笔可查。', 'Drop-in OpenAI and Anthropic SDK compatibility. Route frontier models through one API key with per-request latency and token telemetry.') }}</p>
          <div class="home-hero-actions">
            <router-link v-slot="{ href, navigate }" :to="consolePath" custom><InkSurface as="a" :href="href" magnetic class="home-primary-action" @click="navigate"><span>{{ auth.isAuthenticated ? text('进入控制台', 'Open console') : text('开始使用', 'Get started') }}</span><span class="home-action-arrow" aria-hidden="true">↗</span></InkSurface></router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-secondary-action">{{ text('阅读文档', 'Read the docs') }}<Icon name="arrowRight" size="sm" /></a>
            <a v-else href="#home-examples" class="home-secondary-action">{{ text('调用示例', 'API examples') }}<Icon name="arrowRight" size="sm" /></a>
          </div>
          <form class="home-entry" @submit.prevent="enterApi">
            <Icon name="search" size="sm" aria-hidden="true" />
            <input v-model="modelQuery" :aria-label="text('搜索模型','Search models')" :placeholder="text('搜索模型','Search models')" />
            <button :aria-label="text('搜索模型','Search models')" type="submit"><Icon name="arrowRight" size="md" /></button>
          </form>
          <div class="home-metrics" :aria-label="text('网关核心能力', 'Gateway capabilities')">
            <div><strong>OpenAI + Anthropic</strong><span>{{ text('双原生协议直连', 'Dual native protocols') }}</span></div>
            <div><strong>Tabular Billing</strong><span>{{ text('缓存与倍率透明核算', 'Transparent token accounting') }}</span></div>
            <div><strong>Key Quotas</strong><span>{{ text('分组隔离与额度控制', 'Scoped groups & rate limits') }}</span></div>
          </div>
        </div>
        <div class="home-hero-visual">
          <InkConnection v-model:protocol="protocol" />
          <div class="home-endpoint">
            <span class="home-endpoint-badge">{{ protocol === 'openai' ? 'OpenAI' : 'Anthropic' }} Base URL</span>
            <code>{{ baseURL }}</code>
            <button type="button" :aria-label="copied ? text('已复制接口地址','API base URL copied') : text('复制 API 地址','Copy API base URL')" @click="copyEndpoint">
              <Icon :name="copied ? 'check' : 'copy'" size="sm" />
              <span v-if="copied" role="status">{{ text('已复制','Copied') }}</span>
            </button>
          </div>
        </div>
      </section>
      <section id="home-examples" class="home-examples">
        <div class="home-section-intro">
          <p class="home-section-eyebrow">01 / QUICKSTART</p>
          <h2>{{ text('API 接入', 'API access') }}</h2>
          <p>{{ text('无需改造现有业务代码，替换 Base URL 与环境变量即可发起标准调用。', 'Keep your existing SDK client. Point the base URL and API_KEY to start streaming responses immediately.') }}</p>
          <ol class="home-steps">
            <li><span>01</span>{{ text('在控制台创建 API 密钥并绑定分组', 'Create an API key and select a routing group') }}</li>
            <li><span>02</span>{{ text('配置 OpenAI 或 Anthropic 兼容端点', 'Configure the OpenAI or Anthropic base URL') }}</li>
            <li><span>03</span>{{ text('实时追踪 Token 消耗、缓存命中与耗时', 'Inspect live token usage, cache hits and latency') }}</li>
          </ol>
        </div>
        <PortalApiExamples :show-endpoint="false" v-model:model="exampleModel" v-model:protocol="protocol" />
      </section>
      <section v-if="!compact" id="home-services" class="home-services">
        <div class="home-section-intro home-section-intro--row">
          <div>
            <p class="home-section-eyebrow">02 / WORKSPACE</p>
            <h2>{{ text('控制台', 'Console') }}</h2>
          </div>
          <p>{{ text('从密钥签发、模型目录到逐笔账单审计的一站式工程工作台。', 'Issue scoped keys, browse model pricing, and audit token billing in one workspace.') }}</p>
        </div>
        <div class="home-service-links">
          <InkSurface v-for="item in serviceCards" :key="item.to" interactive class="home-service-card"><router-link :to="item.to"><span class="home-service-number">{{ item.number }}<Icon :name="item.icon" size="md" /></span><h3>{{ item.title }}</h3><p>{{ item.description }}</p><span class="home-service-action">{{ item.action }}<Icon name="arrowRight" size="sm" /></span></router-link></InkSurface>
        </div>
      </section>
    </main>
    <footer class="home-footer" :inert="menuOpen || undefined">
      <div class="home-footer-inner">
        <PatrickBrand />
        <span class="home-footer-copy">© {{ new Date().getFullYear() }} patrickapi</span>
        <div class="home-footer-links">
          <router-link to="/key-usage">{{ text('用量查询','Usage lookup') }}</router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ text('文档','Docs') }}</a>
          <MotionPreferenceToggle show-label />
        </div>
      </div>
    </footer>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/i18n'
import { useAuthStore, useAppStore } from '@/stores'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import Icon from '@/components/icons/Icon.vue'
import ColorModeToggle from '@/components/common/ColorModeToggle.vue'
import MotionPreferenceToggle from '@/components/common/MotionPreferenceToggle.vue'
import InkSurface from '@/components/common/InkSurface.vue'
import InkConnection from '@/components/brand/InkConnection.vue'
import PortalApiExamples from '@/components/portal/PortalApiExamples.vue'
import { useScrollLock } from '@/composables/useScrollLock'
import { useClipboard } from '@/composables/useClipboard'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import '@/styles/patrick-home.css'
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const text = (cn: string, en: string) => zh.value ? cn : en
const auth = useAuthStore()
const app = useAppStore()
const router = useRouter()
const { apiRoot, openAIBase } = usePatrickApiBase()
const protocol = ref<'openai' | 'anthropic'>('openai')
const exampleModel = ref('')
const baseURL = computed(() => protocol.value === 'openai' ? openAIBase.value : apiRoot.value)
const { copyToClipboard } = useClipboard()
const menuOpen = ref(false)
const menuToggle = ref<HTMLButtonElement>()
const mobileNav = ref<HTMLElement>()
const wideNavigation = useMediaQuery('(min-width: 961px)')
useScrollLock(menuOpen)
function closeMenu() { const restore = menuOpen.value; menuOpen.value = false; if (restore) void nextTick(() => menuToggle.value?.focus()) }
function toggleMenu() { if (menuOpen.value) closeMenu(); else { menuOpen.value = true; void nextTick(() => mobileNav.value?.querySelector<HTMLElement>('a')?.focus()) } }
watch(wideNavigation, value => { if (value) menuOpen.value = false })
const servicesOpen = ref(false)
const modelQuery = ref('')
const copied = ref(false)
const homeContent = computed(() => (app.cachedPublicSettings?.home_content || '').trim())
const compact = computed(() => app.cachedPublicSettings?.compact_home_enabled === true)
const docUrl = computed(() => sanitizeUrl(app.cachedPublicSettings?.doc_url || app.docUrl || ''))
const consolePath = computed(() => !auth.isAuthenticated ? '/login' : auth.isAdmin ? '/admin/dashboard' : '/dashboard')
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (auth.isAuthenticated || app.cachedPublicSettings?.model_plaza_require_auth !== true))
const subscriptionsEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.subscription))
function changeLanguage() { setLocale(zh.value ? 'en' : 'zh') }
function enterApi() {
  const model = modelQuery.value.trim()
  void router.push(model && modelPlazaEnabled.value ? { path:'/model-plaza', query:{ model, ...(auth.isAuthenticated ? { embedded:'1' } : {}) } } : { path:'/keys' })
}
let copyTimer: ReturnType<typeof setTimeout> | undefined
let copyRequest = 0
function resetCopy() {
  copyRequest++
  if (copyTimer) clearTimeout(copyTimer)
  copyTimer = undefined
  copied.value = false
}
async function copyEndpoint() {
  resetCopy()
  const request = copyRequest
  const success = await copyToClipboard(baseURL.value)
  if (request !== copyRequest) return
  copied.value = success
  if (copied.value) copyTimer = setTimeout(() => { copied.value = false }, 2200)
}
watch(baseURL, resetCopy)
onBeforeUnmount(resetCopy)
const serviceCards = computed(() => [
  { number:'01', to:'/keys', icon:'key' as const, title:text('API 密钥', 'API keys'), description:text('创建密钥、选择分组、设置额度。', 'Create keys, select groups and set quotas.'), action:text('管理密钥', 'Manage keys') },
  { number:'02', to:'/usage', icon:'chart' as const, title:text('使用记录', 'Usage'), description:text('查看请求、Token、费用与耗时。', 'Requests, tokens, costs and latency.'), action:text('查看用量', 'View usage') },
  { number:'03', to:'/dashboard', icon:'home' as const, title:text('概览', 'Overview'), description:text('余额、使用趋势与最近请求。', 'Balance, usage trends and recent requests.'), action:text('打开概览', 'Open overview') },
])
onMounted(() => { auth.checkAuth(); if (!app.publicSettingsLoaded) void app.fetchPublicSettings() })
</script>
