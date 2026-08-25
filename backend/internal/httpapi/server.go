package httpapi

import (
	"net/http"

	"github.com/AlesixDev/MCAIA/backend/internal/ai"
	"github.com/AlesixDev/MCAIA/backend/internal/auth"
	"github.com/AlesixDev/MCAIA/backend/internal/config"
	"github.com/AlesixDev/MCAIA/backend/internal/pipeline"
	"github.com/AlesixDev/MCAIA/backend/internal/store"
)

type Server struct {
	config   config.Config
	store    *store.Store
	pipeline *pipeline.Pipeline
	engine   ai.Engine
	auth     *auth.Service
}

func NewServer(
	cfg config.Config,
	projects *store.Store,
	flow *pipeline.Pipeline,
	engine ai.Engine,
	accounts *auth.Service,
) *Server {
	return &Server{config: cfg, store: projects, pipeline: flow, engine: engine, auth: accounts}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/formats", s.handleFormats)

	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/me", s.handleMe)
	mux.HandleFunc("PATCH /api/v1/me", s.handleUpdateMe)
	mux.HandleFunc("POST /api/v1/me/avatar", s.handleAvatarUpload)
	mux.HandleFunc("GET /api/v1/avatars/{file}", s.handleAvatar)

	mux.HandleFunc("POST /api/v1/projects", s.handleCreateProject)
	mux.HandleFunc("GET /api/v1/projects", s.handleListProjects)
	mux.HandleFunc("GET /api/v1/projects/{id}", s.handleGetProject)
	mux.HandleFunc("PATCH /api/v1/projects/{id}", s.handleRenameProject)
	mux.HandleFunc("DELETE /api/v1/projects/{id}", s.handleDeleteProject)

	mux.HandleFunc("POST /api/v1/projects/{id}/animations", s.handleSaveAnimation)
	mux.HandleFunc("POST /api/v1/projects/{id}/animations/generate", s.handleGenerateAnimation)
	mux.HandleFunc("DELETE /api/v1/projects/{id}/animations/{name}", s.handleDeleteAnimation)

	mux.HandleFunc("POST /api/v1/projects/{id}/export", s.handleExport)

	return withRecovery(withLogging(withCORS(s.config.AllowedOrigins, s.withUser(mux))))
}
