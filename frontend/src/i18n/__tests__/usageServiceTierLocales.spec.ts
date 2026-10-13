import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('usage service tier locale keys', () => {
  it('contains zh labels for service tier tooltip', () => {
    expect(zh.usage.serviceTier).toBe('服务档位')
    expect(zh.usage.serviceTierPriority).toBe('Fast')
    expect(zh.usage.serviceTierUltrafast).toBe('Ultrafast')
    expect(zh.usage.serviceTierFlex).toBe('Flex')
    expect(zh.usage.serviceTierStandard).toBe('Standard')
  })

  it('contains en labels for service tier tooltip', () => {
    expect(en.usage.serviceTier).toBe('Service tier')
    expect(en.usage.serviceTierPriority).toBe('Fast')
    expect(en.usage.serviceTierUltrafast).toBe('Ultrafast')
    expect(en.usage.serviceTierFlex).toBe('Flex')
    expect(en.usage.serviceTierStandard).toBe('Standard')
  })

  it('exposes localized administrator-only downgrade audit labels', () => {
    expect(zh.admin.usage).toMatchObject({
      serviceTierDowngraded: '上游降档',
      requestedServiceTier: '请求档位',
      upstreamServiceTier: '上游档位',
      billedServiceTier: '计费档位',
    })
    expect(en.admin.usage).toMatchObject({
      serviceTierDowngraded: 'Upstream tier downgrade',
      requestedServiceTier: 'Requested tier',
      upstreamServiceTier: 'Upstream tier',
      billedServiceTier: 'Billed tier',
    })
  })
})
