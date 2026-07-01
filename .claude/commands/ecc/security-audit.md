---
description: Security audit — OWASP Top 10, secrets exposure, input validation, auth gaps
argument-hint: [file-or-directory-path | blank for entire project]
---

# Security Audit

**Input**: $ARGUMENTS

---

## Overview

Run a security audit on the specified files or the entire project. Checks against OWASP Top 10, secret exposure, input validation, and authentication/authorization gaps.

---

## Phase 1 — SCOPE

If `$ARGUMENTS` is empty, default to scanning the current project's source directories (`coordinator/`, `agent/`, `dashboard/`). Otherwise, use the provided path(s).

```bash
# List all files in scope
Get-ChildItem -Recurse -Include *.go,*.js,*.ts,*.vue,*.py,*.yaml,*.yml $SCOPE
```

---

## Phase 2 — SCAN

For each file, check the following categories:

### 1. Secrets & Credential Exposure (CRITICAL)
- Hardcoded API keys, tokens, passwords, connection strings
- `.env` files or environment variable defaults committed
- Private keys (.pem, .key) in source
- JWT secrets or signing keys in code
- Database connection strings with credentials

### 2. Injection Vulnerabilities (CRITICAL)
- SQL injection (string concatenation in queries)
- NoSQL injection
- Command injection (shell execution with user input)
- Template injection (unsafe template rendering)
- LDAP / XML injection

### 3. Authentication & Authorization (HIGH)
- Missing or weak authentication on endpoints
- Insecure direct object references (IDOR)
- Missing role/scope checks
- Hardcoded admin credentials
- Session fixation or weak session management
- Missing JWT validation (exp, iss, aud)

### 4. Input Validation (HIGH)
- Missing input sanitization on user-facing endpoints
- Path traversal (unsafe file path construction)
- Unvalidated redirects/forwards
- Mass assignment / parameter pollution
- File upload without type/size validation

### 5. XSS & CSRF (MEDIUM)
- Reflected XSS (unsafe query parameter rendering)
- Stored XSS (unsafe user content rendering)
- DOM-based XSS
- Missing CSRF tokens on state-changing endpoints

### 6. Security Misconfiguration (MEDIUM)
- Debug/verbose error messages in production
- CORS misconfiguration (wildcard origins with credentials)
- Missing security headers (CSP, HSTS, X-Frame-Options)
- Exposed admin panels or debugging endpoints
- Default credentials still in place

### 7. Dependency Vulnerabilities (MEDIUM)
- Outdated packages with known CVEs
- Unnecessary dependencies with large attack surface
- Insecure package sources or pinned versions

---

## Phase 3 — REPORT

Generate a findings report:

```markdown
# Security Audit Report

**Date**: <date>
**Scope**: <files/directories scanned>
**Files Scanned**: <count>

## Summary
| Severity | Count |
|----------|-------|
| CRITICAL | <n>    |
| HIGH     | <n>    |
| MEDIUM   | <n>    |
| LOW      | <n>    |

## Findings

### CRITICAL
| File | Line | Issue | Recommendation |
|------|------|-------|----------------|
| `path` | N | Description | How to fix |

### HIGH
...

### MEDIUM
...

### LOW
...

## Recommendations
1. Priority actions (ordered by risk)
2. Quick wins
3. Long-term improvements
```

Write the report to `.claude/reviews/security-audit-<date>.md`.

---

## Phase 4 — VALIDATE

If applicable, run automated tools:

```bash
# Go
go vet ./... 2>&1

# Node/TypeScript
npm audit 2>/dev/null
npx eslint . --format json 2>/dev/null

# Generic
# Check for .env files in source
Get-ChildItem -Recurse -Filter ".env*" | Select FullName
# Check for .pem/.key files
Get-ChildItem -Recurse -Filter "*.pem","*.key" | Select FullName
```

---

## Phase 5 — OUTPUT

Report findings to user with actionable remediation steps. Block on any CRITICAL findings.

## Edge Cases
- **Binary files**: Skip non-text files (images, compiled binaries)
- **Large projects**: Limit scan to source directories; skip node_modules, vendor, dist
- **No findings**: Explicitly state "No security issues found" — do not leave empty
- **False positives**: Use reasonable judgment; mark likely false positives as LOW
