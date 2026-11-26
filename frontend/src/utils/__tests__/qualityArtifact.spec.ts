import { describe, expect, it } from 'vitest'
import { qualityArtifactDocument, qualityArtifactPlayableDocument } from '../qualityArtifact'

describe('qualityArtifactDocument', () => {
  it('keeps plain answers as text and sanitizes isolated SVG output', () => {
    expect(qualityArtifactDocument('A normal textual answer.')).toBeNull()
    const doc = qualityArtifactDocument('```svg\n<svg onload="alert(1)"><script>alert(1)</script><circle cx="10" cy="10" r="8" /></svg>\n```')!
    expect(doc).toContain('<circle')
    expect(doc).not.toContain('onload')
    expect(doc).not.toContain('<script')
    expect(doc).toContain('default-src &#39;none&#39;')
    expect(doc).toContain('form-action &#39;none&#39;')
  })
})


describe('qualityArtifactPlayableDocument', () => {
  it('uses a nonempty script nonce even when the browser hides its attribute', () => {
    const empty = document.createElement('script')
    empty.setAttribute('nonce', '')
    const bootstrap = document.createElement('script')
    Object.defineProperty(bootstrap, 'nonce', { value: 'host-nonce' })
    document.head.append(empty, bootstrap)
    try {
      const html = qualityArtifactPlayableDocument('<main>art</main><script>startRide()</script>')!
      expect(html.match(/nonce="host-nonce"/g)).toHaveLength(2)
    } finally {
      empty.remove()
      bootstrap.remove()
    }
  })

  it('keeps inline animation with the host nonce while dropping external and embedded documents', () => {
    const script = document.createElement('script'); script.nonce = 'host-nonce'; document.head.append(script)
    const html = qualityArtifactPlayableDocument('<html><head><meta http-equiv="refresh" content="0;url=https://elsewhere.test"><script src="https://elsewhere.test/x.js"></script></head><body><div onclick="this.textContent=1">art</div><script>requestAnimationFrame(()=>{})</script><iframe src="https://elsewhere.test"></iframe></body></html>')!
    expect(html).toContain('nonce="host-nonce"')
    expect(html).toContain("script-src 'unsafe-inline'")
    expect(html).toContain('requestAnimationFrame')
    expect(html).toContain("addEventListener(\"click\"")
    expect(html).not.toContain('onclick=')
    expect(html).not.toContain('https://elsewhere.test')
    expect(html).toContain("connect-src 'none'")
    expect(html).toContain("frame-src 'none'")
    expect(html.indexOf('Content-Security-Policy')).toBeLessThan(html.indexOf('requestAnimationFrame'))
    script.remove()
  })

  it('preserves document attributes and converts body load into a window listener', () => {
    const html = qualityArtifactPlayableDocument(
      '<html lang="zh" class="ride"><head><style>.ride .ready{color:red}</style></head><body class="ready" onload="startRide()"><main>art</main></body></html>',
    )!
    expect(html).toContain('<html lang="zh" class="ride">')
    expect(html).toContain('<body class="ready"')
    expect(html).toContain('window?.addEventListener("load"')
    expect(html).not.toContain('onload=')
    expect(html.indexOf('Content-Security-Policy')).toBeLessThan(html.indexOf('<style>'))
  })

  it('enables full script and stylesheet motion before artifact initialization', () => {
    const source = `<html><head>
      <style media="screen and (prefers-reduced-motion: no-preference)">
        /* @media (prefers-reduced-motion: reduce) {} */
        .label::before { content: "@media (prefers-reduced-motion: reduce)"; }
        @supports (display: grid) {
          @media (prefers-reduced-motion: reduce), (width < 500px) { .ride { animation: none; } }
        }
      </style>
      <script>startRide()</script></head><body><main>art</main></body></html>`
    const html = qualityArtifactPlayableDocument(source)!
    expect(html.indexOf('Content-Security-Policy')).toBeLessThan(html.indexOf('window.matchMedia'))
    expect(html.indexOf('window.matchMedia')).toBeLessThan(html.indexOf('startRide()'))
    expect(html).toContain('media="screen and (min-width: 0px)"')
    expect(html).toContain('@media (max-width: -1px), (width < 500px)')
    expect(html).toContain('/* @media (prefers-reduced-motion: reduce) {} */')
    expect(html).toContain('content: "@media (prefers-reduced-motion: reduce)"')
  })
})
