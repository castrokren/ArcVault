package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"arcvault/coordinator/business"
	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
	"arcvault/coordinator/notifications"
)

// Version is injected at build time via -ldflags.
var Version = "dev"

// tokenCacheEntry holds a validated agent token with an expiry.
type tokenCacheEntry struct {
	agentID   string
	expiresAt time.Time
}

// loginRateLimiter tracks failed attempts per IP.
type loginRateLimiter struct {
	limiter  *rate.Limiter
	failures int
	lockedAt *time.Time
}

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
	auditService   *business.AuditService
	tokenCacheMu   sync.Mutex
	tokenCache     map[string]tokenCacheEntry // token → validated entry
	loginLimiterMu sync.Mutex
	loginLimiters  map[string]*loginRateLimiter
	wsUpgrader     *websocket.Upgrader // WebSocket upgrader with origin validation
}

func New(cfg *config.Config, database *db.DB) *Server {
	return NewWithFS(cfg, database, nil)
}

func NewWithFS(cfg *config.Config, database *db.DB, staticFS fs.FS) *Server {
	// Set coordinator ID from config or use a default/hostname-based ID.
	coordinatorID := cfg.CoordinatorID
	if coordinatorID == "" {
		coordinatorID = "root"
	}

	s := &Server{
		cfg:           cfg,
		db:            database,
		router:        http.NewServeMux(),
		hub:           newHub(),
		fedHub:        NewFederationHub(database),
		staticFS:      staticFS,
		Notifier:      notifications.NewDispatcher(cfg.Notifications),
		coordinatorID: coordinatorID,
		agentService:  business.NewAgentService(database),
		jobService:    business.NewJobService(database),
		userService:   business.NewUserService(database),
		groupService:  business.NewGroupService(database),
		auditService:  business.NewAuditService(database),
		tokenCache:    make(map[string]tokenCacheEntry),
		loginLimiters: make(map[string]*loginRateLimiter),
	}

	// Initialize WebSocket upgrader with origin validation
	s.initWebSocketUpgrader()

	if cfg.Federation != nil {
		s.fedClient = NewFederationClient(cfg.Federation, database, Version)
	}

	s.registerRoutes()
	return s
}

func NewWithStatic(cfg *config.Config, database *db.DB, staticDir string) *Server {
	return NewWithFS(cfg, database, nil)
}

// initWebSocketUpgrader creates an upgrader with origin validation
// based on AllowedOrigins config. Respects kill-switch env var for emergency rollback.
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

			// Kill-switch: if validation disabled, allow any origin (for emergency rollback)
			if !s.cfg.WebSocketOriginValidationEnabled {
				log.Printf("[WebSocket] Origin validation disabled (kill-switch). Accepting origin: %q", origin)
				return true
			}

			// Check against whitelist
			for _, allowedOrigin := range s.cfg.AllowedOrigins {
				if origin == allowedOrigin {
					log.Printf("[WebSocket] Accepted origin: %q", origin)
					return true
				}
			}

			log.Printf("[WebSocket] Rejected origin: %q (not in whitelist, validation enabled)", origin)
			return false
		},
	}
}

// Test-only helper methods for accessing internal fields
// These are used for unit testing the WebSocket origin validation

// InitWebSocketUpgraderInternal initializes the WebSocket upgrader (test helper)
func (s *Server) InitWebSocketUpgraderInternal() {
	s.initWebSocketUpgrader()
}

// SetConfigInternal sets the config (test helper)
func (s *Server) SetConfigInternal(cfg *config.Config) {
	s.cfg = cfg
}

// GetWebSocketUpgraderInternal returns the WebSocket upgrader (test helper)
func (s *Server) GetWebSocketUpgraderInternal() *websocket.Upgrader {
	return s.wsUpgrader
}

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

func (s *Server) Start() error {
	// Production safety check: WebSocket origin validation must be enabled
	if s.cfg.Environment == "production" && !s.cfg.WebSocketOriginValidationEnabled {
		return fmt.Errorf("CRITICAL: production mode requires WebSocket origin validation enabled. Set ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED=true or fix AllowedOrigins configuration")
	}

	addr := fmt.Sprintf(":%d", s.cfg.Port)

	s.StartOfflineDetector(60*time.Second, 90*time.Second)
	s.StartScheduler()
	go s.StartHeartbeatDetector()

	if s.fedClient != nil {
		log.Printf("Federation: sub mode enabled, connecting to %s", s.cfg.Federation.RootURL)
		go s.fedClient.Start()
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.requestAuditMiddleware(corsMiddleware(s.cfg.AllowedOrigins)(s.router)),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Handle TLS or external terminator
	if s.cfg.ExternalTLS {
		// External TLS (reverse proxy, etc.) — serve plain HTTP
		log.Printf("ArcVault Coordinator listening on %s (external TLS)", addr)
		return srv.ListenAndServe()
	}

	// HTTPS via self-signed cert
	if s.cfg.CertFile == "" || s.cfg.KeyFile == "" {
		if s.cfg.Environment == "production" {
			log.Printf("WARNING: Running in production without TLS configured. Set cert_file and key_file in config.json, or run 'coordinator init'.")
		} else {
			log.Printf("TLS certificate paths not configured -- falling back to plain HTTP on %s", addr)
		}
		return srv.ListenAndServe()
	}

	log.Printf("ArcVault Coordinator listening on %s (HTTPS)", addr)
	return srv.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile)
}

// === Middleware Helper Functions ===

// adminRoute wraps a handler with JWT, password change check, and admin role requirement
func (s *Server) adminRoute(next http.HandlerFunc) http.HandlerFunc {
	return s.JWTMiddleware(
		RequirePasswordChange("/api/auth/change-password")(
			RequireRole("admin")(next),
		),
	)
}

// adminTokenRoute allows either admin token (for CLI testing) or JWT with admin role
func (s *Server) adminTokenRoute(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// Check for admin token first (for testing/CLI)
		if token == s.cfg.AdminToken {
			next(w, r)
			return
		}

		// Fall back to JWT + role check
		s.JWTMiddleware(
			RequirePasswordChange("/api/auth/change-password")(
				RequireRole("admin")(next),
			),
		)(w, r)
	}
}

// operatorRoute wraps a handler with JWT, password change check, and operator+ role requirement
func (s *Server) operatorRoute(next http.HandlerFunc) http.HandlerFunc {
	return s.JWTMiddleware(
		RequirePasswordChange("/api/auth/change-password")(
			RequireRole("admin", "operator")(next),
		),
	)
}

// viewerRoute wraps a handler with JWT, password change check, and viewer+ role requirement
func (s *Server) viewerRoute(next http.HandlerFunc) http.HandlerFunc {
	return s.JWTMiddleware(
		RequirePasswordChange("/api/auth/change-password")(
			RequireRole("admin", "operator", "viewer")(next),
		),
	)
}

// authRoute wraps a handler with JWT and password change check (for endpoints that don't need role checks)
func (s *Server) authRoute(next http.HandlerFunc) http.HandlerFunc {
	return s.JWTMiddleware(
		RequirePasswordChange("/api/auth/change-password")(next),
	)
}

func (s *Server) registerRoutes() {
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("GET /api/version", s.viewerRoute(s.handleVersion))
	s.router.HandleFunc("GET /ws", s.handleWS)
	s.router.HandleFunc("GET /ws/agent", s.handleAgentWS)
	s.router.HandleFunc("GET /ws/federation", s.fedHub.HandleSubConnect)

	// Auth endpoints (public + JWT-protected)
	s.router.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.router.HandleFunc("POST /api/auth/logout", s.JWTMiddleware(s.handleLogout))
	s.router.HandleFunc("GET /api/auth/me", s.JWTMiddleware(s.handleAuthMe))
	s.router.HandleFunc("PUT /api/auth/change-password", s.JWTMiddleware(s.handleChangePassword))
	s.router.HandleFunc("POST /api/auth/refresh", s.JWTMiddleware(s.handleRefreshToken))

	// User management endpoints (admin only)
	s.router.HandleFunc("GET /api/users", s.JWTMiddleware(s.handleListUsers))
	s.router.HandleFunc("POST /api/users", s.JWTMiddleware(s.handleCreateUser))
	s.router.HandleFunc("DELETE /api/users/{id}", s.JWTMiddleware(s.handleDeleteUser))
	s.router.HandleFunc("PUT /api/users/{id}/role", s.JWTMiddleware(s.handleUpdateUserRole))

	// Groups endpoints
	s.router.HandleFunc("GET /api/groups", s.viewerRoute(s.handleListGroups))
	s.router.HandleFunc("POST /api/groups", s.adminRoute(s.handleCreateGroup))
	s.router.HandleFunc("GET /api/groups/{id}", s.viewerRoute(s.handleGetGroup))
	s.router.HandleFunc("PUT /api/groups/{id}", s.adminRoute(s.handleUpdateGroup))
	s.router.HandleFunc("DELETE /api/groups/{id}", s.adminRoute(s.handleDeleteGroup))
	s.router.HandleFunc("POST /api/groups/{id}/agents", s.adminRoute(s.handleAddAgentToGroup))
	s.router.HandleFunc("DELETE /api/groups/{id}/agents/{agentID}", s.adminRoute(s.handleRemoveAgentFromGroup))
	s.router.HandleFunc("GET /api/groups/{id}/agents", s.viewerRoute(s.handleGetGroupAgents))

	// Agent endpoints (keep agent token auth for backward compatibility)
	s.router.HandleFunc("POST /api/agents/register", s.authMiddleware(s.handleRegister))
	s.router.HandleFunc("POST /api/agents/{id}/heartbeat", s.authMiddleware(s.handleHeartbeat))
	s.router.HandleFunc("POST /api/jobs/{id}/results", s.authMiddleware(s.handlePostJobResults))
	s.router.HandleFunc("POST /api/jobs/{id}/progress", s.authMiddleware(s.handlePostJobProgress))
	s.router.HandleFunc("GET /api/jobs/{id}/progress", s.viewerRoute(s.handleGetProgress))
	s.router.HandleFunc("GET /api/jobs/{id}/logs", s.viewerRoute(s.handleGetJobLogs))
	// Agent list (viewer+) and delete (admin only)
	s.router.HandleFunc("GET /api/agents", s.viewerRoute(s.handleListAgents))
	s.router.HandleFunc("DELETE /api/agents/{id}", s.adminRoute(s.handleDeleteAgent))

	// Credential profiles endpoints
	s.router.HandleFunc("POST /api/credential-profiles", s.adminRoute(s.handleCreateCredentialProfile))
	s.router.HandleFunc("GET /api/credential-profiles", s.viewerRoute(s.handleListCredentialProfiles))
	s.router.HandleFunc("DELETE /api/credential-profiles/{id}", s.adminRoute(s.handleDeleteCredentialProfile))

	// Jobs endpoints
	s.router.HandleFunc("POST /api/jobs", s.operatorRoute(s.handleCreateJob))
	s.router.HandleFunc("GET /api/jobs", s.agentOrViewerRoute(s.handleListJobs))
	s.router.HandleFunc("GET /api/jobs/{id}", s.viewerRoute(s.handleGetJob))
	s.router.HandleFunc("DELETE /api/jobs/{id}", s.adminRoute(s.handleDeleteJob))
	s.router.HandleFunc("POST /api/jobs/{id}/cancel", s.operatorRoute(s.handleCancelJob))
	s.router.HandleFunc("PATCH /api/jobs/{id}/status", s.agentOrOperatorRoute(s.handleUpdateJobStatus))
	s.router.HandleFunc("GET /api/jobs/{id}/runs", s.viewerRoute(s.handleGetJobRuns))
	s.router.HandleFunc("GET /api/job-runs", s.viewerRoute(s.handleListAllJobRuns))

	// Update endpoints (admin only)
	s.router.HandleFunc("GET /api/update/check", s.adminRoute(s.handleCheckUpdate))
	s.router.HandleFunc("POST /api/update/apply", s.adminRoute(s.handleApplyUpdate))
	s.router.HandleFunc("POST /api/agents/{id}/update", s.adminRoute(s.handleAgentUpdate))

	// Rollback endpoints (admin only)
	s.router.HandleFunc("GET /api/rollback-available", s.adminRoute(s.handleRollbackAvailable))
	s.router.HandleFunc("POST /api/rollback", s.adminRoute(s.handleRollback))
	s.router.HandleFunc("POST /api/agents/{id}/rollback", s.adminRoute(s.handleAgentRollback))

	// Templates endpoints
	s.router.HandleFunc("GET /api/templates", s.viewerRoute(s.handleListTemplates))
	s.router.HandleFunc("POST /api/templates", s.adminRoute(s.handleCreateTemplate))
	s.router.HandleFunc("GET /api/templates/{id}", s.viewerRoute(s.handleGetTemplate))
	s.router.HandleFunc("PUT /api/templates/{id}", s.adminRoute(s.handleUpdateTemplate))
	s.router.HandleFunc("DELETE /api/templates/{id}", s.adminRoute(s.handleDeleteTemplate))
	s.router.HandleFunc("POST /api/templates/{id}/run", s.operatorRoute(s.handleRunTemplateNow))

	// Federation endpoints (admin only)
	s.router.HandleFunc("GET /api/federation", s.adminRoute(s.handleListFederation))
	s.router.HandleFunc("POST /api/federation", s.adminRoute(s.handleCreateFederation))
	s.router.HandleFunc("GET /api/federation/{id}", s.adminRoute(s.handleGetFederation))
	s.router.HandleFunc("PUT /api/federation/{id}", s.adminRoute(s.handleUpdateFederation))
	s.router.HandleFunc("DELETE /api/federation/{id}", s.adminRoute(s.handleDeleteFederation))
	s.router.HandleFunc("POST /api/federation/{id}/sync", s.adminRoute(s.handleSyncFederation))
	s.router.HandleFunc("GET /api/federation/{id}/agents", s.adminRoute(s.handleFederationAgents))
	s.router.HandleFunc("GET /api/federation/{id}/jobs", s.adminRoute(s.handleFederationJobs))
	s.router.HandleFunc("GET /api/federation/{id}/history", s.adminRoute(s.handleFederationHistory))

	// Federation state sync endpoints (Phase 16: HA federation)
	s.router.HandleFunc("GET /api/federation/sync", s.adminRoute(s.handleFederationSync))
	s.router.HandleFunc("POST /api/federation/sync/ack", s.adminRoute(s.handleFederationSyncAck))
	s.router.HandleFunc("GET /api/federation/health", s.viewerRoute(s.handleFederationHealth))

	// Admin utility endpoints
	s.router.HandleFunc("GET /api/admin/token", s.adminRoute(s.handleGetAdminToken))
	s.router.HandleFunc("GET /api/admin/bootstrap.ps1", s.adminRoute(s.handleBootstrapScript))
	s.router.HandleFunc("GET /downloads/installer", s.adminRoute(s.handleDownloadInstaller))

	// Downloads (agent.exe auth: agent token OR admin token)
	s.router.HandleFunc("GET /downloads/agent.exe", s.agentOrAdminRoute(s.handleDownloadAgent))

	// Alert rules endpoints (Phase 17: Enhanced monitoring & alerting)
	s.router.HandleFunc("GET /api/alert-rules", s.viewerRoute(s.handleListAlertRules))
	s.router.HandleFunc("POST /api/alert-rules", s.adminRoute(s.handleCreateAlertRule))
	s.router.HandleFunc("PUT /api/alert-rules/{id}", s.adminRoute(s.handleUpdateAlertRule))
	s.router.HandleFunc("DELETE /api/alert-rules/{id}", s.adminRoute(s.handleDeleteAlertRule))
	s.router.HandleFunc("GET /api/alert-history", s.viewerRoute(s.handleListAlertHistory))
	s.router.HandleFunc("POST /api/alert-history/{id}/retry", s.adminRoute(s.handleRetryAlert))

	// Audit endpoints (Phase 2A: Command audit logging)
	s.router.HandleFunc("GET /api/audit/commands", s.viewerRoute(s.handleListAuditCommands))
	s.router.HandleFunc("GET /api/audit/non-whitelisted-programs", s.viewerRoute(s.handleGetNonWhitelistedPrograms))
	s.router.HandleFunc("GET /api/audit/stats", s.viewerRoute(s.handleGetAuditStats))
	s.router.HandleFunc("GET /api/audit/user-actions", s.viewerRoute(s.handleListUserAuditLogs))

	if s.staticFS != nil {
		log.Printf("Serving embedded dashboard")
		s.router.Handle("GET /", http.FileServer(http.FS(s.staticFS)))
	}
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Wildcard "*" is never allowed — CORS spec requires explicit origins
			allowed := false
			for _, o := range allowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}

			// Case 1: Disallowed origin (non-empty and not in whitelist)
			if origin != "" && !allowed {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				// For non-OPTIONS requests, fall through without CORS headers
				next.ServeHTTP(w, r)
				return
			}

			// Case 2: Allowed origin (empty or in whitelist)
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Process actual request
			next.ServeHTTP(w, r)
		})
	}
}

// authMiddleware accepts:
// 1. The admin token from config (full access)
// 2. A valid agent token stored in the tokens table
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// admin token — always valid
		if token == s.cfg.AdminToken {
			next(w, r)
			return
		}

		// check agent token in DB
		if _, err := s.db.ValidateToken(token); err == nil {
			next(w, r)
			return
		}

		http.Error(w, "invalid token", http.StatusUnauthorized)
	}
}

// agentOrViewerRoute accepts an agent token OR the admin token OR a valid JWT
// with viewer+ role. Used for endpoints called by both agents and dashboard users.
func (s *Server) agentOrViewerRoute(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		if token == s.cfg.AdminToken {
			next(w, r)
			return
		}
		if _, err := s.db.ValidateToken(token); err == nil {
			next(w, r)
			return
		}
		s.viewerRoute(next)(w, r)
	}
}

// agentOrOperatorRoute accepts an agent token OR the admin token OR a valid JWT
// with operator+ role. Used for status updates called by both agents and operators.
func (s *Server) agentOrOperatorRoute(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		if token == s.cfg.AdminToken {
			next(w, r)
			return
		}
		if _, err := s.db.ValidateToken(token); err == nil {
			next(w, r)
			return
		}
		s.operatorRoute(next)(w, r)
	}
}

// agentOrAdminRoute accepts an agent token OR the admin token OR a valid JWT
// with admin role. Used for agent downloads.
func (s *Server) agentOrAdminRoute(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		if token == s.cfg.AdminToken {
			next(w, r)
			return
		}
		if _, err := s.db.ValidateToken(token); err == nil {
			next(w, r)
			return
		}
		s.adminRoute(next)(w, r)
	}
}

// adminMiddleware accepts only the admin token from config.
func (s *Server) adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		if token != s.cfg.AdminToken {
			http.Error(w, "admin token required", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// broadcastFedDelta sends a delta event to the root if this coordinator
// is running as a sub. No-op if federation is not configured.
func (s *Server) broadcastFedDelta(msg FedMessage) {
	if s.fedClient != nil {
		s.fedClient.BroadcastDelta(msg)
	}
}

// PaginationMeta provides pagination metadata for list responses
type PaginationMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// ErrorResponse defines the standard error response format
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse for simple success messages
type MessageResponse struct {
	Message string `json:"message"`
}
