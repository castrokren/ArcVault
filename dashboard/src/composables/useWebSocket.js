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

    // Pass the token as a query param (gorilla/websocket doesn't support subprotocol auth)
    ws = new WebSocket(`${getWsUrl()}?token=${encodeURIComponent(token)}`)

    ws.onopen = () => {
      connected.value = true
      console.log('WS connected')
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

    ws.onclose = () => {
      connected.value = false
      console.log('WS disconnected, reconnecting in 5s...')
      reconnectTimer = setTimeout(connect, 5000)
    }

    ws.onerror = (err) => {
      console.error('WS error', err)
      ws.close()
    }
  }

  function disconnect() {
    clearTimeout(reconnectTimer)
    if (ws) ws.close()
  }

  onUnmounted(disconnect)

  return { connected, lastEvent, connect, disconnect }
}
