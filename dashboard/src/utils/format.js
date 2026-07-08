/**
 * Shared date / time formatting utilities.
 * Import these instead of copy-pasting across views.
 */

export function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString()
}

export function fmtStaleTime(iso) {
  if (!iso) return 'an unknown time ago'
  const secs = Math.floor((Date.now() - new Date(iso)) / 1000)
  if (secs < 60) return `${secs}s ago`
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`
  return `${Math.floor(secs / 3600)}h ago`
}
