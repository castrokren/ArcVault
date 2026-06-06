/**
 * Shared cron-expression utilities.
 *
 * parseCronParts  – splits and validates a 5-field cron expression
 * fmtTime         – formats numeric hour/minute as 12-hour string
 * describeCron    – returns a human-readable description of a cron expression
 *
 * api.ts re-exports describeCron as cronPreview.
 * ScheduleBuilder.vue uses parseCronParts to avoid duplicating the split/validate logic.
 */

/**
 * Split a cron expression into its five fields.
 * Returns { min, hour, dom, month, dow } or null if the expression is blank/invalid.
 */
export function parseCronParts(expr) {
  if (!expr || !expr.trim()) return null
  const parts = expr.trim().split(/\s+/)
  if (parts.length !== 5) return null
  const [min, hour, dom, month, dow] = parts
  return { min, hour, dom, month, dow }
}

/**
 * Format numeric hour + minute as a 12-hour AM/PM string.
 * @param {number} h  – hour   (0–23)
 * @param {number} m  – minute (0–59)
 */
export function fmtTime(h, m) {
  const suffix = h >= 12 ? 'PM' : 'AM'
  const h12 = h === 0 ? 12 : h > 12 ? h - 12 : h
  const mStr = m === 0 ? '' : `:${String(m).padStart(2, '0')}`
  return `${h12}${mStr} ${suffix}`
}

const DAYS   = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June',
                'July', 'August', 'September', 'October', 'November', 'December']

function isFixed(field) {
  return !field.includes('*') && !field.includes('/')
}

/**
 * Return a human-readable description of a cron expression,
 * or the original expression if it doesn't match a known pattern.
 */
export function describeCron(expr) {
  if (!expr || !expr.trim()) return ''

  const parts = parseCronParts(expr)
  if (!parts) return expr

  const { min, hour, dom, month, dow } = parts

  // Interval: */N * * * *
  if (min.startsWith('*/') && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    const n = min.slice(2)
    return `Every ${n} minute${n === '1' ? '' : 's'}`
  }

  // Hourly: M * * * *
  if (isFixed(min) && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    return `Every hour at minute ${min}`
  }

  const h = parseInt(hour, 10)
  const m = parseInt(min,  10)

  // Daily: M H * * *
  if (isFixed(min) && isFixed(hour) && dom === '*' && month === '*' && dow === '*') {
    return `Every day at ${fmtTime(h, m)}`
  }

  // Weekly – single day: M H * * D
  if (isFixed(min) && isFixed(hour) && dom === '*' && month === '*' &&
      isFixed(dow) && !dow.includes(',')) {
    const d = parseInt(dow, 10)
    const dayName = d >= 0 && d <= 6 ? DAYS[d] : `day ${dow}`
    return `Every ${dayName} at ${fmtTime(h, m)}`
  }

  // Weekdays: M H * * 1-5
  if (isFixed(min) && isFixed(hour) && dom === '*' && month === '*' && dow === '1-5') {
    return `Weekdays at ${fmtTime(h, m)}`
  }

  // Weekends: M H * * 0,6
  if (isFixed(min) && isFixed(hour) && dom === '*' && month === '*' &&
      (dow === '0,6' || dow === '6,0')) {
    return `Weekends at ${fmtTime(h, m)}`
  }

  // Monthly: M H D * *
  if (isFixed(min) && isFixed(hour) && isFixed(dom) && month === '*' && dow === '*') {
    return `Monthly on day ${dom} at ${fmtTime(h, m)}`
  }

  // Yearly: M H D Mo *
  if (isFixed(min) && isFixed(hour) && isFixed(dom) && isFixed(month) && dow === '*') {
    const mo = parseInt(month, 10)
    const monthName = mo >= 1 && mo <= 12 ? MONTHS[mo - 1] : `month ${month}`
    return `Yearly on ${monthName} ${dom} at ${fmtTime(h, m)}`
  }

  return expr
}
