<!-- Adapted from ranxi2001/sub2api @ 9620d315, licensed under LGPL-3.0-or-later. -->
<template>
  <div class="space-y-6">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
          {{ t('admin.groupCacheTest.title') }}
        </h1>
        <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
          {{ t('admin.groupCacheTest.description') }}
        </p>
        <p class="mt-1 text-xs leading-5 text-[var(--atelier-ink-muted)]">
          {{ locale.startsWith('zh') ? '仅在点击开始后执行真实请求，不会自动测试。' : 'Requests run only when you start a test. No automatic probes.' }}
        </p>
      </div>
      <UiButton variant="secondary" :disabled="busy || loading" @click="loadGroups">
        {{ t('admin.groupCacheTest.reload') }}
      </UiButton>
    </header>

    <p
      v-if="error"
      role="alert"
      class="rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-err-bg)] p-3 text-sm text-[var(--atelier-err)]"
    >
      {{ error }}
    </p>
    <p v-if="storageError" role="status" class="text-sm text-[var(--atelier-warn)]">
      {{ t('admin.groupCacheTest.storageError') }}
    </p>

    <div class="grid items-start gap-6 xl:grid-cols-[340px_minmax(0,1fr)]">
      <!-- Configuration Panel -->
      <section class="atelier-surface min-w-0 p-5" aria-labelledby="cache-config-title">
        <h2 id="cache-config-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
          {{ t('admin.groupCacheTest.configuration') }}
        </h2>
        <form class="mt-4 space-y-4" @submit.prevent="start">
          <fieldset :disabled="busy || loading" class="space-y-4">
            <div class="cache-field" data-testid="cache-group">
              <span>{{ t('admin.groupCacheTest.group') }}</span>
              <UiSelect
                :model-value="groupId"
                :options="groupOptions"
                :disabled="busy || loading"
                :aria-label="t('admin.groupCacheTest.group')"
                @update:model-value="(val) => (groupId = val === null ? null : Number(val))"
              />
            </div>

            <div class="cache-field" data-testid="cache-key">
              <span>{{ t('admin.groupCacheTest.key') }}</span>
              <UiSelect
                :model-value="keyId"
                :options="keyOptions"
                :disabled="busy || loading || keysLoading || !groupId"
                :aria-label="t('admin.groupCacheTest.key')"
                @update:model-value="(val) => (keyId = val === null ? null : Number(val))"
              />
            </div>

            <p class="cache-hint">{{ t('admin.groupCacheTest.ownedKeyHint') }}</p>
            <p v-if="keyError" role="alert" class="text-sm text-[var(--atelier-err)]">
              {{ t('admin.groupCacheTest.keysError') }}
              <button type="button" class="underline" @click="loadKeys">{{ t('admin.groupCacheTest.reload') }}</button>
            </p>
            <p v-else-if="groupId && !keysLoading && !keys.length" class="cache-hint">
              {{ t('admin.groupCacheTest.noKeys') }}
              <RouterLink to="/keys" class="text-[var(--atelier-ink)] underline underline-offset-2">{{ t('admin.groupCacheTest.manageKeys') }}</RouterLink>
            </p>

            <div class="cache-field" data-testid="cache-protocol">
              <span>{{ t('admin.groupCacheTest.protocol') }}</span>
              <UiSelect
                :model-value="settings.protocol"
                :options="protocolOptions"
                :disabled="busy || loading"
                :aria-label="t('admin.groupCacheTest.protocol')"
                @update:model-value="(val) => (settings.protocol = val as typeof settings.protocol)"
              />
            </div>

            <label class="cache-field">
              <span>{{ t('admin.groupCacheTest.model') }}</span>
              <input
                v-model.trim="settings.model"
                list="cache-models"
                class="input font-mono"
                maxlength="200"
                required
                autocomplete="off"
                :placeholder="t('admin.groupCacheTest.modelPlaceholder')"
                data-testid="cache-model"
              />
              <datalist id="cache-models">
                <option v-for="model in models" :key="model" :value="model" />
              </datalist>
            </label>

            <div class="grid grid-cols-2 gap-3">
              <div class="cache-field">
                <span>{{ t('admin.groupCacheTest.rounds') }}</span>
                <UiSelect
                  :model-value="settings.rounds"
                  :options="roundsOptions"
                  :disabled="busy || loading"
                  :aria-label="t('admin.groupCacheTest.rounds')"
                  @update:model-value="(val) => (settings.rounds = Number(val))"
                />
              </div>
              <div class="cache-field">
                <span>{{ t('admin.groupCacheTest.prefix') }}</span>
                <UiSelect
                  :model-value="settings.prefixBlocks"
                  :options="prefixBlockOptions"
                  :disabled="busy || loading"
                  :aria-label="t('admin.groupCacheTest.prefix')"
                  @update:model-value="(val) => (settings.prefixBlocks = Number(val))"
                />
              </div>
            </div>

            <p class="cache-hint">{{ t('admin.groupCacheTest.prefixHint', { count: prefixLength.toLocaleString() }) }}</p>

            <details class="rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/50 p-3">
              <summary class="cursor-pointer text-sm font-medium text-[var(--atelier-ink)]">
                {{ t('admin.groupCacheTest.advanced') }}
              </summary>
              <div class="mt-3 space-y-3">
                <label class="cache-field">
                  <span>{{ t('admin.groupCacheTest.interval') }}</span>
                  <input v-model.number="settings.intervalMs" class="input font-mono tabular-nums" type="number" min="0" max="10000" step="100" required />
                </label>
                <div class="cache-field">
                  <span>{{ t('admin.groupCacheTest.timeout') }}</span>
                  <UiSelect
                    :model-value="settings.timeoutMs"
                    :options="timeoutOptions"
                    :disabled="busy || loading"
                    :aria-label="t('admin.groupCacheTest.timeout')"
                    @update:model-value="(val) => (settings.timeoutMs = Number(val))"
                  />
                </div>
                <label class="cache-field">
                  <span>{{ t('admin.groupCacheTest.maxOutput') }}</span>
                  <input v-model.number="settings.maxOutputTokens" class="input font-mono tabular-nums" type="number" min="64" max="4096" required />
                </label>
                <template v-if="settings.protocol !== 'messages'">
                  <div class="cache-field">
                    <span>{{ t('admin.groupCacheTest.reasoning') }}</span>
                    <UiSelect
                      :model-value="settings.reasoningEffort"
                      :options="reasoningEffortOptions"
                      :disabled="busy || loading"
                      :aria-label="t('admin.groupCacheTest.reasoning')"
                      @update:model-value="(val) => (settings.reasoningEffort = String(val ?? '') as typeof settings.reasoningEffort)"
                    />
                  </div>
                  <div class="cache-field">
                    <span>{{ t('admin.groupCacheTest.tier') }}</span>
                    <UiSelect
                      :model-value="settings.serviceTier"
                      :options="serviceTierOptions"
                      :disabled="busy || loading"
                      :aria-label="t('admin.groupCacheTest.tier')"
                      @update:model-value="(val) => (settings.serviceTier = String(val ?? '') as typeof settings.serviceTier)"
                    />
                  </div>
                </template>
                <p class="cache-hint">{{ t('admin.groupCacheTest.compatibility') }}</p>
              </div>
            </details>
          </fieldset>

          <div class="rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-warn-bg)] p-3 text-sm leading-6 text-[var(--atelier-warn)]">
            {{ t('admin.groupCacheTest.chargeWarning', { count: settings.rounds }) }}
          </div>
          <div class="flex gap-2">
            <UiButton type="submit" class="flex-1" :disabled="!canStart" data-testid="cache-start">
              {{ t('admin.groupCacheTest.start') }}
            </UiButton>
            <UiButton v-if="busy" type="button" variant="secondary" :disabled="stopping" data-testid="cache-stop" @click="stop">
              {{ stopping ? t('admin.groupCacheTest.stopping') : t('admin.groupCacheTest.stop') }}
            </UiButton>
          </div>
        </form>
      </section>

      <!-- Results & History -->
      <div class="min-w-0 space-y-5">
        <section class="atelier-surface min-w-0 p-5" aria-labelledby="cache-results-title">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 id="cache-results-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ t('admin.groupCacheTest.results') }}
            </h2>
            <div v-if="selected" class="flex flex-wrap gap-2">
              <UiButton variant="secondary" size="sm" :disabled="busy || !hasPendingBills" data-testid="cache-reconcile" @click="refreshBills">
                {{ t('admin.groupCacheTest.reconcile') }}
              </UiButton>
              <UiButton variant="secondary" size="sm" :disabled="busy" @click="download">
                {{ t('admin.groupCacheTest.export') }}
              </UiButton>
            </div>
          </div>

          <div v-if="!selected" class="py-16 text-center">
            <p class="font-medium text-[var(--atelier-ink)]">{{ t('admin.groupCacheTest.emptyTitle') }}</p>
            <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.emptyHint') }}</p>
          </div>

          <template v-else>
            <div class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-[var(--atelier-ink)]">
              <strong>{{ selected.groupName }}</strong>
              <span class="font-mono text-xs">{{ selected.settings.model }}</span>
              <code class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-xs text-[var(--atelier-ink-secondary)]">{{ CACHE_ENDPOINTS[selected.settings.protocol] }}</code>
              <span role="status" aria-live="polite" class="ml-auto inline-flex items-center gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <UiStatusDot :tone="busy ? 'info' : selected.status === 'completed' ? 'ok' : selected.status === 'failed' ? 'err' : 'neutral'" />
                {{ busy ? t(reconciling ? 'admin.groupCacheTest.reconciling' : 'admin.groupCacheTest.testing') : t('admin.groupCacheTest.status.' + selected.status) }}
              </span>
            </div>
            <p class="mt-2 text-xs text-[var(--atelier-ink-muted)]">{{ runConfiguration(selected) }}</p>

            <div class="my-5 grid grid-cols-2 gap-3 2xl:grid-cols-4">
              <div class="atelier-surface-subtle flex flex-col gap-1 p-3">
                <span class="text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.baselineHit') }}</span>
                <strong class="font-mono text-xl font-semibold tabular-nums text-[var(--atelier-ink)]">{{ percent(cacheHitRate(selected.rounds[0]?.tokens ?? null)) }}</strong>
              </div>
              <div class="atelier-surface-subtle flex flex-col gap-1 p-3">
                <span class="text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.repeatHit') }}</span>
                <strong class="font-mono text-xl font-semibold tabular-nums text-[var(--atelier-ink)]">{{ percent(cacheHitRate(repeatTokens)) }}</strong>
                <small class="text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.reportedRounds', { count: comparableRounds }) }}</small>
              </div>
              <div class="atelier-surface-subtle flex flex-col gap-1 p-3">
                <span class="text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.cacheRead') }}</span>
                <strong class="font-mono text-xl font-semibold tabular-nums text-[var(--atelier-ink)]">{{ number(repeatTokens.read) }}</strong>
                <small class="text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.repeatOnly') }}</small>
              </div>
              <div class="atelier-surface-subtle flex flex-col gap-1 p-3">
                <span class="text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.billed') }}</span>
                <strong class="font-mono text-xl font-semibold tabular-nums text-[var(--atelier-ink)]">{{ money(knownCost) }}</strong>
                <small class="text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.billsMatched', { count: matchedBills, total: selected.rounds.length }) }}</small>
              </div>
            </div>

            <p class="cache-hint">{{ t('admin.groupCacheTest.resultHint') }}</p>

            <div class="mt-4 overflow-x-auto">
              <div v-if="mobile" class="mobile-records">
                <dl v-for="round in selected.rounds" :key="round.index" class="mobile-record">
                  <div class="mobile-record-field">
                    <dt>{{ t('admin.groupCacheTest.round') }}</dt>
                    <dd>
                      <button
                        class="text-left font-medium text-[var(--atelier-ink)] underline underline-offset-4"
                        :aria-expanded="detailIndex === round.index"
                        @click="detailIndex = detailIndex === round.index ? null : round.index"
                      >
                        {{ round.index === 1 ? t('admin.groupCacheTest.baseline') : t('admin.groupCacheTest.repeat', { index: round.index - 1 }) }}
                      </button>
                      <small :class="round.status === 'failed' ? 'text-[var(--atelier-err)]' : 'text-[var(--atelier-ink-muted)]'">
                        {{ t('admin.groupCacheTest.status.' + round.status) }}
                      </small>
                    </dd>
                  </div>
                  <div class="mobile-record-field"><dt>{{ t('admin.groupCacheTest.input') }}</dt><dd class="font-mono tabular-nums">{{ number(round.tokens?.input) }}</dd></div>
                  <div class="mobile-record-field"><dt>{{ t('admin.groupCacheTest.cacheRead') }}</dt><dd class="font-mono tabular-nums">{{ number(round.tokens?.read) }}</dd></div>
                  <div class="mobile-record-field"><dt>{{ t('admin.groupCacheTest.cacheWrite') }}</dt><dd class="font-mono tabular-nums">{{ number(round.tokens?.write) }}</dd></div>
                  <div class="mobile-record-field"><dt>{{ t('admin.groupCacheTest.hitRate') }}</dt><dd class="font-mono tabular-nums">{{ percent(cacheHitRate(round.tokens)) }}</dd></div>
                  <div class="mobile-record-field"><dt>{{ t('admin.groupCacheTest.output') }}</dt><dd class="font-mono tabular-nums">{{ number(round.tokens?.output) }}</dd></div>
                  <div class="mobile-record-field"><dt>{{ t('admin.groupCacheTest.firstText') }}</dt><dd class="font-mono tabular-nums">{{ ms(round.firstTextMs) }}</dd></div>
                  <div class="mobile-record-field">
                    <dt>{{ t('admin.groupCacheTest.billed') }}</dt>
                    <dd class="font-mono tabular-nums">{{ money(round.bill?.actualCost) }}<small>{{ t('admin.groupCacheTest.billing.' + round.billingState) }}</small></dd>
                  </div>
                </dl>
              </div>
              <table v-else class="atelier-table whitespace-nowrap font-mono tabular-nums">
                <thead>
                  <tr>
                    <th>{{ t('admin.groupCacheTest.round') }}</th>
                    <th>{{ t('admin.groupCacheTest.input') }}</th>
                    <th>{{ t('admin.groupCacheTest.cacheRead') }}</th>
                    <th>{{ t('admin.groupCacheTest.cacheWrite') }}</th>
                    <th>{{ t('admin.groupCacheTest.hitRate') }}</th>
                    <th>{{ t('admin.groupCacheTest.output') }}</th>
                    <th>{{ t('admin.groupCacheTest.firstText') }}</th>
                    <th>{{ t('admin.groupCacheTest.billed') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="round in selected.rounds" :key="round.index" :data-selected="detailIndex === round.index ? 'true' : undefined">
                    <td class="font-sans">
                      <button
                        class="text-left font-medium text-[var(--atelier-ink)] underline underline-offset-4"
                        :aria-expanded="detailIndex === round.index"
                        @click="detailIndex = detailIndex === round.index ? null : round.index"
                      >
                        {{ round.index === 1 ? t('admin.groupCacheTest.baseline') : t('admin.groupCacheTest.repeat', { index: round.index - 1 }) }}
                      </button>
                      <small class="mt-0.5 block text-xs" :class="round.status === 'failed' ? 'text-[var(--atelier-err)]' : 'text-[var(--atelier-ink-muted)]'">
                        {{ t('admin.groupCacheTest.status.' + round.status) }}
                      </small>
                    </td>
                    <td>{{ number(round.tokens?.input) }}</td>
                    <td>{{ number(round.tokens?.read) }}</td>
                    <td>{{ number(round.tokens?.write) }}</td>
                    <td>{{ percent(cacheHitRate(round.tokens)) }}</td>
                    <td>{{ number(round.tokens?.output) }}</td>
                    <td>{{ ms(round.firstTextMs) }}</td>
                    <td>
                      {{ money(round.bill?.actualCost) }}
                      <small class="mt-0.5 block font-sans text-xs text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.billing.' + round.billingState) }}</small>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div v-if="detail" class="atelier-surface-subtle mt-4 p-4" data-testid="cache-detail">
              <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">{{ t('admin.groupCacheTest.roundDetails', { index: detail.index }) }}</h3>
              <p v-if="detail.error" role="alert" class="mt-2 text-sm text-[var(--atelier-err)]">
                {{ t('admin.groupCacheTest.errors.' + detail.error) }}<span v-if="detail.httpStatus"> · HTTP {{ detail.httpStatus }}</span>
              </p>
              <dl class="cache-details mt-3">
                <dt>{{ t('admin.groupCacheTest.requestId') }}</dt><dd class="break-all font-mono text-xs">{{ detail.requestId || t('admin.groupCacheTest.notReported') }}</dd>
                <dt>{{ t('admin.groupCacheTest.modelReturned') }}</dt><dd class="font-mono text-xs">{{ detail.responseModel || '—' }}</dd>
                <dt>{{ t('admin.groupCacheTest.tierReturned') }}</dt><dd>{{ detail.serviceTier || t('admin.groupCacheTest.notReported') }}</dd>
                <dt>{{ t('admin.groupCacheTest.duration') }}</dt><dd class="font-mono tabular-nums">{{ ms(detail.durationMs) }}</dd>
              </dl>
              <template v-if="detail.bill">
                <h4 class="mt-4 text-sm font-semibold text-[var(--atelier-ink)]">{{ t('admin.groupCacheTest.billDetails') }}</h4>
                <dl class="cache-details mt-3">
                  <dt>{{ t('admin.groupCacheTest.route') }}</dt><dd>{{ t('admin.groupCacheTest.routeValue', { group: detail.bill.groupId ?? '—', account: detail.bill.accountId ?? '—' }) }} · {{ detail.bill.upstreamModel || detail.bill.model }}</dd>
                  <dt>{{ t('admin.groupCacheTest.tier') }} / {{ t('admin.groupCacheTest.reasoning') }}</dt><dd>{{ detail.bill.serviceTier || '—' }} / {{ detail.bill.reasoningEffort || '—' }}</dd>
                  <dt>{{ t('admin.groupCacheTest.billedTokens') }}</dt><dd class="font-mono tabular-nums">{{ number(detail.bill.tokens.input) }} / {{ number(detail.bill.tokens.read) }} / {{ number(detail.bill.tokens.write) }} / {{ number(detail.bill.tokens.output) }}</dd>
                  <dt>{{ t('admin.groupCacheTest.componentCosts') }}</dt><dd class="font-mono tabular-nums">{{ money(detail.bill.inputCost) }} / {{ money(detail.bill.readCost) }} / {{ money(detail.bill.writeCost) }} / {{ money(detail.bill.outputCost) }}</dd>
                  <dt>{{ t('admin.groupCacheTest.standardCost') }}</dt><dd class="font-mono tabular-nums">{{ money(detail.bill.totalCost) }} · {{ t('admin.groupCacheTest.multiplier') }} {{ number(detail.bill.multiplier) }}</dd>
                  <dt>{{ t('admin.groupCacheTest.billed') }}</dt><dd class="font-mono tabular-nums">{{ money(detail.bill.actualCost) }}</dd>
                  <dt>{{ t('admin.groupCacheTest.serverFirstToken') }}</dt><dd class="font-mono tabular-nums">{{ ms(detail.bill.firstTokenMs) }}</dd>
                </dl>
              </template>
              <p class="mt-3 cache-hint">{{ t('admin.groupCacheTest.billingHint') }}</p>
            </div>
            <p v-if="hasPendingBills" class="mt-4 text-sm text-[var(--atelier-warn)]">{{ t('admin.groupCacheTest.pendingHint') }}</p>
          </template>
        </section>

        <!-- History Panel -->
        <section class="atelier-surface min-w-0 p-5" aria-labelledby="cache-history-title">
          <div class="flex items-center justify-between gap-3">
            <h2 id="cache-history-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ t('admin.groupCacheTest.history') }}
            </h2>
            <button
              class="text-xs text-[var(--atelier-ink-muted)] underline underline-offset-2 hover:text-[var(--atelier-ink)] disabled:opacity-45"
              :disabled="busy || !history.length"
              @click="clearHistory"
            >
              {{ t('admin.groupCacheTest.clearHistory') }}
            </button>
          </div>
          <p class="mt-1.5 cache-hint">{{ t('admin.groupCacheTest.historyHint') }}</p>
          <p v-if="!history.length" class="py-6 text-sm text-[var(--atelier-ink-muted)]">{{ t('admin.groupCacheTest.noHistory') }}</p>
          <ul v-else class="mt-3 divide-y divide-[var(--atelier-border-subtle)]">
            <li v-for="run in history" :key="run.id">
              <button
                class="flex w-full flex-wrap items-center gap-x-4 gap-y-1 rounded-lg px-3 py-2.5 text-left text-sm transition-colors hover:bg-[var(--atelier-hover)] disabled:opacity-50"
                :class="selected?.id === run.id ? 'bg-[var(--atelier-selected-row)]' : ''"
                :disabled="busy"
                :aria-pressed="selected?.id === run.id"
                @click="selected = run; detailIndex = null"
              >
                <span class="min-w-32 font-medium text-[var(--atelier-ink)]">{{ run.groupName }}</span>
                <span class="font-mono text-xs text-[var(--atelier-ink-secondary)]">{{ run.settings.model }}</span>
                <span class="text-xs text-[var(--atelier-ink-muted)]">{{ date(run.startedAt) }}</span>
                <span class="ml-auto font-mono text-xs tabular-nums text-[var(--atelier-ink-secondary)]">
                  {{ t('admin.groupCacheTest.status.' + run.status) }} · {{ run.rounds.length }} / {{ run.settings.rounds }}
                </span>
              </button>
            </li>
          </ul>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { saveAs } from 'file-saver'
import { useAuthStore } from '@/stores/auth'
import * as groupsAPI from '@/api/admin/groups'
import * as keysAPI from '@/api/keys'
import { executeCacheRun, reconcileCacheBills } from '@/api/groupCacheTest'
import { CACHE_ENDPOINTS, DEFAULT_CACHE_SETTINGS, aggregateCacheTokens, buildCachePrefix, cacheHitRate, createCacheTestId, exportCacheRun, loadCacheHistory, saveCacheHistory, validCacheSettings, type CacheTestRun } from '@/utils/groupCacheTest'
import type { AdminGroup, ApiKey } from '@/types'
import cacheTestZh from '@/i18n/locales/zh/admin/groupCacheTest'
import cacheTestEn from '@/i18n/locales/en/admin/groupCacheTest'
import { UiButton, UiSelect, UiStatusDot, type UiSelectOption } from '@/ui'

const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
const { t, locale } = useI18n({ useScope: 'local', messages: { zh: { admin: cacheTestZh }, en: { admin: cacheTestEn } } })
const auth = useAuthStore()
const settings = reactive({ ...DEFAULT_CACHE_SETTINGS })
const groups = ref<AdminGroup[]>([])
const keys = ref<Pick<ApiKey, 'id' | 'name'>[]>([])
const models = ref<string[]>([])
const groupId = ref<number | null>(null)
const keyId = ref<number | null>(null)
const loading = ref(false)
const keysLoading = ref(false)
const keyError = ref(false)
const busy = ref(false)
const reconciling = ref(false)
const stopping = ref(false)
const error = ref('')
const storageError = ref(false)
const selected = ref<CacheTestRun | null>(null)
const history = ref<CacheTestRun[]>([])
const detailIndex = ref<number | null>(null)
let alive = true, generation = 0, keysController: AbortController | null = null, runController: AbortController | null = null

const groupOptions = computed<UiSelectOption[]>(() => [
  { value: null, label: t('admin.groupCacheTest.selectGroup') },
  ...groups.value.map((g) => ({ value: g.id, label: `${g.name} · ${g.platform}` }))
])

const keyOptions = computed<UiSelectOption[]>(() => [
  { value: null, label: keysLoading.value ? t('admin.groupCacheTest.loading') : t('admin.groupCacheTest.selectKey') },
  ...keys.value.map((k) => ({ value: k.id, label: `${k.name} · #${k.id}` }))
])

const protocolOptions: UiSelectOption[] = [
  { value: 'responses', label: 'Responses' },
  { value: 'chat', label: 'Chat Completions' },
  { value: 'messages', label: 'Messages' }
]

const roundsOptions: UiSelectOption[] = [2, 3, 4, 5, 6].map((count) => ({
  value: count,
  label: String(count)
}))

const prefixBlockOptions = computed<UiSelectOption[]>(() =>
  [16, 48, 96, 128].map((count) => ({
    value: count,
    label: t('admin.groupCacheTest.blocks', { count })
  }))
)

const timeoutOptions: UiSelectOption[] = [
  { value: 60000, label: '60 s' },
  { value: 120000, label: '120 s' },
  { value: 300000, label: '300 s' }
]

const reasoningEffortOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: t('admin.groupCacheTest.unspecified') },
  ...['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'].map((effort) => ({
    value: effort,
    label: effort
  }))
])

const serviceTierOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: t('admin.groupCacheTest.unspecified') },
  { value: 'default', label: 'default' },
  { value: 'priority', label: 'priority' }
])

const group = computed(() => groups.value.find(g => g.id === groupId.value))
const canStart = computed(() => !!auth.user && !!group.value && !!keyId.value && keys.value.some(k => k.id === keyId.value) && !busy.value && !loading.value && !keysLoading.value && !keyError.value && validCacheSettings(settings))
const prefixLength = computed(() => buildCachePrefix('00000000-0000-0000-0000-000000000000', settings.prefixBlocks).length)
const detail = computed(() => selected.value?.rounds.find(r => r.index === detailIndex.value))
const repeatTokens = computed(() => aggregateCacheTokens(selected.value?.rounds.slice(1) ?? []))
const comparableRounds = computed(() => selected.value?.rounds.slice(1).filter(r => r.status === 'completed' && cacheHitRate(r.tokens) !== null).length ?? 0)
const matchedBills = computed(() => selected.value?.rounds.filter(r => r.bill?.actualCost != null).length ?? 0)
const knownCost = computed(() => matchedBills.value ? selected.value!.rounds.reduce((sum, r) => sum + (r.bill?.actualCost ?? 0), 0) : null)
const hasPendingBills = computed(() => selected.value?.rounds.some(r => !r.bill) ?? false)
const number = (n: number | null | undefined) => n == null ? '—' : n.toLocaleString()
const percent = (n: number | null) => n == null ? '—' : (n * 100).toFixed(1) + '%'
const money = (n: number | null | undefined) => n == null ? '—' : '$' + n.toFixed(6)
const ms = (n: number | null | undefined) => n == null ? '—' : (n / 1000).toFixed(2) + ' s'
const date = (value: string) => new Date(value).toLocaleString()

function runConfiguration(run: CacheTestRun) {
  const s = run.settings
  return t('admin.groupCacheTest.runConfig', { blocks: s.prefixBlocks, output: s.maxOutputTokens, interval: s.intervalMs, tier: s.serviceTier || t('admin.groupCacheTest.unspecified'), reasoning: s.reasoningEffort || t('admin.groupCacheTest.unspecified') })
}

// Smart-routing keys can select another group, so use a single-group key for an attributable test.
function isSingleGroupKey(key: ApiKey, id: number): boolean {
  return key.group_id === id && (!key.routing_group_ids?.length || (key.routing_group_ids.length === 1 && key.routing_group_ids[0] === id))
}

async function loadGroups() {
  const request = ++generation
  loading.value = true; error.value = ''
  try {
    const result = await groupsAPI.getAll()
    if (!alive || request !== generation) return
    groups.value = result.filter(group => [72, 237, 179, 223].includes(group.id))
    if (!result.some(g => g.id === groupId.value)) groupId.value = null
    else await loadKeys()
  } catch { if (alive && request === generation) error.value = t('admin.groupCacheTest.loadError') }
  finally { if (alive && request === generation) loading.value = false }
}

async function loadKeys() {
  keysController?.abort()
  const controller = new AbortController(); keysController = controller
  const id = groupId.value, userId = auth.user?.id
  keys.value = []; keyId.value = null; keyError.value = false
  if (!id || !userId) { keysLoading.value = false; return }
  keysLoading.value = true
  try {
    const all: Pick<ApiKey, 'id' | 'name'>[] = []
    for (let page = 1; ; page++) {
      const result = await keysAPI.list(page, 100, { group_id: id, status: 'active' }, { signal: controller.signal })
      if (!alive || controller.signal.aborted) return
      all.push(...result.items.filter(k => k.user_id === userId && isSingleGroupKey(k, id) && k.status === 'active').map(k => ({ id: k.id, name: k.name })))
      if (page >= result.pages || !result.items.length) break
    }
    keys.value = all
    if (all.length === 1) keyId.value = all[0].id
  } catch { if (!controller.signal.aborted && alive) keyError.value = true }
  finally { if (!controller.signal.aborted && alive) keysLoading.value = false }
}

watch(groupId, async () => {
  models.value = []; settings.model = [179, 223].includes(groupId.value ?? 0) ? 'deepseek-v4.1-flash' : ''
  settings.protocol = [179, 223].includes(groupId.value ?? 0) ? 'chat' : group.value?.platform === 'anthropic' ? 'messages' : 'responses'
  void loadKeys()
  const id = groupId.value
  if (!id) return
  try { const result = await groupsAPI.getModelAllowlistCandidates(id); if (alive && id === groupId.value) models.value = result } catch { /* Manual model entry stays available. */ }
})

watch(() => settings.protocol, value => { if (value === 'messages') { settings.reasoningEffort = ''; settings.serviceTier = '' } })

function persist(run: CacheTestRun, userId: number) {
  if (!alive || userId !== auth.user?.id) return
  history.value = [run, ...history.value.filter(item => item.id !== run.id)].slice(0, 20)
  storageError.value = !saveCacheHistory(userId, history.value)
}

async function start() {
  if (!canStart.value || !auth.user || !group.value || !keyId.value) return
  const userId = auth.user.id, targetGroup = group.value, targetKey = keyId.value
  const snapshot = { ...settings }, controller = new AbortController(); runController = controller
  busy.value = true; stopping.value = false; error.value = ''; detailIndex.value = null
  try {
    const key = await keysAPI.getById(targetKey)
    if (controller.signal.aborted || !alive || userId !== auth.user?.id) return
    if (key.user_id !== userId || !isSingleGroupKey(key, targetGroup.id) || key.status !== 'active' || !key.key || (key.expires_at && Date.parse(key.expires_at) <= Date.now()) || (key.quota > 0 && key.quota_used >= key.quota)) {
      error.value = t('admin.groupCacheTest.keyChanged'); return
    }
    selected.value = { version: 1, id: createCacheTestId(), startedAt: new Date().toISOString(), finishedAt: null, groupId: targetGroup.id, groupName: targetGroup.name, keyId: key.id, keyName: key.name, settings: snapshot, status: 'running', rounds: [] }
    const run = selected.value
    const changed = () => persist(run, userId)
    await executeCacheRun(run, key.key, controller.signal, changed)
    if (!controller.signal.aborted) {
      reconciling.value = true
      await reconcileCacheBills(run, userId, controller.signal, changed)
    }
  } catch { if (alive && !controller.signal.aborted) error.value = t('admin.groupCacheTest.startError') }
  finally { if (alive) { busy.value = false; reconciling.value = false; stopping.value = false } }
}

function stop() { stopping.value = true; runController?.abort() }

async function refreshBills() {
  if (busy.value || !selected.value || !auth.user) return
  const run = selected.value, userId = auth.user.id, controller = new AbortController(); runController = controller
  busy.value = true; reconciling.value = true; stopping.value = false
  try { await reconcileCacheBills(run, userId, controller.signal, () => persist(run, userId)) }
  finally { if (alive) { busy.value = false; reconciling.value = false; stopping.value = false } }
}

function download() {
  if (selected.value) saveAs(new Blob([exportCacheRun(selected.value)], { type: 'application/json' }), 'group-cache-test-' + selected.value.id + '.json')
}

function clearHistory() { history.value = []; selected.value = null; if (auth.user) storageError.value = !saveCacheHistory(auth.user.id, []) }

watch(() => auth.user?.id, () => {
  generation++; keysController?.abort(); runController?.abort(); selected.value = null; history.value = []; keys.value = []; groups.value = []; groupId.value = null
  if (auth.user && alive) { history.value = loadCacheHistory(auth.user.id); void loadGroups() }
})

onMounted(() => { if (auth.user) history.value = loadCacheHistory(auth.user.id); void loadGroups() })
onBeforeUnmount(() => { alive = false; generation++; keysController?.abort(); runController?.abort() })
</script>

<style scoped>
.cache-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--atelier-ink-secondary);
}
.cache-hint {
  font-size: 0.75rem;
  line-height: 1.25rem;
  color: var(--atelier-ink-muted);
}
.cache-details {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0.5rem 1rem;
  font-size: 0.875rem;
}
@media (min-width: 640px) {
  .cache-details {
    grid-template-columns: auto minmax(0, 1fr);
  }
}
.cache-details dt {
  color: var(--atelier-ink-muted);
}
.cache-details dd {
  overflow-wrap: anywhere;
  color: var(--atelier-ink);
}
</style>
