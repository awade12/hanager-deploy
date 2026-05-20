package deployapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/hangar-sh/hangar/agent/internal/deploy"
)

var errBadFields = errors.New("build_id, toml, and source tarball required")

type Handler struct {
	svc *deploy.Service
}

func New(svc *deploy.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /deploys", h.create)
	mux.HandleFunc("GET /deploys/{id}", h.get)
	mux.HandleFunc("GET /deploys/{id}/events", h.events)
}

type createRequest struct {
	Tenant         string `json:"tenant"`
	BuildID        string `json:"build_id"`
	TOML           string `json:"toml"`
	SourceBase64   string `json:"source_base64"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	tenant, buildID, tomlBytes, tarball, err := parseCreate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if tenant == "" {
		tenant = "default"
	}
	st, err := h.svc.Create(r.Context(), deploy.CreateInput{
		Tenant:  tenant,
		BuildID: buildID,
		TOML:    tomlBytes,
		Tarball: tarball,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusAccepted, st)
}

func parseCreate(r *http.Request) (tenant, buildID string, toml, tarball []byte, err error) {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if strings.HasPrefix(mediaType, "multipart/") {
		return parseMultipart(r)
	}
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", "", nil, nil, err
	}
	if req.BuildID == "" || req.TOML == "" || req.SourceBase64 == "" {
		return "", "", nil, nil, errBadFields
	}
	raw, err := base64.StdEncoding.DecodeString(req.SourceBase64)
	if err != nil {
		return "", "", nil, nil, err
	}
	return req.Tenant, req.BuildID, []byte(req.TOML), raw, nil
}

func parseMultipart(r *http.Request) (tenant, buildID string, toml, tarball []byte, err error) {
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		return "", "", nil, nil, err
	}
	buildID = r.FormValue("build_id")
	tenant = r.FormValue("tenant")
	tomlStr := r.FormValue("toml")
	if buildID == "" || tomlStr == "" {
		return "", "", nil, nil, errBadFields
	}
	file, _, err := r.FormFile("source")
	if err != nil {
		return "", "", nil, nil, errBadFields
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		return "", "", nil, nil, err
	}
	return tenant, buildID, []byte(tomlStr), raw, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	st, err := h.svc.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	last := ""
	for i := 0; i < 240; i++ {
		st, err := h.svc.Get(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		line, _ := json.Marshal(st)
		payload := string(line)
		if payload != last {
			_, _ = io.WriteString(w, "data: ")
			_, _ = io.WriteString(w, payload)
			_, _ = io.WriteString(w, "\n\n")
			flusher.Flush()
			last = payload
		}
		if st.IsTerminal() {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
