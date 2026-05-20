package projectapi

import (
	"encoding/json"
	"net/http"

	"github.com/awade12/hanager-deploy/agent/internal/deploy"
	"github.com/awade12/hanager-deploy/agent/internal/runtime"
)

type Handler struct {
	deploy  *deploy.Service
	runtime *runtime.Store
}

func New(deploySvc *deploy.Service, rt *runtime.Store) *Handler {
	return &Handler{deploy: deploySvc, runtime: rt}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /projects/{tenant}/{project}/rollback", h.rollback)
	mux.HandleFunc("GET /projects/{tenant}/{project}/logs", h.logs)
	mux.HandleFunc("GET /projects/{tenant}/{project}", h.get)
}

func (h *Handler) rollback(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")
	project := r.PathValue("project")
	if err := h.deploy.Rollback(r.Context(), tenant, project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rolled_back"})
}

func (h *Handler) logs(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")
	project := r.PathValue("project")
	service := r.URL.Query().Get("service")
	tail := r.URL.Query().Get("tail")
	follow := r.URL.Query().Get("follow") == "true" || r.URL.Query().Get("follow") == "1"
	w.Header().Set("Content-Type", "text/plain")
	if err := h.deploy.Logs(r.Context(), w, tenant, project, service, follow, tail); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")
	project := r.PathValue("project")
	rt, err := h.runtime.Load(tenant, project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
