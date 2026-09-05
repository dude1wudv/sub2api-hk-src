interface AccountIDRow {
  id: number
}

interface AccountListPage {
  items: AccountIDRow[]
  total: number
  pages?: number
}

type AccountPageFetcher = (
  page: number,
  pageSize: number,
  filters: Record<string, unknown>,
  options?: { signal?: AbortSignal }
) => Promise<AccountListPage>

export interface AccountSelectionProgress {
  completedPages: number
  totalPages: number
  selectedCount: number
}

export interface FetchAllAccountIdsOptions {
  signal?: AbortSignal
  onProgress?: (progress: AccountSelectionProgress) => void
}

export class AccountSelectionCancelledError extends Error {
  constructor() {
    super('Account selection cancelled')
    this.name = 'AccountSelectionCancelledError'
  }
}

const SELECT_ALL_PAGE_SIZE = 1000

const throwIfAborted = (signal?: AbortSignal) => {
  if (signal?.aborted) throw new AccountSelectionCancelledError()
}

export async function fetchAllAccountIds(
  fetchPage: AccountPageFetcher,
  filters: Record<string, unknown>,
  options: FetchAllAccountIdsOptions = {}
): Promise<number[]> {
  const requestFilters = {
    ...filters,
    lite: '1',
    include_scheduler_score: '0'
  }
  const { signal, onProgress } = options

  throwIfAborted(signal)
  const firstPage = await (signal
    ? fetchPage(1, SELECT_ALL_PAGE_SIZE, requestFilters, { signal })
    : fetchPage(1, SELECT_ALL_PAGE_SIZE, requestFilters))
  throwIfAborted(signal)

  const pageCount = Math.max(
    1,
    firstPage.pages ?? 0,
    Math.ceil(firstPage.total / SELECT_ALL_PAGE_SIZE)
  )
  const ids = firstPage.items.map(account => account.id)
  onProgress?.({
    completedPages: 1,
    totalPages: pageCount,
    selectedCount: ids.length
  })

  for (let page = 2; page <= pageCount; page++) {
    throwIfAborted(signal)
    const result = await (signal
      ? fetchPage(page, SELECT_ALL_PAGE_SIZE, requestFilters, { signal })
      : fetchPage(page, SELECT_ALL_PAGE_SIZE, requestFilters))
    throwIfAborted(signal)
    ids.push(...result.items.map(account => account.id))
    onProgress?.({
      completedPages: page,
      totalPages: pageCount,
      selectedCount: ids.length
    })
  }

  const uniqueIDs = Array.from(new Set(ids))
  if (uniqueIDs.length !== firstPage.total) {
    throw new Error('账号列表结果不完整')
  }
  return uniqueIDs
}
