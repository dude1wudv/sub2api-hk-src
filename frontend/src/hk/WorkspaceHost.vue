<script setup lang="ts">
import { inject } from 'vue'
import { RouterView } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AdminShell from '@/components/layout/AdminShell.vue'
import UserShell from '@/components/layout/UserShell.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useAdminSettingsStore } from '@/stores'
import { workspaceChromeKey } from './useWorkspaceChrome'

const chrome = inject(workspaceChromeKey)
if (!chrome) throw new Error('WorkspaceHost requires useWorkspaceChrome()')
const adminSettingsStore = useAdminSettingsStore()
const { locale } = useI18n()
</script>

<template>
  <RouterView v-slot="{ Component, route: viewRoute }">
    <div v-if="chrome.adminCustomPending.value" role="status" data-testid="workspace-surface-loading" class="flex min-h-screen flex-col items-center justify-center gap-4 p-6 text-center">
      <LoadingSpinner v-if="adminSettingsStore.loading" size="md" />
      <p>{{ adminSettingsStore.loading ? (locale.startsWith('zh') ? '正在载入工作空间…' : 'Loading workspace…') : (locale.startsWith('zh') ? '暂时无法载入工作空间。' : 'Unable to load this workspace.') }}</p>
      <button v-if="!adminSettingsStore.loading" type="button" class="btn btn-secondary" @click="adminSettingsStore.fetch(true)">{{ locale.startsWith('zh') ? '重新加载' : 'Try again' }}</button>
    </div>
    <AdminShell v-else-if="chrome.usesAppShell(viewRoute) && chrome.usesAdminShell(viewRoute)">
      <component :is="Component" />
    </AdminShell>
    <UserShell v-else-if="chrome.usesAppShell(viewRoute)">
      <component :is="Component" />
    </UserShell>
    <component :is="Component" v-else />
  </RouterView>
</template>
