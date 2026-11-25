package script

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

type deleteObjectRequest struct {
	ExpectedRowVersion uint32 `json:"expected_row_version"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, auth *identity.Handler) {
	r.Get("/api/v1/scripts", h.listPublic)
	r.Get("/api/v1/scripts/{script_id}", h.publicDetail)

	r.Group(func(protected chi.Router) {
		protected.Use(auth.SessionMiddleware())
		protected.Get("/api/v1/scripts/{script_id}/versions", h.versions)
		protected.Get("/api/v1/scripts/{script_id}/versions/{version_id}/editor", h.editor)

		protected.Group(func(writes chi.Router) {
			writes.Use(auth.CSRFMiddleware())
			writes.Post("/api/v1/scripts", h.create)
			writes.Patch("/api/v1/scripts/{script_id}", h.updateMetadata)
			writes.Delete("/api/v1/scripts/{script_id}", h.deleteScript)
			writes.Post("/api/v1/scripts/{script_id}/versions/{version_id}/drafts", h.createDraft)
			writes.Post(
				"/api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}",
				h.createObject,
			)
			writes.Patch(
				"/api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}",
				h.updateObject,
			)
			writes.Delete(
				"/api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}",
				h.deleteObject,
			)
			writes.Post("/api/v1/scripts/{script_id}/versions/{version_id}/validate", h.validate)
			writes.Post("/api/v1/scripts/{script_id}/versions/{version_id}/publish", h.publish)
			writes.Post("/api/v1/publications/{publication_id}/unpublish", h.unpublish)
		})
	})
}

func (h *Handler) listPublic(w http.ResponseWriter, r *http.Request) {
	minPlayers, err := queryInt(r, "min_players", 0, 0, 100)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	maxPlayers, err := queryInt(r, "max_players", 0, 0, 100)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	durationMax, err := queryInt(r, "duration_max", 0, 0, 1440)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	limit, err := queryInt(r, "limit", 20, 1, 100)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	page, err := h.service.ListPublic(
		r.Context(),
		strings.TrimSpace(r.URL.Query().Get("q")),
		strings.TrimSpace(r.URL.Query().Get("game_type")),
		minPlayers,
		maxPlayers,
		durationMax,
		strings.TrimSpace(r.URL.Query().Get("difficulty")),
		strings.TrimSpace(r.URL.Query().Get("cursor")),
		limit,
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, page, false)
}

func (h *Handler) publicDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.PublicDetail(r.Context(), chi.URLParam(r, "script_id"))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, detail, false)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input CreateScriptInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Create(
		r.Context(),
		session.UserID,
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusCreated, result, result.IdempotentReplay)
}

func (h *Handler) updateMetadata(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input UpdateScriptInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.UpdateMetadata(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) deleteScript(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input DeleteScriptInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.SoftDelete(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) versions(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	limit, err := queryInt(r, "limit", 50, 1, 100)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	items, err := h.service.Versions(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		strings.TrimSpace(r.URL.Query().Get("status")),
		limit,
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, map[string]any{
		"items":       items,
		"next_cursor": nil,
	}, false)
}

func (h *Handler) editor(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	result, err := h.service.EditorContent(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		chi.URLParam(r, "version_id"),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) createDraft(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input CreateDraftInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.CreateDraft(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		chi.URLParam(r, "version_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusCreated, result, false)
}

func (h *Handler) createObject(w http.ResponseWriter, r *http.Request) {
	h.objectMutation(w, r, false)
}

func (h *Handler) updateObject(w http.ResponseWriter, r *http.Request) {
	h.objectMutation(w, r, true)
}

func (h *Handler) objectMutation(w http.ResponseWriter, r *http.Request, update bool) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input ObjectMutationInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	scriptID := chi.URLParam(r, "script_id")
	versionID := chi.URLParam(r, "version_id")
	objectType := chi.URLParam(r, "object_type")
	objectID := chi.URLParam(r, "object_id")
	var result ObjectMutationResult
	var err error
	if update {
		result, err = h.service.UpdateObject(
			r.Context(),
			session.UserID,
			scriptID,
			versionID,
			objectType,
			objectID,
			input,
			idempotencyKey(r),
		)
	} else {
		result, err = h.service.CreateObject(
			r.Context(),
			session.UserID,
			scriptID,
			versionID,
			objectType,
			input,
			idempotencyKey(r),
		)
	}
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	status := http.StatusOK
	if !update {
		status = http.StatusCreated
	}
	httpjson.WriteSuccess(w, r, status, result, false)
}

func (h *Handler) deleteObject(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input deleteObjectRequest
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.DeleteObject(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		chi.URLParam(r, "version_id"),
		chi.URLParam(r, "object_type"),
		chi.URLParam(r, "object_id"),
		input.ExpectedRowVersion,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, false)
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input ValidateInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	report, err := h.service.Validate(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		chi.URLParam(r, "version_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, report, false)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input PublishInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Publish(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "script_id"),
		chi.URLParam(r, "version_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func (h *Handler) unpublish(w http.ResponseWriter, r *http.Request) {
	session, ok := identity.SessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	var input UnpublishInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	result, err := h.service.Unpublish(
		r.Context(),
		session.UserID,
		chi.URLParam(r, "publication_id"),
		input,
		idempotencyKey(r),
	)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, result, result.Replay)
}

func queryInt(r *http.Request, name string, fallback int, minimum int, maximum int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, validationError(name + " is invalid")
	}
	return parsed, nil
}

func idempotencyKey(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("Idempotency-Key"))
}

func authRequiredError() error {
	return apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil)
}
