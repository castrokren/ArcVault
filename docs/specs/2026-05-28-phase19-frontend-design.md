# Phase 19 Frontend Design: Sync Flags Component

**Date:** May 28, 2026  
**Phase:** 19 (Robocopy/Rsync Advanced Flags)  
**Status:** Design Approved  
**Component:** SyncFlagsBuilder.vue

---

## Overview

Add a reusable Vue component `SyncFlagsBuilder.vue` to the Jobs creation form that allows users to configure advanced backup sync options (mirror mode, file age filters, size limits, exclusion patterns) without cluttering the basic form.

Users can:
- Enable mirror mode (delete destination files not in source)
- Filter files by age (max days old, min days old)
- Limit file size (max MB)
- Exclude specific files and directories by pattern
- See a live preview of the generated robocopy/rsync command

---

## Requirements

### Functional Requirements

1. **Component must accept and emit sync_flags object via v-model**
   - Input: `{ mirror, max_age, min_age, max_size, exclude_files, exclude_dirs }`
   - Output: Same structure, updated on user changes

2. **Collapsible Advanced Options section**
   - Collapsed by default
   - Click "Advanced Options" to expand inline within the Jobs form
   - Clean form presentation for basic users

3. **Three grouped field sections**

   **Filtering Section:**
   - Max Age input (number, optional, days)
   - Min Age input (number, optional, days)
   - Max Size input (number, optional, MB)

   **Behavior Section:**
   - Mirror checkbox (boolean)

   **Exclusions Section:**
   - Exclude Files textarea (one pattern per line)
   - Exclude Directories textarea (one pattern per line)

4. **Live command preview**
   - Shows robocopy command (Windows syntax)
   - Shows rsync command (Unix/Mac syntax)
   - Updates as user types
   - Read-only display for reference

5. **Input hints and helper text**
   - Placeholder text in inputs guides format and examples
   - Small helper text below fields explains purpose
   - Real-time validation for min/max age constraint

6. **Validation**
   - Min Age ≤ Max Age (real-time error if violated)
   - Non-negative numbers (enforced by input type="number" min="0")
   - Empty fields treated as null/"no limit"
   - Exclude patterns: no validation (user's responsibility)

---

## Data Model

### Form Structure

```javascript
sync_flags: {
  mirror: boolean,           // mirror mode enabled?
  max_age: number | null,    // days, or null for no limit
  min_age: number | null,    // days, or null for no limit
  max_size: number | null,   // MB, or null for no limit
  exclude_files: string[],   // file patterns to exclude
  exclude_dirs: string[]     // directory patterns to exclude
}
```

### Parsing Exclude Patterns

Users input patterns one per line in text areas. Component parses:
```
textarea value:        Component output:
"*.tmp\n*.log"    -->  ["*.tmp", "*.log"]
"*.tmp"           -->  ["*.tmp"]
""                -->  []
```

Whitespace trimming: `"  *.tmp  ".trim()` → `"*.tmp"`

---

## Component API

### Props
None (data comes via v-model)

### v-model

**Receives:**
```javascript
{
  mirror: false,
  max_age: null,
  min_age: null,
  max_size: null,
  exclude_files: [],
  exclude_dirs: []
}
```

**Emits:**
```javascript
{
  mirror: true,
  max_age: 30,
  min_age: 1,
  max_size: 2048,
  exclude_files: ["*.tmp", "*.log"],
  exclude_dirs: [".git", "node_modules"]
}
```

---

## UI Layout

### Placement in Jobs Form

```
Jobs.vue:
┌─────────────────────────────────────┐
│ Create Job                          │
├─────────────────────────────────────┤
│ Dispatch Mode: [●Agent] [Group]     │
│                                     │
│ Agent: [dropdown]                   │
│ Name: [_____________]               │
│ Source Path: [________________]      │
│ Dest Path: [________________]        │
│                                     │
│ Schedule: [ScheduleBuilder]         │
│                                     │
│ [Advanced Options ▼]  <-- CLICK     │
│                                     │
│ [SYNC FLAGS VISIBLE WHEN EXPANDED]  │
│                                     │
│                  [Create] [Cancel]  │
└─────────────────────────────────────┘
```

### SyncFlagsBuilder Expanded View

```
┌─────────────────────────────────────┐
│ Advanced Options ▲                  │
├─────────────────────────────────────┤
│                                     │
│ Filtering                           │
│   Max Age (days): [____]            │
│     Sync only files modified within │
│     N days. Leave blank for no limit│
│                                     │
│   Min Age (days): [____]            │
│     Sync only files not modified    │
│     within N days.                  │
│                                     │
│   Max Size (MB): [____]             │
│     Sync only files smaller than    │
│     N MB. Leave blank for no limit. │
│                                     │
│ Behavior                            │
│   ☐ Mirror mode                     │
│     Delete destination files not    │
│     in source                       │
│                                     │
│ Exclusions                          │
│   Exclude Files:                    │
│   [textarea: one per line]          │
│     *.tmp                           │
│     *.log                           │
│                                     │
│   Exclude Directories:              │
│   [textarea: one per line]          │
│     .git                            │
│     node_modules                    │
│                                     │
│ Command Preview                     │
│ ┌───────────────────────────────┐   │
│ │ robocopy C:\data D:\backup    │   │
│ │ /MIR /MAXAGE:30 /MAXSIZE:2048M│   │
│ │ /XF *.tmp *.log /XD .git ...  │   │
│ └───────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

---

## Input Field Specifications

### Filtering Section

**Max Age Input**
- Type: `<input type="number" min="0">`
- Placeholder: "Leave blank for no limit. Days, e.g., 30"
- Helper text: "Sync only files modified within N days"
- v-model: `sync_flags.max_age`
- Validation: Error if min_age > 0 AND min_age > max_age

**Min Age Input**
- Type: `<input type="number" min="0">`
- Placeholder: "e.g., 1"
- Helper text: "Sync only files not modified within N days"
- v-model: `sync_flags.min_age`
- Validation: Error if max_age > 0 AND min_age > max_age

**Max Size Input**
- Type: `<input type="number" min="0">`
- Placeholder: "MB, e.g., 2048"
- Helper text: "Sync only files smaller than N MB"
- v-model: `sync_flags.max_size`

### Behavior Section

**Mirror Checkbox**
- Type: `<input type="checkbox">`
- Label: "Mirror mode"
- Helper text: "Delete destination files not in source"
- v-model: `sync_flags.mirror`

### Exclusions Section

**Exclude Files Textarea**
- Type: `<textarea>`
- Placeholder: "One pattern per line\n*.tmp\n*.log"
- Helper text: "File patterns to exclude (wildcards supported: *, ?, [...]). Example: *.tmp, *.log, Config.ini"
- Rows: 3
- v-model: `sync_flags.exclude_files` (parsed as array, split by newline)
- Parsing: Split by `\n`, trim each line, filter empty strings
- **Pattern syntax note:** 
  - Robocopy: Supports wildcards `*`, `?`, `[abc]`, literal filenames
  - Rsync: Supports wildcards `*`, `?`, `[abc]`, `**` for recursive matching
  - Patterns without `/` match filenames only; patterns with `/` match full paths

**Exclude Directories Textarea**
- Type: `<textarea>`
- Placeholder: "One pattern per line\n.git\nnode_modules"
- Helper text: "Directory patterns to exclude (one per line). Example: .git, node_modules, $Recycle.Bin"
- Rows: 3
- v-model: `sync_flags.exclude_dirs` (parsed as array, split by newline)
- Parsing: Split by `\n`, trim each line, filter empty strings
- **Pattern syntax note:**
  - Patterns are matched against directory names
  - Robocopy: `/XD .git node_modules` excludes those directories anywhere in tree
  - Rsync: `--exclude=.git` matches directory name; `--exclude=.git/` explicitly matches directories only

---

## Command Preview

### Display

Shows two command examples (robocopy + rsync) side-by-side or stacked:

```
Robocopy:
robocopy C:\data D:\backup /MIR /MAXAGE:30 /MAXSIZE:2048M /XF *.tmp *.log /XD .git

Rsync:
rsync -a --delete --max-age=2592000 --maxsize=2147483648 --exclude=*.tmp --exclude=.git /data /backup/
```

### Generation Logic

Call the SyncFlags methods from the backend (sync_flags.go):
- `ToRobocopyArgs()` → array of flags → display as command
- `ToRsyncArgs()` → array of flags → display as command

**Unit conversion reference** (verified against robocopy/rsync documentation):
- **Robocopy:** `/MAXAGE:30` and `/MINAGE:30` use days (input directly)
- **Robocopy:** `/MAXSIZE:2048M` uses MB with M suffix (we add the M)
- **Rsync:** `--max-age=N` and `--min-age=N` use seconds (days × 86400)
- **Rsync:** `--maxsize=N` uses bytes (MB × 1,048,576)
- **Both:** Patterns passed as-is (no escaping needed for basic wildcards like `*.tmp`)

**Implementation approach:** Duplicate the command building logic in JavaScript so preview updates instantly as user types. Keep it in sync with backend by:
1. Reference backend code (sync_flags.go) in code comments
2. Match the exact flag names and syntax
3. Test both commands generate identically to backend methods

### Update Trigger

Command preview updates:
- On any input change (debounced if needed)
- Real-time as user types

---

## Validation Rules

### Real-Time Validation

1. **Min Age vs Max Age**
   - If both > 0 and min_age > max_age: Show error
   - Error message: "Min Age must be ≤ Max Age"
   - Prevent form submission while error is active

2. **Negative Values**
   - Prevented by `<input type="number" min="0">`

3. **Exclude Patterns**
   - No validation on pattern syntax (user's responsibility)
   - Basic wildcards (`*`, `?`, `[...]`) are supported in both robocopy and rsync
   - Special characters in patterns should work as-is (e.g., `*.tmp`, `.git`, `$Recycle.Bin`)

4. **Empty Fields**
   - Treated as null/"no limit"
   - No error shown for empty fields
   - Empty exclude lists are valid (just `[]`)

### Form Submission

When user clicks "Create" in Jobs.vue:
- If any sync_flags field is set: include in API payload
- If all sync_flags fields are empty: omit from payload
- Validation errors in SyncFlagsBuilder block form submission

---

## Integration with Jobs.vue

### Parent Component Changes

1. **Import SyncFlagsBuilder**
   ```javascript
   import SyncFlagsBuilder from '../components/SyncFlagsBuilder.vue'
   ```

2. **Add to form data**
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

3. **Add to template**
   ```vue
   <ScheduleBuilder v-model="form.schedule" />
   <SyncFlagsBuilder v-model="form.sync_flags" />  <!-- NEW -->
   <div class="form-actions">...</div>
   ```

4. **Update createJob() payload building**
   ```javascript
   const payload = { ...form.value }
   delete payload.dispatchMode
   if (!payload.schedule) delete payload.schedule
   if (!payload.sync_flags || Object.values(payload.sync_flags).every(v => !v || v.length === 0)) {
     delete payload.sync_flags  // Omit if all empty
   }
   ```

5. **Reset on success**
   ```javascript
   form.value.sync_flags = {}  // Reset along with other fields
   ```

---

## Styling & Theme

- Use existing ArcVault CSS patterns
- Match ScheduleBuilder styling for consistency
- Support dark/light theme
- Responsive on mobile (stack sections vertically if needed)

---

## Testing Strategy

### Unit Tests (Vue component)

- Parse exclude patterns correctly (split by newline, trim)
- Min/Max age validation triggers errors
- v-model updates propagate to parent
- Command preview generates correct flags
- Empty fields don't break command preview

### Integration Tests (Jobs.vue)

- SyncFlagsBuilder integrates with Jobs form
- API payload includes sync_flags when set
- API payload omits sync_flags when empty
- Job created successfully with sync_flags
- Verify sync_flags stored in database

### Manual Testing

- Create job with all sync flags set
- Create job with some flags set
- Create job with no flags (sync_flags omitted)
- Verify command preview updates in real-time
- Verify min/max age validation error shows/clears
- Test on Windows (robocopy) and Unix (rsync) command preview

---

## Files to Create/Modify

### New Files
- `dashboard/src/components/SyncFlagsBuilder.vue` (new component)

### Modified Files
- `dashboard/src/views/Jobs.vue` (import, integrate SyncFlagsBuilder)
- Tests (if applicable)

### No Changes Needed
- Backend API (already accepts sync_flags)
- Database (migration already in place)
- coordinator/agent code (already handles sync_flags)

---

## Rollout Plan

1. Create SyncFlagsBuilder.vue component
2. Integrate into Jobs.vue form
3. Test locally (create job with sync_flags)
4. Rebuild and deploy dashboard
5. Manual QA: verify command preview and database storage
6. Document sync flags in user guide (future task)

---

## Success Criteria

✅ SyncFlagsBuilder component renders correctly  
✅ Users can set mirror, age filters, size limits, exclusions  
✅ Command preview shows accurate robocopy/rsync command  
✅ Min/Max age validation works  
✅ Jobs created with sync_flags are stored and retrieved correctly  
✅ Collapsible section keeps form clean for basic users  
✅ Power users can access advanced options without friction  

---

## Notes

- This design follows the existing ScheduleBuilder pattern for consistency
- Collapsible approach keeps the form clean while remaining discoverable
- Command preview helps users verify their choices before submission
- Real-time validation prevents invalid configurations
- Empty sync_flags are omitted from API payload for backward compatibility

