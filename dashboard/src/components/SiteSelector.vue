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
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  border: 1px solid #444;
  background: #111;
  color: #ccc;
  font-size: 0.85rem;
  cursor: pointer;
}

.site-selector select:focus {
  outline: none;
  border-color: #4f8ef7;
}

[data-theme="light"] .site-selector select {
  background: #fff;
  border-color: #ccc;
  color: #333;
}
</style>
