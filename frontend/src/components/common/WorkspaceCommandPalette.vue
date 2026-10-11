<template>
  <button ref="trigger" type="button" class="workspace-command-trigger" :aria-label="text('搜索页面与快捷操作', 'Search pages and actions')" aria-haspopup="dialog" :aria-expanded="open" @click="openPalette">
    <Icon name="search" size="sm" aria-hidden="true" />
    <span>{{ text('前往…', 'Go to…') }}</span><kbd>{{ modifier }} K</kbd>
  </button>
  <BaseDialog :show="open" :title="text('去往下一步', 'Your next step')" width="wide" close-on-click-outside @close="open = false">
    <div class="workspace-command" data-command-palette>
      <label class="workspace-command-search">
        <Icon name="search" size="md" aria-hidden="true" />
        <input ref="search" v-model="query" autofocus autocomplete="off" role="combobox" aria-autocomplete="list" :aria-expanded="open" :aria-controls="listId" :aria-activedescendant="activeId" :aria-label="text('搜索页面或操作', 'Search pages or actions')" :placeholder="text('搜索页面、功能或操作…', 'Search pages, features or actions…')" @keydown="onSearchKeydown" />
        <kbd>esc</kbd>
      </label>
      <p class="workspace-command-hint">{{ text('输入名称快速抵达，或选择下面的常用入口。', 'Find a page by name, or pick an action below.') }}</p>
      <div :id="listId" ref="list" class="workspace-command-results" role="listbox" :aria-label="text('搜索结果', 'Search results')">
        <button v-for="(item, index) in matches" :id="`${listId}-${index}`" :key="item.id" type="button" role="option" tabindex="-1" :aria-selected="index === selected" :disabled="busy" class="workspace-command-item" @pointermove="selected = index" @click="execute(item)">
          <span class="workspace-command-symbol" aria-hidden="true"><Icon :name="item.run ? 'bolt' : 'arrowRight'" size="sm" /></span>
          <span><strong>{{ item.label }}</strong><small>{{ item.section }}</small></span>
          <Icon v-if="index === selected" name="arrowRight" size="sm" aria-hidden="true" />
        </button>
        <div v-if="!matches.length" class="workspace-command-empty"><Icon name="search" size="lg" aria-hidden="true" /><p>{{ text('没有匹配的入口，换个关键词试试。', 'No matching page. Try another keyword.') }}</p></div>
      </div>
      <p v-if="error" class="workspace-command-error" role="alert">{{ error }} <button type="button" @click="retry">{{ text('重试', 'Retry') }}</button></p>
      <footer class="workspace-command-footer"><span><kbd>↑</kbd> <kbd>↓</kbd> {{ text('选择', 'Select') }}</span><span><kbd>↵</kbd> {{ text('打开', 'Open') }}</span><span>{{ matches.length }} {{ text('个入口', 'results') }}</span></footer>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, getCurrentInstance, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import BaseDialog from './BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { isWorkspaceShortcut, searchWorkspaceCommands, type WorkspaceCommand } from '@/utils/workspaceCommands'

const props = defineProps<{ items: WorkspaceCommand[] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const router = useRouter()
const open = ref(false), query = ref(''), selected = ref(0), busy = ref(false), error = ref('')
const trigger = ref<HTMLButtonElement>(), search = ref<HTMLInputElement>(), list = ref<HTMLElement>()
const listId = `workspace-commands-${getCurrentInstance()?.uid ?? 0}`
const modifier = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘' : 'Ctrl'
const matches = computed(() => searchWorkspaceCommands(props.items, query.value))
const activeId = computed(() => matches.value[selected.value] ? `${listId}-${selected.value}` : undefined)
let failed: WorkspaceCommand | undefined
function hasBlockingDialog() {
  return !!document.querySelector('dialog[open], [role="dialog"][aria-modal="true"], [role="alertdialog"][aria-modal="true"]')
}
function openPalette() {
  if (!open.value && hasBlockingDialog()) return
  query.value = ''; selected.value = 0; error.value = ''; open.value = true
  void nextTick(() => search.value?.focus())
}
function shortcut(event: KeyboardEvent) {
  if (!isWorkspaceShortcut(event)) return
  // Do not steal a form's key handling or open behind another dialog/drawer.
  if (!open.value && hasBlockingDialog()) return
  event.preventDefault()
  if (open.value) open.value = false
  else openPalette()
}
function onSearchKeydown(event: KeyboardEvent) {
  if (event.isComposing) return
  const length = matches.value.length
  if (event.key === 'Enter') {
    event.preventDefault()
    const item = matches.value[selected.value]
    if (item) void execute(item)
  } else if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key) && length) {
    event.preventDefault()
    selected.value = event.key === 'Home' ? 0 : event.key === 'End' ? length - 1 : (selected.value + (event.key === 'ArrowDown' ? 1 : -1) + length) % length
    void nextTick(() => document.getElementById(activeId.value || '')?.scrollIntoView?.({ block: 'nearest' }))
  }
}
async function execute(item: WorkspaceCommand) {
  // Feature flags/roles can change while the panel is open; reject stale entries.
  if (busy.value || !props.items.some(entry => entry.id === item.id)) return
  busy.value = true; error.value = ''
  try {
    if (item.run) await item.run()
    else if (item.to) await router.push(item.to)
    else if (item.href) window.location.assign(item.href)
    open.value = false
  } catch {
    failed = item
    error.value = text('操作未完成，请重试。', 'The action could not be completed. Please retry.')
  } finally { busy.value = false }
}
function retry() { if (failed) void execute(failed) }
watch(matches, () => { selected.value = 0 })
onMounted(() => document.addEventListener('keydown', shortcut))
onBeforeUnmount(() => document.removeEventListener('keydown', shortcut))
</script>

<style scoped>
.workspace-command-trigger { display:inline-flex; align-items:center; gap:10px; min-height:36px; padding:7px 10px; border:1px solid rgb(var(--line)); border-radius:999px; background:rgb(var(--surface)); color:rgb(var(--ink-3)); font-size:13px; white-space:nowrap; }
.workspace-command-trigger:hover { border-color:rgb(var(--brand-moss) / .55); color:rgb(var(--ink)); }
kbd { font:12px/1.4 var(--font-sans); border:1px solid rgb(var(--line)); border-radius:5px; padding:1px 5px; background:rgb(var(--surface-2)); color:rgb(var(--ink-3)); }
.workspace-command { margin:-8px; }
.workspace-command-search { display:flex; align-items:center; gap:12px; padding:14px 16px; border:1px solid rgb(var(--line)); border-radius:14px; background:rgb(var(--surface-2)); color:rgb(var(--ink-3)); }
.workspace-command-search:focus-within { border-color:rgb(var(--focus-ring)); }
.workspace-command-search input { width:100%; min-width:0; border:0; background:transparent; color:rgb(var(--ink)); font-size:16px; outline:none; }
.workspace-command-hint { margin:14px 4px; font-size:13px; color:rgb(var(--ink-3)); }
.workspace-command-results { max-height:min(48dvh,410px); overflow-y:auto; overscroll-behavior:contain; padding:3px; }
.workspace-command-item { display:flex; width:100%; align-items:center; gap:12px; padding:12px; border:1px solid transparent; border-radius:14px; text-align:start; }
.workspace-command-item[aria-selected='true'] { background:rgb(var(--selection-bg)); border-color:rgb(var(--brand-moss) / .22); }
.workspace-command-item > span:nth-child(2) { display:grid; flex:1; gap:2px; min-width:0; }
.workspace-command-item strong { font-size:15px; font-weight:550; overflow-wrap:anywhere; }
.workspace-command-item small { color:rgb(var(--ink-3)); font-size:13px; }
.workspace-command-symbol { display:grid; place-items:center; width:34px; height:34px; border-radius:11px; color:rgb(var(--brand-moss)); background:rgb(var(--brand-moss) / .09); }
.workspace-command-footer { display:flex; align-items:center; gap:16px; border-top:1px solid rgb(var(--line)); margin-top:12px; padding:12px 4px 0; color:rgb(var(--ink-3)); font-size:13px; }
.workspace-command-footer > :last-child { margin-left:auto; }
.workspace-command-empty { display:grid; justify-items:center; gap:12px; padding:34px 16px; color:rgb(var(--ink-3)); text-align:center; }
.workspace-command-error { padding:12px; color:rgb(var(--text-err)); font-size:14px; }
.workspace-command-error button { text-decoration:underline; }
@media(max-width:767px) { .workspace-command-trigger { width:44px; height:44px; padding:0; justify-content:center; } .workspace-command-trigger > span,.workspace-command-trigger > kbd { display:none; } .workspace-command-results { max-height:45dvh; } }
</style>
