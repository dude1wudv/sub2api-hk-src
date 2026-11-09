import DOMPurify from 'dompurify'

/** Render model-produced HTML/SVG in an isolated document without active content or network access. */
export function qualityArtifactDocument(value: string | undefined): string | null {
  if (!value) return null
  const body = value.match(/```(?:html|svg)\s*\n([\s\S]*?)```/i)?.[1] ?? value
  if (!/<(?:svg|html|body|div|canvas|section|article)\b/i.test(body)) return null
  const safe = DOMPurify.sanitize(body, {
    USE_PROFILES: { html: true, svg: true, svgFilters: true },
    FORBID_TAGS: ['script', 'iframe', 'object', 'embed', 'form', 'link', 'meta', 'base'],
    FORBID_ATTR: ['srcdoc'],
  })
  return '<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src &#39;none&#39;; img-src data:; style-src &#39;unsafe-inline&#39;; base-uri &#39;none&#39;; form-action &#39;none&#39;"><style>body{margin:0;padding:12px;font:14px system-ui;overflow-wrap:anywhere}svg,img{max-width:100%;height:auto}</style></head><body>' + safe + '</body></html>'
}
