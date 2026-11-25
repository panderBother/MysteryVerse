package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/contracts"
)

func TestClientSignsInternalRequests(t *testing.T) {
	fixedTime := time.Unix(1_700_000_000, 0).UTC()
	fixedNonce := "nonce-fixed-1234"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if r.Header.Get("X-Request-Timestamp") != "1700000000" ||
			r.Header.Get("X-Request-Nonce") != fixedNonce {
			t.Fatalf("unexpected signing headers: %#v", r.Header)
		}
		if r.Header.Get("X-Request-Signature-Version") != "mv-hmac-sha256-v1" ||
			r.Header.Get("X-Service-Key-Id") != "go-core-v1" {
			t.Fatalf("unexpected signature identity headers: %#v", r.Header)
		}
		if r.Header.Get("X-Service-Scope") != "ai:agent:run,run:01J9Z3K4M7Q2R8T5V6W1XE0000" {
			t.Fatalf("unexpected service scope: %q", r.Header.Get("X-Service-Scope"))
		}
		expected := signRequest(
			[]byte("signing-key-material-32-bytes-long"),
			"go-core-v1",
			"1700000000",
			fixedNonce,
			"go-core",
			r,
			body,
		)
		if r.Header.Get("X-Request-Signature") != expected {
			t.Fatalf("signature mismatch")
		}
		_ = json.NewEncoder(w).Encode(newAgentResponse())
	}))
	t.Cleanup(server.Close)

	client, err := New(Config{
		BaseURL:      server.URL,
		ServiceID:    "go-core",
		ServiceToken: "service-token",
		SigningKey:   []byte("signing-key-material-32-bytes-long"),
		SigningKeyID: "go-core-v1",
		Clock:        func() time.Time { return fixedTime },
		NonceFunc:    func() string { return fixedNonce },
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.Run(context.Background(), newAgentRequest(), "idem-sign"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestClientAgainstPythonAgentRuntime(t *testing.T) {
	baseURL := os.Getenv("MV_AI_SERVICE_URL")
	serviceToken := os.Getenv("MV_AI_SERVICE_TOKEN")
	if baseURL == "" || serviceToken == "" {
		t.Skip("MV_AI_SERVICE_URL and MV_AI_SERVICE_TOKEN are not set")
	}
	signingKey := os.Getenv("MV_AI_SERVICE_SIGNING_KEY")
	client, err := New(Config{
		BaseURL:      baseURL,
		ServiceID:    "go-core",
		ServiceToken: serviceToken,
		SigningKey:   []byte(signingKey),
		SigningKeyID: "go-core-v1",
		Timeout:      5 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	response, err := client.Run(context.Background(), newAgentRequest(), "cross-language-idem")
	if err != nil {
		t.Fatalf("Run() cross-language error = %v", err)
	}
	if response.Status == nil ||
		*response.Status != contracts.AgentExecutionStatusSucceeded ||
		response.StructuredOutput == nil {
		t.Fatalf("Run() cross-language response = %#v", response)
	}
}

func TestClientRunAndHeaderContract(t *testing.T) {
	response := newAgentResponse()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Service-Id") != "go-core" ||
			r.Header.Get("X-Service-Token") != "service-token" {
			t.Fatalf("unexpected service headers: %#v", r.Header)
		}
		if r.Header.Get("Idempotency-Key") != "idem-1" ||
			r.Header.Get("X-Request-Id") != "request-1" {
			t.Fatalf("unexpected trace headers: %#v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)

	client, err := New(Config{
		BaseURL:      server.URL,
		ServiceID:    "go-core",
		ServiceToken: "service-token",
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := client.Run(context.Background(), newAgentRequest(), "idem-1")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.RunID != "01J9Z3K4M7Q2R8T5V6W1XE0000" ||
		result.Status == nil ||
		*result.Status != contracts.AgentExecutionStatusSucceeded {
		t.Fatalf("Run() response = %#v", result)
	}
}

func TestClientRetriesDependencyFailuresWithSameIdempotencyKey(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":"SERVICE_UNAVAILABLE","message":"down","retryable":true}}`))
			return
		}
		if r.Header.Get("Idempotency-Key") != "idem-retry" {
			t.Fatal("retry changed Idempotency-Key")
		}
		_ = json.NewEncoder(w).Encode(newAgentResponse())
	}))
	t.Cleanup(server.Close)

	client, err := New(Config{
		BaseURL:      server.URL,
		ServiceID:    "go-core",
		ServiceToken: "service-token",
		Timeout:      time.Second,
		MaxRetries:   1,
		RetryBackoff: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.Run(context.Background(), newAgentRequest(), "idem-retry"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestClientRejectsUnstructuredSuccess(t *testing.T) {
	response := newAgentResponse()
	response.StructuredOutput = nil
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)

	client, err := New(Config{
		BaseURL:      server.URL,
		ServiceID:    "go-core",
		ServiceToken: "service-token",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.Run(context.Background(), newAgentRequest(), "idem-invalid")
	var clientErr Error
	if !errors.As(err, &clientErr) || clientErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("Run() error = %v", err)
	}
}

func newAgentRequest() contracts.AgentRequest {
	taskType := contracts.AgentTaskTypeDemo
	goal := "Summarize the authorized demo context."
	timeout := 1000
	maxSteps := 4
	maxContextTokens := 1000
	traceID := "trace-1"
	return contracts.AgentRequest{
		RunID:              "01J9Z3K4M7Q2R8T5V6W1XE0000",
		RequestID:          "request-1",
		TraceID:            &traceID,
		ActorType:          contracts.ActorTypeDm,
		ActorID:            "dm:1",
		AllowedEntityIds:   []contracts.Ulid{},
		RequestedTopics:    []string{"demo"},
		TaskType:           &taskType,
		TaskGoal:           &goal,
		AllowedTools:       []string{"mock_context_lookup"},
		AllowedPermissions: []string{"context:read"},
		ExecutionLimits: &contracts.AgentExecutionLimits{
			TimeoutMs:        timeout,
			MaxSteps:         maxSteps,
			MaxContextTokens: maxContextTokens,
		},
		SchemaVersion: 1,
		BudgetMs:      timeout,
	}
}

func newAgentResponse() contracts.AgentResponse {
	status := contracts.AgentExecutionStatusSucceeded
	requestID := "request-1"
	traceID := "trace-1"
	return contracts.AgentResponse{
		RunID:            "01J9Z3K4M7Q2R8T5V6W1XE0000",
		RequestID:        &requestID,
		TraceID:          &traceID,
		SchemaVersion:    1,
		Status:           &status,
		StructuredOutput: map[string]any{"answer": "ok", "confidence": 0.9, "used_tool": true},
		SafetyFlags:      []string{},
		Usage: contracts.AgentUsage{
			InputTokens:  10,
			OutputTokens: 3,
		},
	}
}
