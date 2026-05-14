import { ref, onUnmounted } from 'vue'

const WS_URL = 'ws://localhost:443/ws'

export function useWebSocket() {
  const connected = ref(false)
  const lastEvent = ref(null)
  let ws = null
  let reconnectTimer = null

  function getToken() {
    return localStorage.getItem('arcvault_token') || ''
  }

  function connect() {
    const token = getToken()
    if (!token) return

    // gorilla/websocket doesn't support WS subprotocol auth,
    // so we pass the token as a query param
    ws = new WebSocket(`${WS_URL}?token=${encodeURIComponent(token)}`)

    ws.onopen = () => {
      connected.value = true
      console.log('WS connected')
    }

    ws.onmessage = (e) => {
      try {
        lastEvent.value = JSON.parse(e.data)
      } catch {
        console.warn('WS: bad message', e.data)
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
