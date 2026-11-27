<template>
  <section class="portal-api-models grid min-w-0 grid-cols-1 items-start gap-6 lg:grid-cols-[232px_minmax(0,1fr)]" :aria-label="text('模型目录', 'Model catalogue')" :aria-busy="loading">
    <!-- Left Connection Groups Rail -->
    <aside class="atelier-surface min-w-0 p-3.5 lg:sticky lg:top-16">
      <div class="mb-2.5 flex items-center justify-between gap-2 px-1">
        <h2 class="text-xs font-semibold uppercase tracking-wider text-[var(--atelier-ink-muted)]">
          {{ text('接入分组', 'Connections') }}
        </h2>
        <span class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
          {{ connections.length }}
        </span>
      </div>

      <div class="relative mb-2.5">
        <Icon name="search" size="xs" class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
        <input
          v-model="groupSearch"
          type="search"
          class="atelier-input !min-h-[32px] !pl-8 text-xs"
          :aria-label="text('搜索分组', 'Search connections')"
          :placeholder="text('搜索分组', 'Search connections')"
        />
      </div>

      <div class="mb-2 lg:hidden">
        <UiSelect
          :model-value="selectedGroupId ?? ''"
          :options="connectionSelectOptions"
          size="sm"
          :aria-label="text('接入分组', 'Connections')"
          @update:model-value="onMobileGroupChange"
        />
      </div>

      <nav
        ref="groupNav"
        class="portal-api-group-nav hidden max-h-[calc(100dvh-260px)] flex-col gap-1 overflow-y-auto pr-0.5 lg:flex"
        :aria-label="text('接入分组', 'Connections')"
        tabindex="0"
        @scroll="updateGroupScroll"
      >
        <button
          v-for="connection in visibleConnections"
          :key="connection.id"
          type="button"
          class="flex w-full flex-col gap-1 rounded-lg border px-2.5 py-2 text-left transition-colors"
          :class="selectedGroupId === connection.id ? 'border-[var(--atelier-border-strong)] bg-[var(--atelier-selected-row)] text-[var(--atelier-ink)]' : 'border-transparent text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]'"
          :aria-current="selectedGroupId === connection.id ? 'true' : undefined"
          :aria-pressed="selectedGroupId === connection.id"
          @click="$emit('update:selectedGroupId', connection.id)"
        >
          <span class="flex min-w-0 items-center gap-2">
            <BrandPlatformIcon :platform="connection.platform" size="sm" />
            <span class="truncate text-xs font-medium text-[var(--atelier-ink)]">{{ connection.name }}</span>
          </span>
          <span class="flex items-center justify-between pl-6 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
            <span>{{ connection.count }} {{ text('模型', 'models') }}</span>
            <span v-if="connection.rate != null">×{{ connection.rate }}</span>
          </span>
          <small v-if="showCreate && !bindableGroupIds.includes(connection.id)" class="pl-6 text-[10.5px] text-[var(--atelier-ink-muted)]">
            {{ text('仅价格预览', 'Pricing preview only') }}
          </small>
        </button>
        <p v-if="!visibleConnections.length" class="py-6 text-center text-xs text-[var(--atelier-ink-muted)]">
          {{ text('没有匹配的分组', 'No matching connections') }}
        </p>
      </nav>

      <div v-if="groupScroll.above || groupScroll.below" class="mt-2 hidden items-center justify-between border-t border-[var(--atelier-border-subtle)] pt-2 text-[11px] text-[var(--atelier-ink-muted)] lg:flex">
        <span>{{ text('滚动查看更多', 'Scroll for more') }}</span>
        <div class="flex items-center gap-1">
          <button
            type="button"
            class="inline-flex h-6 w-6 items-center justify-center rounded hover:bg-[var(--atelier-hover)] disabled:opacity-35"
            :disabled="!groupScroll.above"
            :aria-label="text('向上滚动分组', 'Scroll connections up')"
            @click="scrollConnections(-1)"
          >
            <Icon name="chevronUp" size="xs" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="inline-flex h-6 w-6 items-center justify-center rounded hover:bg-[var(--atelier-hover)] disabled:opacity-35"
            :disabled="!groupScroll.below"
            :aria-label="text('向下滚动分组', 'Scroll connections down')"
            @click="scrollConnections(1)"
          >
            <Icon name="chevronDown" size="xs" aria-hidden="true" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Full-Width Pricing Matrix Area -->
    <div class="min-w-0 space-y-4">
      <!-- Group Header + Search -->
      <header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ selectedGroup?.name || connections.find(group => group.id === selectedGroupId)?.name || text('选择接入分组', 'Choose a connection') }}
            </h2>
            <template v-if="selectedGroup">
              <span class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-secondary)]">
                {{ selectedGroup.user_rate_multiplier != null ? text('专属倍率', 'Personal rate') : text('分组倍率', 'Group rate') }} ×{{ selectedGroup.user_rate_multiplier ?? selectedGroup.rate_multiplier }}
              </span>
              <span class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 text-[11px] text-[var(--atelier-ink-muted)]">
                {{ selectedGroup.subscription_type === 'subscription' ? text('订阅分组', 'Subscription') : text('按量计费', 'Pay as you go') }}
              </span>
              <span v-if="selectedGroup.is_exclusive" class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 text-[11px] text-[var(--atelier-ink-secondary)]">
                {{ text('专属分组', 'Exclusive') }}
              </span>
              <span v-if="selectedGroup.image_rate_independent" class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono text-[11px] text-[var(--atelier-ink-muted)]">
                {{ text('图片', 'Image') }} ×{{ selectedGroup.image_rate_multiplier }}
              </span>
              <span v-if="selectedGroup.video_rate_independent" class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono text-[11px] text-[var(--atelier-ink-muted)]">
                {{ text('视频', 'Video') }} ×{{ selectedGroup.video_rate_multiplier }}
              </span>
            </template>
          </div>
          <p v-if="selectedGroup?.description" class="mt-1 whitespace-pre-line text-xs text-[var(--atelier-ink-secondary)]">
            {{ selectedGroup.description }}
          </p>
        </div>

        <div class="relative w-full shrink-0 sm:w-64">
          <Icon name="search" size="xs" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
          <input
            type="search"
            class="atelier-input !min-h-[36px] !pl-8 text-xs"
            :value="search"
            :aria-label="text('搜索模型', 'Search models')"
            :placeholder="text('搜索模型或厂商', 'Search models or vendors')"
            @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
          />
        </div>
      </header>

      <!-- States -->
      <div v-if="!pricingEnabled" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]">
        {{ text('模型价格暂未开放。你仍可创建和管理 API Key。', 'Model pricing is not published. You can still create and manage API keys.') }}
      </div>
      <div v-else-if="loading" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]" role="status">
        {{ text('正在加载模型价格…', 'Loading model pricing…') }}
      </div>
      <div v-else-if="error" class="flex items-center justify-between gap-2 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-xs text-[var(--atelier-err)]" role="alert">
        <span>{{ error }}</span>
        <UiButton variant="ghost" size="sm" @click="$emit('retry')">{{ text('重试', 'Retry') }}</UiButton>
      </div>

      <template v-else>
        <!-- Single Clean Row: Full-Color Vendor Filter Pills + Concise Capability Filters -->
        <div v-if="platforms.length" class="flex flex-wrap items-center justify-between gap-2.5">
          <div class="portal-model-platforms flex flex-wrap items-center gap-1.5" role="group" :aria-label="text('模型厂商', 'Model vendors')">
            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors"
              :class="!selectedPlatform ? 'border-[var(--atelier-ink)] bg-[var(--atelier-ink)] text-[var(--atelier-ink-inverse)]' : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)] hover:border-[var(--atelier-border-strong)] hover:text-[var(--atelier-ink)]'"
              :aria-pressed="!selectedPlatform"
              @click="selectedPlatform = ''"
            >
              <span>{{ text('全部', 'All') }}</span>
              <span class="font-mono text-[11px] opacity-75">{{ searchedModels.length }}</span>
            </button>
            <button
              v-for="platform in platforms"
              :key="platform.id"
              type="button"
              class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors"
              :class="selectedPlatform === platform.id ? 'border-[var(--atelier-ink)] bg-[var(--atelier-ink)] text-[var(--atelier-ink-inverse)]' : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)] hover:border-[var(--atelier-border-strong)] hover:text-[var(--atelier-ink)]'"
              :aria-pressed="selectedPlatform === platform.id"
              @click="selectedPlatform = platform.id"
            >
              <BrandModelIcon :model="platform.models[0]!.name" size="xs" />
              <span>{{ vendorLabel(platform) }}</span>
              <span class="font-mono text-[11px] opacity-75">{{ platform.models.length }}</span>
            </button>
          </div>

          <!-- Concise Capability Filters -->
          <div class="flex flex-wrap items-center gap-1.5" role="group" :aria-label="text('能力筛选', 'Capability filters')">
            <button
              type="button"
              class="inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors"
              :class="capFilter.reasoning ? 'border-[var(--atelier-ink)] bg-[var(--atelier-selected-row)] text-[var(--atelier-ink)]' : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-muted)] hover:text-[var(--atelier-ink)]'"
              :aria-pressed="capFilter.reasoning"
              @click="capFilter.reasoning = !capFilter.reasoning"
            >
              {{ text('推理', 'Reasoning') }}
            </button>
            <button
              type="button"
              class="inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors"
              :class="capFilter.tools ? 'border-[var(--atelier-ink)] bg-[var(--atelier-selected-row)] text-[var(--atelier-ink)]' : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-muted)] hover:text-[var(--atelier-ink)]'"
              :aria-pressed="capFilter.tools"
              @click="capFilter.tools = !capFilter.tools"
            >
              {{ text('工具调用', 'Tools') }}
            </button>
          </div>
        </div>

        <!-- Pricing Unit Legend -->
        <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-[var(--atelier-ink-muted)]">
          <span>
            {{ text('原价 · 未乘分组倍率', 'Base prices · before group multipliers') }}
            <span class="ml-1.5 text-[var(--atelier-ink-secondary)]">({{ text('标记价为倍率后价格', 'Badges show adjusted prices') }})</span>
          </span>
          <span class="font-mono text-[11.5px]">USD / 1M tokens</span>
        </div>

        <!-- Mobile Cards -->
        <div v-if="mobile" class="space-y-4">
          <section v-for="platform in visiblePlatforms" :key="platform.id" class="space-y-2.5">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--atelier-ink-muted)]">
              {{ vendorLabel(platform) }} · {{ platform.models.length }} {{ text('个模型', 'models') }}
            </h3>
            <article
              v-for="model in platform.models"
              :key="model.platform + ':' + model.name"
              class="portal-model-card atelier-surface space-y-2.5 p-3.5"
              :data-selected="isCompared(model) ? 'true' : undefined"
            >
              <div class="flex items-center justify-between gap-2">
                <button
                  type="button"
                  class="portal-api-model-name flex min-w-0 items-center gap-2 text-left text-sm font-medium text-[var(--atelier-ink)]"
                  @click="detailModel = model"
                >
                  <BrandModelIcon :model="model.name" size="sm" />
                  <span class="truncate">{{ model.name }}</span>
                  <Icon name="chevronRight" size="xs" class="shrink-0 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
                </button>
                <div class="flex shrink-0 items-center gap-1">
                  <button
                    type="button"
                    class="inline-flex h-7 w-7 items-center justify-center rounded-md text-[var(--atelier-ink-muted)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                    :title="text('复制模型 ID', 'Copy model ID')"
                    :aria-label="text('复制模型 ID', 'Copy model ID') + ': ' + model.name"
                    @click.stop="copyToClipboard(model.name)"
                  >
                    <Icon name="copy" size="xs" aria-hidden="true" />
                  </button>
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2 text-[11.5px] text-[var(--atelier-ink-muted)]">
                <span>{{ text('接入', 'Platform') }}: {{ platformLabel(model.platform) }}</span>
                <span v-if="model.context_window" class="font-mono tabular-nums">{{ compactTokens(model.context_window) }} {{ text('上下文', 'context') }}</span>
                <span v-if="model.reasoning">{{ text('推理', 'Reasoning') }}</span>
                <span v-if="model.tool_call">{{ text('工具调用', 'Tools') }}</span>
              </div>
              <div v-if="modelBadges(model).length" class="flex flex-wrap gap-1">
                <span v-for="badge in modelBadges(model)" :key="badge" class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 text-[10.5px] text-[var(--atelier-ink-secondary)]">
                  {{ badge }}
                </span>
              </div>
              <p v-if="!model.pricing" class="text-xs text-[var(--atelier-ink-muted)]">
                {{ text('尚未提供实际单价', 'Actual pricing is not provided') }}
              </p>
              <dl v-else-if="model.pricing.billing_mode === 'token'" class="grid grid-cols-2 gap-2.5 border-t border-[var(--atelier-border-subtle)] pt-2.5 text-xs">
                <div v-for="field in summaryTokenFields" :key="field">
                  <dt class="text-[11px] text-[var(--atelier-ink-muted)]">{{ fieldLabel(field) }}</dt>
                  <dd class="mt-0.5 font-mono tabular-nums">
                    <PortalModelPrice v-bind="rangePrice(model, field)" />
                    <template v-if="field === 'cache_write_price'">
                      <template v-for="duration in cacheDurations" :key="duration.field">
                        <PortalModelPrice v-if="hasCacheDuration(model, duration.field)" v-bind="rangePrice(model, duration.field)" :caption="duration.label" />
                      </template>
                    </template>
                  </dd>
                </div>
              </dl>
              <dl v-else class="border-t border-[var(--atelier-border-subtle)] pt-2.5 text-xs">
                <div>
                  <dt class="text-[11px] text-[var(--atelier-ink-muted)]">
                    {{ billingLabel(model) }}
                    <span v-if="independentRate(model)"> · {{ text('独立倍率', 'Independent rate') }} ×{{ effectiveRate(model) }}</span>
                  </dt>
                  <dd class="mt-0.5 font-mono tabular-nums">
                    <PortalModelPrice v-bind="rangePrice(model, 'per_request_price')" :unit="unitLabel(model)" />
                  </dd>
                </div>
              </dl>
            </article>
          </section>
          <p v-if="!visiblePlatforms.length" class="py-10 text-center text-sm text-[var(--atelier-ink-muted)]">
            {{ emptyMessage }}
          </p>
        </div>

        <!-- Desktop High-Density Tabular-Nums Pricing Matrix Table (.atelier-table) -->
        <div v-else class="overflow-x-auto border-y border-[var(--atelier-border)] bg-[var(--atelier-surface)]">
          <table class="atelier-table portal-api-model-table">
            <thead>
              <tr>
                <th scope="col">{{ text('模型', 'Model') }}</th>
                <th scope="col">{{ text('输入', 'Input') }}</th>
                <th scope="col">{{ text('缓存写入', 'Cache write') }}</th>
                <th scope="col">{{ text('缓存读取', 'Cache read') }}</th>
                <th scope="col">{{ text('输出', 'Output') }}</th>
                <th scope="col" class="text-right">{{ text('估算对比', 'Compare') }}</th>
              </tr>
            </thead>
            <tbody v-for="platform in visiblePlatforms" :key="platform.id">
              <tr class="bg-[var(--atelier-subtle)]/70 hover:!bg-[var(--atelier-subtle)]/70">
                <th colspan="6" scope="rowgroup" class="px-3.5 py-2 text-left text-xs font-semibold text-[var(--atelier-ink-secondary)]">
                  <span>{{ vendorLabel(platform) }}</span>
                  <small class="ml-2 font-mono font-normal tabular-nums text-[var(--atelier-ink-muted)]">
                    {{ platform.models.length }} {{ text('个模型', 'models') }}
                  </small>
                </th>
              </tr>
              <tr
                v-for="model in platform.models"
                :key="model.platform + ':' + model.name"
                class="portal-api-model-row group"
                :data-selected="isCompared(model) ? 'true' : undefined"
              >
                <!-- Col 1: Model Identity -->
                <td class="min-w-[240px]">
                  <div class="flex items-center gap-1.5">
                    <button
                      type="button"
                      class="portal-api-model-name inline-flex min-w-0 items-center gap-2 text-left font-medium text-[var(--atelier-ink)] hover:underline"
                      @click="detailModel = model"
                    >
                      <BrandModelIcon :model="model.name" size="sm" />
                      <span class="truncate">{{ model.name }}</span>
                      <Icon name="chevronRight" size="xs" class="shrink-0 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
                    </button>
                    <button
                      type="button"
                      class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-[var(--atelier-ink-muted)] opacity-0 transition-opacity hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)] focus-visible:opacity-100 group-hover:opacity-100"
                      :title="text('复制模型 ID', 'Copy model ID')"
                      :aria-label="text('复制模型 ID', 'Copy model ID') + ': ' + model.name"
                      @click.stop="copyToClipboard(model.name)"
                    >
                      <Icon name="copy" size="xs" aria-hidden="true" />
                    </button>
                  </div>
                  <div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11.5px] text-[var(--atelier-ink-muted)]">
                    <span>{{ text('接入', 'Platform') }}: {{ platformLabel(model.platform) }}</span>
                    <span v-if="model.context_window" class="font-mono tabular-nums">{{ compactTokens(model.context_window) }} {{ text('上下文', 'context') }}</span>
                    <span v-if="model.reasoning">{{ text('推理', 'Reasoning') }}</span>
                    <span v-if="model.tool_call">{{ text('工具调用', 'Tools') }}</span>
                  </div>
                  <div v-if="modelBadges(model).length" class="mt-1 flex flex-wrap gap-1">
                    <span v-for="badge in modelBadges(model)" :key="badge" class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 text-[10.5px] text-[var(--atelier-ink-secondary)]">
                      {{ badge }}
                    </span>
                  </div>
                </td>

                <!-- Cols 2-5: Pricing (td:nth-child(2) preserved strictly as Input!) -->
                <td v-if="!model.pricing" colspan="4" class="text-xs text-[var(--atelier-ink-muted)]">
                  {{ text('尚未提供实际单价', 'Actual pricing is not provided') }}
                </td>
                <template v-else-if="model.pricing.billing_mode === 'token'">
                  <td v-for="field in summaryTokenFields" :key="field" class="whitespace-nowrap font-mono tabular-nums">
                    <PortalModelPrice v-bind="rangePrice(model, field)" />
                    <template v-if="field === 'cache_write_price'">
                      <template v-for="duration in cacheDurations" :key="duration.field">
                        <PortalModelPrice v-if="hasCacheDuration(model, duration.field)" v-bind="rangePrice(model, duration.field)" :caption="duration.label" />
                      </template>
                    </template>
                  </td>
                </template>
                <td v-else colspan="4" class="font-mono tabular-nums">
                  <span class="mr-2 font-sans text-xs text-[var(--atelier-ink-secondary)]">
                    {{ billingLabel(model) }}
                    <span v-if="independentRate(model)"> · {{ text('独立倍率', 'Independent rate') }} ×{{ effectiveRate(model) }}</span>
                  </span>
                  <PortalModelPrice v-bind="rangePrice(model, 'per_request_price')" :unit="unitLabel(model)" />
                </td>

                <!-- Col 6: Compare Toggle + Hover Try Link -->
                <td class="whitespace-nowrap text-right">
                  <div class="inline-flex items-center justify-end gap-1.5">
                    <button
                      v-if="isTokenPriced(model)"
                      type="button"
                      class="inline-flex h-7 items-center gap-1 rounded-md border px-2 text-xs font-medium transition-colors"
                      :class="isCompared(model) ? 'border-[var(--atelier-ink)] bg-[var(--atelier-ink)] text-[var(--atelier-ink-inverse)]' : 'border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)]'"
                      :aria-pressed="isCompared(model)"
                      @click="onToggleCompare(model)"
                    >
                      {{ isCompared(model) ? text('✓ 已选', '✓ Added') : text('+ 对比', '+ Compare') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
            <tbody v-if="!visiblePlatforms.length">
              <tr>
                <td colspan="6" class="py-12 text-center text-sm text-[var(--atelier-ink-muted)]">
                  <p>{{ emptyMessage }}</p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <p class="text-xs leading-relaxed text-[var(--atelier-ink-muted)]">
          {{ text('原价为未乘分组倍率的配置单价，并非官方参考价；标记价已应用当前分组或专属倍率，按标准时段展示。图片、视频仅列原价。— 表示未提供单价，0 表示免费；点击模型查看阶梯与分时规则，实际费用以请求命中的计费规则为准。', 'Base prices are configured prices before group multipliers, not official reference prices. Badges apply the current group or personal rate for the standard period. Images and videos show base prices only. — means unpriced; 0 means free. Select a model for tiers and schedules; charges follow the matched billing rules.') }}
        </p>

        <details v-if="descriptionHtml" class="rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-4 py-2.5 text-xs">
          <summary class="cursor-pointer font-medium text-[var(--atelier-ink)]">{{ text('价格说明', 'Pricing notes') }}</summary>
          <div class="mt-2 leading-relaxed text-[var(--atelier-ink-secondary)]" v-html="descriptionHtml" />
        </details>
      </template>

      <div v-if="showCreate" class="flex flex-wrap items-center gap-3 pt-2">
        <UiButton variant="primary" size="sm" :disabled="creationDisabled || selectedCannotBind" @click="$emit('create')">
          ＋ {{ text('创建 API Key', 'Create API key') }}
        </UiButton>
        <span v-if="selectedCannotBind" class="text-xs text-[var(--atelier-ink-muted)]">
          {{ text('此分组可查看价格，当前账户无法绑定。请选择可用分组。', 'Pricing is visible, but this account cannot bind this group. Choose an available group.') }}
        </span>
      </div>
    </div>

    <!-- Pricing Details Dialog -->
    <PortalDialog :open="!!detailModel" :title="detailModel?.name || text('定价详情', 'Pricing details')" wide @close="detailModel = null">
      <div v-if="detailModel" class="portal-api-price-detail space-y-4 text-sm text-[var(--atelier-ink-secondary)]">
        <p class="portal-api-detail-platform text-xs text-[var(--atelier-ink-muted)]">
          {{ text('接入平台', 'Connection platform') }}: {{ platformLabel(detailModel.platform) }}
        </p>
        <ModelCapabilities :model="detailModel" />
        <p v-if="!detailModel.pricing">{{ text('此模型尚未提供实际单价。', 'Actual pricing is not provided for this model.') }}</p>
        <template v-else-if="selectedGroup">
          <dl class="portal-api-price-summary grid grid-cols-1 gap-3 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] p-3.5 text-xs sm:grid-cols-3">
            <div>
              <dt class="text-[var(--atelier-ink-muted)]">{{ text('接入分组', 'Group') }}</dt>
              <dd class="mt-0.5 font-medium text-[var(--atelier-ink)]">{{ selectedGroup.name }}</dd>
            </div>
            <div>
              <dt class="text-[var(--atelier-ink-muted)]">{{ text('计费方式', 'Billing mode') }}</dt>
              <dd class="mt-0.5 font-medium text-[var(--atelier-ink)]">{{ billingLabel(detailModel) }}</dd>
            </div>
            <div>
              <dt class="text-[var(--atelier-ink-muted)]">{{ text('生效倍率', 'Effective multiplier') }}</dt>
              <dd class="mt-0.5 font-mono font-medium tabular-nums text-[var(--atelier-ink)]">{{ effectiveRate(detailModel) == null ? '—' : '×' + effectiveRate(detailModel) }}<span v-if="independentRate(detailModel)">{{ text('（独立倍率）', ' (independent rate)') }}</span></dd>
            </div>
            <div v-if="selectedGroup.user_rate_multiplier != null && !independentRate(detailModel)" class="sm:col-span-3">
              <dt class="text-[var(--atelier-ink-muted)]">{{ text('用户专属倍率', 'User-specific rate') }}</dt>
              <dd class="mt-0.5 text-[var(--atelier-ink)]">
                {{ text('已使用专属倍率替代分组默认倍率', 'Replaces the default group multiplier') }} ×{{ selectedGroup.rate_multiplier }}
              </dd>
            </div>
          </dl>
          <p class="text-xs leading-relaxed">{{ text('主价为未乘倍率的原价，标记价已应用当前生效倍率，均按标准时段展示；图片、视频仅展示原价。— 表示未提供单价，0 表示免费。', 'Main prices are base prices before multipliers; badges apply the effective rate for the standard period. Images and videos show base prices only. — means unpriced; 0 means free.') }}</p>
          <p v-if="detailModel.pricing.billing_mode === 'token'" class="text-xs">{{ text('单位：USD / 1M tokens。缓存写入按 5 分钟、1 小时及 12 小时分别列出；未配置价格显示 —。', 'Unit: USD / 1M tokens. Cache writes are listed for 5m, 1h and 12h; unconfigured prices show —.') }}</p>
          <p v-if="detailTiers.length > 1" class="text-xs">{{ detailModel.long_context_basis === 'marginal' ? text('阶梯采用边际计价：仅超出阈值的部分按对应档位计费。', 'Marginal tiers: only usage beyond a threshold is charged at that tier.') : text('阶梯采用整单计价：整个请求按所处档位的单价计费。', 'Whole-request tiers: the entire request uses the matching tier price.') }}</p>
          <p v-if="detailModel.pricing.billing_mode === 'token' && selectedGroup.long_context_pricing_enabled === false" class="text-xs">{{ text('此分组未启用长上下文阶梯，当前展示基础档位。', 'Long-context tiering is disabled for this group; the base tier is shown.') }}</p>

          <div v-if="mobile" class="space-y-2.5">
            <article v-for="(tier, index) in detailTiers" :key="index" class="portal-model-card atelier-surface space-y-2 p-3">
              <h3 class="text-xs font-semibold text-[var(--atelier-ink)]">{{ text('档位', 'Tier') }} · {{ tierLabel(tier) }}</h3>
              <dl v-if="detailModel.pricing.billing_mode === 'token'" class="grid grid-cols-2 gap-2 text-xs">
                <div v-for="field in tokenFields" :key="field">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ fieldLabel(field) }}</dt>
                  <dd class="font-mono tabular-nums"><PortalModelPrice v-bind="priceDisplay(detailModel, [tier[field]])" /></dd>
                </div>
              </dl>
              <dl v-else class="text-xs">
                <div>
                  <dt class="text-[var(--atelier-ink-muted)]">{{ text('价格', 'Price') }} {{ unitLabel(detailModel) }}</dt>
                  <dd class="font-mono tabular-nums"><PortalModelPrice v-bind="priceDisplay(detailModel, [tier.per_request_price])" /></dd>
                </div>
              </dl>
            </article>
          </div>
          <div v-else class="overflow-x-auto rounded-xl border border-[var(--atelier-border)]">
            <table class="atelier-table portal-api-detail-table">
              <thead>
                <tr>
                  <th>{{ text('档位', 'Tier') }}</th>
                  <template v-if="detailModel.pricing.billing_mode === 'token'">
                    <th>{{ text('输入', 'Input') }}</th>
                    <th>{{ text('缓存写 5m', 'Cache write 5m') }}</th>
                    <th>{{ text('缓存写 1h', 'Cache write 1h') }}</th>
                    <th>{{ text('缓存写 12h', 'Cache write 12h') }}</th>
                    <th>{{ text('缓存读', 'Cache read') }}</th>
                    <th>{{ text('输出', 'Output') }}</th>
                  </template>
                  <th v-else>{{ text('价格', 'Price') }} {{ unitLabel(detailModel) }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(tier, index) in detailTiers" :key="index">
                  <td class="font-mono text-xs tabular-nums">{{ tierLabel(tier) }}</td>
                  <template v-if="detailModel.pricing.billing_mode === 'token'">
                    <td v-for="field in tokenFields" :key="field" class="font-mono tabular-nums">
                      <PortalModelPrice v-bind="priceDisplay(detailModel, [tier[field]])" />
                    </td>
                  </template>
                  <td v-else class="font-mono tabular-nums">
                    <PortalModelPrice v-bind="priceDisplay(detailModel, [tier.per_request_price])" />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <details v-if="detailModel.pricing.billing_mode === 'token' && detailModel.official_pricing" class="rounded-xl border border-[var(--atelier-border)] px-3.5 py-2.5 text-xs">
            <summary class="cursor-pointer font-medium text-[var(--atelier-ink)]">{{ text('官方参考价 · USD / 1M tokens', 'Official reference · USD / 1M tokens') }}</summary>
            <p class="mt-1.5 text-[var(--atelier-ink-muted)]">{{ text('参考价不参与本站扣费，也不应用分组倍率。', 'Reference prices are not charged here and do not apply group multipliers.') }}</p>
            <div v-if="mobile" class="mt-2 space-y-2">
              <article v-for="(tier, index) in officialTiers" :key="index" class="portal-model-card atelier-surface p-2.5">
                <h3 class="text-xs font-medium text-[var(--atelier-ink)]">{{ text('档位', 'Tier') }} · {{ tierLabel(tier) }}</h3>
                <dl class="mt-1.5 grid grid-cols-2 gap-1.5">
                  <div v-for="field in tokenFields" :key="field">
                    <dt class="text-[var(--atelier-ink-muted)]">{{ fieldLabel(field) }}</dt>
                    <dd class="font-mono tabular-nums">{{ tier[field] == null ? '—' : formatScaled(tier[field]!, 1_000_000, 2) }}</dd>
                  </div>
                </dl>
              </article>
            </div>
            <div v-else class="mt-2 overflow-x-auto">
              <table class="atelier-table portal-api-detail-table">
                <thead>
                  <tr>
                    <th>{{ text('档位', 'Tier') }}</th>
                    <th>{{ text('输入', 'Input') }}</th>
                    <th>{{ text('缓存写 5m', 'Cache write 5m') }}</th>
                    <th>{{ text('缓存写 1h', 'Cache write 1h') }}</th>
                    <th>{{ text('缓存写 12h', 'Cache write 12h') }}</th>
                    <th>{{ text('缓存读', 'Cache read') }}</th>
                    <th>{{ text('输出', 'Output') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(tier, index) in officialTiers" :key="index">
                    <td class="font-mono tabular-nums">{{ tierLabel(tier) }}</td>
                    <td v-for="field in tokenFields" :key="field" class="font-mono tabular-nums">
                      {{ tier[field] == null ? '—' : formatScaled(tier[field]!, 1_000_000, 2) }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </details>

          <div v-if="detailModel.pricing.billing_mode === 'token' && (detailModel.pricing.image_input_price != null || detailModel.pricing.image_output_price != null)" class="space-y-1.5 rounded-xl border border-[var(--atelier-border)] p-3 text-xs">
            <h3 class="font-semibold text-[var(--atelier-ink)]">{{ text('图像 Token 单价', 'Image token prices') }}</h3>
            <div class="flex flex-wrap gap-6 font-mono tabular-nums">
              <div><span class="mr-2 font-sans text-[var(--atelier-ink-muted)]">{{ text('输入', 'Input') }}</span><PortalModelPrice v-bind="priceDisplay(detailModel, [detailModel.pricing.image_input_price])" unit="/ 1M tokens" /></div>
              <div><span class="mr-2 font-sans text-[var(--atelier-ink-muted)]">{{ text('输出', 'Output') }}</span><PortalModelPrice v-bind="priceDisplay(detailModel, [detailModel.pricing.image_output_price])" unit="/ 1M tokens" /></div>
            </div>
          </div>

          <div v-if="detailModel.time_pricing?.periods.length" class="space-y-1 rounded-xl border border-[var(--atelier-border)] p-3 text-xs">
            <h3 class="font-semibold text-[var(--atelier-ink)]">{{ text('分时倍率', 'Time-based multipliers') }}</h3>
            <p>{{ detailModel.time_pricing.timezone }} · {{ detailModel.time_pricing.weekdays_only ? text('仅工作日；周末使用标准价', 'Weekdays only; standard pricing on weekends') : text('每日生效', 'Applies every day') }}</p>
            <ul class="list-disc pl-5 font-mono tabular-nums">
              <li v-for="(period, index) in detailModel.time_pricing.periods" :key="index">{{ period.start_time }}–{{ period.end_time }} · ×{{ period.multiplier }}</li>
            </ul>
            <p class="text-[var(--atelier-ink-muted)]">{{ text('命中时段的请求在标准价基础上乘对应倍率；与高峰窗口重叠时，还会乘高峰倍率。', 'Matching requests multiply the standard price by the period rate. Overlapping peak windows add the peak multiplier.') }}</p>
          </div>

          <div v-if="hasPeakRate(selectedGroup)" class="space-y-1 rounded-xl border border-[var(--atelier-border)] p-3 text-xs">
            <h3 class="font-semibold text-[var(--atelier-ink)]">{{ text('高峰时段', 'Peak window') }}</h3>
            <p class="font-mono tabular-nums">{{ peakWindow }}</p>
            <p class="text-[var(--atelier-ink-muted)]">{{ text('该窗口内的请求还会乘以上述高峰倍率。', 'Requests in this window also apply the peak multiplier above.') }}</p>
          </div>

          <div v-if="reasoningRates(detailModel).length" class="space-y-1 rounded-xl border border-[var(--atelier-border)] p-3 text-xs">
            <h3 class="font-semibold text-[var(--atelier-ink)]">{{ text('推理强度倍率', 'Reasoning-effort multipliers') }}</h3>
            <ul class="list-disc pl-5 font-mono tabular-nums">
              <li v-for="[effort, multiplier] in reasoningRates(detailModel)" :key="effort">{{ effort }} · ×{{ multiplier }}</li>
            </ul>
          </div>
        </template>
      </div>
      <template v-if="detailModel" #footer>
        <div class="flex justify-end gap-2">
          <UiButton variant="secondary" size="sm" @click="detailModel = null">
            {{ text('关闭', 'Close') }}
          </UiButton>
          <UiButton v-if="showCreate && detailModel.pricing?.billing_mode === 'token'" variant="primary" size="sm" @click="chooseExampleModel">
            {{ text('查看调用示例', 'View request example') }}
          </UiButton>
        </div>
      </template>
    </PortalDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useMediaQuery, useResizeObserver } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { getActivePinia } from 'pinia'
import { UiButton, UiSelect, BrandModelIcon, BrandPlatformIcon } from '@/ui'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
import Icon from '@/components/icons/Icon.vue'
import ModelCapabilities from '@/components/modelPlaza/ModelCapabilities.vue'
import { useClipboard } from '@/composables/useClipboard'
import { matchesCatalogModel, modelVendor } from '@/utils/modelCatalog'
import { platformLabel } from '@/utils/platformColors'
import PortalDialog from './PortalDialog.vue'
import PortalModelPrice from './PortalModelPrice.vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import type { UserPricingInterval } from '@/api/channels'
import { formatScaled, resolveIntervalPrices } from '@/utils/pricing'
import { formatPeakRateWindow, hasPeakRate, serverTimezoneLabel } from '@/utils/peak-rate'

const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)

const copiedModel = ref<string | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | undefined
async function copyToClipboard(modelId: string) {
  if (getActivePinia()) {
    await useClipboard().copyToClipboard(modelId)
    return
  }
  await navigator.clipboard?.writeText(modelId).catch(() => {})
  copiedModel.value = modelId
  clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    if (copiedModel.value === modelId) copiedModel.value = null
  }, 1500)
}

const props = withDefaults(
  defineProps<{
    groups: ModelPlazaGroup[]
    availableConnections?: { id: number; name: string; platform?: string }[]
    bindableGroupIds?: number[]
    selectedGroupId: number | null
    search?: string
    description?: string
    serverUtcOffset?: string
    pricingEnabled?: boolean
    loading?: boolean
    error?: string
    showCreate?: boolean
    creationDisabled?: boolean
    comparedKeys?: string[]
  }>(),
  {
    availableConnections: () => [],
    bindableGroupIds: () => [],
    search: '',
    description: '',
    serverUtcOffset: '',
    pricingEnabled: true,
    loading: false,
    error: '',
    showCreate: false,
    creationDisabled: false,
    comparedKeys: () => []
  }
)

const emit = defineEmits<{
  'update:selectedGroupId': [id: number | null]
  'update:search': [search: string]
  retry: []
  create: []
  'select-model': [name: string]
  'toggle-compare': [key: string]
}>()

const { locale } = useI18n()
const text = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en)
const descriptionHtml = computed(() => DOMPurify.sanitize(marked.parse(props.description, { async: false, breaks: true })))
type PriceField = 'input_price' | 'cache_write_price' | 'cache_write_1h_price' | 'cache_write_12h_price' | 'cache_read_price' | 'output_price' | 'per_request_price'
const tokenFields = ['input_price', 'cache_write_price', 'cache_write_1h_price', 'cache_write_12h_price', 'cache_read_price', 'output_price'] as const
const summaryTokenFields = ['input_price', 'cache_write_price', 'cache_read_price', 'output_price'] as const
const cacheDurations = [{ field: 'cache_write_1h_price', label: '1h' }, { field: 'cache_write_12h_price', label: '12h' }] as const
const fieldLabel = (field: typeof tokenFields[number]) => ({
  input_price: text('输入', 'Input'),
  cache_write_price: text('缓存写 5m', 'Cache write 5m'),
  cache_write_1h_price: text('缓存写 1h', 'Cache write 1h'),
  cache_write_12h_price: text('缓存写 12h', 'Cache write 12h'),
  cache_read_price: text('缓存读', 'Cache read'),
  output_price: text('输出', 'Output')
})[field]

const detailModel = ref<PlazaModel | null>(null)
const groupSearch = ref('')
const groupNav = ref<HTMLElement | null>(null)
const groupScroll = ref({ above: false, below: false })
const capFilter = reactive({ reasoning: false, tools: false })

const connections = computed(() => {
  const result = new Map(props.availableConnections.map((group) => [group.id, group]))
  for (const group of props.groups) result.set(group.id, group)
  return [...result.values()].map((connection) => {
    const group = props.groups.find((g) => g.id === connection.id)
    const platform = (group?.platform || (connection as { platform?: string }).platform || group?.models?.[0]?.platform || 'openai') as any
    return { ...connection, platform, count: group?.models.length ?? 0, rate: group ? group.user_rate_multiplier ?? group.rate_multiplier : null }
  })
})

const connectionSelectOptions = computed(() =>
  connections.value.map((connection) => ({
    value: connection.id,
    label: `${connection.name} · ${connection.count}`
  }))
)

function onMobileGroupChange(value: string | number | boolean | null) {
  const id = Number(value)
  if (Number.isFinite(id) && id > 0) emit('update:selectedGroupId', id)
}

const visibleConnections = computed(() =>
  connections.value.filter((group) => group.name.toLowerCase().includes(groupSearch.value.trim().toLowerCase()))
)
const selectedGroup = computed(() => props.groups.find((group) => group.id === props.selectedGroupId))
const selectedPlatform = ref('')

const searchedModels = computed(() =>
  (selectedGroup.value?.models || []).filter((model) => {
    if (!matchesCatalogModel(model, props.search)) return false
    if (capFilter.reasoning && !model.reasoning) return false
    if (capFilter.tools && !model.tool_call) return false
    return true
  })
)

const platforms = computed(() => {
  const result = new Map<string, { id: string; label: string; models: PlazaModel[] }>()
  for (const model of searchedModels.value) {
    const vendor = modelVendor(model)
    if (!result.has(vendor.id)) result.set(vendor.id, { ...vendor, models: [] })
    result.get(vendor.id)!.models.push(model)
  }
  return [...result.values()]
})

const visiblePlatforms = computed(() =>
  platforms.value.filter((platform) => !selectedPlatform.value || platform.id === selectedPlatform.value)
)
const selectedCannotBind = computed(() => props.selectedGroupId != null && !props.bindableGroupIds.includes(props.selectedGroupId))
const vendorLabel = (vendor: { id: string; label: string }) => (vendor.id === 'other' ? text('其他模型', 'Other models') : vendor.label)
const compactTokens = (value: number) => new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
const emptyMessage = computed(() =>
  props.search.trim() || capFilter.reasoning || capFilter.tools
    ? text('没有找到匹配的模型。', 'No matching models.')
    : text('此分组暂未公布模型价格。', 'No model pricing is published for this group.')
)

function compareKeyFor(model: PlazaModel): string {
  return `${selectedGroup.value?.id ?? 0}::${model.platform}::${model.name}`
}

function isTokenPriced(model: PlazaModel): boolean {
  const p: any = model.pricing || model.official_pricing
  if (!p) return false
  if (model.pricing && model.pricing.billing_mode !== 'token') return false
  return Number(p.input_price ?? p.input_cost_per_token ?? 0) > 0 || Number(p.output_price ?? p.output_cost_per_token ?? 0) > 0
}

function isCompared(model: PlazaModel): boolean {
  return props.comparedKeys.includes(compareKeyFor(model))
}

function onToggleCompare(model: PlazaModel) {
  emit('toggle-compare', compareKeyFor(model))
}

const detailTiers = computed(() => (detailModel.value ? pricingTiers(detailModel.value) : []))
const officialTiers = computed<UserPricingInterval[]>(() => {
  const pricing = detailModel.value?.official_pricing
  if (!pricing) return []
  return pricing.intervals?.length
    ? [...pricing.intervals].sort((a, b) => a.min_tokens - b.min_tokens).map((tier) => resolveIntervalPrices(tier, pricing))
    : [{ min_tokens: 0, max_tokens: null, per_request_price: null, tier_label: text('基础价', 'Base'), ...pricing }]
})
const peakWindow = computed(() => formatPeakRateWindow(selectedGroup.value, serverTimezoneLabel(props.serverUtcOffset) || text('服务器时区', 'server timezone')))

watch(
  connections,
  (groups) => {
    if (!groups.some((group) => group.id === props.selectedGroupId)) {
      emit('update:selectedGroupId', groups.find((group) => group.count > 0)?.id ?? groups[0]?.id ?? null)
    }
  },
  { immediate: true }
)
watch(() => props.selectedGroupId, () => { detailModel.value = null; selectedPlatform.value = '' })
watch(() => props.search, () => { selectedPlatform.value = '' })

function updateGroupScroll() {
  const nav = groupNav.value
  groupScroll.value = nav && nav.clientHeight > 0
    ? { above: nav.scrollTop > 1, below: nav.scrollHeight - nav.clientHeight - nav.scrollTop > 1 }
    : { above: false, below: false }
}
function scrollConnections(direction: number) {
  const nav = groupNav.value
  if (nav) nav.scrollBy({ top: direction * nav.clientHeight * 0.75, behavior: 'auto' })
}
useResizeObserver(groupNav, updateGroupScroll)
watch([groupNav, visibleConnections], updateGroupScroll, { flush: 'post' })

function independentRate(model: PlazaModel) {
  return model.pricing?.rate_independent === true
    || (model.pricing?.billing_mode === 'image' && selectedGroup.value?.image_rate_independent === true)
    || (model.pricing?.billing_mode === 'video' && selectedGroup.value?.video_rate_independent === true)
}
function effectiveRate(model: PlazaModel): number | null {
  const group = selectedGroup.value
  if (!group) return null
  if (model.pricing?.rate_independent) return 1
  const rate = model.pricing?.billing_mode === 'image' && group.image_rate_independent
    ? group.image_rate_multiplier ?? 1
    : model.pricing?.billing_mode === 'video' && group.video_rate_independent
      ? group.video_rate_multiplier ?? 1
      : group.user_rate_multiplier ?? group.rate_multiplier
  return Number.isFinite(rate) && rate >= 0 ? rate : null
}
function pricingTiers(model: PlazaModel): UserPricingInterval[] {
  const pricing = model.pricing
  if (!pricing) return []
  const intervals = [...(pricing.intervals || [])].sort((a, b) => a.min_tokens - b.min_tokens)
  if (!intervals.length) return [{ min_tokens: 0, max_tokens: null, tier_label: text('基础价', 'Base'), ...pricing }]
  if (pricing.billing_mode !== 'token') return intervals.map((interval) => ({ ...interval, per_request_price: interval.per_request_price ?? pricing.per_request_price }))
  const effective = selectedGroup.value?.long_context_pricing_enabled === false ? intervals.slice(0, 1) : intervals
  return effective.map((interval) => resolveIntervalPrices(interval, pricing))
}
function priceDisplay(model: PlazaModel, values: (number | null | undefined)[]) {
  const known = values.filter((value): value is number => value != null && Number.isFinite(value) && value >= 0)
  const scale = model.pricing?.billing_mode === 'token' ? 1_000_000 : 1
  const formattedRange = (rate: number) => {
    if (!known.length) return '—'
    if (known.length !== values.length) return text('部分未定价', 'Partly unpriced')
    const low = formatScaled(Math.min(...known) * rate, scale, 2)
    const high = formatScaled(Math.max(...known) * rate, scale, 2)
    return low === high ? low : low + '–' + high
  }
  const rate = effectiveRate(model)
  const media = model.pricing?.billing_mode === 'image' || model.pricing?.billing_mode === 'video'
  return {
    base: formattedRange(1),
    adjusted: !media && rate != null && rate !== 1 && known.length === values.length && known.length > 0 ? formattedRange(rate) : undefined,
    adjustedLabel: rate == null ? undefined : `${rate < 1 ? text('折后', 'Discounted') : text('倍率价', 'Adjusted')} ×${rate}`
  }
}
function rangePrice(model: PlazaModel, field: PriceField) {
  return priceDisplay(model, pricingTiers(model).map((tier) => tier[field]))
}
function hasCacheDuration(model: PlazaModel, field: 'cache_write_1h_price' | 'cache_write_12h_price') {
  return pricingTiers(model).some((tier) => tier[field] != null)
}
function billingLabel(model: PlazaModel) {
  const meteredLabels: Record<string, string> = {
    call: text('按次', 'Per call'),
    page: text('按页', 'Per page'),
    hour: text('按小时（按秒计）', 'Per hour (billed per second)'),
    '10k_characters': text('按万计费字符', 'Per 10k billable characters')
  }
  if (model.pricing?.metered_unit && meteredLabels[model.pricing.metered_unit]) return meteredLabels[model.pricing.metered_unit]
  const mode = model.pricing?.billing_mode
  if (mode === 'token') return text('按 Token', 'Per token')
  if (mode === 'image') return text('按图片', 'Per image')
  if (mode === 'video') return text('按视频请求', 'Per video request')
  return text('按请求', 'Per request')
}
function unitLabel(model: PlazaModel) {
  const meteredLabels: Record<string, string> = {
    call: text('/ 次', '/ call'),
    page: text('/ 页', '/ page'),
    hour: text('/ 小时（按秒计）', '/ hour (billed per second)'),
    '10k_characters': text('/ 万计费字符', '/ 10k billable characters')
  }
  if (model.pricing?.metered_unit && meteredLabels[model.pricing.metered_unit]) return meteredLabels[model.pricing.metered_unit]
  return model.pricing?.billing_mode === 'image' ? text('/ 张', '/ image') : text('/ 次', '/ request')
}
function tierLabel(tier: UserPricingInterval) {
  if (tier.tier_label) return tier.tier_label
  return tier.max_tokens == null ? '>' + tier.min_tokens.toLocaleString() : tier.min_tokens.toLocaleString() + '–' + tier.max_tokens.toLocaleString()
}
function reasoningRates(model: PlazaModel): [string, number][] {
  return Object.entries(model.pricing?.reasoning_effort_multipliers || {}).filter(([, value]) => Number.isFinite(value) && value >= 0)
}
function modelBadges(model: PlazaModel) {
  return [
    pricingTiers(model).length > 1 ? text('阶梯', 'Tiers') : '',
    model.time_pricing?.periods.length ? text('分时', 'Scheduled') : '',
    hasPeakRate(selectedGroup.value) ? text('高峰', 'Peak') : '',
    reasoningRates(model).some(([, value]) => value !== 1) ? text('推理倍率', 'Reasoning') : ''
  ].filter(Boolean)
}
function chooseExampleModel() {
  if (detailModel.value) emit('select-model', detailModel.value.name)
  detailModel.value = null
}
</script>
