<template>
  <div class="patrick-surface patrick-public pp-not-found">
    <header class="pp-nav">
      <nav class="pp-nav-inner" :aria-label="zh ? '主导航' : 'Main navigation'">
        <router-link to="/" class="pp-brand-link" :aria-label="zh ? 'patrickapi 首页' : 'patrickapi home'"><PatrickBrand /></router-link>
        <div class="pp-nav-actions"><div class="pp-locale"><LocaleSwitcher /></div></div>
      </nav>
    </header>

    <main class="pp-missing-main">
      <div class="pp-missing-art" aria-hidden="true">
        <span>404</span>
        <div class="pp-missing-mark"><PatrickBrand compact /></div>
        <i class="pp-missing-orbit"></i>
      </div>
      <p class="p-eyebrow">PAGE NOT FOUND</p>
      <h1>{{ t('errors.pageNotFound') }}</h1>
      <p class="pp-missing-description">
        {{ zh ? '这个页面可能已被移动，或链接地址有误。你可以返回上一页，继续刚才的工作。' : 'This page may have moved, or the link may be incorrect. Go back to pick up where you left off.' }}
      </p>
      <div class="pp-missing-actions">
        <button type="button" @click="goBack" class="p-button">
          <Icon name="arrowLeft" size="md" />
          {{ zh ? '返回上一页' : 'Go back' }}
        </button>
        <router-link to="/dashboard" class="p-button p-button--primary">
          <Icon name="home" size="md" />
          {{ zh ? '前往控制台' : 'Go to console' }}
        </router-link>
      </div>
      <router-link to="/" class="pp-missing-home">{{ zh ? '或返回首页' : 'Or return home' }} <span aria-hidden="true">↗</span></router-link>
    </main>

    <footer class="pp-footer"><span>patrickapi</span><span>patrickapi.microedulab.com</span></footer>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import '@/styles/patrick-public.css'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const router = useRouter()

function goBack(): void {
  router.back()
}
</script>
