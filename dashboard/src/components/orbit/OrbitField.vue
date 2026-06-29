<template>
  <!--
    Stub — Marcus will replace with the full canvas engine (Plan B).
    Must expose `warp(): Promise<void>` via defineExpose.
    Prop `motion="bold"` is locked for this scene.
  -->
  <canvas
    ref="canvasEl"
    class="orbit-canvas"
    aria-hidden="true"
  ></canvas>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  motion: {
    type: String,
    default: 'bold',
  },
})

const canvasEl = ref(null)

onMounted(() => {
  // ponytail: stub renders static frame. Full engine in Plan B.
  const canvas = canvasEl.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  const dpr = window.devicePixelRatio || 1
  canvas.width = window.innerWidth * dpr
  canvas.height = window.innerHeight * dpr
  canvas.style.width = window.innerWidth + 'px'
  canvas.style.height = window.innerHeight + 'px'
  ctx.scale(dpr, dpr)
  ctx.fillStyle = '#07090e'
  ctx.fillRect(0, 0, canvas.width, canvas.height)
})

function warp() {
  return Promise.resolve()
}

defineExpose({ warp })
</script>

<style scoped>
.orbit-canvas {
  position: fixed;
  inset: 0;
  z-index: 0;
  display: block;
  pointer-events: none;
}
</style>
