<template>
  <div class="flex min-h-screen flex-col bg-[var(--atelier-surface)] text-[var(--atelier-ink)]">
    <!-- Minimal Auth Header -->
    <header class="border-b border-[var(--atelier-border-subtle)] bg-[var(--atelier-surface)]">
      <div class="mx-auto flex h-14 max-w-6xl items-center justify-between gap-4 px-5 sm:px-8">
        <router-link to="/home" class="inline-flex items-center" :aria-label="zh ? '返回首页' : 'Home'">
          <PatrickBrand />
        </router-link>
        <nav class="flex items-center gap-3 text-xs text-[var(--atelier-ink-secondary)] sm:gap-5" :aria-label="zh ? '页面导航' : 'Page navigation'">
          <router-link to="/home" class="transition-colors hover:text-[var(--atelier-ink)]">{{ zh ? '首页' : 'Home' }}</router-link>
          <router-link to="/key-usage" class="transition-colors hover:text-[var(--atelier-ink)]">{{ zh ? '用量查询' : 'Usage' }}</router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="transition-colors hover:text-[var(--atelier-ink)]">{{ zh ? '文档' : 'Docs' }}</a>
          <ColorModeToggle />
          <LocaleSwitcher />
        </nav>
      </div>
    </header>

    <!-- Airtable Asymmetric Two-Column Auth Layout: Left Pure White Form + Right Deep Plum Showcase Card -->
    <main class="mx-auto grid w-full max-w-6xl flex-1 items-center gap-12 px-5 py-10 sm:px-8 lg:grid-cols-12 lg:py-14">
      <section class="mx-auto w-full max-w-[420px] lg:col-span-6 lg:mx-0" :aria-label="zh ? '账户访问' : 'Account access'">
        <div class="w-full">
          <slot />
        </div>
        <div v-if="$slots.footer" class="mt-6 text-center text-sm text-[var(--atelier-ink-secondary)]">
          <slot name="footer" />
        </div>
      </section>

      <!-- Right Visual Anchor: Matte Deep Plum Showcase Card with 3 Restrained Asymmetric Micro-UIs -->
      <aside class="hidden lg:col-span-6 lg:block" aria-hidden="true">
        <div class="relative overflow-hidden rounded-3xl bg-[var(--atelier-showcase)] p-9 text-[var(--atelier-showcase-ink)] shadow-[var(--atelier-shadow-md)]">
          <div class="max-w-md">
            <h2 class="text-[28px] font-semibold leading-[1.22] tracking-[-0.025em] text-white">
              {{ zh ? '统一接入全球顶尖大模型，稳健如一。' : 'One resilient gateway for every frontier model.' }}
            </h2>
            <p class="mt-3 text-sm leading-relaxed text-white/70">
              {{ zh ? 'OpenAI / Anthropic 原生协议直通，密钥分组配额与实时 Token 计量清晰可见。' : 'Native OpenAI & Anthropic compatibility with scoped API keys and real-time token metering.' }}
            </p>
            <router-link
              to="/model-plaza"
              class="mt-5 inline-flex h-8 items-center gap-1.5 rounded-lg bg-white px-3.5 text-xs font-medium text-[#14171F] transition-opacity hover:opacity-90"
            >
              <span>{{ zh ? '查看模型与定价' : 'Explore models & pricing' }}</span>
              <Icon name="arrowRight" size="xs" />
            </router-link>
          </div>

          <!-- 3 Asymmetric Restrained Micro-UIs (No stiff 6-box grid, no pastel icon pads) -->
          <div class="mt-9 space-y-3.5">
            <!-- Micro UI 1: Wide Live Streaming Preview Card -->
            <div class="rounded-2xl border border-white/10 bg-[#1B0829]/85 p-4 backdrop-blur-sm">
              <div class="flex items-center justify-between text-[11px] text-white/60">
                <span class="inline-flex items-center gap-2 font-mono text-white/90">
                  <UiStatusDot tone="ok" />
                  POST /v1/chat/completions
                </span>
                <span class="font-mono tabular-nums text-white/60">200 OK · 184ms</span>
              </div>
              <div class="mt-3 flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center gap-1.5 rounded-lg bg-white/10 px-2.5 py-1 text-xs text-white">
                  <BrandModelIcon model="claude" size="xs" /> Claude
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-lg bg-white/10 px-2.5 py-1 text-xs text-white">
                  <BrandModelIcon model="openai" size="xs" /> OpenAI
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-lg bg-white/10 px-2.5 py-1 text-xs text-white">
                  <BrandModelIcon model="gemini" size="xs" /> Gemini
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-lg bg-white/10 px-2.5 py-1 text-xs text-white">
                  <BrandModelIcon model="deepseek" size="xs" /> DeepSeek
                </span>
              </div>
            </div>

            <!-- Micro UI 2 & 3: Two Compact Asymmetric Telemetry Cards -->
            <div class="grid grid-cols-12 gap-3.5">
              <div class="col-span-7 rounded-2xl border border-white/10 bg-white/[0.96] p-4 text-[#14171F]">
                <div class="flex items-center justify-between text-[11px] text-[#525866]">
                  <span>{{ zh ? '首字延迟 TTFT' : 'Median TTFT' }}</span>
                  <span class="inline-flex items-center gap-1 font-mono text-[11px] text-[#14171F]">
                    <UiStatusDot tone="ok" /> 99.98%
                  </span>
                </div>
                <p class="mt-1 font-mono text-xl font-semibold tabular-nums text-[#14171F]">180ms</p>
                <svg class="mt-2 h-8 w-full" viewBox="0 0 180 32" fill="none">
                  <path
                    d="M2 24 C 24 22, 34 14, 54 17 C 74 20, 92 10, 114 13 C 136 16, 152 9, 178 11"
                    stroke="#14171F"
                    stroke-width="1.75"
                    stroke-linecap="round"
                  />
                </svg>
              </div>
              <div class="col-span-5 flex flex-col justify-between rounded-2xl border border-white/10 bg-white/10 p-4 text-white">
                <p class="text-[11px] text-white/65">{{ zh ? '密钥隔离' : 'Scoped Key' }}</p>
                <code class="mt-2 truncate font-mono text-xs text-white/95">sk-sub2-••••9a4f</code>
                <div class="mt-3 h-1 w-full overflow-hidden rounded-full bg-white/15">
                  <div class="h-full w-2/3 rounded-full bg-white/85" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </main>

    <footer class="border-t border-[var(--atelier-border-subtle)] bg-[var(--atelier-surface)] py-5">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-5 text-xs text-[var(--atelier-ink-muted)] sm:px-8">
        <span>&copy; {{ currentYear }} {{ appStore.siteName || 'patrickapi' }}</span>
        <MotionPreferenceToggle show-label />
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import Icon from '@/components/icons/Icon.vue'
import ColorModeToggle from '@/components/common/ColorModeToggle.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import MotionPreferenceToggle from '@/components/common/MotionPreferenceToggle.vue'
import { BrandModelIcon, UiStatusDot } from '@/ui'

const appStore = useAppStore()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || '', { allowRelative: true }))
const currentYear = new Date().getFullYear()

onMounted(() => {
  void appStore.fetchPublicSettings().catch(() => undefined)
})
</script>
