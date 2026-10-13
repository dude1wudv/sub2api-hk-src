<template>
  <div class="distribution-layout" :style="{ '--distribution-table-min-width': `${columns <= 4 ? 26 : columns === 5 ? 29 : 32}rem` }">
    <div class="distribution-content">
      <div class="distribution-chart">
        <slot name="chart" />
      </div>
      <div
        class="distribution-table-scroll"
        role="region"
        :aria-label="tableLabel"
        tabindex="0"
      >
        <slot />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  tableLabel: string
  columns?: number
}>(), { columns: 6 })
</script>

<style scoped>
/* Size against the card, not the viewport. Keep these rules local so the
   global chart container query cannot disagree with a responsive flex class. */
.distribution-layout {
  container-type: inline-size;
  min-width: 0;
  width: 100%;
  max-width: 100%;
}

.distribution-content {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: start;
  gap: 1rem;
  min-width: 0;
}

.distribution-chart {
  justify-self: center;
  width: 11rem;
  max-width: 100%;
  height: 11rem;
  min-width: 0;
}

.distribution-table-scroll {
  min-width: 0;
  width: 100%;
  max-width: 100%;
  max-height: 16rem;
  overflow: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
}

.distribution-table-scroll:focus-visible {
  outline: 2px solid rgb(var(--color-primary-500));
  outline-offset: 2px;
}

.distribution-table-scroll :deep(table) {
  width: 100%;
  min-width: var(--distribution-table-min-width);
}

.distribution-table-scroll :deep(th),
.distribution-table-scroll :deep(td) {
  padding-right: 0.5rem;
  padding-left: 0.5rem;
  white-space: nowrap;
}

.distribution-table-scroll :deep(.distribution-name) {
  min-width: 8rem;
  max-width: 12rem;
  white-space: normal;
  overflow-wrap: anywhere;
}

.distribution-table-scroll :deep(td[colspan]) {
  padding: 0;
}

@container (min-width: 46rem) {
  .distribution-content {
    grid-template-columns: 12rem minmax(0, 1fr);
    gap: 1.5rem;
  }

  .distribution-chart {
    width: 12rem;
    height: 12rem;
  }
}
</style>
