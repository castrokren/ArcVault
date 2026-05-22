# Multi-Module Deploy-from-GitHub Fix
**Date:** 2026-05-13  
**Status:** ✅ TESTED & WORKING

---

## Problem Solved

The `deploy-from-github.bat` script was **failing** because it looked for `setup.bat` in the wrong directory.

**Error:** `[ERROR] setup.bat not found in Multi-module root`

**Root Cause:** The deployment script assumed `setup.bat` was in the Multi-module root directory (`C:\Users\kren\Desktop\Multi-module\`), but the actual location is in the PROJECTS subfolder (`C:\Users\kren\Desktop\Multi-module\PROJECTS\setup.bat`).

---

## Solution Implemented

### The Fix
Changed the deploy script to:
1. Check for `PROJECTS\setup.bat` instead of just `setup.bat`
2. Navigate into the PROJECTS directory before calling setup.bat
3. Let setup.bat run from its correct location

### Code Change
**Before (lines 66-72):**
```batch
if not exist setup.bat (
    echo [ERROR] setup.bat not found in Multi-module root
    pause
    exit /b 1
)

call setup.bat
```

**After (lines 66-72):**
```batch
if not exist PROJECTS\setup.bat (
    echo [ERROR] setup.bat not found in PROJECTS subfolder
    pause
    exit /b 1
)

REM Navigate to PROJECTS directory and run setup
cd PROJECTS
call setup.bat
```

---

## Understanding the Setup Process

The `PROJECTS\setup.bat` performs a 6-step automated setup:

1. **Check Python** — Verifies Python 3.8+ is installed and in PATH
2. **Create venv** — Creates virtual environment (`venv\` folder)
3. **Install deps** — Activates venv and runs `ops\check_dependencies.py`
4. **Create dirs** — Creates required folders (`ops`, `data`, `data\som-in`, results directory)
5. **HTTPS cert** — Generates self-signed certificate via `ops\generate_cert.py`
6. **Start services** — Launches two services in new CMD windows:
   - `folder_monitor_service.py` (monitors for Excel files)
   - `dashboard.py` (runs HTTPS dashboard on localhost)

**Post-setup behavior:**
- Opens dashboard at `https://localhost` in browser
- User accepts self-signed cert warning
- Dashboard shows "Idle - Waiting for input file..."
- Pipeline ready to process Excel files from `data\som-in\`

---

## Testing & Deployment

### Local Testing ✅
- Fixed script created and tested on 2026-05-13
- Successfully clones repo, checks out branch, navigates to PROJECTS, and runs setup.bat
- Services start without errors
- **Status:** WORKING

### Next Steps
1. **Update GitHub** — Replace old deploy-from-github.bat with fixed version
2. **Commit message:** "Fix: correct path to setup.bat in PROJECTS subfolder"
3. **Push to:** `claude/pedantic-hofstadter-313610` branch
4. **Test remote deployment** — Clone from GitHub and verify deploy script works from remote

---

## File Locations
- **Deploy script:** `C:\Users\kren\Desktop\deploy-from-github.bat` (local, tested)
- **Setup script:** `C:\Users\kren\Desktop\Multi-module\PROJECTS\setup.bat`
- **Repository:** `https://github.com/castrokren/Multi-module.git`
- **Target branch:** `claude/pedantic-hofstadter-313610`

---

## Key Insights for Future Sessions

1. **Path issue was critical** — Deploy scripts need to know exact subdirectory structure
2. **The setup.bat uses relative paths** — It assumes it runs from PROJECTS directory (`%~dp0`)
3. **Service startup** — The setup launches two separate CMD windows; normal if they stay open
4. **Self-signed cert** — Dashboard uses self-signed HTTPS; browser will warn but it's expected

---

## Commands for Next Session

**To push to GitHub:**
```powershell
cd C:\Users\kren\Desktop\Multi-module
git add deploy-from-github.bat
git commit -m "Fix: correct path to setup.bat in PROJECTS subfolder"
git push origin claude/pedantic-hofstadter-313610
```

**To test remote deployment:**
1. Delete local Multi-module folder from Desktop
2. Run the newly-pushed `deploy-from-github.bat` from GitHub
3. Verify it clones and sets up correctly

---

## Related Context
- **Scheduled Task:** `test-crossref-standalone-fast` — Verification script for crossref module (requires project folder mounted)
- **Project Structure:** Multi-module is a monorepo with PROJECTS subfolder containing the crawler pipeline
- **User Email:** castrokren@gmail.com
