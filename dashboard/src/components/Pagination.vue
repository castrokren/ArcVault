<template>
  <div v-if="pages > 1 || total > 0" class="pagination">
    <span class="record-count">
      Showing {{ rangeStart }}–{{ rangeEnd }} of {{ total }}
    </span>
    <div class="controls">
      <button class="page-btn" :disabled="page === 1" @click="$emit('page-change', page - 1)">‹</button>

      <template v-for="btn in pageButtons" :key="btn.key">
        <span v-if="btn.ellipsis" class="ellipsis">…</span>
        <button
          v-else
          class="page-btn"
          :class="{ active: btn.num === page }"
          @click="$emit('page-change', btn.num)"
        >{{ btn.num }}</button>
      </template>

      <button class="page-btn" :disabled="page === pages || pages === 0" @click="$emit('page-change', page + 1)">›</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  page:  { type: Number, required: true },
  pages: { type: Number, required: true },
  total: { type: Number, required: true },
  limit: { type: Number, required: true },
})

defineEmits(['page-change'])

const rangeStart = computed(() => props.total === 0 ? 0 : (props.page - 1) * props.limit + 1)
const rangeEnd   = computed(() => Math.min(props.page * props.limit, props.total))

// Builds a button list with at most 7 page buttons and ellipsis placeholders.
// e.g. [1] … [4] [5] [6] … [12]  when in the middle
const pageButtons = computed(() => {
  const n = props.pages
  if (n <= 7) {
    return Array.from({ length: n }, (_, i) => ({ key: i + 1, num: i + 1, ellipsis: false }))
  }

  const cur = props.page
  const result = []

  const push = (num) => result.push({ key: num, num, ellipsis: false })
  const pushEllipsis = (key) => result.push({ key: `e${key}`, ellipsis: true })

  if (cur <= 4) {
    // near start: 1 2 3 4 5 … n
    for (let i = 1; i <= 5; i++) push(i)
    pushEllipsis('end')
    push(n)
  } else if (cur >= n - 3) {
    // near end: 1 … n-4 n-3 n-2 n-1 n
    push(1)
    pushEllipsis('start')
    for (let i = n - 4; i <= n; i++) push(i)
  } else {
    // middle: 1 … cur-1 cur cur+1 … n
    push(1)
    pushEllipsis('start')
    push(cur - 1)
    push(cur)
    push(cur + 1)
    pushEllipsis('end')
    push(n)
  }

  return result
})
</script>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 1.25rem;
  font-family: var(--font-body);
  font-size: 0.82rem;
  color: var(--text-muted);
}

.record-count {
  color: var(--text-muted);
}

.controls {
  display: flex;
  align-items: center;
  gap: 0.2rem;
}

.page-btn {
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.4rem;
  border: 1px solid var(--border-default);
  border-radius: 5px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-family: var(--font-body);
  font-size: 0.82rem;
  transition: all 0.15s;
}

.page-btn:hover:not(:disabled) {
  border-color: var(--accent-border);
  color: var(--accent);
}

.page-btn.active {
  background: var(--accent-dim);
  border-color: var(--accent-border);
  color: var(--accent);
  font-weight: 600;
}

.page-btn:disabled {
  opacity: 0.3;
  cursor: default;
}

.ellipsis {
  padding: 0 0.25rem;
  color: var(--text-muted);
}
</style>
