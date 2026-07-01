# Cost Tracking Hook Validation Plan

**Date:** 2026-07-01

## Prerequisites
- `node` available
- OpenCode session

## Test 1: Hook Syntax Check
```bash
node .claude/hooks/cost-tracker.js
```
Expected: Outputs "[Hook] cost-tracker: logged session end"
Pass: No errors, exits cleanly

## Test 2: Log File Creation
After running Test 1, check:
```bash
type .claude\cost-log.json
```
Expected: Valid JSON with timestamp entry
Pass: File exists, contains valid JSON with at least 1 entry

## Test 3: Multi-Session Logging
1. Start OpenCode session from ArcVault2.0 directory
2. Run several commands
3. End session
4. Check `.claude/cost-log.json` — new entry should appear
5. Repeat 2-3 times

Expected: Multiple entries appended to log
Pass: Each session adds an entry; file remains valid JSON

## Success Criteria
- ✅ Hook runs without errors
- ✅ Cost log created at `.claude/cost-log.json`
- ✅ Multiple sessions correctly append entries
- ✅ No session interference (normal commands still work)

## Rollback
```
git checkout ecc-adoption-baseline -- .claude/hooks/cost-tracker.js .claude/settings.json
```
