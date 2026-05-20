package api

import (
	"net/http"

	"github.com/hangar-sh/hangar/agent/internal/api/database"
	deployapi "github.com/hangar-sh/hangar/agent/internal/api/deploy"
	"github.com/hangar-sh/hangar/agent/internal/api/health"
	projectapi "github.com/hangar-sh/hangar/agent/internal/api/project"
	secretapi "github.com/hangar-sh/hangar/agent/internal/api/secret"
	"github.com/hangar-sh/hangar/agent/internal/auth"
	"github.com/hangar-sh/hangar/agent/internal/deploy"
	"github.com/hangar-sh/hangar/agent/internal/runtime"
	"github.com/hangar-sh/hangar/agent/internal/secret"
	dbpkg "github.com/hangar-sh/hangar/agent/internal/database"
)

type Server struct {
	handler http.Handler
}

func New(deploySvc *deploy.Service, rt *runtime.Store, secrets *secret.Store, dbSvc *dbpkg.Service, token string) *Server {
	mux := http.NewServeMux()
	mux.Handle("GET /health", health.New())
	deployapi.New(deploySvc).Register(mux)
	projectapi.New(deploySvc, rt).Register(mux)
	secretapi.New(secrets).Register(mux)
	database.New(dbSvc).Register(mux)
	return &Server{handler: auth.Middleware(token)(mux)}
}

func (s *Server) Handler() http.Handler {
	return s.handler
}
