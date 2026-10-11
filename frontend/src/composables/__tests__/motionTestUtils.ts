import { vi } from 'vitest'

export const motionQueries = {
  reduced: '(prefers-reduced-motion: reduce)',
  pointer: '(hover: hover) and (pointer: fine)',
  desktop: '(min-width: 768px)',
}

export function mockMotionMedia() {
  const media = new Map<string, {
    matches: boolean
    listeners: Set<() => void>
    addEventListener: ReturnType<typeof vi.fn>
    removeEventListener: ReturnType<typeof vi.fn>
  }>()
  const matchMedia = vi.spyOn(window, 'matchMedia').mockImplementation(query => {
    if (!media.has(query)) {
      const listeners = new Set<() => void>()
      media.set(query, {
        matches: query !== motionQueries.reduced,
        listeners,
        addEventListener: vi.fn((_event: string, callback: () => void) => listeners.add(callback)),
        removeEventListener: vi.fn((_event: string, callback: () => void) => listeners.delete(callback)),
      })
    }
    return media.get(query) as unknown as MediaQueryList
  })
  const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
  return {
    media,
    matchMedia,
    set(query: string, matches: boolean) {
      const item = media.get(query)!
      item.matches = matches
      item.listeners.forEach(callback => callback())
    },
    setVisible(visible: boolean) {
      visibility.mockReturnValue(visible ? 'visible' : 'hidden')
      document.dispatchEvent(new Event('visibilitychange'))
    },
  }
}

export function mockAnimationFrames() {
  let sequence = 0
  const frames = new Map<number, FrameRequestCallback>()
  const request = vi.fn((callback: FrameRequestCallback) => {
    const id = ++sequence
    frames.set(id, callback)
    return id
  })
  const cancel = vi.fn((id: number) => { frames.delete(id) })
  vi.stubGlobal('requestAnimationFrame', request)
  vi.stubGlobal('cancelAnimationFrame', cancel)
  return {
    frames, request, cancel,
    flush() {
      const pending = [...frames.values()]
      frames.clear()
      pending.forEach(callback => callback(16))
    },
  }
}

export function dispatchPointer(element: HTMLElement, type = 'pointermove', x = 50, y = 30, pointerType = 'mouse') {
  const event = new MouseEvent(type, { bubbles: true, clientX: x, clientY: y })
  Object.defineProperty(event, 'pointerType', { value: pointerType })
  element.dispatchEvent(event)
}
