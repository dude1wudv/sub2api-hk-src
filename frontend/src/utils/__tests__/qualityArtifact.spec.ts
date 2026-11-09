import { describe, expect, it } from 'vitest'
import { qualityArtifactDocument } from '../qualityArtifact'

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
