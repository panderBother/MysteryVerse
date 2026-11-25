package identity

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
)

func TestIdentityRegistrationLoginProfileAndLogout(t *testing.T) {
	repo, db := openIdentityIntegrationRepository(t)
	service := newIdentityIntegrationService(t, repo)
	ctx := context.Background()

	suffix := id.ULIDGenerator{}.New()
	identifier := "alice+" + strings.ToLower(suffix) + "@example.com"
	registerInput := RegisterInput{
		Provider:       "email",
		Identifier:     identifier,
		Password:       "correct-password",
		Nickname:       "Alice",
		TermsVersion:   "2026-11",
		PrivacyVersion: "2026-11",
	}
	idempotencyKey := id.ULIDGenerator{}.New()
	meta := RequestMetadata{
		IPAddress:      "127.0.0.1",
		UserAgent:      "integration-test",
		DeviceLabel:    "test",
		IdempotencyKey: idempotencyKey,
	}

	first, err := service.Register(ctx, registerInput, meta)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupIdentityUser(t, db, first.User.ID)
	})
	if first.Session.SessionToken == "" || first.Session.CSRFToken == "" {
		t.Fatal("Register() did not return Session and CSRF credentials")
	}

	replay, err := service.Register(ctx, registerInput, meta)
	if err != nil {
		t.Fatalf("Register() replay error = %v", err)
	}
	if !replay.IdempotentReplay || replay.User.ID != first.User.ID {
		t.Fatalf("Register() replay = %#v", replay)
	}

	_, err = service.Register(ctx, registerInput, RequestMetadata{
		IPAddress:      "127.0.0.1",
		IdempotencyKey: id.ULIDGenerator{}.New(),
	})
	if !hasAppCode(err, "IDENTIFIER_ALREADY_EXISTS") {
		t.Fatalf("duplicate Register() error = %v", err)
	}

	authenticated, err := service.Authenticate(ctx, first.Session.SessionToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if err := service.ValidateCSRF(authenticated, first.Session.CSRFToken); err != nil {
		t.Fatalf("ValidateCSRF() error = %v", err)
	}

	me, err := service.Me(ctx, authenticated)
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if me.User.ID != first.User.ID || me.Profile.Nickname != "Alice" {
		t.Fatalf("Me() = %#v", me)
	}

	newNickname := "Alice Updated"
	profile, privacy, err := service.UpdateMe(ctx, authenticated, UpdateMeInput{
		ProfilePatch: &ProfilePatch{Nickname: &newNickname},
		PrivacyPatch: &PrivacyPatch{HistoryVisibility: stringPointer("friends")},
		RowVersion:   me.Profile.RowVersion,
	})
	if err != nil {
		t.Fatalf("UpdateMe() error = %v", err)
	}
	if profile.Nickname != newNickname || privacy.HistoryVisibility != "friends" {
		t.Fatalf("UpdateMe() = %#v, %#v", profile, privacy)
	}

	_, err = service.Login(ctx, LoginInput{
		Provider:   "email",
		Identifier: identifier,
		Password:   "wrong-password",
	}, meta)
	if !hasAppCode(err, "AUTH_INVALID") {
		t.Fatalf("failed Login() error = %v", err)
	}

	login, err := service.Login(ctx, LoginInput{
		Provider:       "email",
		Identifier:     identifier,
		Password:       "correct-password",
		RememberDevice: true,
	}, meta)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if login.User.ID != first.User.ID {
		t.Fatalf("Login() user = %s, want %s", login.User.ID, first.User.ID)
	}

	loginSession, err := service.Authenticate(ctx, login.Session.SessionToken)
	if err != nil {
		t.Fatalf("Authenticate(login) error = %v", err)
	}
	revoked, err := service.Logout(ctx, loginSession, false)
	if err != nil || revoked != 1 {
		t.Fatalf("Logout() = %d, %v", revoked, err)
	}
	if _, err := service.Authenticate(ctx, login.Session.SessionToken); !hasAppCode(err, "SESSION_REVOKED") {
		t.Fatalf("Authenticate(revoked) error = %v", err)
	}
}

func TestIdentityMiddlewareRequiresLogin(t *testing.T) {
	repo, _ := openIdentityIntegrationRepository(t)
	service := newIdentityIntegrationService(t, repo)
	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func openIdentityIntegrationRepository(t *testing.T) (*storage.MySQLRepository, *database.DB) {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not set")
	}
	db, err := database.Open(context.Background(), config.MySQLConfig{
		DSN:             dsn,
		MaxOpenConns:    8,
		MaxIdleConns:    4,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
		PingTimeout:     2 * time.Second,
	})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return storage.NewMySQLRepository(db), db
}

func newIdentityIntegrationService(t *testing.T, repo storage.Repository) *Service {
	t.Helper()
	cfg := config.AuthConfig{
		IdentityHMACKey:    strings.Repeat("k", 32),
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
	service, err := NewService(repo, cfg, ratelimit.NewInMemory())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func cleanupIdentityUser(t *testing.T, db *database.DB, userID string) {
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
		if _, err := db.SQL().Exec(query, userID); err != nil {
			t.Errorf("cleanup %q: %v", query, err)
		}
	}
}

func hasAppCode(err error, code string) bool {
	var appErr apperror.Error
	return errors.As(err, &appErr) && appErr.Code == code
}

func stringPointer(value string) *string {
	return &value
}
