#!/usr/bin/env pwsh
# Enable the tracked git hooks (doc-drift guard). Run once per clone.
# .git/hooks is not version-controlled, so we point git at the tracked dir instead.
git config core.hooksPath scripts/git-hooks
Write-Host "core.hooksPath set to scripts/git-hooks" -ForegroundColor Green
Write-Host "Pre-commit will now block commits that drift docs/{backend,service,frontend}.md." -ForegroundColor Gray
