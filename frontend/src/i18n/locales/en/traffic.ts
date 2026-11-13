// Adapted from ranxi2001/sub2api @ 9620d315, licensed under LGPL-3.0-or-later.
export default {
  overview: 'Traffic overview', noCharge: 'Statistics only · No charges', composition: 'Traffic breakdown', average: 'Average per request', peakBucket: 'Peak bucket',
  totalHint: 'Uploaded + downloaded payload', uploadHint: 'Client → gateway', downloadHint: 'Gateway → client', requestsHint: 'Includes failed and canceled requests',
  scopeTitle: 'Measurement & recording notes', scopeShort: 'HTTP / SSE · Delayed · May be incomplete', emptyHint: 'Try another date or API key. Previously uncollected traffic cannot be recovered.', recordsHint: 'By request start time, including failures and cancellations.',
  identityFailed: 'User search failed. Please search again.',
  loadingRecords: 'Loading request details…', recordsFailed: 'Could not load request details. Please retry.',
  queueSummary: 'Memory queue: {queued}/{capacity} · Processing: {processing} · Dropped since restart: {dropped}',
  storageUnavailable: 'Traffic statistics storage is unavailable. Business requests continue; statistics may be incomplete.',
  trendScopes: { hour: 'Each bar shows upload plus download within that hour. Hours are independent, not cumulative.', day: 'Each bar shows upload plus download within that day. Days are independent, not cumulative.', month: 'Each bar shows upload plus download within that month. Months are independent, not cumulative.' },
  deletedKey: 'Deleted', keyHistoryHint: 'Search names or #ID, including deleted keys.', keysFailed: 'Could not load key choices. Retry the search.',
  nodeHealth: 'Traffic recording on this node', nodeHealthScope: 'Node-local status; counters reset on restart', healthStatuses: { ok: 'Healthy', delayed: 'Delayed', degraded: 'Needs attention' },
  healthSummary: 'Pending: {pending} · Oldest pending: {age}s · Quarantined: {quarantined} · Persistence failure alerts since restart: {lost}', healthFailed: 'Recording health is unavailable. Refresh to retry.',
  title: 'Traffic records', scope: 'HTTP/SSE payload traffic for authenticated requests, including failures and cancellations. Excludes WebSocket, Live, headers and network overhead. No traffic charges; records may be delayed or incomplete.',
  period: 'Period', periods: { day: 'Day', week: 'Week', month: 'Month', year: 'Year' }, date: 'Date within period', apply: 'Apply',
  previousPeriod: 'Previous period', nextPeriod: 'Next period', upload: 'Upload', download: 'Download', total: 'Total traffic', requests: 'Requests',
  since: 'First traffic record for these filters: {time}.', notRecorded: 'No traffic has been recorded for these filters.', history: 'Previously uncollected traffic cannot be recovered. Requests belong to their start time and are recorded after completion; asynchronous statistics may be delayed or incomplete during overload or failures.',
  trend: 'Traffic trend', trendDetails: 'View trend details', bucket: 'Time', records: 'Request records', user: 'User', endpoint: 'Endpoint / request ID', result: 'Result', partialRead: 'Request body partially read',
  outcomes: { completed: 'HTTP handling completed', http_error: 'HTTP error', canceled: 'Canceled', write_error: 'Response write failed', panic: 'Handler exception' },
  empty: 'No recorded traffic in this period', loading: 'Loading traffic records…', failed: 'Failed to load traffic records. Please retry.', page: 'Page {page}', previous: 'Previous', next: 'Next'
}
