package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
)

type Server struct {
	cfg    *config.Config
	db     *db.DB
	router *http.ServeMux
	hub    *Hub
}

func New(cfg *config.Config, database *db.DB) *Server {
	s := &Server{
		cfg:    cfg,
		db:     database,
		router: http.NewServeMux(),
		hub:    newHub(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("ArcVault Coordinator listening on %s", addr)

	// check every 60s, mark offline after 90s without heartbeat
	s.StartOfflineDetector(60*time.Second, 90*time.Second)

	// start cron-based job scheduler
	s.StartScheduler()

	return http.ListenAndServe(addr, corsMiddleware(s.router))
}

func (s *Server) registerRoutes() {
	// health
	s.router.HandleFunc("GET /health", s.handleHealth)

	// websocket -- auth handled inside handleWS (query param support)
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
