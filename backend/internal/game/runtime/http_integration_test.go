package runtime_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	gameruntime "github.com/mysteryverse/mysteryverse/backend/internal/game/runtime"
	"github.com/mysteryverse/mysteryverse/backend/internal/identity"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/secret"
	"github.com/mysteryverse/mysteryverse/backend/internal/room"
)

func TestHTTPGameStartAndProjection(t *testing.T) {
	repo, db := openGameIntegrationRepository(t)
	cfg := gameAuthConfig()
	limiter := ratelimit.NewInMemory()
	identityService, err := identity.NewService(repo, cfg, limiter)
	if err != nil {
		t.Fatalf("identity.NewService() error = %v", err)
	}
	identityHandler := identity.NewHandler(identityService)
	roomService := room.NewService(repo, cfg, limiter)
	roomHandler := room.NewHandler(roomService)
	cipher, err := secret.NewCipher("task6-game-secret-key-material", "task6-key")
	if err != nil {
		t.Fatalf("secret.NewCipher() error = %v", err)
	}
	gameService := gameruntime.NewService(repo, room.NewStartCoordinator(), cipher)
	gameHandler := gameruntime.NewHandler(gameService)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	identityHandler.RegisterRoutes(router)
	roomHandler.RegisterRoutes(router, identityHandler)
	gameHandler.RegisterRoutes(router, identityHandler)

	host := registerGameHTTPUser(t, router, "GameHTTPHost", "game-http-host")
	player := registerGameHTTPUser(t, router, "GameHTTPPlayer", "game-http-player")
	t.Cleanup(func() {
		cleanupGameHTTPIdentity(t, db, []string{host.userID, player.userID})
	})
	ruleID, scriptID, versionID, phaseIDs := seedMurderGame(t, db, host.userID)
	t.Cleanup(func() {
		cleanupMurderContent(t, db, ruleID, scriptID, versionID, phaseIDs)
	})

	create := performGameHTTP(t, router, http.MethodPost, "/api/v1/rooms", room.CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		RuleTemplateID:   &ruleID,
		Title:            "HTTP Game",
		Visibility:       "public",
		Capacity:         room.Capacity{Min: 2, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, host.cookie, host.csrf, true)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}
	var createBody struct {
		Data room.CreateResult `json:"data"`
	}
	decodeGameHTTP(t, create, &createBody)
	roomID := createBody.Data.Room.ID
	t.Cleanup(func() {
		cleanupGameRooms(t, db, []string{roomID})
	})

	join := performGameHTTP(t, router, http.MethodPost,
		"/api/v1/rooms/"+roomID+"/join", room.JoinInput{}, player.cookie, player.csrf, true)
	if join.Code != http.StatusOK {
		t.Fatalf("join status = %d, body = %s", join.Code, join.Body.String())
	}
	var joinBody struct {
		Data room.JoinResult `json:"data"`
	}
	decodeGameHTTP(t, join, &joinBody)

	readyHost := performGameHTTP(t, router, http.MethodPost,
		"/api/v1/rooms/"+roomID+"/ready", room.ReadyInput{
			Ready:                true,
			ExpectedRoomEventSeq: joinBody.Data.RoomEventSeq,
		}, host.cookie, host.csrf, true)
	if readyHost.Code != http.StatusOK {
		t.Fatalf("ready host status = %d, body = %s", readyHost.Code, readyHost.Body.String())
	}
	var readyHostBody struct {
		Data room.ReadyResult `json:"data"`
	}
	decodeGameHTTP(t, readyHost, &readyHostBody)

	readyPlayer := performGameHTTP(t, router, http.MethodPost,
		"/api/v1/rooms/"+roomID+"/ready", room.ReadyInput{
			Ready:                true,
			ExpectedRoomEventSeq: readyHostBody.Data.RoomEventSeq,
		}, player.cookie, player.csrf, true)
	if readyPlayer.Code != http.StatusOK {
		t.Fatalf("ready player status = %d, body = %s", readyPlayer.Code, readyPlayer.Body.String())
	}
	var readyPlayerBody struct {
		Data room.ReadyResult `json:"data"`
	}
	decodeGameHTTP(t, readyPlayer, &readyPlayerBody)

	start := performGameHTTP(t, router, http.MethodPost,
		"/api/v1/rooms/"+roomID+"/start", gameruntime.StartInput{
			ExpectedRoomEventSeq: readyPlayerBody.Data.RoomEventSeq,
			ContentVersionID:     versionID,
			RulesAck:             true,
		}, host.cookie, host.csrf, true)
	if start.Code != http.StatusCreated {
		t.Fatalf("start status = %d, body = %s", start.Code, start.Body.String())
	}
	var startBody struct {
		Data gameruntime.StartResult `json:"data"`
	}
	decodeGameHTTP(t, start, &startBody)
	t.Cleanup(func() {
		cleanupGameRuntime(t, db, startBody.Data.GameID)
	})

	projection := performGameHTTP(t, router, http.MethodGet,
		"/api/v1/games/"+startBody.Data.GameID+"/projection", nil, player.cookie, "", false)
	if projection.Code != http.StatusOK {
		t.Fatalf("projection status = %d, body = %s", projection.Code, projection.Body.String())
	}
	var projectionBody struct {
		Data gameruntime.Projection `json:"data"`
	}
	decodeGameHTTP(t, projection, &projectionBody)
	if projectionBody.Data.GameID != startBody.Data.GameID ||
		projectionBody.Data.Viewer.PlayerID == "" {
		t.Fatalf("projection = %#v", projectionBody.Data)
	}
}

type gameHTTPUser struct {
	userID string
	cookie *http.Cookie
	csrf   string
}

func registerGameHTTPUser(
	t *testing.T,
	handler http.Handler,
	nickname string,
	prefix string,
) gameHTTPUser {
	t.Helper()
	email := prefix + "+" + strings.ToLower(id.ULIDGenerator{}.New()) + "@example.com"
	recorder := performGameHTTP(t, handler, http.MethodPost, "/api/v1/auth/register", identity.RegisterInput{
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
	decodeGameHTTP(t, recorder, &response)
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("register response did not set a cookie")
	}
	return gameHTTPUser{
		userID: response.Data.User.ID,
		cookie: cookies[0],
		csrf:   response.Data.CSRFToken,
	}
}

func performGameHTTP(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body any,
	cookie *http.Cookie,
	csrf string,
	write bool,
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
	request.Header.Set("Origin", "http://localhost:5173")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	if write {
		request.Header.Set("Idempotency-Key", gameKey())
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeGameHTTP(t *testing.T, recorder *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, recorder.Body.String())
	}
}

func cleanupGameHTTPIdentity(t *testing.T, db *database.DB, userIDs []string) {
	t.Helper()
	for _, userID := range userIDs {
		queries := []string{
			"DELETE FROM idempotency_records WHERE resource_id = ?",
			"DELETE FROM user_sessions WHERE user_id = ?",
			"DELETE FROM user_statistics WHERE user_id = ?",
			"DELETE FROM user_profiles WHERE user_id = ?",
			"DELETE FROM auth_identities WHERE user_id = ?",
			"DELETE FROM users WHERE id = ?",
		}
		for _, query := range queries {
			deleteGameRows(t, db, query, userID)
		}
	}
}
