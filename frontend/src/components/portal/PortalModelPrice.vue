<script setup lang="ts">
defineProps<{
  base: string
  adjusted?: string
  adjustedLabel?: string
  caption?: string
  unit?: string
}>()
</script>

<template>
  <span class="portal-model-price" :class="{ 'has-adjusted': adjusted !== undefined }">
    <span v-if="adjusted !== undefined" class="portal-model-price-adjusted">
      <span v-if="caption" class="portal-model-price-caption-tag">{{ caption }}</span>
      <span v-if="adjustedLabel" class="sr-only">{{ adjustedLabel + ' ' }}</span>
      <strong class="portal-model-price-adjusted-value">{{ adjusted }}</strong><span v-if="unit" class="portal-model-price-unit">{{ ' ' + unit }}</span>
    </span>
    <span class="portal-model-price-base" :class="{ 'is-muted': adjusted !== undefined }">
      <span v-if="caption" class="portal-model-price-caption">{{ caption + ' ' }}</span>{{ base }}<span v-if="unit">{{ ' ' + unit }}</span>
    </span>
  </span>
</template>

<style scoped>
.portal-model-price {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  font-variant-numeric: tabular-nums;
}

.portal-model-price + .portal-model-price {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed rgb(var(--line) / 0.85);
}

.portal-model-price-adjusted {
  display: inline-flex;
  align-items: baseline;
  justify-content: flex-end;
  gap: 5px;
  flex-wrap: nowrap;
}

.portal-model-price-caption-tag {
  padding: 1px 5px;
  border-radius: 4px;
  background: rgb(var(--surface-2));
  border: 1px solid rgb(var(--line));
  color: rgb(var(--ink-2));
  font-size: 10.5px;
  font-weight: 600;
  line-height: 1.3;
}

.portal-model-price-adjusted-value {
  color: rgb(var(--brand-moss));
  font-family: var(--font-mono);
  font-size: 13.5px;
  font-weight: 700;
  line-height: 1.35;
  white-space: nowrap;
}

.portal-model-price-unit {
  font-size: 11px;
  color: rgb(var(--ink-3));
  white-space: nowrap;
}

.portal-model-price-base {
  color: rgb(var(--ink));
  font-family: var(--font-mono);
  white-space: nowrap;
  font-size: 13px;
  font-weight: 550;
  line-height: 1.4;
}

.portal-model-price-base.is-muted {
  color: rgb(var(--ink-3));
  font-size: 11px;
  font-weight: 450;
  text-decoration: line-through;
  text-decoration-color: rgb(var(--ink-3) / 0.55);
}

.portal-model-price-caption {
  color: rgb(var(--ink-3));
  font-family: var(--font-sans);
  font-size: 10.5px;
  text-decoration: none;
}

@media (max-width: 767px), (max-width: 1023px) and (max-height: 500px) and (pointer: coarse) {
  .portal-model-price {
    align-items: flex-start;
    min-width: 0;
    max-width: 100%;
    overflow-wrap: anywhere;
  }

  .portal-model-price-adjusted {
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .portal-model-price-base,
  .portal-model-price-adjusted-value {
    white-space: normal;
  }
}
</style>
