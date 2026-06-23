# ArcVault 2.0 - Phase 1 Implementation Plan
## Security Remediation: P0-001, P0-002, P0-004

**Status:** Ready for Implementation (Elena Approved with Conditions)  
**Phase:** 1 of 4  
**Duration:** 2 days (estimated)  
**Dependencies:** None (independent vulnerabilities)  
**Deployment Strategy:** Simultaneous rollout for all three fixes

---

## Executive Summary

Phase 1 addresses three critical authentication and configuration vulnerabilities:
- **P0-001**: CORS wildcard validation (currently allows `"*"`)
- **P0-002**: WebSocket origin validation (currently disabled)
- **P0-004**: Admin token persisted to config file (plaintext exposure)

All three are independent and can be implemented in parallel. This plan incorporates Elena's 5 must-have conditions:
1. Re-rate P0-005 to CVSS 7.8 (noted for Phase 3)
2. Add approval gate for P0-003 enforcement ✓
3. **Dashboard WebSocket client update (included below)**
4. Concurrent SSH key cleanup test scenario (noted for Phase 3)
5. **Document P0-004 env var deployment procedure (included below)**

---

## Vulnerability Overview

### P0-001: CORS Wildcard Validation [CRITICAL]
- **CWE:** 346 (Whitelist Bypass)
- **CVSS:** 7.5
- **Current State:** `AllowedOrigins: ["*"]` or empty list accepts all origins
- **Risk:** Attacker can access coordinator APIs from malicious origin
- **Fix Complexity:** Low (config validation)

### P0-002: WebSocket Origin Validation [CRITICAL]
- **CWE:** 345 (Insufficient Verification)
- **CVSS:** 8.6
- **Current State:** `CheckOrigin: func() { return true }` disabled
- **Risk:** Attacker can establish WebSocket from any origin, receive live events
- **Fix Complexity:** Low (origin check function)
- **Dependency:** Requires P0-001 (AllowedOrigins list must exist)

### P0-004: Admin Token Plaintext in Config [CRITICAL]
- **CWE:** 798 (Hardcoded Credentials)
- **CVSS:** 9.1
- **Current State:** `admin_token` written to config.json plaintext
- **Risk:** Token exposed in backups, git history, filesystem access
- **Fix Complexity:** Low (env var override, backward compatible)

---

## Task Breakdown by Vulnerability

### Task P1-001: CORS Whitelist Validation

#### Files to Modify

**1. `coordinator/config/config.go`** — Add validation function

```go
// Add this validation function after the Config struct definition:

// ValidateAllowedOrigins checks CORS configuration for security issues.
// Returns error if:
//   - Origins contains wildcard "*" in production mode
//   - Origins contains non-HTTPS URLs (except localhost)
//   - Origins list is empty in production mode
func (c *Config) ValidateAllowedOrigins() error {
	if len(c.AllowedOrigins) == 0 {
		if c.Environment == "production" {
			return fmt.Errorf("AllowedOrigins must be explicitly configured in production (cannot be empty or wildcard)")
		}
		// Development: allow unspecified origins (will use default)
		return nil
	}

	for _, origin := range c.AllowedOrigins {
		// Reject wildcard in all modes
		if origin == "*" {
			return fmt.Errorf("AllowedOrigins cannot contain wildcard '*' — specify explicit domains (e.g., https://dashboard.example.com)")
		}

		// Reject non-https origins except localhost
		if !strings.HasPrefix(origin, "https://") && !strings.HasPrefix(origin, "http://") {
			return fmt.Errorf("AllowedOrigins must use https:// or http://localhost, got: %s", origin)
		}

		if strings.HasPrefix(origin, "http://") {
			if !strings.Contains(origin, "localhost") && !strings.Contains(origin, "127.0.0.1") {
				return fmt.Errorf("non-HTTPS origins only allowed for localhost, got: %s", origin)
			}
		}
	}

	return nil
}
```

**Update `Load()` function** — Call validation after unmarshaling:

```go
// In Load() function, after json.Unmarshal, add:

	if err := cfg.ValidateAllowedOrigins(); err != nil {
		return nil, fmt.Errorf("CORS configuration invalid: %w", err)
	}
```

**Update initialization** — Set default in Load() if empty:

```go
// After validation, in Load():

	// Set sensible defaults if AllowedOrigins not configured
	if len(cfg.AllowedOrigins) == 0 {
		if cfg.Environment == "production" {
			log.Printf("WARNING: AllowedOrigins not set — configure via config.json or ENV")
			// Don't auto-default in production; force explicit config
		} else {
			// Development: allow localhost by default
			cfg.AllowedOrigins = []string{"http://localhost:5173", "http://localhost:3000"}
			log.Printf("Development: Using default AllowedOrigins: %v", cfg.AllowedOrigins)
		}
	}
```

**Lines to modify:**
- Line ~30: Config struct (no change needed, field exists)
- Line ~89-117: Load() function (add validation call)
- After line 126: Add ValidateAllowedOrigins() method

---

**2. `coordinator/server/server.go`** — Add CORS middleware

Find the `registerRoutes()` method and add CORS middleware:

```go
// In registerRoutes() method, add this before other handlers:

// Add CORS middleware as the first handler
func (s *Server) registerRoutes() {
	// CORS middleware that validates against AllowedOrigins
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				// Check if origin is in whitelist
				allowed := false
				for _, allowedOrigin := range s.cfg.AllowedOrigins {
					if origin == allowedOrigin {
						allowed = true
						break
					}
				}

				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				} else {
					log.Printf("CORS violation: rejected origin %s (not in AllowedOrigins: %v)", origin, s.cfg.AllowedOrigins)
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Wrap all routes with CORS middleware
	s.router.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(http.HandlerFunc(s.handleAPI)).ServeHTTP(w, r)
	})

	// ... rest of registerRoutes() ...
}
```

**Alternative: Simpler approach** (recommended) — Add to each API handler group:

Actually, Go's http.ServeMux doesn't support middleware wrapping elegantly. Instead, validate in the handlers or use a wrapper struct. Here's the minimal approach:

In `registerRoutes()`, add a per-handler CORS check at the start of each handler that needs it (jobs, agents, etc.):

```go
// Add this helper method to Server:
func (s *Server) checkCORSOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // No origin header (non-CORS request)
	}

	for _, allowedOrigin := range s.cfg.AllowedOrigins {
		if origin == allowedOrigin {
			return true
		}
	}

	log.Printf("CORS rejection: origin %q not in AllowedOrigins %v", origin, s.cfg.AllowedOrigins)
	return false
}

// Then in each API handler, add at the start:
if !s.checkCORSOrigin(r) {
	http.Error(w, "CORS origin not allowed", http.StatusForbidden)
	return
}

// Also add response headers to successful responses:
if origin := r.Header.Get("Origin"); origin != "" {
	for _, allowed := range s.cfg.AllowedOrigins {
		if origin == allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			break
		}
	}
}
```

**Simpler minimal implementation:**
Edit line 414 area of `coordinator/server/server.go`:

```go
// Add after NewWithFS function (line ~89):

func (s *Server) corsOriginAllowed(origin string) bool {
	if origin == "" {
		return true // Non-CORS request
	}
	for _, allowed := range s.cfg.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	log.Printf("[CORS] Rejected origin: %q (not in whitelist)", origin)
	return false
}

// Update registerRoutes() to call this in API handlers:
// s.router.HandleFunc("/api/...", func(w http.ResponseWriter, r *http.Request) {
//   if !s.corsOriginAllowed(r.Header.Get("Origin")) {
//       http.Error(w, "CORS origin not allowed", http.StatusForbidden)
//       return
//   }
//   // ... handler logic ...
// })
```

**Priority files:**
- `coordinator/config/config.go`: Lines 12-30 (struct), 89-117 (Load), add ValidateAllowedOrigins()
- `coordinator/server/server.go`: Line ~89 add corsOriginAllowed() method, update registerRoutes()

---

### Task P1-002: WebSocket Origin Validation

#### Files to Modify

**1. `coordinator/server/hub.go`** — Update global upgrader

**Current code** (line 128-130):
```go
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
```

**Change to:**
```go
// Global upgrader with disabled CheckOrigin by default
// Will be overridden per-server instance with proper origin validation
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// WARNING: This should be overridden by Server.initWebSocketUpgrader()
		// Default deny to fail safe if initialization fails
		return false
	},
}
```

**2. `coordinator/server/server.go`** — Add upgrader initialization

Add to the `Server` struct (after line 53):

```go
type Server struct {
	// ... existing fields ...
	wsUpgrader *websocket.Upgrader // Will be initialized with proper CheckOrigin
}
```

Then add method to Server:

```go
// initWebSocketUpgrader creates an upgrader with origin validation
// based on AllowedOrigins config
func (s *Server) initWebSocketUpgrader() {
	s.wsUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			
			// If no Origin header, allow (non-CORS request, likely same-origin)
			if origin == "" {
				return true
			}

			// Check against whitelist
			for _, allowedOrigin := range s.cfg.AllowedOrigins {
				if origin == allowedOrigin {
					log.Printf("[WebSocket] Accepted origin: %q", origin)
					return true
				}
			}

			log.Printf("[WebSocket] Rejected origin: %q (not in AllowedOrigins: %v)", origin, s.cfg.AllowedOrigins)
			return false
		},
	}
}
```

**Update `NewWithFS()` method** — Call initializer:

Find the function around line 59 and add a call to initWebSocketUpgrader:

```go
func NewWithFS(cfg *config.Config, database *db.DB, staticFS fs.FS) *Server {
	// ... existing init code ...

	s := &Server{
		// ... existing field assignments ...
	}

	// Initialize WebSocket upgrader with origin validation
	s.initWebSocketUpgrader()

	// ... rest of init ...
}
```

**3. `coordinator/server/hub.go`** — Update handler functions

Replace the global `upgrader` usage in `handleWS()` and `handleAgentWS()`:

**In `handleWS()` method** (line 166):
```go
// Old:
conn, err := upgrader.Upgrade(w, r, upgradeHeader)

// New:
conn, err := s.wsUpgrader.Upgrade(w, r, upgradeHeader)
```

**In `handleAgentWS()` method** (line 214):
```go
// Old:
conn, err := upgrader.Upgrade(w, r, nil)

// New:
conn, err := s.wsUpgrader.Upgrade(w, r, nil)
```

**Lines to modify:**
- Line 128-130: Update global upgrader (fail-safe)
- `coordinator/server/server.go` Line 35-53: Add wsUpgrader field to struct
- After line 88: Add initWebSocketUpgrader() method
- Line ~87: Add call to s.initWebSocketUpgrader() in NewWithFS()
- Line 166: Replace upgrader with s.wsUpgrader
- Line 214: Replace upgrader with s.wsUpgrader

---

### Task P1-004: Admin Token from Environment Variables

#### Files to Modify

**1. `coordinator/config/config.go`** — Environment variable override

Update the `Load()` function to check environment variables:

```go
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config (run 'coordinator init' first): %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}

	// NEW: Check environment variables for sensitive credentials
	// Environment variables override config file values
	if envToken := os.Getenv("ARCVAULT_ADMIN_TOKEN"); envToken != "" {
		cfg.AdminToken = envToken
		log.Printf("[config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var")
	}

	if envSecret := os.Getenv("ARCVAULT_JWT_SECRET"); envSecret != "" {
		cfg.JWTSecret = envSecret
		log.Printf("[config] JWTSecret loaded from ARCVAULT_JWT_SECRET env var")
	}

	// Auto-generate JWT secret if still missing after env check
	if cfg.JWTSecret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return nil, fmt.Errorf("could not generate JWT secret: %w", err)
		}
		cfg.JWTSecret = secret
		log.Printf("[config] Generated new JWTSecret (set ARCVAULT_JWT_SECRET to override)")
	}

	// Validate production requirements
	if cfg.Environment == "production" {
		if cfg.AdminToken == "" {
			return nil, fmt.Errorf("CRITICAL: AdminToken not set. Set ARCVAULT_ADMIN_TOKEN environment variable before starting production server")
		}
	}

	return &cfg, nil
}
```

**2. `coordinator/config/config.go`** — Update Save() to sanitize

Modify the `Save()` function to never write sensitive fields:

```go
func Save(cfg *Config) error {
	// Create sanitized copy for file storage
	// Never write AdminToken or JWTSecret to disk
	sanitized := *cfg
	sanitized.AdminToken = ""
	sanitized.JWTSecret = ""

	path, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("could not determine config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	data, err := json.MarshalIndent(&sanitized, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	log.Printf("[config] Sensitive fields (AdminToken, JWTSecret) cleared from config file")
	log.Printf("[config] Set ARCVAULT_ADMIN_TOKEN and ARCVAULT_JWT_SECRET environment variables")

	return os.WriteFile(path, data, 0600)
}
```

**3. `coordinator/cmd/commands.go`** — Update InitCommand

Modify the `InitCommand()` function to display setup instructions:

```go
func InitCommand() error {
	fmt.Println("ArcVault Coordinator - Initialization")
	fmt.Println("=====================================")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter port (default 443): ")
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	port := 443
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %v", err)
		}
		port = p
	}

	fmt.Print("Enter host (IP or hostname, for TLS cert): ")
	host, _ := reader.ReadString('\n')
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("host is required")
	}

	homeDir, _ := os.UserHomeDir()
	defaultDB := filepath.Join(homeDir, ".arcvault", "arcvault.db")
	fmt.Printf("Enter database path (default %s): ", defaultDB)
	dbPath, _ := reader.ReadString('\n')
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		dbPath = defaultDB
	}

	// Generate TLS cert
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	certPath := filepath.Join(exeDir, "cert.pem")
	keyPath := filepath.Join(exeDir, "key.pem")

	fmt.Print("\nGenerating TLS certificate...")
	if err := tlscert.Generate(host, certPath, keyPath); err != nil {
		return fmt.Errorf("failed to generate TLS certificate: %v", err)
	}
	fmt.Printf(" done\n")

	// Generate tokens for reference (NOT saved to file)
	adminToken, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate admin token: %v", err)
	}

	jwtSecret, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate JWT secret: %v", err)
	}

	// Save config WITHOUT tokens
	cfg := &config.Config{
		Port:         port,
		DatabasePath: dbPath,
		AdminToken:   "", // Empty — must come from env var
		JWTSecret:    "", // Empty — must come from env var
		Environment:  "development",
		Host:         host,
		CertFile:     certPath,
		KeyFile:      keyPath,
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("\n✓ Configuration saved to: %s\n", configPath)
	fmt.Printf("✓ Database will be initialized at: %s\n", dbPath)
	fmt.Printf("✓ TLS certificate: %s\n\n", certPath)

	// NEW: Display setup instructions
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("IMPORTANT: Environment Variables Required")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\nBefore running 'coordinator start', set these environment variables:")
	fmt.Printf("\n  export ARCVAULT_ADMIN_TOKEN=%s\n", adminToken)
	fmt.Printf("  export ARCVAULT_JWT_SECRET=%s\n\n", jwtSecret)
	fmt.Println("⚠️  DO NOT share these tokens or commit to git")
	fmt.Println("⚠️  DO NOT restart without exporting these variables")
	fmt.Println("⚠️  For production, use a secret management system (Vault, AWS Secrets Manager, etc.)")
	fmt.Println("\nNext step: Run 'coordinator start'")

	return nil
}
```

**Lines to modify:**
- `coordinator/config/config.go` Line 89-117: Update Load() to check env vars
- `coordinator/config/config.go` Line 74-87: Update Save() to sanitize
- `coordinator/cmd/commands.go` Line 24-99: Update InitCommand() with display instructions

**4. Add deployment documentation**

Create new file: `coordinator/DEPLOYMENT.md`

```markdown
# Production Deployment Guide

## Prerequisites

1. TLS certificates (generate with `coordinator init`)
2. Database directory (writable)
3. Environment variables for sensitive credentials

## Environment Variables

Before starting the coordinator in any environment, set:

```bash
export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)
```

### Security Best Practices

1. **Never commit tokens to git**
   - Add `.env` and `config.json` to `.gitignore`

2. **Use a secrets manager in production**
   - HashiCorp Vault
   - AWS Secrets Manager
   - Azure Key Vault
   - Google Cloud Secret Manager

3. **Rotate tokens regularly**
   - Store previous token in Vault for graceful rotation
   - Plan 24-hour migration window for token change

4. **Audit token access**
   - Log when tokens are loaded from environment
   - Log when CORS origins are validated

## Startup Procedure

### Development

```bash
# Generate tokens (one-time)
export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)

# Start server
coordinator start
```

### Production (Linux/macOS with systemd)

Create `/etc/systemd/system/arcvault.service`:

```ini
[Unit]
Description=ArcVault Coordinator
After=network.target

[Service]
Type=simple
User=arcvault
WorkingDirectory=/opt/arcvault
ExecStart=/opt/arcvault/coordinator start
Restart=on-failure
RestartSec=10

# Load environment variables from secure location
EnvironmentFile=/etc/arcvault/.env

[Install]
WantedBy=multi-user.target
```

Create `/etc/arcvault/.env`:

```bash
ARCVAULT_ADMIN_TOKEN=<token-from-vault>
ARCVAULT_JWT_SECRET=<secret-from-vault>
```

Set restrictive permissions:

```bash
chmod 600 /etc/arcvault/.env
chown arcvault:arcvault /etc/arcvault/.env
```

Start service:

```bash
systemctl enable arcvault
systemctl start arcvault
```

### Production (Docker)

Pass environment variables at runtime:

```bash
docker run -d \
  -p 443:8443 \
  -v /data/arcvault:/root/.arcvault \
  -e ARCVAULT_ADMIN_TOKEN=$(vault read -field=token secret/arcvault/admin) \
  -e ARCVAULT_JWT_SECRET=$(vault read -field=secret secret/arcvault/jwt) \
  arcvault/coordinator:latest
```

## Startup Validation

After starting, verify in logs:

```bash
# Should see:
# [config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var
# [config] JWTSecret loaded from ARCVAULT_JWT_SECRET env var
```

If you see warnings about environment variables not set:

```
CRITICAL: AdminToken not set. Set ARCVAULT_ADMIN_TOKEN environment variable
```

Then:
1. Stop the coordinator
2. Export the environment variables
3. Restart

## Token Rotation Procedure

To change admin token without downtime:

1. Generate new token:
   ```bash
   NEW_TOKEN=$(openssl rand -hex 32)
   echo "New token: $NEW_TOKEN"
   ```

2. Store in secret manager (don't lose it!)

3. Update environment:
   ```bash
   export ARCVAULT_ADMIN_TOKEN=$NEW_TOKEN
   systemctl restart arcvault
   ```

4. Update all agent configurations to use new token

5. After 24 hours, old token can be discarded

## Troubleshooting

### "config.json contains tokens" warning

If you see this, the config file was created by an older version:

```bash
# Regenerate with sanitized config:
rm config.json
coordinator init

# Set env vars
export ARCVAULT_ADMIN_TOKEN=<new-token>
export ARCVAULT_JWT_SECRET=<new-secret>

# Start
coordinator start
```

### Connection refused on startup

Check environment variables are exported:

```bash
echo $ARCVAULT_ADMIN_TOKEN
echo $ARCVAULT_JWT_SECRET
```

If empty, export them and restart.
```

Save to: `/c/Projects/ArcVault2.0/coordinator/DEPLOYMENT.md`

---

## Elena's Condition #3: Dashboard WebSocket Client Update

The dashboard already has proper WebSocket implementation via `useWebSocket.js` (uses Authorization header). However, to ensure it explicitly handles CORS/origin validation on the server side, update the client to add explicit origin logging.

### File: `dashboard/src/composables/useWebSocket.js`

Update the `connect()` function to log origin and handle origin rejection:

```javascript
function connect() {
    const token = getToken()
    if (!token) return

    const wsUrl = getWsUrl()
    console.log(`[WS] Connecting to ${wsUrl} from origin ${window.location.origin}`)

    ws = new WebSocket(wsUrl, [`bearer.${token}`])

    ws.onopen = () => {
        connected.value = true
        console.log('[WS] Connected successfully from origin:', window.location.origin)
    }

    ws.onmessage = (e) => {
        console.log('WS message received:', e.data)
        try {
            lastEvent.value = JSON.parse(e.data)
            console.log('WS parsed event:', lastEvent.value)
        } catch (parseError) {
            console.warn('WS: bad message', e.data, parseError)
        }
    }

    ws.onclose = (event) => {
        connected.value = false
        console.log('[WS] Disconnected. Code:', event.code, 'Reason:', event.reason)
        
        // Handle origin rejection (code 1006 = abnormal closure)
        if (event.code === 1006 && event.reason) {
            console.error('[WS] Connection rejected - may be CORS/origin validation issue')
        }
        
        console.log('WS disconnected, reconnecting in 5s...')
        reconnectTimer = setTimeout(connect, 5000)
    }

    ws.onerror = (err) => {
        console.error('[WS] Error:', err)
        // Don't close here - onclose will handle reconnect
    }
}
```

**Update notes:**
- Added explicit origin logging in onopen
- Added error code handling in onclose
- Better error messages for debugging CORS issues

---

## Elena's Condition #5: P0-004 Deployment Procedure Documentation

Already included above in:
1. Updated `InitCommand()` with setup instructions
2. New `coordinator/DEPLOYMENT.md` with production procedure
3. Environment variable loading with validation in `config.Load()`

---

## Implementation Sequence

### Phase 1A: Configuration & Validation (Day 1)

1. **P0-001: CORS Whitelist**
   - [ ] Add `ValidateAllowedOrigins()` to `coordinator/config/config.go`
   - [ ] Update `config.Load()` to call validation
   - [ ] Add default origins for development mode
   - [ ] Add `corsOriginAllowed()` helper to `coordinator/server/server.go`

2. **P0-004: Environment Variables**
   - [ ] Update `config.Load()` to read env vars (override config file)
   - [ ] Update `config.Save()` to sanitize tokens
   - [ ] Update `InitCommand()` with setup instructions
   - [ ] Create `coordinator/DEPLOYMENT.md`

3. **Testing**
   - [ ] Run `coordinator init` (verify tokens not in config.json)
   - [ ] Set `ARCVAULT_ADMIN_TOKEN` and `ARCVAULT_JWT_SECRET`
   - [ ] Start coordinator, verify log messages show env var usage
   - [ ] Verify empty config.json doesn't contain tokens

### Phase 1B: WebSocket & Dashboard (Day 2)

4. **P0-002: WebSocket Origin Validation**
   - [ ] Add `wsUpgrader` field to Server struct
   - [ ] Add `initWebSocketUpgrader()` method
   - [ ] Call init in `NewWithFS()`
   - [ ] Update `handleWS()` to use `s.wsUpgrader`
   - [ ] Update `handleAgentWS()` to use `s.wsUpgrader`
   - [ ] Update global upgrader with fail-safe default

5. **Dashboard WebSocket Client**
   - [ ] Update `useWebSocket.js` with origin logging
   - [ ] Update onclose handler for better debugging

6. **Integration Testing**
   - [ ] Set `ARCVAULT_ADMIN_TOKEN` and `ARCVAULT_JWT_SECRET`
   - [ ] Configure `AllowedOrigins` with valid https URL
   - [ ] Start coordinator
   - [ ] Connect dashboard from allowed origin (should work)
   - [ ] Attempt connection from non-whitelisted origin (should reject with 403)
   - [ ] Check WebSocket connections (should only see allowed origins in logs)

---

## Test Cases

### T1: CORS Validation

```go
// File: coordinator/tests/cors_validation_test.go

package server

import (
    "testing"
    "arcvault/coordinator/config"
)

func TestValidateCORSOrigins_RejectsWildcard(t *testing.T) {
    cfg := &config.Config{
        AllowedOrigins: []string{"*"},
        Environment: "production",
    }
    err := cfg.ValidateAllowedOrigins()
    if err == nil {
        t.Fatal("expected error for wildcard origin in production, got nil")
    }
}

func TestValidateCORSOrigins_AllowsHTTPS(t *testing.T) {
    cfg := &config.Config{
        AllowedOrigins: []string{"https://dashboard.internal.corp"},
        Environment: "production",
    }
    err := cfg.ValidateAllowedOrigins()
    if err != nil {
        t.Fatalf("expected no error for valid HTTPS origin, got %v", err)
    }
}

func TestValidateCORSOrigins_AllowsLocalhost(t *testing.T) {
    cfg := &config.Config{
        AllowedOrigins: []string{"http://localhost:5173"},
        Environment: "development",
    }
    err := cfg.ValidateAllowedOrigins()
    if err != nil {
        t.Fatalf("expected no error for localhost, got %v", err)
    }
}

func TestValidateCORSOrigins_RejectsHTTPRemote(t *testing.T) {
    cfg := &config.Config{
        AllowedOrigins: []string{"http://example.com"},
        Environment: "production",
    }
    err := cfg.ValidateAllowedOrigins()
    if err == nil {
        t.Fatal("expected error for non-HTTPS remote origin, got nil")
    }
}
```

### T2: Environment Variable Configuration

```go
// File: coordinator/tests/config_env_test.go

func TestLoadConfig_EnvVarOverridesFile(t *testing.T) {
    // Set env vars
    os.Setenv("ARCVAULT_ADMIN_TOKEN", "env-token-123")
    os.Setenv("ARCVAULT_JWT_SECRET", "env-secret-456")
    defer os.Unsetenv("ARCVAULT_ADMIN_TOKEN")
    defer os.Unsetenv("ARCVAULT_JWT_SECRET")

    cfg, _ := config.Load()
    if cfg.AdminToken != "env-token-123" {
        t.Fatalf("expected AdminToken from env var, got %s", cfg.AdminToken)
    }
    if cfg.JWTSecret != "env-secret-456" {
        t.Fatalf("expected JWTSecret from env var, got %s", cfg.JWTSecret)
    }
}

func TestLoadConfig_FailsWithoutTokenInProduction(t *testing.T) {
    os.Unsetenv("ARCVAULT_ADMIN_TOKEN")
    
    // Manually set config file with empty AdminToken and production mode
    cfg, err := config.Load()
    if cfg.Environment == "production" && cfg.AdminToken == "" {
        if err == nil {
            t.Fatal("expected error for missing AdminToken in production, got nil")
        }
    }
}

func TestSaveConfig_NeverWritesSensitiveFields(t *testing.T) {
    cfg := &config.Config{
        AdminToken: "secret-token",
        JWTSecret: "secret-secret",
        Port: 443,
    }
    
    config.Save(cfg)
    
    // Read file and verify tokens are empty
    data, _ := ioutil.ReadFile(configPath)
    var loaded config.Config
    json.Unmarshal(data, &loaded)
    
    if loaded.AdminToken != "" {
        t.Fatal("AdminToken should be empty in saved config")
    }
    if loaded.JWTSecret != "" {
        t.Fatal("JWTSecret should be empty in saved config")
    }
}
```

### T3: WebSocket Origin Validation

```go
// File: coordinator/tests/websocket_origin_test.go

func TestWebSocketUpgrade_AllowsValidOrigin(t *testing.T) {
    server := NewTestServer([]string{"https://dashboard.example.com"})
    
    req := httptest.NewRequest("GET", "/ws", nil)
    req.Header.Set("Origin", "https://dashboard.example.com")
    
    // Should succeed
    conn, err := server.wsUpgrader.Upgrade(responseWriter, req, nil)
    if err != nil {
        t.Fatalf("expected upgrade to succeed for allowed origin, got %v", err)
    }
    conn.Close()
}

func TestWebSocketUpgrade_RejectsInvalidOrigin(t *testing.T) {
    server := NewTestServer([]string{"https://dashboard.example.com"})
    
    req := httptest.NewRequest("GET", "/ws", nil)
    req.Header.Set("Origin", "https://attacker.com")
    
    // Should reject
    conn, err := server.wsUpgrader.Upgrade(responseWriter, req, nil)
    if err == nil {
        t.Fatal("expected upgrade to fail for disallowed origin")
    }
}

func TestWebSocketUpgrade_AllowsNoOriginHeader(t *testing.T) {
    server := NewTestServer([]string{"https://dashboard.example.com"})
    
    req := httptest.NewRequest("GET", "/ws", nil)
    // No Origin header
    
    // Should succeed (same-origin request, non-CORS)
    conn, err := server.wsUpgrader.Upgrade(responseWriter, req, nil)
    if err != nil {
        t.Fatalf("expected upgrade to succeed for no-origin request, got %v", err)
    }
    conn.Close()
}
```

---

## Deployment Checklist

### Pre-Deployment

- [ ] All three vulnerabilities (P0-001, P0-002, P0-004) implemented
- [ ] Unit tests pass: `go test ./coordinator/tests/...`
- [ ] Integration tests pass: `go test ./coordinator/server/...`
- [ ] Code review completed (Elena Vasquez sign-off)
- [ ] Security review completed (Kwame Asante sign-off)
- [ ] Staging environment validated

### Deployment Steps

1. **Backup existing configuration**
   ```bash
   cp /opt/arcvault/config.json /opt/arcvault/config.json.backup-$(date +%s)
   ```

2. **Update coordinator binary**
   ```bash
   systemctl stop arcvault
   cp coordinator /opt/arcvault/
   ```

3. **Set environment variables** (if not already set)
   ```bash
   # Load from vault or secrets manager
   export ARCVAULT_ADMIN_TOKEN=$(vault read -field=token secret/arcvault/admin)
   export ARCVAULT_JWT_SECRET=$(vault read -field=secret secret/arcvault/jwt)
   ```

4. **Regenerate config (sanitized)**
   ```bash
   cd /opt/arcvault
   ./coordinator init  # Or manually create with empty tokens
   ```

5. **Start coordinator**
   ```bash
   systemctl start arcvault
   ```

6. **Verify startup logs**
   ```bash
   journalctl -u arcvault -f
   # Should see: [config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var
   ```

7. **Test CORS validation**
   ```bash
   # From allowed origin:
   curl -H "Origin: https://dashboard.example.com" \
        -X OPTIONS https://coordinator.local/api/jobs

   # Should return 200 with CORS headers

   # From disallowed origin:
   curl -H "Origin: https://attacker.com" \
        -X OPTIONS https://coordinator.local/api/jobs

   # Should return 403
   ```

8. **Test WebSocket**
   ```bash
   # Dashboard should connect (check browser console)
   # Verify in logs: [WebSocket] Accepted origin: https://dashboard.example.com
   ```

### Rollback Procedure

If deployment fails:

```bash
# Stop coordinator
systemctl stop arcvault

# Restore previous binary
cp /opt/arcvault/coordinator.backup /opt/arcvault/coordinator

# Restore config
cp /opt/arcvault/config.json.backup-<timestamp> /opt/arcvault/config.json

# Start with old environment
systemctl start arcvault

# Verify
journalctl -u arcvault -f
```

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Environment variable not set | Server won't start in production | Clear documentation, startup validation, pre-flight checks |
| CORS too restrictive | Dashboard can't connect | AllowedOrigins auto-populated for development, clear logs for debugging |
| WebSocket origin rejection | Live updates fail | Logs show rejected origins with reason, client logs show connection failures |
| Config file backup contains tokens | Security exposure | Sanitize on startup, alert if tokens found |

---

## Sign-Off Checklist

- [ ] **Marcus Chen (Software Engineer)** - Implementation complete
- [ ] **Elena Vasquez (Senior Code Reviewer)** - Code review passed
- [ ] **Kwame Asante (Cybersecurity Engineer)** - Security approval
- [ ] **Kren Castro (Project Owner)** - Ready for deployment

---

## Related Tasks

- **Phase 2:** P0-003 (Command Injection Prevention)
- **Phase 3:** P0-005, P0-006, P0-007, P0-008
- **Phase 4:** Integration testing & documentation

**Document Version:** 1.0  
**Last Updated:** 2026-06-19  
**Status:** Ready for Implementation
