<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { ChatHostBridge } from '@/api/chatBridge'

const auth = useAuthStore()
const router = useRouter()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const frame = ref<HTMLIFrameElement | null>(null)
const sessionUserID = ref<number | null>(auth.user?.id ?? null)
const signingOut = ref(false)
const bridge = new ChatHostBridge({
  frame: () => frame.value,
  userID: () => auth.token && !signingOut.value ? auth.user?.id ?? sessionUserID.value : null,
  locale: () => locale.value,
  navigate: (path) => router.push(path),
  onSessionInvalid: () => { signingOut.value = true; sessionUserID.value = null },
  logout: async () => {
    signingOut.value = true
    sessionUserID.value = null
    await auth.logout()
    await router.replace('/login')
  },
})
const stopActions = auth.$onAction(({ name }) => {
  if (name !== 'logout') return
  bridge.disconnect()
  signingOut.value = true
  sessionUserID.value = null
})

// setToken briefly clears user while /auth/me reloads. A same-user refresh must
// not unmount the iframe or abort a paid generation; logout/account switch must.
watch([() => auth.token, () => auth.user?.id], ([token, userID]) => {
  if (!token || (userID !== undefined && userID !== sessionUserID.value)) {
    bridge.disconnect()
    sessionUserID.value = token && userID !== undefined ? userID : null
    if (!token) void router.replace('/login')
  }
})

onMounted(() => bridge.mount())
onBeforeUnmount(() => { stopActions(); bridge.destroy() })
</script>

<template>
  <!-- No app shell or fixed overlay: App.vue's toast/announcement/compliance layers remain mounted. -->
  <main class="chat-host" :aria-label="zh ? '站内聊天' : 'Web chat'">
    <nav class="chat-host__navigation" :aria-label="zh ? '聊天导航' : 'Chat navigation'">
      <RouterLink to="/dashboard" class="chat-host__link" data-testid="chat-back">
        <span aria-hidden="true">←</span> {{ zh ? '返回用户中心' : 'Back to dashboard' }}
      </RouterLink>
      <RouterLink to="/chat-models" class="chat-host__link">{{ zh ? '聊天模型' : 'Chat models' }}</RouterLink>
    </nav>
    <iframe
      v-if="sessionUserID && !signingOut"
      :key="sessionUserID"
      ref="frame"
      src="/chat-app/"
      :title="zh ? '聊天' : 'Chat'"
      class="chat-host__frame"
      referrerpolicy="same-origin"
      allow="microphone 'none'; camera 'none'; geolocation 'none'"
      @load="bridge.frameLoaded"
    />
    <p v-else role="status" class="p-6">
      {{ zh ? '聊天连接已关闭。' : 'The chat connection is closed.' }}
      <a href="/dashboard" class="underline">{{ zh ? '重新进入用户中心' : 'Reopen the dashboard' }}</a>
    </p>
  </main>
</template>

<style scoped>
.chat-host {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
}
.chat-host__navigation {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  min-height: 2.75rem;
  padding: env(safe-area-inset-top, 0px) max(0.75rem, env(safe-area-inset-right, 0px)) 0 max(0.75rem, env(safe-area-inset-left, 0px));
  border-bottom: 1px solid rgb(128 128 128 / 20%);
}
.chat-host__link {
  padding: 0.5rem 0;
  font-size: 0.875rem;
}
.chat-host__link:focus-visible { outline: 2px solid currentColor; outline-offset: 2px; }
.chat-host__frame { width: 100%; flex: 1 1 0; min-height: 0; border: 0; }
</style>
