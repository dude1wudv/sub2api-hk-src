<template>
  <div class="min-h-screen bg-[rgb(var(--canvas))] text-[rgb(var(--ink))]">
    <header class="sticky top-0 z-20 border-b border-gray-200/80 bg-white/80 backdrop-blur-md dark:border-dark-800 dark:bg-dark-900/80">
      <div class="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-3.5 sm:px-6">
        <RouterLink to="/home" class="flex min-w-0 items-center gap-3 rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40">
          <template v-if="settings">
            <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white ring-1 ring-gray-200/80 dark:bg-dark-800 dark:ring-dark-700/80">
              <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
            </span>
            <span class="truncate text-base font-semibold text-gray-950 dark:text-white">
              {{ siteName }}
            </span>
          </template>
          <template v-else>
            <span class="h-9 w-9 flex-shrink-0 animate-pulse rounded-lg bg-gray-200 dark:bg-dark-700" aria-hidden="true"></span>
            <span class="h-5 w-28 animate-pulse rounded bg-gray-200 dark:bg-dark-700" aria-hidden="true"></span>
          </template>
        </RouterLink>
        <RouterLink
          to="/login"
          class="inline-flex min-h-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-700 active:bg-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-primary-500 dark:hover:bg-primary-600"
        >
          {{ t('home.login') }}
        </RouterLink>
      </div>
    </header>

    <main class="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:py-12">
      <div v-if="loading" class="flex min-h-[320px] items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-600/20 border-t-primary-600 dark:border-primary-400/20 dark:border-t-primary-400"></div>
      </div>

      <section
        v-else-if="loadError"
        class="rounded-xl border border-red-200 bg-red-50/80 p-6 text-red-700 shadow-sm dark:border-red-900/40 dark:bg-red-950/30 dark:text-red-300"
      >
        <h1 class="text-lg font-semibold">{{ t('legal.loadFailed') }}</h1>
        <p class="mt-2 text-sm">{{ t('legal.retryLater') }}</p>
      </section>

      <section
        v-else-if="!currentDocument"
        class="rounded-xl border border-gray-200/80 bg-white/90 p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800/90"
      >
        <div class="flex items-start gap-3.5">
          <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300">
            <Icon name="document" size="sm" />
          </span>
          <div>
            <h1 class="text-lg font-semibold text-gray-950 dark:text-white">{{ t('legal.notFound') }}</h1>
            <p class="mt-1.5 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('legal.notFoundDescription') }}
            </p>
          </div>
        </div>
      </section>

      <article v-else>
        <div class="mb-8 border-b border-gray-200/80 pb-6 dark:border-dark-700">
          <div class="flex items-start gap-4">
            <span class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl border border-primary-200/60 bg-primary-50 text-primary-700 dark:border-primary-800/50 dark:bg-primary-950/40 dark:text-primary-300">
              <Icon :name="documentIcon" size="md" />
            </span>
            <div class="min-w-0">
              <p class="text-xs font-semibold uppercase tracking-wider text-primary-700 dark:text-primary-400">{{ documentTypeLabel }}</p>
              <h1 class="mt-1.5 break-words text-2xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-3xl lg:text-4xl">
                {{ currentDocument.title }}
              </h1>
              <p v-if="updatedAt" class="mt-2 text-xs text-gray-500 dark:text-dark-400 sm:text-sm">
                {{ t('legal.updatedAt', { date: updatedAt }) }}
              </p>
            </div>
          </div>
        </div>

        <div
          v-if="hasContent"
          class="legal-document-content"
          v-html="renderedHtml"
        ></div>
        <div
          v-else
          class="rounded-xl border border-dashed border-gray-300/80 bg-white/50 px-6 py-14 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800/40 dark:text-dark-400"
        >
          {{ t('legal.empty') }}
        </div>
      </article>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getLocale } from '@/i18n'
import { sanitizeUrl } from '@/utils/url'
import { useAppStore } from '@/stores/app'
import type { LoginAgreementDocument } from '@/types'
import zhAdminCompliance from '../../../../docs/legal/admin-compliance.zh.md?raw'
import enAdminCompliance from '../../../../docs/legal/admin-compliance.en.md?raw'

type LegalDocumentIcon = 'document' | 'shield' | 'globe' | 'cog'

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const settings = computed(() => appStore.cachedPublicSettings)
const loading = ref(!settings.value)
const loadError = ref(false)

marked.setOptions({
  breaks: true,
  gfm: true,
})

const documentId = computed(() => String(route.params.documentId || ''))
const isAdminComplianceDocument = computed(() => documentId.value === 'admin-compliance')
const documents = computed(() => settings.value?.login_agreement_documents ?? [])
const siteName = computed(() => settings.value?.site_name || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || '', {
  allowRelative: true,
  allowDataUrl: true,
}))
const updatedAt = computed(() =>
  isAdminComplianceDocument.value ? '' : settings.value?.login_agreement_updated_at || ''
)
const documentTypeLabel = computed(() =>
  isAdminComplianceDocument.value ? t('legal.adminCompliance') : t('legal.loginAgreement')
)

const currentDocument = computed<LoginAgreementDocument | null>(() => {
  if (isAdminComplianceDocument.value) {
    return {
      id: 'admin-compliance',
      title: t('adminCompliance.title'),
      content_md: getLocale() === 'zh' ? zhAdminCompliance : enAdminCompliance
    }
  }
  const id = documentId.value
  if (!id) {
    return null
  }
  return documents.value.find((doc) => doc.id === id) ?? null
})

const hasContent = computed(() => Boolean(currentDocument.value?.content_md?.trim()))

const renderedHtml = computed(() => {
  const content = currentDocument.value?.content_md?.trim() || ''
  if (!content) {
    return ''
  }
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

const documentIcon = computed<LegalDocumentIcon>(() => {
  const title = currentDocument.value?.title || ''
  if (title.includes('政策') || title.includes('隐私')) {
    return 'shield'
  }
  if (title.includes('国家') || title.includes('地区')) {
    return 'globe'
  }
  if (title.includes('特定')) {
    return 'cog'
  }
  return 'document'
})

onMounted(async () => {
  loadError.value = false
  const loadedSettings = await appStore.fetchPublicSettings()
  if (!loadedSettings) {
    loadError.value = true
  }
  loading.value = false
})
</script>

<style scoped>
.legal-document-content {
  line-height: 1.8;
  overflow-wrap: anywhere;
  color: inherit;
}

.legal-document-content :deep(h1) {
  @apply mb-4 mt-8 border-b border-gray-200/80 pb-3 text-2xl font-bold tracking-tight text-gray-950 dark:border-dark-700 dark:text-white sm:text-3xl;
}

.legal-document-content :deep(h2) {
  @apply mb-3 mt-7 text-xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-2xl;
}

.legal-document-content :deep(h3) {
  @apply mb-2 mt-6 text-lg font-semibold tracking-tight text-gray-900 dark:text-white sm:text-xl;
}

.legal-document-content :deep(h4) {
  @apply mb-2 mt-5 text-base font-semibold text-gray-900 dark:text-white;
}

.legal-document-content :deep(p) {
  @apply mb-4 text-sm sm:text-base leading-relaxed text-gray-700 dark:text-dark-200;
}

.legal-document-content :deep(a) {
  @apply font-medium text-primary-600 underline underline-offset-4 decoration-primary-500/30 transition-colors hover:text-primary-700 hover:decoration-primary-500 dark:text-primary-400 dark:hover:text-primary-300;
}

.legal-document-content :deep(ul) {
  @apply mb-4 list-disc space-y-1 pl-6 text-sm sm:text-base;
}

.legal-document-content :deep(ol) {
  @apply mb-4 list-decimal space-y-1 pl-6 text-sm sm:text-base;
}

.legal-document-content :deep(li) {
  @apply text-gray-700 dark:text-dark-200;
}

.legal-document-content :deep(blockquote) {
  @apply my-5 rounded-r-lg border-l-4 border-primary-500/60 bg-primary-50/30 py-3 pl-4 pr-3 text-sm sm:text-base text-gray-700 dark:bg-primary-950/20 dark:text-dark-200;
}

.legal-document-content :deep(code) {
  @apply rounded border border-gray-200/80 bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-200;
}

.legal-document-content :deep(pre) {
  @apply my-5 overflow-x-auto rounded-xl border border-dark-800 bg-gray-950 p-4 font-mono text-xs sm:text-sm text-gray-100;
}

.legal-document-content :deep(pre code) {
  @apply border-0 bg-transparent p-0 text-inherit;
}

.legal-document-content :deep(table) {
  @apply my-6 block w-full overflow-x-auto border-collapse text-sm;
}

.legal-document-content :deep(th) {
  @apply border border-gray-200/80 bg-gray-50/90 px-4 py-2.5 text-left font-semibold text-gray-900 dark:border-dark-700 dark:bg-dark-800 dark:text-white;
}

.legal-document-content :deep(td) {
  @apply border border-gray-200/80 px-4 py-2.5 text-gray-700 dark:border-dark-700 dark:text-dark-200;
}

.legal-document-content :deep(img) {
  @apply my-6 h-auto max-w-full rounded-xl border border-gray-200/80 shadow-sm dark:border-dark-700;
}

.legal-document-content :deep(hr) {
  @apply my-8 border-gray-200/80 dark:border-dark-700;
}
</style>
