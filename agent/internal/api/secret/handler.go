package secret

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/hangar-sh/hangar/agent/internal/secret"
)

type Handler struct {
	store *secret.Store
}

func New(store *secret.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /secrets", h.list)
	mux.HandleFunc("POST /secrets/{key}", h.set)
	mux.HandleFunc("DELETE /secrets/{key}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tenant := r.URL.Query().Get("tenant")
	if tenant == "" {
		tenant = "default"
	}
	keys, err := h.store.List(tenant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"secrets": keys})
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	tenant := r.URL.Query().Get("tenant")
	if tenant == "" {
		tenant = "default"
	}
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		b, _ := io.ReadAll(r.Body)
		body.Value = string(b)
	}
	if body.Value == "" {
		http.Error(w, "value required", http.StatusBadRequest)
		return
	}
	if err := h.store.Set(tenant, key, body.Value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	tenant := r.URL.Query().Get("tenant")
	if tenant == "" {
		tenant = "default"
	}
	if err := h.store.Delete(tenant, key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
