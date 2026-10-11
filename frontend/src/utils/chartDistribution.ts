export type DistributionMetric = 'tokens' | 'actual_cost'

export function finiteChartValue(value: unknown): number {
  const number = Number(value)
  return Number.isFinite(number) ? number : 0
}

export function distributionValue(
  row: { total_tokens: number; actual_cost: number },
  metric: DistributionMetric,
): number {
  return finiteChartValue(metric === 'actual_cost' ? row.actual_cost : row.total_tokens)
}

export function sortDescending<T>(rows: readonly T[], value: (row: T) => unknown): T[] {
  return [...rows].sort((a, b) => finiteChartValue(value(b)) - finiteChartValue(value(a)))
}

/** Assign colors before ranking, so changing the metric keeps each row's color. */
export function rankChartRows<T>(
  rows: readonly T[],
  value: (row: T) => unknown,
  palette: readonly string[],
): Array<T & { chartColor: string }> {
  const coloredRows = rows.map((row, index) => ({
    ...row,
    chartColor: palette[index % palette.length] || '#6f7068',
  }))
  return sortDescending(coloredRows, value)
}
