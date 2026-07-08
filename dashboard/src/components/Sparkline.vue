<template>
  <svg
    :width="width"
    :height="height"
    :viewBox="`0 0 ${width} ${height}`"
    class="sparkline"
    aria-hidden="true"
  >
    <polygon v-if="fill && norm.length > 1" :points="areaPoints" :fill="color" opacity="0.08" />
    <polyline
      v-if="norm.length > 1"
      :points="polyPoints"
      fill="none"
      :stroke="color"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>

<script>
export default {
  name: 'Sparkline',
  props: {
    points: { type: Array, required: true },
    color: { type: String, default: 'var(--accent)' },
    width: { type: Number, default: 90 },
    height: { type: Number, default: 28 },
    fill: { type: Boolean, default: true }
  },
  computed: {
    norm() {
      const pts = (this.points || []).map(Number)
      if (pts.length === 0) return []
      const max = Math.max(...pts)
      const min = Math.min(...pts)
      const range = max - min || 1
      const stepX = this.width / Math.max(pts.length - 1, 1)
      const pad = 2
      return pts.map((v, i) => [
        i * stepX,
        pad + (this.height - 2 * pad) * (1 - (v - min) / range)
      ])
    },
    polyPoints() {
      return this.norm.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ')
    },
    areaPoints() {
      return `0,${this.height} ${this.polyPoints} ${this.width},${this.height}`
    }
  }
}
</script>

<style scoped>
.sparkline {
  display: block;
}
</style>
