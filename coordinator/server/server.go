package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
)

type Server struct {
	cfg      *config.Config
	db       *db.DB
	router   *http.ServeMux
	hub      *Hub
	staticFS fs.FS
}

// New creates a server with no static file serving.
// Use NewWithFS to serve an embedded or disk-based filesystem.
func New(cfg *config.Config, database *db.DB) *Server {
	return NewWithFS(cfg, database, nil)
}

// NewWithFS creates a server that serves the given filesystem at GET /.
// Pass nil to skip static serving (used in tests).
func NewWithFS(cfg *config.Config, database *db.DB, staticFS fs.FS) *Server {
	s := &Server{
		cfg:      cfg,
		db:       database,
		router:   http.NewServeMux(),
		hub:      newHub(),
		staticFS: staticFS,
	}
	s.registerRoutes()
	return s
}

// NewWithStatic creates a server that serves files from a directory on disk.
// Used for development and backward compatibility. Pass empty string to skip.
func NewWithStatic(cfg *config.Config, database *db.DB, staticDir string) *Server {
	return NewWithFS(cfg, database, nil)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("ArcVault Coordinator listening on %s", addr)

	s.StartOfflineDetector(60*time.Second, 90*time.Second)
	s.StartScheduler()

	return http.ListenAndServe(addr, corsMiddleware(s.router))
}

func (s *Server) registerRoutes() {
	// health
	s.router.HandleFunc("GET /health", s.handleHealth)

	// websocket
	s.router.HandleFunc("GET /ws", s.handleWS)

	// agents
	s.router.HandleFunc("POST /api/agents/register", s.authMiddleware(s.handleRegister))
	s.router.HandleFunc("POST /api/agents/{id}/heartbeat", s.authMiddleware(s.handleHeartbeat))
	s.router.HandleFunc("GET /api/agents", s.authMiddleware(s.handleListAgents))

	// jobs - CRUD
	s.router.HandleFunc("POST /api/jobs", s.authMiddleware(s.handleCreateJob))
	s.router.HandleFunc("GET /api/jobs", s.authMiddleware(s.handleListJobs))
	s.router.HandleFunc("GET /api/jobs/{id}", s.authMiddleware(s.handleGetJob))
	s.router.HandleFunc("DELETE /api/jobs/{id}", s.authMiddleware(s.handleDeleteJob))

	// jobs - lifecycle
	s.router.HandleFunc("PATCH /api/jobs/{id}/status", s.authMiddleware(s.handleUpdateJobStatus))
	s.router.HandleFunc("POST /api/jobs/{id}/results", s.authMiddleware(s.handlePostJobResults))

	// job runs
	s.router.HandleFunc("GET /api/jobs/{id}/runs", s.authMiddleware(s.handleGetJobRuns))

	// static dashboard
	if s.staticFS != nil {
		log.Printf("Serving embedded dashboard")
		s.router.Handle("GET /", http.FileServer(http.FS(s.staticFS)))
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
		if token != s.cfg.AdminToken {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
