import { describe, expect, it, vi } from 'vitest'

import {
  HELIOS_WORKBENCH_BOOTSTRAP_PATH,
  HELIOS_WORKBENCH_ORIGIN,
  launchHeliosWorkbench,
  parseHeliosLaunchUrl
} from '../heliosLaunch'

function makeBrowserWindow(open: (url: string, target: string) => Window | null, replace = vi.fn()) {
  return {
    open,
    location: { replace }
  } as unknown as Pick<Window, 'open' | 'location'>
}

describe('parseHeliosLaunchUrl', () => {
  it.each([
    [`${HELIOS_WORKBENCH_ORIGIN}${HELIOS_WORKBENCH_BOOTSTRAP_PATH}#code=one-time`, true],
    ['https://example.com/bootstrap#code=one-time', false],
    [`${HELIOS_WORKBENCH_ORIGIN}/workflow#code=one-time`, false],
    [`https://user:secret@canvas.sub.sunmmyapi.xyz${HELIOS_WORKBENCH_BOOTSTRAP_PATH}`, false]
  ])('accepts only the fixed origin and bootstrap path: %s', (value, accepted) => {
    const result = parseHeliosLaunchUrl(value)
    if (accepted) {
      expect(result).toEqual(expect.any(URL))
    } else {
      expect(result).toBeNull()
    }
  })
})

describe('launchHeliosWorkbench', () => {
  it('opens a blank popup, severs opener before requesting, then replaces it on success', async () => {
    const events: string[] = []
    let opener: Window | null = window
    const replace = vi.fn((url: string) => events.push(`replace:${url}`))
    const child = {
      get opener() {
        return opener
      },
      set opener(value: Window | null) {
        events.push('opener:null')
        opener = value
      },
      close: vi.fn(() => events.push('close')),
      location: { replace }
    } as unknown as Window
    const request = vi.fn(async () => {
      events.push('request')
      expect(opener).toBeNull()
      return { launch_url: `${HELIOS_WORKBENCH_ORIGIN}/bootstrap#code=one-time`, expires_in: 90 }
    })
    const browserWindow = makeBrowserWindow((url, target) => {
      events.push(`open:${url}:${target}`)
      return child
    })

    await expect(launchHeliosWorkbench({ window: browserWindow, request })).resolves.toBe(true)

    expect(events).toEqual([
      'open:about:blank:_blank',
      'opener:null',
      'request',
      'replace:https://canvas.sub.sunmmyapi.xyz/bootstrap#code=one-time'
    ])
    expect(request).toHaveBeenCalledOnce()
    expect(child.close).not.toHaveBeenCalled()
  })

  it('does not request a grant when the popup is blocked', async () => {
    const request = vi.fn()
    const notify = vi.fn()
    const browserWindow = makeBrowserWindow(() => null)

    await expect(launchHeliosWorkbench({ window: browserWindow, request, notify })).resolves.toBe(false)

    expect(request).not.toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith('popup-blocked')
  })

  it.each([
    'https://example.com/bootstrap#code=one-time',
    `${HELIOS_WORKBENCH_ORIGIN}/workflow#code=one-time`
  ])('closes the blank popup when launch URL is rejected: %s', async (launchUrl) => {
    const close = vi.fn()
    const child = { opener: null, close, location: { replace: vi.fn() } } as unknown as Window
    const notify = vi.fn()
    const browserWindow = makeBrowserWindow(() => child)

    await expect(
      launchHeliosWorkbench({
        window: browserWindow,
        request: async () => ({ launch_url: launchUrl, expires_in: 90 }),
        notify
      })
    ).resolves.toBe(false)

    expect(close).toHaveBeenCalledOnce()
    expect(notify).toHaveBeenCalledWith('launch-failed')
  })

  it('replaces the current tab for the protected /helios fallback without opening a popup', async () => {
    const open = vi.fn()
    const replace = vi.fn()
    const browserWindow = makeBrowserWindow(open, replace)
    const request = vi.fn(async () => ({
      launch_url: `${HELIOS_WORKBENCH_ORIGIN}/bootstrap#code=one-time`,
      expires_in: 90
    }))

    await expect(launchHeliosWorkbench({ mode: 'current', window: browserWindow, request })).resolves.toBe(true)

    expect(open).not.toHaveBeenCalled()
    expect(request).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith(`${HELIOS_WORKBENCH_ORIGIN}/bootstrap#code=one-time`)
  })

  it('reports API failures and closes the temporary popup', async () => {
    const close = vi.fn()
    const child = { opener: null, close, location: { replace: vi.fn() } } as unknown as Window
    const notify = vi.fn()
    const browserWindow = makeBrowserWindow(() => child)

    await expect(
      launchHeliosWorkbench({
        window: browserWindow,
        request: async () => {
          throw new Error('network failure')
        },
        notify
      })
    ).resolves.toBe(false)

    expect(close).toHaveBeenCalledOnce()
    expect(notify).toHaveBeenCalledWith('launch-failed')
  })
})
