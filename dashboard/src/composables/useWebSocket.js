import { ref, onUnmounted } from 'vue'

export function useWebSocket() {
  const connected = ref(false)
  const lastEvent = ref(null)
  let ws = null
  let reconnectTimer = null

  function getToken() {
    return localStorage.getItem('arcvault_jwt') || localStorage.getItem('arcvault_token') || ''
  }

  function getWsUrl() {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}/ws`
  }

  function connect() {
    const token = getToken()
    if (!token) return

    const wsUrl = getWsUrl()
    console.log(`[WS] Connecting to ${wsUrl} from origin ${window.location.origin}`)

    ws = new WebSocket(wsUrl, [`bearer.${token}`])

    ws.onopen = () => {
      connected.value = true
      console.log('[WS] Connected successfully from origin:', window.location.origin)
    }

    ws.onmessage = (e) => {
      console.log('WS message received:', e.data)
      try {
        lastEvent.value = JSON.parse(e.data)
        console.log('WS parsed event:', lastEvent.value)
      } catch (parseError) {
        console.warn('WS: bad message', e.data, parseError)
      }
    }

    ws.onclose = (event) => {
      connected.value = false
      console.log('[WS] Disconnected. Code:', event.code, 'Reason:', event.reason)

      // Handle origin rejection (code 1006 = abnormal closure)
      if (event.code === 1006 && event.reason) {
        console.error('[WS] Connection rejected - may be CORS/origin validation issue')
      }

      console.log('WS disconnected, reconnecting in 5s...')
      reconnectTimer = setTimeout(connect, 5000)
    }

    ws.onerror = (err) => {
      console.error('[WS] Error:', err)
      // Don't close here - onclose will handle reconnect
    }
  }

  function disconnect() {
    clearTimeout(reconnectTimer)
    if (ws) ws.close()
  }

  onUnmounted(disconnect)

  return { connected, lastEvent, connect, disconnect }
}
