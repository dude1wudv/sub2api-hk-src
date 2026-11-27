<!-- Adapted from ranxi2001/sub2api @ 9620d315, licensed under LGPL-3.0-or-later. -->
<template>
  <section class="space-y-5" :aria-label="t('traffic.title')" :aria-busy="loading">
    <!-- Filter Toolbar -->
    <div class="atelier-surface p-4 sm:p-5">
      <form class="flex flex-wrap items-end gap-3.5" @submit.prevent="applyFilters">
        <div class="w-36">
          <label for="traffic-period" class="mb-1.5 block text-xs font-medium text-[var(--atelier-ink-secondary)]">
            {{ t('traffic.period') }}
          </label>
          <UiSelect
            id="traffic-period"
            :model-value="period"
            :options="periodOptions"
            :aria-label="t('traffic.period')"
            @update:model-value="handlePeriodChange"
          />
        </div>

        <div class="w-40">
          <label for="traffic-date" class="mb-1.5 block text-xs font-medium text-[var(--atelier-ink-secondary)]">
            {{ t('traffic.date') }}
          </label>
          <input
            id="traffic-date"
            :value="date || currentDate"
            type="date"
            class="atelier-input"
            @change="setDate"
          />
        </div>

        <UsageFilters
          v-if="admin"
          :key="searchResetKey"
          v-model="searchFilters"
          :api-key-search="searchAdminKeys"
          @change="applyFilters"
        />
        <div v-else class="min-w-[200px] flex-1 max-w-xs">
          <label for="traffic-key" class="mb-1.5 block text-xs font-medium text-[var(--atelier-ink-secondary)]">
            {{ t('usage.apiKeyFilter') }}
          </label>
          <Select
            :key="searchResetKey"
            id="traffic-key"
            v-model="searchFilters.api_key_id"
            :options="ownKeyOptions"
            searchable
            remote
            :loading="ownKeyLoading"
            @click="onOwnKeyOpen"
            @keydown.down="onOwnKeyOpen"
            @keydown.up="onOwnKeyOpen"
            @search="searchOwnKeys"
            @change="selectOwnKey"
          >
            <template #selected="{ option }">
              {{ searchFilters.api_key_id ? selectedKeyLabel : option?.label || t('usage.allApiKeys') }}
            </template>
          </Select>
          <p v-if="ownKeyError" role="alert" class="mt-1.5 text-xs text-[var(--atelier-err)]">
            {{ t('traffic.keysFailed') }}
            <button type="button" class="ml-1 underline" @click="searchOwnKeys(ownKeyQuery)">
              {{ t('common.refresh') }}
            </button>
          </p>
        </div>

        <div class="ml-auto flex flex-wrap items-center gap-2">
          <UiButton variant="primary" size="md" type="submit" :disabled="loading">
            <Icon name="search" size="sm" />
            {{ t('traffic.apply') }}
          </UiButton>
          <UiButton variant="secondary" size="md" type="button" :disabled="loading" @click="load">
            <Icon name="refresh" size="sm" />
            {{ t('common.refresh') }}
          </UiButton>
          <UiButton variant="ghost" size="md" type="button" @click="reset">
            {{ t('common.reset') }}
          </UiButton>
        </div>
      </form>

      <div class="mt-3.5 flex flex-wrap items-center justify-between gap-2 border-t border-[var(--atelier-border)] pt-3 text-xs text-[var(--atelier-ink-muted)]">
        <p>{{ t('traffic.keyHistoryHint') }}</p>
        <span class="inline-flex items-center gap-1.5 rounded-full bg-[var(--atelier-subtle)] px-2.5 py-1 text-[11px] font-medium text-[var(--atelier-ink-secondary)]">
          <Icon name="shield" size="xs" />
          {{ t('traffic.noCharge') }}
        </span>
      </div>
    </div>

    <!-- Period Range & Navigation -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <span class="text-xs font-medium text-[var(--atelier-ink-muted)]">{{ t('traffic.overview') }}</span>
        <p v-if="report" class="mt-0.5 flex flex-wrap items-center gap-2 font-mono text-sm font-medium tabular-nums text-[var(--atelier-ink)]">
          <span>{{ rangeLabel }}</span>
          <span class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 text-[11px] text-[var(--atelier-ink-secondary)]">
            {{ report.timezone }}
          </span>
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UiButton variant="secondary" size="sm" type="button" :disabled="loading || !report" @click="shift(-1)">
          <Icon name="chevronLeft" size="sm" />
          <span>{{ t('traffic.previousPeriod') }}</span>
        </UiButton>
        <UiButton variant="secondary" size="sm" type="button" :disabled="loading || !report" @click="shift(1)">
          <span>{{ t('traffic.nextPeriod') }}</span>
          <Icon name="chevronRight" size="sm" />
        </UiButton>
      </div>
    </div>

    <p v-if="error" role="alert" class="rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-xs text-[var(--atelier-err)]">
      {{ error }}
    </p>
    <p v-if="summaryLoading" role="status" class="text-xs text-[var(--atelier-ink-muted)]">
      {{ t('traffic.loading') }}
    </p>

    <template v-if="summary">
      <!-- Borderless Horizontal Metric Ribbon -->
      <div class="border-b border-[var(--atelier-border)] pb-5">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p class="text-xs font-medium text-[var(--atelier-ink-muted)]">{{ t('traffic.total') }}</p>
            <div class="mt-1 flex items-baseline gap-2.5">
              <strong
                :title="`${summary.total_bytes} B`"
                class="font-mono text-3xl font-semibold tracking-tight tabular-nums text-[var(--atelier-ink)]"
              >
                {{ bytes(summary.total_bytes) }}
              </strong>
              <span class="text-xs text-[var(--atelier-ink-muted)]">{{ t('traffic.totalHint') }}</span>
            </div>
          </div>

          <div class="flex flex-wrap items-start gap-x-6 gap-y-3 lg:flex-nowrap">
            <div class="min-w-[115px]">
              <p class="text-xs text-[var(--atelier-ink-muted)]">{{ t('traffic.upload') }}</p>
              <strong :title="`${summary.upload_bytes} B`" class="mt-1 block font-mono text-lg font-semibold tabular-nums text-[var(--atelier-ink)]">
                {{ bytes(summary.upload_bytes) }}
              </strong>
              <p class="mt-0.5 text-[11px] text-[var(--atelier-ink-muted)]">{{ t('traffic.uploadHint') }}</p>
            </div>
            <div class="hidden h-9 w-px bg-[var(--atelier-border)] sm:block" aria-hidden="true" />
            <div class="min-w-[115px]">
              <p class="text-xs text-[var(--atelier-ink-muted)]">{{ t('traffic.download') }}</p>
              <strong :title="`${summary.download_bytes} B`" class="mt-1 block font-mono text-lg font-semibold tabular-nums text-[var(--atelier-ink)]">
                {{ bytes(summary.download_bytes) }}
              </strong>
              <p class="mt-0.5 text-[11px] text-[var(--atelier-ink-muted)]">{{ t('traffic.downloadHint') }}</p>
            </div>
            <div class="hidden h-9 w-px bg-[var(--atelier-border)] sm:block" aria-hidden="true" />
            <div class="min-w-[115px]">
              <p class="text-xs text-[var(--atelier-ink-muted)]">{{ t('traffic.requests') }}</p>
              <strong :title="String(summary.requests)" class="mt-1 block font-mono text-lg font-semibold tabular-nums text-[var(--atelier-ink)]">
                {{ summary.requests.toLocaleString() }}
              </strong>
              <p class="mt-0.5 text-[11px] text-[var(--atelier-ink-muted)]">{{ t('traffic.requestsHint') }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Asymmetric Trend (8 cols) + Composition (4 cols) -->
      <div v-if="summary.requests" class="grid grid-cols-1 gap-4 lg:grid-cols-12">
        <article class="atelier-surface flex flex-col lg:col-span-8">
          <header class="flex items-center justify-between gap-2 border-b border-[var(--atelier-border)] px-5 py-3.5">
            <div>
              <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">{{ t('traffic.trend') }}</h3>
              <p class="text-xs text-[var(--atelier-ink-muted)]">{{ t(`traffic.trendScopes.${trendBucket}`) }}</p>
            </div>
            <span class="rounded-md bg-[var(--atelier-subtle)] px-2 py-0.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ t(`traffic.periods.${summary.period}`) }}
            </span>
          </header>
          <div class="h-[260px] p-5" role="img" :aria-label="t('traffic.trend')">
            <Bar :data="chartData" :options="chartOptions" />
          </div>
        </article>

        <article class="atelier-surface flex flex-col lg:col-span-4">
          <header class="border-b border-[var(--atelier-border)] px-5 py-3.5">
            <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">{{ t('traffic.composition') }}</h3>
          </header>
          <div class="flex flex-1 flex-col justify-between gap-4 p-5">
            <div class="relative mx-auto h-40 w-40" role="img" :aria-label="t('traffic.composition')">
              <Doughnut :data="directionData" :options="directionOptions" />
              <div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center" aria-hidden="true">
                <strong class="font-mono text-sm font-semibold tabular-nums text-[var(--atelier-ink)]">{{ bytes(summary.total_bytes) }}</strong>
                <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ t('traffic.total') }}</span>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-2 text-xs">
              <div class="flex items-center justify-between rounded-lg bg-[var(--atelier-subtle)] px-3 py-2">
                <span class="flex items-center gap-1.5 text-[var(--atelier-ink-secondary)]">
                  <span class="h-2 w-2 rounded-full" :style="{ background: palette[0] }" />
                  {{ t('traffic.upload') }}
                </span>
                <strong class="font-mono tabular-nums text-[var(--atelier-ink)]">{{ share(summary.upload_bytes) }}</strong>
              </div>
              <div class="flex items-center justify-between rounded-lg bg-[var(--atelier-subtle)] px-3 py-2">
                <span class="flex items-center gap-1.5 text-[var(--atelier-ink-secondary)]">
                  <span class="h-2 w-2 rounded-full" :style="{ background: palette[2] }" />
                  {{ t('traffic.download') }}
                </span>
                <strong class="font-mono tabular-nums text-[var(--atelier-ink)]">{{ share(summary.download_bytes) }}</strong>
              </div>
            </div>

            <dl class="grid grid-cols-2 gap-3 border-t border-[var(--atelier-border)] pt-3 text-xs">
              <div>
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('traffic.average') }}</dt>
                <dd class="mt-0.5 font-mono font-semibold tabular-nums text-[var(--atelier-ink)]">
                  {{ bytes(summary.requests ? Math.round(summary.total_bytes / summary.requests) : 0) }}
                </dd>
              </div>
              <div>
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('traffic.peakBucket') }}</dt>
                <dd class="mt-0.5 font-mono font-semibold tabular-nums text-[var(--atelier-ink)]">
                  {{ peakBucket ? bytes(peakBucket.total_bytes) : '—' }}
                  <span v-if="peakBucket" class="block text-[11px] font-normal text-[var(--atelier-ink-muted)]">{{ peakBucket.bucket }}</span>
                </dd>
              </div>
            </dl>
          </div>
        </article>
      </div>

      <div v-else class="atelier-surface flex flex-col items-center justify-center gap-2 py-12 text-center">
        <Icon name="chart" size="lg" class="text-[var(--atelier-ink-muted)]" />
        <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">{{ t('traffic.empty') }}</h3>
        <p class="text-xs text-[var(--atelier-ink-muted)]">{{ t('traffic.emptyHint') }}</p>
      </div>

      <details v-if="summary.requests" class="atelier-surface px-5 py-3.5">
        <summary class="flex cursor-pointer items-center gap-2 text-xs font-medium text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)]">
          <Icon name="chart" size="sm" />
          {{ t('traffic.trendDetails') }}
        </summary>
        <div class="mt-3 overflow-x-auto">
          <table class="atelier-table text-xs">
            <caption class="sr-only">{{ t('traffic.trend') }} · {{ t('traffic.upload') }} / {{ t('traffic.download') }}</caption>
            <thead>
              <tr>
                <th scope="col">{{ t('traffic.bucket') }}</th>
                <th scope="col" class="!text-right">{{ t('traffic.upload') }}</th>
                <th scope="col" class="!text-right">{{ t('traffic.download') }}</th>
                <th scope="col" class="!text-right">{{ t('traffic.total') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="point in summary.trend" :key="point.bucket">
                <td class="font-mono">{{ point.bucket }}</td>
                <td v-for="direction in directions" :key="direction" :title="`${point[direction]} B`" class="text-right font-mono tabular-nums">
                  {{ bytes(point[direction]) }}
                </td>
                <td class="text-right font-mono font-semibold tabular-nums">{{ bytes(point.total_bytes) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </details>
    </template>

    <!-- Records Table -->
    <section class="atelier-surface overflow-hidden" :aria-label="t('traffic.records')">
      <header class="flex items-center justify-between gap-3 border-b border-[var(--atelier-border)] px-5 py-3.5">
        <div>
          <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">{{ t('traffic.records') }}</h3>
          <p class="text-xs text-[var(--atelier-ink-muted)]">{{ t('traffic.recordsHint') }}</p>
        </div>
        <span v-if="details" class="rounded-md bg-[var(--atelier-subtle)] px-2.5 py-1 font-mono text-xs tabular-nums text-[var(--atelier-ink-secondary)]">
          {{ t('traffic.page', { page }) }}
        </span>
      </header>

      <div class="p-5">
        <p v-if="detailsError" role="alert" class="mb-3 rounded-lg bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]">
          {{ detailsError }}
          <button type="button" class="ml-2 underline" @click="loadDetails(requestedPage)">
            {{ t('common.refresh') }}
          </button>
        </p>
        <p v-if="detailsLoading" role="status" class="py-10 text-center text-xs text-[var(--atelier-ink-muted)]">
          {{ t('traffic.loadingRecords') }}
        </p>

        <template v-if="details && !detailsLoading">
          <div class="overflow-x-auto">
            <div v-if="mobile" class="space-y-3">
              <dl
                v-for="item in details.items"
                :key="item.event_id"
                class="space-y-2 rounded-xl border border-[var(--atelier-border)] p-3.5 text-xs"
              >
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('usage.time') }}</dt>
                  <dd class="font-mono text-[var(--atelier-ink)]">{{ displayTime(item.started_at) }}</dd>
                </div>
                <div v-if="admin" class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('traffic.user') }}</dt>
                  <dd class="font-mono">#{{ item.user_id }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">API Key</dt>
                  <dd class="inline-flex items-center gap-1 font-mono"><Icon name="key" size="xs" />#{{ item.api_key_id }}</dd>
                </div>
                <div class="flex flex-col gap-1">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('traffic.endpoint') }}</dt>
                  <dd>
                    <div class="flex items-center gap-1.5">
                      <span class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-[11px] font-semibold">{{ item.method }}</span>
                      <code class="break-all font-mono text-xs">{{ item.endpoint }}</code>
                    </div>
                    <span class="mt-0.5 block truncate font-mono text-[11px] text-[var(--atelier-ink-muted)]" :title="item.request_id || item.event_id">
                      {{ item.request_id || item.event_id }}
                    </span>
                  </dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('traffic.upload') }}</dt>
                  <dd :title="`${item.upload_bytes} B`" class="font-mono tabular-nums">
                    {{ bytes(item.upload_bytes) }}
                    <small v-if="!item.request_complete" class="ml-1 text-[var(--atelier-warn)]">{{ t('traffic.partialRead') }}</small>
                  </dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('traffic.download') }}</dt>
                  <dd :title="`${item.download_bytes} B`" class="font-mono tabular-nums">{{ bytes(item.download_bytes) }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('traffic.result') }}</dt>
                  <dd class="flex items-center gap-1.5">
                    <UiStatusDot :tone="item.outcome === 'completed' ? 'ok' : item.outcome === 'canceled' ? 'warn' : 'err'" />
                    <span class="font-mono">{{ item.status_code ? 'HTTP ' + item.status_code : '—' }}</span>
                    <small class="text-[var(--atelier-ink-muted)]">{{ t(`traffic.outcomes.${item.outcome}`) }}</small>
                  </dd>
                </div>
              </dl>
              <div v-if="!details.items.length" class="py-10 text-center text-xs text-[var(--atelier-ink-muted)]">
                {{ t('traffic.empty') }}
              </div>
            </div>

            <table v-else class="atelier-table text-xs">
              <caption class="sr-only">{{ t('traffic.records') }}</caption>
              <thead>
                <tr>
                  <th scope="col">{{ t('usage.time') }}</th>
                  <th v-if="admin" scope="col">{{ t('traffic.user') }}</th>
                  <th scope="col">API Key</th>
                  <th scope="col">{{ t('traffic.endpoint') }}</th>
                  <th scope="col" class="!text-right">{{ t('traffic.upload') }}</th>
                  <th scope="col" class="!text-right">{{ t('traffic.download') }}</th>
                  <th scope="col">{{ t('traffic.result') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in details.items" :key="item.event_id">
                  <td class="whitespace-nowrap font-mono tabular-nums text-[var(--atelier-ink-secondary)]">
                    {{ displayTime(item.started_at) }}
                  </td>
                  <td v-if="admin" class="font-mono">#{{ item.user_id }}</td>
                  <td>
                    <span class="inline-flex items-center gap-1 rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-[11px] text-[var(--atelier-ink)]">
                      <Icon name="key" size="xs" />#{{ item.api_key_id }}
                    </span>
                  </td>
                  <td>
                    <div class="flex items-center gap-1.5">
                      <span class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-[11px] font-semibold text-[var(--atelier-ink)]">
                        {{ item.method }}
                      </span>
                      <code class="font-mono text-xs text-[var(--atelier-ink)]">{{ item.endpoint }}</code>
                    </div>
                    <span class="mt-0.5 block max-w-[260px] truncate font-mono text-[11px] text-[var(--atelier-ink-muted)]" :title="item.request_id || item.event_id">
                      {{ item.request_id || item.event_id }}
                    </span>
                  </td>
                  <td :title="`${item.upload_bytes} B`" class="text-right font-mono tabular-nums">
                    {{ bytes(item.upload_bytes) }}
                    <span v-if="!item.request_complete" class="block text-[11px] text-[var(--atelier-warn)]">{{ t('traffic.partialRead') }}</span>
                  </td>
                  <td :title="`${item.download_bytes} B`" class="text-right font-mono tabular-nums">
                    {{ bytes(item.download_bytes) }}
                  </td>
                  <td>
                    <div class="flex items-center gap-1.5">
                      <UiStatusDot :tone="item.outcome === 'completed' ? 'ok' : item.outcome === 'canceled' ? 'warn' : 'err'" />
                      <span class="font-mono font-medium">{{ item.status_code ? 'HTTP ' + item.status_code : '—' }}</span>
                    </div>
                    <span class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ t(`traffic.outcomes.${item.outcome}`) }}</span>
                  </td>
                </tr>
                <tr v-if="!details.items.length">
                  <td :colspan="admin ? 7 : 6" class="py-10 text-center text-[var(--atelier-ink-muted)]">
                    {{ t('traffic.empty') }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <footer class="mt-4 flex items-center justify-between border-t border-[var(--atelier-border)] pt-3.5 text-xs text-[var(--atelier-ink-muted)]">
            <span class="font-mono tabular-nums">{{ t('traffic.page', { page }) }}</span>
            <div class="flex items-center gap-2">
              <UiButton variant="secondary" size="sm" type="button" :disabled="page <= 1 || detailsLoading" @click="previousPage">
                {{ t('traffic.previous') }}
              </UiButton>
              <UiButton variant="secondary" size="sm" type="button" :disabled="!details.next_cursor || detailsLoading" @click="nextPage">
                {{ t('traffic.next') }}
              </UiButton>
            </div>
          </footer>
        </template>
      </div>
    </section>

    <!-- Measurement Notes -->
    <details class="atelier-surface px-5 py-3.5 text-xs text-[var(--atelier-ink-secondary)]">
      <summary class="flex cursor-pointer items-center gap-2 font-medium text-[var(--atelier-ink)]">
        <Icon name="infoCircle" size="sm" class="text-[var(--atelier-ink-muted)]" />
        <span>{{ t('traffic.scopeTitle') }}</span>
        <span class="font-normal text-[var(--atelier-ink-muted)]">{{ t('traffic.scopeShort') }}</span>
      </summary>
      <div class="mt-3 space-y-1.5 border-t border-[var(--atelier-border)] pt-3 leading-relaxed">
        <p>{{ t('traffic.scope') }}</p>
        <p v-if="summary">
          {{ summary.recorded_since ? t('traffic.since', { time: displayTime(summary.recorded_since) }) : t('traffic.notRecorded') }}
          {{ t('traffic.history') }}
        </p>
      </div>
    </details>

    <!-- Admin Node Health -->
    <details
      v-if="admin"
      class="atelier-surface px-5 py-3.5 text-xs"
      :class="healthError || (health && health.status !== 'ok') ? 'border-[var(--atelier-warn)]/40 bg-[var(--atelier-warn-bg)]' : ''"
      :open="healthError || health?.status === 'degraded' || health?.status === 'delayed'"
      data-testid="traffic-health"
    >
      <summary class="cursor-pointer font-medium text-[var(--atelier-ink)]">
        {{ t('traffic.nodeHealth') }} · {{ health ? t(`traffic.healthStatuses.${health.status}`) : t('common.loading') }}
      </summary>
      <div v-if="health" class="mt-3 space-y-1.5 text-[var(--atelier-ink-secondary)]">
        <p>{{ t('traffic.queueSummary', { queued: health.queued_events ?? 0, capacity: health.queue_capacity ?? 0, processing: health.processing_events ?? 0, dropped: health.dropped_events ?? 0 }) }}</p>
        <p>{{ t('traffic.healthSummary', { pending: health.pending_events, age: health.oldest_pending_age_seconds, quarantined: health.quarantined_events, lost: health.persistence_failures }) }}</p>
        <p v-if="health.storage_available === false" role="alert" class="text-[var(--atelier-err)]">{{ t('traffic.storageUnavailable') }}</p>
        <p class="text-[var(--atelier-ink-muted)]">{{ t('traffic.nodeHealthScope') }} · {{ displayTime(health.checked_at) }}</p>
      </div>
      <p v-if="healthError" role="alert" class="mt-2 text-[var(--atelier-err)]">{{ t('traffic.healthFailed') }}</p>
    </details>
  </section>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
import { computed, onUnmounted, ref, watch } from 'vue'
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, CategoryScale, LinearScale, BarElement, ArcElement, Tooltip, Legend, type ChartOptions } from 'chart.js'
import { Bar, Doughnut } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import UsageFilters from '@/components/usage/TrafficIdentityFilters.vue'
import trafficZh from '@/i18n/locales/zh/traffic'
import trafficEn from '@/i18n/locales/en/traffic'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { getTraffic, getTrafficKeys, getTrafficHealth, formatTrafficBytes as bytes, type TrafficPeriod, type TrafficReport, type TrafficKeyOption, type TrafficDeliveryHealth } from '@/api/traffic'
import { UiButton, UiSelect, UiStatusDot, type UiSelectOption } from '@/ui'

const props = withDefaults(defineProps<{ admin?: boolean; apiKeyOptions?: SelectOption[] }>(), { admin: false, apiKeyOptions: () => [] })
const { t } = useI18n({ useScope: 'local', messages: { zh: { traffic: trafficZh }, en: { traffic: trafficEn } } })
ChartJS.register(CategoryScale, LinearScale, BarElement, ArcElement, Tooltip, Legend)
const palette = useChartPalette()
const chartTheme = useChartTheme()
const periods: TrafficPeriod[] = ['day', 'week', 'month', 'year']
const periodOptions = computed<UiSelectOption[]>(() =>
  periods.map(value => ({ value, label: t(`traffic.periods.${value}`) }))
)
const directions = ['upload_bytes', 'download_bytes'] as const
const period = ref<TrafficPeriod>('month'), date = ref(''), page = ref(1)
const searchFilters = ref<{ user_id?: number; api_key_id?: number | null }>({ api_key_id: null })
const searchResetKey = ref(0)
const filters = ref<{ user_id?: number; api_key_id?: number }>({})
const summary = ref<TrafficReport | null>(null), details = ref<TrafficReport | null>(null)
const report = computed(() => details.value || summary.value)
const summaryLoading = ref(false), detailsLoading = ref(false), error = ref(''), detailsError = ref('')
const loading = computed(() => summaryLoading.value || detailsLoading.value)
let cursors: Array<string | undefined> = [undefined]
let queryDate: string | undefined
const requestedPage = ref(1)
const health = ref<TrafficDeliveryHealth | null>(null), healthError = ref(false)
const ownKeyResults = ref<TrafficKeyOption[]>([]), ownKeyQuery = ref(''), ownKeyLoading = ref(false), ownKeyError = ref(false), ownKeyLabel = ref('')
let keyController: AbortController | undefined
let controller: AbortController | undefined
let detailsController: AbortController | undefined

function handlePeriodChange(val: string | number | boolean | null) {
  const next = String(val ?? 'month') as TrafficPeriod
  if (periods.includes(next)) period.value = next
}

const ownKeyOptions = computed<SelectOption[]>(() => {
  const options = new Map<string | number | boolean | null, SelectOption>()
  if (!ownKeyQuery.value) for (const option of props.apiKeyOptions) options.set(option.value, option)
  for (const key of ownKeyResults.value) options.set(key.id, { value: key.id, label: `${key.name || `#${key.id}`}${key.deleted ? ` (${t('traffic.deletedKey')})` : ''}` })
  options.delete(null)
  return [{ value: null, label: t('usage.allApiKeys') }, ...options.values()]
})
const selectedKeyLabel = computed(() => ownKeyLabel.value || ownKeyOptions.value.find(option => option.value === searchFilters.value.api_key_id)?.label || `#${searchFilters.value.api_key_id}`)
const searchAdminKeys = (userId: number | undefined, query: string) => getTrafficKeys(true, userId, query)
async function searchOwnKeys(query = '') {
  keyController?.abort()
  const current = new AbortController(); keyController = current
  ownKeyQuery.value = query; ownKeyLoading.value = true; ownKeyError.value = false; ownKeyResults.value = []
  try {
    const result = await getTrafficKeys(false, undefined, query, current.signal)
    if (!current.signal.aborted) ownKeyResults.value = result
  } catch {
    if (!current.signal.aborted) ownKeyError.value = true
  } finally { if (!current.signal.aborted) ownKeyLoading.value = false }
}
function selectOwnKey(_value: unknown, option: SelectOption | null) { ownKeyLabel.value = option?.label || ''; applyFilters() }
function onOwnKeyOpen(event: Event) {
  if (!(event.target instanceof Element) || !event.target.closest('#traffic-key')) return
  if (ownKeyQuery.value) void searchOwnKeys()
}
const siteDate = (value: string | Date) => {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: report.value?.timezone || 'UTC', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date(value))
  const get = (type: string) => parts.find(p => p.type === type)?.value || ''
  return `${get('year')}-${get('month')}-${get('day')}`
}
const currentDate = computed(() => report.value ? siteDate(new Date()) : '')
const displayTime = (value: string) => new Intl.DateTimeFormat(undefined, { timeZone: report.value?.timezone || 'UTC', dateStyle: 'short', timeStyle: 'medium' }).format(new Date(value))
const rangeLabel = computed(() => report.value ? `${siteDate(report.value.start)} — ${siteDate(new Date(new Date(report.value.end).getTime() - 1))}` : '')
const peakBucket = computed(() => summary.value?.trend.reduce<(TrafficReport['trend'][number]) | null>((peak, point) => !peak || point.total_bytes > peak.total_bytes ? point : peak, null))
const share = (value: number) => summary.value?.total_bytes ? `${(value / summary.value.total_bytes * 100).toFixed(1)}%` : '0%'
const directionData = computed(() => ({ labels: [t('traffic.upload'), t('traffic.download')], datasets: [{ data: [summary.value?.upload_bytes ?? 0, summary.value?.download_bytes ?? 0], backgroundColor: [palette.value[0], palette.value[2]], borderWidth: 0, hoverOffset: 3 }] }))
const directionOptions = computed<ChartOptions<'doughnut'>>(() => ({ responsive: true, maintainAspectRatio: false, cutout: '78%', animation: false, plugins: { legend: { display: false }, tooltip: { ...chartTheme.value.tooltip, callbacks: { label: context => `${context.label}: ${bytes(Number(context.raw))}` } } } }))
const trendBucket = computed(() => summary.value?.period === 'day' ? 'hour' : summary.value?.period === 'year' ? 'month' : 'day')
const chartData = computed(() => ({
  labels: summary.value?.trend.map(point => point.bucket) || [],
  datasets: [
    { label: t('traffic.upload'), data: summary.value?.trend.map(point => point.upload_bytes) || [], backgroundColor: palette.value[0], borderColor: chartTheme.value.pointBorder, borderWidth: 1.5, borderRadius: 4 },
    { label: t('traffic.download'), data: summary.value?.trend.map(point => point.download_bytes) || [], backgroundColor: palette.value[2], borderColor: chartTheme.value.pointBorder, borderWidth: 1.5, borderRadius: 4 }
  ]
}))
const chartOptions = computed<ChartOptions<'bar'>>(() => ({
  responsive: true, maintainAspectRatio: false, animation: false,
  plugins: {
    legend: { position: 'bottom', labels: { color: chartTheme.value.text, usePointStyle: true, pointStyle: 'circle', boxWidth: 8, padding: 14, font: { size: 12 } } },
    tooltip: { ...chartTheme.value.tooltip, callbacks: { label: context => `${context.dataset.label}: ${bytes(Number(context.raw))}` } },
  },
  scales: {
    x: { stacked: true, border: { display: false }, grid: { display: false }, ticks: { color: chartTheme.value.text, maxRotation: 0, maxTicksLimit: 7, font: { size: 12 } } },
    y: { stacked: true, beginAtZero: true, border: { display: false }, grid: { color: chartTheme.value.grid }, ticks: { color: chartTheme.value.text, font: { size: 12 }, callback: value => bytes(Number(value)) } },
  }
}))
function setDate(event: Event) { date.value = (event.target as HTMLInputElement).value }
function applyFilters() {
  const user = searchFilters.value.user_id, key = searchFilters.value.api_key_id
  filters.value = { ...(props.admin && user ? { user_id: user } : {}), ...(key ? { api_key_id: key } : {}) }
}
function reset() {
  searchFilters.value = { api_key_id: null }; searchResetKey.value++; date.value = ''; period.value = 'month'; filters.value = {}; ownKeyLabel.value = ''
  if (!props.admin) void searchOwnKeys()
}
function shift(direction: number) {
  if (!report.value) return
  const anchor = new Date(`${siteDate(report.value.start)}T00:00:00Z`)
  if (period.value === 'year') anchor.setUTCFullYear(anchor.getUTCFullYear() + direction)
  else if (period.value === 'month') anchor.setUTCMonth(anchor.getUTCMonth() + direction)
  else anchor.setUTCDate(anchor.getUTCDate() + direction * (period.value === 'week' ? 7 : 1))
  date.value = anchor.toISOString().slice(0, 10)
}
function nextPage() {
  if (!details.value?.next_cursor || detailsLoading.value) return
  cursors[page.value] = details.value.next_cursor
  cursors.length = page.value + 1
  void loadDetails(page.value + 1)
}
function previousPage() { if (page.value > 1 && !detailsLoading.value) void loadDetails(page.value - 1) }
async function loadDetails(targetPage: number) {
  detailsController?.abort()
  const current = new AbortController(); detailsController = current
  requestedPage.value = targetPage
  detailsLoading.value = true; detailsError.value = ''; details.value = null
  try {
    const result = await getTraffic(props.admin, { period: period.value, date: queryDate, view: 'details', cursor: cursors[targetPage - 1], page_size: 20, ...filters.value }, current.signal)
    if (!current.signal.aborted) {
      details.value = result; page.value = targetPage
      queryDate = siteDate(result.start)
      if (summary.value && summary.value.start !== result.start && controller) void loadSummary(controller)
    }
  } catch {
    if (!current.signal.aborted) detailsError.value = t('traffic.recordsFailed')
  } finally { if (!current.signal.aborted) detailsLoading.value = false }
}
async function load() {
  controller?.abort(); detailsController?.abort()
  const current = new AbortController(); controller = current
  cursors = [undefined]; page.value = 1; queryDate = date.value || undefined
  summaryLoading.value = true; error.value = ''; summary.value = null
  if (props.admin) void loadHealth(current)
  void loadDetails(1)
  void loadSummary(current)
}
async function loadSummary(current: AbortController) {
  summaryLoading.value = true; summary.value = null; error.value = ''
  try {
    let result = await getTraffic(props.admin, { period: period.value, date: queryDate, view: 'summary', page_size: 20, ...filters.value }, current.signal)
    if (current.signal.aborted) return
    if (details.value && details.value.start !== result.start) {
      result = await getTraffic(props.admin, { period: period.value, date: queryDate, view: 'summary', page_size: 20, ...filters.value }, current.signal)
    }
    if (!current.signal.aborted) summary.value = result
  } catch {
    if (!current.signal.aborted) error.value = t('traffic.failed')
  } finally { if (!current.signal.aborted) summaryLoading.value = false }
}
async function loadHealth(current: AbortController) {
  try {
    const result = await getTrafficHealth(current.signal)
    if (!current.signal.aborted) { health.value = result; healthError.value = false }
  } catch {
    if (!current.signal.aborted) { health.value = null; healthError.value = true }
  }
}
watch([period, date, filters], () => { void load() }, { deep: true })
void load()
if (!props.admin) void searchOwnKeys()
onUnmounted(() => { controller?.abort(); detailsController?.abort(); keyController?.abort() })
</script>
