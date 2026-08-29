# ECC MCP Server Validation Plan

**Date:** 2026-07-01
**Prerequisites:**
1. `setx GITHUB_TOKEN "your-github-pat-here"` — replace with actual PAT from https://github.com/settings/tokens
2. Restart OpenCode
3. Verify startup shows "Connected to MCP server: github" and "Connected to MCP server: memory"

## TASK-10: Validate GitHub MCP

### Test 1: List Issues
- **Prompt:** "List open issues in castrokren/ArcVault"
- **Expected:** Agent uses GitHub MCP tool, returns list of issues (or empty if none)
- **Pass criteria:** No error about missing MCP tool

### Test 2: List PRs
- **Prompt:** "List PRs merged in the last week for castrokren/ArcVault"
- **Expected:** Agent returns PR list (or empty)
- **Pass criteria:** Correct data returned

### Test 3: Create Issue (OPTIONAL — creates real issue)
- **Prompt:** "Create a GitHub issue: Bug in agent heartbeat logic"
- **Expected:** Issue created
- **Pass criteria:** Issue appears on GitHub
- **Note:** Only run if you want to create a real issue. Delete it after.

### Success Criteria
- ✅ Agent uses GitHub MCP tool (visible in tool call output)
- ✅ Correct issue/PR data returned
- ✅ No "MCP server not found" errors
- ✅ Time savings vs manual GitHub navigation ≥30%

## TASK-12: Validate Memory MCP

### Test 1: Store Facts
- **Prompt:** "Remember: ArcVault uses Go 1.25 and Vue 3 with Composition API"
- **Expected:** Agent confirms it stored the fact
- **Pass criteria:** No errors

### Test 2: Cross-Session Recall
- Start a NEW OpenCode session
- **Prompt:** "What version of Go does ArcVault use?"
- **Expected:** Agent recalls "Go 1.25" (or whatever was stored)
- **Pass criteria:** Correct recall

### Test 3: Store Multiple Facts
- **Prompt:** "Remember these facts about ArcVault:
  1. Database: SQLite via modernc.org/sqlite (pure Go, no CGO)
  2. Port: Coordinator listens on :8443
  3. Config format: JSON"
- **Expected:** Agent stores all three
- **Pass criteria:** No errors

### Test 4: Cross-Session Recall of Multiple Facts
- Start ANOTHER new session
- **Prompt:** "What port does the coordinator listen on?"
- **Expected:** Agent recalls ":8443"
- **Pass criteria:** Correct recall

### Success Criteria
- ✅ Memory MCP successfully stores facts
- ✅ Recall accuracy ≥90% across sessions
- ✅ Memory persists after session restart
- ✅ No errors or confusion

## Rollback Commands
If validation fails for either server:
1. Remove from `C:\Projects\opencode.jsonc` mcpServers section
2. `git checkout ecc-adoption-baseline -- mcp-configs/`
3. Restart OpenCode