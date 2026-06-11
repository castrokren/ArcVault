<template>
  <div class="sync-flags-builder">
    <!-- Collapsible Header -->
    <div class="advanced-header" @click="expanded = !expanded">
      <span class="header-text">Advanced Options</span>
      <span class="header-icon" :class="{ expanded }">▼</span>
    </div>

    <!-- Expanded Content -->
    <div v-if="expanded" class="advanced-content">
      <!-- Filtering Section -->
      <fieldset class="section filtering-section">
        <legend>Filtering</legend>

        <!-- Max Age Input -->
        <div class="form-group">
          <label for="max-age">Max Age (days)</label>
          <input
            id="max-age"
            v-model.number="flags.max_age"
            type="number"
            min="0"
            placeholder="Leave blank for no limit. Days, e.g., 30"
            @input="updateFlags"
          />
          <p class="helper-text">Sync only files modified within N days</p>
        </div>

        <!-- Min Age Input -->
        <div class="form-group">
          <label for="min-age">Min Age (days)</label>
          <input
            id="min-age"
            v-model.number="flags.min_age"
            type="number"
            min="0"
            placeholder="e.g., 1"
            @input="updateFlags"
          />
          <p class="helper-text">Sync only files not modified within N days</p>
          <p v-if="validationErrors.minMaxAge" class="error-text">
            {{ validationErrors.minMaxAge }}
          </p>
        </div>

        <!-- Max Size Input -->
        <div class="form-group">
          <label for="max-size">Max Size (MB)</label>
          <input
            id="max-size"
            v-model.number="flags.max_size"
            type="number"
            min="0"
            placeholder="MB, e.g., 2048"
            @input="updateFlags"
          />
          <p class="helper-text">Sync only files smaller than N MB</p>
        </div>
      </fieldset>

      <!-- Behavior Section -->
      <fieldset class="section behavior-section">
        <legend>Behavior</legend>

        <div class="form-group checkbox-group">
          <label for="mirror">
            <input
              id="mirror"
              v-model="flags.mirror"
              type="checkbox"
              @input="updateFlags"
            />
            <span>Mirror mode</span>
          </label>
          <p class="helper-text">Delete destination files not in source</p>
        </div>
      </fieldset>

      <!-- Exclusions Section -->
      <fieldset class="section exclusions-section">
        <legend>Exclusions</legend>

        <!-- Exclude Files Textarea -->
        <div class="form-group">
          <label for="exclude-files">Exclude Files</label>
          <textarea
            id="exclude-files"
            v-model="excludeFilesText"
            placeholder="One pattern per line&#10;*.tmp&#10;*.log"
            rows="3"
            @input="updateExcludeFiles"
          ></textarea>
          <p class="helper-text">
            File patterns to exclude (wildcards supported: *, ?, [...]).
            Example: *.tmp, *.log, Config.ini
          </p>
        </div>

        <!-- Exclude Directories Textarea -->
        <div class="form-group">
          <label for="exclude-dirs">Exclude Directories</label>
          <textarea
            id="exclude-dirs"
            v-model="excludeDirsText"
            placeholder="One pattern per line&#10;.git&#10;node_modules"
            rows="3"
            @input="updateExcludeDirs"
          ></textarea>
          <p class="helper-text">
            Directory patterns to exclude (one per line).
            Example: .git, node_modules, $Recycle.Bin
          </p>
        </div>
      </fieldset>

      <!-- Command Preview Section -->
      <div class="command-preview-section">
        <h4>Command Preview</h4>
        <div class="command-preview">
          <div class="command-block">
            <p class="command-label">Robocopy:</p>
            <code class="command-text">{{ robocopyCommand }}</code>
          </div>
          <div class="command-block">
            <p class="command-label">Rsync:</p>
            <code class="command-text">{{ rsyncCommand }}</code>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

// Props & Emits
const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({
      mirror: false,
      max_age: null,
      min_age: null,
      max_size: null,
      exclude_files: [],
      exclude_dirs: []
    })
  }
})

const emit = defineEmits(['update:modelValue'])

// State
const expanded = ref(false)
const flags = ref({ ...props.modelValue })
const validationErrors = ref({})
const excludeFilesText = ref((flags.value.exclude_files || []).join('\n'))
const excludeDirsText = ref((flags.value.exclude_dirs || []).join('\n'))

// Watch for external v-model changes
watch(
  () => props.modelValue,
  (newVal) => {
    flags.value = { ...newVal }
    excludeFilesText.value = newVal.exclude_files.join('\n')
    excludeDirsText.value = newVal.exclude_dirs.join('\n')
  },
  { deep: true }
)

// Validation
const validateMinMaxAge = () => {
  validationErrors.value.minMaxAge = ''
  if (
    flags.value.min_age > 0 &&
    flags.value.max_age > 0 &&
    flags.value.min_age > flags.value.max_age
  ) {
    validationErrors.value.minMaxAge = 'Min Age must be ≤ Max Age'
  }
}

// Parse exclude patterns from textarea
const parsePatterns = (text) => {
  if (!text.trim()) return []
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
}

// Update handlers
const updateFlags = () => {
  validateMinMaxAge()
  emit('update:modelValue', { ...flags.value })
}

const updateExcludeFiles = () => {
  flags.value.exclude_files = parsePatterns(excludeFilesText.value)
  emit('update:modelValue', { ...flags.value })
}

const updateExcludeDirs = () => {
  flags.value.exclude_dirs = parsePatterns(excludeDirsText.value)
  emit('update:modelValue', { ...flags.value })
}

// Command generation (mirrors sync_flags.go logic)
const robocopyCommand = computed(() => {
  let cmd = 'robocopy C:\\source D:\\destination'

  if (flags.value.mirror) {
    cmd += ' /MIR'
  }
  if (flags.value.max_age && flags.value.max_age > 0) {
    cmd += ` /MAXAGE:${flags.value.max_age}`
  }
  if (flags.value.min_age && flags.value.min_age > 0) {
    cmd += ` /MINAGE:${flags.value.min_age}`
  }
  if (flags.value.max_size && flags.value.max_size > 0) {
    cmd += ` /MAXSIZE:${flags.value.max_size}M`
  }
  if (flags.value.exclude_files && Array.isArray(flags.value.exclude_files) && flags.value.exclude_files.length > 0) {
    cmd += ` /XF ${flags.value.exclude_files.join(' ')}`
  }
  if (flags.value.exclude_dirs && Array.isArray(flags.value.exclude_dirs) && flags.value.exclude_dirs.length > 0) {
    cmd += ` /XD ${flags.value.exclude_dirs.join(' ')}`
  }

  return cmd
})

const rsyncCommand = computed(() => {
  let cmd = 'rsync -a'

  if (flags.value.mirror) {
    cmd += ' --delete'
  }
  if (flags.value.max_age && flags.value.max_age > 0) {
    const seconds = flags.value.max_age * 86400
    cmd += ` --max-age=${seconds}`
  }
  if (flags.value.min_age && flags.value.min_age > 0) {
    const seconds = flags.value.min_age * 86400
    cmd += ` --min-age=${seconds}`
  }
  if (flags.value.max_size && flags.value.max_size > 0) {
    const bytes = flags.value.max_size * 1048576
    cmd += ` --maxsize=${bytes}`
  }
  if (flags.value.exclude_files && Array.isArray(flags.value.exclude_files) && flags.value.exclude_files.length > 0) {
    flags.value.exclude_files.forEach((pattern) => {
      cmd += ` --exclude='${pattern}'`
    })
  }
  if (flags.value.exclude_dirs && Array.isArray(flags.value.exclude_dirs) && flags.value.exclude_dirs.length > 0) {
    flags.value.exclude_dirs.forEach((pattern) => {
      cmd += ` --exclude='${pattern}/'`
    })
  }

  cmd += ' /source /destination/'
  return cmd
})
</script>

<style scoped>
.sync-flags-builder {
  margin: 1rem 0;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  background: var(--card-bg, #fff);
}

.advanced-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  cursor: pointer;
  background: var(--header-bg, #f5f5f5);
  border-bottom: 1px solid var(--border-color, #ddd);
  user-select: none;
  font-weight: 500;
}

.advanced-header:hover {
  background: var(--header-hover-bg, #e9e9e9);
}

.header-icon {
  display: inline-block;
  transition: transform 0.2s;
  font-size: 0.9em;
}

.header-icon.expanded {
  transform: rotate(180deg);
}

.advanced-content {
  padding: 1.5rem;
}

.section {
  margin-bottom: 2rem;
  border: none;
  padding: 0;
}

.section legend {
  font-weight: 600;
  margin-bottom: 1rem;
  padding: 0;
  font-size: 0.95em;
  color: var(--text-primary, #333);
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  font-size: 0.95em;
  color: var(--text-primary, #333);
}

.form-group input[type='number'],
.form-group textarea {
  width: 100%;
  padding: 0.6rem;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 3px;
  font-family: inherit;
  font-size: 0.95em;
  background: var(--bg-input);
  color: var(--text-primary);
}

.form-group input[type='number']:focus,
.form-group textarea:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: var(--glow-accent);
}

.form-group textarea {
  resize: vertical;
}

.helper-text {
  margin-top: 0.4rem;
  font-size: 0.85em;
  color: var(--text-secondary, #666);
  line-height: 1.4;
}

.error-text {
  margin-top: 0.4rem;
  font-size: 0.85em;
  color: var(--error-color, #d9534f);
}

.checkbox-group {
  display: flex;
  align-items: flex-start;
}

.checkbox-group label {
  display: flex;
  align-items: center;
  margin: 0;
  font-weight: normal;
}

.checkbox-group input[type='checkbox'] {
  margin-right: 0.6rem;
  margin-top: 0.2rem;
  cursor: pointer;
}

.command-preview-section {
  margin-top: 2rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-color, #ddd);
}

.command-preview-section h4 {
  margin: 0 0 1rem 0;
  font-size: 0.95em;
  color: var(--text-primary, #333);
}

.command-preview {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.command-block {
  background: var(--code-bg, #f5f5f5);
  border: 1px solid var(--border-color, #ddd);
  border-radius: 3px;
  padding: 0.8rem;
}

.command-label {
  margin: 0 0 0.5rem 0;
  font-size: 0.85em;
  font-weight: 600;
  color: var(--text-primary, #333);
}

.command-text {
  display: block;
  margin: 0;
  font-size: 0.8em;
  font-family: 'Monaco', 'Courier New', monospace;
  color: var(--code-color, #333);
  word-break: break-all;
  white-space: pre-wrap;
  line-height: 1.4;
}

@media (max-width: 768px) {
  .command-preview {
    grid-template-columns: 1fr;
  }
}
</style>
