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
  current: '',
  latest: '',
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
  gap: 0.85rem;
  padding: 0.6rem 1.5rem;
  background: var(--bg-warning);
  border-bottom: 1px solid rgba(245, 158, 11, 0.3);
  color: var(--color-warning);
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-warning);
  flex-shrink: 0;
  animation: upd-pulse 1.5s ease-in-out infinite;
}

@keyframes upd-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.message {
  flex: 1;
  font-family: var(--font-body);
  font-size: 0.85rem;
  color: var(--color-warning);
}

.btn-update {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.3rem 0.85rem;
  background: var(--color-warning);
  color: var(--bg-base);
  border: none;
  border-radius: 5px;
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.82rem;
  font-weight: 600;
  transition: filter 0.15s;
  flex-shrink: 0;
}
.btn-update:hover { filter: brightness(1.1); }

.btn-dismiss {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  background: transparent;
  color: var(--color-warning);
  border: none;
  cursor: pointer;
  font-size: 0.9rem;
  opacity: 0.6;
  border-radius: 4px;
  transition: opacity 0.15s, background 0.15s;
  flex-shrink: 0;
}
.btn-dismiss:hover {
  opacity: 1;
  background: rgba(245, 158, 11, 0.15);
}
</style>
