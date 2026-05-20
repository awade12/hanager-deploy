package api

import (
	"net/http"

	"github.com/awade12/hanager-deploy/agent/internal/api/database"
	deployapi "github.com/awade12/hanager-deploy/agent/internal/api/deploy"
	"github.com/awade12/hanager-deploy/agent/internal/api/health"
	projectapi "github.com/awade12/hanager-deploy/agent/internal/api/project"
	secretapi "github.com/awade12/hanager-deploy/agent/internal/api/secret"
	"github.com/awade12/hanager-deploy/agent/internal/auth"
	"github.com/awade12/hanager-deploy/agent/internal/deploy"
	"github.com/awade12/hanager-deploy/agent/internal/runtime"
	"github.com/awade12/hanager-deploy/agent/internal/secret"
	dbpkg "github.com/awade12/hanager-deploy/agent/internal/database"
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
