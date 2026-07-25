import { ref, onMounted, onUnmounted } from 'vue'
import { getFederationHealth } from '../api'

const POLL_INTERVAL = 15_000

/**
 * useFederationLag — polls GET /api/federation/health every 15s.
 *
 * Returns:
 *   isStale   {Ref<boolean>}  true if any peer has lag_events > 0
 *   lagEvents {Ref<number>}   max lag_events across all peers (0 = fully synced)
 *
 * Usage (Agents.vue, Jobs.vue, History.vue):
 *   import { useFederationLag } from '../composables/useFederationLag.js'
 *   const { isStale } = useFederationLag()
 *   // v-if="isStale" on the stale banner — no other changes needed
 */
export function useFederationLag() {
  const isStale = ref(false)
  const lagEvents = ref(0)
  let timer = null

  async function poll() {
    try {
      const peers = await getFederationHealth()
      if (!Array.isArray(peers) || peers.length === 0) {
        // No federation peers — never stale
        isStale.value = false
        lagEvents.value = 0
        return
      }
      const maxLag = peers.reduce((max, p) => Math.max(max, p.lag_events ?? 0), 0)
      lagEvents.value = maxLag
      isStale.value = maxLag > 0
    } catch {
      // Network error or endpoint unavailable — preserve current state,
      // do not flip the banner based on a failed fetch.
    }
  }

  onMounted(() => {
    poll()
    timer = setInterval(poll, POLL_INTERVAL)
  })

  onUnmounted(() => {
    clearInterval(timer)
  })

  return { isStale, lagEvents }
}
