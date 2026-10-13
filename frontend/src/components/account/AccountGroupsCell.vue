<template>
  <div v-if="groups && groups.length > 0" class="relative max-w-56">
    <!-- 限制列宽但不裁切行高，完整显示分组名称和 +N 按钮 -->
    <div class="flex flex-wrap gap-1">
      <GroupBadge
        v-for="group in displayGroups"
        :key="group.id"
        :name="group.name"
        :platform="group.platform"
        :subscription-type="group.subscription_type"
        :rate-multiplier="group.rate_multiplier"
        :show-rate="false"
        :truncate="false"
        class="min-w-0 max-w-full"
      />
      <!-- 更多数量徽章 -->
      <HelpTooltip
        v-if="hiddenCount > 0"
        trigger="click"
        width-class="w-96"
        class="!ml-0"
        :label="t('admin.accounts.groupCountTotal', { count: groups.length })"
      >
        <template #trigger="{ triggerAttrs }">
          <button
            v-bind="triggerAttrs"
            type="button"
            class="inline-flex items-center gap-0.5 rounded-md px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500 transition-colors cursor-pointer whitespace-nowrap max-md:min-h-11 max-md:min-w-11 max-md:justify-center"
          >+{{ hiddenCount }}</button>
        </template>
        <template #default="{ open }">
          <div v-if="open">
            <div class="mb-2 text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.groupCountTotal', { count: groups.length }) }}
            </div>
            <div class="flex flex-wrap gap-1.5">
              <GroupBadge
                v-for="group in groups"
                :key="group.id"
                :name="group.name"
                :platform="group.platform"
                :subscription-type="group.subscription_type"
                :rate-multiplier="group.rate_multiplier"
                :show-rate="false"
                :truncate="false"
                class="min-w-0 max-w-full"
              />
            </div>
          </div>
        </template>
      </HelpTooltip>
    </div>
  </div>
  <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from '@/components/common/GroupBadge.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import type { Group } from '@/types'

interface Props {
  groups: Group[] | null | undefined
  maxDisplay?: number
}

const props = withDefaults(defineProps<Props>(), {
  maxDisplay: 4
})

const { t } = useI18n()

// 显示的分组（最多显示 maxDisplay 个）
const displayGroups = computed(() => {
  if (!props.groups) return []
  if (props.groups.length <= props.maxDisplay) {
    return props.groups
  }
  // 留一个位置给 +N 按钮
  return props.groups.slice(0, props.maxDisplay - 1)
})

// 隐藏的数量
const hiddenCount = computed(() => {
  if (!props.groups) return 0
  if (props.groups.length <= props.maxDisplay) return 0
  return props.groups.length - (props.maxDisplay - 1)
})

</script>
