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

func New(cfg *config.Config, database *db.DB) *Server {
	return NewWithFS(cfg, database, nil)
}

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
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("GET /ws", s.handleWS)

	s.router.HandleFunc("POST /api/agents/register", s.authMiddleware(s.handleRegister))
	s.router.HandleFunc("POST /api/agents/{id}/heartbeat", s.authMiddleware(s.handleHeartbeat))
	s.router.HandleFunc("GET /api/agents", s.authMiddleware(s.handleListAgents))

	s.router.HandleFunc("POST /api/jobs", s.authMiddleware(s.handleCreateJob))
	s.router.HandleFunc("GET /api/jobs", s.authMiddleware(s.handleListJobs))
	s.router.HandleFunc("GET /api/jobs/{id}", s.authMiddleware(s.handleGetJob))
	s.router.HandleFunc("DELETE /api/jobs/{id}", s.authMiddleware(s.handleDeleteJob))
	s.router.HandleFunc("PATCH /api/jobs/{id}/status", s.authMiddleware(s.handleUpdateJobStatus))
	s.router.HandleFunc("POST /api/jobs/{id}/results", s.authMiddleware(s.handlePostJobResults))
	s.router.HandleFunc("GET /api/jobs/{id}/runs", s.authMiddleware(s.handleGetJobRuns))

	s.router.HandleFunc("GET /api/update/check", s.adminMiddleware(s.handleCheckUpdate))
	s.router.HandleFunc("POST /api/update/apply", s.adminMiddleware(s.handleApplyUpdate))

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
