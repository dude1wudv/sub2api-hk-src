<template>
  <UiDrawer
    :open="open"
    :title="drawerTitle"
    :subtitle="drawerSubtitle"
    width="lg"
    :close-on-backdrop="!saving"
    :close-on-escape="!saving"
    @close="!saving && handleClose()"
  >
    <!-- Creation Success State -->
    <div v-if="createdKey" class="space-y-5" data-test="key-created-success">
      <div class="rounded-xl border border-[var(--atelier-ok)]/25 bg-[var(--atelier-ok-bg)] p-4">
        <div class="flex items-center gap-2 text-sm font-medium text-[var(--atelier-ok)]">
          <UiStatusDot tone="ok" />
          <span>{{ text('密钥已成功创建，请立即保存完整密钥', 'API key created — copy and store it safely') }}</span>
        </div>
        <p class="mt-1 text-xs text-[var(--atelier-ink-secondary)]">
          {{ text('为保障账户安全，完整密钥仅在此时展示一次。', 'For your security, the full secret is highlighted here right after creation.') }}
        </p>
      </div>

      <!-- One-time Full Key Box -->
      <div class="space-y-2">
        <div class="flex items-center justify-between text-xs">
          <span class="font-medium text-[var(--atelier-ink-secondary)]">{{ createdKey.name }}</span>
          <span v-if="createdGroupName" class="font-mono text-[var(--atelier-ink-muted)]">{{ createdGroupName }}</span>
        </div>
        <div class="flex items-center gap-2 rounded-lg border border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] p-2.5">
          <code class="min-w-0 flex-1 break-all font-mono text-xs text-[var(--atelier-ink)]" data-test="created-key-secret">
            {{ createdKey.key }}
          </code>
          <UiButton variant="primary" size="sm" @click="copyCreatedSecret">
            <Icon :name="copiedSecret ? 'check' : 'copy'" size="xs" aria-hidden="true" />
            {{ copiedSecret ? text('已复制', 'Copied') : text('复制密钥', 'Copy key') }}
          </UiButton>
        </div>
      </div>

      <!-- Tabbed Quick-Start Snippets -->
      <div class="space-y-3 border-t border-[var(--atelier-border)] pt-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">
            {{ text('快速接入代码', 'Quick-start snippets') }}
          </h3>
          <UiButton variant="ghost" size="sm" @click="copySnippet">
            <Icon :name="copiedSnippet ? 'check' : 'copy'" size="xs" aria-hidden="true" />
            {{ copiedSnippet ? text('已复制代码', 'Copied') : text('复制代码', 'Copy snippet') }}
          </UiButton>
        </div>

        <div class="overflow-x-auto pb-1">
          <UiSegmented
            v-model="snippetTab"
            :options="snippetOptions"
            size="sm"
            :aria-label="text('接入方式', 'Client snippet')"
          />
        </div>

        <pre class="max-h-64 overflow-auto rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] p-3.5 font-mono text-xs leading-relaxed text-[var(--atelier-ink)]"><code>{{ activeSnippet }}</code></pre>
      </div>
    </div>

    <!-- Create / Edit Key Form -->
    <form v-else :id="formId" class="key-editor space-y-5" @submit.prevent="submit">
      <p
        v-if="error"
        role="alert"
        class="rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]"
      >
        {{ error }}
      </p>

      <!-- Step 1: Key Name & Mode -->
      <div class="space-y-3">
        <label class="block space-y-1.5">
          <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
            {{ text('密钥名称', 'Name') }} <span class="text-[var(--atelier-err)]">*</span>
          </span>
          <input
            v-model="form.name"
            class="atelier-input"
            required
            maxlength="100"
            :placeholder="text('为密钥起一个便于识别的名称', 'A memorable name for this key')"
            data-test="key-name"
            data-tour="key-form-name"
          />
        </label>
      </div>

      <!-- Step 2: Group / Smart Routing Selection -->
      <section class="space-y-3 border-t border-[var(--atelier-border)] pt-4" data-tour="key-form-group" :aria-label="text('选择分组', 'Choose a group')">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-sm font-semibold text-[var(--atelier-ink)]">
            {{ text('选择接入分组', 'Choose a group') }}
          </h3>
          <div class="key-mode inline-flex rounded-lg border border-[var(--atelier-border-subtle)] bg-[var(--atelier-subtle)] p-0.5" role="group" :aria-label="text('接入模式', 'Connection mode')">
            <button
              type="button"
              class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
              :class="!form.smart ? 'bg-[var(--atelier-surface)] text-[var(--atelier-ink)] shadow-xs' : 'text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)]'"
              :aria-pressed="!form.smart"
              @click="setMode(false)"
            >
              {{ text('固定分组', 'Fixed group') }}
            </button>
            <button
              type="button"
              class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
              :class="form.smart ? 'bg-[var(--atelier-surface)] text-[var(--atelier-ink)] shadow-xs' : 'text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)]'"
              :aria-pressed="form.smart"
              @click="setMode(true)"
            >
              {{ text('智能路由', 'Smart routing') }}
            </button>
          </div>
        </div>

        <p v-if="form.smart" class="text-xs leading-relaxed text-[var(--atelier-ink-muted)]">
          {{ text('点击下方分组添加候选，最多 10 个，按实际命中的分组计费。', 'Select up to 10 candidate groups; billing follows the serving group.') }}
        </p>

        <!-- Smart Routing Order List -->
        <div v-if="form.smart" class="key-routing space-y-2 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] p-3">
          <div class="flex items-center justify-between text-xs">
            <h4 class="font-medium text-[var(--atelier-ink)]">{{ text('候选分组顺序', 'Routing order') }}</h4>
            <span class="font-mono tabular-nums text-[var(--atelier-ink-muted)]">{{ form.routing.length }} / 10</span>
          </div>
          <ol v-if="form.routing.length" class="routing-order space-y-1.5">
            <li
              v-for="(id, index) in form.routing"
              :key="id"
              class="flex items-center gap-2 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-2.5 py-2 text-xs"
              :class="{ 'border-[var(--atelier-err)]/35 bg-[var(--atelier-err-bg)]': !routingGroups.some(group => group.id === id) }"
            >
              <span class="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded bg-[var(--atelier-subtle)] font-mono text-[11px] font-semibold text-[var(--atelier-ink)]">
                {{ index + 1 }}
              </span>
              <BrandPlatformIcon
                v-if="groups.find(group => group.id === id)"
                :platform="groups.find(group => group.id === id)!.platform"
                size="xs"
              />
              <div class="min-w-0 flex-1">
                <strong class="block truncate font-medium text-[var(--atelier-ink)]">
                  {{ groups.find(group => group.id === id)?.name || `#${id}` }}
                </strong>
                <small v-if="!routingGroups.some(group => group.id === id)" class="text-[11px] text-[var(--atelier-err)]">
                  {{ text('不可用，请移除或替换', 'Unavailable — remove or replace') }}
                </small>
                <small v-else class="text-[11px] text-[var(--atelier-ink-muted)]">
                  {{ index === 0 ? text('同分时首选', 'First when tied') : text('回退候选', 'Fallback group') }}
                </small>
              </div>
              <div class="flex shrink-0 items-center gap-0.5">
                <button
                  type="button"
                  class="inline-flex h-6 w-6 items-center justify-center rounded text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)] disabled:opacity-30"
                  :disabled="index === 0"
                  :aria-label="text(`上移第${index + 1}个分组`, `Move group ${index + 1} up`)"
                  @click="moveRouting(index, -1)"
                >↑</button>
                <button
                  type="button"
                  class="inline-flex h-6 w-6 items-center justify-center rounded text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)] disabled:opacity-30"
                  :disabled="index === form.routing.length - 1"
                  :aria-label="text(`下移第${index + 1}个分组`, `Move group ${index + 1} down`)"
                  @click="moveRouting(index, 1)"
                >↓</button>
                <button
                  type="button"
                  class="inline-flex h-6 w-6 items-center justify-center rounded text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-err-bg)] hover:text-[var(--atelier-err)]"
                  :aria-label="text(`移除第${index + 1}个分组`, `Remove group ${index + 1}`)"
                  @click="removeRouting(index)"
                >×</button>
              </div>
            </li>
          </ol>
          <p v-else class="py-2 text-center text-xs text-[var(--atelier-ink-muted)]">
            {{ text('点击下方分组添加路由候选。', 'Add routing candidates from the group list below.') }}
          </p>
        </div>

        <!-- Fixed Group Selected Summary -->
        <div v-else-if="selectedGroup" class="key-selected-summary rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] p-3 text-xs">
          <div class="flex items-center justify-between gap-2">
            <span class="flex min-w-0 items-center gap-2">
              <BrandPlatformIcon :platform="selectedGroup.platform" size="sm" />
              <strong class="truncate text-sm font-medium text-[var(--atelier-ink)]">{{ selectedGroup.name }}</strong>
            </span>
            <span class="rounded bg-[var(--atelier-surface)] px-2 py-0.5 font-mono text-xs font-semibold tabular-nums text-[var(--atelier-ink)]">
              {{ rate(selectedGroup) }}×
            </span>
          </div>
          <small v-if="selectedGroup.description" class="mt-1.5 block whitespace-pre-line leading-relaxed text-[var(--atelier-ink-secondary)]">
            {{ selectedGroup.description }}
          </small>
          <small v-if="selectedGroup.peak_rate_enabled" class="mt-1 block font-mono text-[11px] text-[var(--atelier-warn)]">
            {{ peakText(selectedGroup) }}
          </small>
        </div>

        <!-- Group Search & Filters -->
        <div v-if="eligibleGroups.length > 1" class="space-y-2">
          <div class="relative">
            <Icon name="search" size="xs" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
            <input
              v-model="search"
              type="search"
              class="atelier-input !pl-8 !min-h-[34px] text-xs"
              :aria-label="text('搜索分组', 'Search groups')"
              :placeholder="text('搜索分组名称、关键词或说明', 'Search group names, tags or descriptions')"
            />
          </div>
          <div class="key-providers flex flex-wrap gap-1.5" role="group" :aria-label="text('分组类别', 'Group categories')">
            <button
              v-for="provider in providers"
              :key="provider.value"
              type="button"
              class="inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors"
              :class="category === provider.value ? 'border-[var(--atelier-ink)] bg-[var(--atelier-ink)] text-[var(--atelier-ink-inverse)]' : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] text-[var(--atelier-ink-secondary)] hover:border-[var(--atelier-border-strong)] hover:text-[var(--atelier-ink)]'"
              :aria-pressed="category === provider.value"
              @click="category = provider.value"
            >
              <span>{{ provider.label }}</span>
              <span class="font-mono text-[11px] opacity-75">{{ provider.count }}</span>
            </button>
          </div>
          <div v-if="hasQuickFilters" class="flex flex-wrap gap-1.5" role="group" :aria-label="text('快捷筛选', 'Quick filters')">
            <button
              v-if="priceDropCount > 0"
              type="button"
              class="inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-[11.5px] font-medium transition-colors"
              :class="quickFilter === 'drop' ? 'border-[var(--atelier-ok)] bg-[var(--atelier-ok)] text-white' : 'border-[var(--atelier-ok)]/30 bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]'"
              :aria-pressed="quickFilter === 'drop'"
              @click="toggleQuickFilter('drop')"
            >
              ↓ {{ text('近期降价', 'Price drop') }} <span class="font-mono">{{ priceDropCount }}</span>
            </button>
            <button
              v-if="valueGroupCount > 0"
              type="button"
              class="inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-[11.5px] font-medium transition-colors"
              :class="quickFilter === 'value' ? 'border-[var(--atelier-warn)] bg-[var(--atelier-warn)] text-white' : 'border-[var(--atelier-warn)]/30 bg-[var(--atelier-warn-bg)] text-[var(--atelier-warn)]'"
              :aria-pressed="quickFilter === 'value'"
              @click="toggleQuickFilter('value')"
            >
              {{ text('超值低倍', 'Best value') }} <span class="font-mono">{{ valueGroupCount }}</span>
            </button>
            <button
              v-if="premiumGroupCount > 0"
              type="button"
              class="inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-[11.5px] font-medium transition-colors"
              :class="quickFilter === 'premium' ? 'border-[var(--atelier-ink)] bg-[var(--atelier-ink)] text-[var(--atelier-ink-inverse)]' : 'border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] text-[var(--atelier-ink)]'"
              :aria-pressed="quickFilter === 'premium'"
              @click="toggleQuickFilter('premium')"
            >
              {{ text('优质满血', 'Premium') }} <span class="font-mono">{{ premiumGroupCount }}</span>
            </button>
          </div>
        </div>

        <!-- Group Cards List -->
        <div class="max-h-56 space-y-1.5 overflow-y-auto pr-1">
          <button
            v-for="group in filteredGroups"
            :key="group.id"
            type="button"
            class="key-group-card flex w-full items-start gap-2.5 rounded-xl border px-3 py-2.5 text-left transition-colors"
            :class="isSelected(group.id) ? 'selected border-[var(--atelier-ink)] bg-[var(--atelier-selected-row)]' : 'border-[var(--atelier-border)] bg-[var(--atelier-surface)] hover:border-[var(--atelier-border-strong)] hover:bg-[var(--atelier-hover)]'"
            :aria-pressed="isSelected(group.id)"
            :disabled="form.smart && (form.routing.includes(group.id) || form.routing.length >= 10)"
            :data-group-id="group.id"
            @click="selectGroup(group.id)"
          >
            <span class="mt-0.5 shrink-0">
              <BrandPlatformIcon :platform="group.platform" size="sm" />
            </span>
            <span class="min-w-0 flex-1 space-y-1">
              <span class="flex items-center gap-1.5">
                <UiStatusDot :tone="healthTone(groupHealth[group.id] ?? 'unknown')" />
                <strong class="truncate text-sm font-medium text-[var(--atelier-ink)]">{{ group.name }}</strong>
              </span>
              <span v-if="groupBadges(group).length || group.is_new || hasRateChange(group)" class="flex flex-wrap items-center gap-1">
                <GroupChangeIndicator v-if="group.is_new || hasRateChange(group)" :group="group" />
                <span
                  v-for="tag in groupBadges(group)"
                  :key="tag.label"
                  class="rounded-full px-1.5 py-0.5 text-[10.5px] font-medium"
                  :class="tag.tone === 'caution' ? 'bg-[var(--atelier-err-bg)] text-[var(--atelier-err)]' : tag.tone === 'deal' ? 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]' : 'bg-[var(--atelier-subtle)] text-[var(--atelier-ink-secondary)]'"
                >
                  {{ tag.label }}
                </span>
              </span>
              <span v-if="group.description" class="key-description block whitespace-pre-line text-xs leading-relaxed text-[var(--atelier-ink-secondary)]">
                {{ group.description }}
              </span>
            </span>
            <span class="shrink-0 text-right font-mono tabular-nums">
              <span class="block text-xs font-semibold text-[var(--atelier-ink)]">{{ rate(group) }}×</span>
              <small v-if="hasUserRate(group)" class="block text-[10.5px] text-[var(--atelier-ok)]">{{ text('专属倍率', 'Your rate') }}</small>
              <small v-if="group.peak_rate_enabled" class="block text-[10.5px] text-[var(--atelier-warn)]">{{ peakText(group) }}</small>
            </span>
          </button>
          <p v-if="!filteredGroups.length" class="py-6 text-center text-xs text-[var(--atelier-ink-muted)]">
            {{ text('没有匹配的可用分组', 'No matching groups') }}
          </p>
        </div>
      </section>

      <!-- Step 3: Expiry (UiSegmented) & Quota (UiSlider + Exact Input) -->
      <section class="key-common-fields space-y-4 border-t border-[var(--atelier-border)] pt-4">
        <!-- Create mode: Expiry Segmented Presets + Days Input -->
        <div v-if="!editing" class="space-y-2">
          <div class="flex items-center justify-between gap-2">
            <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ text('有效期', 'Expiry') }}</span>
            <UiSegmented
              v-model="expiryPreset"
              :options="expiryPresetOptions"
              size="sm"
              :aria-label="text('有效期预设', 'Expiry preset')"
              @update:model-value="onExpiryPreset"
            />
          </div>
          <label class="block space-y-1">
            <input
              v-model.number="form.days"
              type="number"
              min="1"
              step="1"
              class="atelier-input font-mono tabular-nums"
              :placeholder="text('永久有效（留空）', 'Never expires (leave empty)')"
              data-test="key-days"
              @input="syncExpiryPresetFromDays"
            />
            <small class="block text-[11px] text-[var(--atelier-ink-muted)]">
              {{ text('有效天数，留空为永久有效', 'Days until expiry; empty means never') }}
            </small>
          </label>
        </div>

        <!-- Edit mode: Exact Datetime -->
        <label v-else class="block space-y-1.5">
          <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ text('有效期', 'Expiry') }}</span>
          <input v-model="form.expiry" type="datetime-local" class="atelier-input font-mono tabular-nums" />
          <small class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ text('留空为永久有效', 'Empty means never expires') }}</small>
        </label>

        <!-- Quota (USD) with UiSlider + Exact Input -->
        <div class="space-y-2">
          <div class="flex items-center justify-between gap-2">
            <label for="key-quota-input" class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ text('额度（USD）', 'Quota (USD)') }}
            </label>
            <span class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
              {{ Number(form.quota) > 0 ? `$${Number(form.quota).toFixed(2)}` : text('不限额度', 'Unlimited') }}
            </span>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-12 sm:items-center">
            <div class="sm:col-span-7">
              <UiSlider
                :model-value="Math.min(500, Math.max(0, Number(form.quota) || 0))"
                :min="0"
                :max="500"
                :step="5"
                :aria-label="text('快捷额度滑块', 'Quick quota slider')"
                @update:model-value="(val) => { form.quota = val }"
              />
            </div>
            <div class="sm:col-span-5">
              <input
                id="key-quota-input"
                v-model.number="form.quota"
                type="number"
                min="0"
                step="0.01"
                required
                class="atelier-input font-mono tabular-nums"
                data-test="key-quota"
              />
            </div>
          </div>
          <small class="block text-[11px] text-[var(--atelier-ink-muted)]">{{ text('0 为不限额度', '0 means unlimited') }}</small>
        </div>

        <!-- Concurrency Limit -->
        <label class="block space-y-1.5">
          <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
            {{ text('Key 并发上限', 'Key concurrency limit') }}
          </span>
          <input
            v-model.number="form.concurrency"
            type="number"
            min="0"
            step="1"
            required
            class="atelier-input font-mono tabular-nums"
            data-test="key-concurrency"
          />
          <small class="block text-[11px] text-[var(--atelier-ink-muted)]">
            {{ text('0 为不额外限制，仍受用户和账号并发限制。', '0 adds no key limit; user and account limits still apply.') }}
          </small>
        </label>
      </section>

      <!-- Step 4: Advanced Settings -->
      <details class="key-advanced rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-4">
        <summary class="cursor-pointer py-3 text-xs font-semibold text-[var(--atelier-ink)]">
          {{ text('高级设置（缓存策略、窗口限流、IP 限制、自定义密钥）', 'Advanced settings (cache TTL, rate limits, IP rules)') }}
        </summary>
        <div class="space-y-4 pb-4 pt-1">
          <!-- Anthropic Cache TTL via UiSelect -->
          <div class="space-y-1.5">
            <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ text('Anthropic 缓存时长', 'Anthropic cache duration') }}
            </span>
            <UiSelect
              :model-value="form.cacheTtl"
              :options="cacheTtlOptions"
              :aria-label="text('Anthropic 缓存时长', 'Anthropic cache duration')"
              data-test="anthropic-cache-ttl"
              @update:model-value="(val) => { form.cacheTtl = String(val || 'inherit') as NonNullable<ApiKey['anthropic_cache_ttl_mode']> }"
            />
            <small class="block text-[11px] text-[var(--atelier-ink-muted)]">
              {{ text('仅对原生 Anthropic 生效（不包括 Bedrock）；关闭覆盖时按上游返回的用量计费。', 'Applies only to native Anthropic, excluding Bedrock. Disabling the override uses upstream usage.') }}
            </small>
          </div>

          <!-- Window Rate Limits -->
          <div class="space-y-1.5">
            <div class="grid grid-cols-3 gap-2.5">
              <label v-for="field in rateFields" :key="field.key" class="block space-y-1">
                <span class="block text-[11.5px] font-medium text-[var(--atelier-ink-secondary)]">{{ field.label }}</span>
                <input
                  v-model.number="form[field.key]"
                  type="number"
                  min="0"
                  step="0.01"
                  required
                  class="atelier-input !min-h-[34px] font-mono text-xs tabular-nums"
                />
              </label>
            </div>
            <p class="text-[11px] text-[var(--atelier-ink-muted)]">
              {{ text('窗口限额按 USD 计算，0 为不限。', 'Window limits are in USD; 0 means unlimited.') }}
            </p>
          </div>

          <!-- Reset Flags (Edit mode) -->
          <div v-if="editing" class="key-resets space-y-2 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-surface)] p-3 text-xs">
            <label class="flex cursor-pointer items-center gap-2 text-[var(--atelier-ink)]">
              <input v-model="form.resetQuota" type="checkbox" class="h-4 w-4 accent-[var(--atelier-ink)]" />
              <span>{{ text('重置已用额度', 'Reset used quota') }}</span>
            </label>
            <label class="flex cursor-pointer items-center gap-2 text-[var(--atelier-ink)]">
              <input v-model="form.resetRates" type="checkbox" class="h-4 w-4 accent-[var(--atelier-ink)]" />
              <span>{{ text('重置限流计数', 'Reset rate-limit usage') }}</span>
            </label>
            <small class="block text-[11px] text-[var(--atelier-ink-muted)]">
              {{ text('仅勾选后保存才会重置。', 'Resets apply only when selected and saved.') }}
            </small>
          </div>

          <!-- IP Restrictions -->
          <div class="space-y-2">
            <UiSwitch
              v-model="ipRulesEnabled"
              :label="text('配置 IP 白名单 / 黑名单限制', 'Configure IP allowlist / blocklist')"
            />
            <div v-show="ipRulesEnabled || form.allow || form.block" class="space-y-2.5 pt-1">
              <label class="block space-y-1">
                <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
                  {{ text('IP 白名单（每行一个 IP 或 CIDR）', 'IP allowlist (one IP/CIDR per line)') }}
                </span>
                <textarea v-model="form.allow" rows="2" class="atelier-input font-mono text-xs" />
              </label>
              <label class="block space-y-1">
                <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
                  {{ text('IP 黑名单（每行一个 IP 或 CIDR）', 'IP blocklist (one IP/CIDR per line)') }}
                </span>
                <textarea v-model="form.block" rows="2" class="atelier-input font-mono text-xs" />
              </label>
            </div>
          </div>

          <!-- Custom Key (Create mode) -->
          <label v-if="!editing" class="block space-y-1">
            <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ text('自定义密钥（可选）', 'Custom key (optional)') }}
            </span>
            <input v-model="form.custom" type="password" autocomplete="off" class="atelier-input font-mono text-xs" />
            <small class="block text-[11px] text-[var(--atelier-ink-muted)]">
              {{ text('至少16位，仅支持字母、数字、下划线和连字符。', 'At least 16 characters: letters, numbers, underscores or hyphens.') }}
            </small>
          </label>
        </div>
      </details>
    </form>

    <!-- Drawer Footer -->
    <template #footer>
      <template v-if="createdKey">
        <UiButton variant="primary" size="md" @click="handleClose">
          {{ text('完成', 'Done') }}
        </UiButton>
      </template>
      <template v-else>
        <div class="mr-auto min-w-0 truncate text-xs text-[var(--atelier-ink-secondary)]" aria-live="polite">
          <template v-if="form.smart">
            {{ text(`已选择 ${form.routing.length} 个路由分组`, `${form.routing.length} routing groups selected`) }}
          </template>
          <template v-else-if="selectedGroup">
            <strong class="font-medium text-[var(--atelier-ink)]">{{ selectedGroup.name }}</strong>
            <span class="ml-1.5 font-mono tabular-nums">{{ rate(selectedGroup) }}×</span>
          </template>
          <template v-else>{{ text('请选择分组', 'Select a group') }}</template>
        </div>
        <UiButton variant="secondary" size="md" :disabled="saving" @click="handleClose">
          {{ text('取消', 'Cancel') }}
        </UiButton>
        <UiButton
          type="submit"
          :form="formId"
          variant="primary"
          size="md"
          :loading="saving"
          data-tour="key-form-submit"
        >
          {{ saving ? text('保存中…', 'Saving…') : editing ? text('保存修改', 'Save changes') : text('确认并创建', 'Create API key') }}
        </UiButton>
      </template>
    </template>
  </UiDrawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref, toRef, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getActivePinia } from 'pinia'
import {
  UiDrawer,
  UiButton,
  UiSegmented,
  UiSelect,
  UiSlider,
  UiSwitch,
  UiStatusDot,
  BrandPlatformIcon,
  type SegmentedOption
} from '@/ui'
import Icon from '@/components/icons/Icon.vue'
import GroupChangeIndicator from '@/components/common/GroupChangeIndicator.vue'
import { useGroupHealth } from '@/composables/useGroupHealth'
import { useClipboard } from '@/composables/useClipboard'
import { userFacingError } from '@/utils/userFacingError'
import { keysAPI } from '@/api/keys'
import { useOnboardingStore } from '@/stores/onboarding'
import { isSmartRoutingGroup } from '@/utils/smartRouting'
import { getKeyGroupProvider, type KeyGroupProvider } from '@/utils/keyGroupProviders'
import type { ApiKey, Group } from '@/types'

const props = withDefaults(
  defineProps<{
    open: boolean
    editing: ApiKey | null
    groups: Group[]
    selectedGroup?: number
    groupRates?: Record<number, number>
    baseUrl?: string
  }>(),
  {
    groupRates: () => ({}),
    baseUrl: ''
  }
)

const emit = defineEmits<{ close: []; saved: [key: ApiKey] }>()

const groupHealth = useGroupHealth(toRef(props, 'open'))
const formId = useId()
const { locale } = useI18n()
const onboardingStore = useOnboardingStore()
const text = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en)

const saving = ref(false)
const error = ref('')
const search = ref('')
const category = ref<'all' | KeyGroupProvider>('all')
const quickFilter = ref<'all' | 'drop' | 'value' | 'premium'>('all')
const ipRulesEnabled = ref(false)
const createdKey = ref<ApiKey | null>(null)
const copiedSecret = ref(false)
const copiedSnippet = ref(false)

type ExpiryPreset = 'never' | '7' | '30' | '90' | 'custom'
const expiryPreset = ref<ExpiryPreset>('never')

const expiryPresetOptions = computed<SegmentedOption<ExpiryPreset>[]>(() => [
  { value: 'never', label: text('永久', 'Never') },
  { value: '7', label: text('7天', '7d') },
  { value: '30', label: text('30天', '30d') },
  { value: '90', label: text('90天', '90d') }
])

type SnippetTab = 'curl' | 'python' | 'openai' | 'claude' | 'codex'
const snippetTab = ref<SnippetTab>('curl')
const snippetOptions: SegmentedOption<SnippetTab>[] = [
  { value: 'curl', label: 'cURL' },
  { value: 'python', label: 'Python' },
  { value: 'openai', label: 'OpenAI SDK' },
  { value: 'claude', label: 'Claude Code' },
  { value: 'codex', label: 'Codex' }
]

const drawerTitle = computed(() => {
  if (createdKey.value) return text('API Key 已创建', 'API key created')
  return props.editing ? text('编辑 API Key', 'Edit API key') : text('创建 API Key', 'Create API key')
})

const drawerSubtitle = computed(() => {
  if (createdKey.value) return text('复制密钥或直接粘贴接入代码即可开始调用', 'Copy your secret or grab a ready-to-run client snippet')
  return props.editing
    ? text('调整分组绑定、限额、并发及安全规则', 'Update group binding, quota, concurrency, and security rules')
    : text('配置接入分组、额度与有效期', 'Configure group binding, quota, and expiration')
})

const normalizedRoot = computed(() => {
  const raw = (props.baseUrl || (typeof window !== 'undefined' ? window.location.origin : 'https://api.example.com'))
    .replace(/\/+$/, '')
    .replace(/(?:\/v1)+$/, '')
  return raw || 'https://api.example.com'
})

const createdGroupName = computed(() => {
  const gid = createdKey.value?.group_id
  return props.groups.find((g) => g.id === gid)?.name || createdKey.value?.group?.name || ''
})

const activeSnippet = computed(() => {
  const key = createdKey.value?.key || 'sk-...'
  const root = normalizedRoot.value
  const v1 = `${root}/v1`
  switch (snippetTab.value) {
    case 'python':
      return `import requests\n\nresp = requests.post(\n    "${v1}/chat/completions",\n    headers={"Authorization": "Bearer ${key}"},\n    json={\n        "model": "gpt-4o-mini",\n        "messages": [{"role": "user", "content": "Hello!"}],\n    },\n)\nprint(resp.json())`
    case 'openai':
      return `import OpenAI from "openai";\n\nconst client = new OpenAI({\n  apiKey: "${key}",\n  baseURL: "${v1}",\n});\n\nconst res = await client.chat.completions.create({\n  model: "gpt-4o-mini",\n  messages: [{ role: "user", content: "Hello!" }],\n});\nconsole.log(res.choices[0].message.content);`
    case 'claude':
      return `export ANTHROPIC_BASE_URL="${root}"\nexport ANTHROPIC_AUTH_TOKEN="${key}"\nexport CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1\nclaude`
    case 'codex':
      return `export OPENAI_API_KEY="${key}"\nexport OPENAI_BASE_URL="${v1}"\ncodex`
    default:
      return `curl ${v1}/chat/completions \\\n  -H "Content-Type: application/json" \\\n  -H "Authorization: Bearer ${key}" \\\n  -d '{\n    "model": "gpt-4o-mini",\n    "messages": [{"role": "user", "content": "Hello!"}]\n  }'`
  }
})

async function safeCopy(value: string): Promise<boolean> {
  if (getActivePinia()) {
    return useClipboard().copyToClipboard(value)
  }
  try {
    await navigator.clipboard?.writeText(value)
    return true
  } catch {
    return false
  }
}

async function copyCreatedSecret() {
  if (!createdKey.value?.key) return
  if (await safeCopy(createdKey.value.key)) {
    copiedSecret.value = true
    setTimeout(() => { copiedSecret.value = false }, 1800)
  }
}

async function copySnippet() {
  if (await safeCopy(activeSnippet.value)) {
    copiedSnippet.value = true
    setTimeout(() => { copiedSnippet.value = false }, 1800)
  }
}

function handleClose() {
  createdKey.value = null
  emit('close')
}

const routingGroups = computed(() => props.groups.filter(isSmartRoutingGroup))
const form = reactive({
  name: '',
  group: '',
  smart: false,
  quota: 0,
  concurrency: 0,
  cacheTtl: 'inherit' as NonNullable<ApiKey['anthropic_cache_ttl_mode']>,
  days: '' as number | string,
  expiry: '',
  limit5h: 0,
  limit1d: 0,
  limit7d: 0,
  allow: '',
  block: '',
  custom: '',
  routing: [] as number[],
  resetQuota: false,
  resetRates: false
})

const cacheTtlOptions = computed(() => [
  { value: 'inherit', label: text('跟随账号与全局设置', 'Follow account and global settings') },
  { value: 'off', label: text('关闭覆盖', 'Disable override') },
  { value: '5m', label: text('强制 5 分钟', 'Force 5 minutes') },
  { value: '1h', label: text('强制 1 小时', 'Force 1 hour') }
])

const eligibleGroups = computed(() => (form.smart ? routingGroups.value : props.groups))
const selectedGroup = computed(() => props.groups.find((group) => String(group.id) === form.group))
const providers = computed(() =>
  [
    { value: 'all' as const, label: text('全部', 'All'), count: eligibleGroups.value.length },
    ...(['anthropic', 'openai', 'domestic', 'other'] as const).map((value) => ({
      value,
      label: { anthropic: 'Claude', openai: 'OpenAI', domestic: text('国产模型', 'Chinese models'), other: text('其他', 'Other') }[value],
      count: eligibleGroups.value.filter((group) => getKeyGroupProvider(group.platform) === value).length
    }))
  ].filter((provider) => provider.value === 'all' || provider.count > 0)
)

const PREMIUM_RE = /企业|满血|官方|高质量|满缓存|promax|enterprise|official|high[- ]?quality/i
const DEAL_RE = /特价|优惠|折扣|福利|免费|流口水|deal|promo|discount|free/i
const CAUTION_RE = /不可外接|自用|测试|无售后|非满血|internal|private|test/i

function healthTone(status: string): 'ok' | 'warn' | 'err' | 'neutral' {
  if (status === 'healthy') return 'ok'
  if (status === 'warning') return 'warn'
  if (status === 'critical') return 'err'
  return 'neutral'
}

function rate(group: Group) { return props.groupRates[group.id] ?? group.rate_multiplier ?? 1 }
function hasUserRate(group: Group) { return props.groupRates[group.id] != null && props.groupRates[group.id] !== group.rate_multiplier }
function hasRateChange(group: Group) { return Boolean(group.recent_rate_change && group.subscription_type !== 'subscription') }
function isPriceDrop(group: Group) { return Boolean(hasRateChange(group) && group.recent_rate_change && group.recent_rate_change.new_rate < group.recent_rate_change.old_rate) }
function isValueGroup(group: Group) { return rate(group) <= 0.3 || DEAL_RE.test(`${group.name} ${group.description || ''}`) }
function isPremiumGroup(group: Group) {
  const haystack = `${group.name} ${group.description || ''}`
  return PREMIUM_RE.test(haystack) && !CAUTION_RE.test(haystack)
}

function groupBadges(group: Group): Array<{ label: string; tone: 'premium' | 'deal' | 'caution' }> {
  const haystack = `${group.name} ${group.description || ''}`
  const tags: Array<{ label: string; tone: 'premium' | 'deal' | 'caution' }> = []
  if (CAUTION_RE.test(haystack)) {
    tags.push({ label: text('受限 / 内部', 'Restricted'), tone: 'caution' })
  } else if (PREMIUM_RE.test(haystack)) {
    tags.push({ label: text('优质优选', 'Premium'), tone: 'premium' })
  }
  if (rate(group) <= 0.25 || DEAL_RE.test(haystack)) {
    tags.push({ label: rate(group) <= 0.15 ? text('超低倍率', 'Ultra low') : text('特价优选', 'Great value'), tone: 'deal' })
  }
  return tags
}

const priceDropCount = computed(() => eligibleGroups.value.filter(isPriceDrop).length)
const valueGroupCount = computed(() => eligibleGroups.value.filter(isValueGroup).length)
const premiumGroupCount = computed(() => eligibleGroups.value.filter(isPremiumGroup).length)
const hasQuickFilters = computed(() => eligibleGroups.value.length > 1 && (priceDropCount.value > 0 || valueGroupCount.value > 0 || premiumGroupCount.value > 0))

function toggleQuickFilter(next: 'drop' | 'value' | 'premium') {
  quickFilter.value = quickFilter.value === next ? 'all' : next
}

const filteredGroups = computed(() =>
  eligibleGroups.value.filter((group) => {
    if (category.value !== 'all' && getKeyGroupProvider(group.platform) !== category.value) return false
    if (quickFilter.value === 'drop' && !isPriceDrop(group)) return false
    if (quickFilter.value === 'value' && !isValueGroup(group)) return false
    if (quickFilter.value === 'premium' && !isPremiumGroup(group)) return false
    return `${group.name} ${group.description || ''}`.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())
  })
)

const rateFields = computed(() => [
  { key: 'limit5h' as const, label: text('5 小时限额', '5-hour limit') },
  { key: 'limit1d' as const, label: text('每日限额', 'Daily limit') },
  { key: 'limit7d' as const, label: text('7 日限额', '7-day limit') }
])

function peakText(group: Group) {
  return text('高峰 ', 'Peak ') + `${group.peak_start || '—'}–${group.peak_end || '—'} · ${group.peak_rate_multiplier ?? 1}× ` + text('（服务器时间）', '(server time)')
}

function localDate(iso: string | null) {
  if (!iso) return ''
  const date = new Date(iso)
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
}

function onExpiryPreset(val: ExpiryPreset) {
  if (val === 'never') form.days = ''
  else if (val === '7' || val === '30' || val === '90') form.days = Number(val)
}

function syncExpiryPresetFromDays() {
  const str = String(form.days ?? '').trim()
  if (!str) expiryPreset.value = 'never'
  else if (str === '7' || str === '30' || str === '90') expiryPreset.value = str
  else expiryPreset.value = 'custom'
}

watch(
  () => props.open,
  (open) => {
    if (!open) {
      createdKey.value = null
      return
    }
    const key = props.editing
    createdKey.value = null
    copiedSecret.value = false
    copiedSnippet.value = false
    snippetTab.value = 'curl'
    expiryPreset.value = 'never'
    Object.assign(form, {
      name: key?.name || '',
      group: String(key?.group_id ?? props.selectedGroup ?? ''),
      smart: !!key?.routing_group_ids?.length,
      quota: key?.quota || 0,
      concurrency: key?.concurrency_limit ?? 0,
      cacheTtl: key?.anthropic_cache_ttl_mode ?? 'inherit',
      days: '',
      expiry: localDate(key?.expires_at ?? null),
      limit5h: key?.rate_limit_5h || 0,
      limit1d: key?.rate_limit_1d || 0,
      limit7d: key?.rate_limit_7d || 0,
      allow: key?.ip_whitelist?.join('\n') || '',
      block: key?.ip_blacklist?.join('\n') || '',
      custom: '',
      routing: [...(key?.routing_group_ids || [])],
      resetQuota: false,
      resetRates: false
    })
    ipRulesEnabled.value = Boolean(form.allow || form.block)
    error.value = ''
    search.value = ''
    category.value = 'all'
    quickFilter.value = 'all'
  },
  { immediate: true }
)

function isSelected(id: number) { return form.smart ? form.routing.includes(id) : form.group === String(id) }
function setMode(smart: boolean) {
  if (smart && !form.routing.length && routingGroups.value.some((group) => String(group.id) === form.group)) {
    form.routing = [Number(form.group)]
  }
  form.smart = smart
  category.value = 'all'
  quickFilter.value = 'all'
  error.value = ''
}

function selectGroup(id: number) {
  if (form.smart) {
    if (form.routing.length < 10 && !form.routing.includes(id) && routingGroups.value.some((group) => group.id === id)) {
      form.routing.push(id)
    }
  } else {
    form.group = String(id)
  }
  error.value = ''
}

function removeRouting(index: number) { form.routing.splice(index, 1) }
function moveRouting(index: number, direction: number) {
  const target = index + direction
  if (target < 0 || target >= form.routing.length) return
  const [id] = form.routing.splice(index, 1)
  form.routing.splice(target, 0, id)
}

async function submit() {
  if (saving.value || !form.name.trim()) return
  const routing = form.smart ? [...form.routing] : []
  if (form.smart && (!routing.length || routing.length > 10 || routing.some((id) => !routingGroups.value.some((group) => group.id === id)))) {
    error.value = text('请选择 1–10 个支持智能路由的分组。', 'Select 1–10 groups that support smart routing.')
    return
  }
  const groupId = routing[0] ?? (form.group ? Number(form.group) : null)
  if (groupId === null || !props.groups.some((group) => group.id === groupId)) {
    error.value = text('请选择可用的接入分组。', 'Select an available connection group.')
    return
  }
  if (!Number.isInteger(Number(form.concurrency)) || Number(form.concurrency) < 0) {
    error.value = text('并发上限必须是非负整数。', 'Concurrency limit must be a nonnegative integer.')
    return
  }
  if (
    ![form.quota, form.limit5h, form.limit1d, form.limit7d].every((value) => Number.isFinite(Number(value)) && Number(value) >= 0) ||
    (!props.editing && form.days !== '' && (!Number.isInteger(Number(form.days)) || Number(form.days) <= 0))
  ) {
    error.value = text('请输入有效的额度与有效天数。', 'Enter valid limits and expiry days.')
    return
  }
  if (!props.editing && form.custom && !/^[A-Za-z0-9_-]{16,}$/.test(form.custom)) {
    error.value = text('自定义密钥至少16位，仅支持字母、数字、下划线和连字符。', 'Custom keys need at least 16 letters, numbers, underscores or hyphens.')
    return
  }
  saving.value = true
  error.value = ''
  const lines = (value: string) => value.split(/[\n,]+/).map((v) => v.trim()).filter(Boolean)
  const limits = {
    rate_limit_5h: Number(form.limit5h),
    rate_limit_1d: Number(form.limit1d),
    rate_limit_7d: Number(form.limit7d)
  }
  try {
    const key = props.editing
      ? await keysAPI.update(props.editing.id, {
          name: form.name.trim(),
          group_id: groupId,
          quota: Number(form.quota),
          expires_at: form.expiry ? new Date(form.expiry).toISOString() : '',
          ip_whitelist: lines(form.allow),
          ip_blacklist: lines(form.block),
          routing_group_ids: routing,
          concurrency_limit: Number(form.concurrency),
          ...(form.cacheTtl !== (props.editing.anthropic_cache_ttl_mode ?? 'inherit') ? { anthropic_cache_ttl_mode: form.cacheTtl } : {}),
          ...limits,
          ...(form.resetQuota ? { reset_quota: true } : {}),
          ...(form.resetRates ? { reset_rate_limit_usage: true } : {})
        })
      : await keysAPI.create(
          form.name.trim(),
          groupId,
          form.custom || undefined,
          lines(form.allow),
          lines(form.block),
          Number(form.quota),
          form.days ? Number(form.days) : undefined,
          limits,
          routing,
          Number(form.concurrency),
          form.cacheTtl
        )
    if (!props.editing && key?.key) {
      createdKey.value = key
    }
    emit('saved', key)
    if (!props.editing && onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
      void onboardingStore.nextStep(500)
    }
  } catch (e) {
    error.value = userFacingError(e, locale.value, text('保存失败，请重试。', 'Could not save. Please retry.'))
  } finally {
    saving.value = false
  }
}
</script>
