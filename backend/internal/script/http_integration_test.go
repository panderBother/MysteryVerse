package script

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mysteryverse/mysteryverse/backend/internal/identity"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
)

func TestHTTPScriptCreateAndEditorRoutes(t *testing.T) {
	repo, db := openScriptIntegrationRepository(t)
	cfg := scriptHTTPAuthConfig()
	limiter := ratelimit.NewInMemory()
	identityService, err := identity.NewService(repo, cfg, limiter)
	if err != nil {
		t.Fatalf("identity.NewService() error = %v", err)
	}
	identityHandler := identity.NewHandler(identityService)
	scriptHandler := NewHandler(NewService(repo))
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	identityHandler.RegisterRoutes(router)
	scriptHandler.RegisterRoutes(router, identityHandler)

	session := registerScriptHTTPUser(t, router)
	t.Cleanup(func() {
		cleanupScriptHTTPUser(t, db, session.userID)
	})

	create := performScriptHTTPRequest(t, router, http.MethodPost, "/api/v1/scripts", CreateScriptInput{
		Title:           "HTTP Script",
		GameType:        "script_murder",
		RightsType:      "original",
		Language:        "zh-CN",
		TargetPlayers:   TargetPlayers{Min: 1, Max: 2},
		DurationMinutes: 60,
		Difficulty:      "simple",
		ContentRating:   "all",
	}, session.cookie, session.csrf, true)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}
	var createResponse struct {
		Data CreateScriptResult `json:"data"`
	}
	decodeScriptHTTPResponse(t, create, &createResponse)
	scriptID := createResponse.Data.Script.ID
	versionID := createResponse.Data.Version.ID

	editor := performScriptHTTPRequest(
		t,
		router,
		http.MethodGet,
		"/api/v1/scripts/"+scriptID+"/versions/"+versionID+"/editor",
		nil,
		session.cookie,
		"",
		false,
	)
	if editor.Code != http.StatusOK {
		t.Fatalf("editor status = %d, body = %s", editor.Code, editor.Body.String())
	}
	publicList := performScriptHTTPRequest(
		t,
		router,
		http.MethodGet,
		"/api/v1/scripts?game_type=script_murder&limit=20",
		nil,
		nil,
		"",
		false,
	)
	if publicList.Code != http.StatusOK {
		t.Fatalf("public list status = %d, body = %s", publicList.Code, publicList.Body.String())
	}
}

type scriptHTTPSession struct {
	userID string
	cookie *http.Cookie
	csrf   string
}

func scriptHTTPAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		IdentityHMACKey:    strings.Repeat("s", 32),
		CookieName:         "mv_session",
		SessionTTL:         time.Hour,
		RememberSessionTTL: 24 * time.Hour,
		AllowedOrigins:     []string{"http://localhost:5173"},
		RegisterRateLimit:  100,
		RegisterRateWindow: time.Minute,
		LoginRateLimit:     100,
		LoginRateWindow:    time.Minute,
		ChatRateLimit:      100,
		ChatRateWindow:     time.Minute,
		ChatMaxLength:      4000,
		RoomWaitingTTL:     time.Hour,
	}
}

func registerScriptHTTPUser(t *testing.T, handler http.Handler) scriptHTTPSession {
	t.Helper()
	email := "script-http+" + strings.ToLower(id.ULIDGenerator{}.New()) + "@example.com"
	recorder := performScriptHTTPRequest(t, handler, http.MethodPost, "/api/v1/auth/register", identity.RegisterInput{
		Provider:       "email",
		Identifier:     email,
		Password:       "correct-password",
		Nickname:       "Script HTTP",
		TermsVersion:   "2026-11",
		PrivacyVersion: "2026-11",
	}, nil, "", true)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
			CSRFToken string `json:"csrf_token"`
		} `json:"data"`
	}
	decodeScriptHTTPResponse(t, recorder, &response)
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("register response did not set a session cookie")
	}
	return scriptHTTPSession{
		userID: response.Data.User.ID,
		cookie: cookies[0],
		csrf:   response.Data.CSRFToken,
	}
}

func performScriptHTTPRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body any,
	cookie *http.Cookie,
	csrf string,
	idempotentWrite bool,
) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		payload = raw
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Origin", "http://localhost:5173")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	if idempotentWrite {
		request.Header.Set("Idempotency-Key", id.ULIDGenerator{}.New())
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeScriptHTTPResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	destination any,
) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, recorder.Body.String())
	}
}

func cleanupScriptHTTPUser(t *testing.T, db *database.DB, userID string) {
	t.Helper()
	execDelete(t, db, `
DELETE p FROM script_publications p
JOIN scripts s ON s.id = p.script_id
WHERE s.owner_user_id = ?`, userID)
	execDelete(t, db, `
UPDATE script_versions sv
JOIN scripts s ON s.id = sv.script_id
SET sv.base_version_id = NULL
WHERE s.owner_user_id = ?`, userID)
	execDelete(t, db, `
DELETE sv FROM script_versions sv
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, userID)
	execDelete(t, db, "DELETE FROM scripts WHERE owner_user_id = ?", userID)
	execDelete(t, db, "DELETE FROM idempotency_records WHERE actor_scope = ?", "user:"+userID)
	execDelete(t, db, "DELETE FROM user_sessions WHERE user_id = ?", userID)
	execDelete(t, db, "DELETE FROM user_statistics WHERE user_id = ?", userID)
	execDelete(t, db, "DELETE FROM user_profiles WHERE user_id = ?", userID)
	execDelete(t, db, "DELETE FROM auth_identities WHERE user_id = ?", userID)
	execDelete(t, db, "DELETE FROM users WHERE id = ?", userID)
}
