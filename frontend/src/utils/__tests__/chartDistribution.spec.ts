import { describe, expect, it } from 'vitest'
import { distributionValue, rankChartRows, sortDescending } from '../chartDistribution'

describe('chart distribution ranking', () => {
  it('sorts independent copies, preserves ties, and assigns colors before sorting', () => {
    const rows = Object.freeze([
      Object.freeze({ id: 'a', total_tokens: 200, actual_cost: 1 }),
      Object.freeze({ id: 'b', total_tokens: 100, actual_cost: 3 }),
      Object.freeze({ id: 'c', total_tokens: 100, actual_cost: 2 }),
    ])
    const palette = ['#123456', '#abcdef']
    const byTokens = rankChartRows(rows, row => distributionValue(row, 'tokens'), palette)
    const byCost = rankChartRows(rows, row => distributionValue(row, 'actual_cost'), palette)
    expect(byTokens.map(row => row.id)).toEqual(['a', 'b', 'c'])
    expect(byCost.map(row => row.id)).toEqual(['b', 'c', 'a'])
    expect(byCost.map(row => row.chartColor)).toEqual(['#abcdef', '#123456', '#123456'])
    expect(rankChartRows(rows, row => distributionValue(row, 'tokens'), palette)).toEqual(byTokens)
    expect(rows.map(row => row.id)).toEqual(['a', 'b', 'c'])
    expect(rows[0]).not.toHaveProperty('chartColor')
  })

  it('treats non-finite values as zero and keeps a complete palette when empty', () => {
    const rows = [{ value: NaN }, { value: 4 }, { value: undefined }, { value: Infinity }]
    expect(sortDescending(rows, row => row.value).map(row => row.value)).toEqual([4, NaN, undefined, Infinity])
    expect(rankChartRows(rows, row => row.value, []).every(row => /^#[\da-f]{6}$/i.test(row.chartColor))).toBe(true)
  })
})
