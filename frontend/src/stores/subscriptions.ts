/**
 * Subscription Store
 * Global state management for user subscriptions with caching and deduplication
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import subscriptionsAPI, { type SubscriptionAvailability } from '@/api/subscriptions'
import type { UserSubscription } from '@/types'

// Cache TTL: 60 seconds
const CACHE_TTL_MS = 60_000

// Request generation counter to invalidate stale in-flight responses
let requestGeneration = 0

export const useSubscriptionStore = defineStore('subscriptions', () => {
  const availability = ref<SubscriptionAvailability | null>(null)
  const availabilityError = ref(false)
  let availabilityAt = 0
  let availabilityGeneration = 0
  let availabilityPromise: Promise<SubscriptionAvailability> | null = null
  async function fetchAvailability(force = false): Promise<SubscriptionAvailability> {
    if (!force && availability.value && Date.now() - availabilityAt < CACHE_TTL_MS) return availability.value
    if (availabilityPromise) return availabilityPromise
    const generation = ++availabilityGeneration
    const pending = subscriptionsAPI.getAvailability().then(data => {
      if (generation === availabilityGeneration) { availability.value = data; availabilityAt = Date.now(); availabilityError.value = false }
      return data
    }).catch(cause => { if (generation === availabilityGeneration) availabilityError.value = true; throw cause }).finally(() => { if (availabilityPromise === pending) availabilityPromise = null })
    availabilityPromise = pending
    return pending
  }
  // State
  const activeSubscriptions = ref<UserSubscription[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const lastFetchedAt = ref<number | null>(null)

  // In-flight request deduplication
  let activePromise: Promise<UserSubscription[]> | null = null

  // Auto-refresh interval
  let pollerInterval: ReturnType<typeof setInterval> | null = null

  // Computed
  const hasActiveSubscriptions = computed(() => activeSubscriptions.value.length > 0)

  /**
   * Fetch active subscriptions with caching and deduplication
   * @param force - Force refresh even if cache is valid
   */
  async function fetchActiveSubscriptions(force = false): Promise<UserSubscription[]> {
    const now = Date.now()

    // Return cached data if valid
    if (
      !force &&
      loaded.value &&
      lastFetchedAt.value &&
      now - lastFetchedAt.value < CACHE_TTL_MS
    ) {
      return activeSubscriptions.value
    }

    // Return in-flight request if exists (deduplication)
    if (activePromise && !force) {
      return activePromise
    }

    const currentGeneration = ++requestGeneration

    // Start new request
    loading.value = true
    const requestPromise = subscriptionsAPI
      .getActiveSubscriptions()
      .then((data) => {
        if (currentGeneration === requestGeneration) {
          activeSubscriptions.value = data
          loaded.value = true
          lastFetchedAt.value = Date.now()
        }
        return data
      })
      .catch((error) => {
        console.error('Failed to fetch active subscriptions:', error)
        throw error
      })
      .finally(() => {
        if (activePromise === requestPromise) {
          loading.value = false
          activePromise = null
        }
      })

    activePromise = requestPromise

    return activePromise
  }

  /**
   * Start auto-refresh polling 
   */
  function startPolling() {
    if (pollerInterval) return

    pollerInterval = setInterval(() => {
      fetchActiveSubscriptions(true).catch((error) => {
        console.error('Subscription polling failed:', error)
      })
    }, 5 * 60 * 1000)
  }

  /**
   * Stop auto-refresh polling
   */
  function stopPolling() {
    if (pollerInterval) {
      clearInterval(pollerInterval)
      pollerInterval = null
    }
  }

  /**
   * Clear all subscription data and stop polling
   */
  function clear() {
    availabilityGeneration++
    availability.value = null
    availabilityError.value = false
    availabilityAt = 0
    availabilityPromise = null
    requestGeneration++
    activePromise = null
    loading.value = false
    activeSubscriptions.value = []
    loaded.value = false
    lastFetchedAt.value = null
    stopPolling()
  }

  /**
   * Invalidate cache (force next fetch to reload)
   */
  function invalidateCache() {
    availabilityAt = 0
    availabilityGeneration++
    availabilityPromise = null
    lastFetchedAt.value = null
    void fetchAvailability(true).catch(() => undefined)
  }

  return {
    activeSubscriptions,
    availability,
    availabilityError,
    fetchAvailability,
    loading,
    hasActiveSubscriptions,

    // Actions
    fetchActiveSubscriptions,
    startPolling,
    stopPolling,
    clear,
    invalidateCache
  }
})
