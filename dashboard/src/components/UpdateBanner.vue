<template>
  <div v-if="updateStore.available && !dismissed" class="update-banner">
    <span class="dot">●</span>
    <span class="message">
      ArcVault {{ updateStore.latest }} is available — you're on {{ updateStore.current }}
    </span>
    <button class="btn-update" @click="openModal">Update now</button>
    <button class="btn-dismiss" @click="dismissed = true" aria-label="Dismiss">✕</button>
  </div>
</template>

<script setup>
import { ref, inject } from 'vue'

const props = defineProps({
  onUpdate: Function
})

const dismissed = ref(false)

// Get the update store from context
const updateStore = inject('updateStore', {
  current: 'v0.2.0',
  latest: 'v0.2.0',
  available: false,
  releaseUrl: ''
})

function openModal() {
  if (props.onUpdate) {
    props.onUpdate()
  }
}
</script>

<style scoped>
.update-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1.5rem;
  background: linear-gradient(135deg, #2c3e50, #34495e);
  border-bottom: 2px solid #f39c12;
  color: #fff;
}

.dot {
  color: #f39c12;
  font-size: 1.2rem;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.message {
  flex: 1;
  font-size: 0.95rem;
}

.btn-update {
  padding: 0.4rem 1rem;
  background: #f39c12;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  font-weight: 500;
  transition: background 0.2s;
}

.btn-update:hover {
  background: #e67e22;
}

.btn-dismiss {
  padding: 0.4rem 0.6rem;
  background: transparent;
  color: #aaa;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  transition: color 0.2s;
}

.btn-dismiss:hover {
  color: #fff;
}
</style>
