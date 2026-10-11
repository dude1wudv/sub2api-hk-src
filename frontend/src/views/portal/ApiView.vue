<template>
  <div class="portal-page portal-api-page">
    <header class="portal-api-header">
      <div>
        <p class="portal-api-eyebrow">02 / ACCESS</p>
        <div class="portal-api-title-row">
          <h1>{{ text('API 密钥', 'API keys') }}</h1>
          <span class="portal-api-count">{{ keyTotal }}</span>
        </div>
      </div>
      <div class="portal-api-header-actions">
        <button type="button" class="portal-api-icon-button" :disabled="keysLoading || groupsLoading" :aria-label="text('刷新', 'Refresh')" data-test="refresh-keys" @click="refreshPage"><Icon name="refresh" size="sm" aria-hidden="true" /></button>
        <button type="button" class="portal-button secondary" :aria-pressed="bulkMode" data-test="bulk-mode-toggle" @click="toggleBulkMode">{{ bulkMode ? text('退出批量管理', 'Exit bulk management') : text('批量管理', 'Bulk management') }}</button>
        <button type="button" class="portal-button" :disabled="creationDisabled" @click="openCreate()"><Icon name="plus" size="sm" aria-hidden="true" />{{ text('创建密钥', 'Create key') }}</button>
      </div>
    </header>


    <p v-if="groupsLoading" class="portal-api-inline-state" role="status">{{ text('正在读取可绑定分组…', 'Loading groups available for your keys…') }}</p>
    <div v-else-if="groupsError" class="portal-error portal-api-inline-state" role="alert">{{ groupsError }} <button type="button" class="portal-inline-button" @click="loadGroups">{{ text('重试分组加载', 'Retry groups') }}</button></div>
    <p v-else-if="!groups.length" class="portal-api-inline-state">{{ text('当前账户没有可绑定分组，暂时无法创建 API Key。', 'Your account has no available group for creating an API key.') }} <button type="button" class="portal-inline-button" @click="loadGroups">{{ text('刷新分组', 'Refresh groups') }}</button></p>

    <section id="portal-api-keys" class="portal-section portal-api-keys" aria-labelledby="portal-api-keys-title">
      <h2 id="portal-api-keys-title" class="portal-api-table-heading">{{ text('密钥列表', 'Key list') }}</h2>
      <div v-if="lastCreated" class="portal-api-created-key" role="status">
        <span>{{ text('密钥已创建：', 'Key created: ') }}<strong>{{ lastCreated.name }}</strong></span>
        <button type="button" class="portal-inline-button" @click="copyKey(lastCreated)"><Icon name="copy" size="sm" aria-hidden="true" />{{ text('复制密钥', 'Copy key') }}</button>
        <button type="button" class="portal-inline-button" :aria-label="text('关闭创建提示', 'Dismiss creation message')" @click="lastCreated = null">✕</button>
      </div>

      <form class="portal-api-key-filters" @submit.prevent="applyKeyFilters">
        <label class="portal-api-key-search"><Icon name="search" size="sm" aria-hidden="true" /><input v-model="keySearch" type="search" :aria-label="text('搜索 API Key', 'Search API keys')" :placeholder="text('搜索名称或密钥', 'Search name or key')" /><button type="submit" class="portal-inline-button">{{ text('搜索', 'Search') }}</button></label>
        <select v-model="keyStatusFilter" :aria-label="text('密钥状态', 'Key status')" @change="applyKeyFilters"><option value="">{{ text('全部状态', 'All statuses') }}</option><option value="active">{{ text('已启用', 'Active') }}</option><option value="inactive">{{ text('已停用', 'Inactive') }}</option><option value="expired">{{ text('已过期', 'Expired') }}</option><option value="quota_exhausted">{{ text('额度用尽', 'Quota exhausted') }}</option></select>
        <select v-model="keyGroupFilter" :aria-label="text('密钥分组', 'Key group')" @change="applyKeyFilters"><option value="">{{ text('全部分组', 'All groups') }}</option><option value="0">{{ text('未绑定分组', 'No group') }}</option><option v-for="group in groups" :key="group.id" :value="String(group.id)">{{ group.name }}</option></select>
      </form>
      <div class="portal-api-endpoint">
        <span>{{ text('API 端点', 'API endpoint') }}</span>
        <span class="portal-api-endpoint-tag">{{ text('默认', 'Default') }}</span>
        <code :title="apiBaseUrl">{{ apiBaseUrl || '—' }}</code>
        <button type="button" :disabled="!apiBaseUrl" :aria-label="text('复制 API 端点', 'Copy API endpoint')" data-test="copy-endpoint" @click="copyEndpoint"><Icon name="copy" size="sm" aria-hidden="true" /></button>
      </div>

      <div v-if="bulkMode" class="portal-api-bulk-toolbar">
        <label><input type="checkbox" :checked="allPageKeysSelected" :indeterminate="somePageKeysSelected" :disabled="keysLoading || !!keysError || !keys.length || busyKeyIds.size > 0" data-test="bulk-select-all" @change="selectPageKeys(($event.target as HTMLInputElement).checked)" />{{ text('当前页全选', 'Select this page') }}</label>
        <span aria-live="polite">{{ text('已选择 ', 'Selected: ') }}{{ selectedKeys.length }}{{ text(' 个', ' keys') }}</span>
        <button type="button" class="portal-button" :disabled="!selectedKeys.length || keysLoading || !!keysError || busyKeyIds.size > 0" data-test="bulk-edit-keys" @click="bulkFormOpen = true">{{ text('批量编辑', 'Edit selected') }}</button>
        <button type="button" class="portal-inline-button" :disabled="!selectedKeys.length" @click="clearSelection">{{ text('清空选择', 'Clear selection') }}</button>
        <p>{{ text('仅选择当前页；筛选、翻页或更改每页数量会清空选择。', 'Selection is limited to this page and clears when filters, page or page size change.') }}</p>
      </div>

      <p v-if="actionError" class="portal-error" role="alert">{{ actionError }}</p>
      <div v-if="keysLoading" class="portal-api-key-state" role="status">{{ text('正在加载密钥…', 'Loading API keys…') }}</div>
      <div v-else-if="keysError" class="portal-error" role="alert">{{ keysError }} <button type="button" class="portal-inline-button" @click="loadKeys">{{ text('重试', 'Retry') }}</button></div>
      <div v-else-if="!keys.length" class="portal-api-key-state">
        <p>{{ hasKeyFilters ? text('没有符合筛选条件的密钥。', 'No keys match these filters.') : text('还没有 API Key。创建一个密钥，开始接入你的应用。', 'No API keys yet. Create one to connect your application.') }}</p>
        <button v-if="hasKeyFilters" type="button" class="portal-button secondary" @click="resetKeyFilters">{{ text('清除筛选', 'Clear filters') }}</button>
        <button v-else type="button" class="portal-button" :disabled="creationDisabled" @click="openCreate()">{{ text('创建密钥', 'Create key') }}</button>
      </div>
      <PortalApiKeyTable
        v-else
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
        @import-ccs="importToCcs"
        @edit="openEdit"
        @toggle="toggleKey"
        @delete="confirmDelete"
      />

      <nav v-if="keyTotal > 0" class="portal-api-pager" :aria-label="text('密钥分页', 'API key pages')">
        <div class="portal-api-pager-summary">
          <span>{{ text('显示 ', 'Showing ') }}{{ rangeStart }}{{ text(' 至 ', '–') }}{{ rangeEnd }}{{ text(' 共 ', ' of ') }}{{ keyTotal }}{{ text(' 条结果', ' results') }}</span>
          <label>{{ text('每页', 'Per page') }}
            <select v-model.number="keyPageSize" :disabled="keysLoading" :aria-label="text('每页密钥数量', 'Keys per page')" @change="changePageSize">
              <option :value="10">10</option>
              <option :value="20">20</option>
              <option :value="50">50</option>
            </select>
          </label>
        </div>
        <div class="portal-api-pager-pages">
          <button type="button" :disabled="keysLoading || keyPage <= 1" :aria-label="text('上一页', 'Previous')" @click="goToPage(keyPage - 1)">‹</button>
          <template v-for="(page, index) in pageList" :key="page">
            <span v-if="pageSeparated(index)" class="portal-api-pager-gap">…</span>
            <button type="button" :disabled="keysLoading" :aria-current="page === keyPage ? 'page' : undefined" @click="goToPage(page)">{{ page }}</button>
          </template>
          <button type="button" :disabled="keysLoading || keyPage >= keyPages" :aria-label="text('下一页', 'Next')" @click="goToPage(keyPage + 1)">›</button>
        </div>
      </nav>
    </section>

    <section class="portal-api-resources" :aria-label="text('接入资源', 'Connection resources')">
      <RouterLink v-if="plazaEnabled" to="/model-plaza?embedded=1"><Icon name="grid" size="sm" />{{ text('模型与价格', 'Models & pricing') }}<Icon name="arrowRight" size="sm" /></RouterLink>
      <button type="button" @click="examplesOpen = !examplesOpen" :aria-expanded="examplesOpen"><Icon name="terminal" size="sm" />{{ text('调用示例', 'Request examples') }}<Icon name="chevronDown" size="sm" /></button>
      <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer"><Icon name="book" size="sm" />{{ text('接入文档', 'Documentation') }}<Icon name="externalLink" size="sm" /></a>
      <button v-else type="button" @click="docsOpen = true"><Icon name="book" size="sm" />{{ text('接入指南', 'Quick start') }}<Icon name="arrowRight" size="sm" /></button>
    </section>
    <PortalApiExamples v-if="examplesOpen" v-model:model="exampleModel" />

    <PortalKeyForm :open="keyFormOpen" :editing="editingKey" :groups="groups" :group-rates="groupRates" :selected-group="formGroupId" @close="keyFormOpen = false" @saved="onKeySaved" />
    <UseKeyModal :show="!!useKeyTarget" :api-key="useKeyTarget?.key || ''" :base-url="apiBaseUrl" :platform="useKeyTarget?.group?.platform || null" :claude-code-only="useKeyTarget?.group?.claude_code_only || false" :allow-messages-dispatch="useKeyTarget?.group?.allow_messages_dispatch || false" @close="useKeyTarget = null" />
    <PortalDialog :open="ccsClientOpen" :title="t('keys.ccsClientSelect.title')" @close="closeCcsClient">
      <p>{{ t('keys.ccsClientSelect.description') }}</p>
      <div class="portal-api-ccs-choices">
        <button type="button" @click="chooseCcsClient('claude')"><Icon name="terminal" size="md" aria-hidden="true" /><strong>{{ t('keys.ccsClientSelect.claudeCode') }}</strong><span>{{ t('keys.ccsClientSelect.claudeCodeDesc') }}</span></button>
        <button type="button" @click="chooseCcsClient('gemini')"><Icon name="sparkles" size="md" aria-hidden="true" /><strong>{{ t('keys.ccsClientSelect.geminiCli') }}</strong><span>{{ t('keys.ccsClientSelect.geminiCliDesc') }}</span></button>
      </div>
    </PortalDialog>
    <PortalApiBulkForm :open="bulkFormOpen" :selected-keys="selectedKeys" :groups="groups" :groups-loading="groupsLoading" :groups-error="groupsError" @close="bulkFormOpen = false" @updated="onBulkUpdated" />
    <PortalDialog :open="!!deleteTarget" :title="text('删除 API Key', 'Delete API key')" @close="closeDelete">
      <p>{{ text('删除后，使用此密钥的应用将无法继续请求。', 'Applications using this key will no longer be able to send requests.') }}</p>
      <p class="portal-api-delete-name">{{ deleteTarget?.name }}</p>
      <p v-if="deleteError" class="portal-error" role="alert">{{ deleteError }}</p>
      <footer class="portal-dialog-actions"><button type="button" class="portal-button secondary" :disabled="deleting" @click="closeDelete">{{ text('取消', 'Cancel') }}</button><button type="button" class="portal-button" :disabled="deleting" @click="deleteKey">{{ deleting ? text('删除中…', 'Deleting…') : text('确认删除', 'Delete key') }}</button></footer>
    </PortalDialog>
    <PortalDialog :open="docsOpen" :title="text('API 接入指南', 'API quick start')" @close="docsOpen = false">
      <ol class="portal-api-guide">
        <li>{{ text('点击创建 API Key，在弹窗中选择分组并查看分组倍率。完整模型和价格可在模型与价格页面查看。', 'Create an API key to choose a group and review its multiplier. Open Models & pricing for the full catalog.') }}</li>
        <li>{{ text('创建 API Key，并在需要时设置额度、有效期、IP 限制或路由分组。', 'Create an API key and configure quota, expiry, IP restrictions or routing groups as needed.') }}</li>
        <li>{{ text('使用页面下方的接口地址和调用示例，以所选分组支持的模型发送请求。', 'Use the base URL and examples below to request a model supported by your group.') }}</li>
        <li v-if="!authStore.isSimpleMode"><RouterLink to="/usage">{{ text('在用量信息中查看请求与费用。', 'Review requests and costs in Usage.') }}</RouterLink> <RouterLink to="/purchase">{{ text('需要补充额度时前往充值。', 'Recharge when you need more balance.') }}</RouterLink></li>
      </ol>
      <footer class="portal-dialog-actions"><button type="button" class="portal-button" @click="docsOpen = false; showExamples()">{{ text('查看示例', 'View examples') }}</button></footer>
    </PortalDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'
import PortalApiKeyTable from '@/components/portal/PortalApiKeyTable.vue'
import PortalApiExamples from '@/components/portal/PortalApiExamples.vue'
import PortalApiBulkForm from '@/components/portal/PortalApiBulkForm.vue'
import PortalDialog from '@/components/portal/PortalDialog.vue'
import PortalKeyForm from '@/components/portal/PortalKeyForm.vue'
import { keysAPI } from '@/api/keys'
import { usageAPI, type BatchApiKeyUsageStats } from '@/api/usage'
import { userGroupsAPI } from '@/api/groups'
import { CC_SWITCH_USAGE_SCRIPT, buildCcSwitchImportDeeplink, type CcSwitchClientType } from '@/utils/ccswitchImport'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'
import type { ApiKey, Group } from '@/types'

const { locale, t } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const plazaEnabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.modelPlaza))

const groups = ref<Group[]>([])
const groupsLoading = ref(true)
const groupsError = ref('')
const groupRates = ref<Record<number, number>>({})
const exampleModel = ref('')
const examplesOpen = ref(false)
const creationDisabled = computed(() => groupsLoading.value || !!groupsError.value || !groups.value.length)

const keys = ref<ApiKey[]>([])
const keysLoading = ref(true)
const keysError = ref('')
const keySearch = ref('')
const appliedKeySearch = ref('')
const keyStatusFilter = ref('')
const keyGroupFilter = ref('')
const keyPage = ref(1)
const keyPageSize = ref(20)
const keyTotal = ref(0)
const keyPages = computed(() => Math.max(1, Math.ceil(keyTotal.value / keyPageSize.value)))
const rangeStart = computed(() => keyTotal.value === 0 ? 0 : (keyPage.value - 1) * keyPageSize.value + 1)
const rangeEnd = computed(() => Math.min(keyPage.value * keyPageSize.value, keyTotal.value))
const pageList = computed(() => {
  const pages = new Set([1, keyPages.value, keyPage.value - 1, keyPage.value, keyPage.value + 1].filter(page => page >= 1 && page <= keyPages.value))
  return [...pages].sort((left, right) => left - right)
})
function pageSeparated(index: number) {
  const previous = pageList.value[index - 1]
  const current = pageList.value[index]
  return previous != null && current != null && current - previous > 1
}
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const apiBaseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url?.trim()
  if (configured) return configured
  return typeof window === 'undefined' ? '' : window.location.origin
})
const hideCcsImport = computed(() => !!appStore.cachedPublicSettings?.hide_ccs_import_button)
const useKeyTarget = ref<ApiKey | null>(null)
const ccsClientOpen = ref(false)
const pendingCcsKey = ref<ApiKey | null>(null)
const hasKeyFilters = computed(() => !!(appliedKeySearch.value || keyStatusFilter.value || keyGroupFilter.value))
const busyKeyIds = ref(new Set<number>())
const bulkMode = ref(false)
const bulkFormOpen = ref(false)
const selectedKeyIds = ref<number[]>([])
const selectedKeys = computed(() => keys.value.filter(key => selectedKeyIds.value.includes(key.id)))
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

let disposed = false
let groupRequest = 0
let keyRequest = 0
let keyController: AbortController | undefined

function errorMessage(error: unknown, fallback: string) {
  return (error as { message?: string })?.message || fallback
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
  keysLoading.value = true
  keysError.value = ''
  try {
    const result = await keysAPI.list(keyPage.value, keyPageSize.value, {
      search: appliedKeySearch.value || undefined,
      status: keyStatusFilter.value || undefined,
      group_id: keyGroupFilter.value || undefined,
      sort_by: 'created_at',
      sort_order: 'desc',
    }, { signal: controller.signal })
    if (disposed || request !== keyRequest) return
    keys.value = result.items || []
    selectedKeyIds.value = selectedKeyIds.value.filter(id => keys.value.some(key => key.id === id))
    keyTotal.value = result.total || 0
    usageStats.value = {}
    if (keyPage.value > keyPages.value) {
      clearSelection()
      keyPage.value = keyPages.value
      await loadKeys()
      return
    }
    void loadUsage(keys.value.map(key => key.id), controller.signal, request)
  } catch (error) {
    if (!disposed && request === keyRequest) keysError.value = errorMessage(error, text('密钥加载失败，请重试。', 'Could not load API keys. Please retry.'))
  } finally {
    if (!disposed && request === keyRequest) keysLoading.value = false
  }
}
async function loadUsage(ids: number[], signal: AbortSignal, request: number) {
  if (!ids.length) return
  try {
    const usageResponse = await usageAPI.getDashboardApiKeysUsage(ids, { signal })
    if (disposed || request !== keyRequest) return
    usageStats.value = usageResponse.stats || {}
  } catch {
    if (signal.aborted || disposed || request !== keyRequest) return
    usageStats.value = {}
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
    baseUrl: apiBaseUrl.value || window.location.origin,
    platform: key.group?.platform || 'anthropic',
    clientType,
    providerName,
    apiKey: key.key,
    usageScript: CC_SWITCH_USAGE_SCRIPT,
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
function changePageSize() { clearSelection(); keyPage.value = 1; void loadKeys() }
function clearSelection() { selectedKeyIds.value = [] }
function toggleBulkMode() { bulkMode.value = !bulkMode.value; clearSelection() }
function selectKey(id: number, checked: boolean) {
  if (!bulkMode.value || keysLoading.value || keysError.value || busyKeyIds.value.has(id) || !keys.value.some(key => key.id === id)) return
  selectedKeyIds.value = checked ? [...new Set([...selectedKeyIds.value, id])] : selectedKeyIds.value.filter(selected => selected !== id)
}
function selectPageKeys(checked: boolean) {
  if (!bulkMode.value || keysLoading.value || keysError.value || busyKeyIds.value.size) return
  selectedKeyIds.value = checked ? keys.value.map(key => key.id) : []
}
function onBulkUpdated(succeededIds: number[]) {
  const succeeded = new Set(succeededIds)
  selectedKeyIds.value = selectedKeyIds.value.filter(id => !succeeded.has(id))
  void loadKeys()
}
function openCreate(groupId?: number) {
  if (creationDisabled.value) return
  editingKey.value = null
  formGroupId.value = groupId != null && groups.value.some(group => group.id === groupId) ? groupId : undefined
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
async function onKeySaved(key: ApiKey) {
  clearSelection()
  const created = !editingKey.value
  keyFormOpen.value = false
  editingKey.value = null
  if (created) {
    lastCreated.value = key
    keySearch.value = ''
    appliedKeySearch.value = ''
    keyStatusFilter.value = ''
    keyGroupFilter.value = ''
    keyPage.value = 1
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
function showExamples() { examplesOpen.value = true; scrollTo('portal-api-examples') }

// Preserve bookmarked model searches after moving the catalog off the key page.
watch(() => route.query.model, value => {
  const model = typeof value === 'string' ? value : Array.isArray(value) ? value[0] || '' : ''
  if (model && plazaEnabled.value) void router.replace({ path:'/model-plaza', query:{ embedded:'1', model } })
}, { immediate: true })
onMounted(() => { void loadGroups(); void loadKeys() })
onBeforeUnmount(() => {
  disposed = true
  keyController?.abort()
})
</script>

<style scoped>
.portal-api-page {
  width:auto;
  max-width:none;
  min-height:calc(100dvh - 56px);
  margin:0;
  padding:28px 32px 64px;
  background-color:#f4f7fa;
  background-image:
    linear-gradient(rgba(176, 196, 216, .28) 1px, transparent 1px),
    linear-gradient(90deg, rgba(176, 196, 216, .28) 1px, transparent 1px),
    radial-gradient(circle at 12% 18%, #7eb6ff 0 3px, transparent 3.5px),
    radial-gradient(circle at 78% 30%, #5aa2f5 0 2.5px, transparent 3.5px),
    radial-gradient(circle at 88% 72%, #9ec9ff 0 2px, transparent 3px);
  background-size:56px 56px, 56px 56px, 100% 100%, 100% 100%, 100% 100%;
}
.portal-api-header { display:flex; align-items:flex-start; justify-content:space-between; gap:16px; margin-bottom:22px; }
.portal-api-eyebrow { margin:0 0 6px; color:#8b93a1; font-size:11px; font-weight:650; letter-spacing:.12em; }
.portal-api-title-row { display:flex; align-items:center; gap:10px; }
.portal-api-title-row h1 { margin:0; color:#121826; font-size:28px; font-weight:650; line-height:1.2; }
.portal-api-count { display:inline-flex; min-width:24px; align-items:center; justify-content:center; padding:1px 7px; border-radius:999px; background:#e7eef6; color:#3d4b63; font-size:12px; font-weight:600; }
.portal-api-header-actions { display:flex; align-items:center; flex-wrap:wrap; justify-content:flex-end; gap:8px; }
.portal-api-icon-button { display:inline-flex; width:36px; height:36px; align-items:center; justify-content:center; border:1px solid #d7dee7; border-radius:10px; color:#1c2430; background:#fff; cursor:pointer; }
.portal-api-icon-button:hover { background:#f3f6fa; }
.portal-api-icon-button:disabled { opacity:.45; cursor:not-allowed; }
.portal-api-header-actions .portal-button { min-height:36px; border-radius:10px; }
.portal-api-resources { display:flex; flex-wrap:wrap; align-items:center; gap:12px 28px; margin-top:30px; padding-top:22px; border-top:1px solid #e4e9ef; }
.portal-api-resources > :is(button,a) { display:inline-flex; align-items:center; gap:8px; color:#6b7280; background:transparent; border:0; padding:4px 0; font:inherit; font-size:13px; text-decoration:none; }
.portal-api-resources > :is(button,a):hover { color:#111827; }
.portal-api-inline-state { margin-top:18px; font-size:12px; }
.portal-api-keys { scroll-margin-top:80px; }
.portal-api-table-heading { position:absolute; width:1px; height:1px; padding:0; margin:-1px; overflow:hidden; clip:rect(0,0,0,0); white-space:nowrap; border:0; }
.portal-api-bulk-toolbar { display:flex; align-items:center; flex-wrap:wrap; gap:12px; margin:0 0 18px; padding:14px 16px; border:1px solid #e4e9ef; border-radius:10px; background:#fff; font-size:12px; }
.portal-api-bulk-toolbar label { display:inline-flex; align-items:center; gap:8px; }
.portal-api-bulk-toolbar input { width:15px; height:15px; accent-color:#1d4ed8; }
.portal-api-bulk-toolbar p { flex-basis:100%; margin:0; font-size:11px; color:#6b7280; }
.portal-api-bulk-toolbar .portal-button { min-height:30px; padding:5px 12px; }
.portal-api-bulk-toolbar .portal-inline-button:disabled { opacity:.45; cursor:not-allowed; }
.portal-api-key-filters { display:flex; flex-wrap:wrap; align-items:center; gap:10px; margin:18px 0 12px; }
.portal-api-key-search { display:flex; min-width:180px; max-width:320px; flex:1; align-items:center; gap:8px; min-height:38px; padding:0 10px; border:1px solid #d7dee7; border-radius:10px; background:#fff; font-size:12px; }
.portal-api-key-search input { min-width:0; width:100%; padding:7px 0; border:0; outline:none; color:#121826; background:transparent; font:inherit; }
.portal-api-key-search:focus-within { border-color:#2458b5; }
.portal-api-key-search > button { flex-shrink:0; }
.portal-api-key-filters select { min-width:110px; max-width:230px; min-height:38px; padding:7px 28px 7px 10px; border:1px solid #d7dee7; border-radius:10px; background-color:#fff; color:#121826; font:inherit; font-size:12px; }
.portal-api-endpoint { display:inline-flex; max-width:100%; align-items:center; gap:8px; margin:0 0 16px; padding:6px 8px 6px 12px; border:1px solid #d7dee7; border-radius:999px; background:#fff; color:#5d6675; font-size:12px; }
.portal-api-endpoint-tag { padding:1px 6px; border-radius:999px; background:#eef6ff; color:#2458b5; }
.portal-api-endpoint code { min-width:0; overflow:hidden; color:#1c2430; font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace; text-overflow:ellipsis; white-space:nowrap; }
.portal-api-endpoint button { display:inline-flex; padding:4px; border:0; border-radius:999px; color:#2458b5; background:transparent; cursor:pointer; }
.portal-api-endpoint button:hover { background:#eef6ff; }
.portal-api-endpoint button:disabled { opacity:.4; cursor:not-allowed; }
.portal-api-key-state { padding:36px 20px; border:1px solid #e4e9ef; border-radius:12px; color:#6b7280; background:#fff; text-align:center; font-size:13px; }
.portal-api-key-state p { margin:0 0 16px; }
.portal-api-created-key { display:flex; align-items:center; flex-wrap:wrap; gap:10px; margin:14px 0; padding:10px 14px; border:1px solid #d7e7c8; border-radius:10px; background:#f3fbf4; font-size:12px; }
.portal-api-created-key strong { font-weight:600; }
.portal-api-created-key > button:last-child { margin-left:auto; }
.portal-api-pager { display:flex; align-items:center; justify-content:space-between; flex-wrap:wrap; gap:12px; margin-top:16px; color:#5d6675; font-size:12px; }
.portal-api-pager-summary { display:flex; align-items:center; flex-wrap:wrap; gap:12px; }
.portal-api-pager-summary label { display:inline-flex; align-items:center; gap:8px; }
.portal-api-pager select, .portal-api-pager-pages button { min-width:32px; min-height:32px; border:1px solid #d7dee7; border-radius:8px; background:#fff; color:#1c2430; }
.portal-api-pager-pages { display:flex; align-items:center; gap:6px; }
.portal-api-pager-pages button[aria-current='page'] { border-color:#2458b5; background:#2458b5; color:#fff; }
.portal-api-pager button:disabled { color:#c0c6d0; cursor:not-allowed; }
.portal-api-pager-gap { padding:0 2px; }
.portal-api-delete-name { margin:16px 0; font-weight:500; overflow-wrap:anywhere; }
.portal-api-guide { display:grid; gap:16px; padding-left:22px; list-style:decimal; }
.portal-api-guide a { text-decoration:underline; text-underline-offset:3px; }
.portal-api-ccs-choices { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:12px; margin-top:16px; }
.portal-api-ccs-choices button { display:flex; min-height:120px; flex-direction:column; align-items:center; justify-content:center; gap:6px; padding:16px; border:1px solid #d7dee7; border-radius:12px; background:#fff; color:#1c2430; font:inherit; cursor:pointer; }
.portal-api-ccs-choices button:hover { border-color:#2458b5; background:#f4f8ff; }
.portal-api-ccs-choices span { color:#6b7280; font-size:12px; text-align:center; }
@media(max-width:739px) {
  .portal-api-page { padding:16px 16px 48px; }
  .portal-api-header { flex-direction:column; }
  .portal-api-header-actions { justify-content:flex-start; }
  .portal-api-key-search { max-width:none; width:100%; flex-basis:100%; }
  .portal-api-key-filters select { flex:1; min-width:0; max-width:none; }
  .portal-api-ccs-choices { grid-template-columns:1fr; }
}
</style>
