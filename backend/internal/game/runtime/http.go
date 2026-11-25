package runtime

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

type expectedVersionInput struct {
	ExpectedVersion uint64 `json:"expected_version"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, auth *identity.Handler) {
	r.Group(func(protected chi.Router) {
		protected.Use(auth.SessionMiddleware())
		protected.Get("/api/v1/games/{game_id}/projection", h.projection)
		protected.Get("/api/v1/games/{game_id}/events", h.events)
		protected.Get("/api/v1/games/{game_id}/result", h.result)
		protected.Get("/api/v1/games/{game_id}/murder/me", h.myRole)
		protected.Get("/api/v1/games/{game_id}/murder/search-points", h.searchPoints)
		protected.Get("/api/v1/games/{game_id}/murder/clues", h.clues)
		protected.Get("/api/v1/games/{game_id}/murder/votes", h.votes)

		protected.Group(func(writes chi.Router) {
			writes.Use(auth.CSRFMiddleware())
			writes.Post("/api/v1/rooms/{room_id}/start", h.start)
			writes.Post("/api/v1/games/{game_id}/murder/reading-complete", h.readingComplete)
			writes.Post("/api/v1/games/{game_id}/murder/searches", h.search)
			writes.Post("/api/v1/games/{game_id}/murder/votes", h.vote)
			writes.Post("/api/v1/games/{game_id}/pause", h.pause)
			writes.Post("/api/v1/games/{game_id}/resume", h.resume)
		})
	})
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	var input StartInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Start(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "room_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusCreated, result, result.Replay)
}

func (h *Handler) projection(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	result, err := h.service.Projection(r.Context(), session.UserID, chi.URLParam(r, "game_id"))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
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
	result, err := h.service.Events(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "game_id"),
		afterSeq,
		limit,
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) myRole(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	result, err := h.service.MyRole(r.Context(), session.UserID, chi.URLParam(r, "game_id"))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) readingComplete(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	var input expectedVersionInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.ReadingComplete(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "game_id"),
		input.ExpectedVersion,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func (h *Handler) searchPoints(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	result, err := h.service.SearchPoints(r.Context(), session.UserID, chi.URLParam(r, "game_id"))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	var input SearchInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Search(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "game_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func (h *Handler) clues(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	result, err := h.service.Clues(r.Context(), session.UserID, chi.URLParam(r, "game_id"))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, map[string]any{
		"items":       result,
		"next_cursor": nil,
	}, false)
}

func (h *Handler) vote(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	var input VoteInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Vote(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "game_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func (h *Handler) votes(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	result, err := h.service.Votes(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "game_id"),
		r.URL.Query().Get("round_key"),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) pause(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	var input PauseInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Pause(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "game_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func (h *Handler) resume(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	var input PauseInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Resume(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "game_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func (h *Handler) result(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequired())
		return
	}
	result, err := h.service.Result(r.Context(), session.UserID, chi.URLParam(r, "game_id"))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func idempotencyKey(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("Idempotency-Key"))
}

func authRequired() error {
	return apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil)
}
