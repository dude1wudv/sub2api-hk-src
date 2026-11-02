/** Keep the IANA zone for queries; derive the offset at the displayed instant (including DST). */
export function timezoneLabel(zone: string, at: Date = new Date()): string {
  try {
    const parts = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'longOffset' }).formatToParts(at)
    const offset = parts.find(part => part.type === 'timeZoneName')?.value || 'GMT'
    return offset.replace(/^GMT/, 'UTC').replace(/\+0?(\d{1,2}):00$/, '+$1').replace(/-0?(\d{1,2}):00$/, '−$1').replace('-', '−')
  } catch { return 'UTC' }
}
