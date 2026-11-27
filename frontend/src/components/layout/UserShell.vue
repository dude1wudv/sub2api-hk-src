<template>
  <div
    class="user-portal flex min-h-screen bg-[var(--atelier-canvas)] text-[var(--atelier-ink)]"
    data-workspace="user"
    :data-sidebar-collapsed="sidebarCollapsed"
    :data-nav-open="overlayOpen"
    :data-navigation-mode="desktop ? 'desktop' : railViewport ? 'rail' : 'mobile'"
  >
    <a
      class="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[120] focus:rounded-lg focus:bg-[var(--atelier-ink)] focus:px-3 focus:py-2 focus:text-xs focus:text-[var(--atelier-ink-inverse)]"
      href="#user-workspace-content"
    >
      {{ zh ? '跳至主要内容' : 'Skip to content' }}
    </a>

    <button
      v-if="overlayOpen"
      class="user-drawer-scrim fixed inset-0 z-40 bg-[var(--atelier-scrim)] backdrop-blur-[1px]"
      type="button"
      tabindex="-1"
      :aria-label="zh ? '关闭导航' : 'Close navigation'"
      @click="closeMobile()"
      @wheel.prevent
      @touchmove.prevent
    />

    <!-- Unified 232px Warm-Ivory Sidebar (ZERO numeric count badges) -->
    <aside
      v-show="desktop || railViewport || mobileOpen"
      id="user-workspace-navigation"
      ref="sidebarRef"
      class="user-sidebar fixed inset-y-0 left-0 z-50 flex flex-col border-r border-[var(--atelier-border)] bg-[var(--atelier-canvas)] transition-[width,transform] duration-200"
      :class="[
        overlayOpen ? 'w-[248px] shadow-[var(--atelier-shadow-md)]' : sidebarCollapsed ? 'w-[64px]' : 'w-[232px]'
      ]"
      :style="overlayOpen && viewport.width > 0 ? { top: `${viewport.top}px`, left: `${viewport.left}px`, bottom: 'auto', height: `${viewport.height}px`, maxHeight: `${viewport.height}px` } : undefined"
      :role="overlayOpen ? 'dialog' : undefined"
      :aria-modal="overlayOpen ? true : undefined"
      :aria-label="zh ? '用户导航' : 'User navigation'"
      tabindex="-1"
      @keydown="handleDrawerKeydown"
    >
      <!-- Brand Row (52px aligned with topbar) -->
      <div class="flex h-[52px] shrink-0 items-center justify-between gap-2 border-b border-[var(--atelier-border)] px-3.5">
        <RouterLink
          to="/home"
          class="flex min-w-0 items-center gap-2"
          aria-label="Home"
          @click="closeCurrentRoute('/home')"
        >
          <PatrickBrand :compact="sidebarCollapsed" />
        </RouterLink>
        <button
          v-if="overlayOpen"
          ref="drawerCloseRef"
          class="user-drawer-close inline-flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          type="button"
          :aria-label="zh ? '关闭导航' : 'Close navigation'"
          @click="closeMobile()"
        >
          <Icon name="x" size="sm" aria-hidden="true" />
        </button>
      </div>

      <!-- Admin-Only Perspective Switcher (NEVER rendered for non-admin users) -->
      <div v-if="authStore.isAdmin && !sidebarCollapsed" class="px-3 pt-3">
        <div class="atelier-segmented grid w-full grid-cols-2" role="tablist" :aria-label="zh ? '工作区视角' : 'Workspace perspective'">
          <span
            class="atelier-segmented-item !px-2 !py-1 text-center text-xs"
            data-active="true"
            aria-current="true"
          >
            {{ zh ? '用户视角' : 'User' }}
          </span>
          <RouterLink
            to="/admin/dashboard"
            class="user-admin-return atelier-segmented-item !px-2 !py-1 text-center text-xs"
            data-active="false"
            @click="closeMobile()"
          >
            {{ zh ? '管理后台' : 'Admin' }}
          </RouterLink>
        </div>
      </div>

      <!-- 4-Group Navigation (ZERO numeric badges, unified soft neutral active pill) -->
      <nav ref="navigationRef" class="user-navigation flex-1 overflow-y-auto px-2.5 py-3" :aria-label="zh ? '主要导航' : 'Main navigation'">
        <div class="mb-4 space-y-0.5">
          <RouterLink
            v-for="item in primaryItems"
            :key="item.path"
            :to="{ path: item.path, query: item.query }"
            class="user-nav-item flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13.5px] transition-colors"
            :class="[
              isActive(item)
                ? 'user-nav-active bg-[var(--atelier-active)] font-semibold text-[var(--atelier-ink)]'
                : 'font-normal text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]',
              sidebarCollapsed ? 'justify-center px-0' : ''
            ]"
            :aria-current="isActive(item) ? 'page' : undefined"
            :aria-label="sidebarCollapsed ? item.label : undefined"
            :title="sidebarCollapsed ? item.label : undefined"
            :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
            :data-nav-path="item.path"
            @click="activateNavigation(item)"
            @pointerenter="prefetchItem(item, $event)"
            @focus="prefetchItem(item)"
          >
            <Icon :name="item.icon" size="sm" class="shrink-0" aria-hidden="true" />
            <span v-if="!sidebarCollapsed" class="user-nav-label truncate">{{ item.label }}</span>
          </RouterLink>
        </div>

        <div v-if="secondarySections.length || docUrl" class="space-y-4">
          <section v-for="section in secondarySections" :key="section.id" :aria-label="section.label">
            <h2
              v-if="!sidebarCollapsed"
              class="user-nav-heading mb-1 px-2.5 text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--atelier-ink-muted)]"
            >
              {{ section.label }}
            </h2>
            <div class="space-y-0.5">
              <component
                :is="item.action ? 'button' : item.href ? 'a' : RouterLink"
                v-for="item in section.items"
                :key="item.path"
                v-bind="item.action ? { type: 'button' } : item.href ? { href: item.href, target: item.newTab ? '_blank' : undefined, rel: item.newTab ? 'noopener noreferrer' : undefined } : { to: { path: item.path, query: item.query } }"
                class="user-nav-item flex h-9 w-full items-center gap-2.5 rounded-lg px-2.5 text-left text-[13.5px] transition-colors"
                :class="[
                  isActive(item)
                    ? 'user-nav-active bg-[var(--atelier-active)] font-semibold text-[var(--atelier-ink)]'
                    : 'font-normal text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]',
                  sidebarCollapsed ? 'justify-center px-0' : ''
                ]"
                :aria-current="isActive(item) ? 'page' : undefined"
                :aria-label="sidebarCollapsed ? item.label : undefined"
                :title="sidebarCollapsed ? item.label : undefined"
                :data-nav-path="item.path"
                @click="activateNavigation(item)"
                @pointerenter="prefetchItem(item, $event)"
                @focus="prefetchItem(item)"
              >
                <span v-if="item.iconSvg" class="h-4 w-4 shrink-0" aria-hidden="true" v-html="sanitizeSvg(item.iconSvg)" />
                <Icon v-else :name="item.icon" size="sm" class="shrink-0" aria-hidden="true" />
                <span v-if="!sidebarCollapsed" class="min-w-0 flex-1 truncate">{{ item.label }}</span>
                <Icon v-if="!sidebarCollapsed && (item.href || item.action)" name="externalLink" size="xs" class="shrink-0 opacity-60" aria-hidden="true" />
              </component>
            </div>
          </section>

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="user-nav-item user-doc-link flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13.5px] text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
            :class="sidebarCollapsed ? 'justify-center px-0' : ''"
            :aria-label="sidebarCollapsed ? (zh ? '文档' : 'Documentation') : undefined"
            :title="sidebarCollapsed ? (zh ? '文档' : 'Documentation') : undefined"
            @click="closeMobile()"
          >
            <Icon name="book" size="sm" class="shrink-0" aria-hidden="true" />
            <span v-if="!sidebarCollapsed" class="min-w-0 flex-1 truncate">{{ zh ? '文档' : 'Documentation' }}</span>
            <Icon v-if="!sidebarCollapsed" name="externalLink" size="xs" class="shrink-0 opacity-60" aria-hidden="true" />
          </a>
        </div>

        <RouterLink
          v-if="!user"
          to="/home"
          class="user-nav-item mt-3 flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13.5px] text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          :class="sidebarCollapsed ? 'justify-center px-0' : ''"
          :aria-label="sidebarCollapsed ? (zh ? '返回首页' : 'Back to home') : undefined"
          :title="sidebarCollapsed ? (zh ? '返回首页' : 'Back to home') : undefined"
          @click="closeCurrentRoute('/home')"
        >
          <Icon name="home" size="sm" class="shrink-0" aria-hidden="true" />
          <span v-if="!sidebarCollapsed">{{ zh ? '返回首页' : 'Back to home' }}</span>
        </RouterLink>
      </nav>

      <!-- Support Disclosure -->
      <details v-if="appStore.contactInfo && !sidebarCollapsed" class="mx-3 mb-2 rounded-lg border border-[var(--atelier-border-subtle)] bg-[var(--atelier-surface)] px-2.5 py-2 text-xs text-[var(--atelier-ink-secondary)]">
        <summary class="cursor-pointer font-medium text-[var(--atelier-ink)]">{{ zh ? '帮助与客服' : 'Help & support' }}</summary>
        <a v-if="supportEmail" :href="`mailto:${supportEmail}`" class="mt-1 block truncate text-[var(--atelier-focus)]">{{ supportEmail }}</a>
        <p v-else class="mt-1 break-words">{{ appStore.contactInfo.replace(/群管理员/g, zh ? '客服' : 'Support') }}</p>
      </details>

      <!-- Bottom Fixed Utilities & User Strip -->
      <div class="shrink-0 border-t border-[var(--atelier-border)] p-2.5">
        <RouterLink
          v-if="authStore.isAdmin && sidebarCollapsed"
          to="/admin/dashboard"
          class="user-admin-return mb-1.5 flex h-9 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          :title="zh ? '返回管理后台' : 'Back to administration'"
          :aria-label="zh ? '返回管理后台' : 'Back to administration'"
        >
          <Icon name="shield" size="sm" aria-hidden="true" />
        </RouterLink>

        <div v-if="user && !sidebarCollapsed" class="mb-2 flex items-center gap-2.5 rounded-lg px-2 py-1.5">
          <img
            v-if="avatarUrl && !avatarFailed"
            :src="avatarUrl"
            :alt="displayName"
            class="h-7 w-7 shrink-0 rounded-full object-cover"
            @error="avatarFailed = true"
          />
          <span v-else class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] text-[11px] font-semibold text-[var(--atelier-ink)]">
            {{ accountInitials }}
          </span>
          <div class="min-w-0 flex-1">
            <p class="truncate text-xs font-medium text-[var(--atelier-ink)]">{{ displayName }}</p>
            <p class="truncate text-[11px] text-[var(--atelier-ink-muted)]">{{ user.email }}</p>
          </div>
        </div>

        <div class="flex items-center justify-between gap-1.5">
          <div ref="localeRef" class="relative">
            <button
              ref="localeToggleRef"
              type="button"
              class="user-language-toggle inline-flex h-8 items-center gap-1 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-2.5 text-xs font-medium text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
              :aria-label="zh ? '切换语言' : 'Change language'"
              :aria-expanded="localeOpen"
              :disabled="localeSwitching"
              aria-controls="user-language-menu"
              aria-haspopup="menu"
              @click="toggleLocale"
            >
              <span>{{ zh ? '中' : 'EN' }}</span>
              <Icon name="chevronDown" size="xs" aria-hidden="true" />
            </button>
            <div
              v-if="localeOpen"
              id="user-language-menu"
              ref="localeMenuRef"
              class="user-language-menu absolute bottom-full left-0 z-50 mb-1.5 w-36 rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] p-1 shadow-[var(--atelier-shadow-md)]"
              role="menu"
              :aria-label="zh ? '语言' : 'Language'"
              @keydown="handleLocaleKeydown"
            >
              <button
                v-for="language in availableLocales"
                :key="language.code"
                type="button"
                role="menuitemradio"
                :aria-checked="language.code === currentLocaleCode"
                :disabled="localeSwitching"
                class="flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-xs text-[var(--atelier-ink)] hover:bg-[var(--atelier-hover)]"
                @click="changeLocale(language.code)"
              >
                <span>{{ language.name }}</span>
                <Icon v-if="language.code === currentLocaleCode" name="check" size="xs" aria-hidden="true" />
              </button>
            </div>
          </div>

          <button
            v-if="desktop"
            ref="desktopToggleRef"
            type="button"
            class="user-sidebar-toggle inline-flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
            :aria-label="sidebarCollapsed ? (zh ? '展开导航' : 'Expand navigation') : (zh ? '收起导航' : 'Collapse navigation')"
            :title="sidebarCollapsed ? (zh ? '展开导航' : 'Expand navigation') : (zh ? '收起导航' : 'Collapse navigation')"
            :aria-expanded="!sidebarCollapsed"
            aria-controls="user-workspace-navigation"
            @click="toggleNavigation"
          >
            <Icon :name="sidebarCollapsed ? 'chevronRight' : 'chevronLeft'" size="sm" aria-hidden="true" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Workspace Area -->
    <div
      class="user-workspace flex min-w-0 flex-1 flex-col bg-[var(--atelier-surface)] transition-[padding-left] duration-200"
      :class="[
        desktop ? (sidebarCollapsed ? 'pl-[64px]' : 'pl-[232px]') : railViewport ? 'pl-[64px]' : 'pl-0'
      ]"
      :inert="overlayOpen ? true : undefined"
    >
      <!-- Unified Global 52px Topbar -->
      <header class="user-topbar sticky top-0 z-30 flex h-[52px] shrink-0 items-center justify-between gap-3 border-b border-[var(--atelier-border)] bg-[var(--atelier-surface)]/95 px-4 backdrop-blur-sm sm:px-6">
        <div class="flex min-w-0 items-center gap-2.5">
          <button
            v-if="!desktop"
            ref="navToggleRef"
            class="user-nav-toggle inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
            type="button"
            aria-controls="user-workspace-navigation"
            :aria-expanded="overlayOpen"
            :aria-label="zh ? '打开导航' : 'Open navigation'"
            @click="toggleNavigation"
          >
            <Icon name="menu" size="sm" aria-hidden="true" />
          </button>
          <div class="flex min-w-0 items-center gap-1.5 text-xs">
            <span class="hidden text-[var(--atelier-ink-muted)] sm:inline">{{ zh ? '工作台' : 'Workspace' }}</span>
            <span class="hidden text-[var(--atelier-ink-muted)] sm:inline" aria-hidden="true">/</span>
            <strong id="user-page-title" class="truncate font-semibold text-[var(--atelier-ink)]">{{ pageTitle }}</strong>
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-1.5">
          <WorkspaceCommandPalette :items="commands" />
          <button
            type="button"
            class="user-mode-toggle inline-flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
            :aria-label="isDark ? (zh ? '切换至浅色' : 'Switch to light mode') : (zh ? '切换至深色' : 'Switch to dark mode')"
            :title="isDark ? (zh ? '切换至浅色' : 'Switch to light mode') : (zh ? '切换至深色' : 'Switch to dark mode')"
            :aria-pressed="isDark"
            @click="toggleMode"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="sm" aria-hidden="true" />
          </button>

          <RouterLink
            v-if="user && !authStore.isSimpleMode"
            to="/purchase"
            class="user-balance-pill hidden h-8 items-center gap-1.5 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-canvas)] px-2.5 text-xs transition-colors hover:border-[var(--atelier-border-strong)] sm:inline-flex"
            :aria-label="(zh ? '可用余额 ' : 'Available balance ') + formatMoney(availableBalance)"
          >
            <span class="text-[var(--atelier-ink-muted)]">{{ zh ? '余额' : 'Balance' }}</span>
            <span class="font-mono font-semibold tabular-nums text-[var(--atelier-ink)]">{{ formatMoney(availableBalance) }}</span>
          </RouterLink>

          <div v-if="user">
            <AnnouncementBell />
          </div>

          <div v-if="user" ref="accountRef" class="relative">
            <button
              ref="accountToggleRef"
              type="button"
              class="user-account-trigger inline-flex h-8 items-center gap-1.5 rounded-lg px-1.5 text-xs text-[var(--atelier-ink)] transition-colors hover:bg-[var(--atelier-hover)]"
              :aria-label="displayName + ' — ' + (zh ? '账户菜单' : 'Account menu')"
              :aria-expanded="accountOpen"
              aria-controls="user-account-popover"
              @click="toggleAccount"
            >
              <img
                v-if="avatarUrl && !avatarFailed"
                :src="avatarUrl"
                :alt="displayName"
                class="h-6 w-6 rounded-full object-cover"
                @error="avatarFailed = true"
              />
              <span v-else class="inline-flex h-6 w-6 items-center justify-center rounded-full border border-[var(--atelier-border-strong)] bg-[var(--atelier-canvas)] text-[10px] font-semibold">
                {{ accountInitials }}
              </span>
              <span class="hidden max-w-[100px] truncate font-medium sm:inline">{{ displayName }}</span>
              <Icon name="chevronDown" size="xs" class="hidden text-[var(--atelier-ink-muted)] sm:inline" aria-hidden="true" />
            </button>

            <div
              v-if="accountOpen"
              id="user-account-popover"
              class="user-account-popover absolute right-0 top-full z-50 mt-2 w-56 rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] p-1.5 shadow-[var(--atelier-shadow-md)]"
            >
              <div class="border-b border-[var(--atelier-border-subtle)] px-3 py-2">
                <p class="truncate text-xs font-semibold text-[var(--atelier-ink)]">{{ displayName }}</p>
                <p class="truncate text-[11px] text-[var(--atelier-ink-muted)]">{{ user.email }}</p>
              </div>
              <div class="px-1.5 py-1">
                <MotionPreferenceToggle show-label />
              </div>
              <div v-if="frozenBalance > 0 && !authStore.isSimpleMode" class="flex items-center justify-between px-3 py-1.5 text-xs text-[var(--atelier-ink-muted)]">
                <span>{{ zh ? '冻结金额' : 'Frozen balance' }}</span>
                <span class="font-mono tabular-nums">{{ formatMoney(frozenBalance) }}</span>
              </div>
              <RouterLink
                to="/profile"
                class="user-account-action flex items-center gap-2 rounded-lg px-3 py-2 text-xs text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                @click="accountOpen = false"
              >
                <Icon name="user" size="sm" aria-hidden="true" />
                <span>{{ zh ? '账户与安全' : 'Account & security' }}</span>
              </RouterLink>
              <button
                v-if="!authStore.isAdmin && !authStore.isSimpleMode"
                type="button"
                class="user-account-action flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-xs text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                @click="replayUserGuide"
              >
                <Icon name="questionCircle" size="sm" aria-hidden="true" />
                <span>{{ zh ? 'API 接入引导' : 'API quick start' }}</span>
              </button>
              <button
                type="button"
                class="user-account-action user-signout flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-xs text-[var(--atelier-err)] hover:bg-[var(--atelier-err-bg)]"
                :disabled="loggingOut"
                @click="logout"
              >
                <Icon name="arrowRight" size="sm" aria-hidden="true" />
                <span>{{ zh ? '退出登录' : 'Sign out' }}</span>
              </button>
            </div>
          </div>

          <RouterLink v-else to="/login" class="atelier-btn atelier-btn-primary h-8 px-3 text-xs">
            {{ t('home.login') }}
          </RouterLink>
        </div>
      </header>

      <main
        id="user-workspace-content"
        ref="contentRef"
        class="mx-auto w-full max-w-[1360px] flex-1 px-4 py-6 pb-20 sm:px-8 md:pb-8"
        aria-labelledby="user-page-title"
        tabindex="-1"
      >
        <LiveActivityStrip v-if="user && !authStore.isSimpleMode" />
        <slot />
      </main>

      <!-- Mobile Bottom Navigation -->
      <nav
        v-if="!desktop && !railViewport && user"
        class="fixed inset-x-0 bottom-0 z-30 flex h-14 items-center justify-around border-t border-[var(--atelier-border)] bg-[var(--atelier-surface)]/95 px-2 backdrop-blur-md"
        :aria-label="zh ? '底部快捷导航' : 'Mobile quick navigation'"
      >
        <RouterLink
          v-for="item in mobileBottomItems"
          :key="item.path"
          :to="{ path: item.path, query: item.query }"
          class="flex flex-1 flex-col items-center justify-center gap-0.5 py-1 text-[11px]"
          :class="isActive(item) ? 'font-semibold text-[var(--atelier-ink)]' : 'text-[var(--atelier-ink-muted)]'"
          :aria-current="isActive(item) ? 'page' : undefined"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="truncate">{{ item.label }}</span>
        </RouterLink>
        <button
          type="button"
          class="flex flex-1 flex-col items-center justify-center gap-0.5 py-1 text-[11px]"
          :class="mobileOpen ? 'font-semibold text-[var(--atelier-ink)]' : 'text-[var(--atelier-ink-muted)]'"
          :aria-label="zh ? '更多菜单' : 'More navigation'"
          @click="toggleNavigation"
        >
          <Icon name="menu" size="sm" />
          <span>{{ zh ? '更多' : 'More' }}</span>
        </button>
      </nav>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import Icon from '@/components/icons/Icon.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import WorkspaceCommandPalette from '@/components/common/WorkspaceCommandPalette.vue'
import MotionPreferenceToggle from '@/components/common/MotionPreferenceToggle.vue'
import LiveActivityStrip from '@/components/portal/LiveActivityStrip.vue'
import { keysAPI } from '@/api/keys'
import type { ApiKey } from '@/types'
import { useClipboard } from '@/composables/useClipboard'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import type { WorkspaceCommand } from '@/utils/workspaceCommands'
import { availableLocales, setLocale } from '@/i18n'
import { useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import { useUserNavigation, type UserNavigationItem } from '@/composables/useUserNavigation'
import { useColorMode } from '@/composables/useColorMode'
import { useScrollLock } from '@/composables/useScrollLock'
import { useVisualViewport } from '@/composables/useVisualViewport'
import { useReducedMotion } from '@/composables/useReducedMotion'
import { prefetchRoutePath } from '@/composables/useRoutePrefetch'
import { resolveRouteMetaKeys } from '@/router/title'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { launchHeliosWorkbench } from '@/utils/heliosLaunch'
import { sanitizeSvg } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const { items: navigationItems, primaryItems, secondarySections, refreshBatchImageAccess } = useUserNavigation()
const navigationRef = ref<HTMLElement | null>(null)
const { copyToClipboard } = useClipboard()
const { openAIBase } = usePatrickApiBase()
const { isDark, toggleMode } = useColorMode()
const reducedMotion = useReducedMotion()
const quickKeys = ref<ApiKey[]>([])

const mobileBottomItems = computed(() => primaryItems.value.slice(0, 4))

const commands = computed<WorkspaceCommand[]>(() => {
  const entries: WorkspaceCommand[] = navigationItems.value.map((item) => ({
    id: item.path,
    label: item.label,
    section: zh.value ? '页面' : 'Pages',
    keywords: item.path,
    ...(item.action
      ? { run: () => activateNavigation(item) }
      : item.newTab
        ? {
            run: () => {
              window.open(item.href || item.path, '_blank', 'noopener,noreferrer')
            }
          }
        : item.href
          ? { href: item.href }
          : { to: { path: item.path, query: item.query } })
  }))
  entries.unshift({
    id: 'toggle-color-mode',
    label: isDark.value
      ? zh.value
        ? '切换至浅色外观'
        : 'Switch to light theme'
      : zh.value
        ? '切换至深色外观'
        : 'Switch to dark theme',
    section: zh.value ? '快捷操作' : 'Quick actions',
    keywords: 'theme dark light mode color appearance',
    run: () => {
      toggleMode()
    }
  })
  if (navigationItems.value.some((item) => item.path === '/playground')) {
    entries.unshift({
      id: 'open-playground',
      label: zh.value ? '打开 API 调试台试跑模型' : 'Test a model in API Playground',
      section: zh.value ? '快捷操作' : 'Quick actions',
      keywords: 'playground debug test model prompt curl',
      to: '/playground'
    })
  }
  if (navigationItems.value.some((item) => item.path === '/keys')) {
    entries.unshift({
      id: 'create-key',
      label: zh.value ? '创建 API 密钥' : 'Create API key',
      section: zh.value ? '快捷操作' : 'Quick actions',
      keywords: 'new api key',
      to: '/keys?create=1'
    })
  }
  entries.unshift({
    id: 'copy-endpoint',
    label: zh.value ? '复制接口地址' : 'Copy API base URL',
    section: zh.value ? '快捷操作' : 'Quick actions',
    keywords: 'api base url endpoint',
    run: async () => {
      if (!(await copyToClipboard(openAIBase.value))) throw new Error('Copy failed')
    }
  })
  for (const key of quickKeys.value.slice(0, 12)) {
    if (key.status !== 'active' || !key.key) continue
    entries.push({
      id: `copy-key-${key.id}`,
      label: zh.value ? `复制密钥：${key.name}` : `Copy key: ${key.name}`,
      section: zh.value ? '我的 API 密钥' : 'My API keys',
      keywords: `key token copy ${key.name} ${key.group?.name || ''}`,
      run: async () => {
        if (!(await copyToClipboard(key.key))) throw new Error('Copy failed')
        appStore.showSuccess?.(zh.value ? `已复制密钥 ${key.name}` : `Copied key ${key.name}`)
      }
    })
  }
  if (docUrl.value) {
    entries.push({
      id: 'documentation',
      label: zh.value ? '开发者文档' : 'Documentation',
      section: zh.value ? '资源' : 'Resources',
      href: docUrl.value
    })
  }
  return entries
})

const desktop = useMediaQuery('(min-width: 1200px)')
const railViewport = useMediaQuery('(min-width: 768px)')
const collapsed = ref(false)
const mobileOpen = ref(false)
const overlayOpen = computed(() => mobileOpen.value && !desktop.value)
useScrollLock(overlayOpen)
const viewport = useVisualViewport(overlayOpen)
const sidebarCollapsed = computed(() => (desktop.value ? collapsed.value : railViewport.value && !mobileOpen.value))
const accountOpen = ref(false)
const localeOpen = ref(false)
const localeSwitching = ref(false)
const loggingOut = ref(false)
const avatarFailed = ref(false)
const sidebarRef = ref<HTMLElement | null>(null)
const drawerCloseRef = ref<HTMLButtonElement | null>(null)
const navToggleRef = ref<HTMLButtonElement | null>(null)
const desktopToggleRef = ref<HTMLButtonElement | null>(null)
const contentRef = ref<HTMLElement | null>(null)
const accountRef = ref<HTMLElement | null>(null)
const accountToggleRef = ref<HTMLButtonElement | null>(null)
const localeRef = ref<HTMLElement | null>(null)
const localeToggleRef = ref<HTMLButtonElement | null>(null)
const localeMenuRef = ref<HTMLElement | null>(null)

const user = computed(() => authStore.user)
const avatarUrl = computed(() => sanitizeUrl((user.value as { avatar_url?: string } | null)?.avatar_url || ''))
watch(avatarUrl, () => {
  avatarFailed.value = false
})
const displayName = computed(() => user.value?.username || user.value?.email?.split('@')[0] || (zh.value ? '我的账户' : 'My account'))
const accountInitials = computed(() => displayName.value.slice(0, 2).toUpperCase())
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const supportEmail = computed(() => appStore.contactInfo?.match(/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/i)?.[0] || '')
const availableBalance = computed(() => Number(user.value?.balance || 0))
const frozenBalance = computed(() => Number(user.value?.frozen_balance || 0))
const currentLocaleCode = computed(() => (zh.value ? 'zh' : 'en'))
const pageTitle = computed(() => {
  if (route.path === '/dashboard') return zh.value ? '总览' : 'Overview'
  if (route.path === '/keys') return zh.value ? 'API 密钥' : 'API keys'
  if (route.path === '/playground') return zh.value ? 'API 调试台' : 'API Playground'
  if (route.path === '/chat-models') return zh.value ? '聊天模型' : 'Chat models'
  if (route.path === '/usage' || route.path === '/traffic') return zh.value ? '用量与流量' : 'Usage & traffic'
  if (route.name === 'CustomPage') {
    const item = appStore.cachedPublicSettings?.custom_menu_items?.find((menu) => menu.id === route.params.id)
    if (item?.label) return item.label
  }
  const { titleKey } = resolveRouteMetaKeys(route, { billingMode: resolveSiteBillingMode(appStore.cachedPublicSettings) })
  return titleKey ? t(titleKey) : String(route.meta.title || 'Sub2API')
})

function formatMoney(value: number) {
  return 'USD ' + (Number.isFinite(value) ? value : 0).toFixed(2)
}

function isActive(item: UserNavigationItem) {
  return (item.activePaths || [item.path]).some((path) => route.path === path || route.path.startsWith(path + '/'))
}

function focusNavigationToggle() {
  const toggle = desktop.value ? desktopToggleRef.value : navToggleRef.value
  toggle?.focus()
}

function closeMobile(restoreFocus = true) {
  if (!mobileOpen.value) return
  mobileOpen.value = false
  closeLocale(false)
  if (restoreFocus) void nextTick(focusNavigationToggle)
}

function closeCurrentRoute(path: string) {
  if (route.path === path) closeMobile()
}

function openNavigation() {
  accountOpen.value = false
  closeLocale(false)
  mobileOpen.value = true
  void nextTick(() => (drawerCloseRef.value || sidebarRef.value)?.focus())
}

function toggleNavigation() {
  if (desktop.value) {
    closeLocale(false)
    collapsed.value = !collapsed.value
  } else if (mobileOpen.value) {
    closeMobile()
  } else {
    openNavigation()
  }
}

function handleDrawerKeydown(event: KeyboardEvent) {
  if (!overlayOpen.value) return
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    closeMobile()
    return
  }
  if (event.key !== 'Tab') return
  const controls = Array.from(sidebarRef.value?.querySelectorAll<HTMLElement>('a[href], button:not(:disabled), [tabindex="0"]') || [])
  const first = controls[0]
  const last = controls[controls.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last?.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first?.focus()
  }
}

function activateNavigation(item: UserNavigationItem) {
  if (item.action === 'helios') {
    void launchHeliosWorkbench({
      mode: 'popup',
      notify: (failure) => appStore.showError(t(failure === 'popup-blocked' ? 'helios.popupBlocked' : 'helios.launchFailed'))
    })
    closeMobile()
  } else if (item.href) {
    closeMobile(false)
  } else if (router.resolve({ path: item.path, query: item.query }).fullPath === route.fullPath) {
    closeMobile()
  }
  if (item.path === '/keys' && onboardingStore.isCurrentStep('[data-tour="sidebar-my-keys"]')) {
    void onboardingStore.nextStep(500)
  }
}

function prefetchItem(item: UserNavigationItem, event?: PointerEvent) {
  if (item.action || item.href || event?.pointerType === 'touch') return
  prefetchRoutePath(router, item.path)
}

function toggleAccount() {
  closeLocale(false)
  accountOpen.value = !accountOpen.value
}

function closeLocale(restoreFocus = true) {
  if (!localeOpen.value) return
  localeOpen.value = false
  if (restoreFocus) void nextTick(() => localeToggleRef.value?.focus())
}

function toggleLocale() {
  if (localeOpen.value) {
    closeLocale()
    return
  }
  accountOpen.value = false
  localeOpen.value = true
  void nextTick(() => localeMenuRef.value?.querySelector<HTMLButtonElement>('[aria-checked="true"]')?.focus())
}

async function changeLocale(code: string) {
  if (localeSwitching.value) return
  if (code === currentLocaleCode.value) {
    closeLocale()
    return
  }
  localeSwitching.value = true
  try {
    await setLocale(code)
  } catch {
    appStore.showError(zh.value ? '语言切换失败，请重试' : 'Could not change language. Please try again.')
  } finally {
    localeSwitching.value = false
    closeLocale()
  }
}

function handleLocaleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    closeLocale()
    return
  }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const buttons = Array.from(localeMenuRef.value?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') || [])
  if (!buttons.length) return
  event.preventDefault()
  const current = buttons.indexOf(document.activeElement as HTMLButtonElement)
  const index =
    event.key === 'Home'
      ? 0
      : event.key === 'End'
        ? buttons.length - 1
        : (current + (event.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length
  buttons[index]?.focus()
}

async function replayUserGuide() {
  accountOpen.value = false
  if (authStore.isAdmin || authStore.isSimpleMode) return
  closeMobile()
  await router.push({ path: '/keys', hash: '#api-quick-start' })
  await nextTick()
  document
    .getElementById('api-quick-start')
    ?.scrollIntoView({ behavior: reducedMotion.value ? 'auto' : 'smooth', block: 'start' })
}

async function logout() {
  if (loggingOut.value) return
  loggingOut.value = true
  accountOpen.value = false
  try {
    await authStore.logout()
  } catch {
    // Session cleared locally even if server revocation fails.
  } finally {
    loggingOut.value = false
    await router.push('/login')
  }
}

function handlePointerDown(event: PointerEvent) {
  if (!(event.target instanceof Node)) return
  if (accountOpen.value && !accountRef.value?.contains(event.target)) accountOpen.value = false
  if (localeOpen.value && !localeRef.value?.contains(event.target)) closeLocale(false)
}

function handleWindowKeydown(event: KeyboardEvent) {
  if (
    event.key !== 'Escape' ||
    event.defaultPrevented ||
    (event.target instanceof Element && event.target.closest('dialog[open]'))
  ) {
    return
  }
  if (localeOpen.value) {
    closeLocale()
  } else if (accountOpen.value) {
    accountOpen.value = false
    accountToggleRef.value?.focus()
  } else if (mobileOpen.value) {
    closeMobile()
  }
}

watch(
  () => route.fullPath,
  () => {
    const focusContent = overlayOpen.value
    closeMobile(false)
    closeLocale(false)
    accountOpen.value = false
    if (focusContent) void nextTick(() => contentRef.value?.focus())
  },
  { immediate: true }
)

watch([desktop, railViewport], ([isDesktop, hasRail]) => {
  const focusedInSidebar = sidebarRef.value?.contains(document.activeElement)
  closeLocale(!!localeMenuRef.value?.contains(document.activeElement))
  if (isDesktop && mobileOpen.value) {
    closeMobile(false)
    if (focusedInSidebar) void nextTick(focusNavigationToggle)
  } else if (!isDesktop && !hasRail && !mobileOpen.value && focusedInSidebar) {
    void nextTick(focusNavigationToggle)
  }
})

async function loadQuickKeys() {
  if (!authStore.isAuthenticated) {
    quickKeys.value = []
    return
  }
  try {
    const res = await keysAPI.list(1, 20)
    quickKeys.value = Array.isArray(res?.items) ? res.items : []
  } catch {
    // Non-critical enhancement for command palette
  }
}

watch(
  () => authStore.user?.id,
  () => {
    void refreshBatchImageAccess(true)
    void loadQuickKeys()
  }
)

onMounted(() => {
  void refreshBatchImageAccess()
  void loadQuickKeys()
  if (!authStore.isAdmin) {
    onboardingStore.setReplayCallback(() => {
      void replayUserGuide()
    })
  }
  document.addEventListener('pointerdown', handlePointerDown)
  window.addEventListener('keydown', handleWindowKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handlePointerDown)
  window.removeEventListener('keydown', handleWindowKeydown)
})

defineExpose({ replayTour: replayUserGuide })
</script>
