package server

import (
"fmt"
"log"
"net/http"

"arcvault/coordinator/config"
"arcvault/coordinator/db"
)

type Server struct {
cfg    *config.Config
db     *db.DB
router *http.ServeMux
}

func New(cfg *config.Config, database *db.DB) *Server {
s := &Server{
cfg:    cfg,
db:     database,
router: http.NewServeMux(),
}
s.registerRoutes()
return s
}

func (s *Server) Start() error {
addr := fmt.Sprintf(":%d", s.cfg.Port)
log.Printf("ArcVault Coordinator listening on %s", addr)
return http.ListenAndServe(addr, s.router)
}

func (s *Server) registerRoutes() {
s.router.HandleFunc("GET /health", s.handleHealth)
s.router.HandleFunc("POST /api/agents/register", s.authMiddleware(s.handleRegister))
s.router.HandleFunc("POST /api/agents/{id}/heartbeat", s.authMiddleware(s.handleHeartbeat))
s.router.HandleFunc("GET /api/agents", s.authMiddleware(s.handleListAgents))
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
token := r.Header.Get("Authorization")
if token == "" {
http.Error(w, "missing Authorization header", http.StatusUnauthorized)
return
}
// Strip "Bearer " prefix if present
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