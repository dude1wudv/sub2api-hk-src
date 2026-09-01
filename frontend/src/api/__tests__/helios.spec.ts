import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

import { launchHelios } from '../helios'

describe('launchHelios API wrapper', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('requests only a one-time launch URL and exposes no API key field', async () => {
    const response = { launch_url: 'https://canvas.sub.sunmmyapi.xyz/bootstrap#code=one-time', expires_in: 90 }
    post.mockResolvedValue({ data: response })

    await expect(launchHelios()).resolves.toEqual(response)
    expect(post).toHaveBeenCalledWith('/workbenches/helios/launch')
    expect(Object.keys(response)).not.toContain('apiKey')
    expect(Object.keys(response)).not.toContain('api_key')
  })
})
