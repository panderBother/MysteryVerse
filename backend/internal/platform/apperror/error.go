package apperror

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Code      string `json:"code"`
	Details   any    `json:"details,omitempty"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Retryable bool   `json:"retryable"`
	TraceID   string `json:"trace_id,omitempty"`
}

type Response struct {
	Error Error `json:"error"`
}

func (e Error) Error() string {
	return e.Code + ": " + e.Message
}

func New(code string, message string, retryable bool, details any) Error {
	return Error{
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: retryable,
	}
}

func StatusForCode(code string) int {
	switch code {
	case "VALIDATION_ERROR", "INVALID_CURSOR", "MESSAGE_TOO_LONG", "INVALID_SEQ_RANGE",
		"UNSUPPORTED_OBJECT_TYPE":
		return http.StatusBadRequest
	case "AUTH_REQUIRED", "AUTH_INVALID":
		return http.StatusUnauthorized
	case "ACCOUNT_RESTRICTED", "CSRF_INVALID", "SESSION_REVOKED", "FORBIDDEN_VIEW",
		"FORBIDDEN_ACTION", "NOT_ROOM_MEMBER", "HOST_REQUIRED", "MUTED",
		"PASSWORD_INVALID", "ROLE_NOT_BOUND", "FORBIDDEN_EDIT", "MODERATION_REQUIRED":
		return http.StatusForbidden
	case "ROOM_NOT_FOUND", "USER_NOT_FOUND", "NOT_FOUND", "SCRIPT_NOT_FOUND",
		"VERSION_NOT_FOUND", "PUBLICATION_NOT_FOUND", "OBJECT_NOT_FOUND",
		"CONTENT_NOT_FOUND":
		return http.StatusNotFound
	case "ROOM_FULL", "ALREADY_JOINED", "ROOM_NOT_WAITING", "ROOM_CLOSED",
		"PLAYERS_NOT_READY", "PLAYER_COUNT_MISMATCH", "IDEMPOTENCY_CONFLICT",
		"VERSION_CONFLICT", "STATE_VERSION_CONFLICT", "ROOM_IN_GAME",
		"HOST_TRANSFER_REQUIRED", "TARGET_NOT_MEMBER", "CONTENT_UNAVAILABLE",
		"IDENTIFIER_ALREADY_EXISTS", "NICKNAME_CONFLICT", "CLUE_ALREADY_CLAIMED",
		"PHASE_NOT_ALLOWED", "ACTION_NOT_ALLOWED", "TARGET_INVALID",
		"PAUSE_NOT_ALLOWED", "GAME_ALREADY_ENDED", "VERSION_FROZEN",
		"PUBLISH_BLOCKED":
		return http.StatusConflict
	case "INVITATION_INVALID", "INVITATION_EXPIRED":
		return http.StatusGone
	case "RATE_LIMITED":
		return http.StatusTooManyRequests
	case "DEPENDENCY_ERROR", "SERVICE_UNAVAILABLE":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func Write(w http.ResponseWriter, status int, response Error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(Response{Error: response})
}
