Set-Location C:\Projects\ArcVault2.0

# 1. Remove misplaced router/App.vue (force — it has local modifications we don't need)
if (Test-Path dashboard\src\router\App.vue) {
    Write-Host "Removing misplaced dashboard/src/router/App.vue..."
    git rm -f dashboard\src\router\App.vue
}

# 2. Stage all tracked modified files (respects .gitignore, skips binaries)
#    This picks up: agent/main.go, coordinator/main.go, coordinator/config/config.go,
#    dashboard/vite.config.js, CONTEXT.md, MEMORY.md, and the phase14 deletion
Write-Host "Staging tracked changes..."
git add -u

# 3. Also stage untracked files that belong in the repo
Write-Host "Staging new files..."
if (Test-Path .github)                                           { git add .github\ }
if (Test-Path deployment)                                        { git add deployment\ }
if (Test-Path upload_v1_assets.ps1)                              { git add upload_v1_assets.ps1 }
if (Test-Path "planning\2026-05-22-phase18-deployment-package-design.md") {
    git add "planning\2026-05-22-phase18-deployment-package-design.md"
}

# 4. Show what is about to be committed
Write-Host "`nAbout to commit:"
git status --short

# 5. Commit
$msg = @"
feat: Windows SCM fixes, vite config, cleanup, Phase 18 deployment prep

- agent/main.go: resolve config path relative to exe (fixes SCM starting from system32)
- agent/main.go: add run-service subcommand for Windows SCM handshake
- coordinator/main.go: add run-service subcommand for Windows SCM handshake
- coordinator/config/config.go: related config load updates
- dashboard/vite.config.js: set emptyOutDir=false (preserves Go embed on rebuild)
- Remove dashboard/src/router/App.vue (misplaced duplicate of src/App.vue)
- Remove phase14-smoke-test-plan.md (stale)
- Add deployment scripts and .github workflow stubs
- Update CONTEXT.md and MEMORY.md
"@
git commit -m $msg

# 6. Push
Write-Host "`nPushing to origin..."
git push

Write-Host "`nDone!"
