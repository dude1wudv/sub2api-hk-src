<template>
  <div class="empty-state">
    <!-- Icon -->
    <div
      class="empty-state-emblem"
    >
      <slot name="icon">
        <component v-if="icon" :is="icon" class="empty-state-icon h-10 w-10" aria-hidden="true" />
        <svg
          v-else
          class="empty-state-icon h-10 w-10"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="1.5"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
          />
        </svg>
      </slot>
    </div>

    <!-- Title -->
    <h3 class="empty-state-title">
      {{ displayTitle }}
    </h3>

    <!-- Description -->
    <p class="empty-state-description">
      {{ description }}
    </p>

    <!-- Action -->
    <div v-if="actionText || $slots.action" class="mt-6">
      <slot name="action">
        <component
          :is="actionTo ? 'RouterLink' : 'button'"
          v-if="actionText"
          :to="actionTo"
          @click="!actionTo && $emit('action')"
          class="btn btn-primary"
        >
          <Icon v-if="actionIcon" name="plus" size="md" class="mr-2" />
          {{ actionText }}
        </component>
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Component } from 'vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

interface Props {
  icon?: Component | string
  title?: string
  description?: string
  actionText?: string
  actionTo?: string | object
  actionIcon?: boolean
  message?: string
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  actionIcon: true
})

const displayTitle = computed(() => props.title || t('common.noData'))

defineEmits(['action'])
</script>

<style scoped>
.empty-state-emblem { position:relative; display:grid; place-items:center; width:72px; height:72px; margin-bottom:20px; border:1px solid rgb(var(--brand-moss) / .18); border-radius:26px 26px 36px 18px; background:rgb(var(--brand-moss) / .07); color:rgb(var(--brand-moss)); }
.empty-state-emblem::after { content:''; position:absolute; bottom:-3px; right:-3px; width:9px; height:9px; border-radius:50%; background:rgb(var(--brand-copper)); border:2px solid rgb(var(--surface)); }
.empty-state-emblem :deep(svg) { color:inherit; width:32px; height:32px; }
.empty-state-title { color:rgb(var(--ink)); font:550 18px/1.5 var(--font-sans); }
.empty-state-description { max-width:400px; color:rgb(var(--ink-3)); font-size:14px; line-height:1.8; }
</style>
