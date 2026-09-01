import { launchHelios } from '@/api/helios'
import type { HeliosLaunchResponse } from '@/api/helios'

export const HELIOS_WORKBENCH_ORIGIN = 'https://canvas.sub.sunmmyapi.xyz'
export const HELIOS_WORKBENCH_BOOTSTRAP_PATH = '/bootstrap'

export type HeliosLaunchFailure = 'popup-blocked' | 'launch-failed'
export type HeliosLaunchMode = 'popup' | 'current'

export interface HeliosLaunchOptions {
  /** Open an isolated tab, or replace the current tab. */
  mode?: HeliosLaunchMode
  /** Injectable for focused tests; production uses the API wrapper above. */
  request?: () => Promise<HeliosLaunchResponse>
  /** Injectable browser window; production uses the current window. */
  window?: Pick<Window, 'open' | 'location'>
  /** Called without exposing the launch URL or any credential material. */
  notify?: (failure: HeliosLaunchFailure) => void
}

/**
 * Validate the only cross-origin destination the frontend is allowed to open.
 * Credentials are rejected even though URL.origin intentionally excludes them.
 */
export function parseHeliosLaunchUrl(value: unknown): URL | null {
  if (typeof value !== 'string') return null

  try {
    const url = new URL(value)
    if (
      url.origin !== HELIOS_WORKBENCH_ORIGIN ||
      url.pathname !== HELIOS_WORKBENCH_BOOTSTRAP_PATH ||
      url.username !== '' ||
      url.password !== ''
    ) {
      return null
    }
    return url
  } catch {
    return null
  }
}

/**
 * Start the HeliosGen handoff from either the sidebar popup or /helios.
 *
 * In popup mode every browser operation before the API await is deliberately
 * synchronous: a user gesture opens about:blank and severs opener access before
 * the one-time grant is requested. A failed request or URL validation closes
 * that blank tab and only reports a generic localized error to the caller.
 */
export async function launchHeliosWorkbench(options: HeliosLaunchOptions = {}): Promise<boolean> {
  const mode = options.mode ?? 'popup'
  const browserWindow = options.window ?? window
  const request = options.request ?? launchHelios
  const notify = options.notify ?? (() => undefined)
  let child: Window | null = null

  if (mode === 'popup') {
    try {
      child = browserWindow.open('about:blank', '_blank')
    } catch {
      child = null
    }

    if (!child) {
      notify('popup-blocked')
      return false
    }

    try {
      // This must happen before awaiting the launch request.
      child.opener = null
    } catch {
      try {
        child.close()
      } catch {
        // Ignore cleanup failures; the generic launch failure remains safe.
      }
      notify('launch-failed')
      return false
    }
  }

  try {
    const response = await request()
    const launchUrl = parseHeliosLaunchUrl(response?.launch_url)
    if (!launchUrl) throw new Error('Invalid Helios launch URL')

    const target = child ?? browserWindow
    target.location.replace(launchUrl.href)
    return true
  } catch {
    if (child) {
      try {
        child.close()
      } catch {
        // Ignore cleanup failures; no sensitive value is exposed to the user.
      }
    }
    notify('launch-failed')
    return false
  }
}
