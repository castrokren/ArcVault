# Supply Chain Security Scan Results
**Date:** 2026-07-01
**Project:** ArcVault2.0

## Scan Summary
| Category | Count |
|---|---|
| Go dependencies | 19 |
| Dashboard npm dependencies | 12 |
| Root npm dependencies | 0 |
| Known vulnerabilities found | 0 (basic scan) |

## Go Dependencies
- `github.com/golang-jwt/jwt/v5@v5.3.1`
- `github.com/gorilla/websocket@v1.5.3`
- `github.com/robfig/cron/v3@v3.0.1`
- `github.com/stretchr/testify@v1.11.1`
- `golang.org/x/crypto@v0.53.0`
- `golang.org/x/sys@v0.46.0`
- `golang.org/x/time@v0.15.0`
- `gopkg.in/yaml.v3@v3.0.1`
- `modernc.org/sqlite@v1.50.1`
- `github.com/davecgh/go-spew@v1.1.1`
- `github.com/dustin/go-humanize@v1.0.1`
- `github.com/google/uuid@v1.6.0`
- `github.com/mattn/go-isatty@v0.0.20`
- `github.com/ncruces/go-strftime@v1.0.0`
- `github.com/pmezard/go-difflib@v1.0.0`
- `github.com/remyoudompheng/bigfft@v0.0.0-20230129092748-24d4a6f8daec`
- `modernc.org/libc@v1.72.3`
- `modernc.org/mathutil@v1.7.1`
- `modernc.org/memory@v1.11.0`

## Dashboard npm Dependencies
- `@fontsource/inter@5.2.8`
- `@fontsource/jetbrains-mono@5.2.8`
- `@fontsource/space-grotesk@5.2.10`
- `motion-v@2.3.0`
- `vue@3.5.34`
- `vue-router@4.6.4`
- `zod@4.4.3`
- `@vitejs/plugin-vue@6.0.6`
- `@vue/test-utils@2.0.0`
- `jsdom@29.1.1`
- `vite@8.0.12`
- `vitest@4.1.9`

## Notes
- This is a basic dependency inventory scan.
- For full CVE checking, run: `npm audit` in dashboard/
- For Go: `go list -m -u all` and review for available updates
- Consider integrating `govulncheck` for Go vulnerability scanning