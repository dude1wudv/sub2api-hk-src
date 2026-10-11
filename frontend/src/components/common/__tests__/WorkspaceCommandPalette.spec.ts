import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import WorkspaceCommandPalette from '../WorkspaceCommandPalette.vue'
import type { WorkspaceCommand } from '@/utils/workspaceCommands'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('zh'), t: (key: string) => key }) }))
const wrappers: VueWrapper[] = []
const items: WorkspaceCommand[] = [
  { id: 'create', label: '创建 API 密钥', section: '快捷操作', to: '/keys?create=1' },
  { id: 'usage', label: '用量信息', section: '页面', to: '/usage', keywords: 'usage' },
  { id: 'custom', label: '我的文档', section: '自定义资源', to: '/custom/docs' },
]
async function setup(commands = items) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }] })
  await router.push('/dashboard')
  await router.isReady()
  const wrapper = mount(WorkspaceCommandPalette, { attachTo: document.body, props: { items: commands }, global: { plugins: [router], stubs: { transition: true } } })
  wrappers.push(wrapper)
  return { wrapper, router }
}
async function press(key: string, extras: KeyboardEventInit = {}) {
  document.activeElement?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...extras }))
  await flushPromises()
}
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()); document.body.innerHTML = '' })

describe('workspace command palette', () => {
  it.each(['ctrlKey', 'metaKey'] as const)('opens with %s K, focuses search, traps Tab and restores focus on Escape', async modifier => {
    const { wrapper } = await setup()
    const trigger = wrapper.get('button').element as HTMLButtonElement
    trigger.focus()
    await press('k', { [modifier]: true })
    const input = document.querySelector<HTMLInputElement>('[role="combobox"]')!
    expect(document.activeElement).toBe(input)
    expect(document.body.style.overflow).toBe('hidden')
    await press('Tab')
    expect(document.querySelector('[role="dialog"]')?.contains(document.activeElement)).toBe(true)
    await press('Escape')
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger)
    expect(document.body.style.overflow).not.toBe('hidden')
  })
  it('searches only supplied visible navigation including custom menus and consumes Enter', async () => {
    const { wrapper, router } = await setup()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    const input = document.querySelector<HTMLInputElement>('[role="combobox"]')!
    input.value = '我的文档'; input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    expect(document.querySelectorAll('[role="option"]')).toHaveLength(1)
    expect(document.querySelector('[data-command-palette]')?.textContent).not.toContain('管理设置')
    await press('Enter')
    expect(router.currentRoute.value.path).toBe('/custom/docs')
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })
  it('moves the keyboard selection and opens create without submitting anything', async () => {
    const { wrapper, router } = await setup()
    await wrapper.get('button').trigger('click'); await flushPromises()
    await press('ArrowDown')
    expect(document.querySelector('[aria-selected="true"]')?.textContent).toContain('用量信息')
    await press('Home'); await press('Enter')
    expect(router.currentRoute.value.fullPath).toBe('/keys?create=1')
  })
  it('removes revoked navigation while open and does not open over another form', async () => {
    const { wrapper } = await setup()
    const dialog = document.createElement('div'); dialog.setAttribute('role', 'dialog'); dialog.setAttribute('aria-modal', 'true'); document.body.append(dialog)
    await press('k', { ctrlKey: true })
    expect(document.querySelector('[role="combobox"]')).toBeNull()
    dialog.remove()
    await wrapper.get('button').trigger('click'); await flushPromises()
    await wrapper.setProps({ items: [items[1]] })
    expect(document.querySelectorAll('[role="option"]')).toHaveLength(1)
    expect(document.querySelector('[role="option"]')?.textContent).toContain('用量信息')
  })
  it('leaves failures open with a retry and removes the shortcut on unmount', async () => {
    const run = vi.fn().mockRejectedValueOnce(new Error('Unavailable')).mockResolvedValue(undefined)
    const { wrapper } = await setup([{ id: 'copy', label: '复制地址', section: '操作', run }])
    await wrapper.get('button').trigger('click'); await flushPromises(); await press('Enter')
    expect(document.querySelector('[role="alert"]')?.textContent).toContain('重试')
    ;(document.querySelector('[role="alert"] button') as HTMLButtonElement).click()
    await flushPromises()
    expect(run).toHaveBeenCalledTimes(2)
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    wrapper.unmount(); wrappers.splice(wrappers.indexOf(wrapper), 1)
    await press('k', { ctrlKey: true })
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })
  it.each(['native', 'alert'] as const)('does not steal focus or add a scroll lock over a %s dialog', async kind => {
    const { wrapper } = await setup()
    const dialog = document.createElement(kind === 'native' ? 'dialog' : 'div')
    if (kind === 'native') dialog.setAttribute('open', '')
    else { dialog.setAttribute('role', 'alertdialog'); dialog.setAttribute('aria-modal', 'true') }
    const input = document.createElement('input')
    dialog.append(input); document.body.append(dialog); input.focus()
    await press('k', { metaKey: true })
    expect(document.querySelector('[data-command-palette]')).toBeNull()
    expect(document.activeElement).toBe(input)
    expect(document.body.style.overflow).not.toBe('hidden')
    await wrapper.get('button').trigger('click'); await flushPromises()
    expect(document.querySelector('[data-command-palette]')).toBeNull()
    dialog.remove()
    await press('k', { ctrlKey: true })
    expect(document.querySelector('[data-command-palette]')).not.toBeNull()
  })
})
