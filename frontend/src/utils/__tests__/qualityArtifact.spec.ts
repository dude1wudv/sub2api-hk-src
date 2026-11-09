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
})
