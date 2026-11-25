package room

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mysteryverse/mysteryverse/backend/internal/identity"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/httpjson"
)

type Handler struct {
	service *Service
}

type eventPage struct {
	Events     []RoomEventView `json:"events"`
	CurrentSeq uint64          `json:"current_seq"`
	HasMore    bool            `json:"has_more"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, auth *identity.Handler) {
	r.Get("/api/v1/rooms", h.listPublic)

	r.Group(func(protected chi.Router) {
		protected.Use(auth.SessionMiddleware())
		protected.Get("/api/v1/rooms/{room_id}", h.projection)
		protected.Get("/api/v1/rooms/{room_id}/messages", h.messages)
		protected.Get("/api/v1/rooms/{room_id}/events", h.events)

		protected.Group(func(writes chi.Router) {
			writes.Use(auth.CSRFMiddleware())
			writes.Post("/api/v1/rooms", h.create)
			writes.Post("/api/v1/rooms/{room_id}/join", h.join)
			writes.Post("/api/v1/rooms/{room_id}/leave", h.leave)
			writes.Post("/api/v1/rooms/{room_id}/ready", h.ready)
			writes.Post("/api/v1/rooms/{room_id}/host-actions", h.hostAction)
			writes.Post("/api/v1/rooms/{room_id}/messages", h.sendMessage)
		})
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}

	var input CreateInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Create(
		r.Context(),
		session.UserID,
		input,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusCreated, result, result.IdempotentReplay)
}

func (h *Handler) listPublic(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			httpjson.WriteError(w, r, validationError("limit must be between 1 and 100"))
			return
		}
		limit = parsed
	}
	cards, err := h.service.ListPublic(r.Context(), r.URL.Query().Get("game_type"), int32(limit))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, map[string]any{
		"items":       cards,
		"next_cursor": nil,
	}, false)
}

func (h *Handler) projection(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}
	projection, err := h.service.Projection(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, projection, false)
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}
	var input JoinInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Join(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		input,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.IdempotentReplay)
}

func (h *Handler) leave(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}
	var input LeaveInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Leave(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		input,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}
	var input ReadyInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Ready(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		input,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) hostAction(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}
	var input HostActionInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.HostAction(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		input,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}
	var input SendMessageInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.SendMessage(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		input,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func (h *Handler) messages(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}

	var cursor *uint64
	if value := r.URL.Query().Get("cursor"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil || parsed == 0 {
			httpjson.WriteError(w, r, apperror.New("INVALID_CURSOR", "The cursor is invalid.", false, nil))
			return
		}
		cursor = &parsed
	}
	limit := 30
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			httpjson.WriteError(w, r, validationError("limit must be between 1 and 100"))
			return
		}
		limit = parsed
	}
	page, err := h.service.Messages(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		cursor,
		limit,
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, page, false)
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil))
		return
	}
	afterSeq := uint64(0)
	if value := r.URL.Query().Get("after_seq"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			httpjson.WriteError(w, r, apperror.New("INVALID_SEQ_RANGE", "after_seq is invalid.", false, nil))
			return
		}
		afterSeq = parsed
	}
	limit := 100
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			httpjson.WriteError(w, r, validationError("limit must be between 1 and 200"))
			return
		}
		limit = parsed
	}
	events, current, err := h.service.RoomEvents(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		afterSeq,
		limit,
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, eventPage{
		Events:     events,
		CurrentSeq: current,
		HasMore:    len(events) == limit,
	}, false)
}
