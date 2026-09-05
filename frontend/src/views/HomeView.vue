<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-[rgb(var(--canvas))] text-[rgb(var(--ink))]"
  >
    <header class="border-b border-gray-200/80 bg-white/80 px-4 py-3.5 backdrop-blur-md sm:px-6 dark:border-dark-800 dark:bg-dark-900/80">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain ring-1 ring-gray-200/80 dark:ring-dark-700/80"
          />
          <span class="min-w-0 truncate text-base font-semibold text-gray-950 dark:text-white">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <AppearanceSwitcher />
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-700 active:bg-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-primary-500 dark:hover:bg-primary-600"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain shadow-sm ring-1 ring-gray-200/80 dark:ring-dark-700/80"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold tracking-tight text-gray-950 dark:text-white md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-11 items-center justify-center rounded-lg bg-primary-600 px-6 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-primary-700 active:bg-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-primary-500 dark:hover:bg-primary-600"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200/80 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-hidden bg-[rgb(var(--canvas))] text-[rgb(var(--ink))]"
  >
    <!-- Background Decorations: Restrained Warm Paper Ambient Backdrop -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
      <div
        class="absolute -top-32 left-1/2 h-96 w-full max-w-7xl -translate-x-1/2 bg-gradient-to-b from-primary-500/[0.07] via-primary-500/[0.02] to-transparent dark:from-primary-500/[0.05] dark:via-transparent"
      ></div>
      <div
        class="absolute -bottom-48 -right-24 h-96 w-96 rounded-full bg-primary-500/[0.03] blur-3xl dark:bg-primary-500/[0.02]"
      ></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 border-b border-gray-200/60 bg-white/70 px-4 py-3.5 backdrop-blur-md sm:px-6 dark:border-dark-800/80 dark:bg-dark-900/70">
      <nav class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4">
        <!-- Logo -->
        <div class="flex items-center">
          <div class="flex h-9 w-9 items-center justify-center overflow-hidden rounded-lg bg-white ring-1 ring-gray-200/80 dark:bg-dark-800 dark:ring-dark-700/80">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
        </div>

        <!-- Nav Actions -->
        <div class="flex flex-wrap items-center gap-2 sm:gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Model Plaza Link -->
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="inline-flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>

          <!-- Theme Toggle -->
          <AppearanceSwitcher />

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex min-h-10 shrink-0 items-center gap-2 rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-900 transition-colors hover:bg-gray-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-dark-800 dark:text-white dark:hover:bg-dark-700"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="font-medium">{{ t('home.dashboard') }}</span>
            <svg
              class="h-3.5 w-3.5 text-gray-400 dark:text-dark-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25"
              />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-700 active:bg-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-primary-500 dark:hover:bg-primary-600"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-4 py-12 sm:px-6 sm:py-16">
      <div class="mx-auto max-w-6xl">
        <!-- Hero Section - Left/Right Layout -->
        <div class="mb-14 flex flex-col items-center justify-between gap-10 lg:flex-row lg:gap-16">
          <!-- Left: Text Content -->
          <div class="flex-1 text-center lg:text-left">
            <h1
              class="mb-4 [overflow-wrap:anywhere] text-4xl font-bold tracking-tight text-gray-950 dark:text-white md:text-5xl lg:text-6xl"
            >
              {{ siteName }}
            </h1>
            <p class="mb-8 [overflow-wrap:anywhere] text-base leading-relaxed text-gray-600 dark:text-dark-300 md:text-lg lg:text-xl">
              {{ siteSubtitle }}
            </p>

            <!-- CTA Button -->
            <div>
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="btn btn-primary btn-lg shadow-sm transition-shadow hover:shadow"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="md" class="ml-1" :stroke-width="2" />
              </router-link>
            </div>
          </div>

          <!-- Right: Terminal Animation -->
          <div class="flex w-full flex-1 justify-center lg:justify-end">
            <div class="terminal-container">
              <div class="terminal-window">
                <!-- Window header -->
                <div class="terminal-header">
                  <div class="terminal-buttons">
                    <span class="btn-close"></span>
                    <span class="btn-minimize"></span>
                    <span class="btn-maximize"></span>
                  </div>
                  <span class="terminal-title">terminal</span>
                </div>
                <!-- Terminal content -->
                <div class="terminal-body">
                  <div class="code-line line-1">
                    <span class="code-prompt">$</span>
                    <span class="code-cmd">curl</span>
                    <span class="code-flag">-X POST</span>
                    <span class="code-url">/v1/messages</span>
                  </div>
                  <div class="code-line line-2">
                    <span class="code-comment"># Routing to upstream...</span>
                  </div>
                  <div class="code-line line-3">
                    <span class="code-success">200 OK</span>
                    <span class="code-response">{ "content": "Hello!" }</span>
                  </div>
                  <div class="code-line line-4">
                    <span class="code-prompt">$</span>
                    <span class="cursor"></span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Feature Tags - Centered -->
        <div class="mb-14 flex flex-wrap items-center justify-center gap-3 sm:gap-4">
          <div
            class="inline-flex items-center gap-2 rounded-full border border-gray-200/80 bg-white/90 px-4 py-2 text-xs font-medium text-gray-700 shadow-sm backdrop-blur-sm transition-colors dark:border-dark-700 dark:bg-dark-800/90 dark:text-dark-200 sm:text-sm"
          >
            <Icon name="swap" size="sm" class="text-primary-600 dark:text-primary-400" />
            <span>{{ t('home.tags.subscriptionToApi') }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2 rounded-full border border-gray-200/80 bg-white/90 px-4 py-2 text-xs font-medium text-gray-700 shadow-sm backdrop-blur-sm transition-colors dark:border-dark-700 dark:bg-dark-800/90 dark:text-dark-200 sm:text-sm"
          >
            <Icon name="shield" size="sm" class="text-primary-600 dark:text-primary-400" />
            <span>{{ t('home.tags.stickySession') }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2 rounded-full border border-gray-200/80 bg-white/90 px-4 py-2 text-xs font-medium text-gray-700 shadow-sm backdrop-blur-sm transition-colors dark:border-dark-700 dark:bg-dark-800/90 dark:text-dark-200 sm:text-sm"
          >
            <Icon name="chart" size="sm" class="text-primary-600 dark:text-primary-400" />
            <span>{{ t('home.tags.realtimeBilling') }}</span>
          </div>
        </div>

        <!-- Features Grid -->
        <div class="mb-14 grid gap-6 md:grid-cols-3">
          <!-- Feature 1: Unified Gateway -->
          <div
            class="rounded-xl border border-gray-200/80 bg-white/90 p-6 shadow-sm backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/90 dark:hover:border-dark-600"
          >
            <div
              class="mb-4 flex h-11 w-11 items-center justify-center rounded-lg border border-sky-200/60 bg-sky-50 text-sky-600 dark:border-sky-800/40 dark:bg-sky-950/40 dark:text-sky-400"
            >
              <Icon name="server" size="lg" />
            </div>
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white sm:text-lg">
              {{ t('home.features.unifiedGateway') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-300">
              {{ t('home.features.unifiedGatewayDesc') }}
            </p>
          </div>

          <!-- Feature 2: Account Pool -->
          <div
            class="rounded-xl border border-gray-200/80 bg-white/90 p-6 shadow-sm backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/90 dark:hover:border-dark-600"
          >
            <div
              class="mb-4 flex h-11 w-11 items-center justify-center rounded-lg border border-primary-200/60 bg-primary-50 text-primary-600 dark:border-primary-800/40 dark:bg-primary-950/40 dark:text-primary-400"
            >
              <svg
                class="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.75"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white sm:text-lg">
              {{ t('home.features.multiAccount') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-300">
              {{ t('home.features.multiAccountDesc') }}
            </p>
          </div>

          <!-- Feature 3: Billing & Quota -->
          <div
            class="rounded-xl border border-gray-200/80 bg-white/90 p-6 shadow-sm backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/90 dark:hover:border-dark-600"
          >
            <div
              class="mb-4 flex h-11 w-11 items-center justify-center rounded-lg border border-emerald-200/60 bg-emerald-50 text-emerald-600 dark:border-emerald-800/40 dark:bg-emerald-950/40 dark:text-emerald-400"
            >
              <svg
                class="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.75"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white sm:text-lg">
              {{ t('home.features.balanceQuota') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-300">
              {{ t('home.features.balanceQuotaDesc') }}
            </p>
          </div>
        </div>

        <!-- Supported Providers -->
        <div class="mb-8 text-center">
          <h2 class="mb-2 text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
            {{ t('home.providers.title') }}
          </h2>
          <p class="text-sm text-gray-600 dark:text-dark-300">
            {{ t('home.providers.description') }}
          </p>
        </div>

        <div class="mb-16 flex flex-wrap items-center justify-center gap-3 sm:gap-4">
          <!-- Claude - Supported -->
          <div
            class="flex items-center gap-2.5 rounded-xl border border-gray-200/80 bg-white/90 px-4 py-2.5 shadow-sm backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/90 dark:hover:border-dark-600"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-amber-600 to-orange-600 font-bold text-white shadow-sm"
            >
              <span class="text-xs">C</span>
            </div>
            <span class="text-sm font-medium text-gray-800 dark:text-dark-100">{{ t('home.providers.claude') }}</span>
            <span
              class="rounded border border-primary-200/60 bg-primary-50 px-1.5 py-0.5 text-[10px] font-medium text-primary-700 dark:border-primary-800/60 dark:bg-primary-950/50 dark:text-primary-300"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- GPT - Supported -->
          <div
            class="flex items-center gap-2.5 rounded-xl border border-gray-200/80 bg-white/90 px-4 py-2.5 shadow-sm backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/90 dark:hover:border-dark-600"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-emerald-600 to-teal-700 font-bold text-white shadow-sm"
            >
              <span class="text-xs">G</span>
            </div>
            <span class="text-sm font-medium text-gray-800 dark:text-dark-100">GPT</span>
            <span
              class="rounded border border-primary-200/60 bg-primary-50 px-1.5 py-0.5 text-[10px] font-medium text-primary-700 dark:border-primary-800/60 dark:bg-primary-950/50 dark:text-primary-300"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Gemini - Supported -->
          <div
            class="flex items-center gap-2.5 rounded-xl border border-gray-200/80 bg-white/90 px-4 py-2.5 shadow-sm backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/90 dark:hover:border-dark-600"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-sky-600 to-blue-700 font-bold text-white shadow-sm"
            >
              <span class="text-xs">G</span>
            </div>
            <span class="text-sm font-medium text-gray-800 dark:text-dark-100">{{ t('home.providers.gemini') }}</span>
            <span
              class="rounded border border-primary-200/60 bg-primary-50 px-1.5 py-0.5 text-[10px] font-medium text-primary-700 dark:border-primary-800/60 dark:bg-primary-950/50 dark:text-primary-300"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Antigravity - Supported -->
          <div
            class="flex items-center gap-2.5 rounded-xl border border-gray-200/80 bg-white/90 px-4 py-2.5 shadow-sm backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/90 dark:hover:border-dark-600"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-rose-600 to-pink-700 font-bold text-white shadow-sm"
            >
              <span class="text-xs">A</span>
            </div>
            <span class="text-sm font-medium text-gray-800 dark:text-dark-100">{{ t('home.providers.antigravity') }}</span>
            <span
              class="rounded border border-primary-200/60 bg-primary-50 px-1.5 py-0.5 text-[10px] font-medium text-primary-700 dark:border-primary-800/60 dark:bg-primary-950/50 dark:text-primary-300"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- More - Coming Soon -->
          <div
            class="flex items-center gap-2.5 rounded-xl border border-gray-200/60 bg-white/50 px-4 py-2.5 opacity-70 backdrop-blur-sm transition-colors hover:border-gray-300 dark:border-dark-700/60 dark:bg-dark-800/50 dark:hover:border-dark-600"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-500 font-bold text-white"
            >
              <span class="text-xs">+</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.more') }}</span>
            <span
              class="rounded border border-gray-200/80 bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-dark-300"
              >{{ t('home.providers.soon') }}</span
            >
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/80 px-4 py-8 sm:px-6 dark:border-dark-800">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded text-sm text-gray-500 transition-colors hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-400 dark:hover:text-white"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded text-sm text-gray-500 transition-colors hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-400 dark:hover:text-white"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import AppearanceSwitcher from '@/components/common/AppearanceSwitcher.vue'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})


// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())


onMounted(() => {

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
/* Terminal Container */
.terminal-container {
  position: relative;
  display: inline-block;
  width: 100%;
  max-width: 440px;
}

/* Terminal Window */
.terminal-window {
  width: 100%;
  background: #141311;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  box-shadow:
    0 20px 40px -15px rgba(0, 0, 0, 0.5),
    0 0 0 1px rgba(0, 0, 0, 0.2);
  overflow: hidden;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.terminal-window:hover {
  box-shadow:
    0 24px 48px -12px rgba(0, 0, 0, 0.6),
    0 0 0 1px rgba(255, 255, 255, 0.12);
}

/* Terminal Header */
.terminal-header {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  background: #1c1a17;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.terminal-buttons {
  display: flex;
  gap: 6px;
}

.terminal-buttons span {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.btn-close {
  background: #e05252;
}
.btn-minimize {
  background: #e5a13b;
}
.btn-maximize {
  background: #3eb36a;
}

.terminal-title {
  flex: 1;
  text-align: center;
  font-size: 11px;
  font-family: ui-monospace, 'Fira Code', monospace;
  color: #8c8273;
  margin-right: 36px;
}

/* Terminal Body */
.terminal-body {
  padding: 18px 20px;
  font-family: ui-monospace, 'Fira Code', 'Cascadia Code', monospace;
  font-size: 13px;
  line-height: 1.8;
  color: #e6e0d4;
}

.code-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  opacity: 0;
  animation: line-appear 0.4s ease forwards;
}

.line-1 {
  animation-delay: 0.2s;
}
.line-2 {
  animation-delay: 0.8s;
}
.line-3 {
  animation-delay: 1.5s;
}
.line-4 {
  animation-delay: 2.2s;
}

@keyframes line-appear {
  from {
    opacity: 0;
    transform: translateY(3px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.code-prompt {
  color: #4ade80;
  font-weight: 600;
}
.code-cmd {
  color: #38bdf8;
  font-weight: 500;
}
.code-flag {
  color: #fcd34d;
}
.code-url {
  color: #fb923c;
}
.code-comment {
  color: #78716c;
  font-style: italic;
}
.code-success {
  color: #4ade80;
  background: rgba(74, 222, 128, 0.12);
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 600;
  font-size: 12px;
}
.code-response {
  color: #fef08a;
}

/* Blinking Cursor */
.cursor {
  display: inline-block;
  width: 7px;
  height: 15px;
  background: #4ade80;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .terminal-window {
    transition: none !important;
  }
  .code-line {
    animation: none !important;
    opacity: 1 !important;
  }
  .cursor {
    animation: none !important;
    opacity: 1 !important;
  }
}

/* Dark mode adjustments */
:deep(.dark) .terminal-window {
  background: #0f0e0c;
  border-color: rgba(255, 255, 255, 0.08);
}
</style>
