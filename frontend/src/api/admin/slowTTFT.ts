import { apiClient } from '../client'
export async function clearSlowTTFT(id: number): Promise<void> {
  await apiClient.post(`/admin/accounts/${id}/slow-ttft/clear`)
}
