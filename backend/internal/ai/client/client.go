package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/contracts"
)

type Config struct {
	BaseURL       string
	ServiceID     string
	ServiceToken  string
	Timeout       time.Duration
	MaxRetries    int
	RetryBackoff  time.Duration
	HTTPClient    *http.Client
	RequestIDFunc func() string
	SigningKey    []byte
	Clock         func() time.Time
	NonceFunc     func() string
}

type Client struct {
	baseURL       string
	serviceID     string
	serviceToken  string
	maxRetries    int
	retryBackoff  time.Duration
	httpClient    *http.Client
	requestIDFunc func() string
	signingKey    []byte
	clock         func() time.Time
	nonceFunc     func() string
}

type Error struct {
	Code       string
	Message    string
	Retryable  bool
	StatusCode int
}

func (e Error) Error() string {
	return e.Code + ": " + e.Message
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("AI client base URL is required")
	}
	if strings.TrimSpace(cfg.ServiceID) == "" {
		return nil, errors.New("AI client service ID is required")
	}
	if strings.TrimSpace(cfg.ServiceToken) == "" {
		return nil, errors.New("AI client service token is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	requestIDFunc := cfg.RequestIDFunc
	if requestIDFunc == nil {
		requestIDFunc = func() string { return "" }
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	nonceFunc := cfg.NonceFunc
	if nonceFunc == nil {
		nonceFunc = randomNonce
	}
	return &Client{
		baseURL:       strings.TrimRight(cfg.BaseURL, "/"),
		serviceID:     cfg.ServiceID,
		serviceToken:  cfg.ServiceToken,
		maxRetries:    max(cfg.MaxRetries, 0),
		retryBackoff:  max(cfg.RetryBackoff, 0),
		httpClient:    httpClient,
		requestIDFunc: requestIDFunc,
		signingKey:    append([]byte(nil), cfg.SigningKey...),
		clock:         clock,
		nonceFunc:     nonceFunc,
	}, nil
}

func (c *Client) Run(
	ctx context.Context,
	request contracts.AgentRequest,
	idempotencyKey string,
) (contracts.AgentResponse, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return contracts.AgentResponse{}, Error{
			Code:    "VALIDATION_ERROR",
			Message: "Idempotency-Key is required",
		}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return contracts.AgentResponse{}, Error{
			Code:    "VALIDATION_ERROR",
			Message: "The agent request cannot be encoded.",
		}
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		response, err := c.doRun(ctx, body, request, idempotencyKey)
		if err == nil {
			return response, nil
		}
		var clientErr Error
		if !errors.As(err, &clientErr) || !clientErr.Retryable || attempt >= c.maxRetries {
			return contracts.AgentResponse{}, err
		}
		if c.retryBackoff > 0 {
			timer := time.NewTimer(c.retryBackoff * time.Duration(attempt+1))
			select {
			case <-ctx.Done():
				timer.Stop()
				return contracts.AgentResponse{}, dependencyError("The AI request deadline expired.", true)
			case <-timer.C:
			}
		}
	}
	return contracts.AgentResponse{}, dependencyError("The AI request failed.", true)
}

func (c *Client) doRun(
	ctx context.Context,
	body []byte,
	request contracts.AgentRequest,
	idempotencyKey string,
) (contracts.AgentResponse, error) {
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/internal/v1/ai/agent-runs",
		bytes.NewReader(body),
	)
	if err != nil {
		return contracts.AgentResponse{}, dependencyError("The AI request cannot be created.", false)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-Service-Id", c.serviceID)
	httpRequest.Header.Set("X-Service-Token", c.serviceToken)
	httpRequest.Header.Set("X-Request-Id", request.RequestID)
	httpRequest.Header.Set("Idempotency-Key", idempotencyKey)
	if request.TraceID != nil {
		httpRequest.Header.Set("X-Trace-Id", *request.TraceID)
	} else if value := c.requestIDFunc(); value != "" {
		httpRequest.Header.Set("X-Trace-Id", value)
	}
	if len(c.signingKey) > 0 {
		timestamp := strconv.FormatInt(c.clock().UTC().Unix(), 10)
		nonce := c.nonceFunc()
		httpRequest.Header.Set("X-Request-Timestamp", timestamp)
		httpRequest.Header.Set("X-Request-Nonce", nonce)
		httpRequest.Header.Set(
			"X-Request-Signature",
			signRequest(c.signingKey, timestamp, nonce, c.serviceID, httpRequest, body),
		)
	}

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return contracts.AgentResponse{}, dependencyError("The AI request timed out.", true)
		}
		return contracts.AgentResponse{}, dependencyError("The AI service is unavailable.", true)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	payload, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return contracts.AgentResponse{}, dependencyError("The AI response cannot be read.", true)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var errorResponse contracts.ApiErrorResponse
		if err := decodeStrict(payload, &errorResponse); err == nil {
			return contracts.AgentResponse{}, Error{
				Code:       string(errorResponse.Error.Code),
				Message:    errorResponse.Error.Message,
				Retryable:  errorResponse.Error.Retryable,
				StatusCode: response.StatusCode,
			}
		}
		return contracts.AgentResponse{}, Error{
			Code:       "DEPENDENCY_ERROR",
			Message:    "The AI service returned an invalid error response.",
			Retryable:  response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500,
			StatusCode: response.StatusCode,
		}
	}

	var agentResponse contracts.AgentResponse
	if err := decodeStrict(payload, &agentResponse); err != nil {
		return contracts.AgentResponse{}, Error{
			Code:    "WORKFLOW_FAILED",
			Message: "The AI service returned an invalid agent response.",
		}
	}
	if err := validateResponse(request, agentResponse); err != nil {
		return contracts.AgentResponse{}, err
	}
	return agentResponse, nil
}

func validateResponse(
	request contracts.AgentRequest,
	response contracts.AgentResponse,
) error {
	if response.RunID != request.RunID {
		return Error{
			Code:    "WORKFLOW_FAILED",
			Message: "The AI response run_id does not match the request.",
		}
	}
	if response.Status == nil {
		return Error{
			Code:    "WORKFLOW_FAILED",
			Message: "The AI response is missing an execution status.",
		}
	}
	switch *response.Status {
	case contracts.AgentExecutionStatusSucceeded:
		if response.StructuredOutput == nil {
			return Error{
				Code:    "VALIDATION_ERROR",
				Message: "A successful AI response must include structured_output.",
			}
		}
	case contracts.AgentExecutionStatusFailed,
		contracts.AgentExecutionStatusTimeout,
		contracts.AgentExecutionStatusCancelled:
	default:
		return Error{
			Code:    "WORKFLOW_FAILED",
			Message: fmt.Sprintf("Unsupported AI execution status %q.", *response.Status),
		}
	}
	if response.RequestID != nil && *response.RequestID != request.RequestID {
		return Error{
			Code:    "WORKFLOW_FAILED",
			Message: "The AI response request_id does not match the request.",
		}
	}
	return nil
}

func decodeStrict(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("response contains multiple JSON values")
	}
	return nil
}

func dependencyError(message string, retryable bool) Error {
	return Error{
		Code:      "DEPENDENCY_ERROR",
		Message:   message,
		Retryable: retryable,
	}
}

func signRequest(
	key []byte,
	timestamp string,
	nonce string,
	serviceID string,
	request *http.Request,
	body []byte,
) string {
	bodyHash := sha256.Sum256(body)
	canonical := strings.Join(
		[]string{
			timestamp,
			nonce,
			serviceID,
			request.Method,
			request.URL.Path,
			request.URL.RawQuery,
			hex.EncodeToString(bodyHash[:]),
		},
		"\n",
	)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomNonce() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(value)
}
