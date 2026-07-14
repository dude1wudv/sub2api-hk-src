/**
 * Normalize plan features across API string and UI array boundaries.
 * Only escaped line feeds are treated as separators so ordinary backslashes remain intact.
 */
export function normalizePlanFeatures(value: string | string[] | null | undefined): string[] {
  const text = Array.isArray(value) ? value.join('\n') : value || ''
  return text
    .replace(/\r\n?|\\r\\n|\\n/g, '\n')
    .split('\n')
    .map((feature) => feature.trim())
    .filter(Boolean)
}

export function serializePlanFeatures(value: string | string[] | null | undefined): string {
  return normalizePlanFeatures(value).join('\n')
}