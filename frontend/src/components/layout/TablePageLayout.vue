<template>
  <div
    class="table-page-layout"
    :class="{
      'mobile-mode': isMobile,
      'page-scroll-mode': pageScroll
    }"
  >
    <!-- 固定区域：操作按钮 -->
    <div v-if="$slots.actions" class="layout-section-fixed">
      <slot name="actions" />
    </div>

    <!-- 固定区域：搜索和过滤器 -->
    <div v-if="$slots.filters" class="layout-section-fixed">
      <slot name="filters" />
    </div>

    <!-- 滚动区域：表格 -->
    <div class="layout-section-scrollable">
      <div class="card table-scroll-container">
        <slot name="table" />
      </div>
    </div>

    <!-- 固定区域：分页器 -->
    <div v-if="$slots.pagination" class="layout-section-fixed">
      <slot name="pagination" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

withDefaults(defineProps<{
  pageScroll?: boolean
}>(), {
  pageScroll: false
})

const isMobile = ref(false)

const checkMobile = () => {
  isMobile.value = window.innerWidth < 1024
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
/* 桌面端：Flexbox 布局 */
.table-page-layout {
  @apply flex flex-col gap-6;
  height: calc(100vh - 64px - 4rem); /* 减去 header + lg:p-8 的上下padding */
}

.table-page-layout.page-scroll-mode {
  height: auto;
  min-height: calc(100vh - 64px - 4rem);
}

.layout-section-fixed {
  @apply flex-shrink-0;
}

.layout-section-scrollable {
  @apply flex-1 min-h-0 flex flex-col;
}

/* 表格滚动容器 - 增强版表体滚动方案 */
.table-scroll-container {
  @apply flex flex-col overflow-hidden h-full;
  border-radius: 0.875rem;
  border: 1px solid rgba(226, 232, 240, 0.95);
  background: rgba(255, 255, 255, 0.88);
  box-shadow:
    0 18px 48px rgba(15, 23, 42, 0.055),
    inset 0 1px 0 rgba(255, 255, 255, 0.9);
}

.dark .table-scroll-container {
  border-color: rgba(0, 255, 159, 0.22);
  background: linear-gradient(180deg, rgba(2, 15, 13, 0.88), rgba(2, 8, 7, 0.86));
  box-shadow:
    0 0 0 1px rgba(0, 255, 159, 0.06),
    0 24px 70px rgba(0, 0, 0, 0.38),
    inset 0 1px 0 rgba(128, 255, 196, 0.08);
}

.table-page-layout.page-scroll-mode .layout-section-scrollable {
  @apply flex-none min-h-fit;
}

.table-page-layout.page-scroll-mode .table-scroll-container {
  @apply h-auto;
}

.table-scroll-container :deep(.table-wrapper) {
  @apply flex-1 overflow-x-auto overflow-y-auto;
  /* 确保横向滚动条显示在最底部 */
  scrollbar-gutter: stable;
}

.table-page-layout.page-scroll-mode .table-scroll-container :deep(.table-wrapper) {
  @apply flex-none overflow-x-auto overflow-y-visible;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content; /* 关键：确保表格宽度根据内容撑开，从而触发横向滚动 */
  display: table; /* 使用标准 table 布局以支持 sticky 列 */
}

.table-scroll-container :deep(thead) {
  background: rgba(248, 250, 252, 0.92);
  backdrop-filter: blur(8px);
}

.dark .table-scroll-container :deep(thead) {
  background: rgba(2, 20, 16, 0.92);
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply px-5 py-4 text-left text-sm font-medium;
  color: #475569;
  border-bottom: 1px solid rgba(226, 232, 240, 0.82);
}

.dark .table-scroll-container :deep(th) {
  color: #42f6a3;
  border-bottom-color: rgba(0, 255, 159, 0.14);
}

.table-scroll-container :deep(td) {
  @apply px-5 py-4 text-sm;
  color: #334155;
  border-bottom: 1px solid rgba(226, 232, 240, 0.74);
}

.dark .table-scroll-container :deep(td) {
  color: #d8eee6;
  border-bottom-color: rgba(0, 255, 159, 0.1);
}

/* 移动端：恢复正常滚动 */
.table-page-layout.mobile-mode .table-scroll-container {
  @apply h-auto overflow-visible border-none shadow-none bg-transparent;
}

.table-page-layout.mobile-mode .layout-section-scrollable {
  @apply flex-none min-h-fit;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(.table-wrapper) {
  @apply overflow-visible;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(table) {
  @apply flex-none;
  display: table;
  min-width: 100%;
}
</style>
