/**
 * HeliosGen workbench launch API.
 *
 * The launch response intentionally contains only a short-lived URL. Long-lived
 * Sub2API API keys are exchanged by HeliosGen on the server and are never part
 * of this browser-facing contract.
 */

import { apiClient } from './client'

export interface HeliosLaunchResponse {
  launch_url: string
  expires_in: number
}

/** Request a one-time HeliosGen workbench launch URL for the current user. */
export async function launchHelios(): Promise<HeliosLaunchResponse> {
  const { data } = await apiClient.post<HeliosLaunchResponse>('/workbenches/helios/launch')
  return data
}
