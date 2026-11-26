<template>
  <!-- Mobile Card View -->
  <div v-if="mobile" class="grid grid-cols-1 gap-3">
    <article
      v-for="apiKey in keys"
      :key="apiKey.id"
      class="atelier-surface space-y-3 p-4 transition-colors"
      :class="selectable && selectedIds.includes(apiKey.id) ? 'border-[var(--atelier-ink)] bg-[var(--atelier-selected-row)]' : ''"
      :data-key-id="apiKey.id"
      :data-selected="selectable && selectedIds.includes(apiKey.id)"
      data-test="api-key-card"
    >
      <header class="flex items-center gap-2.5">
        <label v-if="selectable" class="inline-flex items-center">
          <input
            type="checkbox"
            class="h-4 w-4 accent-[var(--atelier-ink)]"
            :checked="selectedIds.includes(apiKey.id)"
            :disabled="busyIds.has(apiKey.id)"
            :aria-label="text('选择密钥：', 'Select key: ') + apiKey.name"
            :data-test="'select-key-' + apiKey.id"
            @change="emit('select', apiKey.id, ($event.target as HTMLInputElement).checked)"
          />
        </label>
        <strong class="portal-api-card-name min-w-0 flex-1 truncate text-sm font-semibold text-[var(--atelier-ink)]" :title="apiKey.name">
          {{ apiKey.name }}
        </strong>
        <span
          class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium"
          :class="statusBadgeClass(apiKey.status)"
          :data-status="apiKey.status"
        >
          <UiStatusDot :tone="statusDotTone(apiKey.status)" />
          {{ statusName(apiKey.status) }}
        </span>
      </header>

      <div class="flex items-center justify-between gap-2 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-3 py-1.5">
        <code class="min-w-0 flex-1 truncate font-mono text-xs text-[var(--atelier-ink)]">{{ masked(apiKey.key) }}</code>
        <button
          type="button"
          class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-surface)] hover:text-[var(--atelier-ink)]"
          :aria-label="text('复制密钥：', 'Copy key: ') + apiKey.name"
          @click="emit('copy', apiKey)"
        >
          <Icon name="copy" size="xs" aria-hidden="true" />
        </button>
      </div>

      <button
        type="button"
        class="flex w-full flex-wrap items-center gap-2 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-3 py-2 text-left text-xs transition-colors hover:border-[var(--atelier-border-strong)] hover:bg-[var(--atelier-hover)] disabled:opacity-50"
        :disabled="busyIds.has(apiKey.id)"
        :title="text('点击修改分组', 'Click to change group')"
        :aria-label="groupLabel(apiKey)"
        data-test="edit-group"
        @click="emit('edit', apiKey)"
      >
        <GroupBadge
          v-if="groupFor(apiKey)"
          class="min-w-0 max-w-full flex-wrap"
          :truncate="false"
          :name="groupFor(apiKey)?.name || ''"
          :platform="groupFor(apiKey)?.platform"
          :subscription-type="groupFor(apiKey)?.subscription_type"
          :rate-multiplier="groupFor(apiKey)?.rate_multiplier"
          :user-rate-multiplier="groupRates[groupFor(apiKey)?.id ?? -1] ?? null"
          :peak-rate-enabled="groupFor(apiKey)?.peak_rate_enabled"
          :peak-start="groupFor(apiKey)?.peak_start"
          :peak-end="groupFor(apiKey)?.peak_end"
          :peak-rate-multiplier="groupFor(apiKey)?.peak_rate_multiplier"
        />
        <span v-else class="text-[var(--atelier-ink-muted)]">{{ text('未绑定分组', 'No group') }}</span>
        <span v-if="routingCount(apiKey)" class="portal-api-routing font-mono text-xs font-medium text-[var(--atelier-ink-secondary)]">
          {{ text('智能路由', 'Smart routing') }} · {{ routingCount(apiKey) }}
        </span>
      </button>
      <GroupChangeIndicator :group="groupFor(apiKey)" />

      <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
        <div>
          <dt class="text-[var(--atelier-ink-muted)]">{{ text('今日', 'Today') }}</dt>
          <dd class="mt-0.5 font-mono font-medium tabular-nums text-[var(--atelier-ink)]">
            {{ usageFor(apiKey.id) ? money(usageFor(apiKey.id)?.today_actual_cost ?? 0) : '—' }}
          </dd>
        </div>
        <div>
          <dt class="text-[var(--atelier-ink-muted)]">{{ text('累计', 'Total') }}</dt>
          <dd class="mt-0.5 font-mono font-medium tabular-nums text-[var(--atelier-ink)]">
            {{ usageFor(apiKey.id) ? money(usageFor(apiKey.id)?.total_actual_cost ?? 0) : '—' }}
          </dd>
        </div>
        <div>
          <dt class="text-[var(--atelier-ink-muted)]">{{ text('此 Key 当前请求数', 'Active requests for this key') }}</dt>
          <dd class="mt-0.5 font-mono tabular-nums text-[var(--atelier-ink)]">
            <span class="portal-api-concurrency">{{ concurrency(apiKey) }}</span>
            <small v-if="waiting(apiKey)" class="portal-api-waiting ml-1.5 text-[var(--atelier-ink-muted)]">{{ waiting(apiKey) }}</small>
          </dd>
        </div>
        <div>
          <dt class="text-[var(--atelier-ink-muted)]">{{ text('最近使用', 'Last used') }}</dt>
          <dd class="mt-0.5 font-mono tabular-nums text-[var(--atelier-ink-secondary)]">
            {{ apiKey.last_used_at ? date(apiKey.last_used_at) : text('尚未使用', 'Never') }}
          </dd>
        </div>
        <div>
          <dt class="text-[var(--atelier-ink-muted)]">{{ text('过期时间', 'Expires') }}</dt>
          <dd class="mt-0.5 font-mono tabular-nums" :class="apiKey.status === 'expired' ? 'text-[var(--atelier-err)] font-medium' : 'text-[var(--atelier-ink-secondary)]'">
            {{ apiKey.expires_at ? date(apiKey.expires_at) : text('永久有效', 'Never expires') }}
          </dd>
        </div>
        <div>
          <dt class="text-[var(--atelier-ink-muted)]">{{ text('创建时间', 'Created') }}</dt>
          <dd class="mt-0.5 font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ date(apiKey.created_at) }}</dd>
        </div>
      </dl>

      <PortalKeyUsageLimits :api-key="apiKey" />

      <div class="grid grid-cols-3 gap-2 border-t border-[var(--atelier-border-subtle)] pt-3">
        <button type="button" data-test="key-usage" class="atelier-btn atelier-btn-secondary h-9 px-2 text-xs" @click="emit('usage', apiKey)">
          {{ text('用量详情', 'Usage') }}
        </button>
        <button type="button" data-test="use-key" class="atelier-btn atelier-btn-secondary h-9 px-2 text-xs" :disabled="busyIds.has(apiKey.id)" @click="emit('use', apiKey)">
          <Icon name="terminal" size="xs" aria-hidden="true" />
          <span>{{ text('使用密钥', 'Use key') }}</span>
        </button>
        <button v-if="!hideCcs" type="button" data-test="import-ccs" class="atelier-btn atelier-btn-secondary h-9 px-2 text-xs" :disabled="busyIds.has(apiKey.id)" @click="emit('import-ccs', apiKey)">
          <Icon name="upload" size="xs" aria-hidden="true" />
          <span>{{ text('导入到 CCS', 'Import to CCS') }}</span>
        </button>
        <button type="button" class="atelier-btn atelier-btn-secondary h-9 px-2 text-xs" :disabled="busyIds.has(apiKey.id)" @click="emit('toggle', apiKey)">
          <Icon :name="apiKey.status === 'active' ? 'ban' : 'play'" size="xs" aria-hidden="true" />
          <span>{{ apiKey.status === 'active' ? text('禁用', 'Disable') : text('启用', 'Enable') }}</span>
        </button>
        <button type="button" data-test="edit-key" class="atelier-btn atelier-btn-secondary h-9 px-2 text-xs" :disabled="busyIds.has(apiKey.id)" @click="emit('edit', apiKey)">
          <Icon name="edit" size="xs" aria-hidden="true" />
          <span>{{ text('编辑', 'Edit') }}</span>
        </button>
        <button type="button" class="atelier-btn atelier-btn-ghost h-9 px-2 text-xs text-[var(--atelier-err)] hover:bg-[var(--atelier-err-bg)]" :disabled="busyIds.has(apiKey.id)" @click="emit('delete', apiKey)">
          <Icon name="trash" size="xs" aria-hidden="true" />
          <span>{{ text('删除', 'Delete') }}</span>
        </button>
      </div>
    </article>
  </div>

  <!-- Desktop Full-Width Borderless Editorial Table -->
  <div v-else class="overflow-x-auto border-y border-[var(--atelier-border)] bg-[var(--atelier-surface)]">
    <table class="atelier-table">
      <caption class="sr-only">{{ text('API 密钥', 'API keys') }}</caption>
      <thead>
        <tr>
          <th v-if="selectable" class="w-10">
            <input
              type="checkbox"
              class="h-4 w-4 accent-[var(--atelier-ink)]"
              :checked="allSelected"
              :indeterminate.prop="someSelected"
              :disabled="!keys.length || busy"
              :aria-label="text('当前页全选', 'Select this page')"
              data-test="table-select-all"
              @change="emit('select-all', ($event.target as HTMLInputElement).checked)"
            />
          </th>
          <th>{{ text('密钥名称', 'Key') }}</th>
          <th>{{ text('分组与路由', 'Group') }}</th>
          <th>{{ text('用量与配额', 'Usage & quota') }}</th>
          <th>{{ text('并发', 'Concurrency') }}</th>
          <th>{{ text('状态', 'Status') }}</th>
          <th>{{ text('最近使用 / 过期', 'Last used / Expiry') }}</th>
          <th class="text-right">{{ text('操作', 'Actions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="apiKey in keys"
          :key="apiKey.id"
          :data-key-id="apiKey.id"
          :data-selected="selectable && selectedIds.includes(apiKey.id) ? 'true' : 'false'"
        >
          <td v-if="selectable" class="w-10">
            <input
              type="checkbox"
              class="h-4 w-4 accent-[var(--atelier-ink)]"
              :checked="selectedIds.includes(apiKey.id)"
              :disabled="busyIds.has(apiKey.id)"
              :aria-label="text('选择密钥：', 'Select key: ') + apiKey.name"
              :data-test="'select-key-' + apiKey.id"
              @change="emit('select', apiKey.id, ($event.target as HTMLInputElement).checked)"
            />
          </td>

          <!-- Name + Masked Secret Quick-Copy -->
          <td class="min-w-[200px]">
            <div class="font-medium text-[var(--atelier-ink)]" :title="apiKey.name">
              {{ apiKey.name }}
            </div>
            <div class="mt-1 inline-flex max-w-[210px] items-center gap-1 rounded-md border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2 py-0.5">
              <code class="truncate font-mono text-[11.5px] text-[var(--atelier-ink-secondary)]">
                {{ masked(apiKey.key) }}
              </code>
              <button
                type="button"
                class="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded text-[var(--atelier-ink-muted)] transition-colors hover:bg-[var(--atelier-surface)] hover:text-[var(--atelier-ink)]"
                :aria-label="text('复制密钥：', 'Copy key: ') + apiKey.name"
                @click="emit('copy', apiKey)"
              >
                <Icon name="copy" size="xs" aria-hidden="true" />
              </button>
            </div>
          </td>

          <!-- Group with Full-Color BrandPlatformIcon -->
          <td class="min-w-[180px]">
            <button
              type="button"
              class="group inline-flex max-w-[260px] flex-wrap items-center gap-2 rounded-lg border border-transparent px-2 py-1 -mx-2 text-left transition-colors hover:border-[var(--atelier-border)] hover:bg-[var(--atelier-hover)] disabled:opacity-50"
              :disabled="busyIds.has(apiKey.id)"
              :title="text('点击修改分组', 'Click to change group')"
              :aria-label="groupLabel(apiKey)"
              data-test="edit-group"
              @click="emit('edit', apiKey)"
            >
              <template v-if="groupFor(apiKey)">
                <BrandPlatformIcon :platform="groupFor(apiKey)?.platform" size="sm" />
                <span class="truncate text-sm font-medium text-[var(--atelier-ink)]">
                  {{ groupFor(apiKey)?.name }}
                </span>
                <span class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-secondary)]">
                  {{ effectiveRateText(groupFor(apiKey)!) }}
                </span>
              </template>
              <span v-else class="text-xs text-[var(--atelier-ink-muted)]">{{ text('未绑定分组', 'No group') }}</span>
              <span
                v-if="routingCount(apiKey)"
                class="portal-api-routing w-full font-mono text-[11px] font-medium text-[var(--atelier-ink-secondary)]"
              >
                {{ text('智能路由', 'Smart routing') }} · {{ routingCount(apiKey) }}
              </span>
            </button>
            <GroupChangeIndicator :group="groupFor(apiKey)" />
          </td>

          <!-- Usage & 4px Quota Progress -->
          <td class="min-w-[180px]">
            <div v-if="usageFor(apiKey.id)" class="flex items-baseline gap-3 font-mono text-xs tabular-nums">
              <span>
                <span class="text-[var(--atelier-ink-muted)]">{{ text('今', '1d') }}</span>
                <strong class="ml-1 font-medium text-[var(--atelier-ink)]">{{ money(usageFor(apiKey.id)?.today_actual_cost ?? 0) }}</strong>
              </span>
              <span>
                <span class="text-[var(--atelier-ink-muted)]">{{ text('累', 'All') }}</span>
                <strong class="ml-1 font-medium text-[var(--atelier-ink-secondary)]">{{ money(usageFor(apiKey.id)?.total_actual_cost ?? 0) }}</strong>
              </span>
            </div>
            <span v-else class="font-mono text-xs text-[var(--atelier-ink-muted)]">—</span>
            <PortalKeyUsageLimits :api-key="apiKey" />
          </td>

          <!-- Concurrency & Queue -->
          <td class="whitespace-nowrap">
            <span
              class="portal-api-concurrency inline-flex items-center rounded-md border px-2 py-0.5 font-mono text-xs tabular-nums"
              :class="isConcurrencyFull(apiKey) ? 'border-[var(--atelier-warn)]/30 bg-[var(--atelier-warn-bg)] text-[var(--atelier-warn)]' : 'border-[var(--atelier-border)] bg-[var(--atelier-subtle)] text-[var(--atelier-ink-secondary)]'"
              :data-full="isConcurrencyFull(apiKey)"
            >
              {{ concurrency(apiKey) }}
            </span>
            <small v-if="waiting(apiKey)" class="portal-api-waiting mt-1 block font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
              {{ waiting(apiKey) }}
            </small>
          </td>

          <!-- Status -->
          <td class="whitespace-nowrap">
            <span
              class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium"
              :class="statusBadgeClass(apiKey.status)"
              :data-status="apiKey.status"
            >
              <UiStatusDot :tone="statusDotTone(apiKey.status)" />
              {{ statusName(apiKey.status) }}
            </span>
          </td>

          <!-- Last Used & Expiry -->
          <td class="whitespace-nowrap font-mono text-xs tabular-nums">
            <div class="text-[var(--atelier-ink-secondary)]">
              {{ apiKey.last_used_at ? date(apiKey.last_used_at) : text('尚未使用', 'Never used') }}
            </div>
            <div
              class="mt-0.5 text-[11px]"
              :class="apiKey.status === 'expired' ? 'font-medium text-[var(--atelier-err)]' : 'text-[var(--atelier-ink-muted)]'"
            >
              {{ apiKey.expires_at ? `${text('到期 ', 'Exp ')}${date(apiKey.expires_at)}` : text('永久有效', 'Never expires') }}
            </div>
          </td>

          <!-- Row Actions -->
          <td class="whitespace-nowrap text-right">
            <div class="inline-flex items-center justify-end gap-1">
              <button
                type="button"
                data-test="use-key"
                class="atelier-btn atelier-btn-secondary h-7 px-2.5 text-xs"
                :disabled="busyIds.has(apiKey.id)"
                @click="emit('use', apiKey)"
              >
                <Icon name="terminal" size="xs" aria-hidden="true" />
                <span>{{ text('接入', 'Connect') }}</span>
              </button>
              <button
                type="button"
                data-test="key-usage"
                class="atelier-btn atelier-btn-ghost h-7 px-2 text-xs"
                :title="text('用量详情', 'Usage details')"
                @click="emit('usage', apiKey)"
              >
                {{ text('用量', 'Usage') }}
              </button>
              <button
                v-if="!hideCcs"
                type="button"
                data-test="import-ccs"
                class="atelier-btn atelier-btn-ghost h-7 px-2 text-xs"
                :title="text('导入到 CCS', 'Import to CCS')"
                :disabled="busyIds.has(apiKey.id)"
                @click="emit('import-ccs', apiKey)"
              >
                CCS
              </button>
              <button
                type="button"
                class="atelier-btn atelier-btn-ghost h-7 w-7 p-0"
                :title="apiKey.status === 'active' ? text('禁用', 'Disable') : text('启用', 'Enable')"
                :aria-label="apiKey.status === 'active' ? text('禁用', 'Disable') : text('启用', 'Enable')"
                :disabled="busyIds.has(apiKey.id)"
                @click="emit('toggle', apiKey)"
              >
                <Icon :name="apiKey.status === 'active' ? 'ban' : 'play'" size="xs" aria-hidden="true" />
              </button>
              <button
                type="button"
                data-test="edit-key"
                class="atelier-btn atelier-btn-ghost h-7 w-7 p-0"
                :title="text('编辑', 'Edit')"
                :aria-label="text('编辑', 'Edit')"
                :disabled="busyIds.has(apiKey.id)"
                @click="emit('edit', apiKey)"
              >
                <Icon name="edit" size="xs" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="atelier-btn atelier-btn-ghost h-7 w-7 p-0 text-[var(--atelier-err)] hover:bg-[var(--atelier-err-bg)] hover:text-[var(--atelier-err)]"
                :title="text('删除', 'Delete')"
                :aria-label="text('删除', 'Delete')"
                :disabled="busyIds.has(apiKey.id)"
                @click="emit('delete', apiKey)"
              >
                <Icon name="trash" size="xs" aria-hidden="true" />
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
import { BrandPlatformIcon, UiStatusDot } from '@/ui'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupChangeIndicator from '@/components/common/GroupChangeIndicator.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatUsageMoney } from '@/utils/portalUsage'
import { maskApiKey } from '@/utils/maskApiKey'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import type { APIKeyQueueItem } from '@/api/keys'
import PortalKeyUsageLimits from './PortalKeyUsageLimits.vue'
import type { ApiKey, Group } from '@/types'

const props = withDefaults(
  defineProps<{
    queueCounts?: Record<number, APIKeyQueueItem>
    queueUnavailable?: boolean
    keys: ApiKey[]
    groups: Group[]
    groupRates: Record<number, number>
    usageStats: Record<string, BatchApiKeyUsageStats>
    busyIds: Set<number>
    selectable?: boolean
    selectedIds: number[]
    hideCcs?: boolean
  }>(),
  {
    queueCounts: () => ({}),
    queueUnavailable: false,
    selectable: false,
    hideCcs: false
  }
)

const emit = defineEmits<{
  select: [id: number, checked: boolean]
  'select-all': [checked: boolean]
  copy: [key: ApiKey]
  use: [key: ApiKey]
  usage: [key: ApiKey]
  'import-ccs': [key: ApiKey]
  edit: [key: ApiKey]
  toggle: [key: ApiKey]
  delete: [key: ApiKey]
}>()

const { locale } = useI18n()
const text = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en)
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
const busy = computed(() => props.busyIds.size > 0)
const selectedOnPage = computed(() => props.keys.filter((key) => props.selectedIds.includes(key.id)))
const allSelected = computed(() => props.keys.length > 0 && selectedOnPage.value.length === props.keys.length)
const someSelected = computed(() => selectedOnPage.value.length > 0 && !allSelected.value)

function isConcurrencyFull(key: ApiKey) {
  return (props.queueCounts[key.id]?.current_concurrency ?? 0) >= (key.concurrency_limit || Infinity)
}

function concurrency(key: ApiKey) {
  if (props.queueUnavailable) return text('暂不可用', 'Unavailable')
  const count = props.queueCounts[key.id]?.current_concurrency
  if (count !== undefined) return key.concurrency_limit ? `${count} / ${key.concurrency_limit}` : String(count)
  return key.current_concurrency_status === 'available'
    ? key.current_concurrency
    : key.current_concurrency_status === 'unsupported'
      ? text('未支持', 'Unsupported')
      : text('暂不可用', 'Unavailable')
}

function waiting(key: ApiKey) {
  if (props.queueUnavailable || props.queueCounts[key.id] === undefined) return ''
  return text('排队 ', 'Waiting ') + props.queueCounts[key.id].current_waiting
}

function usageFor(id: number) {
  return props.usageStats[String(id)]
}

function groupFor(apiKey: ApiKey) {
  return (apiKey.group_id == null ? undefined : props.groups.find((group) => group.id === apiKey.group_id)) || apiKey.group
}

function effectiveRateText(group: NonNullable<ReturnType<typeof groupFor>>) {
  const userRate = props.groupRates[group.id]
  const rateVal = userRate != null ? userRate : (group.rate_multiplier ?? 1)
  return `${rateVal}×`
}

function routingCount(apiKey: ApiKey) {
  return apiKey.routing_group_ids?.length ?? 0
}

function groupLabel(apiKey: ApiKey) {
  const groupName = groupFor(apiKey)?.name || text('未绑定分组', 'No group')
  return text('修改分组：', 'Change group: ') + apiKey.name + ' · ' + groupName
}

function masked(value: string) {
  return maskApiKey(value)
}

function statusName(status: ApiKey['status']) {
  return {
    active: text('活跃', 'Active'),
    inactive: text('停用', 'Inactive'),
    expired: text('过期', 'Expired'),
    quota_exhausted: text('额度用尽', 'Quota exhausted')
  }[status]
}

function statusDotTone(status: ApiKey['status']): 'ok' | 'neutral' | 'err' {
  if (status === 'active') return 'ok'
  if (status === 'inactive') return 'neutral'
  return 'err'
}

function statusBadgeClass(status: ApiKey['status']): string {
  if (status === 'active') return 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]'
  if (status === 'inactive') return 'bg-[var(--atelier-subtle)] text-[var(--atelier-ink-muted)]'
  return 'bg-[var(--atelier-err-bg)] text-[var(--atelier-err)]'
}

function date(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '—'
  return parsed.toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

function money(value: number) {
  return formatUsageMoney(value, 4)
}
</script>
