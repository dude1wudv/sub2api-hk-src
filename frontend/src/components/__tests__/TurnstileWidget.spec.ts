import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import TurnstileWidget from '../TurnstileWidget.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const scriptSelector = 'script[src*="challenges.cloudflare.com/turnstile"]'
let wrapper: VueWrapper | undefined

function installSDK() {
  const render = vi.fn<NonNullable<typeof window.turnstile>['render']>(() => 'widget-1')
  window.turnstile = { render, reset: vi.fn(), remove: vi.fn() }
  return render
}

afterEach(() => {
  vi.useRealTimers()
  wrapper?.unmount()
  wrapper = undefined
  delete window.turnstile
  delete window.onTurnstileLoad
  document.querySelectorAll(scriptSelector).forEach((script) => script.remove())
  vi.restoreAllMocks()
})

describe('TurnstileWidget', () => {
  it('shows a loading placeholder until the SDK initializes the widget', async () => {
    wrapper = mount(TurnstileWidget, { props: { siteKey: 'site-key' } })

    expect(wrapper.get('[role="status"]').text()).toBe('auth.captchaLoading')
    expect(document.querySelector(scriptSelector)).not.toBeNull()
    const container = wrapper.get('.turnstile-container').element

    const render = installSDK()
    window.onTurnstileLoad?.()
    await flushPromises()

    expect(render).toHaveBeenCalledWith(container, expect.objectContaining({ sitekey: 'site-key' }))
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(wrapper.get('.turnstile-container').element).toBe(container)
  })

  it('initializes an already loaded SDK and still forwards verification', async () => {
    const render = installSDK()
    wrapper = mount(TurnstileWidget, { props: { siteKey: 'site-key' } })
    await flushPromises()

    expect(render).toHaveBeenCalledOnce()
    expect(document.querySelector(scriptSelector)).toBeNull()
    expect(wrapper.find('[role="status"]').exists()).toBe(false)

    render.mock.calls[0]![1].callback('verified-token')
    expect(wrapper.emitted('verify')).toEqual([['verified-token']])
  })

  it('ends loading and reports a script load failure', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    wrapper = mount(TurnstileWidget, { props: { siteKey: 'site-key' } })

    document.querySelector(scriptSelector)!.dispatchEvent(new Event('error'))
    await flushPromises()

    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(wrapper.emitted('error')).toEqual([[]])
    expect(wrapper.get('[role="alert"]').text()).toContain('auth.captchaRetry')

    await wrapper.get('button').trigger('click')
    const render = installSDK()
    window.onTurnstileLoad?.()
    await flushPromises()
    expect(render).toHaveBeenCalledOnce()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('shares one SDK request across widgets and renders each instance', async () => {
    const Parent = {
      components: { TurnstileWidget },
      template: '<div><TurnstileWidget site-key="first" /><TurnstileWidget site-key="second" /></div>'
    }
    wrapper = mount(Parent)
    const containers = wrapper.findAll('.turnstile-container').map((item) => item.element)
    expect(document.querySelectorAll(scriptSelector)).toHaveLength(1)

    const render = installSDK()
    window.onTurnstileLoad?.()
    await flushPromises()

    expect(render).toHaveBeenCalledTimes(2)
    expect(render.mock.calls.map((call) => call[0])).toEqual(containers)
    expect(render.mock.calls.map((call) => call[1].sitekey)).toEqual(['first', 'second'])
  })

  it('offers retry after the SDK times out and recovers after retry', async () => {
    vi.useFakeTimers()
    vi.spyOn(console, 'error').mockImplementation(() => {})
    wrapper = mount(TurnstileWidget, { props: { siteKey: 'site-key' } })

    await vi.advanceTimersByTimeAsync(15000)
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('auth.captchaRetry')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(document.querySelector(scriptSelector)).toBeNull()

    await wrapper.get('button').trigger('click')
    const render = installSDK()
    window.onTurnstileLoad?.()
    await flushPromises()

    expect(render).toHaveBeenCalledOnce()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })

  it('initializes when the site key changes from empty to configured', async () => {
    wrapper = mount(TurnstileWidget, { props: { siteKey: '' } })
    expect(document.querySelector(scriptSelector)).toBeNull()

    await wrapper.setProps({ siteKey: 'configured-site-key' })
    const render = installSDK()
    window.onTurnstileLoad?.()
    await flushPromises()

    expect(render).toHaveBeenCalledOnce()
    expect(render.mock.calls[0]![1].sitekey).toBe('configured-site-key')
  })

  it('does not render or forward a token when unmounted while the SDK loads', async () => {
    wrapper = mount(TurnstileWidget, { props: { siteKey: 'site-key' } })
    wrapper.unmount()
    const render = installSDK()
    window.onTurnstileLoad?.()
    await flushPromises()

    expect(render).not.toHaveBeenCalled()
    expect(wrapper.emitted('verify')).toBeUndefined()
  })

  it('ignores callbacks from a widget replaced after its site key changes', async () => {
    const render = installSDK()
    wrapper = mount(TurnstileWidget, { props: { siteKey: 'old-site-key' } })
    await flushPromises()
    const oldCallback = render.mock.calls[0]![1].callback

    await wrapper.setProps({ siteKey: 'new-site-key' })
    await flushPromises()
    const newCallback = render.mock.calls[1]![1].callback
    oldCallback('stale-token')

    expect(wrapper.emitted('verify')).toBeUndefined()
    newCallback('fresh-token')
    expect(wrapper.emitted('verify')).toEqual([['fresh-token']])
  })

  it('does not show a placeholder or load the SDK without a site key', async () => {
    wrapper = mount(TurnstileWidget, { props: { siteKey: '' } })
    await flushPromises()

    expect(wrapper.find('.turnstile-wrapper').exists()).toBe(false)
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(document.querySelector(scriptSelector)).toBeNull()
  })
})
