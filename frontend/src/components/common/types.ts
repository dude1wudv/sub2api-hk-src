/**
 * Common component types
 */

export type ColumnAlignment = 'left' | 'center' | 'right'

export interface Column {
  key: string
  label: string
  sortable?: boolean
  /** Cell content alignment. Numeric columns default to right alignment. */
  align?: ColumnAlignment
  /** Render the default cell in a monospace, right-aligned numeric style. */
  numeric?: boolean
  /** Render the default cell in a monospace style. */
  mono?: boolean
  /** Whether users may hide this column in a persisted table view. */
  hideable?: boolean
  class?: string
  formatter?: (value: any, row: any) => string
}
