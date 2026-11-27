<template>
  <div class="space-y-6">
    <!-- Minimal Page Header -->
    <header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-center gap-2.5">
        <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
          {{ text('API 密钥', 'API keys') }}
        </h1>
        <span class="portal-api-count inline-flex h-6 min-w-[26px] items-center justify-center rounded-full border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-2 font-mono text-xs font-medium tabular-nums text-[var(--atelier-ink-secondary)]">
          {{ keysKnown ? keyTotal : '—' }}
        </span>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <UiButton
          variant="ghost"
          size="sm"
          :disabled="keysLoading || groupsLoading || retryRemaining > 0"
          :aria-label="text('刷新', 'Refresh')"
          data-test="refresh-keys"
          @click="refreshPage"
        >
          <Icon name="refresh" size="xs" aria-hidden="true" />
          {{ text('刷新', 'Refresh') }}
        </UiButton>
        <UiButton
          variant="secondary"
          size="sm"
          :aria-pressed="bulkMode"
          data-test="bulk-mode-toggle"
          @click="toggleBulkMode"
        >
          {{ bulkMode ? text('退出批量管理', 'Exit bulk') : text('批量管理', 'Bulk edit') }}
        </UiButton>
        <UiButton
          variant="primary"
          size="sm"
          data-tour="keys-create-btn"
          :disabled="creationDisabled"
          aria-describedby="key-creation-state"
          @click="openCreate()"
        >
          <Icon name="plus" size="xs" aria-hidden="true" />
          {{ text('创建密钥', 'Create key') }}
        </UiButton>
      </div>
    </header>

    <!-- Compact Quickstart + Endpoint Bar -->
    <div id="api-quick-start" class="atelier-surface flex flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="portal-api-endpoint flex min-w-0 flex-wrap items-center gap-2 text-xs">
        <span class="rounded-md bg-[var(--atelier-subtle)] px-2 py-0.5 font-medium text-[var(--atelier-ink-secondary)]">
          {{ text('OpenAI 兼容 Base URL', 'OpenAI Base URL') }}
        </span>
        <code class="truncate font-mono text-[13px] text-[var(--atelier-ink)]" :title="apiBaseUrl">
          {{ apiBaseUrl || '—' }}
        </code>
        <button
          type="button"
          class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)] disabled:opacity-40"
          :disabled="!apiBaseUrl"
          :aria-label="text('复制 API 端点', 'Copy API endpoint')"
          data-test="copy-endpoint"
          @click="copyEndpoint"
        >
          <Icon name="copy" size="xs" aria-hidden="true" />
        </button>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <UiButton variant="secondary" size="sm" @click="quickstartOpen = true">
          <Icon name="bolt" size="xs" aria-hidden="true" />
          {{ text('一键接入配置', 'Quickstart hub') }}
        </UiButton>
        <RouterLink to="/playground" class="atelier-btn atelier-btn-ghost h-8 px-3 text-xs">
          <Icon name="play" size="xs" aria-hidden="true" />
          {{ text('在线调试', 'Playground') }}
        </RouterLink>
        <UiButton variant="ghost" size="sm" @click="docsOpen = true">
          <Icon name="book" size="xs" aria-hidden="true" />
          {{ text('接入指南', 'Guide') }}
        </UiButton>
      </div>
    </div>

    <!-- Group Availability State -->
    <p id="key-creation-state" v-if="groupsLoading" class="text-xs text-[var(--atelier-ink-muted)]" role="status">
      {{ text('正在读取可绑定分组…', 'Loading groups available for your keys…') }}
    </p>
    <div
      id="key-creation-state"
      v-else-if="groupsError"
      class="flex items-center justify-between gap-2 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-2.5 text-xs text-[var(--atelier-err)]"
      role="alert"
    >
      <span>{{ groupsError }}</span>
      <UiButton variant="ghost" size="sm" :disabled="retryRemaining > 0" @click="loadGroups">
        {{ retryRemaining > 0 ? `${retryRemaining}s` : text('重试分组加载', 'Retry groups') }}
      </UiButton>
    </div>
    <div
      id="key-creation-state"
      v-else-if="!groups.length"
      class="flex items-center justify-between gap-2 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-4 py-2.5 text-xs text-[var(--atelier-ink-secondary)]"
    >
      <span>{{ text('当前账户没有可绑定分组，暂时无法创建 API Key。', 'Your account has no available group for creating an API key.') }}</span>
      <UiButton variant="ghost" size="sm" @click="loadGroups">{{ text('刷新分组', 'Refresh groups') }}</UiButton>
    </div>

    <!-- Main Key List Section -->
    <section id="portal-api-keys" class="space-y-4" aria-labelledby="portal-api-keys-title">
      <h2 id="portal-api-keys-title" class="sr-only">{{ text('密钥列表', 'Key list') }}</h2>

      <!-- Newly Created Key Notice Banner -->
      <div
        v-if="lastCreated && !keyFormOpen"
        class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--atelier-ok)]/25 bg-[var(--atelier-ok-bg)] px-4 py-2.5 text-xs text-[var(--atelier-ink)]"
        role="status"
      >
        <div class="flex items-center gap-2">
          <UiStatusDot tone="ok" />
          <span>{{ text('密钥已创建：', 'Key created: ') }}<strong class="font-semibold">{{ lastCreated.name }}</strong></span>
        </div>
        <div class="flex items-center gap-2">
          <UiButton variant="secondary" size="sm" @click="copyKey(lastCreated)">
            <Icon name="copy" size="xs" aria-hidden="true" />
            {{ text('复制密钥', 'Copy key') }}
          </UiButton>
          <button
            type="button"
            class="inline-flex h-7 w-7 items-center justify-center rounded-md text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
            :aria-label="text('关闭创建提示', 'Dismiss creation message')"
            @click="lastCreated = null"
          >
            ✕
          </button>
        </div>
      </div>

      <!-- Filters Bar -->
      <form
        v-if="keysKnown && (keyTotal > 0 || hasKeyFilters)"
        class="flex flex-wrap items-center gap-2.5"
        @submit.prevent="applyKeyFilters"
      >
        <div class="relative min-w-[220px] flex-1 sm:max-w-xs">
          <Icon name="search" size="xs" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
          <input
            v-model="keySearch"
            type="search"
            class="atelier-input !min-h-[36px] !pl-8 !pr-16 text-xs"
            :aria-label="text('搜索 API Key', 'Search API keys')"
            :placeholder="text('搜索名称或密钥', 'Search name or key')"
          />
          <button
            type="submit"
            class="absolute right-1.5 top-1/2 -translate-y-1/2 rounded-md px-2 py-1 text-xs font-medium text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          >
            {{ text('搜索', 'Search') }}
          </button>
        </div>
        <div class="w-36">
          <UiSelect
            :model-value="keyStatusFilter"
            :options="statusFilterOptions"
            size="sm"
            :aria-label="text('密钥状态', 'Key status')"
            @update:model-value="(val) => { keyStatusFilter = String(val ?? ''); applyKeyFilters() }"
          />
        </div>
        <div v-if="groups.length > 1" class="w-48">
          <UiSelect
            :model-value="keyGroupFilter"
            :options="groupFilterOptions"
            size="sm"
            :aria-label="text('密钥分组', 'Key group')"
            @update:model-value="(val) => { keyGroupFilter = String(val ?? ''); applyKeyFilters() }"
          />
        </div>
        <UiButton v-if="hasKeyFilters" variant="ghost" size="sm" @click="resetKeyFilters">
          {{ text('清除筛选', 'Reset') }}
        </UiButton>
      </form>

      <!-- Action / Load Errors -->
      <p v-if="actionError" class="rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-2.5 text-xs text-[var(--atelier-err)]" role="alert">
        {{ actionError }}
      </p>
      <div v-if="keysLoading && !keys.length" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]" role="status">
        {{ text('正在加载密钥…', 'Loading API keys…') }}
      </div>
      <div
        v-if="keysError"
        class="flex items-center justify-between gap-2 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-2.5 text-xs text-[var(--atelier-err)]"
        role="alert"
      >
        <span>{{ keysError }}</span>
        <UiButton variant="ghost" size="sm" :disabled="retryRemaining > 0" @click="loadKeys">
          {{ retryRemaining > 0 ? `${retryRemaining}s` : text('重试', 'Retry') }}
        </UiButton>
      </div>

      <!-- Empty State -->
      <div
        v-if="!keysLoading && !keysError && !keys.length"
        class="atelier-surface relative overflow-hidden px-6 py-12 text-center"
      >
        <DotMatrixBackdrop mask="edges" />
        <div class="relative z-10 mx-auto max-w-md space-y-3">
          <p class="text-sm text-[var(--atelier-ink-secondary)]">
            {{ hasKeyFilters ? text('没有符合筛选条件的密钥。', 'No keys match these filters.') : text('还没有 API Key。创建一个密钥，开始接入你的应用。', 'No API keys yet. Create one to connect your application.') }}
          </p>
          <div>
            <UiButton v-if="hasKeyFilters" variant="secondary" size="sm" @click="resetKeyFilters">
              {{ text('清除筛选', 'Clear filters') }}
            </UiButton>
            <UiButton v-else variant="primary" size="sm" :disabled="creationDisabled" aria-describedby="key-creation-state" @click="openCreate()">
              <Icon name="plus" size="xs" aria-hidden="true" />
              {{ text('创建密钥', 'Create key') }}
            </UiButton>
          </div>
        </div>
      </div>

      <p v-if="keysError && keys.length" class="text-xs text-[var(--atelier-ink-muted)]">
        {{ text('以下为上次成功加载的数据。', 'Showing the last successfully loaded data.') }}
      </p>
      <PortalDataState
        v-if="usageError"
        :error="usageError"
        :retry-label="text('重试用量', 'Retry usage') + (retryRemaining > 0 ? ` (${retryRemaining}s)` : '')"
        :retry-disabled="usageLoading || retryRemaining > 0"
        @retry="retryUsage"
      />
      <p v-if="usageLoading" role="status" class="text-xs text-[var(--atelier-ink-muted)]">
        {{ text('正在刷新密钥用量…', 'Refreshing key usage…') }}
      </p>

      <!-- Borderless Full-Width Editorial Table -->
      <PortalApiKeyTable
        v-if="keys.length"
        :queue-counts="queueCounts"
        :queue-unavailable="queueUnavailable"
        :keys="keys"
        :groups="groups"
        :group-rates="groupRates"
        :usage-stats="usageStats"
        :busy-ids="busyKeyIds"
        :selectable="bulkMode"
        :selected-ids="selectedKeyIds"
        :hide-ccs="hideCcsImport"
        @select="selectKey"
        @select-all="selectPageKeys"
        @copy="copyKey"
        @use="openUseKey"
        @usage="usageKey = $event"
        @import-ccs="importToCcs"
        @edit="openEdit"
        @toggle="toggleKey"
        @delete="confirmDelete"
      />

      <!-- Floating Bulk Action Bar (cleanly appears when bulkMode or selected) -->
      <div
        v-if="bulkMode"
        class="sticky bottom-4 z-20 mx-auto flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] px-4 py-2.5 shadow-[var(--atelier-shadow-md)]"
      >
        <div class="flex items-center gap-3 text-xs">
          <label class="inline-flex cursor-pointer items-center gap-2 font-medium text-[var(--atelier-ink)]">
            <input
              type="checkbox"
              class="h-4 w-4 accent-[var(--atelier-ink)]"
              :checked="allPageKeysSelected"
              :indeterminate="somePageKeysSelected"
              :disabled="keysLoading || !!keysError || !keys.length || busyKeyIds.size > 0"
              data-test="bulk-select-all"
              @change="selectPageKeys(($event.target as HTMLInputElement).checked)"
            />
            <span>{{ text('当前页全选', 'Select page') }}</span>
          </label>
          <span class="font-mono tabular-nums text-[var(--atelier-ink-secondary)]" aria-live="polite">
            {{ text('已选择 ', 'Selected: ') }}{{ selectedKeys.length }}{{ text(' 个', ' keys') }}
          </span>
        </div>
        <div class="flex items-center gap-2">
          <UiButton variant="ghost" size="sm" :disabled="!selectedKeys.length" @click="clearSelection">
            {{ text('清空选择', 'Clear') }}
          </UiButton>
          <UiButton
            variant="primary"
            size="sm"
            :disabled="!selectedKeys.length || keysLoading || !!keysError || busyKeyIds.size > 0"
            data-test="bulk-edit-keys"
            @click="bulkFormOpen = true"
          >
            {{ text('批量编辑', 'Edit selected') }}
          </UiButton>
        </div>
      </div>

      <!-- Pagination -->
      <nav
        v-if="keyTotal > 0 && !keysError"
        class="flex flex-wrap items-center justify-between gap-3 pt-1 text-xs text-[var(--atelier-ink-secondary)]"
        :aria-label="text('密钥分页', 'API key pages')"
      >
        <div class="flex flex-wrap items-center gap-3 font-mono tabular-nums">
          <span>
            {{ text('显示 ', 'Showing ') }}{{ rangeStart }}{{ text(' 至 ', '–') }}{{ rangeEnd }}{{ text(' 共 ', ' of ') }}{{ keyTotal }}{{ text(' 条', '') }}
          </span>
          <div class="inline-flex items-center gap-2">
            <span class="font-sans text-[var(--atelier-ink-muted)]">{{ text('每页', 'Per page') }}</span>
            <div class="w-20">
              <UiSelect
                :model-value="keyPageSize"
                :options="pageSizeOptions"
                size="sm"
                :disabled="keysLoading"
                :aria-label="text('每页密钥数量', 'Keys per page')"
                @update:model-value="(val) => { keyPageSize = Number(val || 20); changePageSize() }"
              />
            </div>
          </div>
        </div>
        <div class="flex items-center gap-1">
          <button
            type="button"
            class="atelier-btn atelier-btn-secondary h-8 w-8 p-0 font-mono text-xs"
            :disabled="keysLoading || keyPage <= 1"
            :aria-label="text('上一页', 'Previous')"
            @click="goToPage(keyPage - 1)"
          >‹</button>
          <template v-for="(page, index) in pageList" :key="page">
            <span v-if="pageSeparated(index)" class="px-1 text-[var(--atelier-ink-muted)]">…</span>
            <button
              type="button"
              class="atelier-btn h-8 min-w-[32px] px-2 font-mono text-xs tabular-nums"
              :class="page === keyPage ? 'atelier-btn-primary' : 'atelier-btn-secondary'"
              :disabled="keysLoading"
              :aria-current="page === keyPage ? 'page' : undefined"
              @click="goToPage(page)"
            >{{ page }}</button>
          </template>
          <button
            type="button"
            class="atelier-btn atelier-btn-secondary h-8 w-8 p-0 font-mono text-xs"
            :disabled="keysLoading || keyPage >= keyPages"
            :aria-label="text('下一页', 'Next')"
            @click="goToPage(keyPage + 1)"
          >›</button>
        </div>
      </nav>
    </section>

    <!-- Bottom Resource Links -->
    <section class="flex flex-wrap items-center gap-2 border-t border-[var(--atelier-border)] pt-5" :aria-label="text('接入资源', 'Connection resources')">
      <RouterLink
        v-if="plazaEnabled"
        to="/model-plaza?embedded=1"
        class="atelier-btn atelier-btn-secondary h-8 px-3 text-xs"
      >
        <Icon name="grid" size="xs" aria-hidden="true" />
        {{ text('模型与价格', 'Models & pricing') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
      <UiButton variant="secondary" size="sm" :aria-expanded="examplesOpen" @click="examplesOpen = !examplesOpen">
        <Icon name="terminal" size="xs" aria-hidden="true" />
        {{ text('调用示例', 'Request examples') }}
        <Icon name="chevronDown" size="xs" aria-hidden="true" />
      </UiButton>
      <a
        v-if="docUrl"
        :href="docUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="atelier-btn atelier-btn-secondary h-8 px-3 text-xs"
      >
        <Icon name="book" size="xs" aria-hidden="true" />
        {{ text('接入文档', 'Documentation') }}
        <Icon name="externalLink" size="xs" aria-hidden="true" />
      </a>
      <UiButton v-else variant="secondary" size="sm" @click="docsOpen = true">
        <Icon name="book" size="xs" aria-hidden="true" />
        {{ text('接入指南', 'Quick start') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </UiButton>
    </section>
    <PortalApiExamples v-if="examplesOpen" v-model:model="exampleModel" />

    <!-- Slide-over Create / Edit Key Drawer -->
    <PortalKeyForm
      :open="keyFormOpen"
      :editing="editingKey"
      :groups="groups"
      :group-rates="groupRates"
      :selected-group="formGroupId"
      :base-url="apiRoot || apiBaseUrl"
      @close="closeKeyForm"
      @saved="onKeySaved"
    />

    <PortalKeyUsageDialog :api-key="usageKey" @close="usageKey = null" />
    <UseKeyModal
      :show="!!useKeyTarget"
      :api-key="useKeyTarget?.key || ''"
      :base-url="apiRoot"
      :platform="useKeyTarget?.group?.platform || null"
      :claude-code-only="useKeyTarget?.group?.claude_code_only || false"
      :allow-messages-dispatch="useKeyTarget?.group?.allow_messages_dispatch || false"
      @close="useKeyTarget = null"
    />

    <PortalDialog :open="ccsClientOpen" :title="t('keys.ccsClientSelect.title')" @close="closeCcsClient">
      <p class="text-sm text-[var(--atelier-ink-secondary)]">{{ t('keys.ccsClientSelect.description') }}</p>
      <div class="portal-api-ccs-choices mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <button
          type="button"
          class="flex flex-col items-center justify-center gap-1.5 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] p-4 text-center transition-colors hover:border-[var(--atelier-ink)] hover:bg-[var(--atelier-hover)]"
          @click="chooseCcsClient('claude')"
        >
          <Icon name="terminal" size="md" aria-hidden="true" />
          <strong class="text-sm font-semibold text-[var(--atelier-ink)]">{{ t('keys.ccsClientSelect.claudeCode') }}</strong>
          <span class="text-xs text-[var(--atelier-ink-muted)]">{{ t('keys.ccsClientSelect.claudeCodeDesc') }}</span>
        </button>
        <button
          type="button"
          class="flex flex-col items-center justify-center gap-1.5 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] p-4 text-center transition-colors hover:border-[var(--atelier-ink)] hover:bg-[var(--atelier-hover)]"
          @click="chooseCcsClient('gemini')"
        >
          <Icon name="sparkles" size="md" aria-hidden="true" />
          <strong class="text-sm font-semibold text-[var(--atelier-ink)]">{{ t('keys.ccsClientSelect.geminiCli') }}</strong>
          <span class="text-xs text-[var(--atelier-ink-muted)]">{{ t('keys.ccsClientSelect.geminiCliDesc') }}</span>
        </button>
      </div>
    </PortalDialog>

    <PortalApiBulkForm
      :open="bulkFormOpen"
      :selected-keys="selectedKeys"
      :groups="groups"
      :groups-loading="groupsLoading"
      :groups-error="groupsError"
      @close="bulkFormOpen = false"
      @updated="onBulkUpdated"
    />

    <PortalDialog :open="!!deleteTarget" :title="text('删除 API Key', 'Delete API key')" @close="closeDelete">
      <p class="text-sm text-[var(--atelier-ink-secondary)]">
        {{ text('删除后，使用此密钥的应用将无法继续请求。', 'Applications using this key will no longer be able to send requests.') }}
      </p>
      <p class="my-4 rounded-lg bg-[var(--atelier-subtle)] px-3 py-2 font-mono text-sm font-medium text-[var(--atelier-ink)]">
        {{ deleteTarget?.name }}
      </p>
      <p v-if="deleteError" class="text-xs text-[var(--atelier-err)]" role="alert">{{ deleteError }}</p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UiButton variant="secondary" size="sm" :disabled="deleting" @click="closeDelete">
            {{ text('取消', 'Cancel') }}
          </UiButton>
          <UiButton variant="danger" size="sm" :loading="deleting" @click="deleteKey">
            {{ deleting ? text('删除中…', 'Deleting…') : text('确认删除', 'Delete key') }}
          </UiButton>
        </div>
      </template>
    </PortalDialog>

    <PortalQuickstartDrawer
      :open="quickstartOpen"
      :keys="keys"
      :base-url="apiRoot || apiBaseUrl"
      :initial-key-id="lastCreated?.id || null"
      @close="quickstartOpen = false"
    />

    <PortalDialog :open="docsOpen" :title="text('API 接入指南', 'API quick start')" @close="docsOpen = false">
      <ol class="list-decimal space-y-2.5 pl-5 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
        <li>{{ text('点击创建 API Key，在抽屉中选择分组并查看分组倍率。完整模型和价格可在模型与价格页面查看。', 'Create an API key to choose a group and review its multiplier. Open Models & pricing for the full catalog.') }}</li>
        <li>{{ text('创建 API Key，并在需要时设置额度、有效期、IP 限制或路由分组。', 'Create an API key and configure quota, expiry, IP restrictions or routing groups as needed.') }}</li>
        <li>{{ text('使用页面顶部的接口地址和调用示例，以所选分组支持的模型发送请求。', 'Use the base URL and examples to request a model supported by your group.') }}</li>
        <li v-if="!authStore.isSimpleMode">
          <RouterLink to="/usage" class="font-medium text-[var(--atelier-ink)] underline underline-offset-2">
            {{ text('在用量信息中查看请求与费用。', 'Review requests and costs in Usage.') }}
          </RouterLink>
          {{ ' ' }}
          <RouterLink to="/purchase" class="font-medium text-[var(--atelier-ink)] underline underline-offset-2">
            {{ text('需要补充额度时前往充值。', 'Recharge when you need more balance.') }}
          </RouterLink>
        </li>
      </ol>
      <template #footer>
        <div class="flex justify-end">
          <UiButton variant="primary" size="sm" @click="docsOpen = false; showExamples()">
            {{ text('查看示例', 'View examples') }}
          </UiButton>
        </div>
      </template>
    </PortalDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { UiButton, UiSelect, UiStatusDot, DotMatrixBackdrop } from '@/ui'
import Icon from '@/components/icons/Icon.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'
import PortalApiKeyTable from '@/components/portal/PortalApiKeyTable.vue'
import PortalApiExamples from '@/components/portal/PortalApiExamples.vue'
import PortalApiBulkForm from '@/components/portal/PortalApiBulkForm.vue'
import PortalDialog from '@/components/portal/PortalDialog.vue'
import PortalKeyForm from '@/components/portal/PortalKeyForm.vue'
import PortalQuickstartDrawer from '@/components/portal/PortalQuickstartDrawer.vue'
import { keysAPI } from '@/api/keys'
import { useApiKeyQueue } from '@/composables/useApiKeyQueue'
import { usageAPI, type BatchApiKeyUsageStats } from '@/api/usage'
import { userGroupsAPI } from '@/api/groups'
import { CC_SWITCH_USAGE_SCRIPT, buildCcSwitchImportDeeplink, type CcSwitchClientType } from '@/utils/ccswitchImport'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import PortalDataState from '@/components/portal/PortalDataState.vue'
import PortalKeyUsageDialog from '@/components/portal/PortalKeyUsageDialog.vue'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { useRetryCooldown } from '@/composables/useRetryCooldown'
import { userFacingError } from '@/utils/userFacingError'
import { sanitizeUrl } from '@/utils/url'
import type { ApiKey, Group } from '@/types'

const { locale, t } = useI18n()
const text = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en)
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const plazaEnabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.modelPlaza))

const groups = ref<Group[]>([])
const statusFilterOptions = computed(() => [
  { value: '', label: text('全部状态', 'All statuses') },
  { value: 'active', label: text('已启用', 'Active') },
  { value: 'inactive', label: text('已停用', 'Inactive') },
  { value: 'expired', label: text('已过期', 'Expired') },
  { value: 'quota_exhausted', label: text('额度用尽', 'Quota exhausted') }
])
const groupFilterOptions = computed(() => [
  { value: '', label: text('全部分组', 'All groups') },
  { value: '0', label: text('未绑定分组', 'No group') },
  ...groups.value.map((group) => ({ value: String(group.id), label: group.name }))
])
const pageSizeOptions = [
  { value: 10, label: '10' },
  { value: 20, label: '20' },
  { value: 50, label: '50' }
]
const { remaining: retryRemaining, start: startRetryCooldown } = useRetryCooldown()
const groupsLoading = ref(true)
const groupsError = ref('')
const groupRates = ref<Record<number, number>>({})
const exampleModel = ref('')
const examplesOpen = ref(false)
const creationDisabled = computed(() => groupsLoading.value || !!groupsError.value || !groups.value.length)

const keys = ref<ApiKey[]>([])
const { counts: queueCounts, unavailable: queueUnavailable } = useApiKeyQueue(keys)
const keysLoading = ref(true)
const keysError = ref('')
const keySearch = ref('')
const appliedKeySearch = ref('')
const keyStatusFilter = ref('')
const keyGroupFilter = ref('')
const keyPage = ref(1)
const keyPageSize = ref(20)
const keyTotal = ref(0)
const keysKnown = ref(false)
const keyPages = computed(() => Math.max(1, Math.ceil(keyTotal.value / keyPageSize.value)))
const rangeStart = computed(() => (keyTotal.value === 0 ? 0 : (keyPage.value - 1) * keyPageSize.value + 1))
const rangeEnd = computed(() => Math.min(keyPage.value * keyPageSize.value, keyTotal.value))
const pageList = computed(() => {
  const pages = new Set([1, keyPages.value, keyPage.value - 1, keyPage.value, keyPage.value + 1].filter((page) => page >= 1 && page <= keyPages.value))
  return [...pages].sort((left, right) => left - right)
})
function pageSeparated(index: number) {
  const previous = pageList.value[index - 1]
  const current = pageList.value[index]
  return previous != null && current != null && current - previous > 1
}
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const usageKey = ref<ApiKey | null>(null)
const usageLoading = ref(false)
const usageError = ref('')
const { apiRoot, openAIBase: apiBaseUrl } = usePatrickApiBase()
const hideCcsImport = computed(() => !!appStore.cachedPublicSettings?.hide_ccs_import_button)
const useKeyTarget = ref<ApiKey | null>(null)
const ccsClientOpen = ref(false)
const pendingCcsKey = ref<ApiKey | null>(null)
const hasKeyFilters = computed(() => !!(appliedKeySearch.value || keyStatusFilter.value || keyGroupFilter.value))
const busyKeyIds = ref(new Set<number>())
const bulkMode = ref(false)
const bulkFormOpen = ref(false)
const selectedKeyIds = ref<number[]>([])
const selectedKeys = computed(() => keys.value.filter((key) => selectedKeyIds.value.includes(key.id)))
const allPageKeysSelected = computed(() => keys.value.length > 0 && selectedKeys.value.length === keys.value.length)
const somePageKeysSelected = computed(() => selectedKeys.value.length > 0 && !allPageKeysSelected.value)
const actionError = ref('')
const keyFormOpen = ref(false)
const editingKey = ref<ApiKey | null>(null)
const formGroupId = ref<number | undefined>()
const lastCreated = ref<ApiKey | null>(null)
const deleteTarget = ref<ApiKey | null>(null)
const deleteError = ref('')
const deleting = ref(false)
const docsOpen = ref(false)
const quickstartOpen = ref(route.query.quickstart === '1')
watch(
  () => route.query.quickstart,
  (val) => {
    if (val === '1') quickstartOpen.value = true
  }
)
let consumingCreate = false

let disposed = false
let groupRequest = 0
let keyRequest = 0
let keyController: AbortController | undefined

function errorMessage(error: unknown, fallback: string) {
  startRetryCooldown(error)
  return userFacingError(error, locale.value, fallback)
}
async function loadGroups() {
  const request = ++groupRequest
  groupsLoading.value = true
  groupsError.value = ''
  try {
    const [result, rates] = await Promise.all([userGroupsAPI.getAvailable(), userGroupsAPI.getUserGroupRates()])
    if (!disposed && request === groupRequest) {
      groups.value = result
      groupRates.value = rates
    }
  } catch (error) {
    if (!disposed && request === groupRequest) groupsError.value = errorMessage(error, text('无法读取可绑定分组，请重试。', 'Could not load available groups. Please retry.'))
  } finally {
    if (!disposed && request === groupRequest) groupsLoading.value = false
  }
}
async function loadKeys() {
  const request = ++keyRequest
  keyController?.abort()
  const controller = new AbortController()
  keyController = controller
  usageLoading.value = false
  usageError.value = ''
  keysLoading.value = true
  keysError.value = ''
  try {
    const result = await keysAPI.list(
      keyPage.value,
      keyPageSize.value,
      {
        search: appliedKeySearch.value || undefined,
        status: keyStatusFilter.value || undefined,
        group_id: keyGroupFilter.value || undefined,
        sort_by: 'created_at',
        sort_order: 'desc'
      },
      { signal: controller.signal }
    )
    if (disposed || request !== keyRequest) return
    keys.value = result.items || []
    selectedKeyIds.value = selectedKeyIds.value.filter((id) => keys.value.some((key) => key.id === id))
    keyTotal.value = result.total || 0
    keysKnown.value = true
    const visible = new Set(keys.value.map((key) => String(key.id)))
    usageStats.value = Object.fromEntries(Object.entries(usageStats.value).filter(([id]) => visible.has(id)))
    if (keyPage.value > keyPages.value) {
      clearSelection()
      keyPage.value = keyPages.value
      await loadKeys()
      return
    }
    void loadUsage(keys.value.map((key) => key.id), controller.signal, request)
  } catch (error) {
    if (!disposed && request === keyRequest) keysError.value = errorMessage(error, text('密钥加载失败，请重试。', 'Could not load API keys. Please retry.'))
  } finally {
    if (!disposed && request === keyRequest) keysLoading.value = false
  }
}
async function loadUsage(ids: number[], signal: AbortSignal, request: number) {
  if (!ids.length) {
    usageStats.value = {}
    usageError.value = ''
    usageLoading.value = false
    return
  }
  usageLoading.value = true
  usageError.value = ''
  try {
    const usageResponse = await usageAPI.getDashboardApiKeysUsage(ids, { signal })
    if (disposed || request !== keyRequest) return
    usageStats.value = usageResponse.stats || {}
  } catch (cause) {
    if (signal.aborted || disposed || request !== keyRequest) return
    usageError.value = errorMessage(cause, text('用量加载失败；已加载的统计保留，请重试。', 'Usage could not be refreshed. Previous statistics are retained; please retry.'))
  } finally {
    if (!disposed && request === keyRequest) usageLoading.value = false
  }
}
function retryUsage() {
  if (keyController && !usageLoading.value && retryRemaining.value <= 0) {
    void loadUsage(keys.value.map((key) => key.id), keyController.signal, keyRequest)
  }
}
function refreshPage() {
  void loadGroups()
  void loadKeys()
}
async function copyEndpoint() {
  if (apiBaseUrl.value) await copyToClipboard(apiBaseUrl.value)
}
function openUseKey(key: ApiKey) {
  useKeyTarget.value = key
}
function importToCcs(key: ApiKey) {
  const platform = key.group?.platform || 'anthropic'
  if (platform === 'antigravity') {
    pendingCcsKey.value = key
    ccsClientOpen.value = true
    return
  }
  executeCcsImport(key, platform === 'gemini' ? 'gemini' : 'claude')
}
function executeCcsImport(key: ApiKey, clientType: CcSwitchClientType) {
  const providerName = (appStore.cachedPublicSettings?.site_name || appStore.siteName || 'sub2api').trim() || 'sub2api'
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl: apiRoot.value || window.location.origin,
    platform: key.group?.platform || 'anthropic',
    clientType,
    providerName,
    apiKey: key.key,
    usageScript: CC_SWITCH_USAGE_SCRIPT
  })
  try {
    window.open(deeplink, '_self')
    window.setTimeout(() => {
      if (!disposed && document.hasFocus()) appStore.showError(t('keys.ccSwitchNotInstalled'))
    }, 100)
  } catch {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}
function chooseCcsClient(clientType: CcSwitchClientType) {
  if (pendingCcsKey.value) executeCcsImport(pendingCcsKey.value, clientType)
  closeCcsClient()
}
function closeCcsClient() {
  ccsClientOpen.value = false
  pendingCcsKey.value = null
}
function applyKeyFilters() {
  clearSelection()
  appliedKeySearch.value = keySearch.value.trim()
  keyPage.value = 1
  void loadKeys()
}
function resetKeyFilters() {
  keySearch.value = ''
  keyStatusFilter.value = ''
  keyGroupFilter.value = ''
  applyKeyFilters()
}
function goToPage(page: number) {
  if (page < 1 || page > keyPages.value || page === keyPage.value || keysLoading.value) return
  clearSelection()
  keyPage.value = page
  void loadKeys()
}
function changePageSize() {
  clearSelection()
  keyPage.value = 1
  void loadKeys()
}
function clearSelection() { selectedKeyIds.value = [] }
function toggleBulkMode() {
  bulkMode.value = !bulkMode.value
  clearSelection()
}
function selectKey(id: number, checked: boolean) {
  if (!bulkMode.value || keysLoading.value || keysError.value || busyKeyIds.value.has(id) || !keys.value.some((key) => key.id === id)) return
  selectedKeyIds.value = checked ? [...new Set([...selectedKeyIds.value, id])] : selectedKeyIds.value.filter((selected) => selected !== id)
}
function selectPageKeys(checked: boolean) {
  if (!bulkMode.value || keysLoading.value || keysError.value || busyKeyIds.value.size) return
  selectedKeyIds.value = checked ? keys.value.map((key) => key.id) : []
}
function onBulkUpdated(succeededIds: number[]) {
  const succeeded = new Set(succeededIds)
  selectedKeyIds.value = selectedKeyIds.value.filter((id) => !succeeded.has(id))
  void loadKeys()
}
function openCreate(groupId?: number) {
  if (creationDisabled.value) return
  editingKey.value = null
  formGroupId.value = groupId != null && groups.value.some((group) => group.id === groupId) ? groupId : undefined
  keyFormOpen.value = true
}
function openEdit(key: ApiKey) {
  if (groupsLoading.value || groupsError.value) {
    actionError.value = text('请先重试加载可绑定分组，再编辑密钥。', 'Load available groups before editing a key.')
    return
  }
  actionError.value = ''
  editingKey.value = key
  formGroupId.value = undefined
  keyFormOpen.value = true
}
function closeKeyForm() {
  keyFormOpen.value = false
  editingKey.value = null
}
async function onKeySaved(key: ApiKey) {
  clearSelection()
  const created = !editingKey.value
  if (created) {
    lastCreated.value = key
    keySearch.value = ''
    appliedKeySearch.value = ''
    keyStatusFilter.value = ''
    keyGroupFilter.value = ''
    keyPage.value = 1
    // Keep drawer open on creation if full key is returned so user sees the Creation Success state
    if (!key?.key) {
      keyFormOpen.value = false
      editingKey.value = null
    }
  } else {
    keyFormOpen.value = false
    editingKey.value = null
  }
  appStore.showSuccess(created ? text('API Key 已创建', 'API key created') : text('API Key 已保存', 'API key saved'))
  await loadKeys()
}
async function copyKey(key: ApiKey) { await copyToClipboard(key.key) }
async function toggleKey(key: ApiKey) {
  if (busyKeyIds.value.has(key.id)) return
  busyKeyIds.value.add(key.id)
  actionError.value = ''
  try {
    await keysAPI.toggleStatus(key.id, key.status === 'active' ? 'inactive' : 'active')
    if (disposed) return
    appStore.showSuccess(text('密钥状态已更新', 'Key status updated'))
    await loadKeys()
  } catch (error) {
    if (!disposed) actionError.value = errorMessage(error, text('密钥状态更新失败，请重试。', 'Could not update this key. Please retry.'))
  } finally {
    busyKeyIds.value.delete(key.id)
  }
}
function confirmDelete(key: ApiKey) {
  deleteError.value = ''
  deleteTarget.value = key
}
function closeDelete() { if (!deleting.value) deleteTarget.value = null }
async function deleteKey() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = true
  deleteError.value = ''
  const id = deleteTarget.value.id
  try {
    await keysAPI.delete(id)
    if (disposed) return
    deleteTarget.value = null
    if (lastCreated.value?.id === id) lastCreated.value = null
    if (useKeyTarget.value?.id === id) useKeyTarget.value = null
    if (pendingCcsKey.value?.id === id) closeCcsClient()
    appStore.showSuccess(text('API Key 已删除', 'API key deleted'))
    await loadKeys()
  } catch (error) {
    if (!disposed) deleteError.value = errorMessage(error, text('删除失败，请重试。', 'Could not delete this key. Please retry.'))
  } finally {
    deleting.value = false
  }
}
function scrollTo(id: string) {
  void nextTick(() => document.getElementById(id)?.scrollIntoView({ block: 'start', behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' }))
}
function showExamples() {
  examplesOpen.value = true
  scrollTo('portal-api-examples')
}

watch(
  () => route.query.model,
  (value) => {
    const model = typeof value === 'string' ? value : Array.isArray(value) ? value[0] || '' : ''
    if (model && plazaEnabled.value) void router.replace({ path: '/model-plaza', query: { embedded: '1', model } })
  },
  { immediate: true }
)
watch(
  [() => route.query.create, groupsLoading, groupsError],
  async ([create, loading, error]) => {
    if (create !== '1' || loading || error || consumingCreate || disposed) return
    consumingCreate = true
    if (!creationDisabled.value) openCreate()
    else actionError.value = text('当前账户没有可绑定分组，暂时无法创建 API Key。', 'Your account has no available group for creating an API key.')
    const query = { ...route.query }
    delete query.create
    try { await router.replace({ path: route.path, query, hash: route.hash }) } finally { consumingCreate = false }
  },
  { immediate: true }
)
watch(
  () => route.query.guide,
  (value) => {
    if (value !== '1') return
    docsOpen.value = true
    const query = { ...route.query }
    delete query.guide
    void router.replace({ path: route.path, query, hash: route.hash })
  },
  { immediate: true }
)
onMounted(() => {
  void loadGroups()
  void loadKeys()
})
onBeforeUnmount(() => {
  disposed = true
  keyController?.abort()
})
</script>
