<template>
  <div class="mx-auto max-w-[1440px] space-y-6 pb-10">
    <!-- Top Header & Controls Surface -->
    <section class="atelier-surface overflow-hidden">
      <header class="flex flex-wrap items-start justify-between gap-4 border-b border-[var(--atelier-border)] px-5 py-4 sm:px-6">
        <div class="min-w-0">
          <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
            {{ t('channelMonitorV2.title') }}
          </h1>
          <div class="mt-1.5 flex flex-wrap items-center gap-2 text-xs tabular-nums text-[var(--atelier-ink-secondary)]">
            <UiStatusDot :tone="loading || refreshing ? 'neutral' : 'ok'" />
            <span v-if="refreshing" class="inline-flex items-center gap-1.5 text-[var(--atelier-ink)]">
              <LoadingSpinner size="sm" />
              {{ t('channelMonitorV2.updating') }}
            </span>
            <span v-else-if="snapshot?.coverage.data_through" class="font-mono">
              {{ t('channelMonitorV2.updatedTo', { time: formatTime(snapshot.coverage.data_through) }) }}
            </span>
            <span v-else class="text-[var(--atelier-ink-muted)]">{{ t('common.loading') }}</span>

            <span
              v-if="snapshot && !snapshot.coverage.coverage_complete && !bootstrapActive"
              class="rounded-full bg-[var(--atelier-warn-bg)] px-2 py-0.5 text-[11px] font-medium text-[var(--atelier-warn)]"
            >
              {{ t('channelMonitorV2.partialCoverage') }}
            </span>
            <span
              v-if="bootstrapActive"
              class="inline-flex items-center gap-1 rounded-full bg-[var(--atelier-info-bg)] px-2 py-0.5 font-mono text-[11px] font-medium tabular-nums text-[var(--atelier-info)]"
            >
              <LoadingSpinner size="sm" />
              {{ t('channelMonitorV2.bootstrap.progress', { percent: bootstrapPercent }) }}
            </span>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <UiButton
            v-for="qg in qualityGroups"
            :key="qg.id"
            variant="secondary"
            size="sm"
            type="button"
            @click="qualityTarget = qg"
          >
            <Icon name="sparkles" size="xs" />
            {{ locale.startsWith('zh') ? `质量评测 · ${qg.name}` : `Quality · ${qg.name}` }}
          </UiButton>
          <UiButton
            variant="secondary"
            size="sm"
            type="button"
            :title="t('common.refresh')"
            :disabled="loading"
            @click="reload(false)"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </UiButton>
        </div>
      </header>

      <!-- First-upgrade silent backfill -->
      <div
        v-if="bootstrapActive"
        class="border-b border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/60 px-5 py-3.5 sm:px-6"
        role="status"
        aria-live="polite"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 flex-1">
            <p class="text-sm font-semibold text-[var(--atelier-ink)]">
              {{ t('channelMonitorV2.bootstrap.title') }}
            </p>
            <p class="mt-0.5 text-xs text-[var(--atelier-ink-secondary)]">
              {{ t('channelMonitorV2.bootstrap.description') }}
            </p>
          </div>
          <span class="shrink-0 font-mono text-xs font-semibold tabular-nums text-[var(--atelier-ink)]">
            {{ t('channelMonitorV2.bootstrap.progress', { percent: bootstrapPercent }) }}
          </span>
        </div>
        <div class="mt-2.5">
          <UiProgressBar :value="bootstrapPercent" tone="ink" />
        </div>
      </div>

      <!-- Compact Toolbar: Range · Filters · GroupBy · View Controls -->
      <div class="flex flex-nowrap items-center gap-2 overflow-x-auto bg-[var(--atelier-subtle)]/35 px-4 py-3 sm:px-6">
        <UiSegmented
          :model-value="filter.range"
          :options="ranges"
          size="sm"
          :aria-label="t('channelMonitorV2.timeRange')"
          class="shrink-0"
          @update:model-value="(val) => setRange(val as MonitorRange)"
        />

        <span class="hidden h-5 w-px shrink-0 bg-[var(--atelier-border)] sm:block" aria-hidden="true" />

        <FilterMultiSelect
          v-model="filter.platforms"
          compact
          :label="t('channelMonitorV2.filters.platform')"
          :all-label="t('channelMonitorV2.filters.allPlatforms')"
          :options="platformOptions"
        />
        <FilterMultiSelect
          v-model="selectedGroupIds"
          compact
          :label="t('channelMonitorV2.filters.group')"
          :all-label="t('channelMonitorV2.filters.allGroups')"
          :options="groupOptions"
        />
        <FilterMultiSelect
          v-model="filter.models"
          compact
          :label="t('channelMonitorV2.filters.model')"
          :all-label="t('channelMonitorV2.filters.allModels')"
          :options="modelOptions"
        />
        <UiButton
          variant="ghost"
          size="sm"
          type="button"
          class="shrink-0"
          :disabled="!hasDimensionFilter"
          @click="clearDimensions"
        >
          {{ t('channelMonitorV2.clearFilters') }}
        </UiButton>

        <span class="hidden h-5 w-px shrink-0 bg-[var(--atelier-border)] md:block" aria-hidden="true" />

        <div class="w-36 shrink-0">
          <UiSelect
            :model-value="matrixGroupBy"
            :options="matrixGroupOptions"
            size="sm"
            :placeholder="t('channelMonitorV2.groupBy.label')"
            :aria-label="t('channelMonitorV2.groupBy.label')"
            @update:model-value="handleMatrixGroupByChange"
          />
        </div>

        <UiSegmented
          v-model="trendView"
          :options="trendViewOptions"
          size="sm"
          :aria-label="t('channelMonitorV2.trendView.label')"
          class="shrink-0"
        />

        <UiSegmented
          v-if="trendView === 'pulse'"
          v-model="healthMode"
          :options="healthModeOptions"
          size="sm"
          :aria-label="t('channelMonitorV2.healthMode.label')"
          class="shrink-0"
        />
      </div>
    </section>

    <!-- Overview KPI Row -->
    <section
      v-if="snapshot"
      class="grid grid-cols-2 gap-3 sm:grid-cols-3"
      :class="showThroughput ? 'xl:grid-cols-5' : 'xl:grid-cols-4'"
      :aria-label="t('channelMonitorV2.summaryAria')"
    >
      <MetricCell
        :label="t('channelMonitorV2.metrics.successRate')"
        :value="formatPercent(1 - snapshot.metrics.error_rate)"
        :detail="t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(snapshot.metrics.error_rate) })"
        :state="snapshot.health.error_rate"
      />
      <MetricCell
        :label="t('channelMonitorV2.metrics.ttftP50')"
        :value="formatMs(snapshot.metrics.ttft.p50_ms)"
        :detail="latencyKpiSecondary(snapshot.metrics.ttft)"
        :title="latencyDetail(snapshot.metrics.ttft)"
        :state="ttftCellState(snapshot.health.ttft, snapshot.metrics.ttft)"
      />
      <MetricCell
        v-if="showThroughput"
        :label="t('channelMonitorV2.metrics.tps')"
        :value="formatTps(snapshot.metrics.tpm)"
        :detail="t('channelMonitorV2.metrics.tpsDetail')"
        :title="exactTps(snapshot.metrics.tpm)"
      />
      <MetricCell
        :label="t('channelMonitorV2.metrics.cacheRate')"
        :value="formatPercent(snapshot.metrics.cache_rate)"
        :detail="t('channelMonitorV2.metrics.cacheDetail')"
        :state="snapshot.health.cache || snapshot.health.overall"
      />
      <MetricCell
        v-if="showThroughput"
        :label="t('channelMonitorV2.metrics.rpm')"
        :value="formatRate(snapshot.metrics.rpm)"
        :detail="t('channelMonitorV2.metrics.rpmDetail')"
        :title="exactRate(snapshot.metrics.rpm)"
      />
    </section>
    <section
      v-else-if="loading"
      class="grid grid-cols-2 gap-3 sm:grid-cols-3"
      :class="showThroughput ? 'xl:grid-cols-5' : 'xl:grid-cols-4'"
      aria-hidden="true"
    >
      <div
        v-for="i in (showThroughput ? 5 : 4)"
        :key="i"
        class="atelier-surface-subtle h-[6.5rem] animate-pulse"
      />
    </section>

    <!-- Pulse Matrix or Trend Chart -->
    <div class="relative min-h-[320px]">
      <MonitorTrendChart
        v-if="trendView === 'line'"
        :trend="snapshot?.trend || []"
        :coverage="snapshot?.coverage || null"
        :loading="loading && !snapshot"
      />
      <RelayPulseMatrix
        v-else-if="matrix"
        :rows="matrixRows"
        :coverage="matrix.coverage"
        :health-mode="healthMode"
        :show-throughput="showThroughput"
      />
      <div
        v-else-if="loading"
        class="atelier-surface flex min-h-[320px] items-center justify-center text-sm text-[var(--atelier-ink-muted)]"
      >
        <span class="animate-pulse">{{ t('common.loading') }}</span>
      </div>
    </div>

    <!-- Detail Tabs: Models / Errors / Users -->
    <section class="atelier-surface flex min-h-0 flex-col overflow-hidden">
      <div class="px-5 pt-2 sm:px-6">
        <UiTabs
          :model-value="activeTab"
          :tabs="tabs"
          :aria-label="t('channelMonitorV2.tabs.aria')"
          @update:model-value="(val) => { activeTab = val as Tab }"
        />
      </div>

      <div class="monitor-records min-h-0 max-h-[min(52vh,520px)] overflow-auto p-5 sm:px-6">
        <!-- Tab: Models -->
        <div v-if="activeTab === 'models'">
          <div v-if="mobile" class="space-y-3">
            <dl
              v-for="row in modelRows"
              :key="`${row.platform}:${row.model}`"
              class="cursor-pointer space-y-2 rounded-xl border border-[var(--atelier-border)] p-3.5 text-xs transition-colors hover:border-[var(--atelier-border-strong)]"
              @click="drillModel(row)"
            >
              <div class="flex items-start justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.table.platformModel') }}</dt>
                <dd class="text-right">
                  <div class="flex items-center justify-end gap-2">
                    <span :class="statusDot(row.health)" aria-hidden="true" />
                    <div>
                      <span class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ row.platform }}</span>
                      <strong class="font-mono font-semibold text-[var(--atelier-ink)]">
                        {{ row.model === '__other__' ? t('channelMonitorV2.otherModels') : row.model }}
                      </strong>
                    </div>
                  </div>
                </dd>
              </div>
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.successRate') }}</dt>
                <dd class="text-right font-mono tabular-nums">
                  <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatPercent(1 - row.metrics.error_rate) }}</span>
                  <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(row.metrics.error_rate) }) }}</span>
                </dd>
              </div>
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.ttftP50') }}</dt>
                <dd class="text-right font-mono tabular-nums">
                  <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatMs(row.metrics.ttft.p50_ms) }}</span>
                  <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ latencyDetail(row.metrics.ttft) }}</span>
                </dd>
              </div>
              <div v-if="showThroughput" class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.tps') }}</dt>
                <dd class="font-mono tabular-nums" :title="exactTps(row.metrics.tpm)">{{ formatTps(row.metrics.tpm) }}</dd>
              </div>
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.cacheRate') }}</dt>
                <dd class="font-mono tabular-nums">{{ formatPercent(row.metrics.cache_rate) }}</dd>
              </div>
              <div v-if="showThroughput" class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.rpm') }}</dt>
                <dd class="font-mono tabular-nums">{{ formatRate(row.metrics.rpm) }}</dd>
              </div>
            </dl>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="atelier-table min-w-[720px] text-xs tabular-nums">
              <thead>
                <tr>
                  <th>{{ t('channelMonitorV2.table.platformModel') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.successRate') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.ttftP50') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.tps') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.cacheRate') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.rpm') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in modelRows"
                  :key="`${row.platform}:${row.model}`"
                  class="cursor-pointer"
                  @click="drillModel(row)"
                >
                  <td>
                    <div class="flex items-center gap-2.5">
                      <span :class="statusDot(row.health)" aria-hidden="true" />
                      <BrandModelIcon v-if="row.model !== '__other__'" :model="row.model" size="sm" />
                      <div>
                        <span class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ row.platform }}</span>
                        <strong class="font-mono font-semibold text-[var(--atelier-ink)]">
                          {{ row.model === '__other__' ? t('channelMonitorV2.otherModels') : row.model }}
                        </strong>
                      </div>
                    </div>
                  </td>
                  <td class="font-mono">
                    <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatPercent(1 - row.metrics.error_rate) }}</span>
                    <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(row.metrics.error_rate) }) }}</span>
                  </td>
                  <td class="font-mono">
                    <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatMs(row.metrics.ttft.p50_ms) }}</span>
                    <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ latencyDetail(row.metrics.ttft) }}</span>
                  </td>
                  <td v-if="showThroughput" class="font-mono" :title="exactTps(row.metrics.tpm)">{{ formatTps(row.metrics.tpm) }}</td>
                  <td class="font-mono">{{ formatPercent(row.metrics.cache_rate) }}</td>
                  <td v-if="showThroughput" class="font-mono">{{ formatRate(row.metrics.rpm) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Tab: Errors -->
        <div v-else-if="activeTab === 'errors'" class="space-y-2.5">
          <div
            v-for="row in errorRows"
            :key="row.category"
            class="atelier-surface-subtle p-4 text-sm"
            :class="row.ignored ? 'opacity-60' : ''"
          >
            <button
              type="button"
              class="grid w-full grid-cols-[minmax(110px,220px)_1fr_auto_auto] items-center gap-3 text-left focus:outline-none"
              @click="toggleError(row.category)"
            >
              <span class="flex min-w-0 items-center gap-1.5 truncate text-xs font-semibold text-[var(--atelier-ink)]">
                <span class="truncate">{{ errorLabel(row.category) }}</span>
                <span
                  v-if="row.ignored"
                  class="shrink-0 rounded-full border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-1.5 py-0 text-[10px] font-medium text-[var(--atelier-ink-muted)]"
                >
                  {{ t('channelMonitorV2.ignored') }}
                </span>
              </span>
              <UiProgressBar :value="Math.max(2, row.rate * 100)" :tone="row.ignored ? 'ink' : 'err'" />
              <small class="w-14 text-right font-mono text-xs tabular-nums text-[var(--atelier-ink-secondary)]">
                {{ formatPercent(row.rate) }}
              </small>
              <Icon
                name="chevronDown"
                size="sm"
                :class="['text-[var(--atelier-ink-muted)] transition-transform', expandedErrors.has(row.category) ? 'rotate-180' : '']"
              />
            </button>

            <div v-if="expandedErrors.has(row.category)" class="mt-3 space-y-2 border-t border-[var(--atelier-border)] pt-3">
              <template v-if="isAdmin && (row.details || []).length">
                <div
                  v-for="(detail, index) in row.details || []"
                  :key="`${row.category}:${index}:${detail.message}`"
                  class="rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-3 py-2 text-xs tabular-nums"
                >
                  <div class="mb-1 flex flex-wrap items-center gap-2">
                    <span class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 text-[10px] font-medium text-[var(--atelier-ink-secondary)]">
                      {{ detail.platform || '-' }}
                    </span>
                    <span class="truncate font-mono font-medium text-[var(--atelier-ink)]">{{ detail.model || '-' }}</span>
                    <span v-if="detail.status_code" class="font-mono text-[var(--atelier-ink-muted)]">
                      {{ t('channelMonitorV2.errorDetail.http', { code: detail.status_code }) }}
                    </span>
                    <span v-if="detail.upstream_status_code" class="font-mono text-[var(--atelier-ink-muted)]">
                      {{ t('channelMonitorV2.errorDetail.upstream', { code: detail.upstream_status_code }) }}
                    </span>
                    <span class="ml-auto font-mono text-[var(--atelier-ink-muted)]">×{{ detail.count }}</span>
                  </div>
                  <p class="break-words leading-relaxed text-[var(--atelier-ink-secondary)]">
                    {{ detail.message || detail.error_type || t('channelMonitorV2.errorDetail.noMessage') }}
                  </p>
                </div>
              </template>
              <p v-else class="text-xs text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.errorDetail.empty') }}</p>
            </div>
          </div>
        </div>

        <!-- Tab: Users -->
        <div v-else>
          <div v-if="mobile" class="space-y-3">
            <dl
              v-for="row in userRows"
              :key="row.user_id || row.display_label"
              class="space-y-2 rounded-xl border border-[var(--atelier-border)] p-3.5 text-xs"
              :class="row.is_self ? 'border-[var(--atelier-ink)] bg-[var(--atelier-selected-row)]' : ''"
            >
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.table.rank') }}</dt>
                <dd><MonitorRankBadge :rank="row.rank" /></dd>
              </div>
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.table.user') }}</dt>
                <dd class="font-semibold text-[var(--atelier-ink)]">
                  {{ row.display_label }}
                  <span
                    v-if="row.is_self"
                    class="ml-1.5 rounded-full bg-[var(--atelier-ink)] px-1.5 py-0 text-[10px] font-medium text-[var(--atelier-ink-inverse)]"
                  >
                    {{ t('channelMonitorV2.currentUser') }}
                  </span>
                </dd>
              </div>
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.successRate') }}</dt>
                <dd class="text-right font-mono tabular-nums">
                  <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatPercent(1 - row.metrics.error_rate) }}</span>
                  <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(row.metrics.error_rate) }) }}</span>
                </dd>
              </div>
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.ttftP50') }}</dt>
                <dd class="text-right font-mono tabular-nums">
                  <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatMs(row.metrics.ttft.p50_ms) }}</span>
                  <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ latencyDetail(row.metrics.ttft) }}</span>
                </dd>
              </div>
              <div v-if="showThroughput" class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.tps') }}</dt>
                <dd class="font-mono tabular-nums" :title="exactTps(row.metrics.tpm)">{{ formatTps(row.metrics.tpm) }}</dd>
              </div>
              <div class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.cacheRate') }}</dt>
                <dd class="font-mono tabular-nums">{{ formatPercent(row.metrics.cache_rate) }}</dd>
              </div>
              <div v-if="showThroughput" class="flex justify-between gap-2">
                <dt class="text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.rpm') }}</dt>
                <dd class="font-mono tabular-nums">{{ formatRate(row.metrics.rpm) }}</dd>
              </div>
            </dl>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="atelier-table min-w-[640px] text-xs tabular-nums">
              <thead>
                <tr>
                  <th class="w-16">{{ t('channelMonitorV2.table.rank') }}</th>
                  <th>{{ t('channelMonitorV2.table.user') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.successRate') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.ttftP50') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.tps') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.cacheRate') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.rpm') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in userRows"
                  :key="row.user_id || row.display_label"
                  :data-selected="row.is_self ? 'true' : undefined"
                >
                  <td><MonitorRankBadge :rank="row.rank" /></td>
                  <td>
                    <strong class="font-semibold text-[var(--atelier-ink)]">
                      {{ row.display_label }}
                      <span
                        v-if="row.is_self"
                        class="ml-2 rounded-full bg-[var(--atelier-ink)] px-2 py-0.5 text-[10px] font-medium text-[var(--atelier-ink-inverse)]"
                      >
                        {{ t('channelMonitorV2.currentUser') }}
                      </span>
                    </strong>
                  </td>
                  <td class="font-mono">
                    <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatPercent(1 - row.metrics.error_rate) }}</span>
                    <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(row.metrics.error_rate) }) }}</span>
                  </td>
                  <td class="font-mono">
                    <span class="block font-semibold text-[var(--atelier-ink)]">{{ formatMs(row.metrics.ttft.p50_ms) }}</span>
                    <span class="text-[11px] text-[var(--atelier-ink-muted)]">{{ latencyDetail(row.metrics.ttft) }}</span>
                  </td>
                  <td v-if="showThroughput" class="font-mono" :title="exactTps(row.metrics.tpm)">{{ formatTps(row.metrics.tpm) }}</td>
                  <td class="font-mono">{{ formatPercent(row.metrics.cache_rate) }}</td>
                  <td v-if="showThroughput" class="font-mono">{{ formatRate(row.metrics.rpm) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="tabLoading" class="py-12 text-center text-sm text-[var(--atelier-ink-muted)]">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="activeRowsEmpty" class="py-12 text-center">
          <p class="text-sm font-semibold text-[var(--atelier-ink)]">
            {{ bootstrapActive ? t('channelMonitorV2.bootstrap.title') : t('channelMonitorV2.empty.title') }}
          </p>
          <p class="mt-1 text-xs text-[var(--atelier-ink-muted)]">
            {{ bootstrapActive ? t('channelMonitorV2.bootstrap.description') : t('channelMonitorV2.empty.description') }}
          </p>
        </div>
      </div>
    </section>

    <ModelQualityResults
      v-if="qualityTarget"
      :group-id="qualityTarget.id"
      :group-name="qualityTarget.name"
      @close="qualityTarget = null"
    />
  </div>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
import { useI18n } from 'vue-i18n'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ModelQualityResults from '@/components/usage/ModelQualityResults.vue'
import { BrandModelIcon, UiButton, UiProgressBar, UiSegmented, UiSelect, UiStatusDot, UiTabs } from '@/ui'
import FilterMultiSelect from '@/features/channel-monitor-v2/FilterMultiSelect.vue'
import MetricCell from '@/features/channel-monitor-v2/MetricCell.vue'
import MonitorRankBadge from '@/features/channel-monitor-v2/MonitorRankBadge.vue'
import MonitorTrendChart from '@/features/channel-monitor-v2/MonitorTrendChart.vue'
import RelayPulseMatrix from '@/features/channel-monitor-v2/RelayPulseMatrix.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isChannelMonitorThroughputHidden, isChannelMonitorUserRankingHidden } from '@/utils/featureFlags'
import * as api from '@/api/channelMonitorV2'
import type {
  HealthState,
  MonitorDimensions,
  MonitorErrorRow,
  MonitorFilter,
  MonitorHealth,
  MonitorMatrixGroupBy,
  MonitorMatrixResponse,
  MonitorModelRow,
  MonitorRange,
  MonitorSnapshot,
  MonitorUserRow,
} from '@/api/channelMonitorV2'
import {
  formatLatencyKpiSecondary,
  formatLatencyPrivacy,
  formatMonitorMs,
  formatMonitorPercent,
  formatMonitorThroughput,
  formatMonitorTokensPerSecond,
  tokensPerSecondFromTpm,
  healthScoreClass,
  monitorErrorCategoryLabel,
  ttftDisplayState,
} from '@/features/channel-monitor-v2/monitorFormat'

type Tab = 'models' | 'errors' | 'users'
type HealthMode = 'overall' | 'success' | 'ttft' | 'cache'
type TrendView = 'pulse' | 'line'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const { t, te, locale } = useI18n()
const isAdmin = computed(() => authStore.isAdmin)
/** Admins always see RPM/TPM; users honor the hide-throughput system setting. */
const showThroughput = computed(() => isAdmin.value || !isChannelMonitorThroughputHidden())
/** Admins always see ranking; users honor the hide-user-ranking system setting. */
const showUserRanking = computed(() => isAdmin.value || !isChannelMonitorUserRankingHidden())

const ranges = computed(() => [
  { value: '90m' as MonitorRange, label: t('channelMonitorV2.ranges.90m') },
  { value: '24h' as MonitorRange, label: t('channelMonitorV2.ranges.24h') },
  { value: '7d' as MonitorRange, label: t('channelMonitorV2.ranges.7d') },
  { value: '30d' as MonitorRange, label: t('channelMonitorV2.ranges.30d') },
])
const tabs = computed(() => {
  const items: Array<{ value: Tab; label: string }> = [
    { value: 'models', label: t('channelMonitorV2.tabs.models') },
    { value: 'errors', label: t('channelMonitorV2.tabs.errors') },
  ]
  if (showUserRanking.value) {
    items.push({ value: 'users', label: t('channelMonitorV2.tabs.users') })
  }
  return items
})
const matrixGroupOptions = computed(() => [
  { value: 'platform' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platform') },
  { value: 'platform_group' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformGroup') },
  { value: 'platform_model' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformModel') },
  { value: 'platform_group_model' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformGroupModel') },
])
const healthModeOptions = computed(() => [
  { value: 'overall' as HealthMode, label: t('channelMonitorV2.healthMode.overall') },
  { value: 'success' as HealthMode, label: t('channelMonitorV2.healthMode.success') },
  { value: 'ttft' as HealthMode, label: t('channelMonitorV2.healthMode.ttft') },
  { value: 'cache' as HealthMode, label: t('channelMonitorV2.healthMode.cache') },
])
const trendViewOptions = computed(() => [
  { value: 'pulse' as TrendView, label: t('channelMonitorV2.trendView.pulse') },
  { value: 'line' as TrendView, label: t('channelMonitorV2.trendView.line') },
])
function handleMatrixGroupByChange(val: string | number | boolean | null) {
  const next = String(val ?? 'platform') as MonitorMatrixGroupBy
  if (['platform', 'platform_group', 'platform_model', 'platform_group_model'].includes(next)) {
    matrixGroupBy.value = next
  }
}

const filter = ref<MonitorFilter>({
  range: parseRange(route.query.range),
  platforms: csv(route.query.platform),
  groupIds: csv(route.query.group).map(Number).filter(Boolean),
  models: csv(route.query.model),
})
const activeTab = ref<Tab>(parseTab(route.query.tab, showUserRanking.value))
const matrixGroupBy = ref<MonitorMatrixGroupBy>(parseMatrixGroupBy(route.query.group_by))
const healthMode = ref<HealthMode>(parseHealthMode(route.query.health_mode))
const trendView = ref<TrendView>(parseTrendView(route.query.trend_view))
const dimensions = ref<MonitorDimensions>({ platforms: [], groups: [], models: [] })
const snapshot = ref<MonitorSnapshot | null>(null)
const matrix = ref<MonitorMatrixResponse | null>(null)
const modelRows = ref<MonitorModelRow[]>([])
const errorRows = ref<MonitorErrorRow[]>([])
const userRows = ref<MonitorUserRow[]>([])
const loading = ref(false)
const tabLoading = ref(false)
const refreshing = ref(false)
const expandedErrors = ref(new Set<string>())
const qualityTarget = ref<{ id: number; name: string } | null>(null)
const qualityGroups = computed(() =>
  (dimensions.value.groups || [])
    .filter((item) => [72, 237].includes(Number(item.id)))
    .map((item) => ({ id: Number(item.id), name: item.name || `#${item.id}` }))
)
let controller: AbortController | null = null
let sequence = 0
let autoRefreshTimer: number | null = null

const hasDimensionFilter = computed(
  () => filter.value.platforms.length + filter.value.groupIds.length + filter.value.models.length > 0
)
// Full platform catalog (never pruned). Groups/models cascade by selected platforms
// so choosing a platform narrows the other pickers without collapsing platforms.
const platformOptions = computed(() =>
  (dimensions.value.platforms || []).map((item) => ({
    value: item.value,
    label: item.label,
  }))
)
const selectedPlatforms = computed(() => new Set(filter.value.platforms))
const groupOptions = computed(() =>
  (dimensions.value.groups || [])
    .filter(
      (item) =>
        selectedPlatforms.value.size === 0 ||
        !item.platform ||
        selectedPlatforms.value.has(item.platform),
    )
    .map((item) => ({
      value: String(item.id),
      label: item.platform ? `${item.platform} / ${item.name || `#${item.id}`}` : item.name || `#${item.id}`,
    }))
)
const modelOptions = computed(() =>
  (dimensions.value.models || [])
    .filter(
      (item) =>
        selectedPlatforms.value.size === 0 ||
        !item.platform ||
        selectedPlatforms.value.has(item.platform),
    )
    .map((item) => ({
      value: item.value,
      label:
        item.platform && !item.label.includes(item.platform)
          ? `${item.platform} / ${item.label}`
          : item.label,
    }))
)
const selectedGroupIds = computed({
  get: () => filter.value.groupIds.map(String),
  set: (value: string[]) => {
    filter.value.groupIds = value.map(Number).filter((id) => Number.isInteger(id) && id > 0)
  },
})
// Soft-prune group/model selections that fall outside the platform cascade.
// Do NOT wipe when options are temporarily empty (loading); only drop invalid ids.
watch(
  [groupOptions, modelOptions],
  () => {
    if (groupOptions.value.length > 0) {
      const allowed = new Set(groupOptions.value.map((item) => item.value))
      const next = filter.value.groupIds.filter((id) => allowed.has(String(id)))
      if (next.length !== filter.value.groupIds.length) {
        filter.value.groupIds = next
      }
    }
    if (modelOptions.value.length > 0) {
      const allowed = new Set(modelOptions.value.map((item) => item.value))
      const next = filter.value.models.filter((model) => allowed.has(model))
      if (next.length !== filter.value.models.length) {
        filter.value.models = next
      }
    }
  },
  { flush: 'post' },
)
const activeRowsEmpty = computed(() =>
  activeTab.value === 'models'
    ? modelRows.value.length === 0
    : activeTab.value === 'errors'
      ? errorRows.value.length === 0
      : userRows.value.length === 0
)
/** First-upgrade backfill toward 90m/24h/7d/30d; banner hides when backend omits bootstrap. */
const bootstrapActive = computed(() => Boolean(snapshot.value?.coverage?.bootstrap?.active))
const bootstrapPercent = computed(() => {
  const raw = snapshot.value?.coverage?.bootstrap?.progress_percent
  if (typeof raw !== 'number' || Number.isNaN(raw)) return 0
  return Math.min(100, Math.max(0, Math.round(raw)))
})
const matrixRows = computed(() => {
  const items = matrix.value?.items || []
  // platform_group views should only show real groups, never bare platform placeholders.
  if (matrixGroupBy.value === 'platform_group' || matrixGroupBy.value === 'platform_group_model') {
    return items.filter((row) => row.group_id != null && Number(row.group_id) > 0)
  }
  return items
})

function csv(value: unknown) {
  return typeof value === 'string' ? value.split(',').filter(Boolean) : []
}
function parseRange(value: unknown): MonitorRange {
  return ['90m', '24h', '7d', '30d'].includes(String(value)) ? (value as MonitorRange) : '90m'
}
function parseMatrixGroupBy(value: unknown): MonitorMatrixGroupBy {
  const allowed: MonitorMatrixGroupBy[] = [
    'platform',
    'platform_group',
    'platform_model',
    'platform_group_model',
  ]
  return allowed.includes(value as MonitorMatrixGroupBy)
    ? (value as MonitorMatrixGroupBy)
    : 'platform_group'
}
function parseTab(value: unknown, allowUsers: boolean): Tab {
  const allowed: Tab[] = allowUsers ? ['models', 'errors', 'users'] : ['models', 'errors']
  return allowed.includes(value as Tab) ? (value as Tab) : 'models'
}
function parseHealthMode(value: unknown): HealthMode {
  const allowed: HealthMode[] = ['overall', 'success', 'ttft', 'cache']
  return allowed.includes(value as HealthMode) ? (value as HealthMode) : 'overall'
}
function parseTrendView(value: unknown): TrendView {
  return value === 'line' ? 'line' : 'pulse'
}
function syncQuery() {
  void router.replace({
    query: {
      range: filter.value.range,
      platform: filter.value.platforms.join(',') || undefined,
      group: filter.value.groupIds.join(',') || undefined,
      model: filter.value.models.join(',') || undefined,
      group_by: matrixGroupBy.value,
      health_mode: healthMode.value,
      trend_view: trendView.value === 'line' ? 'line' : undefined,
      tab: activeTab.value,
    },
  })
}
/** Dimensions catalog: range only — never re-filtered by platform/group/model selection. */
async function loadDimensions(signal?: AbortSignal, id = sequence) {
  const rangeOnly: MonitorFilter = {
    range: filter.value.range,
    platforms: [],
    groupIds: [],
    models: [],
  }
  const next = await api.getDimensions(rangeOnly, isAdmin.value, signal)
  if (id !== sequence) return
  dimensions.value = next
}

async function loadMetrics(signal?: AbortSignal, id = sequence) {
  const [nextSnapshot, nextMatrix] = await Promise.all([
    api.getSnapshot(filter.value, isAdmin.value, signal),
    api.getMatrix(filter.value, matrixGroupBy.value, isAdmin.value, signal),
  ])
  if (id !== sequence) return
  snapshot.value = nextSnapshot
  matrix.value = nextMatrix
  scheduleAutoRefresh()
  await loadTab(signal, id)
}

async function reload(silent = true) {
  controller?.abort()
  const request = new AbortController()
  controller = request
  const id = ++sequence
  refreshing.value = true
  if (!silent) loading.value = true
  try {
    // Catalog + metrics in parallel; catalog ignores dimension filters so options never shrink.
    await Promise.all([
      loadDimensions(request.signal, id),
      loadMetrics(request.signal, id),
    ])
  } catch (error) {
    if ((error as { name?: string }).name !== 'CanceledError') {
      appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2.loadFailed')))
    }
  } finally {
    if (id === sequence) {
      loading.value = false
      tabLoading.value = false
      refreshing.value = false
    }
  }
}

/** When only range changes, still refresh dimensions; dimension filters only re-load metrics. */
async function reloadMetricsOnly(silent = true) {
  controller?.abort()
  const request = new AbortController()
  controller = request
  const id = ++sequence
  refreshing.value = true
  if (!silent) loading.value = true
  try {
    await loadMetrics(request.signal, id)
  } catch (error) {
    if ((error as { name?: string }).name !== 'CanceledError') {
      appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2.loadFailed')))
    }
  } finally {
    if (id === sequence) {
      loading.value = false
      tabLoading.value = false
      refreshing.value = false
    }
  }
}
async function loadTab(signal?: AbortSignal, id = sequence) {
  tabLoading.value = true
  try {
    if (activeTab.value === 'models') {
      modelRows.value = (await api.getModels(filter.value, isAdmin.value, signal)).items || []
    } else if (activeTab.value === 'errors') {
      errorRows.value = (await api.getErrors(filter.value, isAdmin.value, signal)).items || []
    } else if (showUserRanking.value) {
      userRows.value = (await api.getUsers(filter.value, isAdmin.value, signal)).items || []
    } else {
      userRows.value = []
    }
  } catch (error) {
    const e = error as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.name === 'CanceledError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2.detailLoadFailed')))
  } finally {
    if (id === sequence) tabLoading.value = false
  }
}
function setRange(value: MonitorRange) {
  filter.value.range = value
}
function clearDimensions() {
  // Replace arrays so deep watch always fires and metrics reload full window.
  filter.value = {
    ...filter.value,
    platforms: [],
    groupIds: [],
    models: [],
  }
}
function scheduleAutoRefresh() {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
  // Poll faster while first-upgrade bootstrap is filling 90m→30d so the progress bar moves.
  const seconds = bootstrapActive.value
    ? 10
    : snapshot.value?.config?.refresh_interval_seconds || 300
  autoRefreshTimer = window.setInterval(() => {
    if (!loading.value && !refreshing.value) {
      void reload(true)
    }
  }, Math.max(bootstrapActive.value ? 10 : 60, seconds) * 1000)
}
function drillModel(row: MonitorModelRow) {
  filter.value.platforms = [row.platform]
  filter.value.models = [row.model]
}
function formatRate(value: number) {
  return formatMonitorThroughput(value)
}
function exactRate(value: number) {
  return Intl.NumberFormat(locale.value || undefined, { maximumFractionDigits: 2 }).format(value || 0)
}
function formatTps(tpm: number | null | undefined) {
  return formatMonitorTokensPerSecond(tpm)
}
function exactTps(tpm: number | null | undefined) {
  return Intl.NumberFormat(locale.value || undefined, { maximumFractionDigits: 3 }).format(
    tokensPerSecondFromTpm(tpm),
  )
}
function formatPercent(value: number) {
  return formatMonitorPercent(value)
}
function formatMs(value: number | null) {
  return formatMonitorMs(value)
}
function ttftCellState(state: HealthState | undefined, metric: { p50_ms: number | null; sample_count?: number }) {
  return ttftDisplayState(state, metric)
}
function latencyDetail(metric: {
  p50_ms: number | null
  p90_ms?: number | null
  p95_ms: number | null
  avg_ms?: number | null
}) {
  return formatLatencyPrivacy(metric.p50_ms, metric.p90_ms, metric.avg_ms, metric.p95_ms)
}
/** KPI secondary: AVG · P90 under the P50 primary value. */
function latencyKpiSecondary(metric: {
  p90_ms?: number | null
  p95_ms: number | null
  avg_ms?: number | null
}) {
  return formatLatencyKpiSecondary(metric.avg_ms, metric.p90_ms, metric.p95_ms)
}
function formatTime(value: string) {
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
function statusDot(health?: MonitorHealth | HealthState) {
  if (!health || typeof health === 'string') {
    return `status-dot health-${health || 'unknown'}`
  }
  // Prefer multi-band score when available; otherwise fall back to the coarse
  // overall state for mixed-version/older payloads.
  const klass =
    health.score != null
      ? healthScoreClass(health, 'overall', 0)
      : `health-${health.overall || 'unknown'}`
  return `status-dot ${klass}`
}
function errorLabel(value: string) {
  const key = `channelMonitorV2.errorCategories.${value}`
  return te(key) ? t(key) : monitorErrorCategoryLabel(value)
}
function toggleError(category: string) {
  const next = new Set(expandedErrors.value)
  if (next.has(category)) next.delete(category)
  else next.add(category)
  expandedErrors.value = next
}

let lastRange: MonitorRange = filter.value.range
watch(
  filter,
  () => {
    syncQuery()
    const rangeChanged = filter.value.range !== lastRange
    lastRange = filter.value.range
    if (rangeChanged) void reload(true)
    else void reloadMetricsOnly(true)
  },
  { deep: true }
)
watch(matrixGroupBy, () => {
  syncQuery()
  void reloadMetricsOnly(true)
})
watch(healthMode, syncQuery)
watch(trendView, syncQuery)
watch(activeTab, () => {
  syncQuery()
  void loadTab()
})
watch(showUserRanking, (allowed) => {
  if (!allowed && activeTab.value === 'users') {
    activeTab.value = 'models'
  }
})
onMounted(() => void reload(false))
onBeforeUnmount(() => {
  controller?.abort()
  if (autoRefreshTimer) window.clearInterval(autoRefreshTimer)
})
</script>

<style scoped>
@media (max-width: 767px), (max-width: 1023px) and (max-height: 500px) and (pointer: coarse) {
  .monitor-records {
    max-height: none;
    overflow: visible;
  }
}
.status-dot {
  display: inline-block;
  height: 0.5rem;
  width: 0.5rem;
  flex: none;
  border-radius: 9999px;
}
/* Multi-stop green → yellow → red score bands */
.health-score10 { background: #0d8a6a; }
.health-score9  { background: #10946c; }
.health-score8  { background: #2db692; }
.health-score7  { background: #6cd3b7; }
.health-score6  { background: #eab308; }
.health-score5  { background: #fbbf24; }
.health-score4  { background: #d97706; }
.health-score3  { background: #ea580c; }
.health-score2  { background: #e34948; }
.health-score1  { background: #dc2626; }
.health-score0  { background: #b91c1c; }
.health-healthy  { background: #10946c; }
.health-warning  { background: #d97706; }
.health-critical { background: #dc2626; }
.health-unknown  { background: var(--atelier-ink-muted); opacity: 0.45; }
details > summary::-webkit-details-marker {
  display: none;
}
</style>
