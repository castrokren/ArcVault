package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
	"arcvault/coordinator/notifications"
)

// Version is injected at build time via -ldflags.
var Version = "dev"

type Server struct {
	cfg           *config.Config
	db            *db.DB
	router        *http.ServeMux
	hub           *Hub
	fedHub        *FederationHub
	fedClient     *FederationClient
	staticFS      fs.FS
	Notifier      *notifications.Dispatcher
	coordinatorID string
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
	}

	if cfg.Federation != nil {
		s.fedClient = NewFederationClient(cfg.Federation, database, Version)
	}

	s.registerRoutes()
	return s
}

func NewWithStatic(cfg *config.Config, database *db.DB, staticDir string) *Server {
	return NewWithFS(cfg, database, nil)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("ArcVault Coordinator listening on %s", addr)

	s.StartOfflineDetector(60*time.Second, 90*time.Second)
	s.StartScheduler()
	go s.StartHeartbeatDetector()

	if s.fedClient != nil {
		log.Printf("Federation: sub mode enabled, connecting to %s", s.cfg.Federation.RootURL)
		go s.fedClient.Start()
	}

	return http.ListenAndServe(addr, corsMiddleware(s.router))
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

	// Agent list (viewer+)
	s.router.HandleFunc("GET /api/agents", s.viewerRoute(s.handleListAgents))

	// Jobs endpoints
	s.router.HandleFunc("POST /api/jobs", s.operatorRoute(s.handleCreateJob))
	s.router.HandleFunc("GET /api/jobs", s.viewerRoute(s.handleListJobs))
	s.router.HandleFunc("GET /api/jobs/{id}", s.viewerRoute(s.handleGetJob))
	s.router.HandleFunc("DELETE /api/jobs/{id}", s.adminRoute(s.handleDeleteJob))
	s.router.HandleFunc("PATCH /api/jobs/{id}/status", s.operatorRoute(s.handleUpdateJobStatus))
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

	// Alert rules endpoints (Phase 17: Enhanced monitoring & alerting)
	s.router.HandleFunc("GET /api/alert-rules", s.viewerRoute(s.handleListAlertRules))
	s.router.HandleFunc("POST /api/alert-rules", s.adminRoute(s.handleCreateAlertRule))
	s.router.HandleFunc("PUT /api/alert-rules/{id}", s.adminRoute(s.handleUpdateAlertRule))
	s.router.HandleFunc("DELETE /api/alert-rules/{id}", s.adminRoute(s.handleDeleteAlertRule))
	s.router.HandleFunc("GET /api/alert-history", s.viewerRoute(s.handleListAlertHistory))
	s.router.HandleFunc("POST /api/alert-history/{id}/retry", s.adminRoute(s.handleRetryAlert))

	if s.staticFS != nil {
		log.Printf("Serving embedded dashboard")
		s.router.Handle("GET /", http.FileServer(http.FS(s.staticFS)))
	}
}

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
