<template>
  <div v-if="homeContent" class="min-h-screen" data-ink-hover="off">
    <iframe
      v-if="/^https?:\/\//.test(homeContent.trim())"
      :src="homeContent.trim()"
      :title="text('自定义首页', 'Custom homepage')"
      class="h-screen w-full min-w-0 max-w-full border-0 supports-[height:100dvh]:h-dvh"
      allowfullscreen
    />
    <div v-else v-html="homeContent" />
  </div>

  <div
    v-else
    class="min-h-screen bg-[var(--atelier-canvas)] text-[var(--atelier-ink)]"
    :data-testid="compact ? 'compact-home' : 'patrick-home'"
    @keydown.esc="closeMenu(); servicesOpen = false"
  >
    <a
      href="#home-main"
      class="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[120] focus:rounded-lg focus:bg-[var(--atelier-ink)] focus:px-3 focus:py-2 focus:text-xs focus:text-[var(--atelier-ink-inverse)]"
    >
      {{ text('跳到主要内容', 'Skip to content') }}
    </a>

    <!-- Minimal Single-Row Header -->
    <header class="sticky top-0 z-40 border-b border-[var(--atelier-border)] bg-[var(--atelier-canvas)]/90 backdrop-blur-md">
      <div class="mx-auto flex h-14 max-w-6xl items-center justify-between gap-4 px-5 sm:px-8">
        <div class="flex items-center gap-8">
          <router-link to="/home" class="inline-flex items-center" aria-label="Home">
            <PatrickBrand />
          </router-link>

          <nav class="hidden items-center gap-6 text-sm text-[var(--atelier-ink-secondary)] md:flex" :aria-label="text('主导航', 'Main navigation')">
            <div class="relative">
              <button
                type="button"
                class="inline-flex items-center gap-1 transition-colors hover:text-[var(--atelier-ink)]"
                :aria-expanded="servicesOpen"
                @click="servicesOpen = !servicesOpen"
              >
                <span>{{ text('API 服务', 'API services') }}</span>
                <Icon name="chevronDown" size="xs" />
              </button>
              <div
                v-if="servicesOpen"
                class="absolute left-0 top-full mt-2 w-52 rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] p-1.5 shadow-[var(--atelier-shadow-md)]"
              >
                <router-link
                  :to="consolePath"
                  class="flex items-center justify-between rounded-lg px-3 py-2 text-sm text-[var(--atelier-ink)] hover:bg-[var(--atelier-hover)]"
                  @click="servicesOpen = false"
                >
                  <span>{{ text('API 接入', 'API access') }}</span>
                  <Icon name="arrowRight" size="xs" />
                </router-link>
                <router-link
                  v-if="modelPlazaEnabled"
                  to="/model-plaza"
                  class="block rounded-lg px-3 py-2 text-sm text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                  @click="servicesOpen = false"
                >
                  {{ text('模型与定价', 'Models & pricing') }}
                </router-link>
                <router-link
                  to="/key-usage"
                  class="block rounded-lg px-3 py-2 text-sm text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                  @click="servicesOpen = false"
                >
                  {{ text('用量查询', 'Usage lookup') }}
                </router-link>
              </div>
            </div>
            <router-link v-if="modelPlazaEnabled" to="/model-plaza" class="transition-colors hover:text-[var(--atelier-ink)]">
              {{ text('模型与定价', 'Models & pricing') }}
            </router-link>
            <router-link v-if="!auth.isSimpleMode" to="/purchase" class="transition-colors hover:text-[var(--atelier-ink)]">
              {{ text('充值兑换', 'Redeem') }}
            </router-link>
            <router-link v-if="subscriptionsEnabled && !auth.isSimpleMode" to="/subscriptions" class="transition-colors hover:text-[var(--atelier-ink)]">
              {{ text('订阅', 'Subscriptions') }}
            </router-link>
            <router-link to="/key-usage" class="transition-colors hover:text-[var(--atelier-ink)]">
              {{ text('用量查询', 'Usage lookup') }}
            </router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="transition-colors hover:text-[var(--atelier-ink)]">
              {{ text('开发者文档', 'Docs') }}
            </a>
          </nav>
        </div>

        <div class="flex items-center gap-2">
          <ColorModeToggle />
          <button
            type="button"
            class="atelier-btn atelier-btn-ghost h-8 px-2.5 text-xs"
            @click="changeLanguage"
          >
            {{ zh ? '中 / EN' : 'EN / 中' }}
          </button>
          <router-link
            v-if="!auth.isAuthenticated"
            to="/login"
            class="atelier-btn atelier-btn-secondary hidden h-8 px-3 text-xs sm:inline-flex"
          >
            {{ text('登录', 'Sign in') }}
          </router-link>
          <router-link
            :to="consolePath"
            class="atelier-btn atelier-btn-primary h-8 px-3.5 text-xs"
            data-testid="home-console"
          >
            {{ auth.isAuthenticated ? text('控制台', 'Console') : text('免费开始', 'Get started') }}
          </router-link>
          <button
            ref="menuToggle"
            type="button"
            class="atelier-btn atelier-btn-ghost h-8 w-8 p-0 md:hidden"
            aria-controls="home-mobile-navigation"
            :aria-label="text('打开菜单', 'Open menu')"
            :aria-expanded="menuOpen"
            @click="toggleMenu"
          >
            <Icon :name="menuOpen ? 'x' : 'menu'" size="md" />
          </button>
        </div>
      </div>

      <!-- Mobile Drawer Nav -->
      <nav
        v-if="menuOpen"
        id="home-mobile-navigation"
        ref="mobileNav"
        class="border-t border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-5 py-3 md:hidden"
        :aria-label="text('移动导航', 'Mobile navigation')"
      >
        <div class="flex flex-col gap-1">
          <router-link :to="consolePath" class="flex items-center justify-between rounded-lg px-3 py-2.5 text-sm font-medium text-[var(--atelier-ink)] hover:bg-[var(--atelier-hover)]" @click="closeMenu">
            <span>{{ auth.isAuthenticated ? text('进入控制台', 'Open console') : text('开始使用', 'Get started') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <router-link v-if="modelPlazaEnabled" to="/model-plaza" class="flex items-center justify-between rounded-lg px-3 py-2.5 text-sm text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)]" @click="closeMenu">
            <span>{{ text('模型与定价', 'Models & pricing') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <router-link v-if="!auth.isSimpleMode" to="/purchase" class="flex items-center justify-between rounded-lg px-3 py-2.5 text-sm text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)]" @click="closeMenu">
            <span>{{ text('充值兑换', 'Redeem') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <router-link v-if="subscriptionsEnabled && !auth.isSimpleMode" to="/subscriptions" class="flex items-center justify-between rounded-lg px-3 py-2.5 text-sm text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)]" @click="closeMenu">
            <span>{{ text('订阅', 'Subscriptions') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <router-link to="/key-usage" class="flex items-center justify-between rounded-lg px-3 py-2.5 text-sm text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)]" @click="closeMenu">
            <span>{{ text('用量查询', 'Usage lookup') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="flex items-center justify-between rounded-lg px-3 py-2.5 text-sm text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)]" @click="closeMenu">
            <span>{{ text('开发者文档', 'Documentation') }}</span>
            <Icon name="arrowRight" size="sm" />
          </a>
        </div>
      </nav>
    </header>

    <main id="home-main" :inert="menuOpen || undefined">
      <!-- Airtable Warm-Ivory Hero with Restrained 1.5px Radial-Faded Edge Dot-Matrix -->
      <section class="relative overflow-hidden border-b border-[var(--atelier-border)] bg-[var(--atelier-canvas)]">
        <DotMatrixBackdrop mask="edges" class="opacity-90" />

        <div class="relative z-10 mx-auto max-w-5xl px-5 pb-16 pt-14 text-center sm:px-8 sm:pb-20 sm:pt-20">
          <div class="mb-5 inline-flex items-center gap-2 rounded-full border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-3 py-1 text-xs text-[var(--atelier-ink-secondary)]">
            <UiStatusDot tone="ok" />
            <span>{{ text('OpenAI & Anthropic 双协议原生兼容 · 逐笔透明计费', 'Native OpenAI & Anthropic protocols · Transparent per-token billing') }}</span>
          </div>

          <h1 class="mx-auto max-w-3xl text-4xl font-semibold tracking-[-0.035em] text-[var(--atelier-ink)] sm:text-5xl sm:leading-[1.12]">
            {{ text('一个端点，直连全球前沿大模型', 'One endpoint for frontier models.') }}
          </h1>

          <p class="mx-auto mt-4 max-w-2xl text-base leading-relaxed text-[var(--atelier-ink-secondary)] sm:text-[17px]">
            {{ text('保持现有 SDK 与工程代码零改造，通过统一 API 密钥调用 OpenAI、Claude、Gemini、DeepSeek 等全量主流模型，实时追踪首字延迟与缓存命中。', 'Keep your existing SDK untouched. Route OpenAI, Claude, Gemini, and DeepSeek through a single key with live TTFT telemetry and transparent billing.') }}
          </p>

          <!-- Floating Pure-White Interactive API Preview Card -->
          <div class="mx-auto mt-10 max-w-3xl rounded-2xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] p-5 text-left shadow-[var(--atelier-shadow-sm)] sm:p-6">
            <div class="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--atelier-border-subtle)] pb-4">
              <div class="atelier-segmented" role="tablist" :aria-label="text('选择示例模型', 'Choose sample model')">
                <button
                  v-for="item in previewModels"
                  :key="item.id"
                  type="button"
                  role="tab"
                  :aria-selected="activePreviewId === item.id"
                  :data-active="activePreviewId === item.id ? 'true' : 'false'"
                  class="atelier-segmented-item !px-2.5 !py-1.5 text-xs"
                  @click="selectPreviewModel(item)"
                >
                  <BrandModelIcon :model="item.brand" size="xs" />
                  <span>{{ item.label }}</span>
                </button>
              </div>

              <UiSegmented
                v-model="protocol"
                :options="[
                  { value: 'openai', label: 'OpenAI SDK' },
                  { value: 'anthropic', label: 'Anthropic SDK' }
                ]"
                size="sm"
                :aria-label="text('接口协议', 'API protocol')"
              />
            </div>

            <!-- Live Base URL Bar -->
            <div class="home-endpoint mt-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-canvas)] px-3.5 py-2" :data-protocol="protocol">
              <div class="flex min-w-0 items-center gap-2">
                <span class="text-[11px] font-medium uppercase tracking-wider text-[var(--atelier-ink-muted)]">
                  {{ protocol === 'openai' ? 'POST /v1/chat/completions' : 'POST /v1/messages' }}
                </span>
                <span class="hidden text-[var(--atelier-border-strong)] sm:inline">|</span>
                <code class="truncate font-mono text-xs text-[var(--atelier-ink)]">{{ baseURL }}</code>
              </div>
              <button
                type="button"
                class="atelier-btn atelier-btn-ghost h-7 px-2 text-xs"
                :aria-label="copied ? text('已复制接口地址', 'API base URL copied') : text('复制 API 地址', 'Copy API base URL')"
                @click="copyEndpoint"
              >
                <Icon :name="copied ? 'check' : 'copy'" size="xs" />
                <span>{{ copied ? text('已复制', 'Copied') : text('复制地址', 'Copy URL') }}</span>
              </button>
            </div>

            <!-- Real Spec-Accurate JSON Request Body (messages array, never prompt!) -->
            <pre class="mt-3 overflow-x-auto rounded-lg border border-[var(--atelier-border-subtle)] bg-[var(--atelier-canvas)] p-4 font-mono text-[12.5px] leading-relaxed text-[var(--atelier-ink)]"><code>{{ samplePayload }}</code></pre>

            <!-- Card Footer Actions & Quick Model Search -->
            <div class="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <form class="home-entry relative flex-1 sm:max-w-xs" @submit.prevent="enterApi">
                <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" />
                <input
                  v-model="modelQuery"
                  :aria-label="text('搜索模型', 'Search models')"
                  :placeholder="text('搜索模型，如 claude-sonnet-4...', 'Search models, e.g. claude-sonnet-4...')"
                  class="atelier-input h-9 pl-9 pr-9 text-xs"
                />
                <button
                  type="submit"
                  :aria-label="text('搜索模型', 'Search models')"
                  class="absolute right-1.5 top-1/2 inline-flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-md text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                >
                  <Icon name="arrowRight" size="xs" />
                </button>
              </form>

              <div class="flex flex-wrap items-center gap-2.5">
                <router-link
                  v-if="modelPlazaEnabled"
                  to="/model-plaza"
                  class="atelier-btn atelier-btn-secondary h-9 px-3.5 text-xs"
                >
                  {{ text('查看模型与定价', 'Models & pricing') }}
                </router-link>
                <a
                  v-else-if="docUrl"
                  :href="docUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="atelier-btn atelier-btn-secondary h-9 px-3.5 text-xs"
                >
                  {{ text('阅读文档', 'Read docs') }}
                </a>
                <router-link
                  :to="consolePath"
                  class="atelier-btn atelier-btn-primary h-9 px-4 text-xs"
                >
                  <span>{{ auth.isAuthenticated ? text('进入控制台', 'Open console') : text('获取 API Key', 'Get API key') }}</span>
                  <Icon name="arrowRight" size="xs" />
                </router-link>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Lower Pure-White Section: Balanced Two-Tier Vendor Strip + 3-Column Capabilities -->
      <section class="border-b border-[var(--atelier-border)] bg-[var(--atelier-surface)] py-16">
        <div class="mx-auto max-w-5xl px-5 sm:px-8">
          <div class="text-center">
            <p class="text-xs font-medium uppercase tracking-[0.12em] text-[var(--atelier-ink-muted)]">
              {{ text('全量前沿生态', 'FRONTIER ECOSYSTEM') }}
            </p>
            <h2 class="mt-2 text-2xl font-semibold tracking-tight text-[var(--atelier-ink)] sm:text-3xl">
              {{ text('全量主流模型，透明按量计费', 'Every major model, transparent metering') }}
            </h2>
          </div>

          <!-- Tier 1: Centered Full-Color Official Provider Logo Strip -->
          <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
            <div
              v-for="vendor in vendorStrip"
              :key="vendor.key"
              class="inline-flex items-center gap-2 rounded-full border border-[var(--atelier-border)] bg-[var(--atelier-canvas)] px-3.5 py-1.5 text-xs font-medium text-[var(--atelier-ink)]"
            >
              <BrandModelIcon :model="vendor.key" size="sm" />
              <span>{{ vendor.name }}</span>
            </div>
          </div>

          <!-- Tier 2: Clean 3-Column Editorial Feature Row -->
          <div class="mt-14 grid gap-8 border-t border-[var(--atelier-border)] pt-10 md:grid-cols-3">
            <div>
              <p class="font-mono text-xs text-[var(--atelier-ink-muted)]">01</p>
              <h3 class="mt-2 text-base font-semibold text-[var(--atelier-ink)]">
                {{ text('双原生协议直连', 'Dual native protocols') }}
              </h3>
              <p class="mt-2 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
                {{ text('完整支持 OpenAI Chat/Responses 与 Anthropic Messages、流式 SSE、Tool Calls 与 Prompt Cache，现有 SDK 仅需替换 Base URL。', 'Full support for OpenAI Chat/Responses, Anthropic Messages, streaming SSE, tool calls, and prompt caching with zero SDK rewrite.') }}
              </p>
            </div>
            <div>
              <p class="font-mono text-xs text-[var(--atelier-ink-muted)]">02</p>
              <h3 class="mt-2 text-base font-semibold text-[var(--atelier-ink)]">
                {{ text('精密密钥隔离与额度控制', 'Scoped keys & quota guardrails') }}
              </h3>
              <p class="mt-2 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
                {{ text('按项目签发独立 API 密钥，灵活绑定分组倍率、模型白名单、消费上限与有效期，杜绝异常消耗。', 'Issue per-project API keys with routing groups, model allowlists, spend ceilings, and expiration controls.') }}
              </p>
            </div>
            <div>
              <p class="font-mono text-xs text-[var(--atelier-ink-muted)]">03</p>
              <h3 class="mt-2 text-base font-semibold text-[var(--atelier-ink)]">
                {{ text('逐笔账单与延迟遥测', 'Per-request billing & latency') }}
              </h3>
              <p class="mt-2 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
                {{ text('每次调用的输入/输出/缓存 Token、首字耗时、渠道健康与费用明细实时落账，随时导出对账。', 'Every call records input, output, cache tokens, TTFT, and exact cost in real time with CSV export.') }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- Quickstart SDK Examples Section -->
      <section id="home-examples" class="border-b border-[var(--atelier-border)] bg-[var(--atelier-canvas)] py-16">
        <div class="mx-auto max-w-5xl px-5 sm:px-8">
          <div class="grid gap-10 lg:grid-cols-12 lg:items-start">
            <div class="lg:col-span-4">
              <p class="font-mono text-xs uppercase tracking-[0.12em] text-[var(--atelier-ink-muted)]">
                01 / QUICKSTART
              </p>
              <h2 class="mt-2 text-2xl font-semibold tracking-tight text-[var(--atelier-ink)]">
                {{ text('三步完成接入', 'Ready in three steps') }}
              </h2>
              <p class="mt-2.5 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
                {{ text('无需改造现有业务代码，替换 Base URL 与 API Key 即可发起生产级调用。', 'Keep your client stack intact. Swap in your Base URL and API key to start shipping immediately.') }}
              </p>
              <ol class="mt-6 space-y-3 text-sm text-[var(--atelier-ink-secondary)]">
                <li class="flex items-start gap-3">
                  <span class="font-mono text-xs font-semibold text-[var(--atelier-ink)]">01</span>
                  <span>{{ text('在控制台创建 API 密钥并绑定路由分组', 'Create an API key and choose a routing group') }}</span>
                </li>
                <li class="flex items-start gap-3">
                  <span class="font-mono text-xs font-semibold text-[var(--atelier-ink)]">02</span>
                  <span>{{ text('配置 OpenAI 或 Anthropic 兼容端点地址', 'Configure the OpenAI or Anthropic base URL') }}</span>
                </li>
                <li class="flex items-start gap-3">
                  <span class="font-mono text-xs font-semibold text-[var(--atelier-ink)]">03</span>
                  <span>{{ text('实时追踪 Token 消耗、缓存命中与耗时', 'Monitor token spend, cache hits, and TTFT live') }}</span>
                </li>
              </ol>
            </div>

            <div class="lg:col-span-8">
              <div class="atelier-surface p-4 sm:p-5">
                <PortalApiExamples
                  v-model:model="exampleModel"
                  v-model:protocol="protocol"
                  :show-endpoint="false"
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Workspace Quick Links (hidden in compact mode) -->
      <section v-if="!compact" id="home-services" class="bg-[var(--atelier-surface)] py-16">
        <div class="mx-auto max-w-5xl px-5 sm:px-8">
          <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
            <div>
              <p class="font-mono text-xs uppercase tracking-[0.12em] text-[var(--atelier-ink-muted)]">
                02 / WORKSPACE
              </p>
              <h2 class="mt-2 text-2xl font-semibold tracking-tight text-[var(--atelier-ink)]">
                {{ text('工程化控制台入口', 'Developer console') }}
              </h2>
            </div>
            <p class="max-w-md text-sm text-[var(--atelier-ink-secondary)]">
              {{ text('从密钥签发、模型定价对比到逐笔账单审计，统一收口在极简工作台。', 'Issue keys, compare live pricing, and inspect request telemetry in one calm workspace.') }}
            </p>
          </div>

          <div class="mt-8 grid gap-4 md:grid-cols-3">
            <router-link
              v-for="item in serviceCards"
              :key="item.to"
              :to="item.to"
              class="group flex flex-col justify-between rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-canvas)] p-5 transition-colors hover:border-[var(--atelier-border-strong)] hover:bg-[var(--atelier-surface)]"
            >
              <div>
                <div class="flex items-center justify-between text-[var(--atelier-ink-muted)]">
                  <span class="font-mono text-xs">{{ item.number }}</span>
                  <Icon :name="item.icon" size="sm" />
                </div>
                <h3 class="mt-4 text-base font-semibold text-[var(--atelier-ink)]">{{ item.title }}</h3>
                <p class="mt-1.5 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">{{ item.description }}</p>
              </div>
              <span class="mt-5 inline-flex items-center gap-1.5 text-xs font-medium text-[var(--atelier-ink)]">
                <span>{{ item.action }}</span>
                <Icon name="arrowRight" size="xs" class="transition-transform group-hover:translate-x-0.5" />
              </span>
            </router-link>
          </div>
        </div>
      </section>
    </main>

    <footer class="border-t border-[var(--atelier-border)] bg-[var(--atelier-canvas)] py-8" :inert="menuOpen || undefined">
      <div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 px-5 text-xs text-[var(--atelier-ink-muted)] sm:px-8">
        <div class="flex items-center gap-4">
          <PatrickBrand />
          <span>© {{ new Date().getFullYear() }} {{ app.siteName || 'Sub2API' }}</span>
        </div>
        <div class="flex flex-wrap items-center gap-5">
          <router-link to="/key-usage" class="transition-colors hover:text-[var(--atelier-ink)]">{{ text('用量查询', 'Usage lookup') }}</router-link>
          <router-link v-if="modelPlazaEnabled" to="/model-plaza" class="transition-colors hover:text-[var(--atelier-ink)]">{{ text('模型与定价', 'Models & pricing') }}</router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="transition-colors hover:text-[var(--atelier-ink)]">{{ text('文档', 'Docs') }}</a>
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
import PortalApiExamples from '@/components/portal/PortalApiExamples.vue'
import { BrandModelIcon, DotMatrixBackdrop, UiSegmented, UiStatusDot } from '@/ui'
import { useScrollLock } from '@/composables/useScrollLock'
import { useClipboard } from '@/composables/useClipboard'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const text = (cn: string, en: string) => (zh.value ? cn : en)
const auth = useAuthStore()
const app = useAppStore()
const router = useRouter()
const { apiRoot, openAIBase } = usePatrickApiBase()
const protocol = ref<'openai' | 'anthropic'>('openai')
const exampleModel = ref('')
const baseURL = computed(() => (protocol.value === 'openai' ? openAIBase.value : apiRoot.value))
const { copyToClipboard } = useClipboard()

const previewModels = [
  { id: 'claude-sonnet-4-20250514', label: 'Claude', brand: 'claude', preferredProtocol: 'anthropic' as const },
  { id: 'gpt-4o', label: 'OpenAI', brand: 'openai', preferredProtocol: 'openai' as const },
  { id: 'gemini-2.5-pro', label: 'Gemini', brand: 'gemini', preferredProtocol: 'openai' as const },
  { id: 'deepseek-reasoner', label: 'DeepSeek', brand: 'deepseek', preferredProtocol: 'openai' as const }
]
const activePreviewId = ref(previewModels[0].id)

function selectPreviewModel(item: (typeof previewModels)[number]) {
  activePreviewId.value = item.id
  exampleModel.value = item.id
}

const samplePayload = computed(() => {
  if (protocol.value === 'anthropic') {
    return JSON.stringify(
      {
        model: activePreviewId.value,
        max_tokens: 1024,
        stream: true,
        messages: [
          {
            role: 'user',
            content: zh.value ? '用三句话总结流式工具调用的最佳实践。' : 'Summarize streaming tool-calling best practices in three bullets.'
          }
        ]
      },
      null,
      2
    )
  }
  return JSON.stringify(
    {
      model: activePreviewId.value,
      stream: true,
      messages: [
        { role: 'system', content: 'You are a concise engineering assistant.' },
        {
          role: 'user',
          content: zh.value ? '用三句话总结流式工具调用的最佳实践。' : 'Summarize streaming tool-calling best practices in three bullets.'
        }
      ]
    },
    null,
    2
  )
})

const vendorStrip = [
  { key: 'openai', name: 'OpenAI' },
  { key: 'claude', name: 'Anthropic Claude' },
  { key: 'gemini', name: 'Google Gemini' },
  { key: 'deepseek', name: 'DeepSeek' },
  { key: 'grok', name: 'xAI Grok' },
  { key: 'kimi', name: 'Moonshot Kimi' },
  { key: 'zhipu', name: 'Zhipu GLM' },
  { key: 'qwen', name: 'Alibaba Qwen' },
  { key: 'minimax', name: 'MiniMax' }
]

const menuOpen = ref(false)
const menuToggle = ref<HTMLButtonElement>()
const mobileNav = ref<HTMLElement>()
const wideNavigation = useMediaQuery('(min-width: 961px)')
useScrollLock(menuOpen)
function closeMenu() {
  const restore = menuOpen.value
  menuOpen.value = false
  if (restore) void nextTick(() => menuToggle.value?.focus())
}
function toggleMenu() {
  if (menuOpen.value) closeMenu()
  else {
    menuOpen.value = true
    void nextTick(() => mobileNav.value?.querySelector<HTMLElement>('a')?.focus())
  }
}
watch(wideNavigation, (value) => {
  if (value) menuOpen.value = false
})

const servicesOpen = ref(false)
const modelQuery = ref('')
const copied = ref(false)
const homeContent = computed(() => (app.cachedPublicSettings?.home_content || '').trim())
const compact = computed(() => app.cachedPublicSettings?.compact_home_enabled === true)
const docUrl = '/docs'
const consolePath = computed(() => (!auth.isAuthenticated ? '/login' : auth.isAdmin ? '/admin/dashboard' : '/dashboard'))
const modelPlazaEnabled = computed(
  () => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (auth.isAuthenticated || app.cachedPublicSettings?.model_plaza_require_auth !== true)
)
const subscriptionsEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.subscription))

function changeLanguage() {
  setLocale(zh.value ? 'en' : 'zh')
}

function enterApi() {
  const model = modelQuery.value.trim()
  void router.push(
    model && modelPlazaEnabled.value
      ? { path: '/model-plaza', query: { model, ...(auth.isAuthenticated ? { embedded: '1' } : {}) } }
      : { path: '/keys' }
  )
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
  if (copied.value) {
    copyTimer = setTimeout(() => {
      copied.value = false
    }, 2200)
  }
}
watch(baseURL, resetCopy)
onBeforeUnmount(resetCopy)

const serviceCards = computed(() => [
  {
    number: '01',
    to: '/keys',
    icon: 'key' as const,
    title: text('API 密钥', 'API keys'),
    description: text('创建密钥、绑定分组倍率、设置额度上限与智能路由。', 'Issue keys, assign routing groups, set spend limits and failover.'),
    action: text('管理密钥', 'Manage keys')
  },
  {
    number: '02',
    to: '/usage',
    icon: 'chart' as const,
    title: text('用量与账单', 'Usage & billing'),
    description: text('查看逐笔请求、输入输出与缓存 Token 费用及首字延迟。', 'Inspect per-request token usage, cache hits, cost and TTFT.'),
    action: text('查看用量', 'View usage')
  },
  {
    number: '03',
    to: '/dashboard',
    icon: 'home' as const,
    title: text('工作台总览', 'Overview'),
    description: text('账户余额、消耗趋势与活跃模型分布一屏掌握。', 'Balance, spend trend and model mix in a single calm view.'),
    action: text('打开总览', 'Open overview')
  }
])

onMounted(() => {
  auth.checkAuth()
  if (!app.publicSettingsLoaded) void app.fetchPublicSettings()
})
</script>
