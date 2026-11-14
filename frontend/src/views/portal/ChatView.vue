<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
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
.chat-host__frame { width: 100%; flex: 1 1 0; min-height: 0; border: 0; }
</style>
