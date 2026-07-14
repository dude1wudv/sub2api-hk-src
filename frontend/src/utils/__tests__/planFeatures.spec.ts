import { describe, expect, it } from 'vitest'
import { normalizePlanFeatures, serializePlanFeatures } from '../planFeatures'

describe('plan feature normalization', () => {
  it('accepts API strings and arrays with actual or escaped line endings', () => {
    expect(normalizePlanFeatures(['First\\nSecond', 'Third\r\nFourth', 'Path C:\\temp'])).toEqual([
      'First',
      'Second',
      'Third',
      'Fourth',
      'Path C:\\temp',
    ])
  })

  it('serializes normalized entries for the admin API', () => {
    expect(serializePlanFeatures('First\\r\\nSecond\n\nThird')).toBe('First\nSecond\nThird')
  })
})