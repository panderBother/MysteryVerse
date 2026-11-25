package room

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
)

func TestRoomLifecycleChatPermissionsAndConcurrency(t *testing.T) {
	repo, db := openRoomIntegrationRepository(t)
	cfg := roomTestConfig(time.Hour)
	service := NewService(repo, cfg, ratelimit.NewInMemory())
	service.now = func() time.Time {
		return time.Now().UTC()
	}
	ctx := context.Background()

	hostID := integrationRoomID()
	playerID := integrationRoomID()
	thirdID := integrationRoomID()
	outsiderID := integrationRoomID()
	for index, userID := range []string{hostID, playerID, thirdID, outsiderID} {
		seedRoomUser(t, db, userID, fmt.Sprintf("Player%d", index+1))
	}
	t.Cleanup(func() {
		cleanupRoomUsers(t, db, []string{hostID, playerID, thirdID, outsiderID})
	})

	scriptID, versionID := seedPlayableScript(t, db, hostID)
	t.Cleanup(func() {
		deleteRows(t, db, "DELETE FROM script_versions WHERE id = ?", versionID)
		deleteRows(t, db, "DELETE FROM scripts WHERE id = ?", scriptID)
	})

	var roomIDs []string
	t.Cleanup(func() {
		cleanupRooms(t, db, roomIDs)
	})

	_, err := service.Create(ctx, hostID, CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		Title:            "Invalid Capacity",
		Visibility:       "public",
		Capacity:         Capacity{Min: 1, Max: 9},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, integrationKey())
	if !hasRoomAppCode(err, "PLAYER_COUNT_MISMATCH") {
		t.Fatalf("Create(invalid capacity) error = %v", err)
	}

	create, err := service.Create(ctx, hostID, CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		Title:            "Integration Room",
		Visibility:       "public",
		Capacity:         Capacity{Min: 1, Max: 2},
		ModeConfig:       map[string]any{"search_rounds": 1},
		PrivacyConfirmed: true,
	}, integrationKey())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	roomIDs = append(roomIDs, create.Room.ID)

	joined, err := service.Join(ctx, playerID, create.Room.ID, JoinInput{}, integrationKey())
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}
	replay, err := service.Join(ctx, playerID, create.Room.ID, JoinInput{}, integrationKey())
	if err != nil {
		t.Fatalf("duplicate Join() error = %v", err)
	}
	if !replay.IdempotentReplay || replay.Member.ID != joined.Member.ID {
		t.Fatalf("duplicate Join() = %#v", replay)
	}

	projection, err := service.Projection(ctx, hostID, create.Room.ID)
	if err != nil {
		t.Fatalf("Projection() error = %v", err)
	}
	if len(projection.Members) != 2 {
		t.Fatalf("Projection members = %d, want 2", len(projection.Members))
	}

	_, err = service.Ready(ctx, outsiderID, create.Room.ID, ReadyInput{
		Ready:                true,
		ExpectedRoomEventSeq: projection.RoomEventSeq,
	}, integrationKey())
	if !hasRoomAppCode(err, "NOT_ROOM_MEMBER") {
		t.Fatalf("Ready(outsider) error = %v", err)
	}

	ready, err := service.Ready(ctx, playerID, create.Room.ID, ReadyInput{
		Ready:                true,
		ExpectedRoomEventSeq: projection.RoomEventSeq,
	}, integrationKey())
	if err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
	if ready.Member.MemberStatus != "ready" {
		t.Fatalf("Ready() member = %#v", ready.Member)
	}

	_, err = service.Join(ctx, thirdID, create.Room.ID, JoinInput{}, integrationKey())
	if !hasRoomAppCode(err, "ROOM_FULL") {
		t.Fatalf("Join(full) error = %v", err)
	}

	message, err := service.SendMessage(ctx, hostID, create.Room.ID, SendMessageInput{
		ClientMessageID: integrationRoomID(),
		Text:            "Are you ready?",
	}, integrationKey())
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if message.ChatSeq != 1 {
		t.Fatalf("SendMessage chat_seq = %d, want 1", message.ChatSeq)
	}
	history, err := service.Messages(ctx, playerID, create.Room.ID, nil, 30)
	if err != nil || len(history.Items) != 1 {
		t.Fatalf("Messages() = %d, %v", len(history.Items), err)
	}
	_, err = service.Messages(ctx, outsiderID, create.Room.ID, nil, 30)
	if !hasRoomAppCode(err, "NOT_ROOM_MEMBER") {
		t.Fatalf("Messages(outsider) error = %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO user_sanctions (
    id, user_id, sanction_type, scope, scope_ref_id, reason_code, reason_text,
    created_by, starts_at, expires_at
) VALUES (?, ?, 'mute', 'room', ?, 'test_mute', 'integration test', ?,
          UTC_TIMESTAMP(3), DATE_ADD(UTC_TIMESTAMP(3), INTERVAL 1 HOUR))
`, integrationRoomID(), playerID, create.Room.ID, hostID); err != nil {
		t.Fatalf("seed mute: %v", err)
	}
	_, err = service.SendMessage(ctx, playerID, create.Room.ID, SendMessageInput{
		ClientMessageID: integrationRoomID(),
		Text:            "This should be muted",
	}, integrationKey())
	if !hasRoomAppCode(err, "MUTED") {
		t.Fatalf("SendMessage(muted) error = %v", err)
	}

	projection, err = service.Projection(ctx, hostID, create.Room.ID)
	if err != nil {
		t.Fatalf("Projection(host) error = %v", err)
	}
	leave, err := service.Leave(ctx, hostID, create.Room.ID, LeaveInput{
		Reason:               "host_left",
		ExpectedRoomEventSeq: projection.RoomEventSeq,
	}, integrationKey())
	if err != nil {
		t.Fatalf("Leave(host) error = %v", err)
	}
	if !leave.HostTransferRequired || leave.RoomStatus != "waiting" {
		t.Fatalf("Leave(host) = %#v", leave)
	}
	playerProjection, err := service.Projection(ctx, playerID, create.Room.ID)
	if err != nil {
		t.Fatalf("Projection(new host) error = %v", err)
	}
	if playerProjection.Room.HostUserID != playerID {
		t.Fatalf("new host = %s, want %s", playerProjection.Room.HostUserID, playerID)
	}

	concurrentRoom, err := service.Create(ctx, hostID, CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		Title:            "Concurrent Room",
		Visibility:       "public",
		Capacity:         Capacity{Min: 1, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, integrationKey())
	if err != nil {
		t.Fatalf("Create(concurrent) error = %v", err)
	}
	roomIDs = append(roomIDs, concurrentRoom.Room.ID)

	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, userID := range []string{playerID, thirdID} {
		userID := userID
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := service.Join(ctx, userID, concurrentRoom.Room.ID, JoinInput{}, integrationKey())
			results <- err
		}()
	}
	wait.Wait()
	close(results)

	successes := 0
	full := 0
	for result := range results {
		switch {
		case result == nil:
			successes++
		case hasRoomAppCode(result, "ROOM_FULL"):
			full++
		default:
			t.Fatalf("concurrent Join() error = %v", result)
		}
	}
	if successes != 1 || full != 1 {
		t.Fatalf("concurrent join = %d success, %d full", successes, full)
	}

	concurrentProjection, err := service.Projection(ctx, hostID, concurrentRoom.Room.ID)
	if err != nil {
		t.Fatalf("Projection(concurrent) error = %v", err)
	}
	closed, err := service.HostAction(ctx, hostID, concurrentRoom.Room.ID, HostActionInput{
		Action:               "close",
		ExpectedRoomEventSeq: concurrentProjection.RoomEventSeq,
		Reason:               "test_done",
	}, integrationKey())
	if err != nil {
		t.Fatalf("HostAction(close) error = %v", err)
	}
	if closed.RoomStatus != "cancelled" {
		t.Fatalf("HostAction(close) status = %s", closed.RoomStatus)
	}

	startedRoom, err := service.Create(ctx, hostID, CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		Title:            "Started Room",
		Visibility:       "public",
		Capacity:         Capacity{Min: 1, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, integrationKey())
	if err != nil {
		t.Fatalf("Create(started) error = %v", err)
	}
	roomIDs = append(roomIDs, startedRoom.Room.ID)
	if _, err := db.SQL().Exec(
		"UPDATE game_rooms SET status = 'in_game' WHERE id = ?",
		startedRoom.Room.ID,
	); err != nil {
		t.Fatalf("mark room in_game: %v", err)
	}
	_, err = service.Join(ctx, thirdID, startedRoom.Room.ID, JoinInput{}, integrationKey())
	if !hasRoomAppCode(err, "ROOM_NOT_WAITING") {
		t.Fatalf("Join(started room) error = %v", err)
	}
}

func TestRoomExpiry(t *testing.T) {
	repo, db := openRoomIntegrationRepository(t)
	cfg := roomTestConfig(40 * time.Millisecond)
	service := NewService(repo, cfg, ratelimit.NewInMemory())
	ctx := context.Background()

	hostID := integrationRoomID()
	seedRoomUser(t, db, hostID, "ExpiryHost")
	t.Cleanup(func() {
		cleanupRoomUsers(t, db, []string{hostID})
	})
	scriptID, versionID := seedPlayableScript(t, db, hostID)
	t.Cleanup(func() {
		deleteRows(t, db, "DELETE FROM script_versions WHERE id = ?", versionID)
		deleteRows(t, db, "DELETE FROM scripts WHERE id = ?", scriptID)
	})

	created, err := service.Create(ctx, hostID, CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		Title:            "Expiring Room",
		Visibility:       "public",
		Capacity:         Capacity{Min: 1, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, integrationKey())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupRooms(t, db, []string{created.Room.ID})
	})

	time.Sleep(60 * time.Millisecond)
	_, err = service.Projection(ctx, hostID, created.Room.ID)
	if !hasRoomAppCode(err, "ROOM_CLOSED") {
		t.Fatalf("Projection(expired) error = %v", err)
	}
	_, err = service.Ready(ctx, hostID, created.Room.ID, ReadyInput{
		Ready:                true,
		ExpectedRoomEventSeq: created.RoomEventSeq,
	}, integrationKey())
	if !hasRoomAppCode(err, "ROOM_CLOSED") {
		t.Fatalf("Ready(expired) error = %v", err)
	}

	second, err := service.Create(ctx, hostID, CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		Title:            "Batch Expiring Room",
		Visibility:       "public",
		Capacity:         Capacity{Min: 1, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, integrationKey())
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	t.Cleanup(func() {
		cleanupRooms(t, db, []string{second.Room.ID})
	})
	time.Sleep(60 * time.Millisecond)
	closed, err := service.ExpireWaitingRooms(ctx)
	if err != nil {
		t.Fatalf("ExpireWaitingRooms() error = %v", err)
	}
	if closed < 1 {
		t.Fatalf("ExpireWaitingRooms() closed = %d", closed)
	}
}

func openRoomIntegrationRepository(t *testing.T) (*storage.MySQLRepository, *database.DB) {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not set")
	}
	db, err := database.Open(context.Background(), config.MySQLConfig{
		DSN:             dsn,
		MaxOpenConns:    12,
		MaxIdleConns:    6,
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

func roomTestConfig(waitingTTL time.Duration) config.AuthConfig {
	return config.AuthConfig{
		IdentityHMACKey:    strings.Repeat("r", 32),
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
		RoomWaitingTTL:     waitingTTL,
	}
}

func seedRoomUser(t *testing.T, db *database.DB, userID string, nickname string) {
	t.Helper()
	publicNo := "MV" + userID[len(userID)-10:]
	if _, err := db.SQL().Exec(`
INSERT INTO users (
    id, public_no, status, risk_level, terms_version, privacy_version
) VALUES (?, ?, 'active', 0, '2026-11', '2026-11')
`, userID, publicNo); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO user_profiles (
    id, user_id, nickname, avatar_uri, bio, locale, timezone,
    preferences_json, privacy_json, moderation_status, row_version
) VALUES (?, ?, ?, NULL, NULL, 'zh-CN', 'Asia/Shanghai', ?, ?, 'approved', 1)
`, integrationRoomID(), userID, nickname,
		[]byte(`{"schema_version":"1.0.0"}`),
		[]byte(`{"history_visibility":"private","online_visibility":"friends","profile_visibility":"public","friend_request_policy":"friends_of_friends"}`),
	); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
}

func seedPlayableScript(t *testing.T, db *database.DB, ownerID string) (string, string) {
	t.Helper()
	scriptID := integrationRoomID()
	versionID := integrationRoomID()
	if _, err := db.SQL().Exec(`
INSERT INTO scripts (
    id, owner_user_id, public_slug, title, summary, game_type, source_kind,
    rights_type, status, cover_asset_uri, language, content_rating,
    min_players, max_players, duration_minutes, difficulty, ai_assisted_ratio_bp
) VALUES (?, ?, NULL, 'Integration Script', NULL, 'script_murder', 'original',
          'original', 'published', NULL, 'zh-CN', '12+', 1, 8, 120, 'standard', 0)
`, scriptID, ownerID); err != nil {
		t.Fatalf("seed script: %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO script_versions (
    id, script_id, version_no, version_kind, status, schema_version,
    workflow_version, blueprint_json, validation_status, blocking_issue_count, created_by
) VALUES (?, ?, 1, 'release', 'published_public', '1.0.0', 'test-v1', ?,
          'passed', 0, ?)
`, versionID, scriptID, []byte(`{"schema_version":"1.0.0"}`), ownerID); err != nil {
		t.Fatalf("seed script version: %v", err)
	}
	return scriptID, versionID
}

func cleanupRooms(t *testing.T, db *database.DB, roomIDs []string) {
	t.Helper()
	for _, roomID := range roomIDs {
		deleteRows(t, db, "DELETE FROM idempotency_records WHERE resource_id = ?", roomID)
		deleteRows(t, db, "DELETE FROM user_sanctions WHERE scope_ref_id = ?", roomID)
		deleteRows(t, db, "DELETE FROM room_messages WHERE room_id = ?", roomID)
		deleteRows(t, db, "DELETE FROM invitation_tokens WHERE room_id = ?", roomID)
		deleteRows(t, db, "DELETE FROM room_events WHERE room_id = ?", roomID)
		deleteRows(t, db, "DELETE FROM room_members WHERE room_id = ?", roomID)
		deleteRows(t, db, "DELETE FROM game_rooms WHERE id = ?", roomID)
	}
}

func cleanupRoomUsers(t *testing.T, db *database.DB, userIDs []string) {
	t.Helper()
	for _, userID := range userIDs {
		deleteRows(t, db, "DELETE FROM user_profiles WHERE user_id = ?", userID)
		deleteRows(t, db, "DELETE FROM users WHERE id = ?", userID)
	}
}

func deleteRows(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.SQL().Exec(query, args...); err != nil {
		t.Errorf("cleanup %q: %v", query, err)
	}
}

func integrationRoomID() string {
	return id.ULIDGenerator{}.New()
}

func integrationKey() string {
	return id.ULIDGenerator{}.New()
}

func hasRoomAppCode(err error, code string) bool {
	var appErr apperror.Error
	return errors.As(err, &appErr) && appErr.Code == code
}
