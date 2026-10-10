<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe v-if="/^https?:\/\//.test(homeContent.trim())" :src="homeContent.trim()" :title="text('自定义首页','Custom homepage')" class="h-screen w-full border-0" allowfullscreen />
    <div v-else v-html="homeContent" />
  </div>
  <div v-else class="patrick-surface bai-home" :data-testid="compact ? 'compact-home' : 'patrick-home'" @keydown.esc="menuOpen = false; servicesOpen = false">
    <a href="#home-main" class="home-skip">{{ text('跳到主要内容','Skip to content') }}</a>
    <div class="home-strip"><span>OpenAI API</span><span>Anthropic API</span><span>{{ text('一个入口，连接模型','One API. More models.') }}</span></div>
    <header class="home-header">
      <router-link to="/home" aria-label="patrickapi home"><PatrickBrand /></router-link>
      <nav class="home-nav" :aria-label="text('主导航','Main navigation')">
        <div class="home-nav-group"><button :aria-expanded="servicesOpen" @click="servicesOpen = !servicesOpen">{{ text('API 服务','API services') }}<Icon name="chevronDown" size="xs" /></button>
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
        <button class="home-language" @click="changeLanguage">{{ zh ? 'CN' : 'EN' }}<Icon name="chevronDown" size="xs" /></button>
        <router-link :to="consolePath" class="home-try" data-testid="home-console">{{ auth.isAuthenticated ? text('控制台','CONSOLE') : text('开始使用','TRY PATRICKAPI') }}</router-link>
        <button class="home-menu-toggle" :aria-label="text('打开菜单','Open menu')" :aria-expanded="menuOpen" @click="menuOpen = !menuOpen"><Icon :name="menuOpen ? 'x' : 'menu'" size="md" /></button>
      </div>
      <nav v-if="menuOpen" class="home-mobile-nav" :aria-label="text('移动导航','Mobile navigation')">
        <router-link :to="consolePath">{{ text('API 服务','API services') }}<Icon name="arrowRight" size="sm" /></router-link>
        <router-link v-if="modelPlazaEnabled" to="/model-plaza">{{ text('模型与价格','Models & pricing') }}</router-link>
        <router-link v-if="!auth.isSimpleMode" to="/purchase">{{ text('充值','Top up') }}</router-link><router-link v-if="subscriptionsEnabled && !auth.isSimpleMode" to="/subscriptions">{{ text('订阅','Subscriptions') }}</router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ text('开发者文档','Documentation') }}</a>
        <router-link to="/key-usage">{{ text('用量查询','Usage lookup') }}</router-link>
      </nav>
    </header>
    <main id="home-main">
      <section class="home-hero" :class="{ 'is-compact':compact }">
        <div class="home-hero-center">
          <router-link :to="consolePath" class="home-notice"><Icon name="terminal" size="sm" /><span>{{ text('统一 API 接入，让模型服务你的下一步。','One API for your next idea.') }}</span><Icon name="chevronRight" size="xs" /></router-link>
          <h1>patrickapi</h1>
          <form class="home-entry" @submit.prevent="enterApi">
            <input v-model="modelQuery" :aria-label="text('搜索模型','Search models')" :placeholder="text('你想接入什么模型？','Which model would you like to use?')" />
            <button :aria-label="text('进入控制台','Open console')" type="submit"><Icon name="arrowRight" size="lg" /></button>
          </form>
          <div class="home-endpoint"><code>{{ openAIBase }}</code><button :aria-label="text('复制 API 地址','Copy API base URL')" @click="copyEndpoint"><Icon :name="copied ? 'check' : 'copy'" size="sm" /></button></div>
        </div>
        <a v-if="!compact" href="#home-services" class="home-vision"><span aria-hidden="true">✳</span>{{ text('让每一次调用，都通向更多可能','A world of possibilities. One API.') }}</a>
      </section>
      <section v-if="!compact" id="home-services" class="home-services">
        <h2>{{ text('一个 API，连接你的工作方式。','One API. Built around your work.') }}</h2>
        <div class="home-service-links">
          <router-link :to="consolePath"><Icon name="terminal" size="lg" /><h3>{{ text('API 接入','API access') }}</h3><p>{{ text('选择模型，创建密钥，开始你的第一次请求。','Choose a model, create a key and send your first request.') }}</p><span>{{ text('开始接入','Get started') }} →</span></router-link>
          <router-link to="/usage"><Icon name="chart" size="lg" /><h3>{{ text('用量信息','Usage') }}</h3><p>{{ text('查看每一次调用的 Token、费用与延迟。','Understand tokens, costs and latency for every request.') }}</p><span>{{ text('查看用量','View usage') }} →</span></router-link>
          <router-link to="/profile"><Icon name="user" size="lg" /><h3>{{ text('账户管理','Your account') }}</h3><p>{{ text('余额、订阅和账户设置，集中在一处。','Manage balance, subscriptions and settings in one place.') }}</p><span>{{ text('管理账户','Manage account') }} →</span></router-link>
        </div>
      </section>
    </main>
    <footer class="home-footer"><PatrickBrand /><span>© {{ new Date().getFullYear() }} patrickapi</span><router-link to="/key-usage">{{ text('用量查询','Usage lookup') }}</router-link><a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ text('文档','Docs') }}</a></footer>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/i18n'
import { useAuthStore, useAppStore } from '@/stores'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import Icon from '@/components/icons/Icon.vue'
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
const { openAIBase } = usePatrickApiBase()
const { copyToClipboard } = useClipboard()
const menuOpen = ref(false)
const servicesOpen = ref(false)
const modelQuery = ref('')
const copied = ref(false)
const homeContent = computed(() => (app.cachedPublicSettings?.home_content || '').trim())
const compact = computed(() => app.cachedPublicSettings?.compact_home_enabled === true)
const docUrl = computed(() => sanitizeUrl(app.cachedPublicSettings?.doc_url || app.docUrl || ''))
const consolePath = computed(() => !auth.isAuthenticated ? '/login' : auth.isAdmin ? '/admin/dashboard' : '/keys')
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (auth.isAuthenticated || app.cachedPublicSettings?.model_plaza_require_auth !== true))
const subscriptionsEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.subscription))
function changeLanguage() { setLocale(zh.value ? 'en' : 'zh') }
function enterApi() {
  const model = modelQuery.value.trim()
  void router.push(model && modelPlazaEnabled.value ? { path:'/model-plaza', query:{ model, ...(auth.isAuthenticated ? { embedded:'1' } : {}) } } : { path:'/keys' })
}
async function copyEndpoint() { copied.value = await copyToClipboard(openAIBase.value) }
onMounted(() => { auth.checkAuth(); if (!app.publicSettingsLoaded) void app.fetchPublicSettings() })
</script>
