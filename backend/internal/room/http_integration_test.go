package room

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mysteryverse/mysteryverse/backend/internal/identity"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
)

func TestHTTPUserRoomAndWaitingChatFlow(t *testing.T) {
	repo, db := openRoomIntegrationRepository(t)
	cfg := roomTestConfig(time.Hour)
	limiter := ratelimit.NewInMemory()
	identityService, err := identity.NewService(repo, cfg, limiter)
	if err != nil {
		t.Fatalf("identity.NewService() error = %v", err)
	}
	identityHandler := identity.NewHandler(identityService)
	roomHandler := NewHandler(NewService(repo, cfg, limiter))

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	identityHandler.RegisterRoutes(router)
	roomHandler.RegisterRoutes(router, identityHandler)

	firstSession := registerHTTPUser(t, router, "HTTPAlice", "http-alice")
	secondSession := registerHTTPUser(t, router, "HTTPBob", "http-bob")
	t.Cleanup(func() {
		cleanupHTTPIdentity(t, db, firstSession.userID)
		cleanupHTTPIdentity(t, db, secondSession.userID)
	})

	scriptID, versionID := seedPlayableScript(t, db, firstSession.userID)
	t.Cleanup(func() {
		deleteRows(t, db, "DELETE FROM script_versions WHERE id = ?", versionID)
		deleteRows(t, db, "DELETE FROM scripts WHERE id = ?", scriptID)
	})

	createRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/rooms", CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		Title:            "HTTP Room",
		Visibility:       "public",
		Capacity:         Capacity{Min: 1, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, firstSession.cookie, firstSession.csrf, true)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create room status = %d, body = %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createResponse struct {
		Data CreateResult `json:"data"`
	}
	decodeResponse(t, createRecorder, &createResponse)
	roomID := createResponse.Data.Room.ID
	t.Cleanup(func() {
		cleanupRooms(t, db, []string{roomID})
	})

	joinRecorder := performJSONRequest(t, router, http.MethodPost,
		"/api/v1/rooms/"+roomID+"/join", JoinInput{}, secondSession.cookie, secondSession.csrf, true)
	if joinRecorder.Code != http.StatusOK {
		t.Fatalf("join room status = %d, body = %s", joinRecorder.Code, joinRecorder.Body.String())
	}

	projectionRecorder := performJSONRequest(t, router, http.MethodGet,
		"/api/v1/rooms/"+roomID, nil, secondSession.cookie, "", false)
	if projectionRecorder.Code != http.StatusOK {
		t.Fatalf("projection status = %d, body = %s", projectionRecorder.Code, projectionRecorder.Body.String())
	}
	var projectionResponse struct {
		Data Projection `json:"data"`
	}
	decodeResponse(t, projectionRecorder, &projectionResponse)
	if len(projectionResponse.Data.Members) != 2 {
		t.Fatalf("projection members = %d, want 2", len(projectionResponse.Data.Members))
	}

	invalidCSRF := performJSONRequest(t, router, http.MethodPost,
		"/api/v1/rooms/"+roomID+"/messages",
		SendMessageInput{ClientMessageID: id.ULIDGenerator{}.New(), Text: "Blocked"},
		secondSession.cookie,
		"invalid-csrf-token",
		true,
	)
	if invalidCSRF.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status = %d, body = %s", invalidCSRF.Code, invalidCSRF.Body.String())
	}

	messageRecorder := performJSONRequest(t, router, http.MethodPost,
		"/api/v1/rooms/"+roomID+"/messages",
		SendMessageInput{ClientMessageID: id.ULIDGenerator{}.New(), Text: "Hello room"},
		secondSession.cookie,
		secondSession.csrf,
		true,
	)
	if messageRecorder.Code != http.StatusOK {
		t.Fatalf("send message status = %d, body = %s", messageRecorder.Code, messageRecorder.Body.String())
	}

	historyRecorder := performJSONRequest(t, router, http.MethodGet,
		"/api/v1/rooms/"+roomID+"/messages", nil, firstSession.cookie, "", false)
	if historyRecorder.Code != http.StatusOK {
		t.Fatalf("history status = %d, body = %s", historyRecorder.Code, historyRecorder.Body.String())
	}
	var historyResponse struct {
		Data MessagePage `json:"data"`
	}
	decodeResponse(t, historyRecorder, &historyResponse)
	if len(historyResponse.Data.Items) != 1 {
		t.Fatalf("messages = %d, want 1", len(historyResponse.Data.Items))
	}
}

type httpSession struct {
	userID string
	cookie *http.Cookie
	csrf   string
}

func registerHTTPUser(t *testing.T, handler http.Handler, nickname string, prefix string) httpSession {
	t.Helper()
	email := prefix + "+" + strings.ToLower(id.ULIDGenerator{}.New()) + "@example.com"
	recorder := performJSONRequest(t, handler, http.MethodPost, "/api/v1/auth/register", identity.RegisterInput{
		Provider:       "email",
		Identifier:     email,
		Password:       "correct-password",
		Nickname:       nickname,
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
	decodeResponse(t, recorder, &response)
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("register response did not set a session cookie")
	}
	return httpSession{
		userID: response.Data.User.ID,
		cookie: cookies[0],
		csrf:   response.Data.CSRFToken,
	}
}

func performJSONRequest(
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
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
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

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, recorder.Body.String())
	}
}

func cleanupHTTPIdentity(t *testing.T, db *database.DB, userID string) {
	t.Helper()
	queries := []string{
		"DELETE FROM idempotency_records WHERE resource_id = ?",
		"DELETE FROM user_sessions WHERE user_id = ?",
		"DELETE FROM user_statistics WHERE user_id = ?",
		"DELETE FROM user_profiles WHERE user_id = ?",
		"DELETE FROM auth_identities WHERE user_id = ?",
		"DELETE FROM users WHERE id = ?",
	}
	for _, query := range queries {
		deleteRows(t, db, query, userID)
	}
}

func Example() {
	fmt.Println("POST /api/v1/rooms -> POST /api/v1/rooms/{room_id}/join -> POST /api/v1/rooms/{room_id}/messages")
	// Output:
	// POST /api/v1/rooms -> POST /api/v1/rooms/{room_id}/join -> POST /api/v1/rooms/{room_id}/messages
}
