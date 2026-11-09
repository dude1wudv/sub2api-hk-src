<template>
  <div class="space-y-6">
  <ModelQualityResults v-if="isV2" class="portal-page" />
  <ChannelStatusV1View v-if="isV1" />
  <ChannelStatusV2View v-else />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelQualityResults from '@/components/usage/ModelQualityResults.vue'
import { isChannelMonitorV1Mode, isChannelMonitorV2Mode } from '@/utils/featureFlags'
import ChannelStatusV1View from './ChannelStatusV1View.vue'
import ChannelStatusV2View from './ChannelStatusV2View.vue'

const isV1 = computed(() => isChannelMonitorV1Mode())
const isV2 = computed(() => isChannelMonitorV2Mode())
</script>
