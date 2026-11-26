<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    class="shrink-0"
  >
    <path
      d="m19.25 7.65-2.55-3.2a1.33 1.33 0 0 0-1.03-.5H8.58c-.34 0-.67.13-.91.37L4.15 7.65c-.93.88-.6 1.52.65 2.3l9.55 5.97"
      stroke="currentColor"
      stroke-width="2.1"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="m4.75 16.35 2.55 3.2c.25.31.63.5 1.03.5h7.09c.34 0 .67-.13.91-.37l3.52-3.33c.93-.88.6-1.52-.65-2.3L9.65 8.08"
      stroke="currentColor"
      stroke-width="2.1"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: number | string
  }>(),
  { size: 22 }
)
</script>
