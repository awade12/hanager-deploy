package database

import (
	"encoding/json"
	"net/http"

	"github.com/awade12/hanager-deploy/agent/internal/database"
)

type Handler struct {
	svc *database.Service
}

func New(svc *database.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /databases", h.list)
	mux.HandleFunc("POST /databases", h.create)
	mux.HandleFunc("GET /databases/{name}/url", h.url)
	mux.HandleFunc("DELETE /databases/{name}", h.destroy)
}

type createRequest struct {
	Tenant  string `json:"tenant"`
	Name    string `json:"name"`
	Engine  string `json:"engine"`
	Version string `json:"version"`
	Size    string `json:"size"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Engine == "" {
		http.Error(w, "name and engine required", http.StatusBadRequest)
		return
	}
	if req.Tenant == "" {
		req.Tenant = "default"
	}
	rec, err := h.svc.Create(r.Context(), req.Tenant, req.Name, req.Engine, req.Version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tenant := r.URL.Query().Get("tenant")
	if tenant == "" {
		tenant = "default"
	}
	recs, err := h.svc.List(tenant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"databases": recs})
}

func (h *Handler) url(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tenant := r.URL.Query().Get("tenant")
	if tenant == "" {
		tenant = "default"
	}
	u, err := h.svc.URL(tenant, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": u})
}

func (h *Handler) destroy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tenant := r.URL.Query().Get("tenant")
	if tenant == "" {
		tenant = "default"
	}
	if err := h.svc.Destroy(r.Context(), tenant, name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
