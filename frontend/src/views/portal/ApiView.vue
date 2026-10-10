<template>
  <div class="portal-page portal-api-page">
    <h1 class="portal-heading">API</h1>

    <p class="portal-api-intro">{{ text('管理你的 API Keys。创建密钥时选择分组与计费规则。', 'Manage your API keys. Choose a group and billing rules when creating a key.') }}</p>

    <p v-if="groupsLoading" class="portal-api-inline-state" role="status">{{ text('正在读取可绑定分组…', 'Loading groups available for your keys…') }}</p>
    <div v-else-if="groupsError" class="portal-error portal-api-inline-state" role="alert">{{ groupsError }} <button type="button" class="portal-inline-button" @click="loadGroups">{{ text('重试分组加载', 'Retry groups') }}</button></div>
    <p v-else-if="!groups.length" class="portal-api-inline-state">{{ text('当前账户没有可绑定分组，暂时无法创建 API Key。', 'Your account has no available group for creating an API key.') }} <button type="button" class="portal-inline-button" @click="loadGroups">{{ text('刷新分组', 'Refresh groups') }}</button></p>

    <section id="portal-api-keys" class="portal-section portal-api-keys" aria-labelledby="portal-api-keys-title">
      <div class="portal-section-heading"><h2 id="portal-api-keys-title">{{ text('我的 API Keys', 'My API keys') }}</h2><div class="portal-api-key-heading-actions"><button type="button" class="portal-inline-button" :aria-pressed="bulkMode" data-test="bulk-mode-toggle" @click="toggleBulkMode">{{ bulkMode ? text('退出批量管理', 'Exit bulk management') : text('批量管理', 'Bulk management') }}</button><button type="button" class="portal-button" :disabled="creationDisabled" @click="openCreate()">＋ {{ text('创建 API Key', 'Create API key') }}</button></div></div>
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
        <button v-else type="button" class="portal-button" :disabled="creationDisabled" @click="openCreate()">{{ text('创建 API Key', 'Create API key') }}</button>
      </div>
      <div v-else class="portal-api-key-grid">
        <PortalApiKeyCard v-for="key in keys" :key="key.id" :api-key="key" :groups="groups" :busy="busyKeyIds.has(key.id)" :selectable="bulkMode" :selected="selectedKeyIds.includes(key.id)" @select="selectKey" @copy="copyKey" @edit="openEdit" @toggle="toggleKey" @delete="confirmDelete" />
      </div>

      <nav v-if="keyTotal > 0" class="portal-pagination" :aria-label="text('密钥分页', 'API key pages')">
        <span>{{ text('共 ', 'Total: ') }}{{ keyTotal }}{{ text(' 个', ' keys') }}</span>
        <select v-model.number="keyPageSize" :disabled="keysLoading" :aria-label="text('每页密钥数量', 'Keys per page')" @change="changePageSize"><option :value="6">6 / {{ text('页', 'page') }}</option><option :value="12">12 / {{ text('页', 'page') }}</option><option :value="24">24 / {{ text('页', 'page') }}</option></select>
        <button type="button" :disabled="keysLoading || keyPage <= 1" @click="goToPage(keyPage - 1)">{{ text('上一页', 'Previous') }}</button>
        <span>{{ keyPage }} / {{ keyPages }}</span>
        <button type="button" :disabled="keysLoading || keyPage >= keyPages" @click="goToPage(keyPage + 1)">{{ text('下一页', 'Next') }}</button>
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
import PortalApiKeyCard from '@/components/portal/PortalApiKeyCard.vue'
import PortalApiExamples from '@/components/portal/PortalApiExamples.vue'
import PortalApiBulkForm from '@/components/portal/PortalApiBulkForm.vue'
import PortalDialog from '@/components/portal/PortalDialog.vue'
import PortalKeyForm from '@/components/portal/PortalKeyForm.vue'
import { keysAPI } from '@/api/keys'
import { userGroupsAPI } from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'
import type { ApiKey, Group } from '@/types'

const { locale } = useI18n()
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
const keyPageSize = ref(6)
const keyTotal = ref(0)
const keyPages = computed(() => Math.max(1, Math.ceil(keyTotal.value / keyPageSize.value)))
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
  keyController = new AbortController()
  keysLoading.value = true
  keysError.value = ''
  try {
    const result = await keysAPI.list(keyPage.value, keyPageSize.value, {
      search: appliedKeySearch.value || undefined,
      status: keyStatusFilter.value || undefined,
      group_id: keyGroupFilter.value || undefined,
      sort_by: 'created_at',
      sort_order: 'desc',
    }, { signal: keyController.signal })
    if (disposed || request !== keyRequest) return
    keys.value = result.items || []
    selectedKeyIds.value = selectedKeyIds.value.filter(id => keys.value.some(key => key.id === id))
    keyTotal.value = result.total || 0
    if (keyPage.value > keyPages.value) {
      clearSelection()
      keyPage.value = keyPages.value
      await loadKeys()
    }
  } catch (error) {
    if (!disposed && request === keyRequest) keysError.value = errorMessage(error, text('密钥加载失败，请重试。', 'Could not load API keys. Please retry.'))
  } finally {
    if (!disposed && request === keyRequest) keysLoading.value = false
  }
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
  if (page < 1 || page > keyPages.value || keysLoading.value) return
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
.portal-api-intro { margin:-14px 0 28px; color:#76736b; font-size:13px; }
.portal-api-resources { display:flex; flex-wrap:wrap; align-items:center; gap:12px 28px; margin-top:30px; padding-top:22px; border-top:1px solid #efede3; }
.portal-api-resources > :is(button,a) { display:inline-flex; align-items:center; gap:8px; color:#76736b; background:transparent; border:0; padding:4px 0; font:inherit; font-size:13px; text-decoration:none; }
.portal-api-resources > :is(button,a):hover { color:#000; }
.portal-api-inline-state { margin-top:18px; font-size:12px; }
.portal-api-keys { scroll-margin-top:80px; }
.portal-api-key-heading-actions { display:flex; align-items:center; flex-wrap:wrap; gap:12px; font-size:12px; }
.portal-api-bulk-toolbar { display:flex; align-items:center; flex-wrap:wrap; gap:12px; margin:0 0 18px; padding:14px 16px; border:1px solid #efede3; border-radius:8px; background:#f7f5ee; font-size:12px; }
.portal-api-bulk-toolbar label { display:inline-flex; align-items:center; gap:8px; }
.portal-api-bulk-toolbar input { width:15px; height:15px; accent-color:#000; }
.portal-api-bulk-toolbar p { flex-basis:100%; margin:0; font-size:11px; color:#76736b; }
.portal-api-bulk-toolbar .portal-button { min-height:30px; padding:5px 12px; }
.portal-api-bulk-toolbar .portal-inline-button:disabled { opacity:.45; cursor:not-allowed; }
.portal-api-key-filters { display:flex; flex-wrap:wrap; align-items:center; gap:10px; margin:18px 0; }
.portal-api-key-search { display:flex; min-width:180px; max-width:320px; flex:1; align-items:center; gap:8px; min-height:36px; padding:0 10px; border:1px solid #efede3; border-radius:8px; font-size:12px; }
.portal-api-key-search input { min-width:0; width:100%; padding:7px 0; border:0; outline:none; color:#000; background:transparent; font:inherit; }
.portal-api-key-search:focus-within { border-color:#000; }
.portal-api-key-search > button { flex-shrink:0; }
.portal-api-key-filters select { min-width:110px; max-width:230px; min-height:36px; padding:7px 28px 7px 10px; border:1px solid #efede3; border-radius:8px; background-color:#fffdf7; color:#000; font:inherit; font-size:12px; }
.portal-api-key-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:14px; }
.portal-api-key-state { padding:36px 20px; border:1px solid #efede3; border-radius:10px; color:#76736b; text-align:center; font-size:13px; }
.portal-api-key-state p { margin:0 0 16px; }
.portal-api-created-key { display:flex; align-items:center; flex-wrap:wrap; gap:10px; margin:14px 0; padding:10px 14px; border:1px solid #efede3; border-radius:8px; background:#f7f5ee; font-size:12px; }
.portal-api-created-key strong { font-weight:500; }
.portal-api-created-key > button:last-child { margin-left:auto; }
.portal-api-delete-name { margin:16px 0; font-weight:500; overflow-wrap:anywhere; }
.portal-api-guide { display:grid; gap:16px; padding-left:22px; list-style:decimal; }
.portal-api-guide a { text-decoration:underline; text-underline-offset:3px; }
@media(min-width:1700px) { .portal-api-key-grid { grid-template-columns:repeat(3,minmax(0,1fr)); } }
@media(max-width:540px) { .portal-api-key-grid { grid-template-columns:1fr; } .portal-api-key-search { max-width:none; width:100%; flex-basis:100%; } .portal-api-key-filters select { flex:1; min-width:0; max-width:none; } }
</style>
