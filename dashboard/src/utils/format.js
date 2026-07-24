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

// Parse "v1.2.3" / "1.2.3-rc1" into [1,2,3] (leading v and pre-release dropped).
function semverParts(v) {
  return String(v).replace(/^v/, '').split('-')[0].split('.').map(n => parseInt(n, 10) || 0)
}

// True only when `version` is strictly older than `baseline` (semver-aware).
// Returns false when either side is missing so drift never shows on unknown data.
// ponytail: 3-field semver only; extend semverParts if build metadata ever matters.
export function versionBehind(version, baseline) {
  if (!version || !baseline) return false
  const a = semverParts(version), b = semverParts(baseline)
  for (let i = 0; i < 3; i++) {
    if ((a[i] || 0) !== (b[i] || 0)) return (a[i] || 0) < (b[i] || 0)
  }
  return false
}
