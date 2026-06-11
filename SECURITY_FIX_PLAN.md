# ArcVault 2.0 — Security Fix Plan

**Generated:** 2026-06-11  
**Based on:** ArcVault-Security-Review.docx  
**For Claude to execute in a fresh session.**

---

## How to use this plan

Load this file at the start of a new session, then work through fixes in order.
Each fix is self-contained with exact file paths, the current code, and the replacement.
After all code changes are made, run the manual steps in the **Secrets Rotation** section first —
those cannot be done via code edit.

---

## CRITICAL: Secrets Rotation (Manual — do first, before any code commits)

These steps must be done by a human before deploying any code changes.

1. **Generate new secrets:**
   ```powershell
   # Run these in PowerShell on the coordinator host
   # New admin token
   [System.BitConverter]::ToString([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).Replace("-","").ToLower()
   # New JWT secret
   [System.BitConverter]::ToString([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).Replace("-","").ToLower()
   # New credential key (if using config-based key)
   [System.BitConverter]::ToString([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).Replace("-","").ToLower()
   ```

2. **Update `config.json`** with the new values (file is already gitignored).

3. **Regenerate TLS cert + key:**
   ```
   coordinator init   # or the equivalent init command for your environment
   ```

4. **Audit git history** for old secrets:
   ```bash
   git log --all -S "81593fbb740999bc" --oneline   # replace with your old token prefix
   git log --all -S "a2e442a3bf8d5b0c" --oneline   # replace with your old jwt_secret prefix
   ```
   If any commits are found, use `git filter-repo` to purge them and force-push.

5. **Invalidate all existing agent tokens** by running:
   ```sql
   DELETE FROM tokens;
   ```
   Then re-register all agents (they'll get new tokens on next heartbeat after restart).

---

## FIX 1 (HIGH-2): Login rate limiting

**File:** `coordinator/server/auth.go`  
**Goal:** Reject excessive login attempts per IP with a 429 before bcrypt is even called.

### Step 1 — Add rate limiter to server struct

**File:** `coordinator/server/server.go`

Add import `"golang.org/x/time/rate"` and `"sync"` (sync is already imported).

Find the `Server` struct and add two fields after `tokenCache`:

```go
// Current code (around line 30):
type Server struct {
	cfg            *config.Config
	db             *db.DB
	router         *http.ServeMux
	hub            *Hub
	fedHub         *FederationHub
	fedClient      *FederationClient
	staticFS       fs.FS
	Notifier       *notifications.Dispatcher
	coordinatorID  string
	agentService   *business.AgentService
	jobService     *business.JobService
	userService    *business.UserService
	groupService   *business.GroupService
	tokenCacheMu   sync.Mutex
	tokenCache     map[string]tokenCacheEntry
}
```

Replace with:

```go
type Server struct {
	cfg            *config.Config
	db             *db.DB
	router         *http.ServeMux
	hub            *Hub
	fedHub         *FederationHub
	fedClient      *FederationClient
	staticFS       fs.FS
	Notifier       *notifications.Dispatcher
	coordinatorID  string
	agentService   *business.AgentService
	jobService     *business.JobService
	userService    *business.UserService
	groupService   *business.GroupService
	tokenCacheMu   sync.Mutex
	tokenCache     map[string]tokenCacheEntry
	loginLimiterMu sync.Mutex
	loginLimiters  map[string]*loginRateLimiter
}

// loginRateLimiter tracks failed attempts per IP.
type loginRateLimiter struct {
	limiter  *rate.Limiter
	failures int
	lockedAt *time.Time
}
```

### Step 2 — Initialize the map in `NewWithFS`

Find `tokenCache: make(map[string]tokenCacheEntry),` in `NewWithFS` and add the line below it:

```go
// ADD after tokenCache line:
loginLimiters: make(map[string]*loginRateLimiter),
```

### Step 3 — Add rate limiter helper methods

**File:** `coordinator/server/auth.go`

Add a new import `"golang.org/x/time/rate"` to the import block (it's already in go.mod as an indirect dep; if not present run `go get golang.org/x/time/rate`).

Add these methods anywhere in `auth.go` before `handleLogin`:

```go
// loginAllowed checks per-IP rate limit for login attempts.
// Returns false (and writes 429) if the IP is rate-limited or locked out.
// Call this at the top of handleLogin before any bcrypt work.
func (s *Server) loginAllowed(w http.ResponseWriter, r *http.Request) bool {
	ip := r.RemoteAddr
	// Strip port
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	s.loginLimiterMu.Lock()
	entry, ok := s.loginLimiters[ip]
	if !ok {
		// 5 attempts per minute, burst of 5
		entry = &loginRateLimiter{
			limiter: rate.NewLimiter(rate.Every(time.Minute/5), 5),
		}
		s.loginLimiters[ip] = entry
	}

	// Check lockout (10-minute lockout after 10 consecutive failures)
	if entry.lockedAt != nil {
		if time.Since(*entry.lockedAt) < 10*time.Minute {
			s.loginLimiterMu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "too many failed attempts, try again later"})
			return false
		}
		// Lockout expired — reset
		entry.lockedAt = nil
		entry.failures = 0
	}
	s.loginLimiterMu.Unlock()

	if !entry.limiter.Allow() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "too many requests"})
		return false
	}
	return true
}

// recordLoginFailure increments the failure counter for an IP and locks it out at 10.
func (s *Server) recordLoginFailure(r *http.Request) {
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	s.loginLimiterMu.Lock()
	defer s.loginLimiterMu.Unlock()
	if entry, ok := s.loginLimiters[ip]; ok {
		entry.failures++
		if entry.failures >= 10 {
			now := time.Now()
			entry.lockedAt = &now
		}
	}
}

// recordLoginSuccess resets the failure counter for an IP on successful login.
func (s *Server) recordLoginSuccess(r *http.Request) {
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	s.loginLimiterMu.Lock()
	defer s.loginLimiterMu.Unlock()
	if entry, ok := s.loginLimiters[ip]; ok {
		entry.failures = 0
		entry.lockedAt = nil
	}
}
```

### Step 4 — Wire into `handleLogin`

In `handleLogin`, find the line:

```go
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
```

Replace with:

```go
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limit before any bcrypt work
	if !s.loginAllowed(w, r) {
		return
	}

	var req LoginRequest
```

Then find the block that returns `StatusUnauthorized` after `ValidateCredentials` fails:

```go
	user, err := s.userService.ValidateCredentials(req.Username, req.Password)
	if err != nil {
		// Return generic error to prevent user enumeration
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid username or password"})
		return
	}
```

Replace with:

```go
	user, err := s.userService.ValidateCredentials(req.Username, req.Password)
	if err != nil {
		s.recordLoginFailure(r)
		log.Printf("Login failed for username %q from %s", req.Username, r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid username or password"})
		return
	}
	s.recordLoginSuccess(r)
```

### Step 5 — go.mod

Run in the project root:
```bash
go get golang.org/x/time/rate
go mod tidy
```

---

## FIX 2 (MEDIUM-1): Remove WebSocket token query param

**File:** `coordinator/server/hub.go`

### handleWS — remove ?token= fallback (lines ~139–141)

Find:

```go
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	// Accept admin token OR any valid JWT (issued after Phase 15 login).
```

Replace with:

```go
	// Note: ?token= query param removed — tokens must be in Authorization header.
	// Accept admin token OR any valid JWT (issued after Phase 15 login).
```

### handleAgentWS — remove ?token= fallback (lines ~181–183)

Find:

```go
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	// Accept both admin token and valid agent tokens.
```

Replace with:

```go
	// Note: ?token= query param removed — tokens must be in Authorization header.
	// Accept both admin token and valid agent tokens.
```

### Update dashboard WebSocket client

**File:** `dashboard/src/` — find where `new WebSocket(url)` is called with a `?token=` param.

```bash
grep -rn "token\|WebSocket\|ws://" dashboard/src/ | grep -i "token\|ws"
```

The dashboard WebSocket connection needs to pass the token in the protocol header instead.
In Vue, this is done via the second argument to `WebSocket`:

```js
// BEFORE (insecure — token in URL):
const ws = new WebSocket(`wss://${host}/ws?token=${token}`)

// AFTER (token in Sec-WebSocket-Protocol header as a bearer scheme):
const ws = new WebSocket(`wss://${host}/ws`, [`bearer.${token}`])
```

On the Go side, update `upgrader` in hub.go to read from the protocol header:

```go
// In handleWS, AFTER removing the query param fallback, add:
if token == "" {
    for _, proto := range r.Header["Sec-Websocket-Protocol"] {
        if strings.HasPrefix(proto, "bearer.") {
            token = strings.TrimPrefix(proto, "bearer.")
            break
        }
    }
}
```

And tell the upgrader to echo back the subprotocol:

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
    Subprotocols: []string{},  // will be set dynamically per connection
}
```

> **Note:** Confirm the exact location of the dashboard WebSocket instantiation before making this change, as the file path depends on the current dashboard src layout.

---

## FIX 3 (MEDIUM-2): Restrict CORS

**File:** `coordinator/server/server.go`

### Step 1 — Add `AllowedOrigins` to config

**File:** `coordinator/config/config.go`

Find the `Config` struct and add one field:

```go
// ADD to Config struct:
AllowedOrigins []string `json:"allowed_origins,omitempty"`
```

### Step 2 — Replace `corsMiddleware`

Find the function (line ~293):

```go
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

Replace with:

```go
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// If no allowed origins configured, deny all cross-origin requests.
			// If "*" is explicitly listed (dev only), allow all.
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if origin != "" && allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			} else if origin != "" && !allowed {
				// Non-matching origin: don't set ACAO header — browser will block it.
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

### Step 3 — Update the call site in `Start()`

Find:

```go
		Handler:      corsMiddleware(s.router),
```

Replace with:

```go
		Handler:      corsMiddleware(s.cfg.AllowedOrigins)(s.router),
```

### Step 4 — Add to config.json

```json
"allowed_origins": ["https://YOUR-COORDINATOR-HOST"]
```

For local dev, use `["*"]`. For production, use the real dashboard URL.

---

## FIX 4 (MEDIUM-3): Upgrade golang.org/x/crypto

Run in project root:

```bash
go get golang.org/x/crypto@latest
go mod tidy
go build ./...
go test ./...
```

Verify go.mod now shows `golang.org/x/crypto v0.31.0` or later.

---

## FIX 5 (MEDIUM-4): Binary update integrity check

**File:** `coordinator/updater/updater.go`

### Step 1 — Add `ExpectedSHA256` to `UpdateInfo`

Find:

```go
type UpdateInfo struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
	AssetURL        string `json:"asset_url"`
}
```

Replace with:

```go
type UpdateInfo struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
	AssetURL        string `json:"asset_url"`
	ChecksumURL     string `json:"checksum_url"` // URL to SHA256SUMS file
}
```

### Step 2 — Populate `ChecksumURL` in `CheckLatestRelease`

In `CheckLatestRelease`, after resolving `assetURL`, add:

```go
	checksumURL, _ := resolveAsset("SHA256SUMS", runtime.GOOS, runtime.GOARCH, release.Assets)
	// SHA256SUMS may be a single file (not platform-specific); try plain name too
	if checksumURL == "" {
		for _, asset := range release.Assets {
			if strings.EqualFold(asset.Name, "SHA256SUMS") || strings.EqualFold(asset.Name, "sha256sums.txt") {
				checksumURL = asset.DownloadURL
				break
			}
		}
	}
```

### Step 3 — Add `VerifyChecksum` function

Add this function after `DownloadBinary`:

```go
// VerifyChecksum downloads the SHA256SUMS file from checksumURL and verifies
// that the file at localPath matches the expected hash for assetName.
// Returns nil if the checksum matches or if checksumURL is empty (skips check).
func VerifyChecksum(checksumURL, assetName, localPath string) error {
	if checksumURL == "" {
		log.Printf("Updater: no checksum URL available, skipping integrity check")
		return nil
	}

	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}

	// Parse "hash  filename" lines
	var expected string
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.EqualFold(fields[1], assetName) {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum for %s not found in SHA256SUMS", assetName)
	}

	// Hash the local file
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}
	actual := hex.EncodeToString(h.Sum(nil))

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	log.Printf("Updater: checksum verified OK for %s (%s)", assetName, actual[:12]+"...")
	return nil
}
```

Add `"crypto/sha256"` and `"encoding/hex"` to the import block.

### Step 4 — Call `VerifyChecksum` in the update apply handler

**File:** `coordinator/server/update.go`

Find where `DownloadBinary` is called and add a `VerifyChecksum` call after it:

```go
// AFTER DownloadBinary call, BEFORE VerifyBinary call:
if err := updater.VerifyChecksum(info.ChecksumURL, filepath.Base(tmpPath), tmpPath); err != nil {
    os.Remove(tmpPath)
    // Send error event via WebSocket
    s.hub.Broadcast(Event{Type: "update_error", Payload: map[string]string{"error": err.Error()}})
    return
}
```

---

## FIX 6 (LOW-1): Reduce JWT lifetime + add revocation on logout

**File:** `coordinator/server/auth.go`

### Step 1 — Reduce token lifetime

Find:

```go
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
```

Replace with:

```go
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(4 * time.Hour)),
```

Update `expires_in` in the login response:

```go
// Find:
"expires_in":            86400,
// Replace with:
"expires_in":            14400,
```

### Step 2 — Add jti claim for revocation

In `JWTClaims`, add a `JTI` field:

```go
type JWTClaims struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	MustChange  bool   `json:"must_change"`
	jwt.RegisteredClaims
}
```

Replace with:

```go
type JWTClaims struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	MustChange  bool   `json:"must_change"`
	jwt.RegisteredClaims
	// JWTID (jti) is set per token; used for revocation.
}
```

In `GenerateJWT`, add a `JWTID` to `RegisteredClaims`:

```go
// ADD to RegisteredClaims in GenerateJWT:
ID: generateJTI(),
```

Add the helper:

```go
func generateJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

### Step 3 — Add revoked_tokens table

**File:** `coordinator/db/db.go`

Add to the schema init SQL:

```sql
CREATE TABLE IF NOT EXISTS revoked_tokens (
    jti TEXT PRIMARY KEY,
    revoked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires ON revoked_tokens(expires_at);
```

Add methods:

```go
func (d *DB) RevokeToken(jti string, expiresAt time.Time) error {
    _, err := d.conn.Exec(
        `INSERT OR IGNORE INTO revoked_tokens (jti, expires_at) VALUES (?, ?)`,
        jti, expiresAt,
    )
    return err
}

func (d *DB) IsTokenRevoked(jti string) (bool, error) {
    var count int
    err := d.conn.QueryRow(
        `SELECT COUNT(*) FROM revoked_tokens WHERE jti = ? AND expires_at > datetime('now')`,
        jti,
    ).Scan(&count)
    return count > 0, err
}

// PruneExpiredTokens removes revoked tokens that have already expired (run periodically).
func (d *DB) PruneExpiredTokens() error {
    _, err := d.conn.Exec(`DELETE FROM revoked_tokens WHERE expires_at <= datetime('now')`)
    return err
}
```

### Step 4 — Check revocation in `JWTMiddleware`

In `JWTMiddleware`, after `ValidateJWT` succeeds, add:

```go
			// Check token revocation
			if claims.ID != "" {
				if revoked, err := s.db.IsTokenRevoked(claims.ID); err == nil && revoked {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "token has been revoked"})
					return
				}
			}
```

### Step 5 — Revoke on logout

In `handleLogout`, before the OK response:

```go
	claims := GetUserClaims(r)
	if claims != nil && claims.ID != "" {
		expiry := time.Now().Add(4 * time.Hour) // matches token lifetime
		if claims.ExpiresAt != nil {
			expiry = claims.ExpiresAt.Time
		}
		_ = s.db.RevokeToken(claims.ID, expiry)
	}
```

---

## FIX 7 (LOW-2): Panic recovery around executor

**File:** `agent/runner/runner.go`

Find the `process` function body. Wrap the executor call with a recover:

```go
	// 3. execute the job
	log.Printf("Runner: executing job %s (src=%q dst=%q)", job.ID, job.SourcePath, job.DestPath)
	exitCode, output := r.executor(job, Noop)
```

Replace with:

```go
	// 3. execute the job — wrapped in recover so deferred cleanup (credentials) always runs
	log.Printf("Runner: executing job %s (src=%q dst=%q)", job.ID, job.SourcePath, job.DestPath)
	exitCode := 1
	output := ""
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("Runner: panic executing job %s: %v", job.ID, rec)
				output = fmt.Sprintf("executor panic: %v", rec)
			}
		}()
		exitCode, output = r.executor(job, Noop)
	}()
```

---

## FIX 8 (LOW-3): Per-machine bootstrap tokens

**File:** `coordinator/server/bootstrap_handler.go`

### Step 1 — Accept optional hostname param

In `handleBootstrapScript`, after the admin-only guard, read an optional hostname:

```go
	// Optional: caller passes ?hostname=WORKSTATION01 to get a per-machine token.
	// Falls back to "bootstrap" role tag if not provided.
	hostnameHint := r.URL.Query().Get("hostname")
	tokenRole := "bootstrap"
	if hostnameHint != "" {
		tokenRole = "bootstrap:" + hostnameHint
	}
```

### Step 2 — Use `tokenRole` when minting

Find:

```go
	agentToken, err := s.db.CreateAgentToken("bootstrap")
```

Replace with:

```go
	agentToken, err := s.db.CreateAgentToken(tokenRole)
```

### Step 3 — Make bootstrap tokens short-lived (1 hour)

**File:** `coordinator/db/agents.go` (or wherever `CreateAgentToken` is implemented)

Find the token creation and add an `expires_at` column set to `NOW() + 1 hour`.
If the tokens table does not have an `expires_at` column, add it:

```sql
ALTER TABLE tokens ADD COLUMN expires_at DATETIME;
```

Update `ValidateToken` to reject expired tokens:

```go
// In ValidateToken, add to the WHERE clause:
// AND (expires_at IS NULL OR expires_at > datetime('now'))
```

---

## Verification steps after all fixes

Run these after implementing the changes above:

```bash
# 1. Build
go build ./coordinator/... ./agent/...

# 2. Tests
go test ./...

# 3. Confirm login rate limiting works
curl -s -o /dev/null -w "%{http_code}" -X POST https://localhost/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"x","password":"wrongpassword"}'
# Run 6+ times — should get 429 after the 5th attempt

# 4. Confirm CORS rejects unexpected origins
curl -s -o /dev/null -w "%{http_code}" https://localhost/api/jobs \
  -H "Origin: https://evil.example.com" \
  -H "Authorization: Bearer <valid_token>"
# Access-Control-Allow-Origin header should NOT be present in response

# 5. Confirm WS ?token= is rejected
# Open browser DevTools → Network → try connecting to /ws?token=<token>
# Should get 401

# 6. Confirm token revocation on logout
# 1. Login → get token
# 2. POST /api/auth/logout with that token
# 3. Try GET /api/auth/me with the same token — should get 401

# 7. govulncheck (install once)
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

---

## Files changed summary

| File | Fix |
|------|-----|
| `coordinator/config/config.go` | Add `AllowedOrigins` field |
| `coordinator/server/server.go` | Add rate limiter fields to struct; update CORS call |
| `coordinator/server/auth.go` | Rate limiting methods; jti claim; revocation check; logout revoke; reduce lifetime |
| `coordinator/server/hub.go` | Remove `?token=` query param fallback |
| `coordinator/db/db.go` | Add `revoked_tokens` table + methods |
| `coordinator/db/agents.go` | Add token expiry to bootstrap tokens |
| `coordinator/updater/updater.go` | Add `ChecksumURL` field + `VerifyChecksum` function |
| `coordinator/server/update.go` | Call `VerifyChecksum` before `VerifyBinary` |
| `agent/runner/runner.go` | Wrap executor in recover() |
| `coordinator/server/bootstrap_handler.go` | Per-machine token role + short-lived tokens |
| `go.mod` | Upgrade x/crypto; add x/time/rate |
| `config.json` | Add `allowed_origins` (manual, not committed) |
| `dashboard/src/` | Update WS connection to use subprotocol instead of ?token= |

**Do not commit `config.json`, `cert.pem`, or `key.pem`.**
