package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/awade12/hanager-deploy/agent/internal/auth"
)

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	var called bool
	h := auth.Middleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if called {
		t.Fatal("handler should not run")
	}
}

func TestMiddlewareAcceptsBearer(t *testing.T) {
	var called bool
	h := auth.Middleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !called {
		t.Fatal("handler should run")
	}
}
