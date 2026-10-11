import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import postcss from 'postcss'

const stylesheet = readFileSync(resolve(process.cwd(), 'src/styles/design-system.css'), 'utf8')
function tokens(selector: string): Record<string, string> {
  const block = stylesheet.slice(stylesheet.indexOf(`${selector} {`)).split('}')[0]
  return Object.fromEntries([...block.matchAll(/(--[\w-]+):\s*([^;]+);/g)].map(([, key, value]) => [key, value.trim()]))
}
const light = tokens(':root')
const dark = { ...light, ...tokens(':root.dark') }
function luminance(name: string, palette: Record<string, string>): number {
  const value = palette[name].replace(/var\((--[\w-]+)\)/g, (_, key: string) => palette[key])
  const channels = value.split(/\s+/).map(Number).map(channel => {
    const v = channel / 255
    return v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4
  })
  return channels.reduce((sum, channel, index) => sum + channel * [.2126, .7152, .0722][index], 0)
}
function contrast(foreground: string, background: string, palette: Record<string, string>): number {
  const a = luminance(foreground, palette)
  const b = luminance(background, palette)
  return (Math.max(a, b) + .05) / (Math.min(a, b) + .05)
}

describe.each([['light', light], ['dark', dark]] as const)('%s semantic text contrast', (_, palette) => {
  it.each(['--canvas', '--surface', '--surface-2', '--surface-elevated'])('keeps normal and supporting text readable on %s', background => {
    for (const foreground of ['--ink', '--ink-2', '--ink-3']) {
      expect(contrast(foreground, background, palette)).toBeGreaterThanOrEqual(4.5)
    }
  })
  it('keeps primary actions and status labels readable', () => {
    expect(contrast('--action-text', '--action-bg', palette)).toBeGreaterThanOrEqual(4.5)
    for (const status of ['ok', 'warn', 'err', 'link']) {
      expect(contrast(`--text-${status}`, `--status-${status}-bg`, palette)).toBeGreaterThanOrEqual(4.5)
    }
  })
})

describe('mobile switch targets', () => {
  function renderMobileControls(size: 'regular' | 'small') {
    const style = document.createElement('style')
    // jsdom does not evaluate media queries, so apply the real mobile declarations.
    const rules: string[] = []
    postcss.parse(stylesheet).walkAtRules('media', rule => {
      if (rule.params === '(max-width: 767px)') {
        rules.push(...(rule.nodes || []).map(node => node.toString()))
      }
    })
    style.textContent = rules.join('\n')
    document.head.append(style)
    const container = document.createElement('div')
    container.className = 'modal-content'
    const dimensions = size === 'regular' ? 'h-6 w-11' : 'h-5 w-9'
    container.innerHTML = `<button type="button" role="switch" aria-checked="false" class="relative inline-flex ${dimensions} rounded-full border-2"><span class="pointer-events-none rounded-full"></span></button><button class="${dimensions} rounded-full border-2">Other action</button>`
    document.body.append(container)
    return {
      style,
      track: container.querySelector<HTMLButtonElement>('[role="switch"]')!,
      ordinary: container.querySelectorAll('button')[1],
      cleanup: () => { container.remove(); style.remove() },
    }
  }

  it.each([
    ['regular', '44px', '24px', '10px', '0'],
    ['small', '36px', '20px', '12px', '4px'],
  ] as const)('keeps the %s track compact and reserves a full touch footprint', (size, width, height, blockMargin, inlineMargin) => {
    const fixture = renderMobileControls(size)
    try {
      const trackStyle = getComputedStyle(fixture.track)
      expect(trackStyle.width).toBe(width)
      expect(trackStyle.height).toBe(height)
      expect(trackStyle.minWidth).toBe(width)
      expect(trackStyle.minHeight).toBe(height)
      expect(trackStyle.alignItems).toBe('center')
      expect(trackStyle.marginBlock).toBe(blockMargin)
      expect(trackStyle.marginInline).toBe(inlineMargin)
      expect(parseFloat(height) + 2 * parseFloat(blockMargin)).toBe(44)
      expect(parseFloat(width) + 2 * parseFloat(inlineMargin)).toBe(44)
      // Matching dimensions alone must not turn an ordinary button into a switch.
      expect(getComputedStyle(fixture.ordinary).minHeight).toBe('44px')

      const hitArea = Array.from(fixture.style.sheet!.cssRules)
        .filter((rule): rule is CSSStyleRule => 'selectorText' in rule)
        .find(rule => rule.selectorText.endsWith('::after') && fixture.track.matches(rule.selectorText.slice(0, -7)))
      expect(hitArea?.style.width).toBe('44px')
      expect(hitArea?.style.height).toBe('44px')
      expect(hitArea?.style.background).toBe('transparent')
      expect(hitArea?.style.transform).toBe('translate(-50%,-50%)')
    } finally {
      fixture.cleanup()
    }
  })
})
