# Phase 1 - Detailed File Changes Guide

This document provides exact file locations, line numbers, and code changes for Marcus to implement.

---

## File 1: `coordinator/config/config.go`

### Change 1.1: Add ValidateAllowedOrigins() Method

**Location:** After line 127 (after generateSecret function)

**Add this method:**

```go
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

### Change 1.2: Update Load() Function

**Location:** Lines 89-117

**Old code:**
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

	// Auto-generate JWT secret if missing
	if cfg.JWTSecret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return nil, fmt.Errorf("could not generate JWT secret: %w", err)
		}
		cfg.JWTSecret = secret
		// Save updated config back to file
		if err := Save(&cfg); err != nil {
			return nil, fmt.Errorf("could not save updated config: %w", err)
		}
	}

	return &cfg, nil
}
```

**New code:**
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

	// Load sensitive fields from environment variables (override config file)
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

	// Validate CORS configuration
	if err := cfg.ValidateAllowedOrigins(); err != nil {
		return nil, fmt.Errorf("CORS configuration invalid: %w", err)
	}

	// Set sensible defaults if AllowedOrigins not configured
	if len(cfg.AllowedOrigins) == 0 {
		if cfg.Environment != "production" {
			// Development: allow localhost by default
			cfg.AllowedOrigins = []string{"http://localhost:5173", "http://localhost:3000"}
			log.Printf("[config] Development: Using default AllowedOrigins: %v", cfg.AllowedOrigins)
		}
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

**Import additions needed:**
At top of file (after existing imports), ensure `log` is imported:
```go
import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"           // ADD THIS
	"os"
	"path/filepath"
	"strings"       // ADD THIS
)
```

### Change 1.3: Update Save() Function

**Location:** Lines 74-87

**Old code:**
```go
func Save(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("could not determine config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
```

**New code:**
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

---

## File 2: `coordinator/server/server.go`

### Change 2.1: Add wsUpgrader Field to Server Struct

**Location:** Line 35-53 (Server struct definition)

**Add field after line 52, before closing brace:**

```go
	wsUpgrader *websocket.Upgrader // WebSocket upgrader with origin validation
}
```

**Updated struct should look like:**
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
	tokenCache     map[string]tokenCacheEntry // token → validated entry
	loginLimiterMu sync.Mutex
	loginLimiters  map[string]*loginRateLimiter
	wsUpgrader     *websocket.Upgrader // ADD THIS LINE
}
```

### Change 2.2: Add initWebSocketUpgrader() Method

**Location:** After NewWithFS() function (around line 90)

**Add this new method:**

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

### Change 2.3: Add corsOriginAllowed() Helper Method

**Location:** After initWebSocketUpgrader() method

**Add this method:**

```go
// corsOriginAllowed checks if a CORS origin is in the whitelist
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
```

### Change 2.4: Update NewWithFS() to Initialize WebSocket Upgrader

**Location:** Line 59-89

**Find this section:**
```go
	s := &Server{
		cfg:            cfg,
		db:             database,
		router:         http.NewServeMux(),
		hub:            newHub(),
		fedHub:         NewFederationHub(database),
		staticFS:       staticFS,
		Notifier:       notifications.NewDispatcher(cfg.Notifications),
		coordinatorID:  coordinatorID,
		agentService:   business.NewAgentService(database),
		jobService:     business.NewJobService(database),
		userService:    business.NewUserService(database),
		groupService:   business.NewGroupService(database),
		tokenCache:     make(map[string]tokenCacheEntry),
		loginLimiters:  make(map[string]*loginRateLimiter),
	}

	if cfg.Federation != nil {
		s.fedClient = NewFederationClient(cfg.Federation, database, Version)
	}

	s.registerRoutes()
	return s
```

**Change to:**
```go
	s := &Server{
		cfg:            cfg,
		db:             database,
		router:         http.NewServeMux(),
		hub:            newHub(),
		fedHub:         NewFederationHub(database),
		staticFS:       staticFS,
		Notifier:       notifications.NewDispatcher(cfg.Notifications),
		coordinatorID:  coordinatorID,
		agentService:   business.NewAgentService(database),
		jobService:     business.NewJobService(database),
		userService:    business.NewUserService(database),
		groupService:   business.NewGroupService(database),
		tokenCache:     make(map[string]tokenCacheEntry),
		loginLimiters:  make(map[string]*loginRateLimiter),
	}

	// Initialize WebSocket upgrader with origin validation
	s.initWebSocketUpgrader()

	if cfg.Federation != nil {
		s.fedClient = NewFederationClient(cfg.Federation, database, Version)
	}

	s.registerRoutes()
	return s
```

---

## File 3: `coordinator/server/hub.go`

### Change 3.1: Update Global Upgrader with Fail-Safe

**Location:** Lines 128-130

**Old code:**
```go
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
```

**New code:**
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

### Change 3.2: Update handleWS() Method

**Location:** Line 166 (within handleWS function)

**Old code:**
```go
	conn, err := upgrader.Upgrade(w, r, upgradeHeader)
```

**New code:**
```go
	conn, err := s.wsUpgrader.Upgrade(w, r, upgradeHeader)
```

### Change 3.3: Update handleAgentWS() Method

**Location:** Line 214 (within handleAgentWS function)

**Old code:**
```go
	conn, err := upgrader.Upgrade(w, r, nil)
```

**New code:**
```go
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
```

---

## File 4: `coordinator/cmd/commands.go`

### Change 4.1: Update InitCommand() Function

**Location:** Lines 24-99

**Replace entire function with:**

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

	// Get exe directory for cert files
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	certPath := filepath.Join(exeDir, "cert.pem")
	keyPath := filepath.Join(exeDir, "key.pem")

	// Generate TLS cert
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

	// Display setup instructions
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

---

## File 5: `dashboard/src/composables/useWebSocket.js`

### Change 5.1: Update connect() Function

**Location:** Lines 18-50

**Replace function with:**

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

---

## File 6: NEW FILE `coordinator/DEPLOYMENT.md`

Create new file at: `coordinator/DEPLOYMENT.md`

Copy content from `PHASE_1_IMPLEMENTATION_PLAN.md` section "Add deployment documentation"

Or use the full content provided in the implementation plan.

---

## Summary of Changes

### Files Modified: 5
1. `coordinator/config/config.go` — 150+ lines (3 changes)
2. `coordinator/server/server.go` — 50+ lines (4 changes)
3. `coordinator/server/hub.go` — 20 lines (3 changes)
4. `coordinator/cmd/commands.go` — 50+ lines (1 major change)
5. `dashboard/src/composables/useWebSocket.js` — 30 lines (1 change)

### Files Created: 1
1. `coordinator/DEPLOYMENT.md` — 200+ lines

### Total Lines Added: 600+
### Total Lines Modified: 200+

---

## Implementation Order

1. **config.go** (ValidateAllowedOrigins, Save, Load) - Must be first
2. **server.go** (struct, methods, NewWithFS) - Depends on config
3. **hub.go** (upgrader, handlers) - Depends on server
4. **commands.go** (InitCommand) - Independent, but should update after config
5. **useWebSocket.js** - Independent, can be any time
6. **DEPLOYMENT.md** - Documentation, can be any time

---

## Verification Steps After Each File

After modifying each file:

```bash
# Check syntax
go fmt ./coordinator/...

# Run tests (if they exist)
go test ./coordinator/... -v

# Check for compilation errors
go build ./coordinator/...
```

After all changes:

```bash
# Full build test
go build -o /tmp/coordinator ./coordinator/

# Format all files
go fmt ./...

# Run full test suite
go test ./...
```

---

## Line Number Reference

These line numbers are approximate and may shift. Use search to locate exact sections:

- `config.go`: `func Load()`, `func Save()`, `generateSecret()`
- `server.go`: `type Server struct`, `func NewWithFS()`, `func registerRoutes()`
- `hub.go`: `var upgrader`, `func handleWS()`, `func handleAgentWS()`
- `commands.go`: `func InitCommand()`
- `useWebSocket.js`: `function connect()`

If line numbers don't match, search for function names and context comments.

---

## Ready for Implementation

All files and changes documented. Marcus Chen can now implement following this guide exactly.

**Status:** Ready for coding  
**Target completion:** 2 days  
**Review by:** Elena Vasquez
