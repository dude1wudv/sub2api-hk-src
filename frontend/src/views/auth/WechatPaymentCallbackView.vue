<template>
  <div class="patrick-surface patrick-flow">
    <header class="pf-header">
      <router-link to="/home" class="pf-brand-link" aria-label="patrickapi"><PatrickBrand /></router-link>
      <span class="pf-domain">patrickapi.microedulab.com</span>
    </header>
    <main class="pf-content pf-content--callback">
      <div class="card p-6">
        <h1 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ callbackTitleText }}
        </h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {{ errorMessage || callbackProcessingText }}
        </p>

        <div
          v-if="!errorMessage"
          class="mt-6 flex items-center justify-center py-10"
        >
          <div
            class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"
          ></div>
        </div>

        <div
          v-else
          class="mt-6 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/80"
        >
          <p class="text-sm text-gray-700 dark:text-gray-300">
            {{ errorMessage }}
          </p>
          <button
            class="btn btn-primary mt-4"
            type="button"
            @click="goToRetiredOnlinePayment"
          >
            {{ backToPaymentText }}
          </button>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import '@/styles/patrick-flow.css'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const errorMessage = ref('')

watch(errorMessage, (message) => {
  if (message) {
    appStore.showError(message)
  }
})

const callbackProcessingText = computed(() => t('auth.wechatPayment.callbackProcessing'))
const callbackTitleText = computed(() => t('auth.wechatPayment.callbackTitle'))
const backToPaymentText = computed(() => t('auth.wechatPayment.backToPayment'))

function readQueryString(key: string): string {
  const value = route.query[key]
  if (Array.isArray(value)) {
    return typeof value[0] === 'string' ? value[0] : ''
  }
  return typeof value === 'string' ? value : ''
}

function parseFragmentParams(): URLSearchParams {
  const raw = typeof window !== 'undefined' ? window.location.hash : ''
  const hash = raw.startsWith('#') ? raw.slice(1) : raw
  return new URLSearchParams(hash)
}

function goToRetiredOnlinePayment() {
  void router.replace({ path: '/subscriptions', query: { online_payment: 'retired' } })
}

onMounted(() => {
  const fragment = parseFragmentParams()
  const readParam = (key: string) => fragment.get(key) || readQueryString(key)

  const error = readParam('error') || readParam('err_msg') || readParam('errmsg')
  const errorDescription = readParam('error_description') || readParam('message')

  if (error) {
    errorMessage.value = errorDescription || error
    return
  }

  const resumeToken = readParam('wechat_resume_token')
  const openid = readParam('openid')

  if (!resumeToken && !openid) {
    errorMessage.value = t('auth.wechatPayment.callbackMissingResumeToken')
    return
  }

  // Gateway checkout is retired. Do not forward resume tokens into a page that can spend them.
  goToRetiredOnlinePayment()
})
</script>
