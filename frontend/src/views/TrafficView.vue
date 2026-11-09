<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import TrafficPanel from '@/components/usage/TrafficPanel.vue'
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
withDefaults(defineProps<{ admin?: boolean }>(), { admin: false })
const { locale } = useI18n()
</script>
<template>
  <div class="portal-page traffic-page">
    <PortalPageHeader :title="locale.startsWith('zh') ? '流量记录' : 'Traffic records'" :description="locale.startsWith('zh') ? 'HTTP/SSE 应用流量' : 'HTTP/SSE application traffic'" />
    <TrafficPanel :admin="admin" />
  </div>
</template>
