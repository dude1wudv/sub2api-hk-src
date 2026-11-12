<template>
  <div class="patrick-surface patrick-auth">
    <header class="pa-header">
      <div class="pa-header-inner">
        <router-link to="/home" class="pa-brand-link" :aria-label="zh ? 'patrickapi 首页' : 'patrickapi home'">
          <PatrickBrand />
        </router-link>
        <nav class="pa-header-actions" :aria-label="zh ? '页面导航' : 'Page navigation'">
          <router-link to="/home" class="pa-nav-link">{{ zh ? '首页' : 'Home' }}</router-link>
          <router-link to="/key-usage" class="pa-nav-link">{{ zh ? '查额' : 'Usage' }}</router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="pa-nav-link">{{ zh ? '文档' : 'Docs' }}</a>
          <ColorModeToggle />
          <label class="pa-language">
            <span class="sr-only">{{ zh ? '语言' : 'Language' }}</span>
            <select :value="locale" :disabled="switchingLanguage" @change="changeLanguage">
              <option v-for="language in availableLocales" :key="language.code" :value="language.code">{{ language.code === 'zh' ? 'CN' : 'EN' }}</option>
            </select>
          </label>
        </nav>
      </div>
    </header>

    <main class="pa-main">
      <InkSurface class="pa-story">
        <div class="pa-story-top">
          <p class="pa-story-eyebrow"><i aria-hidden="true" />PATRICKAPI // ACCESS</p>
          <h1>{{ zh ? '统一网关账户中心' : 'Unified Gateway Console' }}</h1>
          <p class="pa-story-subtitle">{{ zh ? '集中管理多协议 API 密钥、分组路由、速率配额与实时 Token 计量账单。' : 'Manage multi-protocol API keys, group routing, rate quotas and live token billing in one place.' }}</p>
        </div>
        <svg class="pa-story-art" viewBox="0 0 440 210" fill="none" aria-hidden="true">
          <rect x="16" y="18" width="408" height="174" rx="16" class="pa-art-frame" />
          <line x1="16" y1="54" x2="424" y2="54" class="pa-art-divider" />
          <circle cx="38" cy="36" r="4" class="pa-story-moss" />
          <circle cx="52" cy="36" r="4" class="pa-story-drop" />
          <rect x="344" y="29" width="62" height="14" rx="4" class="pa-art-pill" />
          <path d="M84 124H174M266 124H356" class="pa-art-bus" />
          <path d="M174 124H192M248 124H266" class="pa-art-bus-active" />
          <rect x="36" y="96" width="66" height="56" rx="11" class="pa-art-node" />
          <path d="M62 117L55 124L62 131M76 117L83 124L76 131" class="pa-art-glyph" />
          <rect x="188" y="84" width="64" height="80" rx="14" class="pa-art-core" />
          <path d="M211 139V111C211 108.5 213 106.5 215.5 106.5H224.5a9.5 9.5 0 0 1 0 19H211" class="pa-art-p" />
          <circle cx="232" cy="136" r="3.2" class="pa-story-moss" />
          <rect x="338" y="82" width="66" height="36" rx="9" class="pa-art-node" />
          <rect x="338" y="130" width="66" height="36" rx="9" class="pa-art-node" />
          <circle cx="352" cy="100" r="3.5" class="pa-story-moss" />
          <circle cx="352" cy="148" r="3.5" class="pa-story-drop" />
          <line x1="363" y1="100" x2="390" y2="100" class="pa-art-line" />
          <line x1="363" y1="148" x2="390" y2="148" class="pa-art-line" />
        </svg>
        <div class="pa-story-notes">
          <span><i class="pa-dot pa-dot--moss" aria-hidden="true" />{{ zh ? 'API 密钥隔离' : 'Scoped API keys' }}</span>
          <span><i class="pa-dot pa-dot--copper" aria-hidden="true" />{{ zh ? '实时用量遥测' : 'Live telemetry' }}</span>
          <span><i class="pa-dot pa-dot--moss" aria-hidden="true" />{{ zh ? '多因素安全保护' : '2FA & Passkey' }}</span>
        </div>
      </InkSurface>
      <section class="pa-form-area" :aria-label="zh ? '账户访问' : 'Account access'">
        <div class="pa-form-card"><slot /></div>
        <div v-if="$slots.footer" class="pa-form-footer"><slot name="footer" /></div>
      </section>
    </main>

    <footer class="pa-footer">
      <div class="pa-footer-inner">
        <span>&copy; {{ currentYear }} patrickapi</span>
        <MotionPreferenceToggle show-label />
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { availableLocales, setLocale } from '@/i18n'
import { sanitizeUrl } from '@/utils/url'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import ColorModeToggle from '@/components/common/ColorModeToggle.vue'
import InkSurface from '@/components/common/InkSurface.vue'
import MotionPreferenceToggle from '@/components/common/MotionPreferenceToggle.vue'
import '@/styles/patrick-auth.css'

const appStore = useAppStore()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || '', { allowRelative: true }))
const currentYear = new Date().getFullYear()
const switchingLanguage = ref(false)

async function changeLanguage(event: Event) {
  const code = (event.target as HTMLSelectElement).value
  if (switchingLanguage.value || code === locale.value) return
  switchingLanguage.value = true
  try { await setLocale(code) }
  catch { appStore.showError(zh.value ? '语言切换失败，请重试。' : 'Could not change the language. Please retry.') }
  finally { switchingLanguage.value = false }
}

onMounted(() => { void appStore.fetchPublicSettings().catch(() => undefined) })
</script>
