Set-Location "C:\Projects\ArcVault2.0"

Write-Host "==> Staging all changes..." -ForegroundColor Cyan
git add -A

Write-Host "==> Committing reorganization..." -ForegroundColor Cyan
git commit -m "chore: reorganize project file structure

- Move build/test scripts to scripts/
- Move guides and runbooks to docs/
- Move planning docs to planning/
- Move task summaries to tasks/completed/
- Add framework/, system/, memory/, tasks/ directories"

Write-Host "==> Pushing to GitHub..." -ForegroundColor Cyan
git push origin feature/phase-18-installers

Write-Host "==> Deleting local phase-14-federation branch..." -ForegroundColor Cyan
git branch -d phase-14-federation

Write-Host ""
Write-Host "Done! Now go merge PR #1 at:" -ForegroundColor Green
Write-Host "https://github.com/castrokren/ArcVault/pull/1" -ForegroundColor Yellow
