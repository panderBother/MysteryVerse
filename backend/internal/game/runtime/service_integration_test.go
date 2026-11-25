package runtime_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	gameruntime "github.com/mysteryverse/mysteryverse/backend/internal/game/runtime"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/secret"
	"github.com/mysteryverse/mysteryverse/backend/internal/room"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
)

func TestGameEngineLifecycleRolesSearchPauseVoteAndSettlement(t *testing.T) {
	repo, db := openGameIntegrationRepository(t)
	ctx := context.Background()
	hostID := gameID()
	playerID := gameID()
	seedGameUser(t, db, hostID, "Host")
	seedGameUser(t, db, playerID, "Player")
	t.Cleanup(func() {
		cleanupGameUsers(t, db, []string{hostID, playerID})
	})

	ruleID, scriptID, versionID, phaseIDs := seedMurderGame(t, db, hostID)
	t.Cleanup(func() {
		cleanupMurderContent(t, db, ruleID, scriptID, versionID, phaseIDs)
	})

	authCfg := gameAuthConfig()
	roomService := room.NewService(repo, authCfg, ratelimit.NewInMemory())
	cipher, err := secret.NewCipher("task6-game-secret-key-material", "task6-key")
	if err != nil {
		t.Fatalf("secret.NewCipher() error = %v", err)
	}
	gameService := gameruntime.NewService(repo, room.NewStartCoordinator(), cipher)

	created, err := roomService.Create(ctx, hostID, room.CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		RuleTemplateID:   &ruleID,
		Title:            "Task 6 Game",
		Visibility:       "public",
		Capacity:         room.Capacity{Min: 2, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, gameKey())
	if err != nil {
		t.Fatalf("room Create() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupGameRooms(t, db, []string{created.Room.ID})
	})
	if _, err := roomService.Join(ctx, playerID, created.Room.ID, room.JoinInput{}, gameKey()); err != nil {
		t.Fatalf("room Join() error = %v", err)
	}

	projection, err := roomService.Projection(ctx, hostID, created.Room.ID)
	if err != nil {
		t.Fatalf("room Projection() error = %v", err)
	}
	_, err = gameService.Start(ctx, hostID, created.Room.ID, gameruntime.StartInput{
		ExpectedRoomEventSeq: projection.RoomEventSeq,
		ContentVersionID:     versionID,
		RulesAck:             true,
	}, gameKey())
	if !hasCode(err, "PLAYERS_NOT_READY") {
		t.Fatalf("Start(unready) error = %v", err)
	}
	if _, err := repo.GetGameSessionByRoom(ctx, created.Room.ID); !isStorageCode(err, database.CodeNotFound) {
		t.Fatalf("invalid start left a GameSession: %v", err)
	}

	readyHost, err := roomService.Ready(ctx, hostID, created.Room.ID, room.ReadyInput{
		Ready:                true,
		ExpectedRoomEventSeq: projection.RoomEventSeq,
	}, gameKey())
	if err != nil {
		t.Fatalf("Ready(host) error = %v", err)
	}
	readyPlayer, err := roomService.Ready(ctx, playerID, created.Room.ID, room.ReadyInput{
		Ready:                true,
		ExpectedRoomEventSeq: readyHost.RoomEventSeq,
	}, gameKey())
	if err != nil {
		t.Fatalf("Ready(player) error = %v", err)
	}
	if !readyPlayer.CanStart {
		t.Fatal("room should be startable after both players are ready")
	}

	startKey := gameKey()
	started, err := gameService.Start(ctx, hostID, created.Room.ID, gameruntime.StartInput{
		ExpectedRoomEventSeq: readyPlayer.RoomEventSeq,
		ContentVersionID:     versionID,
		RulesAck:             true,
	}, startKey)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupGameRuntime(t, db, started.GameID)
	})
	if started.StateVersion != 1 || started.GameEventSeq != 4 {
		t.Fatalf("Start() = %#v", started)
	}
	replayedStart, err := gameService.Start(ctx, hostID, created.Room.ID, gameruntime.StartInput{
		ExpectedRoomEventSeq: readyPlayer.RoomEventSeq,
		ContentVersionID:     versionID,
		RulesAck:             true,
	}, gameKey())
	if err != nil || !replayedStart.Replay || replayedStart.GameID != started.GameID {
		t.Fatalf("Start() replay = %#v, %v", replayedStart, err)
	}

	hostProjection, err := gameService.Projection(ctx, hostID, started.GameID)
	if err != nil {
		t.Fatalf("Projection() error = %v", err)
	}
	hostRole, err := gameService.MyRole(ctx, hostID, started.GameID)
	if err != nil {
		t.Fatalf("MyRole(host) error = %v", err)
	}
	playerRole, err := gameService.MyRole(ctx, playerID, started.GameID)
	if err != nil {
		t.Fatalf("MyRole(player) error = %v", err)
	}
	if hostRole.RoleID == "" || playerRole.RoleID == "" || hostRole.RoleID == playerRole.RoleID {
		t.Fatalf("role assignments = %#v / %#v", hostRole, playerRole)
	}
	roleState, _ := hostProjection.PrivateState["role"].(map[string]any)
	if roleState["role_id"] != hostRole.RoleID {
		t.Fatalf("private projection role = %#v", roleState)
	}
	if _, leaked := hostProjection.PrivateState["other_roles"]; leaked {
		t.Fatal("projection leaked other role assignments")
	}
	hostEvents, err := gameService.Events(ctx, hostID, started.GameID, 0, 100)
	if err != nil {
		t.Fatalf("Events(host) error = %v", err)
	}
	roleEvents := 0
	for _, event := range hostEvents.Events {
		if event.EventType == "game.role_assigned" {
			roleEvents++
		}
	}
	if roleEvents != 1 {
		t.Fatalf("visible role events = %d, want only own role", roleEvents)
	}

	paused, err := gameService.Pause(ctx, hostID, started.GameID, gameruntime.PauseInput{
		ExpectedVersion: hostProjection.StateVersion,
	}, gameKey())
	if err != nil || paused.StateVersion != 2 {
		t.Fatalf("Pause() = %#v, %v", paused, err)
	}
	resumed, err := gameService.Resume(ctx, hostID, started.GameID, gameruntime.PauseInput{
		ExpectedVersion: paused.StateVersion,
	}, gameKey())
	if err != nil || resumed.StateVersion != 3 {
		t.Fatalf("Resume() = %#v, %v", resumed, err)
	}

	readingHost, err := gameService.ReadingComplete(ctx, hostID, started.GameID, resumed.StateVersion, gameKey())
	if err != nil {
		t.Fatalf("ReadingComplete(host) error = %v", err)
	}
	readingPlayer, err := gameService.ReadingComplete(ctx, playerID, started.GameID, readingHost.StateVersion, gameKey())
	if err != nil {
		t.Fatalf("ReadingComplete(player) error = %v", err)
	}
	openingProjection, err := gameService.Projection(ctx, hostID, started.GameID)
	if err != nil || openingProjection.Phase.Key != "opening" {
		t.Fatalf("phase after reading = %#v, %v", openingProjection.Phase, err)
	}
	advanced, err := gameService.AdvancePhase(
		ctx,
		hostID,
		started.GameID,
		readingPlayer.StateVersion,
		gameKey(),
	)
	if err != nil {
		t.Fatalf("AdvancePhase(opening) error = %v", err)
	}
	searchPoints, err := gameService.SearchPoints(ctx, hostID, started.GameID)
	if err != nil || len(searchPoints.SearchPoints) != 1 {
		t.Fatalf("SearchPoints() = %#v, %v", searchPoints, err)
	}

	type searchOutcome struct {
		userID string
		key    string
		result gameruntime.ActionResult
		err    error
	}
	outcomes := make(chan searchOutcome, 2)
	var wait sync.WaitGroup
	for _, userID := range []string{hostID, playerID} {
		userID := userID
		key := gameKey()
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := gameService.Search(ctx, userID, started.GameID, gameruntime.SearchInput{
				SearchPointID:   searchPoints.SearchPoints[0].PointID,
				ExpectedVersion: advanced.StateVersion,
			}, key)
			outcomes <- searchOutcome{userID: userID, key: key, result: result, err: err}
		}()
	}
	wait.Wait()
	close(outcomes)

	var winner searchOutcome
	var loser searchOutcome
	successes := 0
	conflicts := 0
	for outcome := range outcomes {
		switch {
		case outcome.err == nil:
			successes++
			winner = outcome
		case hasCode(outcome.err, "STATE_VERSION_CONFLICT"):
			conflicts++
			loser = outcome
		default:
			t.Fatalf("concurrent search error = %v", outcome.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent search = %d success, %d conflict", successes, conflicts)
	}
	loserProjection, err := gameService.Projection(ctx, loser.userID, started.GameID)
	if err != nil {
		t.Fatalf("Projection(loser) error = %v", err)
	}
	_, err = gameService.Search(ctx, loser.userID, started.GameID, gameruntime.SearchInput{
		SearchPointID:   searchPoints.SearchPoints[0].PointID,
		ExpectedVersion: loserProjection.StateVersion,
	}, loser.key)
	if !hasCode(err, "CLUE_ALREADY_CLAIMED") {
		t.Fatalf(
			"Search(second claimant) error = %v (expected_version=%d)",
			err,
			loserProjection.StateVersion,
		)
	}
	winnerClues, err := gameService.Clues(ctx, winner.userID, started.GameID)
	if err != nil || len(winnerClues) != 1 {
		t.Fatalf("Clues(winner) = %#v, %v", winnerClues, err)
	}
	loserClues, err := gameService.Clues(ctx, loser.userID, started.GameID)
	if err != nil || len(loserClues) != 0 {
		t.Fatalf("Clues(loser) = %#v, %v", loserClues, err)
	}

	postSearch, err := gameService.Projection(ctx, hostID, started.GameID)
	if err != nil {
		t.Fatalf("Projection(post search) error = %v", err)
	}
	voteStart, err := gameService.AdvancePhase(
		ctx,
		hostID,
		started.GameID,
		postSearch.StateVersion,
		gameKey(),
	)
	if err != nil {
		t.Fatalf("AdvancePhase(search) error = %v", err)
	}
	culpritPlayerID := culpritPlayer(t, hostRole, playerRole)

	type voteOutcome struct {
		userID string
		key    string
		result gameruntime.ActionResult
		err    error
	}
	voteOutcomes := make(chan voteOutcome, 2)
	for _, userID := range []string{hostID, playerID} {
		userID := userID
		key := gameKey()
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := gameService.Vote(ctx, userID, started.GameID, gameruntime.VoteInput{
				TargetPlayerID:  culpritPlayerID,
				ExpectedVersion: voteStart.StateVersion,
				RoundKey:        "vote",
			}, key)
			voteOutcomes <- voteOutcome{userID: userID, key: key, result: result, err: err}
		}()
	}
	wait.Wait()
	close(voteOutcomes)

	var firstVoter voteOutcome
	var secondVoter voteOutcome
	voteSuccess := 0
	voteConflicts := 0
	for outcome := range voteOutcomes {
		switch {
		case outcome.err == nil:
			voteSuccess++
			firstVoter = outcome
		case hasCode(outcome.err, "STATE_VERSION_CONFLICT"):
			voteConflicts++
			secondVoter = outcome
		default:
			t.Fatalf("concurrent vote error = %v", outcome.err)
		}
	}
	if voteSuccess != 1 || voteConflicts != 1 {
		t.Fatalf("concurrent vote = %d success, %d conflict", voteSuccess, voteConflicts)
	}
	secondProjection, err := gameService.Projection(ctx, secondVoter.userID, started.GameID)
	if err != nil {
		t.Fatalf("Projection(second voter) error = %v", err)
	}
	finalVote, err := gameService.Vote(ctx, secondVoter.userID, started.GameID, gameruntime.VoteInput{
		TargetPlayerID:  culpritPlayerID,
		ExpectedVersion: secondProjection.StateVersion,
		RoundKey:        "vote",
	}, gameKey())
	if err != nil {
		t.Fatalf("final Vote() error = %v", err)
	}
	if finalVote.ResultCode != "murder_solved" {
		t.Fatalf("final vote result = %#v", finalVote)
	}
	result, err := gameService.Result(ctx, playerID, started.GameID)
	if err != nil {
		t.Fatalf("Result() error = %v", err)
	}
	if result.ResultCode != "murder_solved" || result.WinnerTeam != "innocents" {
		t.Fatalf("Result() = %#v", result)
	}
	if len(result.RevealedRoles) != 2 {
		t.Fatalf("revealed roles = %d", len(result.RevealedRoles))
	}
	allEvents, err := gameService.Events(ctx, playerID, started.GameID, 0, 200)
	if err != nil {
		t.Fatalf("Events(final) error = %v", err)
	}
	lastSeq := uint64(0)
	for _, event := range allEvents.Events {
		if event.Seq <= lastSeq {
			t.Fatalf("visible event seq = %d after %d", event.Seq, lastSeq)
		}
		lastSeq = event.Seq
	}
	rows, err := db.SQL().Query("SELECT seq FROM game_events WHERE game_session_id = ? ORDER BY seq", started.GameID)
	if err != nil {
		t.Fatalf("query event sequence: %v", err)
	}
	defer rows.Close()
	expectedSeq := uint64(1)
	for rows.Next() {
		var seq uint64
		if err := rows.Scan(&seq); err != nil {
			t.Fatalf("scan event seq: %v", err)
		}
		if seq != expectedSeq {
			t.Fatalf("stored event seq = %d, want %d", seq, expectedSeq)
		}
		expectedSeq++
	}

	_, err = gameService.Vote(ctx, firstVoter.userID, started.GameID, gameruntime.VoteInput{
		TargetPlayerID:  culpritPlayerID,
		ExpectedVersion: voteStart.StateVersion,
		RoundKey:        "vote",
	}, firstVoter.key)
	if err != nil {
		t.Fatalf("idempotent vote replay error = %v", err)
	}
}

func TestGamePlayerExitAbortsWhenAllPlayersLeave(t *testing.T) {
	repo, db := openGameIntegrationRepository(t)
	ctx := context.Background()
	hostID := gameID()
	playerID := gameID()
	seedGameUser(t, db, hostID, "ExitHost")
	seedGameUser(t, db, playerID, "ExitPlayer")
	t.Cleanup(func() {
		cleanupGameUsers(t, db, []string{hostID, playerID})
	})
	ruleID, scriptID, versionID, phaseIDs := seedMurderGame(t, db, hostID)
	t.Cleanup(func() {
		cleanupMurderContent(t, db, ruleID, scriptID, versionID, phaseIDs)
	})

	roomService := room.NewService(repo, gameAuthConfig(), ratelimit.NewInMemory())
	cipher, err := secret.NewCipher("task6-game-secret-key-material", "task6-key")
	if err != nil {
		t.Fatalf("secret.NewCipher() error = %v", err)
	}
	gameService := gameruntime.NewService(repo, room.NewStartCoordinator(), cipher)

	roomResult, err := roomService.Create(ctx, hostID, room.CreateInput{
		ScriptVersionID:  versionID,
		GameType:         "script_murder",
		RuleTemplateID:   &ruleID,
		Title:            "Exit Game",
		Visibility:       "public",
		Capacity:         room.Capacity{Min: 2, Max: 2},
		ModeConfig:       map[string]any{},
		PrivacyConfirmed: true,
	}, gameKey())
	if err != nil {
		t.Fatalf("room Create() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupGameRooms(t, db, []string{roomResult.Room.ID})
	})
	if _, err := roomService.Join(ctx, playerID, roomResult.Room.ID, room.JoinInput{}, gameKey()); err != nil {
		t.Fatalf("room Join() error = %v", err)
	}
	projection, err := roomService.Projection(ctx, hostID, roomResult.Room.ID)
	if err != nil {
		t.Fatalf("room Projection() error = %v", err)
	}
	readyHost, err := roomService.Ready(ctx, hostID, roomResult.Room.ID, room.ReadyInput{
		Ready:                true,
		ExpectedRoomEventSeq: projection.RoomEventSeq,
	}, gameKey())
	if err != nil {
		t.Fatalf("Ready(host) error = %v", err)
	}
	readyPlayer, err := roomService.Ready(ctx, playerID, roomResult.Room.ID, room.ReadyInput{
		Ready:                true,
		ExpectedRoomEventSeq: readyHost.RoomEventSeq,
	}, gameKey())
	if err != nil {
		t.Fatalf("Ready(player) error = %v", err)
	}
	started, err := gameService.Start(ctx, hostID, roomResult.Room.ID, gameruntime.StartInput{
		ExpectedRoomEventSeq: readyPlayer.RoomEventSeq,
		ContentVersionID:     versionID,
		RulesAck:             true,
	}, gameKey())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupGameRuntime(t, db, started.GameID)
	})

	firstExit, err := gameService.PlayerExit(ctx, hostID, started.GameID)
	if err != nil || firstExit.Status != "running" {
		t.Fatalf("PlayerExit(host) = %#v, %v", firstExit, err)
	}
	secondExit, err := gameService.PlayerExit(ctx, playerID, started.GameID)
	if err != nil || secondExit.Status != "aborted" {
		t.Fatalf("PlayerExit(player) = %#v, %v", secondExit, err)
	}
	result, err := gameService.Result(ctx, playerID, started.GameID)
	if err != nil || result.ResultCode != "aborted_player_exit" {
		t.Fatalf("Result() = %#v, %v", result, err)
	}
}

func openGameIntegrationRepository(t *testing.T) (*storage.MySQLRepository, *database.DB) {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not set")
	}
	db, err := database.Open(context.Background(), config.MySQLConfig{
		DSN:             dsn,
		MaxOpenConns:    16,
		MaxIdleConns:    8,
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

func gameAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		IdentityHMACKey:    strings.Repeat("g", 32),
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

func seedGameUser(t *testing.T, db *database.DB, userID string, nickname string) {
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
`, gameID(), userID, nickname,
		[]byte(`{"schema_version":"1.0.0"}`),
		[]byte(`{"history_visibility":"private","online_visibility":"friends","profile_visibility":"public","friend_request_policy":"friends_of_friends"}`),
	); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
}

func seedMurderGame(
	t *testing.T,
	db *database.DB,
	ownerID string,
) (string, string, string, []string) {
	t.Helper()
	ruleID := gameID()
	scriptID := gameID()
	versionID := gameID()
	readPhaseID := gameID()
	openingPhaseID := gameID()
	searchPhaseID := gameID()
	votePhaseID := gameID()
	culpritCharacterID := gameID()
	civilianCharacterID := gameID()
	clueID := gameID()
	pointID := gameID()

	policy := map[string]any{
		"schema_version": "1.0.0",
		"murder": map[string]any{
			"schema_version":         "1.0.0",
			"default_search_quota":   1,
			"allow_abstain":          true,
			"allow_revote":           true,
			"allow_self_vote":        true,
			"vote_visibility":        "count_only",
			"tie_policy":             "no_elimination",
			"culprit_character_keys": []string{"killer"},
			"voting_character_keys":  []string{"killer", "civilian"},
			"role_factions": map[string]string{
				"killer":   "culprit",
				"civilian": "civilian",
			},
			"role_teams": map[string]string{},
		},
	}
	policyJSON, _ := json.Marshal(policy)
	hash := sha256.Sum256(policyJSON)
	if _, err := db.SQL().Exec(`
INSERT INTO rule_templates (
    id, template_key, version_no, game_type, name, status, schema_version,
    engine_version, config_json, owner_user_id, content_hash, published_at
) VALUES (?, ?, 1, 'script_murder', 'Task 6 Rule', 'active', '1.0.0',
          'murder-engine-1.0.0', ?, ?, ?, UTC_TIMESTAMP(3))
`, ruleID, "task6-"+strings.ToLower(ruleID), policyJSON, ownerID, hash[:]); err != nil {
		t.Fatalf("seed rule template: %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO scripts (
    id, owner_user_id, public_slug, title, summary, game_type, source_kind,
    rights_type, status, cover_asset_uri, language, content_rating,
    min_players, max_players, duration_minutes, difficulty, ai_assisted_ratio_bp
) VALUES (?, ?, NULL, 'Task 6 Script', NULL, 'script_murder', 'original',
          'original', 'published', NULL, 'zh-CN', '12+', 2, 2, 90, 'standard', 0)
`, scriptID, ownerID); err != nil {
		t.Fatalf("seed script: %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO script_versions (
    id, script_id, version_no, version_kind, status, schema_version,
    workflow_version, rule_template_id, blueprint_json, validation_status,
    blocking_issue_count, created_by, frozen_at, published_at
) VALUES (?, ?, 1, 'release', 'published_public', '1.0.0', 'test-v1', ?, ?,
          'passed', 0, ?, UTC_TIMESTAMP(3), UTC_TIMESTAMP(3))
`, versionID, scriptID, ruleID, []byte(`{"schema_version":"1.0.0"}`), ownerID); err != nil {
		t.Fatalf("seed script version: %v", err)
	}

	phases := []struct {
		id       string
		key      string
		kind     string
		sequence int
		actions  []string
		fallback string
	}{
		{readPhaseID, "read", "read", 1, []string{"reading_complete"}, "advance_to_opening"},
		{openingPhaseID, "opening", "opening", 2, []string{"advance_phase"}, "advance_to_search"},
		{searchPhaseID, "search", "search", 3, []string{"search", "advance_phase"}, "advance_to_vote"},
		{votePhaseID, "vote", "vote", 4, []string{"vote"}, "settle"},
	}
	for _, phase := range phases {
		actionsJSON, _ := json.Marshal(phase.actions)
		if _, err := db.SQL().Exec(`
INSERT INTO script_phases (
    id, script_version_id, phase_key, phase_type, sequence_no, title,
    timeout_seconds, fallback_action, allowed_actions_json, visibility_scope
) VALUES (?, ?, ?, ?, ?, ?, 600, ?, ?, 'room')
`, phase.id, versionID, phase.key, phase.kind, phase.sequence,
			strings.ToUpper(phase.key), phase.fallback, actionsJSON); err != nil {
			t.Fatalf("seed phase %s: %v", phase.key, err)
		}
	}

	for index, character := range []struct {
		id    string
		key   string
		name  string
		order int
	}{
		{culpritCharacterID, "killer", "Killer", 1},
		{civilianCharacterID, "civilian", "Civilian", 2},
	} {
		if _, err := db.SQL().Exec(`
INSERT INTO script_characters (
    id, script_version_id, character_key, display_name, role_class,
    public_profile, public_goals_json, attributes_json, visibility_scope,
    sensitivity, generated_origin, sort_order
) VALUES (?, ?, ?, ?, 'player_role', ?, ?, ?, 'role', 2, 'source', ?)
`, character.id, versionID, character.key, character.name,
			character.name+" profile", []byte(`["Find the culprit"]`),
			[]byte(`{}`), character.order); err != nil {
			t.Fatalf("seed character %d: %v", index, err)
		}
	}

	if _, err := db.SQL().Exec(`
INSERT INTO script_clues (
    id, script_version_id, clue_key, title, public_text, clue_type,
    is_repeatable, max_discoveries, visibility_scope, sensitivity, sort_order
) VALUES (?, ?, 'private_clue', 'Private Clue', 'Only the finder can see this.',
          'private', 0, 1, 'player', 2, 1)
`, clueID, versionID); err != nil {
		t.Fatalf("seed clue: %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO script_search_points (
    id, script_version_id, point_key, phase_id, title, public_description,
    max_searches, conflict_policy, visibility_scope
) VALUES (?, ?, 'point_1', ?, 'Cabinet', 'A locked cabinet.', 1, 'first_wins', 'room')
`, pointID, versionID, searchPhaseID); err != nil {
		t.Fatalf("seed search point: %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO script_search_point_clues (
    script_version_id, search_point_id, clue_id, weight, is_default
) VALUES (?, ?, ?, 1, 1)
`, versionID, pointID, clueID); err != nil {
		t.Fatalf("seed search point clue: %v", err)
	}
	return ruleID, scriptID, versionID, []string{
		readPhaseID,
		openingPhaseID,
		searchPhaseID,
		votePhaseID,
	}
}

func culpritPlayer(
	t *testing.T,
	hostRole gameruntime.RoleView,
	playerRole gameruntime.RoleView,
) string {
	t.Helper()
	if hostRole.Faction == "culprit" {
		return hostRole.PlayerID
	}
	if playerRole.Faction == "culprit" {
		return playerRole.PlayerID
	}
	t.Fatal("no culprit role assigned")
	return ""
}

func cleanupGameRuntime(t *testing.T, db *database.DB, gameID string) {
	t.Helper()
	queries := []string{
		"DELETE FROM replay_reports WHERE game_session_id = ?",
		"DELETE FROM game_results WHERE game_session_id = ?",
		"DELETE FROM game_search_claims WHERE game_session_id = ?",
		"DELETE FROM game_votes WHERE game_session_id = ?",
		"DELETE FROM game_timers WHERE game_session_id = ?",
		"DELETE FROM game_events WHERE game_session_id = ?",
		"DELETE FROM game_intents WHERE game_session_id = ?",
		"DELETE FROM game_states WHERE game_session_id = ?",
		"DELETE FROM game_roles WHERE game_session_id = ?",
		"DELETE FROM game_players WHERE game_session_id = ?",
		"DELETE FROM game_sessions WHERE id = ?",
		"DELETE FROM secret_payloads WHERE scope_type = 'game' AND scope_id = ?",
	}
	for _, query := range queries {
		deleteGameRows(t, db, query, gameID)
	}
}

func cleanupGameRooms(t *testing.T, db *database.DB, roomIDs []string) {
	t.Helper()
	for _, roomID := range roomIDs {
		deleteGameRows(t, db, "DELETE FROM idempotency_records WHERE resource_id = ?", roomID)
		deleteGameRows(t, db, "DELETE FROM user_sanctions WHERE scope_ref_id = ?", roomID)
		deleteGameRows(t, db, "DELETE FROM room_messages WHERE room_id = ?", roomID)
		deleteGameRows(t, db, "DELETE FROM invitation_tokens WHERE room_id = ?", roomID)
		deleteGameRows(t, db, "DELETE FROM room_events WHERE room_id = ?", roomID)
		deleteGameRows(t, db, "DELETE FROM room_members WHERE room_id = ?", roomID)
		deleteGameRows(t, db, "DELETE FROM game_rooms WHERE id = ?", roomID)
	}
}

func cleanupMurderContent(
	t *testing.T,
	db *database.DB,
	ruleID string,
	scriptID string,
	versionID string,
	phaseIDs []string,
) {
	t.Helper()
	deleteGameRows(t, db, "DELETE FROM script_search_point_clues WHERE script_version_id = ?", versionID)
	deleteGameRows(t, db, "DELETE FROM script_search_points WHERE script_version_id = ?", versionID)
	deleteGameRows(t, db, "DELETE FROM script_clues WHERE script_version_id = ?", versionID)
	deleteGameRows(t, db, "DELETE FROM script_characters WHERE script_version_id = ?", versionID)
	for _, phaseID := range phaseIDs {
		deleteGameRows(t, db, "DELETE FROM script_phases WHERE id = ?", phaseID)
	}
	deleteGameRows(t, db, "DELETE FROM script_versions WHERE id = ?", versionID)
	deleteGameRows(t, db, "DELETE FROM scripts WHERE id = ?", scriptID)
	deleteGameRows(t, db, "DELETE FROM rule_templates WHERE id = ?", ruleID)
}

func cleanupGameUsers(t *testing.T, db *database.DB, userIDs []string) {
	t.Helper()
	for _, userID := range userIDs {
		deleteGameRows(t, db, "DELETE FROM user_profiles WHERE user_id = ?", userID)
		deleteGameRows(t, db, "DELETE FROM users WHERE id = ?", userID)
	}
}

func deleteGameRows(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.SQL().Exec(query, args...); err != nil {
		t.Errorf("cleanup %q: %v", query, err)
	}
}

func gameID() string {
	return id.ULIDGenerator{}.New()
}

func gameKey() string {
	return id.ULIDGenerator{}.New()
}

func hasCode(err error, code string) bool {
	if err == nil {
		return false
	}
	var appErr apperror.Error
	return errors.As(err, &appErr) && appErr.Code == code
}

func isStorageCode(err error, code string) bool {
	var storageErr *database.Error
	return errors.As(err, &storageErr) && storageErr.Code == code
}
