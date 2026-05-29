# Phase 19 Frontend Implementation Plan

**Date:** May 28, 2026  
**Phase:** 19 (Robocopy/Rsync Advanced Flags — Frontend Integration)  
**Design Spec:** `docs/specs/2026-05-28-phase19-frontend-design.md`  
**Status:** Ready for Execution

---

## Overview

Implement the SyncFlagsBuilder Vue component to allow users to configure advanced backup sync options in the ArcVault dashboard Jobs creation form.

**Deliverables:**
1. ✓ SyncFlagsBuilder.vue component (reusable, self-contained)
2. ✓ Command preview logic (robocopy + rsync)
3. ✓ Integration into Jobs.vue form
4. ✓ API payload building (include/omit sync_flags)
5. ✓ End-to-end testing (create job with sync_flags)

**Success Criteria:**
- ✓ Users can set all 6 sync flag options (mirror, max_age, min_age, max_size, exclude_files, exclude_dirs)
- ✓ Command preview shows accurate commands for both robocopy and rsync
- ✓ Min/Max age validation prevents invalid configurations
- ✓ Jobs created with sync_flags are persisted and retrieved correctly
- ✓ Form remains clean and uncluttered (collapsible Advanced section)

---

## Task Breakdown

### Task 1: Create SyncFlagsBuilder.vue Component Structure

**Objective:** Build the Vue component with data model, state management, and template structure.

**Steps:**

1. Create file: `dashboard/src/components/SyncFlagsBuilder.vue`

2. **Script section (setup):**
   - Import: `ref`, `computed` from Vue
   - Define props: none (uses v-model)
   - Define data model:
     ```javascript
     const flags = ref({
       mirror: false,
       max_age: null,
       min_age: null,
       max_size: null,
       exclude_files: [],
       exclude_dirs: []
     })
     ```
   - Define state:
     - `expanded: ref(false)` — collapsible state
     - `errors: ref({})` — validation errors

3. **Methods:**
   - `emitValue()` — parse excludes, validate, emit to parent via v-model
   - `parseExcludePattern(text)` — split by newline, trim, filter empty
   - `validateMinMaxAge()` — check min_age ≤ max_age
   - `generateRobocopyPreview()` — build command string
   - `generateRsyncPreview()` — build command string
   - `toggleExpanded()` — toggle collapsed/expanded state

4. **Template structure (outline):**
   ```vue
   <div class="sync-flags-builder">
     <button @click="toggleExpanded">Advanced Options ▼/▲</button>
     <div v-if="expanded" class="advanced-section">
       <div class="filtering-section">...</div>
       <div class="behavior-section">...</div>
       <div class="exclusions-section">...</div>
       <div class="command-preview">...</div>
     </div>
   </div>
   ```

5. **Verification:**
   - File exists at correct path ✓
   - Component exports correctly ✓
   - No syntax errors ✓
   - Data structure matches spec ✓

---

### Task 2: Build Form Fields and Validation

**Objective:** Implement all input fields with proper types, hints, and real-time validation.

**Steps:**

1. **Filtering Section:**
   ```vue
   <div class="filtering-section">
     <h4>Filtering</h4>
     <div class="field">
       <label>Max Age (days)</label>
       <input type="number" min="0" v-model.number="flags.max_age" 
              placeholder="Leave blank for no limit. Days, e.g., 30"
              @input="emitValue">
       <small>Sync only files modified within N days</small>
     </div>
     <div class="field">
       <label>Min Age (days)</label>
       <input type="number" min="0" v-model.number="flags.min_age"
              placeholder="e.g., 1"
              @input="emitValue">
       <small>Sync only files not modified within N days</small>
     </div>
     <div class="field">
       <label>Max Size (MB)</label>
       <input type="number" min="0" v-model.number="flags.max_size"
              placeholder="MB, e.g., 2048"
              @input="emitValue">
       <small>Sync only files smaller than N MB</small>
     </div>
   </div>
   ```

2. **Behavior Section:**
   ```vue
   <div class="behavior-section">
     <h4>Behavior</h4>
     <label>
       <input type="checkbox" v-model="flags.mirror" @change="emitValue">
       Mirror mode
     </label>
     <small>Delete destination files not in source</small>
   </div>
   ```

3. **Exclusions Section:**
   ```vue
   <div class="exclusions-section">
     <h4>Exclusions</h4>
     <div class="field">
       <label>Exclude Files</label>
       <textarea v-model="exclude_files_text" @input="emitValue"
                 placeholder="One pattern per line&#10;*.tmp&#10;*.log"
                 rows="3"></textarea>
       <small>File patterns to exclude (wildcards supported: *, ?, [...]).</small>
     </div>
     <div class="field">
       <label>Exclude Directories</label>
       <textarea v-model="exclude_dirs_text" @input="emitValue"
                 placeholder="One pattern per line&#10;.git&#10;node_modules"
                 rows="3"></textarea>
       <small>Directory patterns to exclude (one per line).</small>
     </div>
   </div>
   ```

4. **Validation display:**
   ```vue
   <div v-if="errors.minMaxAge" class="error">
     {{ errors.minMaxAge }}
   </div>
   ```

5. **Verification:**
   - All 6 field types render correctly ✓
   - Placeholder text and helper text visible ✓
   - v-model binding works (type in field, see in data) ✓
   - Number inputs enforce min="0" ✓
   - Checkbox toggles boolean ✓
   - Textareas display with correct rows ✓

---

### Task 3: Implement Command Preview Logic

**Objective:** Build JavaScript versions of robocopy/rsync command generators to show live preview.

**Steps:**

1. **Create helper functions:**
   ```javascript
   // Match sync_flags.go ToRobocopyArgs() logic
   function generateRobocopyArgs(flags) {
     const args = [];
     if (flags.mirror) args.push('/MIR');
     if (flags.max_age > 0) args.push(`/MAXAGE:${flags.max_age}`);
     if (flags.min_age > 0) args.push(`/MINAGE:${flags.min_age}`);
     if (flags.max_size > 0) args.push(`/MAXSIZE:${flags.max_size}M`);
     if (flags.exclude_files.length > 0) {
       args.push('/XF', ...flags.exclude_files);
     }
     if (flags.exclude_dirs.length > 0) {
       args.push('/XD', ...flags.exclude_dirs);
     }
     return args;
   }

   // Match sync_flags.go ToRsyncArgs() logic
   function generateRsyncArgs(flags) {
     const args = [];
     if (flags.mirror) args.push('--delete');
     if (flags.max_age > 0) {
       args.push(`--max-age=${flags.max_age * 86400}`);
     }
     if (flags.min_age > 0) {
       args.push(`--min-age=${flags.min_age * 86400}`);
     }
     if (flags.max_size > 0) {
       args.push(`--maxsize=${flags.max_size * 1048576}`);
     }
     for (const pattern of flags.exclude_files) {
       args.push(`--exclude=${pattern}`);
     }
     for (const pattern of flags.exclude_dirs) {
       args.push(`--exclude=${pattern}`);
     }
     return args;
   }
   ```

2. **Create command formatter:**
   ```javascript
   function formatRobocopyCommand(args) {
     // robocopy C:\src D:\dest [args]
     return `robocopy C:\\src D:\\dest ${args.join(' ')}`;
   }

   function formatRsyncCommand(args) {
     // rsync -a [args] /src /dest/
     const allArgs = ['-a', ...args];
     return `rsync ${allArgs.join(' ')} /src /dest/`;
   }
   ```

3. **Add computed properties:**
   ```javascript
   const robocopyPreview = computed(() => {
     const args = generateRobocopyArgs(flags.value);
     return args.length > 0 ? formatRobocopyCommand(args) : 'robocopy C:\\src D:\\dest';
   });

   const rsyncPreview = computed(() => {
     const args = generateRsyncArgs(flags.value);
     return args.length > 0 ? formatRsyncCommand(args) : 'rsync -a /src /dest/';
   });
   ```

4. **Verification:**
   - Command preview updates when flags change ✓
   - Robocopy command includes correct flags ✓
   - Rsync command includes correct flags ✓
   - Unit conversions correct (days→seconds, MB→bytes) ✓
   - Empty flags show default command ✓
   - Test cases:
     - Mirror only: `/MIR` and `--delete` appear
     - Max age 30: `/MAXAGE:30` and `--max-age=2592000`
     - Max size 2048: `/MAXSIZE:2048M` and `--maxsize=2147483648`
     - Exclude patterns appear in correct format

---

### Task 4: Implement Validation Logic

**Objective:** Add real-time validation for min/max age constraint.

**Steps:**

1. **Add validation function:**
   ```javascript
   function validateMinMaxAge() {
     const errors = {};
     if (flags.value.max_age > 0 && flags.value.min_age > 0) {
       if (flags.value.min_age > flags.value.max_age) {
         errors.minMaxAge = 'Min Age must be ≤ Max Age';
       }
     }
     return errors;
   }
   ```

2. **Hook into emitValue():**
   ```javascript
   function emitValue() {
     // Parse exclude patterns
     flags.value.exclude_files = parseExcludePattern(exclude_files_text.value);
     flags.value.exclude_dirs = parseExcludePattern(exclude_dirs_text.value);
     
     // Validate
     errors.value = validateMinMaxAge();
     
     // Emit to parent if valid
     if (Object.keys(errors.value).length === 0) {
       emit('update:modelValue', flags.value);
     }
   }
   ```

3. **Add isValid computed:**
   ```javascript
   const isValid = computed(() => {
     return Object.keys(errors.value).length === 0;
   });
   ```

4. **Verification:**
   - Min age > Max age shows error ✓
   - Error clears when corrected ✓
   - Form submission blocked while error active ✓
   - Negative values prevented by input type ✓
   - Empty fields allowed (treated as null) ✓

---

### Task 5: Integrate into Jobs.vue

**Objective:** Add SyncFlagsBuilder to the Jobs creation form.

**Steps:**

1. **Import component** (near top of Jobs.vue script):
   ```javascript
   import SyncFlagsBuilder from '../components/SyncFlagsBuilder.vue'
   ```

2. **Add to form data** (in reactive form ref):
   ```javascript
   const form = ref({
     dispatchMode: 'agent',
     agent_id: '',
     group_id: '',
     name: '',
     source_path: '',
     dest_path: '',
     schedule: '',
     sync_flags: {}  // NEW
   })
   ```

3. **Add to template** (after ScheduleBuilder):
   ```vue
   <ScheduleBuilder v-model="form.schedule" />
   <SyncFlagsBuilder v-model="form.sync_flags" />
   <div class="form-actions">
     <button class="primary" @click="createJob">Create</button>
   </div>
   ```

4. **Update createJob() function:**
   ```javascript
   async function createJob() {
     formError.value = null

     // Existing validation...
     if (!form.value.name || !form.value.source_path || !form.value.dest_path) {
       formError.value = 'Please fill in all required fields'
       return
     }

     creating.value = true
     try {
       const payload = { ...form.value }
       delete payload.dispatchMode

       // Remove empty fields
       if (!payload.schedule) delete payload.schedule
       
       // NEW: Omit sync_flags if all empty
       if (!payload.sync_flags || 
           Object.values(payload.sync_flags).every(v => !v || v.length === 0)) {
         delete payload.sync_flags
       }

       // Remove unused dispatch field based on mode
       if (form.value.dispatchMode === 'agent') delete payload.group_id
       if (form.value.dispatchMode === 'group') delete payload.agent_id

       await apiCreateJob(payload)
       
       // Reset form
       form.value = { 
         dispatchMode: 'agent', 
         agent_id: '', 
         group_id: '', 
         name: '', 
         source_path: '', 
         dest_path: '', 
         schedule: '',
         sync_flags: {}  // Reset
       }
       showForm.value = false
       page.value = 1
       await load()
     } catch (e) {
       formError.value = e.message
     } finally {
       creating.value = false
     }
   }
   ```

5. **Verification:**
   - SyncFlagsBuilder renders in form ✓
   - Collapsible "Advanced Options" works ✓
   - Can set all sync flags ✓
   - Command preview visible and updates ✓
   - Job creation includes sync_flags in payload ✓
   - API call succeeds with sync_flags ✓
   - Job can be created without sync_flags (backward compat) ✓

---

### Task 6: End-to-End Testing

**Objective:** Create a job with sync flags and verify it persists in the database.

**Steps:**

1. **Rebuild and restart coordinator:**
   ```bash
   .\scripts\rebuild-and-restart.ps1
   ```
   - Verify coordinator starts successfully
   - Verify dashboard loads at http://localhost:8080

2. **Test Case 1: Create job WITH sync_flags**
   - Navigate to Jobs
   - Click "+ New Job"
   - Fill in:
     - Agent: (select any agent)
     - Name: "test-with-flags"
     - Source: "C:\test-src"
     - Dest: "D:\test-dst"
   - Click "Advanced Options" to expand
   - Set:
     - Mirror: ✓ checked
     - Max Age: 30
     - Exclude Files: `*.tmp` and `*.log`
   - Verify command preview shows:
     - Robocopy: `/MIR /MAXAGE:30 /XF *.tmp *.log`
     - Rsync: `--delete --max-age=2592000 --exclude=*.tmp --exclude=*.log`
   - Click "Create"
   - Verify job appears in list
   - Click job to view details
   - Verify sync_flags are displayed ✓

3. **Test Case 2: Create job WITHOUT sync_flags**
   - Fill in basic fields only (no Advanced Options)
   - Click "Create"
   - Verify job created successfully
   - Verify sync_flags not shown in details (or empty) ✓

4. **Test Case 3: Validation error**
   - Set Min Age: 30, Max Age: 10
   - Verify error message shows: "Min Age must be ≤ Max Age"
   - Verify Create button disabled or inactive ✓
   - Fix: Set Max Age: 30
   - Verify error clears ✓
   - Create succeeds ✓

5. **Database verification:**
   - Open coordinator database (SQLite)
   - Query: `SELECT id, name, sync_flags FROM jobs ORDER BY created_at DESC LIMIT 3`
   - Verify sync_flags column has JSON data for jobs with flags ✓
   - Verify sync_flags column is NULL/empty for jobs without flags ✓

6. **Verification:**
   - Jobs created with all sync flags ✓
   - Jobs created with partial sync flags ✓
   - Jobs created without sync flags ✓
   - Min/Max age validation works ✓
   - Exclude patterns work ✓
   - Data persists to database ✓
   - Data can be retrieved (GET /api/jobs) ✓

---

## Dependencies & Prerequisites

- ✓ Backend sync_flags implementation (Phase 19 complete)
- ✓ Database migration applied (sync_flags column exists)
- ✓ API endpoint accepts sync_flags (POST /api/jobs)
- ✓ Vue 3 with `<script setup>` available
- ✓ Existing ScheduleBuilder component for pattern reference
- ✓ Dashboard build system (npm build)

---

## Rollback Plan

If critical issues found:

1. **Revert SyncFlagsBuilder component:**
   ```bash
   git checkout dashboard/src/components/SyncFlagsBuilder.vue
   ```

2. **Revert Jobs.vue changes:**
   ```bash
   git checkout dashboard/src/views/Jobs.vue
   ```

3. **Rebuild dashboard:**
   ```bash
   cd dashboard && npm run build
   ```

4. **Restart coordinator:**
   ```bash
   .\scripts\rebuild-and-restart.ps1
   ```

---

## Success Checklist

- [ ] SyncFlagsBuilder.vue created and compiles
- [ ] All form fields render correctly
- [ ] Command preview logic working (both robocopy and rsync)
- [ ] Min/Max age validation enforced
- [ ] Jobs.vue imports and uses SyncFlagsBuilder
- [ ] API payload includes/omits sync_flags correctly
- [ ] Test job created with sync_flags
- [ ] Test job created without sync_flags
- [ ] Database stores sync_flags correctly
- [ ] Backward compatibility verified (old jobs still work)
- [ ] Dashboard deployed and running
- [ ] All 6 sync flag options fully functional

---

## Timeline

| Task | Est. Time | Status |
|------|-----------|--------|
| Task 1: Component structure | 15 min | — |
| Task 2: Form fields & validation | 20 min | — |
| Task 3: Command preview logic | 15 min | — |
| Task 4: Validation logic | 10 min | — |
| Task 5: Jobs.vue integration | 15 min | — |
| Task 6: End-to-end testing | 20 min | — |
| **Total** | **95 min (~1.5 hrs)** | — |

---

## Notes

- Design spec provides complete field details and validation rules
- Reference `agent/runner/sync_flags.go` for exact flag syntax
- Command preview must exactly match backend ToRobocopyArgs() and ToRsyncArgs()
- Keep component reusable — may be used in other forms later
- Test on Windows (robocopy paths) and mentally verify Unix (rsync paths)

