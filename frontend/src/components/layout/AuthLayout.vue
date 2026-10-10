<template>
  <div class="patrick-surface patrick-auth">
    <header class="pa-header">
      <router-link to="/home" class="pa-brand-link" :aria-label="zh ? 'patrickapi 首页' : 'patrickapi home'">
        <PatrickBrand />
      </router-link>
      <nav class="pa-header-actions" :aria-label="zh ? '页面导航' : 'Page navigation'">
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
      <section class="pa-form-area" :aria-label="zh ? '账户访问' : 'Account access'">
        <div class="pa-form-card"><slot /></div>
        <div v-if="$slots.footer" class="pa-form-footer"><slot name="footer" /></div>
      </section>
    </main>

    <footer class="pa-footer">
      <span>&copy; {{ currentYear }} patrickapi</span>
      <span>patrickapi.microedulab.com</span>
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
