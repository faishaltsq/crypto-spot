package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/crypto-spot-signal/internal/config"
)

func TestIdentityFailsClosedOutsideDevelopment(t *testing.T) {
	server := Server{cfg: config.Config{AppEnv: "production"}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-User-ID", "e8b1da7c-af4e-4d15-b41c-d0e54f46e3c8")
	request.Header.Set("X-User-Role", "admin")
	if _, ok := server.identity(request); ok {
		t.Fatal("production identity must fail closed without auth middleware")
	}
}

func TestRequireAdminRejectsViewer(t *testing.T) {
	server := Server{cfg: config.Config{AppEnv: "development"}}
	handler := server.requireAdmin(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPut, "/", nil)
	request.Header.Set("X-User-ID", "e8b1da7c-af4e-4d15-b41c-d0e54f46e3c8")
	request.Header.Set("X-User-Role", "viewer")
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("got %d", response.Code)
	}
}

func TestRequireAdminAllowsDevelopmentAdmin(t *testing.T) {
	server := Server{cfg: config.Config{AppEnv: "development"}}
	handler := server.requireAdmin(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPut, "/", nil)
	request.Header.Set("X-User-ID", "e8b1da7c-af4e-4d15-b41c-d0e54f46e3c8")
	request.Header.Set("X-User-Role", "admin")
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("got %d", response.Code)
	}
}

func TestIdentityRejectsInvalidDevelopmentUserID(t *testing.T) {
	server := Server{cfg: config.Config{AppEnv: "development"}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-User-ID", "not-a-uuid")
	request.Header.Set("X-User-Role", "admin")
	if _, ok := server.identity(request); ok {
		t.Fatal("invalid development user ID must be rejected")
	}
}
