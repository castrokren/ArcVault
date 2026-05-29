<template>
  <div class="progress-container">
    <div class="progress-bar" :style="{ width: percentage + '%' }"></div>
  </div>
</template>

<script>
export default {
  name: 'ProgressBar',
  props: {
    percentage: {
      type: Number,
      required: true,
      validator: (v) => v >= 0 && v <= 100
    }
  }
}
</script>

<style scoped>
.progress-container {
  height: 4px;
  background: #e0e0e0;
  border-radius: 2px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: #4CAF50;
  transition: width 0.2s ease-out;
}
</style>
