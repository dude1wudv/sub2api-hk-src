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

/** Active outputs run only inside an opaque-origin iframe with sandbox=allow-scripts. */
export function qualityArtifactPlayableDocument(value: string | undefined): string | null {
  if (!value) return null
  const source = value.match(/```(?:html|svg)\s*\n([\s\S]*?)```/i)?.[1] ?? value
  if (!/<(?:svg|html|body|div|canvas|section|article)\b/i.test(source)) return null
  const template = document.createElement('template')
  template.innerHTML = source
  const fragment = template.content
  fragment.querySelectorAll('base,meta,iframe,frame,frameset,object,embed,form,link').forEach(node => node.remove())
  const nonce = document.querySelector<HTMLScriptElement>('script[nonce]')?.nonce || ''
  fragment.querySelectorAll('script').forEach(script => {
    if (script.hasAttribute('src') || script.hasAttribute('href') || script.hasAttribute('xlink:href')) script.remove()
    else if (nonce) script.setAttribute('nonce', nonce)
  })
  const handlers: string[] = []
  fragment.querySelectorAll('*').forEach((element, index) => {
    for (const attr of [...element.attributes]) {
      if (!/^on[a-z]+$/i.test(attr.name)) continue
      element.removeAttribute(attr.name)
      const type = attr.name.slice(2).toLowerCase()
      element.setAttribute('data-quality-handler', String(index))
      const target = `document.querySelector('[data-quality-handler="${index}"]')`
      const code = attr.value.replace(/<\/script/gi, '<\\/script')
      handlers.push(`${target}?.addEventListener(${JSON.stringify(type)},function(event){const result=(function(event){\n${code}\n}).call(this,event);if(result===false){event.preventDefault();event.stopPropagation()}});`)
    }
  })
  if (handlers.length) {
    const script = document.createElement('script')
    if (nonce) script.setAttribute('nonce', nonce)
    script.textContent = handlers.join('\n')
    fragment.append(script)
  }
  const csp = `default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; connect-src 'none'; frame-src 'none'; child-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; media-src data:; font-src data:`
  // The trusted policy precedes all source content. Original metadata cannot replace it.
  return '<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="' + csp.replace(/"/g, '&quot;') + '">' + '</head><body>' + template.innerHTML + '</body></html>'
}
