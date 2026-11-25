package identity

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/httpjson"
)

const csrfHeader = "X-CSRF-Token"

type contextKey string

const sessionContextKey contextKey = "identity.session"

type Handler struct {
	service *Service
}

type sessionResponse struct {
	User             UserView    `json:"user"`
	Profile          ProfileView `json:"profile"`
	SessionExpiresAt time.Time   `json:"session_expires_at"`
	CSRFToken        string      `json:"csrf_token"`
}

type updateMeResponse struct {
	Profile          ProfileView `json:"profile"`
	Privacy          PrivacyView `json:"privacy"`
	ModerationStatus string      `json:"moderation_status"`
	RowVersion       uint32      `json:"row_version"`
}

type logoutRequest struct {
	AllDevices bool `json:"all_devices"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/auth/register", h.requireOrigin(h.register))
	r.Post("/api/v1/auth/login", h.requireOrigin(h.login))

	r.Group(func(protected chi.Router) {
		protected.Use(h.SessionMiddleware())
		protected.Get("/api/v1/me", h.me)

		protected.Group(func(writes chi.Router) {
			writes.Use(h.CSRFMiddleware())
			writes.Post("/api/v1/auth/logout", h.logout)
			writes.Patch("/api/v1/me", h.updateMe)
		})
	})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}

	result, err := h.service.Register(r.Context(), input, requestMetadata(r))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	h.setSessionCookie(w, result.Session)

	httpjson.WriteSuccess(w, r, http.StatusCreated, sessionResponse{
		User:             result.User,
		Profile:          result.Profile,
		SessionExpiresAt: result.Session.ExpiresAt,
		CSRFToken:        result.Session.CSRFToken,
	}, result.IdempotentReplay)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}

	result, err := h.service.Login(r.Context(), input, requestMetadata(r))
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	h.setSessionCookie(w, result.Session)

	httpjson.WriteSuccess(w, r, http.StatusOK, sessionResponse{
		User:             result.User,
		Profile:          result.Profile,
		SessionExpiresAt: result.Session.ExpiresAt,
		CSRFToken:        result.Session.CSRFToken,
	}, false)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		httpjson.WriteError(w, r, apperror.New(
			"VALIDATION_ERROR",
			"Idempotency-Key is required.",
			false,
			nil,
		))
		return
	}

	input := logoutRequest{}
	if r.ContentLength > 0 {
		if err := httpjson.DecodeJSON(w, r, &input); err != nil {
			httpjson.WriteError(w, r, err)
			return
		}
	}

	count, err := h.service.Logout(r.Context(), session, input.AllDevices)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	h.clearSessionCookie(w)
	httpjson.WriteSuccess(w, r, http.StatusOK, map[string]int64{
		"revoked_session_count": count,
	}, false)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}

	view, err := h.service.Me(r.Context(), session)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, view, false)
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, r, authRequiredError())
		return
	}
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		httpjson.WriteError(w, r, apperror.New(
			"VALIDATION_ERROR",
			"Idempotency-Key is required.",
			false,
			nil,
		))
		return
	}

	var input UpdateMeInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	profile, privacy, err := h.service.UpdateMe(r.Context(), session, input)
	if err != nil {
		httpjson.WriteError(w, r, err)
		return
	}
	httpjson.WriteSuccess(w, r, http.StatusOK, updateMeResponse{
		Profile:          profile,
		Privacy:          privacy,
		ModerationStatus: profile.ModerationStatus,
		RowVersion:       profile.RowVersion,
	}, false)
}

func (h *Handler) SessionMiddleware() func(http.Handler) http.Handler {
	return h.sessionMiddleware
}

func (h *Handler) CSRFMiddleware() func(http.Handler) http.Handler {
	return h.csrfMiddleware
}

func (h *Handler) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.service.cfg.CookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			httpjson.WriteError(w, r, authRequiredError())
			return
		}

		session, err := h.service.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			httpjson.WriteError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessionFromContext(r.Context())
		if !ok {
			httpjson.WriteError(w, r, authRequiredError())
			return
		}
		if err := h.service.ValidateCSRF(session, r.Header.Get(csrfHeader)); err != nil {
			httpjson.WriteError(w, r, err)
			return
		}
		if err := h.service.validateOrigin(r.Header.Get("Origin")); err != nil {
			httpjson.WriteError(w, r, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) requireOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.service.validateOrigin(r.Header.Get("Origin")); err != nil {
			httpjson.WriteError(w, r, err)
			return
		}
		next(w, r)
	}
}

func (s *Service) validateOrigin(origin string) error {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return apperror.New("CSRF_INVALID", "Origin header is required.", false, nil)
	}
	for _, allowed := range s.cfg.AllowedOrigins {
		if origin == strings.TrimRight(allowed, "/") {
			return nil
		}
	}
	return apperror.New("CSRF_INVALID", "Origin is not allowed.", false, nil)
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, session SessionCredentials) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.service.cfg.CookieName,
		Value:    session.SessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.service.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
		MaxAge:   max(int(time.Until(session.ExpiresAt).Seconds()), 1),
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.service.cfg.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.service.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
	})
}

func sessionFromContext(ctx context.Context) (AuthenticatedSession, bool) {
	session, ok := ctx.Value(sessionContextKey).(AuthenticatedSession)
	return session, ok
}

func SessionFromContext(ctx context.Context) (AuthenticatedSession, bool) {
	return sessionFromContext(ctx)
}

func requestMetadata(r *http.Request) RequestMetadata {
	return RequestMetadata{
		IPAddress:      clientIP(r),
		UserAgent:      r.UserAgent(),
		DeviceLabel:    strings.TrimSpace(r.Header.Get("X-Device-Label")),
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
