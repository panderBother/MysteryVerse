package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

var errRollback = errors.New("intentional test rollback")

func TestRepositoryTransactionRollbackAndUniqueConstraint(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()

	userID := testID(101)
	secondUserID := testID(102)
	publicNo := testPublicNo(101)

	err := repo.WithTransaction(ctx, func(ctx context.Context, tx Repository) error {
		if err := tx.CreateUser(ctx, newUserParams(userID, publicNo)); err != nil {
			return err
		}
		if _, err := tx.GetUser(ctx, userID); err != nil {
			return err
		}

		err := tx.CreateUser(ctx, newUserParams(secondUserID, publicNo))
		var storageErr *database.Error
		if !errors.As(err, &storageErr) || storageErr.Code != database.CodeConflict {
			return fmt.Errorf("duplicate user error = %v, want conflict", err)
		}

		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("WithTransaction() error = %v, want rollback sentinel", err)
	}

	_, err = repo.GetUser(ctx, userID)
	var storageErr *database.Error
	if !errors.As(err, &storageErr) || storageErr.Code != database.CodeNotFound {
		t.Fatalf("GetUser() after rollback error = %v, want not found", err)
	}
}

func TestRepositoryCommitAndRead(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()

	userID := testID(110)
	if err := repo.CreateUser(ctx, newUserParams(userID, testPublicNo(110))); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	t.Cleanup(func() {
		deleteRows(t, db, "DELETE FROM users WHERE id = ?", userID)
	})

	user, err := repo.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if user.ID != userID || user.Status != "active" {
		t.Fatalf("user = %#v", user)
	}
}

func TestRepositoryRoomGameEventMemoryAndConcurrentVersion(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()

	hostID := testID(201)
	scriptID := testID(202)
	scriptVersionID := testID(203)
	roomID := testID(204)
	memberID := testID(205)
	gameID := testID(206)
	eventID := testID(207)
	npcProfileID := testID(208)
	gameNPCID := testID(209)
	memorySecretID := testID(210)
	memoryID := testID(211)

	if err := repo.CreateUser(ctx, newUserParams(hostID, testPublicNo(201))); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if err := repo.CreateScript(ctx, newScriptParams(scriptID, hostID)); err != nil {
		t.Fatalf("CreateScript() error = %v", err)
	}
	seedScriptVersion(t, db, scriptVersionID, scriptID, hostID)
	seedNPCProfile(t, db, npcProfileID, scriptVersionID, memorySecretID)

	if err := repo.CreateGameRoom(ctx, newRoomParams(roomID, hostID, scriptVersionID)); err != nil {
		t.Fatalf("CreateGameRoom() error = %v", err)
	}
	if err := repo.CreateRoomMember(ctx, newRoomMemberParams(memberID, roomID, hostID)); err != nil {
		t.Fatalf("CreateRoomMember() error = %v", err)
	}
	if err := repo.CreateGameSession(ctx, newGameSessionParams(gameID, roomID, scriptVersionID)); err != nil {
		t.Fatalf("CreateGameSession() error = %v", err)
	}
	seedGameNPC(t, db, gameNPCID, gameID, npcProfileID)

	t.Cleanup(func() {
		deleteRows(t, db, "DELETE FROM npc_memories WHERE id = ?", memoryID)
		deleteRows(t, db, "DELETE FROM game_npcs WHERE id = ?", gameNPCID)
		deleteRows(t, db, "DELETE FROM game_events WHERE id = ?", eventID)
		deleteRows(t, db, "DELETE FROM game_sessions WHERE id = ?", gameID)
		deleteRows(t, db, "DELETE FROM room_members WHERE id = ?", memberID)
		deleteRows(t, db, "DELETE FROM game_rooms WHERE id = ?", roomID)
		deleteRows(t, db, "DELETE FROM npc_profiles WHERE id = ?", npcProfileID)
		deleteRows(t, db, "DELETE FROM secret_payloads WHERE id = ?", memorySecretID)
		deleteRows(t, db, "DELETE FROM script_versions WHERE id = ?", scriptVersionID)
		deleteRows(t, db, "DELETE FROM scripts WHERE id = ?", scriptID)
		deleteRows(t, db, "DELETE FROM users WHERE id = ?", hostID)
	})

	members, err := repo.ListRoomMembers(ctx, roomID)
	if err != nil || len(members) != 1 {
		t.Fatalf("ListRoomMembers() = %d, %v", len(members), err)
	}
	seatCount, err := repo.CountActiveRoomSeats(ctx, roomID)
	if err != nil || seatCount != 1 {
		t.Fatalf("CountActiveRoomSeats() = %d, %v", seatCount, err)
	}

	if err := repo.CreateGameEvent(ctx, newGameEventParams(eventID, gameID)); err != nil {
		t.Fatalf("CreateGameEvent() error = %v", err)
	}
	events, err := repo.ListGameEventsAfter(ctx, sqlc.ListGameEventsAfterParams{
		GameSessionID: gameID,
		Seq:           0,
		Limit:         10,
	})
	if err != nil || len(events) != 1 {
		t.Fatalf("ListGameEventsAfter() = %d, %v", len(events), err)
	}

	if err := repo.CreateNPCMemory(ctx, newMemoryParams(memoryID, gameID, gameNPCID, memorySecretID, eventID)); err != nil {
		t.Fatalf("CreateNPCMemory() error = %v", err)
	}
	memories, err := repo.ListNPCMemories(ctx, gameID, gameNPCID, time.Now().UTC(), 10)
	if err != nil || len(memories) != 1 {
		t.Fatalf("ListNPCMemories() = %d, %v", len(memories), err)
	}

	update := sqlc.UpdateGameSessionVersionParams{
		StateVersion:    1,
		CurrentEventSeq: 1,
		StateHash:       []byte("01234567890123456789012345678901"),
		ID:              gameID,
		StateVersion_2:  0,
	}
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- repo.UpdateGameSessionVersion(ctx, update)
		}()
	}
	wait.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for result := range results {
		switch {
		case result == nil:
			successes++
		case errors.Is(result, ErrNoRowsAffected):
			conflicts++
		default:
			t.Fatalf("UpdateGameSessionVersion() unexpected error = %v", result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent update = %d success, %d conflict", successes, conflicts)
	}
}

func TestAuditTriggerRejectsUpdateAndDelete(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()
	auditID := testID(301)

	err := db.WithTransaction(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO audit_logs (
    id, actor_type, actor_id, action, object_type, object_id, created_at
) VALUES (?, 'service', ?, 'test.append', 'test', ?, UTC_TIMESTAMP(3))
`, auditID, testID(302), testID(303)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE audit_logs SET reason = 'changed' WHERE id = ?", auditID); err == nil {
			return errors.New("audit UPDATE unexpectedly succeeded")
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM audit_logs WHERE id = ?", auditID); err == nil {
			return errors.New("audit DELETE unexpectedly succeeded")
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("audit transaction error = %v, want rollback sentinel", err)
	}
}

func openIntegrationDB(t *testing.T) *database.DB {
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
	return db
}

func newUserParams(id string, publicNo string) sqlc.CreateUserParams {
	return sqlc.CreateUserParams{
		ID:             id,
		PublicNo:       publicNo,
		Status:         "active",
		RiskLevel:      0,
		TermsVersion:   "2026-11",
		PrivacyVersion: "2026-11",
	}
}

func newScriptParams(id string, ownerID string) sqlc.CreateScriptParams {
	return sqlc.CreateScriptParams{
		ID:                id,
		OwnerUserID:       ownerID,
		Title:             "Test Script",
		GameType:          "script_murder",
		SourceKind:        "original",
		RightsType:        "original",
		Status:            "draft",
		Language:          "zh-CN",
		ContentRating:     "12+",
		MinPlayers:        3,
		MaxPlayers:        6,
		DurationMinutes:   120,
		Difficulty:        "standard",
		AiAssistedRatioBp: 0,
	}
}

func newRoomParams(id string, hostID string, scriptVersionID string) sqlc.CreateGameRoomParams {
	return sqlc.CreateGameRoomParams{
		ID:                      id,
		RoomCodeHash:            bytes32(),
		HostUserID:              hostID,
		ScriptVersionID:         scriptVersionID,
		GameType:                "script_murder",
		Title:                   "Test Room",
		Visibility:              "private",
		CapacityMin:             3,
		CapacityMax:             6,
		ModeConfigJson:          []byte(`{"schema_version":"1.0.0"}`),
		ModeConfigSchemaVersion: "1.0.0",
		PrivacyPolicyVersion:    "2026-11",
		Status:                  "waiting",
	}
}

func newRoomMemberParams(id string, roomID string, userID string) sqlc.CreateRoomMemberParams {
	return sqlc.CreateRoomMemberParams{
		ID:              id,
		RoomID:          roomID,
		UserID:          sqlNullString(userID),
		ParticipantKind: "human",
		SeatNo:          sqlNullInt16(1),
		RoomRole:        "host",
		MemberStatus:    "joined",
		JoinedAt:        time.Now().UTC(),
	}
}

func newGameSessionParams(id string, roomID string, scriptVersionID string) sqlc.CreateGameSessionParams {
	return sqlc.CreateGameSessionParams{
		ID:                id,
		RoomID:            roomID,
		GameType:          "script_murder",
		ScriptVersionID:   scriptVersionID,
		EngineKey:         "script_murder",
		EngineVersion:     "test-v1",
		PackageUri:        "s3://test/package.json",
		PackageHash:       bytes32(),
		SchemaVersion:     "1.0.0",
		Status:            "starting",
		StateVersion:      0,
		CurrentEventSeq:   0,
		StateHash:         bytes32(),
		OwnerFencingToken: 0,
		StartedAt:         time.Now().UTC(),
	}
}

func newGameEventParams(id string, gameID string) sqlc.CreateGameEventParams {
	now := time.Now().UTC()
	return sqlc.CreateGameEventParams{
		ID:              id,
		GameSessionID:   gameID,
		Seq:             1,
		EventType:       "game.created",
		EventCategory:   "room",
		VisibilityScope: "public",
		Sensitivity:     0,
		PayloadJson:     jsonRaw(`{"schema_version":"1.0.0"}`),
		PayloadHash:     bytes32(),
		SchemaVersion:   "1.0.0",
		OccurredAt:      now,
		RecordedAt:      now,
	}
}

func newMemoryParams(
	id string,
	gameID string,
	gameNPCID string,
	secretID string,
	eventID string,
) sqlc.CreateNPCMemoryParams {
	now := time.Now().UTC()
	return sqlc.CreateNPCMemoryParams{
		ID:                     id,
		GameSessionID:          gameID,
		GameNpcID:              gameNPCID,
		OwnerActorScope:        "npc:" + gameNPCID,
		MemoryType:             "promise",
		AuthorityClass:         "fact",
		RecordVersion:          1,
		Importance:             80,
		Confidence:             "0.9300",
		SummarySecretPayloadID: secretID,
		SourceEventIdsJson:     []byte(fmt.Sprintf(`["%s"]`, eventID)),
		SourceType:             "event",
		OccurredAt:             now,
		ValidFrom:              now,
		DedupeKey:              "promise:test",
		Sensitivity:            2,
		AccessScope:            "npc_only",
	}
}

func seedScriptVersion(t *testing.T, db *database.DB, id string, scriptID string, userID string) {
	t.Helper()
	_, err := db.SQL().Exec(`
INSERT INTO script_versions (
    id, script_id, version_no, version_kind, status, schema_version,
    workflow_version, blueprint_json, validation_status, blocking_issue_count, created_by
) VALUES (?, ?, 1, 'draft', 'draft', '1.0.0', 'test-v1', ?, 'passed', 0, ?)
`, id, scriptID, []byte(`{"schema_version":"1.0.0"}`), userID)
	if err != nil {
		t.Fatalf("seed script version: %v", err)
	}
}

func seedNPCProfile(t *testing.T, db *database.DB, id string, versionID string, secretID string) {
	t.Helper()
	_, err := db.SQL().Exec(`
INSERT INTO secret_payloads (
    id, scope_type, scope_id, owner_type, owner_id, purpose_code, version_no,
    sensitivity, visibility_scope, schema_version, key_id, nonce, ciphertext,
    content_hash, compression, retention_class
) VALUES (?, 'game', ?, 'npc', ?, 'npc_private_memory', 1, 2, 'npc_only',
          '1.0.0', 'test-key', ?, ?, ?, 'none', 'test')
`, secretID, testID(220), id, bytes24(), []byte("ciphertext"), bytes32())
	if err != nil {
		t.Fatalf("seed secret payload: %v", err)
	}

	_, err = db.SQL().Exec(`
INSERT INTO npc_profiles (
    id, script_version_id, npc_key, display_name, public_profile, persona_json,
    goals_json, knowledge_scope_json, visibility_scope, sensitivity, tool_policy_json
) VALUES (?, ?, 'npc-test', 'NPC Test', 'public', ?, ?, ?, 'npc_only', 2, ?)
`, id, versionID,
		[]byte(`{"schema_version":"1.0.0"}`),
		[]byte(`{"schema_version":"1.0.0"}`),
		[]byte(`{"schema_version":"1.0.0"}`),
		[]byte(`{"schema_version":"1.0.0"}`),
	)
	if err != nil {
		t.Fatalf("seed npc profile: %v", err)
	}
}

func seedGameNPC(t *testing.T, db *database.DB, id string, gameID string, profileID string) {
	t.Helper()
	_, err := db.SQL().Exec(`
INSERT INTO game_npcs (
    id, game_session_id, npc_instance_key, script_npc_profile_id,
    display_name_snapshot, visibility_scope, runtime_status, last_event_seq
) VALUES (?, ?, 'npc-test-instance', ?, 'NPC Test', 'room', 'active', 0)
`, id, gameID, profileID)
	if err != nil {
		t.Fatalf("seed game npc: %v", err)
	}
}

func deleteRows(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.SQL().Exec(query, args...); err != nil {
		t.Errorf("cleanup %q: %v", query, err)
	}
}

func testID(number int) string {
	return fmt.Sprintf("01J%023d", number)
}

func testPublicNo(number int) string {
	return fmt.Sprintf("P%011d", number)
}

func bytes24() []byte {
	return []byte("012345678901234567890123")
}

func bytes32() []byte {
	return []byte("01234567890123456789012345678901")
}

func sqlNullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func sqlNullInt16(value int16) sql.NullInt16 {
	return sql.NullInt16{Int16: value, Valid: true}
}

func jsonRaw(value string) *json.RawMessage {
	raw := json.RawMessage(value)
	return &raw
}
