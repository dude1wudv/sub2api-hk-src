import { readdirSync, readFileSync, statSync } from 'node:fs'
import { relative, resolve } from 'node:path'
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
  it('keeps popovers and code panels readable', () => {
    expect(contrast('--ink', '--surface-elevated', palette)).toBeGreaterThanOrEqual(4.5)
    expect(contrast('--ink', '--surface-2', palette)).toBeGreaterThanOrEqual(4.5)
  })
})

describe('theme surfaces', () => {
  function hexLuminance(hex: string): number {
    const channels = [0, 2, 4].map(offset => Number.parseInt(hex.slice(offset + 1, offset + 3), 16)).map(channel => {
      const value = channel / 255
      return value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4
    })
    return channels.reduce((sum, channel, index) => sum + channel * [.2126, .7152, .0722][index], 0)
  }
  function hexContrast(foreground: string, background: string): number {
    const a = hexLuminance(foreground)
    const b = hexLuminance(background)
    return (Math.max(a, b) + .05) / (Math.min(a, b) + .05)
  }

  it('uses light chart tooltip surfaces and keeps both modes readable', () => {
    expect(light['--chart-tooltip-bg']).toBe('#fffffc')
    expect(light['--chart-tooltip-text']).toBe('#22231f')
    expect(light['--chart-tooltip-border']).toBe('#e8e4da')
    expect(hexContrast(light['--chart-tooltip-text'], light['--chart-tooltip-bg'])).toBeGreaterThanOrEqual(4.5)
    expect(hexContrast(dark['--chart-tooltip-text'], dark['--chart-tooltip-bg'])).toBeGreaterThanOrEqual(4.5)
  })

  it('defines popover and code panel colors from surface tokens', () => {
    expect(stylesheet).toMatch(/\.ui-popover \{[^}]*rgb\(var\(--surface-elevated\)\)[^}]*rgb\(var\(--ink\)\)/)
    expect(stylesheet).toMatch(/\.ui-popover-arrow \{[^}]*rgb\(var\(--surface-elevated\)\)/)
    expect(stylesheet).toMatch(/\.ui-code-panel \{[^}]*rgb\(var\(--surface-2\)\)[^}]*rgb\(var\(--ink\)\)/)
  })
})

describe('light mode surfaces', () => {
  const solidDarkBackground = /(?:[\w-]+:)*!?(?:bg-(?:gray|dark)-(?:700|800|900|950)|bg-black)(?![\w/-])/g
  // Video letterboxing stays black in both modes.
  const allowedVideoSurface = 'components/admin/account/AccountTestModal.vue'

  function vueFiles(dir: string): string[] {
    return readdirSync(dir).flatMap(name => {
      const full = resolve(dir, name)
      return statSync(full).isDirectory() ? vueFiles(full) : name.endsWith('.vue') ? [full] : []
    })
  }

  it('does not use a solid dark background without a dark-mode prefix', () => {
    const root = resolve(process.cwd(), 'src')
    const violations: string[] = []
    for (const file of vueFiles(root)) {
      const rel = relative(root, file).replaceAll('\\', '/')
      readFileSync(file, 'utf8').split('\n').forEach((line, index) => {
        for (const match of line.matchAll(solidDarkBackground)) {
          if (match[0].includes('dark:')) continue
          if (rel === allowedVideoSurface && match[0] === 'bg-black' && line.includes('border-gray-200 bg-black')) continue
          violations.push(`${rel}:${index + 1} ${match[0]}`)
        }
      })
    }
    expect(violations).toEqual([])
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
