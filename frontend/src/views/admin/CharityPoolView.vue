<template>
  <div class="space-y-6" data-admin-page="charity">
      <!-- 顶部标题与操作条 -->
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div class="flex items-center gap-2.5">
            <h1 class="text-xl font-semibold tracking-tight text-gray-900 dark:text-white">
              公益池管理（free.xjtu.shop）
            </h1>
            <span
              class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
              :class="
                settingsForm.enabled
                  ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
                  : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
              "
            >
              {{ settingsForm.enabled ? '已开启' : '未开启' }}
            </span>
          </div>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            管理独立公益域名总预算（按 Sub2API 折前 1× 原价计费，含 Fast/Ultra）、分组独立子配额、并发限流与每日北京时间 00:00 公共 Key 轮换。
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <a
            :href="primaryCharityURL"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary text-sm"
          >
            打开公益前端 ↗
          </a>
          <button
            type="button"
            class="btn btn-secondary text-sm"
            :disabled="loading"
            @click="loadAll"
          >
            刷新
          </button>
          <button
            type="button"
            class="btn btn-primary text-sm"
            :disabled="rotatingAll || groups.length === 0"
            @click="handleRotateAll"
          >
            {{ rotatingAll ? '轮换中…' : '一键全部轮换 Key' }}
          </button>
        </div>
      </div>

      <!-- 实时概览指标卡 -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="card p-4">
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
            今日共享总池（折前 1×）
          </div>
          <div class="mt-1.5 flex items-baseline gap-2">
            <span class="text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">
              ${{ formatMoney(overview?.used_today_usd ?? 0) }}
            </span>
            <span class="text-sm text-gray-500 dark:text-dark-400">
              / {{ budgetCapLabel }}
            </span>
          </div>
          <div class="mt-2.5 h-2 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
            <div
              class="h-full rounded-full transition-all"
              :class="poolProgressPercent >= 95 ? 'bg-red-500' : 'bg-emerald-500'"
              :style="{ width: `${Math.min(100, poolProgressPercent)}%` }"
            />
          </div>
          <div class="mt-1.5 text-xs text-gray-500 dark:text-dark-400">
            <template v-if="overview?.remaining_today_usd !== null && overview?.remaining_today_usd !== undefined">
              剩余 ${{ formatMoney(overview.remaining_today_usd) }}（{{ poolProgressPercent.toFixed(1) }}%）
            </template>
            <template v-else>
              总池不设上限
            </template>
          </div>
        </div>

        <div class="card p-4">
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
            实时在线并发
          </div>
          <div class="mt-1.5 flex items-baseline gap-2">
            <span class="text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">
              {{ overview?.current_concurrency ?? 0 }}
            </span>
            <span class="text-sm text-gray-500 dark:text-dark-400">
              / {{ (overview?.settings.max_concurrency ?? 0) > 0 ? overview?.settings.max_concurrency : '不限' }}
            </span>
          </div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            活跃公益分组：{{ enabledGroupCount }} / {{ groups.length }}
          </div>
        </div>

        <div class="card p-4">
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
            今日公益请求 / Tokens
          </div>
          <div class="mt-1.5 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">
            {{ (overview?.request_count_today ?? 0).toLocaleString() }}
          </div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            Tokens：{{ (overview?.total_tokens_today ?? 0).toLocaleString() }}
          </div>
        </div>

        <div class="card p-4">
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
            统计日期与自动轮换
          </div>
          <div class="mt-1.5 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">
            {{ overview?.usage_date || '—' }}
          </div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            {{ settingsForm.auto_rotate_daily ? '每日 00:00 (UTC+8) 自动轮换 Key' : '仅手动轮换 Key' }}
          </div>
        </div>
      </div>

      <!-- 全局配置表单 -->
      <div class="card p-5">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              全局预算、并发与域名配置
            </h2>
            <p class="text-xs text-gray-500 dark:text-dark-400">
              填 0 表示对应维度不设上限；单 IP 限流可在面对校园网 NAT 或公网流量时随时热调。
            </p>
          </div>
          <button
            type="button"
            class="btn btn-primary text-sm"
            :disabled="savingSettings"
            @click="handleSaveSettings"
          >
            {{ savingSettings ? '保存中…' : '保存全局配置' }}
          </button>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <label class="flex items-center justify-between rounded-lg border border-gray-200 px-3.5 py-2.5 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">开启公益池</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">关闭后公益 Key 立即暂停服务</div>
            </div>
            <input v-model="settingsForm.enabled" type="checkbox" class="h-4 w-4 accent-primary-600" />
          </label>

          <label class="flex items-center justify-between rounded-lg border border-gray-200 px-3.5 py-2.5 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">每日 00:00 自动轮换 Key</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">按北京时间跨日自动失效昨日公共 Key</div>
            </div>
            <input v-model="settingsForm.auto_rotate_daily" type="checkbox" class="h-4 w-4 accent-primary-600" />
          </label>

          <div>
            <label class="input-label">每日全站共享总预算（USD，0 = 不限）</label>
            <input
              v-model.number="settingsForm.daily_budget_usd"
              type="number"
              min="0"
              step="1"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">全站公益最大并发数（0 = 不限）</label>
            <input
              v-model.number="settingsForm.max_concurrency"
              type="number"
              min="0"
              step="1"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">单 IP 每日额度上限（USD，0 = 不限）</label>
            <input
              v-model.number="settingsForm.per_ip_daily_limit_usd"
              type="number"
              min="0"
              step="0.5"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">单 IP 最大并发 / RPM（0 = 不限）</label>
            <div class="grid grid-cols-2 gap-2">
              <input
                v-model.number="settingsForm.per_ip_max_concurrency"
                type="number"
                min="0"
                step="1"
                placeholder="并发"
                class="input"
              />
              <input
                v-model.number="settingsForm.per_ip_rpm_limit"
                type="number"
                min="0"
                step="1"
                placeholder="RPM"
                class="input"
              />
            </div>
          </div>

          <div>
            <label class="input-label">公益站标题</label>
            <input v-model="settingsForm.site_title" type="text" class="input" placeholder="XJTU Free AI" />
          </div>

          <div>
            <label class="input-label">公益站域名白名单（逗号分隔）</label>
            <input
              v-model="hostAllowlistText"
              type="text"
              class="input"
              placeholder="free.xjtu.shop"
            />
          </div>

          <div>
            <label class="input-label">重置时区</label>
            <input
              v-model="settingsForm.rotation_timezone"
              type="text"
              class="input"
              placeholder="Asia/Shanghai"
            />
          </div>

          <div class="md:col-span-2 lg:col-span-3">
            <label class="input-label">公益站公告（可选，展示在 free.xjtu.shop 顶部）</label>
            <input
              v-model="settingsForm.site_notice"
              type="text"
              class="input"
              placeholder="例如：面向西交同学与开源爱好者的免费公益 API，每日北京时间 00:00 重置额度并轮换 Key。"
            />
          </div>
        </div>
      </div>

      <!-- 公益分组列表 -->
      <div class="card overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              公益专用分组与公共 Key
            </h2>
            <p class="text-xs text-gray-500 dark:text-dark-400">
              可为特定分组设置独立每日子上限（如 Claude $50 / GPT $50），留空则不设独立上限、直接共享总池余额；点击「管理账号」即可给分组增删上游账号。
            </p>
          </div>
          <button type="button" class="btn btn-primary text-sm" @click="openAddModal">
            + 添加公益分组
          </button>
        </div>

        <div v-if="groups.length === 0" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-dark-400">
          暂未配置公益分组。点击右上角「+ 添加公益分组」绑定已有分组或一键新建专属公益分组。
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800/60">
              <tr class="text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                <th class="px-4 py-3">分组 / 平台</th>
                <th class="px-4 py-3">今日消耗 / 子上限</th>
                <th class="px-4 py-3">实时并发</th>
                <th class="px-4 py-3">关联账号</th>
                <th class="px-4 py-3">今日公共 Key</th>
                <th class="px-4 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in groups" :key="item.id" class="hover:bg-gray-50/70 dark:hover:bg-dark-800/40">
                <td class="px-4 py-3.5 align-top">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-gray-900 dark:text-white">
                      {{ item.display_name || item.group_name }}
                    </span>
                    <span class="rounded bg-gray-100 px-1.5 py-0.5 text-[11px] font-medium uppercase text-gray-600 dark:bg-dark-700 dark:text-dark-300">
                      {{ item.group_platform }}
                    </span>
                    <span
                      v-if="!item.enabled"
                      class="rounded bg-amber-100 px-1.5 py-0.5 text-[11px] text-amber-800 dark:bg-amber-900/40 dark:text-amber-300"
                    >
                      已停用
                    </span>
                  </div>
                  <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                    底层分组 #{{ item.group_id }} · {{ item.group_name }}
                  </div>
                </td>

                <td class="px-4 py-3.5 align-top">
                  <div class="font-medium tabular-nums text-gray-900 dark:text-white">
                    ${{ formatMoney(item.used_today_usd) }}
                    <span class="text-xs font-normal text-gray-500 dark:text-dark-400">
                      / {{ item.daily_limit_usd ? `$${formatMoney(item.daily_limit_usd)}` : '共享总池（无单组上限）' }}
                    </span>
                  </div>
                  <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                    {{ item.request_count_today }} 次请求
                  </div>
                </td>

                <td class="px-4 py-3.5 align-top tabular-nums text-gray-700 dark:text-gray-200">
                  {{ item.current_concurrency }} / {{ item.max_concurrency > 0 ? item.max_concurrency : '跟随总池' }}
                </td>

                <td class="px-4 py-3.5 align-top">
                  <div class="text-sm text-gray-900 dark:text-white">
                    {{ item.active_account_count }} 可用 / {{ item.account_count }} 总计
                  </div>
                  <router-link
                    :to="{ path: '/admin/accounts', query: { group_id: String(item.group_id) } }"
                    class="mt-0.5 inline-block text-xs font-medium text-primary-600 hover:underline dark:text-primary-400"
                  >
                    管理账号 →
                  </router-link>
                </td>

                <td class="px-4 py-3.5 align-top">
                  <div class="flex items-center gap-1.5">
                    <code class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-800 dark:bg-dark-700 dark:text-gray-200">
                      {{ maskKey(item.active_api_key) }}
                    </code>
                    <button
                      v-if="item.active_api_key"
                      type="button"
                      class="text-xs text-primary-600 hover:underline dark:text-primary-400"
                      @click="copyText(item.active_api_key)"
                    >
                      复制
                    </button>
                  </div>
                  <div class="mt-0.5 text-[11px] text-gray-500 dark:text-dark-400">
                    轮换日期：{{ item.last_rotated_date || '尚未轮换' }}
                  </div>
                </td>

                <td class="px-4 py-3.5 align-top text-right">
                  <div class="inline-flex items-center gap-2">
                    <button
                      type="button"
                      class="text-xs font-medium text-amber-600 hover:underline dark:text-amber-400"
                      :disabled="rotatingGroupID === item.id"
                      @click="handleRotateGroup(item)"
                    >
                      {{ rotatingGroupID === item.id ? '轮换中…' : '立即轮换' }}
                    </button>
                    <button
                      type="button"
                      class="text-xs font-medium text-primary-600 hover:underline dark:text-primary-400"
                      @click="openEditModal(item)"
                    >
                      编辑
                    </button>
                    <button
                      type="button"
                      class="text-xs font-medium text-red-600 hover:underline dark:text-red-400"
                      @click="handleRemoveGroup(item)"
                    >
                      移出
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 近 14 天消耗趋势 -->
      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-3.5 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">
            近 14 天公益折前消耗汇总
          </h2>
        </div>
        <div v-if="!overview?.trend?.length" class="px-5 py-6 text-center text-sm text-gray-500 dark:text-dark-400">
          暂无历史消耗记录
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800/60">
              <tr class="text-left text-xs font-medium text-gray-500 dark:text-dark-400">
                <th class="px-4 py-2.5">日期</th>
                <th class="px-4 py-2.5">折前总消耗（USD）</th>
                <th class="px-4 py-2.5">请求数</th>
                <th class="px-4 py-2.5">总 Tokens</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="row in overview.trend" :key="row.usage_date">
                <td class="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-200">{{ row.usage_date }}</td>
                <td class="px-4 py-2.5 font-medium tabular-nums text-gray-900 dark:text-white">${{ formatMoney(row.total_cost_usd) }}</td>
                <td class="px-4 py-2.5 tabular-nums text-gray-700 dark:text-gray-300">{{ row.request_count.toLocaleString() }}</td>
                <td class="px-4 py-2.5 tabular-nums text-gray-700 dark:text-gray-300">{{ row.total_tokens.toLocaleString() }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 添加 / 编辑公益分组弹窗 -->
      <div
        v-if="showGroupModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
        @click.self="closeGroupModal"
      >
        <div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-dark-800">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ editingGroup ? '编辑公益分组' : '添加公益分组' }}
          </h3>

          <div class="mt-4 space-y-4">
            <template v-if="!editingGroup">
              <div class="flex gap-2">
                <button
                  type="button"
                  class="flex-1 rounded-lg border px-3 py-2 text-xs font-medium"
                  :class="
                    groupModalMode === 'existing'
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                      : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-dark-300'
                  "
                  @click="groupModalMode = 'existing'"
                >
                  绑定已有分组
                </button>
                <button
                  type="button"
                  class="flex-1 rounded-lg border px-3 py-2 text-xs font-medium"
                  :class="
                    groupModalMode === 'create'
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                      : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-dark-300'
                  "
                  @click="groupModalMode = 'create'"
                >
                  一步新建专属公益分组
                </button>
              </div>

              <div v-if="groupModalMode === 'existing'">
                <label class="input-label">选择现有分组</label>
                <select v-model.number="groupForm.group_id" class="input">
                  <option :value="null" disabled>请选择一个分组</option>
                  <option
                    v-for="g in availableBaseGroups"
                    :key="g.id"
                    :value="g.id"
                  >
                    #{{ g.id }} {{ g.name }} ({{ g.platform }})
                  </option>
                </select>
              </div>

              <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label class="input-label">新分组名称</label>
                  <input v-model="groupForm.create_group_name" type="text" class="input" placeholder="如：公益 Claude 组" />
                </div>
                <div>
                  <label class="input-label">平台类型</label>
                  <select v-model="groupForm.create_group_platform" class="input">
                    <option value="openai">openai</option>
                    <option value="anthropic">anthropic</option>
                    <option value="gemini">gemini</option>
                    <option value="deepseek">deepseek</option>
                    <option value="composite">composite（多平台聚合）</option>
                    <option value="antigravity">antigravity</option>
                    <option value="grok">grok</option>
                  </select>
                </div>
              </div>
            </template>

            <div>
              <label class="input-label">对外展示名称（留空则沿用底层分组名）</label>
              <input v-model="groupForm.display_name" type="text" class="input" placeholder="如：Claude 公益通道 / GPT 公益通道" />
            </div>

            <div>
              <label class="input-label">分组简介（展示在 free.xjtu.shop 卡片上）</label>
              <input v-model="groupForm.description" type="text" class="input" placeholder="如：支持 Claude Code / Anthropic Messages 协议" />
            </div>

            <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <label class="input-label">独立每日子上限（USD）</label>
                <input
                  v-model="groupForm.daily_limit_text"
                  type="number"
                  min="0"
                  step="1"
                  class="input"
                  placeholder="留空=共享总池"
                />
              </div>
              <div>
                <label class="input-label">分组并发上限（0=跟随总池）</label>
                <input v-model.number="groupForm.max_concurrency" type="number" min="0" step="1" class="input" />
              </div>
              <div>
                <label class="input-label">排序权重</label>
                <input v-model.number="groupForm.sort_order" type="number" step="1" class="input" />
              </div>
            </div>

            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
              <input v-model="groupForm.enabled" type="checkbox" class="h-4 w-4 accent-primary-600" />
              启用该公益分组
            </label>
          </div>

          <div class="mt-6 flex justify-end gap-2">
            <button type="button" class="btn btn-secondary text-sm" @click="closeGroupModal">
              取消
            </button>
            <button
              type="button"
              class="btn btn-primary text-sm"
              :disabled="savingGroup"
              @click="handleSaveGroup"
            >
              {{ savingGroup ? '提交中…' : '保存' }}
            </button>
          </div>
        </div>
      </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useAppStore } from '@/stores/app'
import { groupsAPI } from '@/api/admin/groups'
import {
  adminCharityAPI,
  type CharityAdminGroup,
  type CharityAdminOverview
} from '@/api/admin/charity'
import type { AdminGroup } from '@/types'

const appStore = useAppStore()

const loading = ref(false)
const savingSettings = ref(false)
const rotatingAll = ref(false)
const rotatingGroupID = ref<number | null>(null)
const overview = ref<CharityAdminOverview | null>(null)
const allBaseGroups = ref<AdminGroup[]>([])

const settingsForm = reactive({
  enabled: false,
  daily_budget_usd: 100,
  max_concurrency: 20,
  per_ip_daily_limit_usd: 0,
  per_ip_max_concurrency: 0,
  per_ip_rpm_limit: 0,
  auto_rotate_daily: true,
  rotation_timezone: 'Asia/Shanghai',
  site_title: 'XJTU Free AI',
  site_notice: ''
})
const hostAllowlistText = ref('free.xjtu.shop')

const showGroupModal = ref(false)
const editingGroup = ref<CharityAdminGroup | null>(null)
const groupModalMode = ref<'existing' | 'create'>('existing')
const savingGroup = ref(false)
const groupForm = reactive({
  group_id: null as number | null,
  create_group_name: '',
  create_group_platform: 'openai',
  display_name: '',
  description: '',
  daily_limit_text: '' as string | number,
  max_concurrency: 0,
  sort_order: 0,
  enabled: true
})

const groups = computed(() => overview.value?.groups ?? [])
const enabledGroupCount = computed(() => groups.value.filter(g => g.enabled).length)

const primaryCharityURL = computed(() => {
  const firstHost = (overview.value?.settings.host_allowlist?.[0] || 'free.xjtu.shop').trim()
  return `https://${firstHost}`
})

const budgetCapLabel = computed(() => {
  const cap = overview.value?.settings.daily_budget_usd ?? 0
  return cap > 0 ? `$${formatMoney(cap)}` : '不限'
})

const poolProgressPercent = computed(() => {
  const cap = overview.value?.settings.daily_budget_usd ?? 0
  const used = overview.value?.used_today_usd ?? 0
  if (cap <= 0) return 0
  return Math.min(100, (used / cap) * 100)
})

const availableBaseGroups = computed(() => {
  const bound = new Set(groups.value.map(g => g.group_id))
  return allBaseGroups.value.filter(g => !bound.has(g.id))
})

function formatMoney(v: number): string {
  if (!Number.isFinite(v)) return '0.00'
  return v >= 1 ? v.toFixed(2) : v.toFixed(4)
}

function maskKey(key: string): string {
  if (!key) return '未生成'
  if (key.length <= 16) return key
  return `${key.slice(0, 12)}…${key.slice(-6)}`
}

async function copyText(val: string) {
  try {
    await navigator.clipboard.writeText(val)
    appStore.showSuccess('已复制公共 Key')
  } catch {
    appStore.showError('复制失败，请手动复制')
  }
}

async function loadAll() {
  loading.value = true
  try {
    const [ov, baseGroups] = await Promise.all([
      adminCharityAPI.getOverview(),
      groupsAPI.getAll().catch(() => [] as AdminGroup[])
    ])
    overview.value = ov
    allBaseGroups.value = baseGroups
    Object.assign(settingsForm, {
      enabled: ov.settings.enabled,
      daily_budget_usd: ov.settings.daily_budget_usd,
      max_concurrency: ov.settings.max_concurrency,
      per_ip_daily_limit_usd: ov.settings.per_ip_daily_limit_usd,
      per_ip_max_concurrency: ov.settings.per_ip_max_concurrency,
      per_ip_rpm_limit: ov.settings.per_ip_rpm_limit,
      auto_rotate_daily: ov.settings.auto_rotate_daily,
      rotation_timezone: ov.settings.rotation_timezone || 'Asia/Shanghai',
      site_title: ov.settings.site_title || 'XJTU Free AI',
      site_notice: ov.settings.site_notice || ''
    })
    hostAllowlistText.value = (ov.settings.host_allowlist || ['free.xjtu.shop']).join(', ')
  } catch (err: unknown) {
    appStore.showError(err instanceof Error ? err.message : '加载公益池数据失败')
  } finally {
    loading.value = false
  }
}

async function handleSaveSettings() {
  savingSettings.value = true
  try {
    const hosts = hostAllowlistText.value
      .split(',')
      .map(s => s.trim())
      .filter(Boolean)
    await adminCharityAPI.updateSettings({
      ...settingsForm,
      host_allowlist: hosts.length > 0 ? hosts : ['free.xjtu.shop']
    })
    appStore.showSuccess('公益池全局配置已保存')
    await loadAll()
  } catch (err: unknown) {
    appStore.showError(err instanceof Error ? err.message : '保存失败')
  } finally {
    savingSettings.value = false
  }
}

function openAddModal() {
  editingGroup.value = null
  groupModalMode.value = availableBaseGroups.value.length > 0 ? 'existing' : 'create'
  Object.assign(groupForm, {
    group_id: availableBaseGroups.value[0]?.id ?? null,
    create_group_name: '',
    create_group_platform: 'openai',
    display_name: '',
    description: '',
    daily_limit_text: '',
    max_concurrency: 0,
    sort_order: groups.value.length,
    enabled: true
  })
  showGroupModal.value = true
}

function openEditModal(item: CharityAdminGroup) {
  editingGroup.value = item
  Object.assign(groupForm, {
    group_id: item.group_id,
    display_name: item.display_name,
    description: item.description,
    daily_limit_text: item.daily_limit_usd !== null && item.daily_limit_usd > 0 ? String(item.daily_limit_usd) : '',
    max_concurrency: item.max_concurrency,
    sort_order: item.sort_order,
    enabled: item.enabled
  })
  showGroupModal.value = true
}

function closeGroupModal() {
  showGroupModal.value = false
}

function parseOptionalLimit(val: string | number): number | null {
  const s = String(val ?? '').trim()
  if (!s) return null
  const n = Number(s)
  if (!Number.isFinite(n) || n <= 0) return null
  return n
}

async function handleSaveGroup() {
  savingGroup.value = true
  try {
    const dailyLimit = parseOptionalLimit(groupForm.daily_limit_text)
    if (editingGroup.value) {
      await adminCharityAPI.updateGroup(editingGroup.value.id, {
        display_name: groupForm.display_name,
        description: groupForm.description,
        enabled: groupForm.enabled,
        daily_limit_usd: dailyLimit,
        max_concurrency: groupForm.max_concurrency,
        sort_order: groupForm.sort_order
      })
      appStore.showSuccess('公益分组已更新')
    } else {
      await adminCharityAPI.addGroup({
        group_id: groupModalMode.value === 'existing' ? groupForm.group_id : null,
        create_group_name: groupModalMode.value === 'create' ? groupForm.create_group_name : undefined,
        create_group_platform: groupModalMode.value === 'create' ? groupForm.create_group_platform : undefined,
        create_group_description: groupModalMode.value === 'create' ? groupForm.description : undefined,
        display_name: groupForm.display_name,
        description: groupForm.description,
        enabled: groupForm.enabled,
        daily_limit_usd: dailyLimit,
        max_concurrency: groupForm.max_concurrency,
        sort_order: groupForm.sort_order
      })
      appStore.showSuccess('已加入公益池并自动签发今日公共 Key')
    }
    showGroupModal.value = false
    await loadAll()
  } catch (err: unknown) {
    appStore.showError(err instanceof Error ? err.message : '保存公益分组失败')
  } finally {
    savingGroup.value = false
  }
}

async function handleRotateGroup(item: CharityAdminGroup) {
  rotatingGroupID.value = item.id
  try {
    await adminCharityAPI.rotateGroupKey(item.id)
    appStore.showSuccess(`已立即轮换「${item.display_name || item.group_name}」公共 Key`)
    await loadAll()
  } catch (err: unknown) {
    appStore.showError(err instanceof Error ? err.message : '轮换失败')
  } finally {
    rotatingGroupID.value = null
  }
}

async function handleRotateAll() {
  rotatingAll.value = true
  try {
    await adminCharityAPI.rotateAllKeys()
    appStore.showSuccess('已全部轮换所有公益分组公共 Key')
    await loadAll()
  } catch (err: unknown) {
    appStore.showError(err instanceof Error ? err.message : '全量轮换失败')
  } finally {
    rotatingAll.value = false
  }
}

async function handleRemoveGroup(item: CharityAdminGroup) {
  if (!window.confirm(`确定将「${item.display_name || item.group_name}」移出公益池并失效其公共 Key？（底层分组与账号会保留）`)) {
    return
  }
  try {
    await adminCharityAPI.removeGroup(item.id)
    appStore.showSuccess('已移出公益池')
    await loadAll()
  } catch (err: unknown) {
    appStore.showError(err instanceof Error ? err.message : '移出失败')
  }
}

onMounted(() => {
  void loadAll()
})
</script>
