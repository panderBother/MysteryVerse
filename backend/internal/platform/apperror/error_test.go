package apperror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrite(t *testing.T) {
	recorder := httptest.NewRecorder()

	Write(recorder, http.StatusConflict, Error{
		Code:      "STATE_VERSION_CONFLICT",
		Message:   "state changed",
		Retryable: true,
		RequestID: "request-1",
	})

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}

	var response Response
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != "STATE_VERSION_CONFLICT" {
		t.Fatalf("code = %q", response.Error.Code)
	}

	if !response.Error.Retryable {
		t.Fatal("Retryable = false, want true")
	}
}
