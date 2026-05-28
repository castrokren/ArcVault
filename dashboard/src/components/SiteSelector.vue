<template>
  <div class="site-selector" v-if="subs.length > 0">
    <select :value="modelValue" @change="onChange">
      <option :value="null">All Sites</option>
      <option v-for="sub in subs" :key="sub.id" :value="sub.id">
        {{ sub.name }}{{ sub.status === 'offline' ? ' (offline)' : '' }}
      </option>
    </select>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { listFederation } from '../api.js'

defineProps(['modelValue'])
const emit = defineEmits(['update:modelValue'])

const subs = ref([])

async function load() {
  try {
    const data = await listFederation()
    subs.value = data || []
  } catch {
    // federation not configured or unreachable — hide selector
    subs.value = []
  }
}

function onChange(e) {
  const val = e.target.value || null
  emit('update:modelValue', val)
}

let timer
onMounted(() => {
  load()
  timer = setInterval(load, 30000)
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.site-selector select {
  padding: 0.3rem 0.6rem;
  border-radius: 5px;
  border: 1px solid var(--border-default);
  background: var(--bg-input);
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 0.82rem;
  cursor: pointer;
  transition: border-color 0.15s;
}

.site-selector select:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
}

.site-selector select:focus {
  outline: none;
  border-color: var(--accent-border);
  box-shadow: 0 0 0 3px var(--accent-dim);
}
</style>
