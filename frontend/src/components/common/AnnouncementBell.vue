<template>
  <div>
    <!-- 铃铛按钮 -->
    <button
      @click="openModal"
      class="relative flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
      :class="{ 'text-[var(--atelier-ink)]': unreadCount > 0 }"
      :aria-label="t('announcements.title')"
    >
      <Icon name="bell" size="sm" />
      <!-- 6px 未读微点（无数字大红块） -->
      <span
        v-if="unreadCount > 0"
        class="absolute right-1.5 top-1.5 inline-flex h-1.5 w-1.5 rounded-full bg-[var(--atelier-err)]"
      />
    </button>

    <!-- 公告列表 Modal -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div
          v-if="isModalOpen"
          class="announcement-overlay fixed z-[100] flex items-center justify-center bg-[var(--atelier-scrim)] p-3 backdrop-blur-[2px]"
          :style="viewportStyle"
          @click="closeModal"
        >
          <div
            class="announcement-panel w-full max-w-[600px] overflow-hidden rounded-2xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] shadow-[var(--atelier-shadow-md)]"
            @click.stop
          >
            <!-- Clean Editorial Header -->
            <div class="border-b border-[var(--atelier-border)] bg-[var(--atelier-canvas)] px-6 py-4">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="flex items-center gap-2">
                    <Icon name="bell" size="sm" class="text-[var(--atelier-ink)]" />
                    <h2 class="text-base font-semibold text-[var(--atelier-ink)]">
                      {{ t('announcements.title') }}
                    </h2>
                  </div>
                  <p v-if="unreadCount > 0" class="mt-1 text-xs text-[var(--atelier-ink-secondary)]">
                    <span class="font-mono font-semibold tabular-nums text-[var(--atelier-ink)]">{{ unreadCount }}</span>
                    {{ t('announcements.unread') }}
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <button
                    v-if="unreadCount > 0"
                    @click="markAllAsRead"
                    :disabled="loading"
                    class="atelier-btn atelier-btn-primary h-8 px-3 text-xs"
                  >
                    {{ t('announcements.markAllRead') }}
                  </button>
                  <button
                    @click="closeModal"
                    class="flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                    :aria-label="t('common.close')"
                  >
                    <Icon name="x" size="sm" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Body -->
            <div class="announcement-body overflow-y-auto">
              <!-- Loading -->
              <div v-if="loading" class="flex items-center justify-center py-14">
                <div class="h-6 w-6 animate-spin rounded-full border-2 border-[var(--atelier-border-strong)] border-t-[var(--atelier-ink)]" />
              </div>

              <!-- Announcements List -->
              <div v-else-if="announcements.length > 0">
                <div
                  v-for="item in announcements"
                  :key="item.id"
                  class="group relative flex cursor-pointer items-center gap-3.5 border-b border-[var(--atelier-border-subtle)] px-6 py-3.5 transition-colors hover:bg-[var(--atelier-hover)]"
                  :class="{ 'bg-[var(--atelier-canvas)]/60': !item.read_at }"
                  @click="openDetail(item)"
                >
                  <!-- Semantic Status Dot -->
                  <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center">
                    <span
                      class="h-2 w-2 rounded-full"
                      :class="!item.read_at ? 'bg-[var(--atelier-ink)]' : 'bg-[var(--atelier-border-strong)]'"
                    />
                  </div>

                  <!-- Content -->
                  <div class="flex min-w-0 flex-1 items-center justify-between gap-4">
                    <div class="min-w-0 flex-1">
                      <h3 class="truncate text-sm font-medium text-[var(--atelier-ink)]">
                        {{ item.title }}
                      </h3>
                      <div class="mt-1 flex items-center gap-2">
                        <time class="text-xs text-gray-500 dark:text-gray-400">
                          {{ formatRelativeTime(item.created_at) }}
                        </time>
                        <span
                          v-if="!item.read_at"
                          class="inline-flex items-center gap-1 rounded-md bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/40 dark:text-blue-300"
                        >
                          <span class="relative flex h-1.5 w-1.5">
                            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-500 opacity-75"></span>
                            <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-blue-600"></span>
                          </span>
                          {{ t('announcements.unread') }}
                        </span>
                      </div>
                    </div>

                    <!-- Arrow -->
                    <div class="flex-shrink-0">
                      <svg
                        class="h-5 w-5 text-gray-400 transition-transform group-hover:translate-x-1 dark:text-gray-600"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
                      </svg>
                    </div>
                  </div>

                  <!-- Unread indicator bar -->
                  <div
                    v-if="!item.read_at"
                    class="absolute left-0 top-0 h-full w-1 bg-gradient-to-b from-blue-500 to-indigo-600"
                  ></div>
                </div>
              </div>

              <!-- Empty State -->
              <div v-else class="flex flex-col items-center justify-center py-16">
                <div class="relative mb-4">
                  <div class="flex h-20 w-20 items-center justify-center rounded-full bg-gradient-to-br from-gray-100 to-gray-200 dark:from-dark-700 dark:to-dark-600">
                    <Icon name="inbox" size="xl" class="text-gray-400 dark:text-gray-500" />
                  </div>
                  <div class="absolute -right-1 -top-1 flex h-6 w-6 items-center justify-center rounded-full bg-green-500 text-white">
                    <svg class="h-3.5 w-3.5" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                    </svg>
                  </div>
                </div>
                <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('announcements.empty') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('announcements.emptyDescription') }}</p>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 公告详情 Modal -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div
          v-if="detailModalOpen && selectedAnnouncement"
          class="announcement-overlay fixed z-[110] flex items-center justify-center bg-[var(--atelier-scrim)] p-3 backdrop-blur-[2px]"
          :style="viewportStyle"
          @click="closeDetail"
        >
          <div
            class="announcement-panel w-full max-w-[740px] overflow-hidden rounded-2xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] shadow-[var(--atelier-shadow-md)]"
            @click.stop
          >
            <!-- Clean Editorial Detail Header -->
            <div class="border-b border-[var(--atelier-border)] bg-[var(--atelier-canvas)] px-6 py-5">
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <div class="mb-2 flex items-center gap-2">
                    <span class="rounded-full border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-2.5 py-0.5 text-[11px] font-medium text-[var(--atelier-ink-secondary)]">
                      {{ t('announcements.title') }}
                    </span>
                    <span
                      v-if="!selectedAnnouncement.read_at"
                      class="inline-flex items-center gap-1.5 rounded-full bg-[var(--atelier-subtle)] px-2.5 py-0.5 text-[11px] font-medium text-[var(--atelier-ink)]"
                    >
                      <span class="h-1.5 w-1.5 rounded-full bg-[var(--atelier-ink)]" />
                      {{ t('announcements.unread') }}
                    </span>
                  </div>

                  <h2 class="text-xl font-semibold leading-snug tracking-tight text-[var(--atelier-ink)]">
                    {{ selectedAnnouncement.title }}
                  </h2>

                  <div class="mt-2 flex items-center gap-4 text-xs text-[var(--atelier-ink-muted)]">
                    <time>{{ formatRelativeWithDateTime(selectedAnnouncement.created_at) }}</time>
                    <span>{{ selectedAnnouncement.read_at ? t('announcements.read') : t('announcements.unread') }}</span>
                  </div>
                </div>

                <button
                  @click="closeDetail"
                  class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                  :aria-label="t('common.close')"
                >
                  <Icon name="x" size="sm" />
                </button>
              </div>
            </div>

            <!-- Body -->
            <div class="announcement-body overflow-y-auto bg-[var(--atelier-surface)] px-6 py-6">
              <div class="border-l-2 border-[var(--atelier-border-strong)] pl-5">
                <div
                  class="markdown-body prose prose-sm max-w-none dark:prose-invert"
                  v-html="renderMarkdown(selectedAnnouncement.content)"
                ></div>
              </div>
            </div>

            <!-- Footer -->
            <div class="border-t border-[var(--atelier-border)] bg-[var(--atelier-canvas)] px-6 py-4">
              <div class="flex items-center justify-between">
                <span class="text-xs text-[var(--atelier-ink-muted)]">
                  {{ selectedAnnouncement.read_at ? t('announcements.readStatus') : t('announcements.markReadHint') }}
                </span>
                <div class="flex items-center gap-3">
                  <button
                    @click="closeDetail"
                    class="rounded-xl border border-gray-300 bg-white px-5 py-2.5 text-sm font-medium text-gray-700 shadow-sm transition-all hover:bg-gray-50 hover:shadow dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600"
                  >
                    {{ t('common.close') }}
                  </button>
                  <button
                    v-if="!selectedAnnouncement.read_at"
                    @click="markAsReadAndClose(selectedAnnouncement.id)"
                    class="rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 px-5 py-2.5 text-sm font-medium text-white shadow-lg shadow-blue-500/30 transition-all hover:shadow-xl hover:scale-105"
                  >
                    <span class="flex items-center gap-2">
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                      {{ t('announcements.markRead') }}
                    </span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import type { UserAnnouncement } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import '@/styles/announcement-markdown.css'
import { useScrollLock } from '@/composables/useScrollLock'
import { useVisualViewport } from '@/composables/useVisualViewport'

const { t } = useI18n()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()

// Configure marked
marked.setOptions({
  breaks: true,
  gfm: true,
})

// Use store state (storeToRefs for reactivity)
const { announcements, loading } = storeToRefs(announcementStore)
const unreadCount = computed(() => announcementStore.unreadCount)

// Local modal state
const isModalOpen = ref(false)
const detailModalOpen = ref(false)
const selectedAnnouncement = ref<UserAnnouncement | null>(null)
useScrollLock(() => isModalOpen.value || detailModalOpen.value)
const viewport = useVisualViewport(() => isModalOpen.value || detailModalOpen.value)
const viewportStyle = computed(() => ({ top: `${viewport.value.top}px`, left: `${viewport.value.left}px`, width: `${viewport.value.width}px`, height: `${viewport.value.height}px` }))

// Methods
function renderMarkdown(content: string): string {
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
}

function openModal() {
  isModalOpen.value = true
}

function closeModal() {
  isModalOpen.value = false
}

function openDetail(announcement: UserAnnouncement) {
  selectedAnnouncement.value = announcement
  detailModalOpen.value = true
  if (!announcement.read_at) {
    markAsRead(announcement.id)
  }
}

function closeDetail() {
  detailModalOpen.value = false
  selectedAnnouncement.value = null
}

async function markAsRead(id: number) {
  try {
    const marked = await announcementStore.markAsRead(id)
    if (!marked) appStore.showError(t('common.unknownError'))
    return marked
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
    return false
  }
}

async function markAsReadAndClose(id: number) {
  if (!await markAsRead(id)) return
  appStore.showSuccess(t('announcements.markedAsRead'))
  closeDetail()
}

async function markAllAsRead() {
  try {
    await announcementStore.markAllAsRead()
    appStore.showSuccess(t('announcements.allMarkedAsRead'))
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

function handleEscape(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (detailModalOpen.value) {
      closeDetail()
    } else if (isModalOpen.value) {
      closeModal()
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEscape)
})

</script>

<style scoped>
.announcement-panel { display: flex; flex-direction: column; min-width: 0; min-height: 0; max-height: 100%; overflow-wrap: anywhere; }
.announcement-panel > :not(.announcement-body) { flex-shrink: 0; }
.announcement-body { min-height: 0; overscroll-behavior: contain; }
@media (max-width: 767px), (max-width: 1023px) and (max-height: 500px) and (pointer: coarse) {
  .announcement-panel > div { padding: 16px; }
  .announcement-panel button { min-width: 44px; min-height: 44px; }
  .announcement-panel .justify-between { flex-wrap: wrap; gap: 8px; }
}
/* Modal Animations */
.modal-fade-enter-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-fade-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 1, 1);
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-from > div {
  transform: scale(0.94) translateY(-12px);
  opacity: 0;
}

.modal-fade-leave-to > div {
  transform: scale(0.96) translateY(-8px);
  opacity: 0;
}

/* Scrollbar Styling */
.overflow-y-auto::-webkit-scrollbar {
  width: 8px;
}

.overflow-y-auto::-webkit-scrollbar-track {
  background: transparent;
}

.overflow-y-auto::-webkit-scrollbar-thumb {
  background: linear-gradient(to bottom, #cbd5e1, #94a3b8);
  border-radius: 4px;
}

.dark .overflow-y-auto::-webkit-scrollbar-thumb {
  background: linear-gradient(to bottom, #4b5563, #374151);
}

.overflow-y-auto::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(to bottom, #94a3b8, #64748b);
}

.dark .overflow-y-auto::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(to bottom, #6b7280, #4b5563);
}
</style>
