package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/health"
	"github.com/mysteryverse/mysteryverse/backend/pkg/version"
)

func TestHealthRoutes(t *testing.T) {
	handler := NewHandler(nil, health.New("go-api", version.ServiceVersion))

	for _, path := range []string{"/health/live", "/health/ready", "/api/v1/health"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, recorder.Code, http.StatusOK)
		}

		var payload successEnvelope[health.Payload]
		if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
			t.Fatalf("%s decode: %v", path, err)
		}

		if payload.Data.Service != "go-api" {
			t.Fatalf("%s service = %q", path, payload.Data.Service)
		}
	}
}

func TestNotFoundUsesErrorEnvelope(t *testing.T) {
	handler := NewHandler(nil, health.New("go-api", version.ServiceVersion))
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if response.Error.Code != "NOT_FOUND" {
		t.Fatalf("code = %q, want NOT_FOUND", response.Error.Code)
	}
}
