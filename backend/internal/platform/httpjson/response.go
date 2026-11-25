package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/pkg/version"
)

const maxRequestBytes = 1 << 20

type successEnvelope[T any] struct {
	Data T    `json:"data"`
	Meta Meta `json:"meta"`
}

type Meta struct {
	APIVersion       string    `json:"api_version"`
	RequestID        string    `json:"request_id"`
	ServerTime       time.Time `json:"server_time"`
	IdempotentReplay bool      `json:"idempotent_replay,omitempty"`
}

func WriteSuccess(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	data any,
	idempotentReplay bool,
) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(successEnvelope[any]{
		Data: data,
		Meta: Meta{
			APIVersion:       version.ContractAPI,
			RequestID:        middleware.GetReqID(r.Context()),
			ServerTime:       time.Now().UTC(),
			IdempotentReplay: idempotentReplay,
		},
	})
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	response := apperror.Error{
		Code:      "INTERNAL_ERROR",
		Message:   "An internal error occurred.",
		Retryable: true,
	}
	status := http.StatusInternalServerError

	var appErr apperror.Error
	if errors.As(err, &appErr) {
		response = appErr
		status = apperror.StatusForCode(appErr.Code)
	}
	response.RequestID = middleware.GetReqID(r.Context())

	apperror.Write(w, status, response)
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return apperror.New("VALIDATION_ERROR", "A JSON request body is required.", false, nil)
		}
		return apperror.New("VALIDATION_ERROR", "The JSON request body is invalid.", false, nil)
	}

	if decoder.Decode(&struct{}{}) != io.EOF {
		return apperror.New("VALIDATION_ERROR", "The request body must contain one JSON object.", false, nil)
	}
	return nil
}
