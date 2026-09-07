import { ref, type Ref } from 'vue'

export type TableDensity = 'compact' | 'comfortable'
export type TableIdentity = {
  routeId: string
  tableId: string
  /** Optional explicit user scope. When omitted, the signed-in user ID is read safely. */
  userId?: string | number | null
}

export type PersistedFilterValue = string | number | boolean | null | Array<string | number | boolean | null>
export type SavedTableFilter = {
  id: string
  name: string
  filters: Record<string, PersistedFilterValue>
}

type PersistedTableView = {
  version: 1
  density: TableDensity
  visibleColumns: string[]
}

type PersistedTableQuery = {
  version: 1
  filters: Record<string, PersistedFilterValue>
  sort?: { key: string; order: 'asc' | 'desc' }
  page?: number
  pageSize?: number
}

type PersistedSavedFilters = {
  version: 1
  filters: SavedTableFilter[]
}

const STORAGE_PREFIX = 'sub2api:table:v1'
const SENSITIVE_KEY = /(?:api_?key|(?:^|_)key(?:$|_)|pass(?:word)?|secret|token|credential|authorization|cookie|session|payload|body|prompt)/i
const MAX_FILTER_KEYS = 32
const MAX_ARRAY_VALUES = 30
const MAX_ID_LENGTH = 256

function storageAvailable(): boolean {
  try {
    return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined'
  } catch {
    return false
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}


function resolveUserScope(userId: TableIdentity['userId']): string | null {
  if (typeof userId === 'string' && userId.trim()) return userId.trim().slice(0, MAX_ID_LENGTH)
  if (typeof userId === 'number' && Number.isFinite(userId)) return String(userId)
  if (!storageAvailable()) return null

  try {
    const storedUser = JSON.parse(window.localStorage.getItem('auth_user') || 'null') as unknown
    if (!isRecord(storedUser)) return null
    const id = storedUser.id
    if (typeof id === 'string' && id.trim()) return id.trim().slice(0, MAX_ID_LENGTH)
    if (typeof id === 'number' && Number.isFinite(id)) return String(id)
  } catch {
    // The auth session may be unavailable or malformed; do not share query state.
  }
  return null
}

export function getTablePreferenceKey(
  identity: TableIdentity,
  scope: 'view' | 'query' | 'filters' = 'view'
): string | null {
  const routeId = identity.routeId.trim()
  const tableId = identity.tableId.trim()
  if (!routeId || !tableId) return null

  const userScope = resolveUserScope(identity.userId)
  // Query and saved filters may contain user-specific criteria. Never share them between sessions.
  if (!userScope && scope !== 'view') return null
  const namespace = userScope ? `user:${userScope}` : 'ui'
  return `${STORAGE_PREFIX}:${scope}:${encodeURIComponent(namespace)}:${encodeURIComponent(routeId)}:${encodeURIComponent(tableId)}`
}

function readJson(key: string | null): unknown {
  if (!key || !storageAvailable()) return null
  try {
    const raw = window.localStorage.getItem(key)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

function writeJson(key: string | null, value: unknown): void {
  if (!key || !storageAvailable()) return
  try {
    window.localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // Preferences are optional; privacy mode and storage quotas are both supported.
  }
}

function isPersistableFilterKey(key: string): boolean {
  // IDs are metadata, not their corresponding secret values (for example api_key_id).
  return key.endsWith('_id') || !SENSITIVE_KEY.test(key)
}

function isPrimitive(value: unknown): value is string | number | boolean | null {
  return value === null
    || typeof value === 'string'
    || typeof value === 'boolean'
    || (typeof value === 'number' && Number.isFinite(value))
}

function sanitizeFilterValue(value: unknown): PersistedFilterValue | undefined {
  if (isPrimitive(value)) return value
  if (Array.isArray(value) && value.length <= MAX_ARRAY_VALUES && value.every(isPrimitive)) return [...value]
  return undefined
}

function sanitizeFilters(source: unknown, allowedKeys: readonly string[]): Record<string, PersistedFilterValue> {
  if (!isRecord(source)) return {}
  const allowedKeysSet = new Set(allowedKeys.filter(isPersistableFilterKey).slice(0, MAX_FILTER_KEYS))
  const result: Record<string, PersistedFilterValue> = {}
  for (const key of allowedKeysSet) {
    const value = sanitizeFilterValue(source[key])
    if (value !== undefined) result[key] = value
  }
  return result
}

function parseView(value: unknown): PersistedTableView | null {
  if (!isRecord(value) || value.version !== 1 || !Array.isArray(value.visibleColumns)) return null
  return {
    version: 1,
    density: value.density === 'comfortable' ? 'comfortable' : 'compact',
    visibleColumns: value.visibleColumns.filter((column): column is string => typeof column === 'string')
  }
}

function parseQuery(value: unknown): PersistedTableQuery | null {
  if (!isRecord(value) || value.version !== 1) return null
  const sort = isRecord(value.sort) && typeof value.sort.key === 'string'
    ? { key: value.sort.key, order: value.sort.order === 'desc' ? 'desc' as const : 'asc' as const }
    : undefined
  return {
    version: 1,
    filters: sanitizeFilters(value.filters, Object.keys(isRecord(value.filters) ? value.filters : {})),
    ...(sort ? { sort } : {}),
    ...(typeof value.page === 'number' && Number.isInteger(value.page) && value.page > 0 ? { page: value.page } : {}),
    ...(typeof value.pageSize === 'number' && Number.isInteger(value.pageSize) && value.pageSize > 0 ? { pageSize: value.pageSize } : {})
  }
}

function parseSavedFilters(value: unknown, allowedKeys: readonly string[], maxSaved: number): SavedTableFilter[] {
  if (!isRecord(value) || value.version !== 1 || !Array.isArray(value.filters)) return []
  return value.filters.flatMap((item) => {
    if (!isRecord(item) || typeof item.id !== 'string' || typeof item.name !== 'string') return []
    const id = item.id.slice(0, 64)
    const name = item.name.trim().slice(0, 80)
    return id && name ? [{ id, name, filters: sanitizeFilters(item.filters, allowedKeys) }] : []
  }).slice(0, maxSaved)
}

export interface UsePersistedTableViewOptions extends TableIdentity {
  columnKeys: readonly string[]
  nonHideableColumnKeys?: readonly string[]
  defaultHiddenColumnKeys?: readonly string[]
  defaultDensity?: TableDensity
}

export function usePersistedTableView(options: UsePersistedTableViewOptions) {
  const allowedColumns = new Set(options.columnKeys)
  const nonHideableColumns = new Set(options.nonHideableColumnKeys ?? [])
  const defaultHiddenColumns = new Set(options.defaultHiddenColumnKeys ?? [])
  const stored = parseView(readJson(getTablePreferenceKey(options)))
  const storedColumns = stored?.visibleColumns.filter((column) => allowedColumns.has(column)) ?? []
  const visibleColumnKeys = ref(
    storedColumns.length > 0
      ? options.columnKeys.filter((column) => storedColumns.includes(column) || nonHideableColumns.has(column))
      : options.columnKeys.filter((column) => !defaultHiddenColumns.has(column) || nonHideableColumns.has(column))
  )
  const density = ref<TableDensity>(stored?.density ?? options.defaultDensity ?? 'compact')

  const persist = () => {
    writeJson(getTablePreferenceKey(options), {
      version: 1,
      density: density.value,
      visibleColumns: visibleColumnKeys.value.filter((column) => allowedColumns.has(column))
    } satisfies PersistedTableView)
  }

  const setVisibleColumns = (columns: readonly string[]) => {
    const selectedColumns = new Set(columns)
    visibleColumnKeys.value = options.columnKeys.filter(
      (column) => selectedColumns.has(column) || nonHideableColumns.has(column)
    )
    persist()
  }

  const toggleColumn = (column: string, visible?: boolean) => {
    if (!allowedColumns.has(column) || nonHideableColumns.has(column)) return
    const selectedColumns = new Set(visibleColumnKeys.value)
    if (visible ?? !selectedColumns.has(column)) selectedColumns.add(column)
    else selectedColumns.delete(column)
    setVisibleColumns([...selectedColumns])
  }

  const setDensity = (nextDensity: TableDensity) => {
    density.value = nextDensity === 'comfortable' ? 'comfortable' : 'compact'
    persist()
  }

  const reset = () => {
    visibleColumnKeys.value = options.columnKeys.filter(
      (column) => !defaultHiddenColumns.has(column) || nonHideableColumns.has(column)
    )
    density.value = options.defaultDensity ?? 'compact'
    persist()
  }

  const syncColumns = (columnKeys: readonly string[], nonHideableColumnKeys: readonly string[] = []) => {
    const previousColumns = new Set(allowedColumns)
    allowedColumns.clear()
    columnKeys.forEach((column) => allowedColumns.add(column))
    nonHideableColumns.clear()
    nonHideableColumnKeys.forEach((column) => nonHideableColumns.add(column))
    const selectedColumns = new Set(visibleColumnKeys.value)
    columnKeys.forEach((column) => {
      if (!previousColumns.has(column) && !defaultHiddenColumns.has(column)) selectedColumns.add(column)
    })
    visibleColumnKeys.value = columnKeys.filter(
      (column) => selectedColumns.has(column) || nonHideableColumns.has(column)
    )
  }

  return { density, visibleColumnKeys, setDensity, setVisibleColumns, toggleColumn, reset, syncColumns, persist }
}

export interface UsePersistedTableQueryOptions<T extends Record<string, unknown>> extends TableIdentity {
  filters: T
  filterKeys: readonly (keyof T & string)[]
  sort?: { key: string; order: 'asc' | 'desc' }
  page?: Ref<number>
  pageSize?: Ref<number>
  /** Compatibility for existing reactive pagination objects. Prefer page/pageSize refs in new code. */
  pagination?: { page: number; page_size?: number }
}

export function usePersistedTableQuery<T extends Record<string, unknown>>(options: UsePersistedTableQueryOptions<T>) {
  const restore = () => {
    const stored = parseQuery(readJson(getTablePreferenceKey(options, 'query')))
    if (!stored) return false
    Object.assign(options.filters, sanitizeFilters(stored.filters, options.filterKeys))
    if (options.sort && stored.sort) {
      options.sort.key = stored.sort.key
      options.sort.order = stored.sort.order
    }
    if (stored.page) {
      if (options.page) options.page.value = stored.page
      else if (options.pagination) options.pagination.page = stored.page
    }
    if (stored.pageSize) {
      if (options.pageSize) options.pageSize.value = stored.pageSize
      else if (options.pagination) options.pagination.page_size = stored.pageSize
    }
    return true
  }

  const persist = () => {
    const page = options.page?.value ?? options.pagination?.page
    const pageSize = options.pageSize?.value ?? options.pagination?.page_size
    writeJson(getTablePreferenceKey(options, 'query'), {
      version: 1,
      filters: sanitizeFilters(options.filters, options.filterKeys),
      ...(options.sort?.key ? { sort: { key: options.sort.key, order: options.sort.order === 'desc' ? 'desc' : 'asc' } } : {}),
      ...(typeof page === 'number' && Number.isInteger(page) && page > 0 ? { page } : {}),
      ...(typeof pageSize === 'number' && Number.isInteger(pageSize) && pageSize > 0 ? { pageSize } : {})
    } satisfies PersistedTableQuery)
  }

  const reset = () => {
    const key = getTablePreferenceKey(options, 'query')
    if (!key || !storageAvailable()) return
    try {
      window.localStorage.removeItem(key)
    } catch {
      // Preferences are optional.
    }
  }

  return { restore, persist, reset }
}

export interface UseSavedTableFiltersOptions<T extends Record<string, unknown>> extends TableIdentity {
  filterKeys: readonly (keyof T & string)[]
  maxSaved?: number
}

export function useSavedTableFilters<T extends Record<string, unknown>>(options: UseSavedTableFiltersOptions<T>) {
  const maxSaved = Math.max(1, Math.min(options.maxSaved ?? 8, 20))
  const savedFilters = ref(parseSavedFilters(readJson(getTablePreferenceKey(options, 'filters')), options.filterKeys, maxSaved))

  const persist = () => {
    writeJson(getTablePreferenceKey(options, 'filters'), { version: 1, filters: savedFilters.value } satisfies PersistedSavedFilters)
  }

  const save = (name: string, filters: T): SavedTableFilter | null => {
    const normalizedName = name.trim().slice(0, 80)
    if (!normalizedName) return null
    const savedFilter: SavedTableFilter = {
      id: `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
      name: normalizedName,
      filters: sanitizeFilters(filters, options.filterKeys)
    }
    savedFilters.value = [savedFilter, ...savedFilters.value].slice(0, maxSaved)
    persist()
    return savedFilter
  }

  const apply = (id: string): Record<string, PersistedFilterValue> | null => {
    const savedFilter = savedFilters.value.find((item) => item.id === id)
    return savedFilter ? { ...savedFilter.filters } : null
  }

  const remove = (id: string) => {
    const nextFilters = savedFilters.value.filter((item) => item.id !== id)
    if (nextFilters.length === savedFilters.value.length) return
    savedFilters.value = nextFilters
    persist()
  }

  return { savedFilters, save, apply, remove }
}
