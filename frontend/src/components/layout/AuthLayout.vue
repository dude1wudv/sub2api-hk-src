<template>
  <div class="patrick-surface patrick-auth">
    <header class="pa-header">
      <router-link to="/home" class="pa-brand-link" :aria-label="zh ? 'patrickapi 首页' : 'patrickapi home'">
        <PatrickBrand />
      </router-link>
      <nav class="pa-header-actions" :aria-label="zh ? '页面导航' : 'Page navigation'">
        <ColorModeToggle />
        <router-link to="/home" class="pa-nav-link">{{ zh ? '首页' : 'Home' }}</router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="pa-nav-link">{{ zh ? '文档' : 'Docs' }}</a>
        <label class="pa-language">
          <span class="sr-only">{{ zh ? '语言' : 'Language' }}</span>
          <select :value="locale" :disabled="switchingLanguage" @change="changeLanguage">
            <option v-for="language in availableLocales" :key="language.code" :value="language.code">{{ language.code === 'zh' ? 'CN' : 'EN' }}</option>
          </select>
        </label>
      </nav>
    </header>

    <main class="pa-main">
      <InkSurface class="pa-story">
        <p class="pa-story-eyebrow">PATRICKAPI</p>
        <h1>{{ zh ? '账户中心' : 'Your account' }}</h1>
        <svg class="pa-story-art" viewBox="0 0 440 280" fill="none" aria-hidden="true"><path d="M16 166c76-133 114 103 205-15S332 94 423 27" stroke="currentColor" stroke-opacity=".35" /><path d="M-20 202c124-141 123 150 255-5S376 121 470 66" stroke="currentColor" stroke-opacity=".18" /><path d="M185 244V93c0-14 10-24 24-24h40c45 0 72 24 72 61s-27 61-72 61h-64" stroke="currentColor" stroke-width="23" stroke-linecap="round" /><path d="M226 111h22c15 0 22 7 22 20s-7 20-22 20h-22" stroke="currentColor" stroke-width="17" stroke-linecap="round" /><circle cx="329" cy="231" r="12" class="pa-story-drop" /><circle cx="103" cy="74" r="8" class="pa-story-moss" /></svg>
        <div class="pa-story-notes"><span>{{ zh ? 'API 密钥' : 'API keys' }}</span><i aria-hidden="true" /><span>{{ zh ? '使用记录' : 'Usage' }}</span><i aria-hidden="true" /><span>{{ zh ? '账户安全' : 'Security' }}</span></div>
      </InkSurface>
      <section class="pa-form-area" :aria-label="zh ? '账户访问' : 'Account access'">
        <div class="pa-form-card"><slot /></div>
        <div v-if="$slots.footer" class="pa-form-footer"><slot name="footer" /></div>
      </section>
    </main>

    <footer class="pa-footer">
      <span>&copy; {{ currentYear }} patrickapi</span>
      <MotionPreferenceToggle show-label />
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
