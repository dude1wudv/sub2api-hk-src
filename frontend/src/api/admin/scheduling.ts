import { apiClient } from '../client'
export interface SchedulingAccount { account_id: number; name: string; priority: number; load_factor: number; concurrency: number }
export interface GroupScheduling { version: string; accounts: SchedulingAccount[] }
export async function getGroupScheduling(id: number): Promise<GroupScheduling> {
  return (await apiClient.get<GroupScheduling>(`/admin/groups/${id}/scheduling`)).data
}
export async function saveGroupScheduling(id: number, settings: GroupScheduling): Promise<GroupScheduling> {
  return (await apiClient.put<GroupScheduling>(`/admin/groups/${id}/scheduling`, settings)).data
}
export async function clearSlowTTFT(id: number): Promise<void> {
  await apiClient.post(`/admin/accounts/${id}/slow-ttft/clear`)
}
